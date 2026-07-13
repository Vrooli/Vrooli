package evidence

import (
	"context"
	"fmt"
	"strings"
)

// OwnerIndex is one authoritative ownership index. Both indexes must be
// queried for every run: no precedence rule is allowed to hide dual owners.
type OwnerIndex interface {
	LookupOwners(ctx context.Context, runID string) ([]Owner, error)
}

type RunOwnerResolver struct {
	Sessions       OwnerIndex
	OperatingModes OwnerIndex
}

func (r RunOwnerResolver) Resolve(ctx context.Context, runID string) (OwnershipStatus, *Owner, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return OwnershipUnresolved, nil, fmt.Errorf("run id is required")
	}
	if r.Sessions == nil || r.OperatingModes == nil {
		return OwnershipUnavailable, nil, fmt.Errorf("all owner indexes are required")
	}
	// Execute both lookups before making any decision. The individual indexes
	// may be slow or unavailable independently, so neither is allowed to
	// short-circuit the other.
	sessions, sessionErr := r.Sessions.LookupOwners(ctx, runID)
	modes, modeErr := r.OperatingModes.LookupOwners(ctx, runID)
	if sessionErr != nil || modeErr != nil {
		return OwnershipUnavailable, nil, fmt.Errorf("query run owner indexes: sessions=%v operating_modes=%v", sessionErr, modeErr)
	}
	owners := append(append([]Owner(nil), sessions...), modes...)
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
