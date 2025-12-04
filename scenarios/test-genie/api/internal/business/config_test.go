package business

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// LoadExpectations Tests
// =============================================================================

func TestLoadExpectations_DefaultsWhenFileNotExists(t *testing.T) {
	tempDir := t.TempDir()

	exp, err := LoadExpectations(tempDir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}

	assertDefaultExpectations(t, exp)
}

func TestLoadExpectations_DefaultsWhenNoBusinessSection(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Write config without business section
	configContent := `{"structure": {"additional_dirs": []}}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	exp, err := LoadExpectations(tempDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	assertDefaultExpectations(t, exp)
}

func TestLoadExpectations_ParsesAllFields(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configContent := `{
		"business": {
			"require_modules": false,
			"require_index": false,
			"min_coverage_percent": 90,
			"error_coverage_percent": 70
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	exp, err := LoadExpectations(tempDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if exp.RequireModules {
		t.Error("expected RequireModules to be false")
	}
	if exp.RequireIndex {
		t.Error("expected RequireIndex to be false")
	}
	if exp.MinCoveragePercent != 90 {
		t.Errorf("expected MinCoveragePercent=90, got %d", exp.MinCoveragePercent)
	}
	if exp.ErrorCoveragePercent != 70 {
		t.Errorf("expected ErrorCoveragePercent=70, got %d", exp.ErrorCoveragePercent)
	}
}

func TestLoadExpectations_PartialOverrides(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Only override one field
	configContent := `{"business": {"min_coverage_percent": 95}}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	exp, err := LoadExpectations(tempDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Overridden field
	if exp.MinCoveragePercent != 95 {
		t.Errorf("expected MinCoveragePercent=95, got %d", exp.MinCoveragePercent)
	}

	// Defaults should be preserved
	if !exp.RequireModules {
		t.Error("expected RequireModules to remain true (default)")
	}
	if !exp.RequireIndex {
		t.Error("expected RequireIndex to remain true (default)")
	}
	if exp.ErrorCoveragePercent != 60 {
		t.Errorf("expected ErrorCoveragePercent=60 (default), got %d", exp.ErrorCoveragePercent)
	}
}

func TestLoadExpectations_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configContent := `{"business": {invalid json`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := LoadExpectations(tempDir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadExpectations_UnreadableFile(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configPath := filepath.Join(vrooliDir, "testing.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Make file unreadable
	if err := os.Chmod(configPath, 0o000); err != nil {
		t.Skipf("cannot change file permissions on this system: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(configPath, 0o644) // Restore for cleanup
	})

	_, err := LoadExpectations(tempDir)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

// =============================================================================
// DefaultExpectations Tests
// =============================================================================

// Note: TestDefaultExpectations is in runner_test.go

func TestDefaultExpectations_ReturnsNewInstance(t *testing.T) {
	exp1 := DefaultExpectations()
	exp2 := DefaultExpectations()

	// Modify exp1
	exp1.RequireModules = false
	exp1.MinCoveragePercent = 50

	// exp2 should be unaffected
	if !exp2.RequireModules {
		t.Error("expected exp2 to be independent of exp1")
	}
	if exp2.MinCoveragePercent != 80 {
		t.Errorf("expected exp2.MinCoveragePercent=80, got %d", exp2.MinCoveragePercent)
	}
}

// =============================================================================
// Helpers
// =============================================================================

func assertDefaultExpectations(t *testing.T, exp *Expectations) {
	t.Helper()

	if exp == nil {
		t.Fatal("expected non-nil expectations")
	}
	if !exp.RequireModules {
		t.Error("expected RequireModules to default to true")
	}
	if !exp.RequireIndex {
		t.Error("expected RequireIndex to default to true")
	}
	if exp.MinCoveragePercent != 80 {
		t.Errorf("expected MinCoveragePercent to default to 80, got %d", exp.MinCoveragePercent)
	}
	if exp.ErrorCoveragePercent != 60 {
		t.Errorf("expected ErrorCoveragePercent to default to 60, got %d", exp.ErrorCoveragePercent)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkLoadExpectations_FileNotExists(b *testing.B) {
	tempDir := b.TempDir()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		LoadExpectations(tempDir)
	}
}

func BenchmarkLoadExpectations_WithFile(b *testing.B) {
	tempDir := b.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0o755)

	configContent := `{"business": {"require_modules": true, "min_coverage_percent": 80}}`
	os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(configContent), 0o644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadExpectations(tempDir)
	}
}

func BenchmarkDefaultExpectations(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DefaultExpectations()
	}
}
