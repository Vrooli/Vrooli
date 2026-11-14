package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Caller holds the resource lifecycle lock. Certain rejected admissions have
// no exec authority and must not accumulate secret snapshots or RAM leases.
// A visible admitted receipt, unreadable ledger or unknown ledger-write outcome
// retains inputs for reconciliation instead. The last case is handled by the
// admission owner's deferred cleanup guard before calling here.
func cleanupManagedUpdateRejectedAdmission(ctx context.Context, store *managedUpdateStore, u ManagedServiceUpdate, stateDir string) error {
	ledger, err := store.read()
	if err != nil {
		return fmt.Errorf("rejected admission cleanup retained inputs: %w", err)
	}
	for _, record := range ledger.Records {
		if record.State.ID == u.ID || record.State.RequestID == u.RequestID {
			return nil
		}
	}
	if !managedUpdateKey.MatchString(u.ID) || !filepath.IsAbs(stateDir) || filepath.Clean(stateDir) != stateDir {
		return fmt.Errorf("rejected admission cleanup has invalid owner paths")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	root := filepath.Join(stateDir, "managed-update-exec", u.ID)
	if _, err := os.Lstat(root); err == nil {
		if err := ensureManagedUpdatePrivateDirectory(stateDir, root); err != nil {
			return err
		}
		if err := os.RemoveAll(root); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return releaseManagedUpdateCapacityClaims(ctx, u)
}
