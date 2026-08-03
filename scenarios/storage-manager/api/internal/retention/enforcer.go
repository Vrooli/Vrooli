// Package retention contains the storage-manager-owned adapter for directory
// budgets declared in storage.entries. It delegates selection and cleanup to
// api-core/retention's builtin directory pruner; storage-manager only resolves
// the owner path and records whether the bounded operation completed.
package retention

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	coreRetention "github.com/vrooli/api-core/retention"
	coreStorage "github.com/vrooli/api-core/storage"
)

// Enforcer applies directory budgets from the normalized owner inventory.
// Non-directory entries are intentionally left to their owner-specific
// retention manager; storage-manager must never guess how to prune a file or
// SQLite table it does not own.
type Enforcer struct {
	RepoRoot string
	Platform coreStorage.Platform
}

// Result records the outcome for one owner entry. A successful result means
// the builtin provider measured and reconciled the directory, including the
// no-op case where it was already within its ceiling.
type Result struct {
	Owner   string
	Entry   string
	Deleted int
	Freed   int64
	Error   string
}

// Enforce applies every supported storage-entry budget and returns successful
// owner IDs. A failure is returned immediately: retention status must not call
// an owner governed when one of its declared directories could not be checked.
func (e Enforcer) Enforce(ctx context.Context, inventory coreStorage.OwnerInventory) (map[string]Result, error) {
	platform := e.Platform
	if platform == "" {
		platform = coreStorage.Platform(runtime.GOOS)
	}
	results := make(map[string]Result)
	for _, owner := range inventory.Owners {
		for _, entry := range owner.StorageEntries {
			if entry.Budget == nil || entry.Kind != "dir" {
				continue
			}
			path, err := coreStorage.ResolveOwnerStoragePath(e.RepoRoot, owner, entry, platform, coreStorage.PlatformSeams{})
			if err != nil {
				var notApplicable *coreStorage.NotApplicable
				if errors.As(err, &notApplicable) {
					continue
				}
				results[owner.ID] = Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("resolve storage path: %w", err).Error()}
				continue
			}
			budget := coreRetention.Budget{Name: entry.Name}
			if strings.TrimSpace(entry.Budget.MaxBytes) != "" {
				budget.MaxBytes, err = coreRetention.ParseBytes(entry.Budget.MaxBytes)
				if err != nil {
					results[owner.ID] = Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("parse max_bytes: %w", err).Error()}
					continue
				}
			}
			if strings.TrimSpace(entry.Budget.MaxAge) != "" {
				budget.MaxAge, err = coreRetention.ParseAge(entry.Budget.MaxAge)
				if err != nil {
					results[owner.ID] = Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("parse max_age: %w", err).Error()}
					continue
				}
			}
			pruner, err := coreRetention.NewDirectoryPruner(coreRetention.DirectoryConfig{Path: path})
			if err != nil {
				results[owner.ID] = Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("build provider: %w", err).Error()}
				continue
			}
			out, err := pruner.Prune(ctx, budget)
			if err != nil {
				results[owner.ID] = Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("enforce: %w", err).Error()}
				continue
			}
			results[owner.ID] = Result{Owner: owner.ID, Entry: entry.Name, Deleted: int(out.Deleted), Freed: out.FreedBytes}
		}
	}
	return results, nil
}
