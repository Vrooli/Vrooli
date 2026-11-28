package desktoplink

import "context"

// Repository hides the database engine from the link protocol and service.
type Repository interface {
	CreateAuthorization(context.Context, Authorization) error
	RedeemAuthorization(context.Context, string, string, string, string, string) (Link, error)
	StatusByLocal(context.Context, string, string, string) (Link, error)
	RevokeByLPBS(context.Context, string, string, string, string) (int64, error)
	RevokeByLocal(context.Context, string, string, string, string) (int64, error)
}
