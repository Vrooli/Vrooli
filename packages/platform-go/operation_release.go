package platform

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// OwnedOperationIntentAbsent observes both the deterministic unit and cgroup.
// It neither recovers an exit outcome nor prevents a future dispatch. The
// domain owner must durably revoke its execution fence before using this fact.
func OwnedOperationIntentAbsent(ctx context.Context, intent OperationIntent) (bool, error) {
	if ctx == nil {
		return false, fmt.Errorf("platform: operation absence context is required")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := intent.ValidateIdentity(); err != nil {
		return false, err
	}
	if intent.Kind != "" {
		return false, ErrOperationIdentity
	}
	return ownedWorkloadIntentAbsent(ctx, intent)
}

// OwnedOperationReleasedForIncarnation requires unit/cgroup absence and
// excludes the recorded root using its boot-id:start-ticks identity. It is
// observation only, never proof of successful execution or a dispatch fence.
func OwnedOperationReleasedForIncarnation(ctx context.Context, ref OperationRef, incarnation string) (bool, error) {
	if ctx == nil {
		return false, fmt.Errorf("platform: operation release context is required")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := ref.ValidateIdentity(); err != nil {
		return false, err
	}
	if ref.Native != nil {
		if ref.Intent.Kind != "" || !ref.MatchesIncarnation(incarnation) {
			return false, ErrOperationIdentity
		}
		return ownedWorkloadReleased(ctx, ref, ref.Native.BootID)
	}
	boot, ticks, ok := strings.Cut(incarnation, ":")
	n, err := strconv.ParseUint(ticks, 10, 64)
	if ref.Intent.Kind != "" || !ok || !serviceBootIDPattern.MatchString(boot) || err != nil || n == 0 || ticks != ref.RootStartTime || ref.RootPID <= 0 {
		return false, ErrOperationIdentity
	}
	return ownedWorkloadReleased(ctx, ref, boot)
}
