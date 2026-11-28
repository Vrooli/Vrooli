package setupprivilege

import (
	"context"
	"fmt"
	"os/user"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/safeguards/kdump-observability"
	"github.com/vrooli/vrooli/internal/safeguards/observabilityinstall"
	"github.com/vrooli/vrooli/internal/safeguards/pstore-observability"
)

// RunSystemObservabilityInstall applies one registered observability safeguard
// as the root-side operation of the setup privilege worker.
func RunSystemObservabilityInstall(ctx context.Context, target observabilityinstall.Target, targetUser string, retention int) error {
	if err := observabilityinstall.ValidateSetupRequest(target, retention); err != nil {
		return err
	}
	if targetUser == "" || targetUser != hostreqkit.InvokingUser() {
		return fmt.Errorf("observability install target must match the invoking setup user")
	}
	if _, err := user.Lookup(targetUser); err != nil {
		return fmt.Errorf("resolve invoking setup user: %w", err)
	}
	switch target {
	case observabilityinstall.Pstore:
		return pstoreobservability.ApplyForSetupPrivilegeWorker(ctx, targetUser)
	case observabilityinstall.Kdump:
		return kdumpobservability.ApplyForSetupPrivilegeWorker(ctx, targetUser, retention)
	default:
		return fmt.Errorf("unknown observability installer target %d", target)
	}
}
