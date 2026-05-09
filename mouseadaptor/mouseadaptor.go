// Package mouseadaptor converts pixel-space mouse state into Bubble Tea mouse
// messages, translating pixel coordinates into terminal character-cell
// coordinates using the cell dimensions reported by a charmingui Renderer.
package mouseadaptor

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/darkliquid/charmingui"
)

// MouseState describes raw mouse input captured in pixel (screen) coordinates.
// Buttons are true when held. Wheel fields are transient: set them to true for
// a single call to Convert to fire a wheel event, then clear them.
type MouseState struct {
	// X and Y are pixel coordinates within the rendered surface.
	X, Y int

	// Primary buttons.
	Left   bool
	Middle bool
	Right  bool

	// Extra buttons.
	Backward bool
	Forward  bool

	// Wheel scrolling (transient per-frame events).
	WheelUp    bool
	WheelDown  bool
	WheelLeft  bool
	WheelRight bool

	// Modifier keys held during the mouse event.
	Shift bool
	Alt   bool
	Ctrl  bool
}

// Adaptor converts pixel-space mouse state into Bubble Tea mouse messages.
// It tracks the previous state to detect button press, release and motion
// transitions, mapping pixel coordinates to character-cell coordinates.
type Adaptor struct {
	cellWidth  int
	cellHeight int
	prev       MouseState
}

// New creates an Adaptor using explicit cell dimensions. Both cellWidth and
// cellHeight must be positive.
func New(cellWidth, cellHeight int) (*Adaptor, error) {
	if cellWidth <= 0 || cellHeight <= 0 {
		return nil, fmt.Errorf("mouseadaptor: cell dimensions must be positive, got %dx%d", cellWidth, cellHeight)
	}
	return &Adaptor{cellWidth: cellWidth, cellHeight: cellHeight}, nil
}

// NewFromRenderer creates an Adaptor whose cell dimensions match those of r.
func NewFromRenderer(r *charmingui.Renderer) (*Adaptor, error) {
	if r == nil {
		return nil, fmt.Errorf("mouseadaptor: renderer cannot be nil")
	}
	size := r.CellSize()
	return New(size.X, size.Y)
}

// toCellCoords maps pixel coordinates to character-cell (column, row) indices.
func (a *Adaptor) toCellCoords(pixelX, pixelY int) (int, int) {
	col := pixelX / a.cellWidth
	row := pixelY / a.cellHeight
	if col < 0 {
		col = 0
	}
	if row < 0 {
		row = 0
	}
	return col, row
}

func (a *Adaptor) buildMod(s MouseState) tea.KeyMod {
	var mod tea.KeyMod
	if s.Shift {
		mod |= tea.ModShift
	}
	if s.Alt {
		mod |= tea.ModAlt
	}
	if s.Ctrl {
		mod |= tea.ModCtrl
	}
	return mod
}

func makeMouse(col, row int, btn tea.MouseButton, mod tea.KeyMod) tea.Mouse {
	return tea.Mouse{X: col, Y: row, Button: btn, Mod: mod}
}

// heldButton returns the highest-priority button that is currently held.
func heldButton(s MouseState) tea.MouseButton {
	switch {
	case s.Left:
		return tea.MouseLeft
	case s.Middle:
		return tea.MouseMiddle
	case s.Right:
		return tea.MouseRight
	case s.Backward:
		return tea.MouseBackward
	case s.Forward:
		return tea.MouseForward
	default:
		return tea.MouseNone
	}
}

// Convert compares state against the previous call and returns zero or more
// Bubble Tea mouse messages that reflect any changes. The order of emitted
// messages per call is: motion (if the cell position changed), button
// press/release transitions, then wheel events.
//
// Coordinates in the returned messages are character-cell indices (column,
// row), not pixels.
func (a *Adaptor) Convert(state MouseState) []tea.Msg {
	prev := a.prev
	a.prev = state

	col, row := a.toCellCoords(state.X, state.Y)
	prevCol, prevRow := a.toCellCoords(prev.X, prev.Y)
	mod := a.buildMod(state)

	mouse := func(btn tea.MouseButton) tea.Mouse {
		return makeMouse(col, row, btn, mod)
	}

	var msgs []tea.Msg

	// Motion: emit when character-cell position changes.
	if col != prevCol || row != prevRow {
		msgs = append(msgs, tea.MouseMotionMsg(mouse(heldButton(state))))
	}

	// Button press and release transitions.
	type buttonEntry struct {
		cur, was bool
		btn      tea.MouseButton
	}
	buttons := [...]buttonEntry{
		{state.Left, prev.Left, tea.MouseLeft},
		{state.Middle, prev.Middle, tea.MouseMiddle},
		{state.Right, prev.Right, tea.MouseRight},
		{state.Backward, prev.Backward, tea.MouseBackward},
		{state.Forward, prev.Forward, tea.MouseForward},
	}
	for _, b := range buttons {
		switch {
		case b.cur && !b.was:
			msgs = append(msgs, tea.MouseClickMsg(mouse(b.btn)))
		case !b.cur && b.was:
			msgs = append(msgs, tea.MouseReleaseMsg(mouse(b.btn)))
		}
	}

	// Wheel events are transient: emit whenever the field is true.
	type wheelEntry struct {
		active bool
		btn    tea.MouseButton
	}
	wheels := [...]wheelEntry{
		{state.WheelUp, tea.MouseWheelUp},
		{state.WheelDown, tea.MouseWheelDown},
		{state.WheelLeft, tea.MouseWheelLeft},
		{state.WheelRight, tea.MouseWheelRight},
	}
	for _, w := range wheels {
		if w.active {
			msgs = append(msgs, tea.MouseWheelMsg(mouse(w.btn)))
		}
	}

	return msgs
}
