package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/testquality/calibration"
)

func TestArchitectureCalibrationEmitsRuleFromObservedFixtureShape(t *testing.T) {
	root := t.TempDir()
	input := calibration.Input{Profile: "go-architecture-v1", Root: ".", Files: []string{"helper.go"}}
	if err := os.WriteFile(filepath.Join(root, "helper.go"), []byte("package p\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(root, "case"+string(rune('a'+i))+"_test.go"), []byte("package p\nfunc TestCase(){}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := evaluateArchitectureCalibration(context.Background(), root, input)
	if err != nil || len(rows) != 1 || rows[0].RuleID != "test-util-root" || rows[0].Status != "violation" {
		t.Fatalf("missing helper calibration = %+v, %v", rows, err)
	}
	if err := os.MkdirAll(filepath.Join(root, "testutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "testutil", "helper.go"), []byte("package testutil\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rows, err = evaluateArchitectureCalibration(context.Background(), root, input)
	if err != nil || rows[0].Status != "checked_clean" {
		t.Fatalf("helper calibration = %+v, %v", rows, err)
	}
}
