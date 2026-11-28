package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	// It returns the protecting owner (for example "smoke_test"), or "" when
	// the entry is not in use, so a cycle can say why it could not reclaim.
	InUse      func(app, path string) string
	KeepLatest int
}

// Protection reasons reported for over-budget entries the owner refused to
// delete. InUse reasons are reported verbatim alongside these.
const (
	ProtectedMissing            = "missing"
	ProtectedNotDirectory       = "not_directory"
	ProtectedBuildingMarker     = "building_marker"
	ProtectedPipelineActive     = "pipeline_incomplete"
	ProtectedUnknownGracePeriod = "unknown_grace_period"
)

// StagingResult is the shared retention result plus the owner-specific
// reasons that kept over-budget entries alive this cycle.
type StagingResult struct {
	retention.Result
	// Protected counts over-budget candidates the owner refused, by reason.
	Protected map[string]int
}

// ProtectedTotal is the number of over-budget candidates the owner refused.
func (r StagingResult) ProtectedTotal() int {
	total := 0
	for _, n := range r.Protected {
		total += n
	}
	return total
}

// ProtectedSummary renders Protected deterministically, e.g.
// "smoke_test=3,pipeline_incomplete=1".
func (r StagingResult) ProtectedSummary() string {
	reasons := make([]string, 0, len(r.Protected))
	for reason := range r.Protected {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, fmt.Sprintf("%s=%d", reason, r.Protected[reason]))
	}
	return strings.Join(parts, ",")
}

func (s StagingRetention) Prune(ctx context.Context, budget retention.Budget) (StagingResult, error) {
	out := StagingResult{Protected: map[string]int{}}
	p, err := retention.NewDirectoryPruner(retention.DirectoryConfig{
		Path: s.Root, EntryDepth: 2, MaxItems: 20, KeepLatest: s.KeepLatest,
		Eligible: func(ctx context.Context, path string) (bool, error) {
			reason, err := s.protection(ctx, path)
			if err != nil {
				return false, err
			}
			if reason != "" {
				out.Protected[reason]++
				return false, nil
			}
			return true, nil
		},
	})
	if err != nil {
		return out, err
	}
	out.Result, err = p.Prune(ctx, budget)
	return out, err
}

// protection returns why path must be kept, or "" when it may be deleted.
func (s StagingRetention) protection(_ context.Context, path string) (string, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return ProtectedMissing, nil
	}
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ProtectedNotDirectory, nil
	}
	app := filepath.Base(filepath.Dir(path))
	if s.InUse != nil {
		if reason := s.InUse(app, path); reason != "" {
			return reason, nil
		}
	}
	if _, err := os.Lstat(filepath.Join(path, ".building")); err == nil {
		return ProtectedBuildingMarker, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if s.Status == nil {
		return "", fmt.Errorf("pipeline status unavailable")
	}
	if state, ok := s.Status(filepath.Base(path)); ok {
		if state != nil && state.IsComplete() {
			return "", nil
		}
		return ProtectedPipelineActive, nil
	}
	// Unknown/legacy work receives a grace period; active direct generation is
	// protected by InUse. Unknown directories are never recursively interpreted.
	if time.Since(info.ModTime()) >= 2*time.Hour {
		return "", nil
	}
	return ProtectedUnknownGracePeriod, nil
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
