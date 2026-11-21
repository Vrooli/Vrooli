package lifecycleinference_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"flow-verifier/internal/lifecycleinference"
)

// BenchmarkAnalyzeLargeOwnerScenarios is the reusable API-operation regression
// probe for the two large DR2 owners previously exceeding the ordinary client
// budget. Run with -benchtime=1x to measure a cold source scan.
func BenchmarkAnalyzeLargeOwnerScenarios(b *testing.B) {
	root := repositoryRoot(b)
	for _, scenario := range []string{"swarm-manager", "scenario-to-cloud"} {
		b.Run(scenario, func(b *testing.B) {
			for range b.N {
				inventory, err := lifecycleinference.AnalyzeContext(context.Background(), root, lifecycleinference.Options{ScenarioID: scenario})
				if err != nil {
					b.Fatal(err)
				}
				if inventory.ScenarioCount != 1 {
					b.Fatalf("expected one focused scenario, got %d", inventory.ScenarioCount)
				}
			}
		})
	}
}

func repositoryRoot(b *testing.B) string {
	b.Helper()
	root, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	for dir := root; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "scenarios", "flow-verifier")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			b.Fatal("repository root not found")
		}
	}
}
