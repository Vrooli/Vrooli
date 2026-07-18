package evidence

import (
	"context"
	"fmt"
	"strings"
)

// OwnerIndex is the authoritative live session ownership index.
type OwnerIndex interface {
	LookupOwners(ctx context.Context, runID string) ([]Owner, error)
}

type RunOwnerResolver struct {
	Sessions OwnerIndex
}

func (r RunOwnerResolver) Resolve(ctx context.Context, runID string) (OwnershipStatus, *Owner, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return OwnershipUnresolved, nil, fmt.Errorf("run id is required")
	}
	if r.Sessions == nil {
		return OwnershipUnavailable, nil, fmt.Errorf("session owner index is required")
	}
	sessions, err := r.Sessions.LookupOwners(ctx, runID)
	if err != nil {
		return OwnershipUnavailable, nil, fmt.Errorf("query session owner index: %w", err)
	}
	owners := append([]Owner(nil), sessions...)
	if len(owners) == 0 {
		return OwnershipUnresolved, nil, nil
	}
	if len(owners) != 1 {
		return OwnershipAmbiguous, nil, nil
	}
	if err := owners[0].Validate(); err != nil {
		return OwnershipUnavailable, nil, err
	}
	return OwnershipResolved, &owners[0], nil
}
