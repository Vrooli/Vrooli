package catalogcoverage

import (
	"path/filepath"
	"testing"
)

func TestCartesianChartsHasSubstantiveExperienceContract(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	impls, err := LoadImplementations(filepath.Join(root, "scenarios", "react-component-library", "library"))
	if err != nil {
		t.Fatal(err)
	}
	for _, impl := range impls {
		if impl.Name == "CartesianCharts" {
			if !impl.ExperienceRegistered || impl.ExperienceVacuous {
				t.Fatalf("cartesian charts experience = registered %v vacuous %v", impl.ExperienceRegistered, impl.ExperienceVacuous)
			}
			return
		}
	}
	t.Fatal("CartesianCharts implementation not found")
}
