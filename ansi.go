package charmingui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"unicode/utf8"
)

func parseANSI(screen *Screen, input string, tabWidth int) error {
	for i := 0; i < len(input); {
		switch input[i] {
		case 0x1b:
			if i+1 >= len(input) {
				i++
				continue
			}
			next := input[i+1]
			switch next {
			case '[':
				end := i + 2
				for end < len(input) && !isCSIFinalByte(input[end]) {
					end++
				}
				if end >= len(input) {
					return fmt.Errorf("unterminated CSI sequence")
				}
				if err := handleCSI(screen, input[i+2:end], input[end]); err != nil {
					return err
				}
				i = end + 1
			case '7':
				screen.SaveCursor()
				i += 2
			case '8':
				screen.RestoreCursor()
				i += 2
			default:
				i += 2
			}
		case '\r':
			screen.CarriageReturn()
			i++
		case '\n':
			screen.NewLine()
			i++
		case '\b':
			screen.Backspace()
			i++
		case '\t':
			screen.Tab(tabWidth)
			i++
		default:
			r, size := utf8.DecodeRuneInString(input[i:])
			if r == utf8.RuneError && size == 1 {
				i++
				continue
			}
			screen.WriteRune(r)
			i += size
		}
	}
	return nil
}

func isCSIFinalByte(b byte) bool {
	return b >= 0x40 && b <= 0x7e
}

func handleCSI(screen *Screen, params string, final byte) error {
	private := strings.HasPrefix(params, "?")
	if private {
		params = strings.TrimPrefix(params, "?")
	}
	raw := splitParams(params)

	switch final {
	case 'A':
		screen.MoveCursor(0, -paramOr(raw, 0, 1))
	case 'B':
		screen.MoveCursor(0, paramOr(raw, 0, 1))
	case 'C':
		screen.MoveCursor(paramOr(raw, 0, 1), 0)
	case 'D':
		screen.MoveCursor(-paramOr(raw, 0, 1), 0)
	case 'G':
		screen.MoveCursorTo(screen.Cursor.Y, paramOr(raw, 0, 1)-1)
	case 'H', 'f':
		row := paramOr(raw, 0, 1) - 1
		col := paramOr(raw, 1, 1) - 1
		screen.MoveCursorTo(row, col)
	case 'd':
		screen.MoveCursorTo(paramOr(raw, 0, 1)-1, screen.Cursor.X)
	case 'J':
		screen.EraseInDisplay(paramOr(raw, 0, 0))
	case 'K':
		screen.EraseInLine(paramOr(raw, 0, 0))
	case 'm':
		applySGR(screen, raw)
	case 's':
		screen.SaveCursor()
	case 'u':
		screen.RestoreCursor()
	case 'h', 'l':
		if !private {
			return nil
		}
		enable := final == 'h'
		for _, p := range raw {
			switch p {
			case "25", "":
				screen.Cursor.Visible = enable
			}
		}
	}
	return nil
}

func splitParams(params string) []string {
	if params == "" {
		return []string{""}
	}
	return strings.Split(params, ";")
}

func paramOr(params []string, idx, fallback int) int {
	if idx >= len(params) || params[idx] == "" {
		return fallback
	}
	v, err := strconv.Atoi(params[idx])
	if err != nil {
		return fallback
	}
	return v
}

func applySGR(screen *Screen, params []string) {
	if len(params) == 0 {
		screen.CurrentStyle = screen.DefaultStyle
		return
	}
	for i := 0; i < len(params); i++ {
		p := params[i]
		if p == "" {
			screen.CurrentStyle = screen.DefaultStyle
			continue
		}
		code, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		switch {
		case code == 0:
			screen.CurrentStyle = screen.DefaultStyle
		case code == 1:
			screen.CurrentStyle.Bold = true
		case code == 2:
			screen.CurrentStyle.Faint = true
		case code == 3:
			screen.CurrentStyle.Italic = true
		case code == 4:
			screen.CurrentStyle.Underline = true
		case code == 7:
			screen.CurrentStyle.Inverse = true
		case code == 9:
			screen.CurrentStyle.Strikethrough = true
		case code == 22:
			screen.CurrentStyle.Bold = false
			screen.CurrentStyle.Faint = false
		case code == 23:
			screen.CurrentStyle.Italic = false
		case code == 24:
			screen.CurrentStyle.Underline = false
		case code == 27:
			screen.CurrentStyle.Inverse = false
		case code == 29:
			screen.CurrentStyle.Strikethrough = false
		case code == 39:
			screen.CurrentStyle.FG = screen.DefaultStyle.FG
		case code == 49:
			screen.CurrentStyle.BG = screen.DefaultStyle.BG
		case code >= 30 && code <= 37:
			screen.CurrentStyle.FG = ansiColor(code - 30)
		case code >= 40 && code <= 47:
			screen.CurrentStyle.BG = ansiColor(code - 40)
		case code >= 90 && code <= 97:
			screen.CurrentStyle.FG = ansiColor(code - 90 + 8)
		case code >= 100 && code <= 107:
			screen.CurrentStyle.BG = ansiColor(code - 100 + 8)
		case code == 38 || code == 48:
			if i+1 >= len(params) {
				continue
			}
			mode := paramOr(params, i+1, 0)
			switch mode {
			case 5:
				if i+2 >= len(params) {
					continue
				}
				col := xtermColor(paramOr(params, i+2, 0))
				if code == 38 {
					screen.CurrentStyle.FG = col
				} else {
					screen.CurrentStyle.BG = col
				}
				i += 2
			case 2:
				if i+4 >= len(params) {
					continue
				}
				col := color.RGBA{
					R: uint8(paramOr(params, i+2, 0)),
					G: uint8(paramOr(params, i+3, 0)),
					B: uint8(paramOr(params, i+4, 0)),
					A: 0xff,
				}
				if code == 38 {
					screen.CurrentStyle.FG = col
				} else {
					screen.CurrentStyle.BG = col
				}
				i += 4
			}
		}
	}
}

func ansiColor(idx int) color.RGBA {
	palette := [16]color.RGBA{
		{A: 0xff},
		{R: 0xcd, G: 0x31, B: 0x31, A: 0xff},
		{R: 0x0d, G: 0xbc, B: 0x79, A: 0xff},
		{R: 0xe5, G: 0xe5, B: 0x10, A: 0xff},
		{R: 0x24, G: 0x73, B: 0xc1, A: 0xff},
		{R: 0xbc, G: 0x3f, B: 0xbc, A: 0xff},
		{R: 0x11, G: 0xa8, B: 0xcd, A: 0xff},
		{R: 0xe5, G: 0xe5, B: 0xe5, A: 0xff},
		{R: 0x66, G: 0x66, B: 0x66, A: 0xff},
		{R: 0xf1, G: 0x4c, B: 0x4c, A: 0xff},
		{R: 0x23, G: 0xd1, B: 0x8b, A: 0xff},
		{R: 0xf5, G: 0xf5, B: 0x43, A: 0xff},
		{R: 0x3b, G: 0x8e, B: 0xea, A: 0xff},
		{R: 0xd6, G: 0x70, B: 0xd6, A: 0xff},
		{R: 0x29, G: 0xb8, B: 0xdb, A: 0xff},
		{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}
	if idx < 0 || idx >= len(palette) {
		return palette[0]
	}
	return palette[idx]
}

func xtermColor(code int) color.RGBA {
	if code < 16 {
		return ansiColor(code)
	}
	if code >= 232 {
		shade := uint8((code-232)*10 + 8)
		return color.RGBA{R: shade, G: shade, B: shade, A: 0xff}
	}
	code -= 16
	r := uint8(code / 36)
	g := uint8((code % 36) / 6)
	b := uint8(code % 6)
	return color.RGBA{
		R: cubeComponent(r),
		G: cubeComponent(g),
		B: cubeComponent(b),
		A: 0xff,
	}
}

func cubeComponent(v uint8) uint8 {
	if v == 0 {
		return 0
	}
	return 55 + (v * 40)
}
