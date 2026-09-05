package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/database"
)

// evidenceBudgets reads the same declarations storage-manager inventories.
// These are retained-capacity ceilings; cleanup request max_bytes remains a
// separate per-request deletion limit.
func evidenceBudgets() (map[string]coreRetention.Budget, error) {
	specs, err := coreRetention.LoadSpecs(coreRetention.ScenarioConfig{StartDir: os.Getenv("SCENARIO_ROOT")})
	if err != nil {
		return nil, err
	}
	budgets := make(map[string]coreRetention.Budget)
	for _, spec := range specs {
		name := spec.Budget.Name
		if name != "recordings" && name != "captures" {
			continue
		}
		prefix := "BAS_" + strings.ToUpper(name) + "_RETENTION_"
		age := os.Getenv(prefix + "MAX_AGE")
		if age == "" {
			age = os.Getenv("BAS_OWNER_RETENTION_MAX_AGE")
		}
		b, err := coreRetention.ConfigureBudget(spec.Budget, age, os.Getenv(prefix+"MAX_BYTES"))
		if err != nil {
			return nil, err
		}
		if !b.HasAgeBound() || !b.HasByteBound() {
			return nil, fmt.Errorf("%s retention requires age and byte bounds", name)
		}
		budgets[name] = b
	}
	if len(budgets) != 2 {
		return nil, errors.New("recordings and captures retention declarations are required")
	}
	return budgets, nil
}

func (s *ownerCleanupService) enforceEvidenceBudget(ctx context.Context, name string, budget coreRetention.Budget, keep int) (coreRetention.Result, error) {
	root := s.root
	if name == "captures" {
		root = s.capturesRoot
	}
	if strings.TrimSpace(root) == "" {
		return coreRetention.Result{}, fmt.Errorf("%s root is not configured", name)
	}
	pruner, err := coreRetention.NewDirectoryPruner(coreRetention.DirectoryConfig{
		Path: root, MaxItems: ownerCleanupBatchCap, KeepLatest: keep,
		Eligible: func(ctx context.Context, path string) (bool, error) {
			info, err := os.Lstat(path)
			if os.IsNotExist(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return false, nil
			}
			if name == "captures" {
				return s.captureEligible(ctx, filepath.Base(path), path)
			}
			id, err := uuid.Parse(filepath.Base(path))
			if err != nil {
				return false, nil
			}
			if s.repo == nil {
				return false, errors.New("execution repository unavailable")
			}
			exec, err := s.repo.GetExecution(ctx, id)
			if errors.Is(err, database.ErrNotFound) {
				// An unindexed directory may belong to an export still starting.
				return time.Since(info.ModTime()) >= time.Hour, nil
			}
			if err != nil {
				return false, err
			}
			return exec != nil && database.IsTerminalStatus(exec.Status), nil
		},
		RemoveHook: func(path string, remove func() error) error {
			if err := remove(); err != nil {
				return err
			}
			if name == "recordings" {
				id, err := uuid.Parse(filepath.Base(path))
				if err != nil {
					return err
				}
				if err := s.repo.DeleteExecution(ctx, id); err != nil && !errors.Is(err, database.ErrNotFound) {
					return err
				}
			}
			return nil
		},
	})
	if err != nil {
		return coreRetention.Result{}, err
	}
	return pruner.Prune(ctx, budget)
}

func (s *ownerCleanupService) enforceEvidenceBudgets(ctx context.Context, budgets map[string]coreRetention.Budget, keep int) error {
	var cycleErr error
	for _, name := range []string{"recordings", "captures"} {
		result, err := s.enforceEvidenceBudget(ctx, name, budgets[name], keep)
		if err != nil {
			cycleErr = errors.Join(cycleErr, fmt.Errorf("%s: %w", name, err))
		}
		if s.log != nil {
			s.log.WithFields(map[string]interface{}{"budget": name, "used_bytes": result.After.Bytes,
				"max_bytes": budgets[name].MaxBytes, "freed_bytes": result.FreedBytes,
				"deleted": result.Deleted, "incomplete": result.Incomplete}).Info("browser evidence retention cycle")
		}
	}
	namespace, err := storage.ScenarioNamespace("browser-automation-studio")
	if err != nil {
		return errors.Join(cycleErr, err)
	}
	if err := coreRetention.RecordEnforcementReceipt(namespace, time.Now().UTC(), cycleErr); err != nil {
		cycleErr = errors.Join(cycleErr, err)
	}
	return cycleErr
}

func (s *ownerCleanupService) captureEligible(ctx context.Context, name, path string) (bool, error) {
	if s.repo == nil {
		return false, errors.New("execution repository unavailable")
	}
	ids := []string{name}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		ids = append(ids, entry.Name())
	}
	indexed := false
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		exec, err := s.repo.GetExecution(ctx, id)
		if errors.Is(err, database.ErrNotFound) {
			continue
		}
		if err != nil {
			return false, err
		}
		if exec != nil {
			indexed = true
			if !database.IsTerminalStatus(exec.Status) {
				return false, nil
			}
		}
	}
	if !indexed {
		info, err := os.Stat(path)
		if err != nil {
			return false, err
		}
		return time.Since(info.ModTime()) >= time.Hour, nil
	}
	return true, nil
}
