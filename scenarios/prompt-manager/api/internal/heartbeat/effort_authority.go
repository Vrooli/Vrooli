package heartbeat

import (
	"context"
	"time"

	"github.com/vrooli/api-core/owneridentity"
)

// requireQualifiedCaller enforces the live create-run caller contract. Standing
// grants / ask-don't-block replaced the retired finite effort-authority broker,
// so the only qualification path is the owneridentity create-run caller seam.
func requireQualifiedCaller(ctx context.Context) error {
	return owneridentity.RequireCreateRunCaller(ctx, time.Now(), owneridentity.CreateRunCallerEnforced())
}

// carryQualifiedCaller forwards the live create-run caller proof across a
// context boundary (e.g. from a request context to a background execution).
func carryQualifiedCaller(from, to context.Context) (context.Context, error) {
	return owneridentity.CarryCreateRunCaller(from, to, time.Now(), owneridentity.CreateRunCallerEnforced())
}
