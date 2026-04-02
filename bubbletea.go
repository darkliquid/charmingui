package charmingui

import (
	"fmt"
	"image/draw"

	tea "charm.land/bubbletea/v2"
)

// UpdateResult pairs a Bubble Tea command with a rendered frame.
type UpdateResult struct {
	RenderResult
	Cmd tea.Cmd
}

// Dispatch runs the update command and expands any tea.Batch results into
// individual commands so embedders can schedule them without reimplementing
// Bubble Tea batch handling.
func (r UpdateResult) Dispatch(enqueue func(tea.Cmd), deliver func(tea.Msg)) {
	DispatchCmd(r.Cmd, enqueue, deliver)
}

// ModelAdapter renders a Bubble Tea model as full frames while keeping model state local.
type ModelAdapter struct {
	renderer *Renderer
	model    tea.Model
}

func NewModelAdapter(renderer *Renderer, model tea.Model) (*ModelAdapter, error) {
	if renderer == nil {
		return nil, fmt.Errorf("renderer cannot be nil")
	}
	if model == nil {
		return nil, fmt.Errorf("model cannot be nil")
	}
	return &ModelAdapter{renderer: renderer, model: model}, nil
}

func (a *ModelAdapter) Model() tea.Model {
	return a.model
}

func (a *ModelAdapter) Render(dst draw.Image) (RenderResult, error) {
	return a.renderer.RenderModel(dst, a.model)
}

func (a *ModelAdapter) Update(msg tea.Msg, dst draw.Image) (UpdateResult, error) {
	next, cmd := a.model.Update(msg)
	a.model = next
	result, err := a.renderer.RenderModel(dst, a.model)
	if err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{RenderResult: result, Cmd: cmd}, nil
}

// DispatchCmd runs a Bubble Tea command and expands tea.Batch results into
// individual commands while forwarding all other messages to deliver.
func DispatchCmd(cmd tea.Cmd, enqueue func(tea.Cmd), deliver func(tea.Msg)) {
	if cmd == nil {
		return
	}
	if enqueue == nil {
		enqueue = func(tea.Cmd) {}
	}
	if deliver == nil {
		deliver = func(tea.Msg) {}
	}

	msg := cmd()
	switch msg := msg.(type) {
	case nil:
		return
	case tea.BatchMsg:
		for _, next := range msg {
			if next != nil {
				enqueue(next)
			}
		}
	default:
		deliver(msg)
	}
}
