package charmingui

import (
	"image"
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type testMsg struct{}

type testModel struct {
	view string
}

func (m testModel) Init() tea.Cmd { return nil }
func (m testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(testMsg); ok {
		m.view = "updated"
	}
	return m, nil
}
func (m testModel) View() tea.View { return tea.NewView(m.view) }

func TestRenderFrameParsesCursorAndStyles(t *testing.T) {
	r, err := New(Config{Columns: 5, Rows: 3})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = r.RenderFrame(nil, "hi\x1b[2;3H\x1b[31m!")
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}

	if got := r.screen.Cell(0, 0).Text; got != "h" {
		t.Fatalf("expected h at 0,0, got %q", got)
	}
	if got := r.screen.Cell(1, 0).Text; got != "i" {
		t.Fatalf("expected i at 1,0, got %q", got)
	}
	cell := r.screen.Cell(2, 1)
	if cell.Text != "!" {
		t.Fatalf("expected ! at 2,1, got %q", cell.Text)
	}
	if cell.Style.FG != ansiColor(1) {
		t.Fatalf("expected red foreground, got %#v", cell.Style.FG)
	}
}

func TestRenderFrameDamageDetectionOnReusedSurface(t *testing.T) {
	r, err := New(Config{Columns: 4, Rows: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	img := image.NewRGBA(r.SurfaceBounds())
	first, err := r.RenderFrame(img, "test")
	if err != nil {
		t.Fatalf("first render: %v", err)
	}
	if !first.FullRedraw {
		t.Fatalf("first render to an external surface should be a full redraw")
	}

	second, err := r.RenderFrame(img, "tent")
	if err != nil {
		t.Fatalf("second render: %v", err)
	}
	if second.ChangedCells == 0 {
		t.Fatalf("expected dirty cells on second render")
	}
	if len(second.DirtyRects) == 0 {
		t.Fatalf("expected dirty rects on second render")
	}
}

func TestRenderModelAndAdapterUpdate(t *testing.T) {
	r, err := New(Config{Columns: 10, Rows: 2})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	model := testModel{view: "\x1b[32mhello"}

	result, err := r.RenderModel(nil, model)
	if err != nil {
		t.Fatalf("RenderModel: %v", err)
	}
	if result.Image == nil {
		t.Fatalf("expected owned image")
	}

	viewResult, err := r.RenderView(nil, tea.NewView("\x1b[34mview"))
	if err != nil {
		t.Fatalf("RenderView: %v", err)
	}
	if viewResult.Image == nil {
		t.Fatalf("expected image from RenderView")
	}

	adapter, err := NewModelAdapter(r, model)
	if err != nil {
		t.Fatalf("NewModelAdapter: %v", err)
	}
	update, err := adapter.Update(testMsg{}, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := adapter.Model().View().Content; got != "updated" {
		t.Fatalf("expected updated model view, got %q", got)
	}
	if update.Image == nil {
		t.Fatalf("expected updated image")
	}
}

func TestRenderedImageContainsGlyphPixels(t *testing.T) {
	r, err := New(Config{Columns: 2, Rows: 1, DefaultBG: color.Black, DefaultFG: color.White})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := r.RenderFrame(nil, "A")
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	img := result.Image.(*image.RGBA)
	found := false
	for y := 0; y < img.Bounds().Dy() && !found; y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.RGBAAt(x, y) != (color.RGBA{A: 0xff}) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected non-background glyph pixels")
	}
}
