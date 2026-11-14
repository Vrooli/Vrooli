package approval

import "context"

type Repository interface {
	Create(context.Context, Request) (Request, error)
	Get(context.Context, string) (Request, error)
	List(context.Context, Status, string) ([]Request, error)
	Resolve(context.Context, string, Status, string, string) (Request, error)
	RecordRelay(context.Context, RelayAttempt) error
	ListRelayAttempts(context.Context, string) ([]RelayAttempt, error)
}
