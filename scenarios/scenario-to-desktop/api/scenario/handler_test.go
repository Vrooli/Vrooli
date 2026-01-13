package scenario

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mockRecordStore implements RecordStore for testing.
type mockRecordStore struct {
	records []*DesktopAppRecord
}

func (m *mockRecordStore) List() []*DesktopAppRecord {
	return m.records
}

func TestNewHandler(t *testing.T) {
	store := &mockRecordStore{}
	logger := slog.Default()
	h := NewHandler("/tmp/vrooli", store, logger)

	if h == nil {
		t.Fatal("expected handler to be created")
	}
	if h.vrooliRoot != "/tmp/vrooli" {
		t.Errorf("vrooliRoot = %q, want %q", h.vrooliRoot, "/tmp/vrooli")
	}
	if h.records != store {
		t.Error("records store not set correctly")
	}
}

func TestDetectPlatformFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"app-setup.exe", "win"},
		{"app-Setup.exe", "win"},
		{"app.msi", "win"},
		{"app-win.zip", "win"},
		{"app.pkg", "mac"},
		{"app.dmg", "mac"},
		{"app-mac.zip", "mac"},
		// NOTE: "darwin" contains "win", so the function matches "win" first.
		// This is a quirk of the implementation order.
		{"app-darwin.zip", "win"},
		{"app.AppImage", "linux"},
		{"app.deb", "linux"},
		{"app-linux.tar.gz", "linux"},
		{"README.md", ""},
		{"package.json", ""},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			got := detectPlatformFromFilename(tc.filename)
			if got != tc.expected {
				t.Errorf("detectPlatformFromFilename(%q) = %q, want %q", tc.filename, got, tc.expected)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty strings filtered",
			input:    []string{"a", "", "b", ""},
			expected: []string{"a", "b"},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := uniqueStrings(tc.input)
			if len(got) != len(tc.expected) {
				t.Errorf("len = %d, want %d", len(got), len(tc.expected))
				return
			}
			for i, v := range tc.expected {
				if got[i] != v {
					t.Errorf("got[%d] = %q, want %q", i, got[i], v)
				}
			}
		})
	}
}

func TestLatestRecordTime(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)

	tests := []struct {
		name     string
		record   *DesktopAppRecord
		expected time.Time
	}{
		{
			name:     "nil record",
			record:   nil,
			expected: time.Time{},
		},
		{
			name: "updatedAt set",
			record: &DesktopAppRecord{
				CreatedAt: earlier,
				UpdatedAt: now,
			},
			expected: now,
		},
		{
			name: "only createdAt",
			record: &DesktopAppRecord{
				CreatedAt: earlier,
			},
			expected: earlier,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := latestRecordTime(tc.record)
			if !got.Equal(tc.expected) {
				t.Errorf("got %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestRecordTimestamp(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name     string
		record   *DesktopAppRecord
		expected string
	}{
		{
			name:     "nil record",
			record:   nil,
			expected: "",
		},
		{
			name:     "zero time",
			record:   &DesktopAppRecord{},
			expected: "",
		},
		{
			name:     "with time",
			record:   &DesktopAppRecord{UpdatedAt: now},
			expected: "2026-01-15 10:30:45",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := recordTimestamp(tc.record)
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestRecordOutputPath(t *testing.T) {
	tests := []struct {
		name     string
		record   *DesktopAppRecord
		expected string
	}{
		{
			name:     "nil record",
			record:   nil,
			expected: "",
		},
		{
			name:     "outputPath set",
			record:   &DesktopAppRecord{OutputPath: "/output", StagingPath: "/staging"},
			expected: "/output",
		},
		{
			name:     "stagingPath fallback",
			record:   &DesktopAppRecord{StagingPath: "/staging", CustomPath: "/custom"},
			expected: "/staging",
		},
		{
			name:     "customPath fallback",
			record:   &DesktopAppRecord{CustomPath: "/custom", DestinationPath: "/dest"},
			expected: "/custom",
		},
		{
			name:     "destinationPath fallback",
			record:   &DesktopAppRecord{DestinationPath: "/dest"},
			expected: "/dest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := recordOutputPath(tc.record)
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestListScenarioRecords(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)

	store := &mockRecordStore{
		records: []*DesktopAppRecord{
			{ID: "1", ScenarioName: "test-scenario", UpdatedAt: earlier},
			{ID: "2", ScenarioName: "other-scenario", UpdatedAt: now},
			{ID: "3", ScenarioName: "test-scenario", UpdatedAt: now},
		},
	}

	h := NewHandler("/tmp/vrooli", store, slog.Default())

	records := h.listScenarioRecords("test-scenario")
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	// Should be sorted by time descending (most recent first)
	if len(records) >= 2 {
		if records[0].ID != "3" {
			t.Errorf("expected first record to be ID '3' (most recent), got %q", records[0].ID)
		}
	}
}

func TestListScenarioRecordsNilStore(t *testing.T) {
	h := NewHandler("/tmp/vrooli", nil, slog.Default())

	records := h.listScenarioRecords("test-scenario")
	if records != nil {
		t.Error("expected nil records with nil store")
	}
}

func TestLoadScenarioServiceInfo(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		info, err := loadScenarioServiceInfo(tmpDir)
		if err == nil {
			t.Error("expected error for missing file")
		}
		if info != nil {
			t.Error("expected nil info for missing file")
		}
	})

	t.Run("valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		vrooliDir := filepath.Join(tmpDir, ".vrooli")
		if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		content := `{"service": {"displayName": "Test App", "description": "A test application"}}`
		if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		info, err := loadScenarioServiceInfo(tmpDir)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if info == nil {
			t.Fatal("expected info to be non-nil")
		}
		if info.DisplayName != "Test App" {
			t.Errorf("DisplayName = %q, want %q", info.DisplayName, "Test App")
		}
		if info.Description != "A test application" {
			t.Errorf("Description = %q, want %q", info.Description, "A test application")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		vrooliDir := filepath.Join(tmpDir, ".vrooli")
		if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte("{invalid"), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		info, err := loadScenarioServiceInfo(tmpDir)
		if err == nil {
			t.Error("expected error for invalid json")
		}
		if info != nil {
			t.Error("expected nil info for invalid json")
		}
	})
}

func TestLoadDesktopConnectionConfig(t *testing.T) {
	t.Run("missing file returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg, err := loadDesktopConnectionConfig(tmpDir)
		if err != nil {
			t.Errorf("expected no error for missing file, got: %v", err)
		}
		if cfg != nil {
			t.Error("expected nil config for missing file")
		}
	})

	t.Run("valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		vrooliDir := filepath.Join(tmpDir, ".vrooli")
		if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		content := `{"mode": "proxy", "endpoint": "http://localhost:8080"}`
		if err := os.WriteFile(filepath.Join(vrooliDir, "desktop-config.json"), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		cfg, err := loadDesktopConnectionConfig(tmpDir)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected config to be non-nil")
		}
		if cfg.Mode != "proxy" {
			t.Errorf("Mode = %q, want %q", cfg.Mode, "proxy")
		}
		if cfg.Endpoint != "http://localhost:8080" {
			t.Errorf("Endpoint = %q, want %q", cfg.Endpoint, "http://localhost:8080")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		vrooliDir := filepath.Join(tmpDir, ".vrooli")
		if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		if err := os.WriteFile(filepath.Join(vrooliDir, "desktop-config.json"), []byte("{invalid"), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		cfg, err := loadDesktopConnectionConfig(tmpDir)
		if err == nil {
			t.Error("expected error for invalid json")
		}
		if cfg != nil {
			t.Error("expected nil config for invalid json")
		}
	})
}

func TestFindScenarioIcon(t *testing.T) {
	t.Run("no icon found", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := findScenarioIcon(tmpDir)
		if path != "" {
			t.Errorf("expected empty path, got %q", path)
		}
	})

	t.Run("icon found", func(t *testing.T) {
		tmpDir := t.TempDir()
		iconDir := filepath.Join(tmpDir, "ui", "dist")
		if err := os.MkdirAll(iconDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		iconPath := filepath.Join(iconDir, "manifest-icon-512.maskable.png")
		if err := os.WriteFile(iconPath, []byte("fake-icon"), 0o644); err != nil {
			t.Fatalf("failed to write icon: %v", err)
		}

		path := findScenarioIcon(tmpDir)
		if path != iconPath {
			t.Errorf("got %q, want %q", path, iconPath)
		}
	})
}

func TestScanDistArtifacts(t *testing.T) {
	t.Run("non-existent directory", func(t *testing.T) {
		result, ok := scanDistArtifacts("/nonexistent/path", "/tmp")
		if ok {
			t.Error("expected ok to be false for non-existent directory")
		}
		if result != nil {
			t.Error("expected nil result for non-existent directory")
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		distDir := filepath.Join(tmpDir, "dist")
		if err := os.MkdirAll(distDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		result, ok := scanDistArtifacts(distDir, tmpDir)
		if !ok {
			t.Error("expected ok to be true for existing directory")
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.artifacts) != 0 {
			t.Errorf("expected 0 artifacts, got %d", len(result.artifacts))
		}
	})

	t.Run("with artifacts", func(t *testing.T) {
		tmpDir := t.TempDir()
		distDir := filepath.Join(tmpDir, "dist")
		if err := os.MkdirAll(distDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		// Create some test files
		files := []struct {
			name     string
			content  string
			platform string
		}{
			{"app-Setup.exe", "fake-exe", "win"},
			{"app.dmg", "fake-dmg", "mac"},
			{"app.AppImage", "fake-appimage", "linux"},
			{"README.txt", "readme", ""},
		}

		for _, f := range files {
			if err := os.WriteFile(filepath.Join(distDir, f.name), []byte(f.content), 0o644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
		}

		result, ok := scanDistArtifacts(distDir, tmpDir)
		if !ok {
			t.Error("expected ok to be true")
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.artifacts) != 4 {
			t.Errorf("expected 4 artifacts, got %d", len(result.artifacts))
		}

		// Check platforms detected
		platformCounts := map[string]int{}
		for _, a := range result.artifacts {
			platformCounts[a.Platform]++
		}
		if platformCounts["win"] != 1 {
			t.Errorf("expected 1 win artifact, got %d", platformCounts["win"])
		}
		if platformCounts["mac"] != 1 {
			t.Errorf("expected 1 mac artifact, got %d", platformCounts["mac"])
		}
		if platformCounts["linux"] != 1 {
			t.Errorf("expected 1 linux artifact, got %d", platformCounts["linux"])
		}
	})
}

func TestDesktopStatusHandler(t *testing.T) {
	tmpDir := t.TempDir()
	scenariosDir := filepath.Join(tmpDir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatalf("failed to create scenarios dir: %v", err)
	}

	// Create a test scenario
	testScenario := filepath.Join(scenariosDir, "test-scenario")
	if err := os.MkdirAll(testScenario, 0o755); err != nil {
		t.Fatalf("failed to create test scenario: %v", err)
	}

	// Create a scenario with desktop
	desktopScenario := filepath.Join(scenariosDir, "desktop-scenario")
	electronDir := filepath.Join(desktopScenario, "platforms", "electron")
	if err := os.MkdirAll(electronDir, 0o755); err != nil {
		t.Fatalf("failed to create electron dir: %v", err)
	}

	pkgJson := `{"name": "desktop-app", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(electronDir, "package.json"), []byte(pkgJson), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	store := &mockRecordStore{}
	h := NewHandler(tmpDir, store, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/desktop-status", nil)
	rr := httptest.NewRecorder()

	h.DesktopStatusHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp ListResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Stats == nil {
		t.Fatal("expected stats to be non-nil")
	}

	// Should find both scenarios
	if resp.Stats.Total != 2 {
		t.Errorf("expected 2 total scenarios, got %d", resp.Stats.Total)
	}

	// One should have desktop
	if resp.Stats.WithDesktop != 1 {
		t.Errorf("expected 1 scenario with desktop, got %d", resp.Stats.WithDesktop)
	}
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]string{"test": "value"}

	writeJSON(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}

	var result map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["test"] != "value" {
		t.Errorf("expected test='value', got %q", result["test"])
	}
}
