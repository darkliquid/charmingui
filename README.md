# charmingui

`charmingui` is a Go library for rendering Bubble Tea views and ANSI terminal output onto `image.Image` surfaces.

It is useful when you want terminal-style UI in a non-terminal environment, such as:

- embedding a Bubble Tea interface inside a game or graphical app
- turning ANSI output into an image for previews, exports, or snapshots
- reusing Bubble Tea models where you control drawing yourself

The package exposes a `Renderer` that rasterizes terminal cells into pixels, plus a `ModelAdapter` for driving a Bubble Tea model over repeated updates.

## Minimal example

```go
package main

import (
	"image/png"
	"log"
	"os"

	"charm.land/bubbletea/v2"
	"github.com/darkliquid/charmingui"
)

type model struct{}

func (m model) Init() tea.Cmd { return nil }
func (m model) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m model) View() tea.View {
	return tea.NewView("Hello from CharmingUI!\n\x1b[32mRendered to an image.\x1b[0m")
}

func main() {
	renderer, err := charmingui.New(charmingui.Config{
		Columns: 40,
		Rows:    4,
	})
	if err != nil {
		log.Fatal(err)
	}

	result, err := renderer.RenderModel(nil, model{})
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Create("frame.png")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if err := png.Encode(file, result.Image); err != nil {
		log.Fatal(err)
	}
}
```

That creates a renderer, renders a Bubble Tea model, and writes the resulting frame to `frame.png`.

## Main APIs

- `charmingui.New(Config)` creates a renderer with a terminal grid size, font, and colors.
- `(*Renderer).RenderModel` renders a Bubble Tea model to an image.
- `(*Renderer).RenderView` renders a `tea.View` directly.
- `(*Renderer).RenderFrame` renders a full ANSI frame.
- `(*Renderer).RenderStream` applies ANSI output incrementally to the current virtual screen.
- `NewModelAdapter` helps keep a Bubble Tea model updated and rendered in apps with their own event loop.

See `cmd/ebiten-demo` for a full example that embeds Bubble Tea inside an Ebiten application.
