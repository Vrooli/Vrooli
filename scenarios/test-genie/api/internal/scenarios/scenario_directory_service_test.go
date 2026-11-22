package scenarios

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type fakeScenarioRepo struct {
	listSummaries []ScenarioSummary
	listErr       error
	getSummary    *ScenarioSummary
	getErr        error
}

func TestExperienceFreshnessIsServerDerivedAndFailsClosed(t *testing.T) {
	root := t.TempDir()
	uiRoot := filepath.Join(root, "demo", "ui")
	sourceDir := filepath.Join(uiRoot, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(sourceDir, "App.tsx")
	if err := os.WriteFile(page, []byte("export default function App() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	changeAt := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	for _, path := range []string{page, sourceDir, uiRoot} {
		if err := os.Chtimes(path, changeAt, changeAt); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewScenarioDirectoryService(&fakeScenarioRepo{}, nil, root)
	verdictAt := changeAt.Add(time.Hour)
	decorate := func(at *time.Time) ScenarioSummary {
		t.Helper()
		summary := ScenarioSummary{ScenarioName: "demo", LastExperienceAt: at}
		svc.decorateScenario(&summary)
		return summary
	}

	if got := decorate(&verdictAt).ExperienceFreshness; got != ExperienceEvaluatedAfterLastUIChange {
		t.Fatalf("current verdict freshness = %q, want %q", got, ExperienceEvaluatedAfterLastUIChange)
	}
	if got := decorate(&changeAt).ExperienceFreshness; got != ExperienceOlderThanLastUIChange {
		t.Fatalf("timestamp tie must fail closed as stale, got %q", got)
	}
	if got := decorate(nil).ExperienceFreshness; got != ExperienceNeverRun {
		t.Fatalf("missing verdict freshness = %q, want %q", got, ExperienceNeverRun)
	}

	// A deletion leaves no file timestamp, so the containing directory's
	// timestamp must advance the UI change frontier.
	removedAt := changeAt.Add(2 * time.Hour)
	if err := os.Remove(page); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{sourceDir, uiRoot} {
		if err := os.Chtimes(path, removedAt, removedAt); err != nil {
			t.Fatal(err)
		}
	}
	stale := decorate(&verdictAt)
	if stale.ExperienceFreshness != ExperienceOlderThanLastUIChange || stale.LastUIChangeAt == nil || !stale.LastUIChangeAt.Equal(removedAt) {
		t.Fatalf("deleted UI source did not stale verdict: freshness=%q uiChange=%v", stale.ExperienceFreshness, stale.LastUIChangeAt)
	}

	if err := os.RemoveAll(uiRoot); err != nil {
		t.Fatal(err)
	}
	if got := decorate(&verdictAt).ExperienceFreshness; got != ExperienceUISurfaceUnavailable {
		t.Fatalf("absent UI surface freshness = %q, want %q", got, ExperienceUISurfaceUnavailable)
	}
	if err := os.Symlink(filepath.Join(root, "missing-ui-target"), uiRoot); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if got := decorate(&verdictAt).ExperienceFreshness; got != ExperienceUIChangeTimeUnavailable {
		t.Fatalf("uninspectable UI tree freshness = %q, want %q", got, ExperienceUIChangeTimeUnavailable)
	}
}

func (f *fakeScenarioRepo) List(context.Context) ([]ScenarioSummary, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listSummaries, nil
}

func (f *fakeScenarioRepo) Get(context.Context, string) (*ScenarioSummary, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getSummary, nil
}

type fakeScenarioLister struct {
	items []ScenarioMetadata
	err   error
}

func (f *fakeScenarioLister) ListScenarios(context.Context) ([]ScenarioMetadata, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func TestScenarioDirectoryServiceListSummariesHydratesCatalog(t *testing.T) {
	repo := &fakeScenarioRepo{
		listSummaries: []ScenarioSummary{
			{
				ScenarioName: "swarm-manager",
			},
		},
	}
	lister := &fakeScenarioLister{
		items: []ScenarioMetadata{
			{Name: "swarm-manager", Description: "Ops hub", Status: "running"},
			{Name: "test-genie", Description: "AI testing", Status: "stopped"},
		},
	}

	svc := NewScenarioDirectoryService(repo, lister, "")
	results, err := svc.ListSummaries(context.Background())
	if err != nil {
		t.Fatalf("ListSummaries returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(results))
	}

	first := results[0]
	if first.ScenarioName != "swarm-manager" || first.ScenarioDescription != "Ops hub" {
		t.Fatalf("expected metadata to hydrate tracked scenario: %#v", first)
	}

	second := results[1]
	if second.ScenarioName != "test-genie" {
		t.Fatalf("expected CLI-only scenario to be included, got %#v", second)
	}
	if second.TotalExecutions != 0 {
		t.Fatalf("expected zeroed stats for new scenario: %#v", second)
	}
}

func TestScenarioDirectoryServiceListSummariesFallsBackOnListerError(t *testing.T) {
	repo := &fakeScenarioRepo{
		listSummaries: []ScenarioSummary{{ScenarioName: "swarm-manager"}},
	}
	lister := &fakeScenarioLister{err: errors.New("boom")}

	svc := NewScenarioDirectoryService(repo, lister, "")
	results, err := svc.ListSummaries(context.Background())
	if err != nil {
		t.Fatalf("expected fallback list, got error: %v", err)
	}
	if len(results) != 1 || results[0].ScenarioName != "swarm-manager" {
		t.Fatalf("unexpected fallback results: %#v", results)
	}
}

func TestScenarioDirectoryServiceGetSummaryFallsBackToCli(t *testing.T) {
	repo := &fakeScenarioRepo{
		getErr: sql.ErrNoRows,
	}
	lister := &fakeScenarioLister{
		items: []ScenarioMetadata{{Name: "test-genie", Description: "AI testing"}},
	}

	svc := NewScenarioDirectoryService(repo, lister, "")
	result, err := svc.GetSummary(context.Background(), "test-genie")
	if err != nil {
		t.Fatalf("expected CLI fallback, got error: %v", err)
	}
	if result == nil || result.ScenarioName != "test-genie" {
		t.Fatalf("unexpected fallback summary: %#v", result)
	}
	if result.ScenarioDescription != "AI testing" {
		t.Fatalf("expected metadata to populate description: %#v", result)
	}
}

func TestApplyScenarioMetadataCopiesTags(t *testing.T) {
	original := ScenarioSummary{ScenarioName: "demo"}
	meta := ScenarioMetadata{
		Name:        "demo",
		Description: "desc",
		Status:      "running",
		Tags:        []string{"a", "b"},
	}
	result := applyScenarioMetadata(original, meta)
	if !reflect.DeepEqual(result.ScenarioTags, meta.Tags) {
		t.Fatalf("expected tags to copy: %#v", result.ScenarioTags)
	}
	if &result.ScenarioTags[0] == &meta.Tags[0] {
		t.Fatalf("expected tags slice to be cloned")
	}
}

func TestScenarioDirectoryServiceDecoratesTestingCapabilities(t *testing.T) {
	t.Setenv("TEST_GENIE_DISABLE", "1")
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "coverage"), 0o755); err != nil {
		t.Fatalf("mkdir coverage dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "coverage", "run-tests.sh"), []byte("#!/usr/bin/env bash"), 0o755); err != nil {
		t.Fatalf("write run-tests: %v", err)
	}
	repo := &fakeScenarioRepo{
		listSummaries: []ScenarioSummary{{ScenarioName: "demo"}},
	}
	svc := NewScenarioDirectoryService(repo, nil, root)
	results, err := svc.ListSummaries(context.Background())
	if err != nil {
		t.Fatalf("ListSummaries error: %v", err)
	}
	if len(results) != 1 || results[0].Testing == nil || !results[0].Testing.Phased {
		t.Fatalf("expected testing capabilities to be populated, got %#v", results)
	}
	if len(results[0].Testing.Commands) == 0 || results[0].Testing.Commands[0].Type != "phased" {
		t.Fatalf("expected phased command to be included, got %#v", results[0].Testing.Commands)
	}
}

func TestScenarioDirectoryServiceRunScenarioTests(t *testing.T) {
	t.Setenv("TEST_GENIE_DISABLE", "1")
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "coverage"), 0o755); err != nil {
		t.Fatalf("mkdir coverage dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "coverage", "run-tests.sh"), []byte("#!/usr/bin/env bash"), 0o755); err != nil {
		t.Fatalf("write run-tests: %v", err)
	}
	svc := NewScenarioDirectoryService(&fakeScenarioRepo{}, nil, root)
	called := false
	svc.runTests = func(ctx context.Context, caps TestingCapabilities, opts RunScenarioTestOptions) (*TestingRunnerResult, error) {
		called = true
		if opts.Preferred != "" {
			t.Fatalf("expected empty preferred value, got %s", opts.Preferred)
		}
		if len(opts.ExtraArgs) != 0 {
			t.Fatalf("expected no extra args, got %v", opts.ExtraArgs)
		}
		return &TestingRunnerResult{Command: []string{"./coverage/run-tests.sh"}}, nil
	}
	cmd, result, err := svc.RunScenarioTests(context.Background(), "demo", "", nil, "")
	if err != nil {
		t.Fatalf("RunScenarioTests returned error: %v", err)
	}
	if cmd == nil || cmd.Type != "phased" {
		t.Fatalf("expected phased command, got %#v", cmd)
	}
	if result == nil || len(result.Command) == 0 {
		t.Fatalf("expected runner result, got %#v", result)
	}
	if !called {
		t.Fatal("expected testing runner to be invoked")
	}
}

func TestScenarioDirectoryServiceRunScenarioTestsValidation(t *testing.T) {
	root := t.TempDir()
	svc := NewScenarioDirectoryService(&fakeScenarioRepo{}, nil, root)
	if _, _, err := svc.RunScenarioTests(context.Background(), "missing", "", nil, ""); err == nil {
		t.Fatalf("expected error for missing scenario")
	}
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}
	if _, _, err := svc.RunScenarioTests(context.Background(), "demo", "", nil, ""); err == nil {
		t.Fatalf("expected validation error when no tests defined")
	}
}

func TestScenarioDirectoryServiceRunScenarioTestsRunnerFailure(t *testing.T) {
	t.Setenv("TEST_GENIE_DISABLE", "1")
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "coverage"), 0o755); err != nil {
		t.Fatalf("mkdir coverage dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "coverage", "run-tests.sh"), []byte("#!/usr/bin/env bash"), 0o755); err != nil {
		t.Fatalf("write run-tests: %v", err)
	}
	svc := NewScenarioDirectoryService(&fakeScenarioRepo{}, nil, root)
	svc.runTests = func(ctx context.Context, caps TestingCapabilities, opts RunScenarioTestOptions) (*TestingRunnerResult, error) {
		return nil, errors.New("runner failed")
	}
	if _, _, err := svc.RunScenarioTests(context.Background(), "demo", "", nil, ""); err == nil || err.Error() != "runner failed" {
		t.Fatalf("expected runner error, got %v", err)
	}
}
