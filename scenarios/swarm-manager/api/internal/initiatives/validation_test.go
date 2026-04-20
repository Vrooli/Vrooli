package initiatives

import (
	"strings"
	"testing"
)

func TestValidatePriority(t *testing.T) {
	cases := []struct {
		in   int
		want bool
	}{
		{0, true},
		{1, true},
		{5, true},
		{10, true},
		{-1, false},
		{11, false},
		{100, false},
	}
	for _, c := range cases {
		if got := ValidatePriority(c.in); got != c.want {
			t.Errorf("ValidatePriority(%d) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNormalizeDependsOn(t *testing.T) {
	got := normalizeDependsOn([]string{" a ", "b", "", "a", "c"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if normalizeDependsOn(nil) != nil {
		t.Error("nil input should return nil")
	}
	if normalizeDependsOn([]string{"", "  "}) != nil {
		t.Error("all-blank should return nil")
	}
}

func TestService_Create_WithPriorityAndDependsOn(t *testing.T) {
	svc := newTestService(t, nil)

	if _, err := svc.Create(CreateRequest{Name: "base", Title: "Base"}); err != nil {
		t.Fatalf("base Create failed: %v", err)
	}
	init, err := svc.Create(CreateRequest{
		Name: "dep", Title: "Depends", Priority: 3, DependsOn: []string{"base"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if init.Priority != 3 {
		t.Errorf("priority = %d, want 3", init.Priority)
	}
	if len(init.DependsOn) != 1 || init.DependsOn[0] != "base" {
		t.Errorf("depends_on = %v, want [base]", init.DependsOn)
	}
}

func TestService_Create_InvalidPriority(t *testing.T) {
	svc := newTestService(t, nil)
	_, err := svc.Create(CreateRequest{Name: "x", Title: "X", Priority: 99})
	if err == nil || !strings.Contains(err.Error(), "priority") {
		t.Fatalf("expected priority error, got %v", err)
	}
}

func TestService_Create_UnknownDependency(t *testing.T) {
	svc := newTestService(t, nil)
	_, err := svc.Create(CreateRequest{Name: "x", Title: "X", DependsOn: []string{"ghost"}})
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("expected unknown-dep error, got %v", err)
	}
}

func TestService_Create_SelfDependency(t *testing.T) {
	svc := newTestService(t, nil)
	_, err := svc.Create(CreateRequest{Name: "x", Title: "X", DependsOn: []string{"x"}})
	if err == nil || !strings.Contains(err.Error(), "itself") {
		t.Fatalf("expected self-dep error, got %v", err)
	}
}

func TestService_Update_CycleRejected(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Create(CreateRequest{Name: "a", Title: "A"}); err != nil {
		t.Fatalf("A: %v", err)
	}
	if _, err := svc.Create(CreateRequest{Name: "b", Title: "B", DependsOn: []string{"a"}}); err != nil {
		t.Fatalf("B: %v", err)
	}
	// Now make A depend on B — would form a cycle a -> b -> a.
	deps := []string{"b"}
	_, err := svc.Update("a", UpdateRequest{DependsOn: &deps})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestService_Update_ClearDependencies(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Create(CreateRequest{Name: "a", Title: "A"}); err != nil {
		t.Fatalf("A: %v", err)
	}
	if _, err := svc.Create(CreateRequest{Name: "b", Title: "B", DependsOn: []string{"a"}}); err != nil {
		t.Fatalf("B: %v", err)
	}
	empty := []string{}
	updated, err := svc.Update("b", UpdateRequest{DependsOn: &empty})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(updated.DependsOn) != 0 {
		t.Errorf("depends_on should be empty, got %v", updated.DependsOn)
	}
}

func TestUpdateRequest_HasChanges_PriorityOrDeps(t *testing.T) {
	p := 5
	if !(UpdateRequest{Priority: &p}).HasChanges() {
		t.Error("Priority change should be counted")
	}
	deps := []string{"x"}
	if !(UpdateRequest{DependsOn: &deps}).HasChanges() {
		t.Error("DependsOn change should be counted")
	}
}
