package runtime

import "testing"

func TestBuildDependenciesRequiresConfig(t *testing.T) {
	if _, err := BuildDependencies(nil); err == nil {
		t.Fatal("expected error when config is nil")
	}
}
