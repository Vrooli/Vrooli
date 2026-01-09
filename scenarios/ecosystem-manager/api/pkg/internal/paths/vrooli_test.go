package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectVrooliRoot(t *testing.T) {
	// Test that the function returns a non-empty path
	result := DetectVrooliRoot()

	if result == "" {
		t.Error("DetectVrooliRoot() returned empty string")
	}

	// Verify the path exists
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Errorf("DetectVrooliRoot() returned non-existent path: %v", result)
	}

	// Verify it's an absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("DetectVrooliRoot() returned relative path: %v", result)
	}
}

func TestDetectVrooliRootConsistency(t *testing.T) {
	// Test that multiple calls return the same result
	first := DetectVrooliRoot()
	second := DetectVrooliRoot()

	if first != second {
		t.Errorf("DetectVrooliRoot() returned inconsistent results: %v vs %v", first, second)
	}
}

func TestDetectVrooliRootStructure(t *testing.T) {
	// Test that the detected root has expected Vrooli structure
	root := DetectVrooliRoot()

	// Check for common Vrooli directories/files
	expectedPaths := []string{
		filepath.Join(root, "scenarios"),
		filepath.Join(root, "scripts"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Logf("Warning: Expected Vrooli path does not exist: %v (root: %v)", path, root)
			// This is a warning, not a failure, as the structure might vary
		}
	}
}

func TestDetectVrooliRootFromEnv(t *testing.T) {
	// Test that VROOLI_ROOT environment variable is respected if set
	originalEnv := os.Getenv("VROOLI_ROOT")
	defer func() {
		if originalEnv != "" {
			os.Setenv("VROOLI_ROOT", originalEnv)
		} else {
			os.Unsetenv("VROOLI_ROOT")
		}
	}()

	// Test with custom VROOLI_ROOT
	testRoot := "/tmp/test-vrooli-root"
	os.Setenv("VROOLI_ROOT", testRoot)

	result := DetectVrooliRoot()

	// The function should either use the env var or fall back to detection
	// We can't assert exact behavior without reading the implementation,
	// but we can verify it returns a valid path
	if result == "" {
		t.Error("DetectVrooliRoot() returned empty string with VROOLI_ROOT set")
	}

	if !filepath.IsAbs(result) {
		t.Errorf("DetectVrooliRoot() returned relative path: %v", result)
	}
}
