package rewrite

import (
	"errors"
	"testing"
)

func TestMemoryPlanStore_SaveGet(t *testing.T) {
	s := NewMemoryPlanStore()
	plan := Plan{ID: "abc", ProjectPath: "/abs/proj", Operations: []Operation{
		{FileMove: &FileMove{FromPath: "a", ToPath: "b"}},
	}}
	if err := s.Save(plan); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Get("/abs/proj", "abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "abc" || got.ProjectPath != "/abs/proj" || len(got.Operations) != 1 {
		t.Errorf("Get returned wrong plan: %+v", got)
	}
}

func TestMemoryPlanStore_GetMissing(t *testing.T) {
	s := NewMemoryPlanStore()
	_, err := s.Get("/abs/proj", "nope")
	if !errors.Is(err, ErrPlanNotFound) {
		t.Errorf("want ErrPlanNotFound, got %v", err)
	}
}

func TestMemoryPlanStore_ScenarioScope(t *testing.T) {
	s := NewMemoryPlanStore()
	plan := Plan{ID: "abc", ProjectPath: "/abs/one", Operations: nil}
	if err := s.Save(plan); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Same PlanID under a different project must not collide.
	_, err := s.Get("/abs/two", "abc")
	if !errors.Is(err, ErrPlanNotFound) {
		t.Errorf("project scope should isolate plans; got %v", err)
	}
}
