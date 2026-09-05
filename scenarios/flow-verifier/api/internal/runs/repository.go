package runs

import "context"

// Repository is the persistence seam for verification history. The
// SQLite implementation lives in sqlite.go; tests substitute fakes
// without reaching inside the struct.
type Repository interface {
	Insert(ctx context.Context, run Run) (Run, error)
	Get(ctx context.Context, id string) (Run, error)
	List(ctx context.Context, q ListQuery) ([]Run, error)
}

// ListQuery filters and bounds a runs listing. FlowID is optional;
// Limit <= 0 means "use the default of 50".
type ListQuery struct {
	FlowID string
	Limit  int
}
