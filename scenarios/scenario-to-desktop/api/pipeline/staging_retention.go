package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/retention"
)

// StagingRetention uses the shared age/capacity selector, with pipeline state
// supplying the deletion eligibility that a generic host cache cannot know.
type StagingRetention struct {
	Root   string
	Status func(string) (*Status, bool)
	// InUse also protects direct builds, smoke tests and live desktop sessions.
	InUse      func(string, string) bool
	KeepLatest int
}

func (s StagingRetention) Prune(ctx context.Context, budget retention.Budget) (retention.Result, error) {
	p, err := retention.NewDirectoryPruner(retention.DirectoryConfig{
		Path: s.Root, EntryDepth: 2, MaxItems: 20, KeepLatest: s.KeepLatest,
		Eligible: s.eligible,
	})
	if err != nil {
		return retention.Result{}, err
	}
	return p.Prune(ctx, budget)
}

func (s StagingRetention) eligible(_ context.Context, path string) (bool, error) {
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
	app := filepath.Base(filepath.Dir(path))
	if s.InUse != nil && s.InUse(app, path) {
		return false, nil
	}
	if _, err := os.Lstat(filepath.Join(path, ".building")); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if s.Status == nil {
		return false, fmt.Errorf("pipeline status unavailable")
	}
	if state, ok := s.Status(filepath.Base(path)); ok {
		return state != nil && state.IsComplete(), nil
	}
	// Unknown/legacy work receives a grace period; active direct generation is
	// protected by InUse. Unknown directories are never recursively interpreted.
	return time.Since(info.ModTime()) >= 2*time.Hour, nil
}

func StagingBudget(manifestPath string) (retention.Budget, error) {
	specs, err := retention.LoadSpecs(retention.ScenarioConfig{ManifestPath: manifestPath})
	if err != nil {
		return retention.Budget{}, err
	}
	for _, spec := range specs {
		if spec.Budget.Name != "staging" {
			continue
		}
		budget, err := retention.ConfigureBudget(spec.Budget,
			strings.TrimSpace(os.Getenv("DESKTOP_STAGING_RETENTION_MAX_AGE")),
			strings.TrimSpace(os.Getenv("DESKTOP_STAGING_RETENTION_MAX_BYTES")))
		if err != nil {
			return retention.Budget{}, err
		}
		if !budget.HasByteBound() || !budget.HasAgeBound() {
			return retention.Budget{}, fmt.Errorf("staging requires age and byte bounds")
		}
		return budget, nil
	}
	return retention.Budget{}, fmt.Errorf("staging retention declaration is required")
}
