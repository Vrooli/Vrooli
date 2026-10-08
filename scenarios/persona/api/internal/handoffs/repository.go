package handoffs

import "context"

type Repository interface {
	Create(context.Context, Handoff) (Handoff, error)
	Get(context.Context, string) (Handoff, error)
	List(context.Context, string, int) ([]Handoff, error)
	UpdateState(context.Context, string, State, string, string) (Handoff, error)
	SetRelayState(context.Context, string, string) (Handoff, error)
}
