package charmingui

import (
	"image"
	"image/color"
	imagedraw "image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func (r *Renderer) drawDirty(dst imagedraw.Image, current *Screen, dirty []bool, fullRedraw bool) []image.Rectangle {
	if fullRedraw {
		dirty = make([]bool, len(current.cells))
		for i := range dirty {
			dirty[i] = true
		}
	}

	rects := make([]image.Rectangle, 0, current.Height)
	for y := 0; y < current.Height; y++ {
		runStart := -1
		for x := 0; x < current.Width; x++ {
			idx := current.index(x, y)
			if !dirty[idx] {
				if runStart >= 0 {
					rects = append(rects, r.pixelRect(runStart, y, x-runStart, 1))
					runStart = -1
				}
				continue
			}
			if runStart < 0 {
				runStart = x
			}
			var cursor *CursorState
			if current.Cursor.Visible && current.Cursor.X == x && current.Cursor.Y == y {
				cursor = &current.Cursor
			}
			r.drawCell(dst, x, y, current.Cell(x, y), cursor)
		}
		if runStart >= 0 {
			rects = append(rects, r.pixelRect(runStart, y, current.Width-runStart, 1))
		}
	}
	return rects
}

func (r *Renderer) drawCell(dst imagedraw.Image, x, y int, cell Cell, cursor *CursorState) {
	fg, bg := effectiveColors(cell.Style)
	rect := r.pixelRect(x, y, 1, 1)
	imagedraw.Draw(dst, rect, image.NewUniform(bg), image.Point{}, imagedraw.Src)

	if !cell.Continuation && cell.Text != "" && cell.Text != " " {
		face := r.cfg.Face
		if cell.Style.Bold && r.cfg.BoldFace != nil {
			face = r.cfg.BoldFace
		}
		drawer := font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(fg),
			Face: face,
			Dot:  fixed.P(rect.Min.X, rect.Min.Y+r.baseline),
		}
		drawer.DrawString(cell.Text)
		if cell.Style.Bold && face == r.cfg.Face {
			drawer.Dot = fixed.P(rect.Min.X+1, rect.Min.Y+r.baseline)
			drawer.DrawString(cell.Text)
		}
	}
	if cell.Style.Underline {
		r.drawHorizontalLine(dst, rect.Min.X, rect.Max.X, rect.Max.Y-2, fg)
	}
	if cell.Style.Strikethrough {
		r.drawHorizontalLine(dst, rect.Min.X, rect.Max.X, rect.Min.Y+(r.cellHeight/2), fg)
	}
	if cursor != nil {
		r.drawCursor(dst, rect, cursor.Color)
	}
}

func (r *Renderer) drawCursor(dst imagedraw.Image, rect image.Rectangle, cursorColor color.RGBA) {
	lineHeight := 2
	if lineHeight > rect.Dy() {
		lineHeight = rect.Dy()
	}
	cursorRect := image.Rect(rect.Min.X, rect.Max.Y-lineHeight, rect.Max.X, rect.Max.Y)
	if cursorColor == (color.RGBA{}) {
		cursorColor = r.cfg.CursorColor
	}
	imagedraw.Draw(dst, cursorRect, image.NewUniform(cursorColor), image.Point{}, imagedraw.Src)
}

func (r *Renderer) drawHorizontalLine(dst imagedraw.Image, minX, maxX, y int, c color.RGBA) {
	if y < 0 {
		return
	}
	if maxX <= minX {
		return
	}
	line := image.Rect(minX, y, maxX, y+1)
	imagedraw.Draw(dst, line, image.NewUniform(c), image.Point{}, imagedraw.Src)
}

func effectiveColors(style CellStyle) (color.RGBA, color.RGBA) {
	fg := style.FG
	bg := style.BG
	if style.Inverse {
		fg, bg = bg, fg
	}
	if style.Faint {
		fg = blend(fg, bg, 0.5)
	}
	return fg, bg
}

func blend(a, b color.RGBA, alpha float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-alpha) + float64(b.R)*alpha),
		G: uint8(float64(a.G)*(1-alpha) + float64(b.G)*alpha),
		B: uint8(float64(a.B)*(1-alpha) + float64(b.B)*alpha),
		A: 0xff,
	}
}
