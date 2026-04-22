package graph

import (
	"context"
	"errors"
	"testing"

	"swarm-manager/internal/scenarios"
)

type stubScenarioInventorySource struct {
	items []scenarios.ScenarioSource
	err   error
}

func (s *stubScenarioInventorySource) List(_ context.Context) ([]scenarios.ScenarioSource, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

func TestScenarioSourceAdapterList_NormalizesStatuses(t *testing.T) {
	adapter := NewScenarioSourceAdapter(&stubScenarioInventorySource{
		items: []scenarios.ScenarioSource{
			{Name: "running-app", Status: "running"},
			{Name: "available-app", Status: "available"},
			{Name: "error-app", Status: "error"},
			{Name: "mystery-app", Status: "mystery"},
		},
	})

	result, err := adapter.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []ScenarioEntry{
		{Name: "running-app", Status: "running"},
		{Name: "available-app", Status: "stopped"},
		{Name: "error-app", Status: "error"},
		{Name: "mystery-app", Status: "unknown"},
	}
	if len(result) != len(expected) {
		t.Fatalf("expected %d scenarios, got %d", len(expected), len(result))
	}
	for index, entry := range result {
		if entry != expected[index] {
			t.Fatalf("expected entry %d to be %+v, got %+v", index, expected[index], entry)
		}
	}
}

func TestScenarioSourceAdapterList_PropagatesErrors(t *testing.T) {
	expectedErr := errors.New("boom")
	adapter := NewScenarioSourceAdapter(&stubScenarioInventorySource{err: expectedErr})

	_, err := adapter.List(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
