package charmingui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRenderViewExactWidthLinesDoNotSkipRows(t *testing.T) {
	r, err := New(Config{Columns: 4, Rows: 3})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	view := tea.NewView(strings.Join([]string{
		"abcd",
		"efgh",
		"ijkl",
	}, "\n"))

	if _, err := r.RenderView(nil, view); err != nil {
		t.Fatalf("RenderView: %v", err)
	}

	for row, want := range []string{"abcd", "efgh", "ijkl"} {
		if got := testRowText(r.screen, row); got != want {
			t.Fatalf("row %d = %q, want %q", row, got, want)
		}
	}
}

func testRowText(screen *Screen, row int) string {
	var b strings.Builder
	for col := 0; col < screen.Width; col++ {
		cell := screen.Cell(col, row)
		if cell.Continuation {
			continue
		}
		if cell.Text == "" {
			b.WriteByte(' ')
			continue
		}
		b.WriteString(cell.Text)
	}
	return b.String()
}
