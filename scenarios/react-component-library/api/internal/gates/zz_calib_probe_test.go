package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestZZCalibProbe(t *testing.T) {
	if os.Getenv("RCL_CALIB_PROBE") == "" {
		t.Skip("set RCL_CALIB_PROBE=1")
	}
	wd, _ := os.Getwd()
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))
	fixtures, err := loadCalibrationFixtures(root, "types")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		overlay, cleanup, mErr := materializeFixture(root, "types", fixture)
		if mErr != nil {
			t.Fatal(mErr)
		}
		result, runErr := ValidateTypes(overlay)
		cleanup()
		fmt.Printf("runErr=%v findings=%d runnerErrors=%d\n", runErr, len(result.Findings), len(result.RunnerError))
		for _, f := range result.Findings {
			fmt.Printf("  FINDING asset=%q %.100s\n", f.AssetID, f.Message)
		}
		for _, f := range result.RunnerError {
			fmt.Printf("  RUNNER-ERROR %.1200s\n", f.Message)
		}
	}
}
