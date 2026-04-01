package charmingui

import (
	"image/color"

	"github.com/mattn/go-runewidth"
)

// CellStyle captures the visual style of a terminal cell.
type CellStyle struct {
	FG            color.RGBA
	BG            color.RGBA
	Bold          bool
	Faint         bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	Inverse       bool
}

// Cell holds the retained visual state for a single terminal cell.
type Cell struct {
	Text         string
	Style        CellStyle
	Continuation bool
}

type CursorState struct {
	X       int
	Y       int
	Visible bool
	Color   color.RGBA
}

type Screen struct {
	Width        int
	Height       int
	DefaultStyle CellStyle
	CurrentStyle CellStyle
	Cursor       CursorState
	SavedCursor  CursorState
	wrapPending  bool
	savedWrap    bool
	cells        []Cell
}

func newScreen(width, height int, defaultStyle CellStyle) *Screen {
	s := &Screen{
		Width:        width,
		Height:       height,
		DefaultStyle: defaultStyle,
		CurrentStyle: defaultStyle,
		Cursor: CursorState{
			Visible: true,
		},
		SavedCursor: CursorState{
			Visible: true,
		},
		cells: make([]Cell, width*height),
	}
	for i := range s.cells {
		s.cells[i] = s.blankCell(defaultStyle)
	}
	return s
}

func (s *Screen) Clone() *Screen {
	clone := *s
	clone.cells = append([]Cell(nil), s.cells...)
	return &clone
}

func (s *Screen) Reset() {
	s.CurrentStyle = s.DefaultStyle
	s.Cursor = CursorState{Visible: true}
	s.SavedCursor = CursorState{Visible: true}
	s.wrapPending = false
	s.savedWrap = false
	for i := range s.cells {
		s.cells[i] = s.blankCell(s.DefaultStyle)
	}
}

func (s *Screen) blankCell(style CellStyle) Cell {
	return Cell{Style: style}
}

func (s *Screen) index(x, y int) int {
	return y*s.Width + x
}

func (s *Screen) inBounds(x, y int) bool {
	return x >= 0 && x < s.Width && y >= 0 && y < s.Height
}

func (s *Screen) Cell(x, y int) Cell {
	if !s.inBounds(x, y) {
		return s.blankCell(s.DefaultStyle)
	}
	return s.cells[s.index(x, y)]
}

func (s *Screen) setCell(x, y int, cell Cell) {
	if !s.inBounds(x, y) {
		return
	}
	s.cells[s.index(x, y)] = cell
}

func (s *Screen) blankAt(x, y int, style CellStyle) {
	if !s.inBounds(x, y) {
		return
	}
	s.cells[s.index(x, y)] = s.blankCell(style)
}

func (s *Screen) clearOccupied(x, y int) {
	if !s.inBounds(x, y) {
		return
	}
	idx := s.index(x, y)
	cell := s.cells[idx]
	if cell.Continuation {
		if x > 0 {
			s.blankAt(x-1, y, s.DefaultStyle)
		}
		s.blankAt(x, y, s.DefaultStyle)
		return
	}
	if x+1 < s.Width && s.cells[s.index(x+1, y)].Continuation {
		s.blankAt(x+1, y, s.DefaultStyle)
	}
	s.blankAt(x, y, s.DefaultStyle)
}

func (s *Screen) WriteRune(r rune) {
	width := runewidth.RuneWidth(r)
	if width <= 0 {
		s.appendToPrevious(string(r))
		return
	}
	if width > 2 {
		width = 1
	}
	if s.wrapPending {
		s.lineFeed()
		s.Cursor.X = 0
		s.wrapPending = false
	}
	if width == 2 && s.Cursor.X == s.Width-1 {
		s.lineFeed()
		s.Cursor.X = 0
	}
	if s.Cursor.Y >= s.Height {
		s.scrollUp(1)
		s.Cursor.Y = s.Height - 1
	}

	s.clearOccupied(s.Cursor.X, s.Cursor.Y)
	if width == 2 {
		s.clearOccupied(s.Cursor.X+1, s.Cursor.Y)
	}
	cell := Cell{Text: string(r), Style: s.CurrentStyle}
	s.setCell(s.Cursor.X, s.Cursor.Y, cell)
	if width == 2 {
		s.setCell(s.Cursor.X+1, s.Cursor.Y, Cell{Style: s.CurrentStyle, Continuation: true})
	}

	s.Cursor.X += width
	if s.Cursor.X >= s.Width {
		s.Cursor.X = s.Width - 1
		s.wrapPending = true
		return
	}
	s.wrapPending = false
}

func (s *Screen) appendToPrevious(text string) {
	x, y := s.Cursor.X-1, s.Cursor.Y
	if s.wrapPending {
		x = s.Width - 1
	}
	if x < 0 {
		x = s.Width - 1
		y--
	}
	if !s.inBounds(x, y) {
		return
	}
	cell := s.Cell(x, y)
	if cell.Continuation && x > 0 {
		x--
		cell = s.Cell(x, y)
	}
	cell.Text += text
	s.setCell(x, y, cell)
}

func (s *Screen) CarriageReturn() {
	s.Cursor.X = 0
	s.wrapPending = false
}

func (s *Screen) NewLine() {
	s.wrapPending = false
	s.Cursor.X = 0
	s.lineFeed()
}

func (s *Screen) lineFeed() {
	s.Cursor.Y++
	if s.Cursor.Y >= s.Height {
		s.scrollUp(1)
		s.Cursor.Y = s.Height - 1
	}
}

func (s *Screen) Backspace() {
	if s.wrapPending {
		s.wrapPending = false
		return
	}
	if s.Cursor.X > 0 {
		s.Cursor.X--
	}
}

func (s *Screen) Tab(width int) {
	spaces := width - (s.Cursor.X % width)
	if spaces == 0 {
		spaces = width
	}
	for i := 0; i < spaces; i++ {
		s.WriteRune(' ')
	}
}

func (s *Screen) MoveCursor(dx, dy int) {
	s.Cursor.X = clamp(s.Cursor.X+dx, 0, s.Width-1)
	s.Cursor.Y = clamp(s.Cursor.Y+dy, 0, s.Height-1)
	s.wrapPending = false
}

func (s *Screen) MoveCursorTo(row, col int) {
	s.Cursor.Y = clamp(row, 0, s.Height-1)
	s.Cursor.X = clamp(col, 0, s.Width-1)
	s.wrapPending = false
}

func (s *Screen) SaveCursor() {
	s.SavedCursor = s.Cursor
	s.savedWrap = s.wrapPending
}

func (s *Screen) RestoreCursor() {
	s.Cursor = s.SavedCursor
	s.wrapPending = s.savedWrap
}

func (s *Screen) EraseInDisplay(mode int) {
	switch mode {
	case 0:
		for y := s.Cursor.Y; y < s.Height; y++ {
			startX := 0
			if y == s.Cursor.Y {
				startX = s.Cursor.X
			}
			for x := startX; x < s.Width; x++ {
				s.blankAt(x, y, s.CurrentStyle)
			}
		}
	case 1:
		for y := 0; y <= s.Cursor.Y; y++ {
			endX := s.Width - 1
			if y == s.Cursor.Y {
				endX = s.Cursor.X
			}
			for x := 0; x <= endX; x++ {
				s.blankAt(x, y, s.CurrentStyle)
			}
		}
	default:
		for i := range s.cells {
			s.cells[i] = s.blankCell(s.CurrentStyle)
		}
	}
}

func (s *Screen) EraseInLine(mode int) {
	startX := 0
	endX := s.Width - 1
	switch mode {
	case 0:
		startX = s.Cursor.X
	case 1:
		endX = s.Cursor.X
	}
	for x := startX; x <= endX; x++ {
		s.blankAt(x, s.Cursor.Y, s.CurrentStyle)
	}
}

func (s *Screen) scrollUp(lines int) {
	if lines <= 0 || s.Height == 0 {
		return
	}
	if lines > s.Height {
		lines = s.Height
	}
	copy(s.cells, s.cells[lines*s.Width:])
	start := (s.Height - lines) * s.Width
	for i := start; i < len(s.cells); i++ {
		s.cells[i] = s.blankCell(s.DefaultStyle)
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
