package scenarios

import (
	"context"
	"database/sql"
	"errors"
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

	svc := NewScenarioDirectoryService(repo, lister)
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

	svc := NewScenarioDirectoryService(repo, lister)
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

	svc := NewScenarioDirectoryService(repo, lister)
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
