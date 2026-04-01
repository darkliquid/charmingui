package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"

	"github.com/darkliquid/charmingui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	demoColumns = 80
	demoRows    = 18
	demoScale   = 2
)

type demoModel struct {
	input   textinput.Model
	initCmd tea.Cmd
}

func newDemoModel(cols int) demoModel {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "Type into the Bubble Tea input"
	input.SetWidth(max(16, cols-6))
	input.SetValue("Hello, Ebiten")
	input.SetSuggestions([]string{"Bubble Tea", "Bubbles", "Ebiten", "CharmingUI"})
	input.ShowSuggestions = true
	input.SetVirtualCursor(true)
	initCmd := input.Focus()
	return demoModel{input: input, initCmd: initCmd}
}

func (m demoModel) Init() tea.Cmd {
	return m.initCmd
}

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.input.SetWidth(max(16, msg.Width-6))
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m demoModel) View() tea.View {
	content := strings.Join([]string{
		"\x1b[1;36mCharmingUI + Ebiten demo\x1b[0m",
		"",
		"This adapts the Bubbles v2 textinput example into an Ebiten window.",
		"The Bubble Tea view is rasterized by charmingui and drawn onto an *ebiten.Image.",
		"",
		m.input.View(),
		"",
		fmt.Sprintf("\x1b[2mCurrent value:\x1b[0m %s", displayValue(m.input.Value())),
		"",
		"\x1b[2mControls: type text, use arrows/home/end, Tab for suggestions, Backspace/Delete, Esc or Ctrl+C to quit.\x1b[0m",
	}, "\n")

	view := tea.NewView(content)
	view.Cursor = m.input.Cursor()
	return view
}

type game struct {
	renderer *charmingui.Renderer
	adapter  *charmingui.ModelAdapter
	frame    *image.RGBA
	surface  *ebiten.Image
	msgs     chan tea.Msg
	runes    []rune
	keys     []ebiten.Key
}

func newGame() (*game, error) {
	renderer, err := charmingui.New(charmingui.Config{
		Columns:     demoColumns,
		Rows:        demoRows,
		DefaultFG:   color.RGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff},
		DefaultBG:   color.RGBA{R: 0x10, G: 0x14, B: 0x1a, A: 0xff},
		CursorColor: color.RGBA{R: 0xff, G: 0xd8, B: 0x66, A: 0xff},
	})
	if err != nil {
		return nil, err
	}

	adapter, err := charmingui.NewModelAdapter(renderer, newDemoModel(demoColumns))
	if err != nil {
		return nil, err
	}

	g := &game{
		renderer: renderer,
		adapter:  adapter,
		surface:  ebiten.NewImage(renderer.SurfaceBounds().Dx(), renderer.SurfaceBounds().Dy()),
		msgs:     make(chan tea.Msg, 32),
	}
	if err := g.updateModel(tea.WindowSizeMsg{Width: demoColumns, Height: demoRows}); err != nil {
		return nil, err
	}
	g.enqueueCmd(g.adapter.Model().Init())
	if err := g.renderCurrent(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *game) Update() error {
	if err := g.drainMsgs(); err != nil {
		return err
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) && controlPressed() {
		return g.updateModel(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c", Mod: tea.ModCtrl}))
	}

	g.runes = ebiten.AppendInputChars(g.runes[:0])
	for _, r := range g.runes {
		if err := g.updateModel(tea.KeyPressMsg(tea.Key{Code: r, ShiftedCode: r, Text: string(r)})); err != nil {
			return err
		}
	}

	g.keys = inpututil.AppendJustPressedKeys(g.keys[:0])
	for _, key := range g.keys {
		msg, ok := mapSpecialKey(key)
		if !ok {
			continue
		}
		if err := g.updateModel(msg); err != nil {
			return err
		}
	}

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0x08, G: 0x0b, B: 0x10, A: 0xff})
	ops := &ebiten.DrawImageOptions{}
	ops.GeoM.Scale(demoScale, demoScale)
	screen.DrawImage(g.surface, ops)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	bounds := g.renderer.SurfaceBounds()
	return bounds.Dx() * demoScale, bounds.Dy() * demoScale
}

func (g *game) drainMsgs() error {
	for {
		select {
		case msg := <-g.msgs:
			if err := g.updateModel(msg); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func (g *game) updateModel(msg tea.Msg) error {
	if _, ok := msg.(tea.QuitMsg); ok {
		return ebiten.Termination
	}
	result, err := g.adapter.Update(msg, nil)
	if err != nil {
		return err
	}
	if err := g.uploadFrame(result.Image); err != nil {
		return err
	}
	g.enqueueCmd(result.Cmd)
	return nil
}

func (g *game) renderCurrent() error {
	result, err := g.adapter.Render(nil)
	if err != nil {
		return err
	}
	return g.uploadFrame(result.Image)
}

func (g *game) uploadFrame(img image.Image) error {
	rgba, ok := img.(*image.RGBA)
	if !ok {
		return fmt.Errorf("unexpected frame type %T", img)
	}
	g.frame = rgba
	g.surface.WritePixels(rgba.Pix)
	return nil
}

func (g *game) enqueueCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		msg := cmd()
		if msg == nil {
			return
		}
		g.msgs <- msg
	}()
}

func mapSpecialKey(key ebiten.Key) (tea.Msg, bool) {
	switch key {
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), true
	case ebiten.KeyTab:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}), true
	case ebiten.KeyBackspace:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}), true
	case ebiten.KeyDelete:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDelete}), true
	case ebiten.KeyArrowLeft:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}), true
	case ebiten.KeyArrowRight:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}), true
	case ebiten.KeyArrowUp:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}), true
	case ebiten.KeyArrowDown:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}), true
	case ebiten.KeyHome:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyHome}), true
	case ebiten.KeyEnd:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}), true
	case ebiten.KeyEscape:
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}), true
	default:
		return nil, false
	}
}

func controlPressed() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)
}

func displayValue(value string) string {
	if value == "" {
		return "<empty>"
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	game, err := newGame()
	if err != nil {
		log.Fatal(err)
	}

	bounds := game.renderer.SurfaceBounds()
	ebiten.SetWindowTitle("CharmingUI + Ebiten")
	ebiten.SetWindowSize(bounds.Dx()*demoScale, bounds.Dy()*demoScale)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
