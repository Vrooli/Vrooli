package pathutil

import (
	"reflect"
	"testing"
)

func TestScenarioFromRepoPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
		ok       bool
	}{
		{name: "direct scenario file", path: "scenarios/swarm-manager/api/main.go", expected: "swarm-manager", ok: true},
		{name: "scenario root", path: "scenarios/web-console", expected: "web-console", ok: true},
		{name: "shared package", path: "packages/proto/README.md", ok: false},
		{name: "empty", path: " ", ok: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, ok := ScenarioFromRepoPath(tc.path)
			if ok != tc.ok {
				t.Fatalf("expected ok=%t, got %t", tc.ok, ok)
			}
			if actual != tc.expected {
				t.Fatalf("expected scenario %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestGroupChangedPaths(t *testing.T) {
	t.Parallel()

	scope := GroupChangedPaths([]string{
		"scenarios/swarm-manager/api/main.go",
		"packages/proto/schemas/swarm-manager/v1/domain/execution.proto",
		"scenarios/web-console/ui/src/App.tsx",
		"scenarios/swarm-manager/api/main.go",
		"README.md",
	})

	expectedDirect := map[string][]string{
		"swarm-manager": {"scenarios/swarm-manager/api/main.go"},
		"web-console":   {"scenarios/web-console/ui/src/App.tsx"},
	}
	expectedShared := []string{
		"README.md",
		"packages/proto/schemas/swarm-manager/v1/domain/execution.proto",
	}

	if !reflect.DeepEqual(scope.DirectScenarioPaths, expectedDirect) {
		t.Fatalf("unexpected direct scenario paths: %#v", scope.DirectScenarioPaths)
	}
	if !reflect.DeepEqual(scope.SharedPaths, expectedShared) {
		t.Fatalf("unexpected shared paths: %#v", scope.SharedPaths)
	}
}

func TestUniqueSortedStrings(t *testing.T) {
	t.Parallel()

	actual := UniqueSortedStrings([]string{"beta", "alpha", "beta", " ", "gamma"})
	expected := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}
