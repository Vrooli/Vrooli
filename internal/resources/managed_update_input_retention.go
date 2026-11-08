package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/vrooli/internal/config"
)

type managedUpdateInputCleanup struct {
	SchemaVersion           int       `json:"schema_version"`
	OperationID             string    `json:"operation_id"`
	Revision                uint64    `json:"revision"`
	PreservedInstance       string    `json:"preserved_instance,omitempty"`
	RemovedRuntimeInstances []string  `json:"removed_runtime_instances"`
	PrunedAt                time.Time `json:"pruned_at"`
}

// PruneManagedServiceUpdateInputs keeps the canonical serving role's private
// runtime files and all immutable ledger/fence receipts. Terminal shadows must
// be freshly fenced and quiescent before their runtime files are removed.
// Operation-wide credential copies and launcher images have no exec authority
// after a terminal commit and are removed even while the serving role lives.
func (c *Controller) PruneManagedServiceUpdateInputs(ctx context.Context, resource string) error {
	if !managedUpdateResource.MatchString(resource) {
		return fmt.Errorf("invalid update resource")
	}
	release, err := acquireManagedServiceLifecycleLock(ctx, resource)
	if err != nil {
		return err
	}
	defer release()
	return c.pruneManagedServiceUpdateInputsLocked(ctx, resource)
}

// The caller holds the resource lifecycle lock throughout proof and removal.
func (c *Controller) pruneManagedServiceUpdateInputsLocked(ctx context.Context, resource string) error {
	store, err := newManagedUpdateStore(resource)
	if err != nil {
		return err
	}
	ledger, err := store.read()
	if err != nil {
		return err
	}
	sup, paths, err := managedServiceSupervisorFor(resource)
	if err != nil {
		return err
	}
	canonical, err := sup.readState()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, entry := range ledger.Records {
		u := entry.State
		if !u.Phase.Terminal() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		preserve := ""
		if canonical.UpdateLaunch != nil && canonical.UpdateLaunch.OperationID == u.ID {
			preserve = canonical.InstanceID
		}
		root := filepath.Join(paths.StateDir, "managed-update-exec", u.ID)
		receiptPath := filepath.Join(root, "inputs-cleanup.json")
		if data, e := readManagedUpdatePrivateFile(receiptPath, 64<<10); e == nil {
			var prior managedUpdateInputCleanup
			if validateManagedUpdateJSON(data) != nil || json.Unmarshal(data, &prior) != nil {
				return fmt.Errorf("invalid managed update input cleanup receipt")
			}
			if prior.SchemaVersion != managedUpdateInputCleanupSchemaVersion || prior.OperationID != u.ID || prior.Revision != u.Revision {
				return fmt.Errorf("input cleanup receipt differs from terminal operation")
			}
			if prior.PreservedInstance == preserve {
				continue
			}
		} else if !os.IsNotExist(e) {
			return e
		}
		roles := []ManagedUpdateProcess{u.RollbackProbe, u.Candidate, u.Primary, u.Restored}
		for _, retired := range u.RetiredRestores {
			roles = append(roles, retired.Process)
		}
		removed := []string{}
		for _, role := range roles {
			if role.Intent == nil || role.Intent.InstanceID == preserve {
				continue
			}
			if role.Intent.Runtime.StateDir != paths.StateDir {
				return fmt.Errorf("input cleanup runtime escaped the resource owner")
			}
			if err := cleanupManagedUpdateRole(ctx, u, role); err != nil {
				return fmt.Errorf("retain unresolved role inputs: %w", err)
			}
			roleRoot, err := role.Intent.Runtime.roleRoot(u.ID, role.Intent.InstanceID)
			if err != nil {
				return err
			}
			if err := removeManagedUpdatePrivateTree(paths.StateDir, roleRoot); err != nil {
				return err
			}
			removed = append(removed, role.Intent.InstanceID)
		}
		for _, name := range []string{"credentials", "launcher"} {
			if err := removeManagedUpdatePrivateTree(paths.StateDir, filepath.Join(root, name)); err != nil {
				return err
			}
		}
		if err := ensureManagedUpdatePrivateDirectory(paths.StateDir, root); err != nil {
			return err
		}
		receipt := managedUpdateInputCleanup{SchemaVersion: managedUpdateInputCleanupSchemaVersion, OperationID: u.ID, Revision: u.Revision, PreservedInstance: preserve, RemovedRuntimeInstances: removed, PrunedAt: time.Now().UTC()}
		data, _ := json.Marshal(receipt)
		if err := config.WriteOwnedFileDurable(receiptPath, data, 0o600); err != nil {
			return fmt.Errorf("%w: private input cleanup acknowledgement: %w", ErrManagedUpdateCommitUnknown, err)
		}
	}
	return nil
}

func removeManagedUpdatePrivateTree(stateDir, path string) error {
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := ensureManagedUpdatePrivateDirectory(stateDir, path); err != nil {
		return err
	}
	return os.RemoveAll(path)
}
