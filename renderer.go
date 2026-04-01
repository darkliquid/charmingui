package charmingui

import (
	"fmt"
	"image"
	"image/draw"
	"reflect"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/image/font"
)

// RenderResult describes the outcome of a render pass.
type RenderResult struct {
	Image        image.Image
	DirtyRects   []image.Rectangle
	ChangedCells int
	FullRedraw   bool
}

// Renderer turns Bubble Tea views or ANSI terminal streams into pixels.
type Renderer struct {
	cfg         normalizedConfig
	screen      *Screen
	surface     *image.RGBA
	cellWidth   int
	cellHeight  int
	baseline    int
	initialized bool
	lastSurface uintptr
}

func New(cfg Config) (*Renderer, error) {
	normalized, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}
	cellWidth, cellHeight, baseline, err := measureFace(normalized.Face)
	if err != nil {
		return nil, err
	}
	return &Renderer{
		cfg:        normalized,
		screen:     newScreen(normalized.Columns, normalized.Rows, normalized.defaultStyle()),
		cellWidth:  cellWidth,
		cellHeight: cellHeight,
		baseline:   baseline,
	}, nil
}

func measureFace(face font.Face) (int, int, int, error) {
	if face == nil {
		return 0, 0, 0, fmt.Errorf("font face cannot be nil")
	}
	metrics := face.Metrics()
	height := metrics.Height.Ceil()
	if height == 0 {
		height = (metrics.Ascent + metrics.Descent).Ceil()
	}
	width := font.MeasureString(face, "W").Ceil()
	if width <= 0 || height <= 0 {
		return 0, 0, 0, fmt.Errorf("font face must report positive metrics")
	}
	return width, height, metrics.Ascent.Ceil(), nil
}

func (r *Renderer) SurfaceBounds() image.Rectangle {
	return image.Rect(0, 0, r.cfg.Columns*r.cellWidth, r.cfg.Rows*r.cellHeight)
}

func (r *Renderer) CellSize() image.Point {
	return image.Pt(r.cellWidth, r.cellHeight)
}

func (r *Renderer) Reset() {
	r.screen.Reset()
}

func (r *Renderer) Surface() image.Image {
	return r.surface
}

func (r *Renderer) RenderFrame(dst draw.Image, frame string) (RenderResult, error) {
	next := newScreen(r.cfg.Columns, r.cfg.Rows, r.cfg.defaultStyle())
	if err := parseANSI(next, frame, r.cfg.TabWidth); err != nil {
		return RenderResult{}, err
	}
	return r.render(dst, next)
}

func (r *Renderer) RenderStream(dst draw.Image, stream string) (RenderResult, error) {
	next := r.screen.Clone()
	if err := parseANSI(next, stream, r.cfg.TabWidth); err != nil {
		return RenderResult{}, err
	}
	return r.render(dst, next)
}

func (r *Renderer) RenderModel(dst draw.Image, model tea.Model) (RenderResult, error) {
	if model == nil {
		return RenderResult{}, fmt.Errorf("model cannot be nil")
	}
	return r.RenderView(dst, model.View())
}

func (r *Renderer) RenderView(dst draw.Image, view tea.View) (RenderResult, error) {
	defaultStyle := r.cfg.defaultStyle()
	if view.ForegroundColor != nil {
		defaultStyle.FG = toRGBA(view.ForegroundColor)
	}
	if view.BackgroundColor != nil {
		defaultStyle.BG = toRGBA(view.BackgroundColor)
	}
	next := newScreen(r.cfg.Columns, r.cfg.Rows, defaultStyle)
	next.Cursor.Visible = false
	next.SavedCursor.Visible = false
	if err := parseANSI(next, view.Content, r.cfg.TabWidth); err != nil {
		return RenderResult{}, err
	}
	if view.Cursor != nil {
		next.Cursor = CursorState{
			X:       clamp(view.Cursor.X, 0, r.cfg.Columns-1),
			Y:       clamp(view.Cursor.Y, 0, r.cfg.Rows-1),
			Visible: true,
			Color:   toRGBA(view.Cursor.Color),
		}
	}
	return r.render(dst, next)
}

func (r *Renderer) render(dst draw.Image, next *Screen) (RenderResult, error) {
	previous := r.screen
	target, fullRedraw, err := r.prepareSurface(dst)
	if err != nil {
		return RenderResult{}, err
	}

	dirty, changedCells := diffScreens(previous, next, fullRedraw)
	dirtyRects := r.drawDirty(target, next, dirty, fullRedraw)
	r.screen = next

	return RenderResult{
		Image:        target,
		DirtyRects:   dirtyRects,
		ChangedCells: changedCells,
		FullRedraw:   fullRedraw,
	}, nil
}

func (r *Renderer) prepareSurface(dst draw.Image) (draw.Image, bool, error) {
	bounds := r.SurfaceBounds()
	if dst != nil {
		if !dst.Bounds().Eq(bounds) {
			return nil, false, fmt.Errorf("surface bounds %v do not match renderer bounds %v", dst.Bounds(), bounds)
		}
		key := surfaceKey(dst)
		fullRedraw := !r.initialized || r.lastSurface != key
		r.initialized = true
		r.lastSurface = key
		return dst, fullRedraw, nil
	}
	if r.surface == nil || !r.surface.Bounds().Eq(bounds) {
		r.surface = image.NewRGBA(bounds)
	}
	key := surfaceKey(r.surface)
	fullRedraw := !r.initialized || r.lastSurface != key
	r.initialized = true
	r.lastSurface = key
	if fullRedraw {
		return r.surface, true, nil
	}
	return r.surface, false, nil
}

func diffScreens(previous, current *Screen, fullRedraw bool) ([]bool, int) {
	dirty := make([]bool, len(current.cells))
	if fullRedraw || previous == nil {
		for i := range dirty {
			dirty[i] = true
		}
		return dirty, len(dirty)
	}
	for i := range current.cells {
		if current.cells[i] != previous.cells[i] {
			dirty[i] = true
		}
	}
	markCursorDirty(dirty, previous)
	markCursorDirty(dirty, current)
	changed := 0
	for _, flag := range dirty {
		if flag {
			changed++
		}
	}
	return dirty, changed
}

func markCursorDirty(dirty []bool, screen *Screen) {
	if screen == nil || !screen.Cursor.Visible || !screen.inBounds(screen.Cursor.X, screen.Cursor.Y) {
		return
	}
	dirty[screen.index(screen.Cursor.X, screen.Cursor.Y)] = true
}

func (r *Renderer) pixelRect(cellX, cellY, width, height int) image.Rectangle {
	return image.Rect(
		cellX*r.cellWidth,
		cellY*r.cellHeight,
		(cellX+width)*r.cellWidth,
		(cellY+height)*r.cellHeight,
	)
}

func surfaceKey(surface draw.Image) uintptr {
	value := reflect.ValueOf(surface)
	if value.Kind() == reflect.Pointer || value.Kind() == reflect.UnsafePointer {
		return value.Pointer()
	}
	return 0
}
