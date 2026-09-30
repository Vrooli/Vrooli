package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSafetyAdmissionGate keeps the API test tree from regressing into real
// repository setup. Tests may inspect filesystem fixtures and use read-only
// Git records, but mutation belongs only to the human writer path.
func TestSafetyAdmissionGate(t *testing.T) {
	root := "."
	forbidden := []string{
		"exec.Command(\"git\", \"init\"",
		"exec.Command(\"git\", \"add\"",
		"exec.Command(\"git\", \"commit\"",
		"exec.Command(\"git\", \"mv\"",
		"RunGitCommand(",
		"SetupTestRepo(",
	}
	var violations []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Base(path) == "safety_admission_gate_test.go" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(data)
		for _, needle := range forbidden {
			if strings.Contains(text, needle) {
				violations = append(violations, path+": "+needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan API test tree: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("unsafe Git test effects admitted:\n%s", strings.Join(violations, "\n"))
	}
}
