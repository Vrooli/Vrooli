package instrument

import "context"

type Repository interface {
	Create(context.Context, Instrument) (Instrument, error)
	Get(context.Context, string) (Instrument, error)
}
