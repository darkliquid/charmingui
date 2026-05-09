package mouseadaptor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func newAdaptor(t *testing.T) *Adaptor {
	t.Helper()
	a, err := New(8, 16)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a
}

func assertMsgTypes(t *testing.T, msgs []tea.Msg, want ...string) {
	t.Helper()
	if len(msgs) != len(want) {
		t.Fatalf("got %d messages, want %d: %v", len(msgs), len(want), msgs)
	}
	for i, msg := range msgs {
		var got string
		switch msg.(type) {
		case tea.MouseClickMsg:
			got = "click"
		case tea.MouseReleaseMsg:
			got = "release"
		case tea.MouseMotionMsg:
			got = "motion"
		case tea.MouseWheelMsg:
			got = "wheel"
		default:
			got = "unknown"
		}
		if got != want[i] {
			t.Errorf("msg[%d]: got %q, want %q", i, got, want[i])
		}
	}
}

func TestNew_InvalidDimensions(t *testing.T) {
	cases := [][2]int{{0, 16}, {8, 0}, {-1, 16}, {8, -1}}
	for _, c := range cases {
		_, err := New(c[0], c[1])
		if err == nil {
			t.Errorf("New(%d, %d) expected error, got nil", c[0], c[1])
		}
	}
}

func TestNew_NilRenderer(t *testing.T) {
	_, err := NewFromRenderer(nil)
	if err == nil {
		t.Fatal("expected error for nil renderer")
	}
}

func TestConvert_NoChange(t *testing.T) {
	a := newAdaptor(t)
	msgs := a.Convert(MouseState{X: 0, Y: 0})
	if len(msgs) != 0 {
		t.Fatalf("expected no messages on first call with no buttons, got %d", len(msgs))
	}
	msgs = a.Convert(MouseState{X: 0, Y: 0})
	if len(msgs) != 0 {
		t.Fatalf("expected no messages when state unchanged, got %d", len(msgs))
	}
}

func TestConvert_ButtonClick(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 8, Y: 16}) // establish baseline at cell (1,1)

	msgs := a.Convert(MouseState{X: 8, Y: 16, Left: true})
	assertMsgTypes(t, msgs, "click")

	click := msgs[0].(tea.MouseClickMsg)
	if click.X != 1 || click.Y != 1 {
		t.Errorf("expected cell (1,1), got (%d,%d)", click.X, click.Y)
	}
	if click.Button != tea.MouseLeft {
		t.Errorf("expected MouseLeft, got %v", click.Button)
	}
}

func TestConvert_ButtonRelease(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0, Left: true})

	msgs := a.Convert(MouseState{X: 0, Y: 0, Left: false})
	assertMsgTypes(t, msgs, "release")

	rel := msgs[0].(tea.MouseReleaseMsg)
	if rel.Button != tea.MouseLeft {
		t.Errorf("expected MouseLeft release, got %v", rel.Button)
	}
}

func TestConvert_MotionNoButtton(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0})

	// Move to a different cell.
	msgs := a.Convert(MouseState{X: 8, Y: 0})
	assertMsgTypes(t, msgs, "motion")

	m := msgs[0].(tea.MouseMotionMsg)
	if m.Button != tea.MouseNone {
		t.Errorf("expected MouseNone for free motion, got %v", m.Button)
	}
	if m.X != 1 || m.Y != 0 {
		t.Errorf("expected cell (1,0), got (%d,%d)", m.X, m.Y)
	}
}

func TestConvert_MotionWithButton(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0, Left: true})

	msgs := a.Convert(MouseState{X: 16, Y: 32, Left: true})
	assertMsgTypes(t, msgs, "motion")

	m := msgs[0].(tea.MouseMotionMsg)
	if m.Button != tea.MouseLeft {
		t.Errorf("expected MouseLeft for drag motion, got %v", m.Button)
	}
	if m.X != 2 || m.Y != 2 {
		t.Errorf("expected cell (2,2), got (%d,%d)", m.X, m.Y)
	}
}

func TestConvert_MotionAndClick(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0})

	// Cursor moves to a new cell AND button is pressed in the same frame.
	msgs := a.Convert(MouseState{X: 8, Y: 0, Left: true})
	assertMsgTypes(t, msgs, "motion", "click")
}

func TestConvert_WheelEvents(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0})

	msgs := a.Convert(MouseState{X: 0, Y: 0, WheelUp: true})
	assertMsgTypes(t, msgs, "wheel")
	if msgs[0].(tea.MouseWheelMsg).Button != tea.MouseWheelUp {
		t.Error("expected WheelUp")
	}

	msgs = a.Convert(MouseState{X: 0, Y: 0, WheelDown: true, WheelLeft: true})
	assertMsgTypes(t, msgs, "wheel", "wheel")
}

func TestConvert_ModifierKeys(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0})

	msgs := a.Convert(MouseState{X: 0, Y: 0, Left: true, Shift: true, Ctrl: true})
	assertMsgTypes(t, msgs, "click")

	click := msgs[0].(tea.MouseClickMsg)
	if click.Mod&tea.ModShift == 0 {
		t.Error("expected ModShift set")
	}
	if click.Mod&tea.ModCtrl == 0 {
		t.Error("expected ModCtrl set")
	}
	if click.Mod&tea.ModAlt != 0 {
		t.Error("expected ModAlt not set")
	}
}

func TestConvert_PixelToCell(t *testing.T) {
	// cellWidth=8, cellHeight=16 → pixel (23, 47) → cell (2, 2)
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0})

	msgs := a.Convert(MouseState{X: 23, Y: 47, Left: true})
	assertMsgTypes(t, msgs, "motion", "click")
	click := msgs[1].(tea.MouseClickMsg)
	if click.X != 2 || click.Y != 2 {
		t.Errorf("pixel (23,47) with cell 8x16: expected cell (2,2), got (%d,%d)", click.X, click.Y)
	}
}

func TestConvert_SubCellMotionIgnored(t *testing.T) {
	// Moving within the same cell should not produce motion.
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0})

	msgs := a.Convert(MouseState{X: 3, Y: 7}) // still cell (0,0)
	if len(msgs) != 0 {
		t.Fatalf("expected no messages for sub-cell motion, got %d", len(msgs))
	}
}

func TestConvert_NegativeCoordsClamped(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 8, Y: 16})

	// Negative coords should clamp to cell (0,0).
	msgs := a.Convert(MouseState{X: -5, Y: -10, Left: true})
	assertMsgTypes(t, msgs, "motion", "click")
	click := msgs[1].(tea.MouseClickMsg)
	if click.X != 0 || click.Y != 0 {
		t.Errorf("expected clamped cell (0,0), got (%d,%d)", click.X, click.Y)
	}
}

func TestConvert_MultipleButtonsIndependent(t *testing.T) {
	a := newAdaptor(t)
	a.Convert(MouseState{X: 0, Y: 0, Left: true})

	// Right button pressed while left is still held – no left click event.
	msgs := a.Convert(MouseState{X: 0, Y: 0, Left: true, Right: true})
	assertMsgTypes(t, msgs, "click")
	if msgs[0].(tea.MouseClickMsg).Button != tea.MouseRight {
		t.Error("expected right button click")
	}
}
