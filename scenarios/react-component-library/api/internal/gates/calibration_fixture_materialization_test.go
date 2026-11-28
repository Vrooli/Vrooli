package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEveryAuthoredCalibrationFixtureBuildsAndCleansItsOverlay(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	calibrationRoot := filepath.Join(root, "scenarios", "react-component-library", "catalog", "calibration")
	entries, err := os.ReadDir(calibrationRoot)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gate := entry.Name()
		fixtures, err := loadCalibrationFixtures(root, gate)
		if err != nil {
			t.Fatalf("load %s fixtures: %v", gate, err)
		}
		for _, fixture := range fixtures {
			if fixture.Runner == "external" {
				continue
			}
			t.Run(gate+"/"+fixture.AssetID, func(t *testing.T) {
				overlay, cleanup, err := materializeFixture(root, gate, fixture)
				if err != nil {
					t.Fatalf("materialize authored fixture: %v", err)
				}
				if filepath.Dir(overlay) != os.TempDir() {
					cleanup()
					t.Fatalf("fixture overlay escaped the private temp root: %q", overlay)
				}
				if _, err := os.Stat(overlay); err != nil {
					cleanup()
					t.Fatalf("materialized overlay is unavailable: %v", err)
				}
				cleanup()
				if _, err := os.Lstat(overlay); !os.IsNotExist(err) {
					t.Fatalf("fixture cleanup left its overlay behind: %v", err)
				}
			})
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no authored calibration fixtures were validated")
	}
}
