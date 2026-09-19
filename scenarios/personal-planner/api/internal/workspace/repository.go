package workspace

import "context"

type Repository interface {
	Get(context.Context) (Profile, error)
	Update(context.Context, UpdateInput) (Profile, error)
	ListAvailability(context.Context) (Availability, error)
	ReplaceAvailability(context.Context, AvailabilityInput) (Availability, error)
}
