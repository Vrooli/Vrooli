package lighthouse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FileNotExists(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Enabled {
		t.Error("expected config to be disabled when file doesn't exist")
	}
}

func TestLoadConfig_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	vrooliDir := filepath.Join(dir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	configContent := `{
		"enabled": true,
		"pages": [
			{
				"id": "home",
				"path": "/",
				"thresholds": {
					"performance": { "error": 0.75, "warn": 0.85 }
				}
			}
		]
	}`

	if err := os.WriteFile(filepath.Join(vrooliDir, "lighthouse.json"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Enabled {
		t.Error("expected config to be enabled")
	}
	if len(cfg.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(cfg.Pages))
	}
	if cfg.Pages[0].ID != "home" {
		t.Errorf("expected page ID 'home', got %q", cfg.Pages[0].ID)
	}
	if cfg.Pages[0].Path != "/" {
		t.Errorf("expected page path '/', got %q", cfg.Pages[0].Path)
	}

	threshold := cfg.Pages[0].Thresholds["performance"]
	if threshold.Error != 0.75 {
		t.Errorf("expected error threshold 0.75, got %f", threshold.Error)
	}
	if threshold.Warn != 0.85 {
		t.Errorf("expected warn threshold 0.85, got %f", threshold.Warn)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	vrooliDir := filepath.Join(dir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(vrooliDir, "lighthouse.json"), []byte("invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestConfig_Validate_Disabled(t *testing.T) {
	cfg := &Config{Enabled: false}
	if err := cfg.Validate(); err != nil {
		t.Errorf("disabled config should be valid: %v", err)
	}
}

func TestConfig_Validate_NoPages(t *testing.T) {
	cfg := &Config{Enabled: true, Pages: nil}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for enabled config with no pages")
	}
}

func TestConfig_Validate_MissingPageID(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{ID: "", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}}},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for page without ID")
	}
}

func TestConfig_Validate_MissingPagePath(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{ID: "home", Path: "", Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}}},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for page without path")
	}
}

func TestConfig_Validate_MissingThresholds(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{ID: "home", Path: "/", Thresholds: nil},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for page without thresholds")
	}
}

func TestConfig_Validate_InvalidThresholdRange(t *testing.T) {
	tests := []struct {
		name      string
		threshold ThresholdLevel
		wantErr   bool
	}{
		{"valid", ThresholdLevel{Error: 0.5, Warn: 0.7}, false},
		{"error too low", ThresholdLevel{Error: -0.1, Warn: 0.7}, true},
		{"error too high", ThresholdLevel{Error: 1.5, Warn: 0.7}, true},
		{"warn too low", ThresholdLevel{Error: 0.5, Warn: -0.1}, true},
		{"warn too high", ThresholdLevel{Error: 0.5, Warn: 1.5}, true},
		{"error > warn", ThresholdLevel{Error: 0.8, Warn: 0.6}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Enabled: true,
				Pages: []PageConfig{
					{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": tt.threshold}},
				},
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestConfig_GetTimeout_Default(t *testing.T) {
	cfg := &Config{}
	if timeout := cfg.GetTimeout(); timeout != 90000 {
		t.Errorf("expected default timeout 90000, got %d", timeout)
	}
}

func TestConfig_GetTimeout_Custom(t *testing.T) {
	cfg := &Config{
		GlobalOptions: &GlobalOptions{TimeoutMs: 120000},
	}
	if timeout := cfg.GetTimeout(); timeout != 120000 {
		t.Errorf("expected timeout 120000, got %d", timeout)
	}
}

func TestConfig_GetCategories_Default(t *testing.T) {
	cfg := &Config{}
	categories := cfg.GetCategories()
	expected := []string{"performance", "accessibility", "best-practices", "seo"}
	if len(categories) != len(expected) {
		t.Errorf("expected %d categories, got %d", len(expected), len(categories))
	}
}

func TestConfig_GetCategories_Custom(t *testing.T) {
	cfg := &Config{
		GlobalOptions: &GlobalOptions{
			Lighthouse: &LighthouseSettings{
				Settings: &LighthouseRunnerSettings{
					OnlyCategories: []string{"performance"},
				},
			},
		},
	}
	categories := cfg.GetCategories()
	if len(categories) != 1 || categories[0] != "performance" {
		t.Errorf("expected [performance], got %v", categories)
	}
}

func TestConfig_BuildLighthouseConfig(t *testing.T) {
	cfg := &Config{
		GlobalOptions: &GlobalOptions{
			Lighthouse: &LighthouseSettings{
				Extends: "lighthouse:default",
				Settings: &LighthouseRunnerSettings{
					OnlyCategories:   []string{"performance", "accessibility"},
					ThrottlingMethod: "simulate",
					FormFactor:       "desktop",
				},
			},
		},
	}

	result := cfg.BuildLighthouseConfig()

	if result["extends"] != "lighthouse:default" {
		t.Errorf("expected extends 'lighthouse:default', got %v", result["extends"])
	}

	settings, ok := result["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("expected settings to be a map")
	}

	if settings["throttlingMethod"] != "simulate" {
		t.Errorf("expected throttlingMethod 'simulate', got %v", settings["throttlingMethod"])
	}
	if settings["formFactor"] != "desktop" {
		t.Errorf("expected formFactor 'desktop', got %v", settings["formFactor"])
	}
}

// Tests for new functionality

func TestConfig_Validate_InvalidViewport(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{
				ID:        "home",
				Path:      "/",
				Viewport:  "tablet", // invalid
				Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}},
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid viewport")
	}
}

func TestConfig_Validate_ValidViewport(t *testing.T) {
	viewports := []string{"", "desktop", "mobile"}
	for _, viewport := range viewports {
		t.Run(viewport, func(t *testing.T) {
			cfg := &Config{
				Enabled: true,
				Pages: []PageConfig{
					{
						ID:        "home",
						Path:      "/",
						Viewport:  viewport,
						Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}},
					},
				},
			}
			if err := cfg.Validate(); err != nil {
				t.Errorf("unexpected error for viewport %q: %v", viewport, err)
			}
		})
	}
}

func TestConfig_Validate_InvalidWaitForMs(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{
				ID:         "home",
				Path:       "/",
				WaitForMs:  -100, // invalid
				Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}},
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative waitForMs")
	}
}

func TestConfig_Validate_InvalidRetries(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}}},
		},
		GlobalOptions: &GlobalOptions{
			Retries: -1, // invalid
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative retries")
	}
}

func TestConfig_Validate_InvalidReportFormat(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}}},
		},
		Reporting: &ReportingConfig{
			Formats: []string{"json", "invalid"}, // "invalid" is not valid
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid report format")
	}
}

func TestConfig_Validate_ValidReportFormats(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}}},
		},
		Reporting: &ReportingConfig{
			Formats: []string{"json", "html"},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfig_GetRetries_Default(t *testing.T) {
	cfg := &Config{}
	if retries := cfg.GetRetries(); retries != 0 {
		t.Errorf("expected default retries 0, got %d", retries)
	}
}

func TestConfig_GetRetries_Custom(t *testing.T) {
	cfg := &Config{
		GlobalOptions: &GlobalOptions{Retries: 3},
	}
	if retries := cfg.GetRetries(); retries != 3 {
		t.Errorf("expected retries 3, got %d", retries)
	}
}

func TestConfig_ShouldGenerateReport(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		format   string
		expected bool
	}{
		{
			name:     "nil reporting",
			config:   &Config{},
			format:   "json",
			expected: false,
		},
		{
			name:     "empty formats",
			config:   &Config{Reporting: &ReportingConfig{}},
			format:   "json",
			expected: false,
		},
		{
			name:     "json format present",
			config:   &Config{Reporting: &ReportingConfig{Formats: []string{"json"}}},
			format:   "json",
			expected: true,
		},
		{
			name:     "json format not present",
			config:   &Config{Reporting: &ReportingConfig{Formats: []string{"html"}}},
			format:   "json",
			expected: false,
		},
		{
			name:     "multiple formats",
			config:   &Config{Reporting: &ReportingConfig{Formats: []string{"json", "html"}}},
			format:   "html",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.ShouldGenerateReport(tt.format); got != tt.expected {
				t.Errorf("ShouldGenerateReport(%q) = %v, want %v", tt.format, got, tt.expected)
			}
		})
	}
}

func TestConfig_ShouldFailOnError(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil reporting - default true",
			config:   &Config{},
			expected: true,
		},
		{
			name:     "nil fail_on_error - default true",
			config:   &Config{Reporting: &ReportingConfig{}},
			expected: true,
		},
		{
			name:     "explicit true",
			config:   &Config{Reporting: &ReportingConfig{FailOnError: &trueVal}},
			expected: true,
		},
		{
			name:     "explicit false",
			config:   &Config{Reporting: &ReportingConfig{FailOnError: &falseVal}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.ShouldFailOnError(); got != tt.expected {
				t.Errorf("ShouldFailOnError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_BuildPageLighthouseConfig_Viewport(t *testing.T) {
	cfg := &Config{}

	// Test mobile viewport
	mobilePage := PageConfig{
		ID:        "mobile",
		Path:      "/",
		Viewport:  "mobile",
		Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}},
	}

	result := cfg.BuildPageLighthouseConfig(mobilePage)
	settings, ok := result["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("expected settings to be a map")
	}

	if settings["formFactor"] != "mobile" {
		t.Errorf("expected formFactor 'mobile', got %v", settings["formFactor"])
	}

	// Check screenEmulation for mobile
	screenEmulation, ok := settings["screenEmulation"].(map[string]interface{})
	if !ok {
		t.Fatal("expected screenEmulation to be a map for mobile viewport")
	}
	if screenEmulation["mobile"] != true {
		t.Error("expected screenEmulation.mobile to be true")
	}
}

func TestConfig_BuildPageLighthouseConfig_Desktop(t *testing.T) {
	cfg := &Config{}

	desktopPage := PageConfig{
		ID:        "desktop",
		Path:      "/",
		Viewport:  "desktop",
		Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}},
	}

	result := cfg.BuildPageLighthouseConfig(desktopPage)
	settings, ok := result["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("expected settings to be a map")
	}

	if settings["formFactor"] != "desktop" {
		t.Errorf("expected formFactor 'desktop', got %v", settings["formFactor"])
	}

	// Should not have screenEmulation for desktop
	if _, hasScreenEmulation := settings["screenEmulation"]; hasScreenEmulation {
		t.Error("expected no screenEmulation for desktop viewport")
	}
}

func TestConfig_BuildPageLighthouseConfig_GlobalFormFactorNotOverridden(t *testing.T) {
	// If global settings have formFactor, page viewport shouldn't override it
	cfg := &Config{
		GlobalOptions: &GlobalOptions{
			Lighthouse: &LighthouseSettings{
				Settings: &LighthouseRunnerSettings{
					FormFactor: "desktop",
				},
			},
		},
	}

	mobilePage := PageConfig{
		ID:        "mobile",
		Path:      "/",
		Viewport:  "mobile",
		Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}},
	}

	result := cfg.BuildPageLighthouseConfig(mobilePage)
	settings := result["settings"].(map[string]interface{})

	// Global setting should be preserved
	if settings["formFactor"] != "desktop" {
		t.Errorf("expected formFactor 'desktop' from global, got %v", settings["formFactor"])
	}
}

func TestPageConfig_WithRequirements(t *testing.T) {
	page := PageConfig{
		ID:           "home",
		Path:         "/",
		Requirements: []string{"PERF-001", "PERF-002"},
		Thresholds:   CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}},
	}

	if len(page.Requirements) != 2 {
		t.Errorf("expected 2 requirements, got %d", len(page.Requirements))
	}
	if page.Requirements[0] != "PERF-001" {
		t.Errorf("expected PERF-001, got %s", page.Requirements[0])
	}
}
