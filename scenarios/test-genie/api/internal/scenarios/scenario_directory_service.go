package scenarios

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	apistorage "github.com/vrooli/api-core/storage"

	"test-genie/internal/shared"
)

type scenarioSummaryStore interface {
	List(ctx context.Context) ([]ScenarioSummary, error)
	Get(ctx context.Context, scenario string) (*ScenarioSummary, error)
}

// ScenarioLister exposes scenario metadata from the Vrooli CLI.
type ScenarioLister interface {
	ListScenarios(ctx context.Context) ([]ScenarioMetadata, error)
}

// ScenarioMetadata captures high-level scenario info returned by the CLI.
type ScenarioMetadata struct {
	Name        string
	Description string
	Status      string
	Tags        []string
}

// ScenarioDirectoryService orchestrates catalog lookups.
type ScenarioDirectoryService struct {
	repo          scenarioSummaryStore
	lister        ScenarioLister
	logger        *log.Logger
	scenariosRoot string
	detectTesting func(string) TestingCapabilities
	runTests      func(context.Context, TestingCapabilities, RunScenarioTestOptions) (*TestingRunnerResult, error)
}

// RunScenarioTestOptions carries execution preferences and extra args passed to the runner.
type RunScenarioTestOptions struct {
	Preferred string
	ExtraArgs []string
}

// scenarioTestLogDir resolves where scenario-test logs are written.
//
// These logs used to land in <scenariosRoot>/_artifacts/scenario-tests, i.e.
// inside the scenarios/ SOURCE namespace, where the directory reads like a
// scenario named "_artifacts" and every run dirtied the repo. Run logs are
// runtime state, so they belong in the platform's logs class alongside every
// other scenario's operational output.
//
// Resolution failure falls back to the OS temp dir, never to the repo: a log
// path is not worth failing startup over, but it must never write into source.
func scenarioTestLogDir() string {
	resolver, err := apistorage.NewResolver(apistorage.ResolverConfig{
		AppID:   "vrooli",
		Profile: apistorage.ProfileAuto,
	})
	if err == nil {
		path, perr := resolver.Path(
			apistorage.Options{ScenarioID: "test-genie"},
			apistorage.ClassLogs,
			"scenario-tests",
		)
		if perr == nil {
			return path
		}
	}
	return filepath.Join(os.TempDir(), "vrooli-test-genie", "scenario-tests")
}

func NewScenarioDirectoryService(repo scenarioSummaryStore, lister ScenarioLister, scenariosRoot string) *ScenarioDirectoryService {
	return NewScenarioDirectoryServiceWithLogger(repo, lister, scenariosRoot, log.Default())
}

// NewScenarioDirectoryServiceWithLogger constructs the directory service with
// an owner-scoped logger. A nil logger retains the process default.
func NewScenarioDirectoryServiceWithLogger(repo scenarioSummaryStore, lister ScenarioLister, scenariosRoot string, logger *log.Logger) *ScenarioDirectoryService {
	if logger == nil {
		logger = log.Default()
	}
	defaultRunner := TestingRunner{
		Timeout: defaultTestingTimeout,
		Output:  log.Writer(),
		LogDir:  scenarioTestLogDir(),
	}
	return &ScenarioDirectoryService{
		repo:          repo,
		lister:        lister,
		logger:        logger,
		scenariosRoot: strings.TrimSpace(scenariosRoot),
		detectTesting: DetectTestingCapabilities,
		runTests: func(ctx context.Context, caps TestingCapabilities, opts RunScenarioTestOptions) (*TestingRunnerResult, error) {
			return defaultRunner.RunWithArgs(ctx, caps, opts.Preferred, opts.ExtraArgs)
		},
	}
}

// ListSummaries returns every tracked scenario with queue + execution metadata and hydrates
// missing scenarios with CLI metadata so the catalog includes the entire workspace.
func (s *ScenarioDirectoryService) ListSummaries(ctx context.Context) ([]ScenarioSummary, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}

	summaries, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range summaries {
		s.decorateScenario(&summaries[i])
	}
	if s.lister == nil {
		return summaries, nil
	}

	metadata, err := s.lister.ListScenarios(ctx)
	if err != nil {
		s.logger.Printf("scenario lister failed: %v", err)
		return summaries, nil
	}

	if len(metadata) == 0 {
		return summaries, nil
	}

	summaryMap := make(map[string]ScenarioSummary, len(summaries))
	for _, summary := range summaries {
		key := strings.ToLower(summary.ScenarioName)
		if key == "" {
			continue
		}
		summaryMap[key] = summary
	}

	merged := make([]ScenarioSummary, 0, len(metadata)+len(summaries))
	for _, meta := range metadata {
		name := strings.TrimSpace(meta.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if summary, ok := summaryMap[key]; ok {
			summary := applyScenarioMetadata(summary, meta)
			s.decorateScenario(&summary)
			merged = append(merged, summary)
			delete(summaryMap, key)
			continue
		}
		summary := applyScenarioMetadata(ScenarioSummary{
			ScenarioName:    name,
			TotalExecutions: 0,
		}, meta)
		s.decorateScenario(&summary)
		merged = append(merged, summary)
	}

	// Preserve any tracked scenarios not currently returned by the CLI.
	if len(summaryMap) > 0 {
		leftovers := make([]ScenarioSummary, 0, len(summaryMap))
		for _, summary := range summaryMap {
			s.decorateScenario(&summary)
			leftovers = append(leftovers, summary)
		}
		sort.Slice(leftovers, func(i, j int) bool {
			return strings.ToLower(leftovers[i].ScenarioName) < strings.ToLower(leftovers[j].ScenarioName)
		})
		merged = append(merged, leftovers...)
	}

	return merged, nil
}

// GetSummary loads a single scenario summary by name and falls back to CLI metadata if needed.
func (s *ScenarioDirectoryService) GetSummary(ctx context.Context, scenario string) (*ScenarioSummary, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, sql.ErrNoRows
	}

	summary, err := s.repo.Get(ctx, scenario)
	if err == nil {
		if meta := s.lookupScenarioMetadata(ctx, scenario); meta != nil {
			result := applyScenarioMetadata(*summary, *meta)
			s.decorateScenario(&result)
			return &result, nil
		}
		s.decorateScenario(summary)
		return summary, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	meta := s.lookupScenarioMetadata(ctx, scenario)
	if meta == nil {
		return nil, err
	}

	placeholder := applyScenarioMetadata(ScenarioSummary{
		ScenarioName:    meta.Name,
		TotalExecutions: 0,
	}, *meta)
	s.decorateScenario(&placeholder)
	return &placeholder, nil
}

func (s *ScenarioDirectoryService) lookupScenarioMetadata(ctx context.Context, scenario string) *ScenarioMetadata {
	if s.lister == nil {
		return nil
	}
	metadata, err := s.lister.ListScenarios(ctx)
	if err != nil {
		s.logger.Printf("scenario metadata lookup failed: %v", err)
		return nil
	}
	target := strings.ToLower(strings.TrimSpace(scenario))
	for _, item := range metadata {
		if strings.ToLower(strings.TrimSpace(item.Name)) == target {
			meta := item
			return &meta
		}
	}
	return nil
}

func applyScenarioMetadata(summary ScenarioSummary, meta ScenarioMetadata) ScenarioSummary {
	summary.ScenarioDescription = strings.TrimSpace(meta.Description)
	summary.ScenarioStatus = strings.TrimSpace(meta.Status)
	if len(meta.Tags) > 0 {
		summary.ScenarioTags = append([]string(nil), meta.Tags...)
	} else {
		summary.ScenarioTags = nil
	}
	return summary
}

func (s *ScenarioDirectoryService) decorateScenario(summary *ScenarioSummary) {
	if summary == nil || summary.ScenarioName == "" {
		return
	}
	if summary.LastExperienceAt == nil {
		summary.ExperienceFreshness = ExperienceNeverRun
	}
	if s == nil || s.scenariosRoot == "" || s.detectTesting == nil {
		if summary.LastExperienceAt != nil {
			summary.ExperienceFreshness = ExperienceUIChangeTimeUnavailable
		}
		return
	}
	dir := filepath.Join(s.scenariosRoot, summary.ScenarioName)
	if _, err := os.Stat(dir); err != nil {
		if summary.LastExperienceAt != nil {
			summary.ExperienceFreshness = ExperienceUIChangeTimeUnavailable
		}
		return
	}
	changedAt, ok, err := latestUIChange(filepath.Join(dir, "ui"))
	if err != nil {
		if summary.LastExperienceAt != nil {
			summary.ExperienceFreshness = ExperienceUIChangeTimeUnavailable
		}
	} else if ok {
		summary.LastUIChangeAt = &changedAt
		if summary.LastExperienceAt != nil {
			if !summary.LastExperienceAt.After(changedAt) {
				summary.ExperienceFreshness = ExperienceOlderThanLastUIChange
			} else {
				summary.ExperienceFreshness = ExperienceEvaluatedAfterLastUIChange
			}
		}
	} else if summary.LastExperienceAt != nil {
		summary.ExperienceFreshness = ExperienceUISurfaceUnavailable
	}
	caps := s.detectTesting(dir)
	if !caps.HasTests {
		summary.Testing = nil
		return
	}
	summary.Testing = &caps
}

func latestUIChange(root string) (time.Time, bool, error) {
	var latest time.Time
	rootInfo, err := os.Lstat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return time.Time{}, false, fmt.Errorf("UI source root is not a regular directory")
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root {
			switch entry.Name() {
			case "node_modules", "dist", "build", "coverage", ".git", ".vite":
				if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
					return filepath.SkipDir
				}
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("UI source tree contains symlink %q", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return time.Time{}, false, err
	}
	return latest, !latest.IsZero(), nil
}

// RunScenarioTests executes the preferred testing command for the provided scenario.
// When scenarioDirOverride is non-empty, it is used as the scenario directory instead
// of resolving via scenariosRoot. This supports sandboxed agents whose files live
// in an overlay filesystem. See packages/cli-core/cliutil/sandbox.go.
func (s *ScenarioDirectoryService) RunScenarioTests(ctx context.Context, scenario string, preferred string, extraArgs []string, scenarioDirOverride string) (*TestingCommand, *TestingRunnerResult, error) {
	if s == nil || s.runTests == nil {
		return nil, nil, sql.ErrConnDone
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, nil, shared.NewValidationError("scenario name is required")
	}
	dir := strings.TrimSpace(scenarioDirOverride)
	if dir == "" {
		if s.scenariosRoot == "" {
			return nil, nil, fmt.Errorf("scenarios root is not configured")
		}
		dir = filepath.Join(s.scenariosRoot, scenario)
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("scenario directory not found: %w", os.ErrNotExist)
		}
		return nil, nil, fmt.Errorf("failed to access scenario directory: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("scenario path is not a directory")
	}
	caps := s.detectTesting(dir)
	if !caps.HasTests {
		return nil, nil, shared.NewValidationError("scenario does not define runnable tests")
	}
	cmd := caps.SelectCommand(preferred)
	if cmd == nil {
		return nil, nil, shared.NewValidationError("requested test type is unavailable for this scenario")
	}
	result, err := s.runTests(ctx, caps, RunScenarioTestOptions{Preferred: preferred, ExtraArgs: extraArgs})
	if err != nil {
		return nil, nil, err
	}
	return cmd, result, nil
}
