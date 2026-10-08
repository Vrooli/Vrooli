package retention

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/paths"
)

// AutomaticRetentionPolicy returns the existing operator-configurable cleanup defaults.
func AutomaticRetentionPolicy() (bool, int64, int, int64) {
	enabled := true
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_ENABLED")); raw != "" {
		enabled = raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
	}
	seconds := int64((7 * 24 * time.Hour) / time.Second)
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_MAX_AGE")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed >= 0 {
			seconds = int64(parsed / time.Second)
		}
	}
	keep := 0
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_KEEP_COUNT")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			keep = parsed
		}
	}
	maxBytes := int64(0)
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_MAX_BYTES")); raw != "" {
		if parsed, err := coreRetention.ParseBytes(raw); err == nil && parsed >= 0 {
			maxBytes = parsed
		}
	}
	return enabled, seconds, keep, maxBytes
}

// StartAutomaticRetention owns the delayed, bounded enforcement schedule.
func (s *Service) StartAutomaticRetention(ctx context.Context) {
	enabled, _, keep, _ := AutomaticRetentionPolicy()
	if !enabled {
		return
	}
	budgets, err := ConfiguredEvidenceBudgets()
	if err != nil {
		if s.log != nil {
			s.log.WithError(err).Error("browser evidence retention configuration invalid")
		}
		return
	}
	interval := 15 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_INTERVAL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			interval = parsed
		}
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sweepCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
				release, lockErr := s.acquireRecoveryLock()
				if lockErr != nil {
					cancel()
					if s.log != nil {
						s.log.WithError(lockErr).Debug("automatic owner retention deferred while storage recovery is active")
					}
					continue
				}
				if err := s.EnforceEvidenceBudgets(sweepCtx, budgets, keep); err != nil && s.log != nil {
					s.log.WithError(err).Warn("automatic owner retention sweep failed")
				}
				release()
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// ConfiguredEvidenceBudgets reads the retained-capacity declarations storage-manager inventories.
func ConfiguredEvidenceBudgets() (map[string]coreRetention.Budget, error) {
	root := strings.TrimSpace(os.Getenv("SCENARIO_ROOT"))
	if root == "" {
		root = paths.ResolveScenarioDir(nil)
	}
	specs, err := coreRetention.LoadSpecs(coreRetention.ScenarioConfig{ManifestPath: filepath.Join(root, ".vrooli", "service.json")})
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

func (s *Service) EnforceEvidenceBudget(ctx context.Context, name string, budget coreRetention.Budget, keep int) (coreRetention.Result, error) {
	root := s.recordingsRoot
	if name == "captures" {
		root = s.capturesRoot
	}
	if strings.TrimSpace(root) == "" {
		return coreRetention.Result{}, fmt.Errorf("%s root is not configured", name)
	}
	batch, err := EvidenceRetentionBatchSize()
	if err != nil {
		return coreRetention.Result{}, err
	}
	var groups []string
	if name == "recordings" {
		groups = []string{"artifacts"}
	}
	pruner, err := coreRetention.NewDirectoryPruner(coreRetention.DirectoryConfig{
		Path: root, MaxItems: batch, KeepLatest: keep, ExpandDirs: groups,
		Eligible: func(ctx context.Context, path string) (bool, error) {
			if EvidenceActive(path) || EvidenceActive("execution:"+filepath.Base(path)) {
				return false, nil
			}
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
			if s.store == nil {
				return false, errors.New("execution repository unavailable")
			}
			exec, err := s.store.GetExecution(ctx, id)
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
			if err := WithInactiveEvidence([]string{path, "execution:" + filepath.Base(path)}, remove); err != nil {
				return err
			}
			if name == "recordings" {
				id, err := uuid.Parse(filepath.Base(path))
				if err != nil {
					return err
				}
				if err := s.store.DeleteExecution(ctx, id); err != nil && !errors.Is(err, database.ErrNotFound) {
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

func (s *Service) EnforceEvidenceBudgets(ctx context.Context, budgets map[string]coreRetention.Budget, keep int) error {
	var cycleErr error
	for _, name := range []string{"recordings", "captures"} {
		result, err := s.EnforceEvidenceBudget(ctx, name, budgets[name], keep)
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

func (s *Service) captureEligible(ctx context.Context, name, path string) (bool, error) {
	if EvidenceActive(path) {
		return false, nil
	}
	if s.store == nil {
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
		exec, err := s.store.GetExecution(ctx, id)
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

// EvidenceRetentionBatchSize bounds scheduled capacity enforcement separately from pressure recovery.
func EvidenceRetentionBatchSize() (int, error) {
	raw := strings.TrimSpace(os.Getenv("BAS_EVIDENCE_RETENTION_BATCH_SIZE"))
	if raw == "" {
		return 2000, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 100000 {
		return 0, fmt.Errorf("BAS_EVIDENCE_RETENTION_BATCH_SIZE must be between 1 and 100000")
	}
	return n, nil
}
