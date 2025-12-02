package scenarios

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fakeScenarioRepo struct {
	listSummaries []ScenarioSummary
	listErr       error
	getSummary    *ScenarioSummary
	getErr        error
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
				ScenarioName:    "ecosystem-manager",
				PendingRequests: 2,
				TotalRequests:   3,
			},
		},
	}
	lister := &fakeScenarioLister{
		items: []ScenarioMetadata{
			{Name: "ecosystem-manager", Description: "Ops hub", Status: "running"},
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
	if first.ScenarioName != "ecosystem-manager" || first.ScenarioDescription != "Ops hub" {
		t.Fatalf("expected metadata to hydrate tracked scenario: %#v", first)
	}

	second := results[1]
	if second.ScenarioName != "test-genie" {
		t.Fatalf("expected CLI-only scenario to be included, got %#v", second)
	}
	if second.TotalExecutions != 0 || second.TotalRequests != 0 {
		t.Fatalf("expected zeroed stats for new scenario: %#v", second)
	}
}

func TestScenarioDirectoryServiceListSummariesFallsBackOnListerError(t *testing.T) {
	repo := &fakeScenarioRepo{
		listSummaries: []ScenarioSummary{{ScenarioName: "ecosystem-manager"}},
	}
	lister := &fakeScenarioLister{err: errors.New("boom")}

	svc := NewScenarioDirectoryService(repo, lister, "")
	results, err := svc.ListSummaries(context.Background())
	if err != nil {
		t.Fatalf("expected fallback list, got error: %v", err)
	}
	if len(results) != 1 || results[0].ScenarioName != "ecosystem-manager" {
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
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "test"), 0o755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "run-tests.sh"), []byte("#!/usr/bin/env bash"), 0o755); err != nil {
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
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "test"), 0o755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "run-tests.sh"), []byte("#!/usr/bin/env bash"), 0o755); err != nil {
		t.Fatalf("write run-tests: %v", err)
	}
	svc := NewScenarioDirectoryService(&fakeScenarioRepo{}, nil, root)
	called := false
	svc.runTests = func(ctx context.Context, caps TestingCapabilities, preferred string) (*TestingRunnerResult, error) {
		called = true
		if preferred != "" {
			t.Fatalf("expected empty preferred value, got %s", preferred)
		}
		return &TestingRunnerResult{Command: []string{"./test/run-tests.sh"}}, nil
	}
	cmd, result, err := svc.RunScenarioTests(context.Background(), "demo", "")
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
	if _, _, err := svc.RunScenarioTests(context.Background(), "missing", ""); err == nil {
		t.Fatalf("expected error for missing scenario")
	}
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}
	if _, _, err := svc.RunScenarioTests(context.Background(), "demo", ""); err == nil {
		t.Fatalf("expected validation error when no tests defined")
	}
}

func TestScenarioDirectoryServiceRunScenarioTestsRunnerFailure(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "test"), 0o755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "run-tests.sh"), []byte("#!/usr/bin/env bash"), 0o755); err != nil {
		t.Fatalf("write run-tests: %v", err)
	}
	svc := NewScenarioDirectoryService(&fakeScenarioRepo{}, nil, root)
	svc.runTests = func(ctx context.Context, caps TestingCapabilities, preferred string) (*TestingRunnerResult, error) {
		return nil, errors.New("runner failed")
	}
	if _, _, err := svc.RunScenarioTests(context.Background(), "demo", ""); err == nil || err.Error() != "runner failed" {
		t.Fatalf("expected runner error, got %v", err)
	}
}
