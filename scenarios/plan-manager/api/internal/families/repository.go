package families

import (
	"context"
	"errors"

	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
)

var (
	ErrNotFound = errors.New("plan family not found")
	ErrConflict = errors.New("plan family revision conflict")
)

// Repository hides the storage engine and commits aggregate, graph and review
// records together. Graph revisions and reviews are append-only.
type Repository interface {
	Create(context.Context, *familiesv1.PlanFamily) error
	Get(context.Context, string) (*familiesv1.PlanFamily, error)
	List(context.Context, uint32, string) ([]*familiesv1.PlanFamily, string, error)
	Commit(context.Context, *familiesv1.PlanFamily, uint64, *familiesv1.GraphRevision, *familiesv1.GraphReview) error
}
