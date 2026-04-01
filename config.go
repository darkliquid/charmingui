package charmingui

import (
	"fmt"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// Config controls how terminal content is rasterized to an image-backed surface.
type Config struct {
	Columns     int
	Rows        int
	Face        font.Face
	BoldFace    font.Face
	DefaultFG   color.Color
	DefaultBG   color.Color
	CursorColor color.Color
	TabWidth    int
}

type normalizedConfig struct {
	Columns     int
	Rows        int
	Face        font.Face
	BoldFace    font.Face
	DefaultFG   color.RGBA
	DefaultBG   color.RGBA
	CursorColor color.RGBA
	TabWidth    int
}

func normalizeConfig(cfg Config) (normalizedConfig, error) {
	if cfg.Columns <= 0 {
		cfg.Columns = 80
	}
	if cfg.Rows <= 0 {
		cfg.Rows = 24
	}
	if cfg.Face == nil {
		cfg.Face = basicfont.Face7x13
	}
	if cfg.BoldFace == nil {
		cfg.BoldFace = cfg.Face
	}
	if cfg.DefaultFG == nil {
		cfg.DefaultFG = color.RGBA{R: 0xea, G: 0xea, B: 0xea, A: 0xff}
	}
	if cfg.DefaultBG == nil {
		cfg.DefaultBG = color.RGBA{A: 0xff}
	}
	if cfg.CursorColor == nil {
		cfg.CursorColor = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	}
	if cfg.TabWidth <= 0 {
		cfg.TabWidth = 8
	}
	if cfg.Columns <= 0 || cfg.Rows <= 0 {
		return normalizedConfig{}, fmt.Errorf("columns and rows must be positive")
	}

	return normalizedConfig{
		Columns:     cfg.Columns,
		Rows:        cfg.Rows,
		Face:        cfg.Face,
		BoldFace:    cfg.BoldFace,
		DefaultFG:   toRGBA(cfg.DefaultFG),
		DefaultBG:   toRGBA(cfg.DefaultBG),
		CursorColor: toRGBA(cfg.CursorColor),
		TabWidth:    cfg.TabWidth,
	}, nil
}

func (c normalizedConfig) defaultStyle() CellStyle {
	return CellStyle{FG: c.DefaultFG, BG: c.DefaultBG}
}

func toRGBA(c color.Color) color.RGBA {
	if c == nil {
		return color.RGBA{}
	}
	return color.RGBAModel.Convert(c).(color.RGBA)
}
