package templatevalidation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const DefaultCleanupOlderThan = 24 * time.Hour

type CleanupOptions struct {
	RepoRoot        string
	SearchRoots     []string
	OlderThan       time.Duration
	IncludeRetained bool
	RunID           string
	DryRun          bool
	Now             time.Time
}

type Run struct {
	MarkerPath string    `json:"markerPath"`
	Marker     RunMarker `json:"marker"`
	Age        string    `json:"age,omitempty"`
}

type SkippedRun struct {
	Run    *Run   `json:"run,omitempty"`
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
}

type FailedRun struct {
	Run   *Run   `json:"run,omitempty"`
	Path  string `json:"path,omitempty"`
	Error string `json:"error"`
}

type CleanupPlan struct {
	DryRun          bool          `json:"dryRun"`
	OlderThan       time.Duration `json:"olderThan"`
	IncludeRetained bool          `json:"includeRetained"`
	RunID           string        `json:"runId,omitempty"`
	Eligible        []Run         `json:"eligible,omitempty"`
	Skipped         []SkippedRun  `json:"skipped,omitempty"`
	Failures        []FailedRun   `json:"failures,omitempty"`
}

type CleanupResult struct {
	CleanupPlan
	Removed            []Run  `json:"removed,omitempty"`
	NeedsProtoGenerate bool   `json:"needsProtoGenerate"`
	ProtoGenerateRan   bool   `json:"protoGenerateRan,omitempty"`
	Message            string `json:"message,omitempty"`
}

func RelocationArtifactPaths(relocationTargets []string) []string {
	seen := map[string]struct{}{}
	var paths []string
	add := func(path string) {
		path = filepath.Clean(path)
		if strings.TrimSpace(path) == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for _, target := range relocationTargets {
		target = filepath.Clean(target)
		add(target)
		schemasParent := filepath.Dir(target)
		if filepath.Base(schemasParent) != "schemas" {
			continue
		}
		protoRoot := filepath.Dir(schemasParent)
		genRoot := filepath.Join(protoRoot, "gen")
		scenarioID := filepath.Base(target)
		scenarioPythonID := strings.ReplaceAll(scenarioID, "-", "_")
		for _, path := range []string{
			filepath.Join(genRoot, "go", scenarioID),
			filepath.Join(genRoot, "typescript", scenarioID),
			filepath.Join(genRoot, "typescript", "js", scenarioID),
			filepath.Join(genRoot, "python", scenarioID),
			filepath.Join(genRoot, "python", scenarioPythonID),
			// The per-surface lock manifest is the sixth codegen output and the
			// easiest to forget: it is a FILE beside the gen trees rather than a
			// directory inside one, so a loop over gen/<lang>/<id> misses it and
			// leaves a manifest referencing schemas that no longer exist.
			filepath.Join(genRoot, "manifests", scenarioID+".lock.json"),
		} {
			add(path)
		}
	}
	sort.Strings(paths)
	return paths
}

func PlanCleanup(opts CleanupOptions) CleanupPlan {
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.OlderThan <= 0 {
		opts.OlderThan = DefaultCleanupOlderThan
	}
	plan := CleanupPlan{
		DryRun:          opts.DryRun,
		OlderThan:       opts.OlderThan,
		IncludeRetained: opts.IncludeRetained,
		RunID:           strings.TrimSpace(opts.RunID),
	}
	searchRoots := opts.SearchRoots
	if len(searchRoots) == 0 {
		searchRoots = []string{os.TempDir()}
	}
	markerPaths := findMarkerPaths(searchRoots)
	for _, markerPath := range markerPaths {
		marker, err := ReadMarker(markerPath)
		if err != nil {
			if plan.RunID != "" {
				plan.Failures = append(plan.Failures, FailedRun{Path: markerPath, Error: err.Error()})
			} else {
				plan.Skipped = append(plan.Skipped, SkippedRun{Path: markerPath, Reason: "invalid marker: " + err.Error()})
			}
			continue
		}
		if err := ValidateMarker(opts.RepoRoot, markerPath, marker); err != nil {
			if plan.RunID != "" && marker.RunID == plan.RunID {
				plan.Failures = append(plan.Failures, FailedRun{Path: markerPath, Error: err.Error()})
			} else {
				plan.Skipped = append(plan.Skipped, SkippedRun{Path: markerPath, Reason: "unsafe marker: " + err.Error()})
			}
			continue
		}
		run := Run{MarkerPath: markerPath, Marker: marker, Age: formatAge(opts.Now.Sub(marker.CreatedAt))}
		if plan.RunID != "" {
			if marker.RunID == plan.RunID {
				plan.Eligible = append(plan.Eligible, run)
			}
			continue
		}
		if marker.Retained && !opts.IncludeRetained {
			plan.Skipped = append(plan.Skipped, SkippedRun{Run: &run, Reason: "retained run; use --include-retained or --run " + marker.RunID})
			continue
		}
		if opts.Now.Sub(marker.CreatedAt) < opts.OlderThan {
			plan.Skipped = append(plan.Skipped, SkippedRun{Run: &run, Reason: "younger than " + opts.OlderThan.String()})
			continue
		}
		plan.Eligible = append(plan.Eligible, run)
	}
	sortRuns(plan.Eligible)
	sortSkipped(plan.Skipped)
	sortFailures(plan.Failures)
	if plan.RunID != "" && len(plan.Eligible) == 0 && len(plan.Failures) == 0 {
		plan.Failures = append(plan.Failures, FailedRun{Error: "template validation run not found: " + plan.RunID})
	}
	return plan
}

func ExecuteCleanup(plan CleanupPlan) CleanupResult {
	result := CleanupResult{CleanupPlan: plan}
	for _, run := range plan.Eligible {
		for _, path := range run.Marker.RelocationArtifacts {
			if isProtoArtifact(run.Marker.RepoRoot, path) {
				result.NeedsProtoGenerate = true
			}
			if plan.DryRun {
				continue
			}
			if err := os.RemoveAll(path); err != nil {
				result.Failures = append(result.Failures, FailedRun{Run: &run, Path: path, Error: err.Error()})
			}
		}
		if plan.DryRun {
			continue
		}
		if err := os.RemoveAll(run.Marker.TempRoot); err != nil {
			result.Failures = append(result.Failures, FailedRun{Run: &run, Path: run.Marker.TempRoot, Error: err.Error()})
			continue
		}
		result.Removed = append(result.Removed, run)
	}
	result.Message = fmt.Sprintf("%d eligible, %d removed, %d skipped, %d failed", len(result.Eligible), len(result.Removed), len(result.Skipped), len(result.Failures))
	if plan.DryRun {
		result.Message = fmt.Sprintf("%d eligible, 0 removed (dry-run), %d skipped, %d failed", len(result.Eligible), len(result.Skipped), len(result.Failures))
	}
	return result
}

func ResultError(result CleanupResult) error {
	if len(result.Failures) == 0 {
		return nil
	}
	var messages []string
	for _, failure := range result.Failures {
		if strings.TrimSpace(failure.Path) != "" {
			messages = append(messages, failure.Path+": "+failure.Error)
			continue
		}
		messages = append(messages, failure.Error)
	}
	return errors.New(strings.Join(messages, "; "))
}

func findMarkerPaths(searchRoots []string) []string {
	seen := map[string]struct{}{}
	var paths []string
	for _, root := range searchRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "vrooli-template-deep-") {
				continue
			}
			markerPath := MarkerPath(filepath.Join(root, entry.Name()))
			if _, err := os.Stat(markerPath); err != nil {
				continue
			}
			if _, ok := seen[markerPath]; ok {
				continue
			}
			seen[markerPath] = struct{}{}
			paths = append(paths, markerPath)
		}
	}
	sort.Strings(paths)
	return paths
}

func isProtoArtifact(repoRoot, path string) bool {
	path = filepath.Clean(path)
	for _, parent := range []string{
		filepath.Join(repoRoot, "packages", "proto", "schemas"),
		filepath.Join(repoRoot, "packages", "proto", "gen"),
	} {
		if isInside(path, filepath.Clean(parent)) {
			return true
		}
	}
	return false
}

func formatAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d >= time.Hour {
		return d.Truncate(time.Hour).String()
	}
	if d >= time.Minute {
		return d.Truncate(time.Minute).String()
	}
	return d.Truncate(time.Second).String()
}

func sortRuns(runs []Run) {
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Marker.RunID < runs[j].Marker.RunID
	})
}

func sortSkipped(items []SkippedRun) {
	sort.Slice(items, func(i, j int) bool {
		return skippedKey(items[i]) < skippedKey(items[j])
	})
}

func sortFailures(items []FailedRun) {
	sort.Slice(items, func(i, j int) bool {
		return failureKey(items[i]) < failureKey(items[j])
	})
}

func skippedKey(item SkippedRun) string {
	if item.Run != nil {
		return item.Run.Marker.RunID
	}
	return item.Path
}

func failureKey(item FailedRun) string {
	if item.Run != nil {
		return item.Run.Marker.RunID
	}
	return item.Path + item.Error
}
