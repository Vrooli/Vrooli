package integrations

import "context"

type Repository interface {
	List(context.Context) ([]Connection, error)
	CreateFixture(context.Context, string) (Connection, error)
	Sync(context.Context, SyncInput) (Connection, error)
	Disconnect(context.Context, SyncInput) (Connection, error)
}
