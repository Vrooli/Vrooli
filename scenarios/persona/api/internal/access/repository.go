package access

import "context"

type Repository interface {
	CreateGrant(context.Context, Grant) (Grant, error)
	ListGrants(context.Context, string) ([]Grant, error)
	GetGrant(context.Context, string) (Grant, error)
	RemoveGrant(context.Context, string) error
}
