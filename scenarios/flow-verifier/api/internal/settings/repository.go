package settings

import "context"

// Repository is the persistence seam for the single 'local'-principal
// settings row. The SQLite implementation lives in sqlite.go; tests
// substitute fakes without reaching inside the struct.
//
// Get returns DefaultSettings (with PrincipalID set, UpdatedAt zero)
// when the row does not yet exist — handlers and the CLI never need a
// branch for "first time the user opened the app".
type Repository interface {
	Get(ctx context.Context, principalID string) (Settings, error)
	Upsert(ctx context.Context, s Settings) (Settings, error)
}
