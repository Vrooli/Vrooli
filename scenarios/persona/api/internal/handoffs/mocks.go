package handoffs

import "context"

type FakeRepository struct {
	CreateFunc        func(context.Context, Handoff) (Handoff, error)
	GetFunc           func(context.Context, string) (Handoff, error)
	ListFunc          func(context.Context, string, int) ([]Handoff, error)
	UpdateStateFunc   func(context.Context, string, State, string, string) (Handoff, error)
	SetRelayStateFunc func(context.Context, string, string) (Handoff, error)
}

func (f FakeRepository) Create(ctx context.Context, h Handoff) (Handoff, error) {
	return f.CreateFunc(ctx, h)
}

func (f FakeRepository) Get(ctx context.Context, id string) (Handoff, error) {
	return f.GetFunc(ctx, id)
}

func (f FakeRepository) List(ctx context.Context, id string, limit int) ([]Handoff, error) {
	return f.ListFunc(ctx, id, limit)
}

func (f FakeRepository) UpdateState(ctx context.Context, id string, state State, actor, reason string) (Handoff, error) {
	return f.UpdateStateFunc(ctx, id, state, actor, reason)
}

func (f FakeRepository) SetRelayState(ctx context.Context, id, relayState string) (Handoff, error) {
	if f.SetRelayStateFunc == nil {
		return f.Get(ctx, id)
	}
	return f.SetRelayStateFunc(ctx, id, relayState)
}
