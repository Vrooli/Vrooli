package channels

import "context"

type Repository interface {
	Create(context.Context, Channel) (Channel, error)
	List(context.Context, string) ([]Channel, error)
	Get(context.Context, string) (Channel, error)
}
