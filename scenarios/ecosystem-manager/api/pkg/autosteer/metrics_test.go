package autosteer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetricsCollector_HasData(t *testing.T) {
	collector := NewMetricsCollector("/tmp")

	t.Run("hasUXData", func(t *testing.T) {
		tests := []struct {
			name    string
			metrics *UXMetrics
			want    bool
		}{
			{
				name:    "nil metrics",
				metrics: nil,
				want:    false,
			},
			{
				name:    "all zero",
				metrics: &UXMetrics{},
				want:    false,
			},
			{
				name: "accessibility score present",
				metrics: &UXMetrics{
					AccessibilityScore: 85.0,
				},
				want: true,
			},
			{
				name: "ui test coverage present",
				metrics: &UXMetrics{
					UITestCoverage: 60.0,
				},
				want: true,
			},
			{
				name: "multiple metrics present",
				metrics: &UXMetrics{
					AccessibilityScore:    90.0,
					UITestCoverage:        75.0,
					ResponsiveBreakpoints: 3,
					UserFlowsImplemented:  5,
					LoadingStatesCount:    8,
					ErrorHandlingCoverage: 80.0,
				},
				want: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := collector.hasUXData(tt.metrics)
				if got != tt.want {
					t.Errorf("hasUXData() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("hasRefactorData", func(t *testing.T) {
		tests := []struct {
			name    string
			metrics *RefactorMetrics
			want    bool
		}{
			{
				name:    "nil metrics",
				metrics: nil,
				want:    false,
			},
			{
				name:    "all zero",
				metrics: &RefactorMetrics{},
				want:    false,
			},
			{
				name: "complexity present",
				metrics: &RefactorMetrics{
					CyclomaticComplexityAvg: 12.5,
				},
				want: true,
			},
			{
				name: "tidiness score present",
				metrics: &RefactorMetrics{
					TidinessScore: 85.0,
				},
				want: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := collector.hasRefactorData(tt.metrics)
				if got != tt.want {
					t.Errorf("hasRefactorData() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("hasTestData", func(t *testing.T) {
		tests := []struct {
			name    string
			metrics *TestMetrics
			want    bool
		}{
			{
				name:    "nil metrics",
				metrics: nil,
				want:    false,
			},
			{
				name:    "all zero",
				metrics: &TestMetrics{},
				want:    false,
			},
			{
				name: "unit coverage present",
				metrics: &TestMetrics{
					UnitTestCoverage: 75.0,
				},
				want: true,
			},
			{
				name: "integration coverage present",
				metrics: &TestMetrics{
					IntegrationTestCoverage: 65.0,
				},
				want: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := collector.hasTestData(tt.metrics)
				if got != tt.want {
					t.Errorf("hasTestData() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("hasPerformanceData", func(t *testing.T) {
		tests := []struct {
			name    string
			metrics *PerformanceMetrics
			want    bool
		}{
			{
				name:    "nil metrics",
				metrics: nil,
				want:    false,
			},
			{
				name:    "all zero",
				metrics: &PerformanceMetrics{},
				want:    false,
			},
			{
				name: "bundle size present",
				metrics: &PerformanceMetrics{
					BundleSizeKB: 250.5,
				},
				want: true,
			},
			{
				name: "load time present",
				metrics: &PerformanceMetrics{
					InitialLoadTimeMS: 1200,
				},
				want: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := collector.hasPerformanceData(tt.metrics)
				if got != tt.want {
					t.Errorf("hasPerformanceData() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("hasSecurityData", func(t *testing.T) {
		tests := []struct {
			name    string
			metrics *SecurityMetrics
			want    bool
		}{
			{
				name:    "nil metrics",
				metrics: nil,
				want:    false,
			},
			{
				name:    "all zero - vulnerability count",
				metrics: &SecurityMetrics{VulnerabilityCount: 0},
				want:    false,
			},
			{
				name: "input validation present",
				metrics: &SecurityMetrics{
					InputValidationCoverage: 85.0,
				},
				want: true,
			},
			{
				name: "security scan score present",
				metrics: &SecurityMetrics{
					SecurityScanScore: 92.0,
				},
				want: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := collector.hasSecurityData(tt.metrics)
				if got != tt.want {
					t.Errorf("hasSecurityData() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func TestMetricsCollector_ParseOperationalTargets(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "metrics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test scenario structure
	scenarioDir := filepath.Join(tmpDir, "scenarios", "test-scenario", "requirements")
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	t.Run("parse valid requirements file", func(t *testing.T) {
		// Create test requirements file
		requirementsJSON := `{
			"modules": [
				{
					"id": "module-1",
					"operationalTargets": [
						{"id": "target-1", "status": "passing"},
						{"id": "target-2", "status": "passing"},
						{"id": "target-3", "status": "failing"}
					]
				},
				{
					"id": "module-2",
					"operationalTargets": [
						{"id": "target-4", "status": "passing"},
						{"id": "target-5"}
					]
				}
			]
		}`

		requirementsPath := filepath.Join(scenarioDir, "index.json")
		if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0644); err != nil {
			t.Fatalf("Failed to write requirements file: %v", err)
		}

		collector := NewMetricsCollector(tmpDir)
		counts, err := collector.parseOperationalTargets("test-scenario")
		if err != nil {
			t.Fatalf("parseOperationalTargets() error = %v", err)
		}

		if counts.Total != 5 {
			t.Errorf("Expected 5 total targets, got %d", counts.Total)
		}
		if counts.Passing != 3 {
			t.Errorf("Expected 3 passing targets, got %d", counts.Passing)
		}
	})

	t.Run("parse file with no targets", func(t *testing.T) {
		requirementsJSON := `{
			"modules": [
				{
					"id": "module-1",
					"operationalTargets": []
				}
			]
		}`

		requirementsPath := filepath.Join(scenarioDir, "index.json")
		if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0644); err != nil {
			t.Fatalf("Failed to write requirements file: %v", err)
		}

		collector := NewMetricsCollector(tmpDir)
		counts, err := collector.parseOperationalTargets("test-scenario")
		if err != nil {
			t.Fatalf("parseOperationalTargets() error = %v", err)
		}

		if counts.Total != 0 {
			t.Errorf("Expected 0 total targets, got %d", counts.Total)
		}
		if counts.Passing != 0 {
			t.Errorf("Expected 0 passing targets, got %d", counts.Passing)
		}
	})

	t.Run("error on missing file", func(t *testing.T) {
		collector := NewMetricsCollector(tmpDir)
		_, err := collector.parseOperationalTargets("nonexistent-scenario")
		if err == nil {
			t.Error("Expected error for missing requirements file")
		}
	})

	t.Run("error on invalid JSON", func(t *testing.T) {
		requirementsPath := filepath.Join(scenarioDir, "index.json")
		if err := os.WriteFile(requirementsPath, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("Failed to write invalid file: %v", err)
		}

		collector := NewMetricsCollector(tmpDir)
		_, err := collector.parseOperationalTargets("test-scenario")
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestMetricsCollector_CollectMetrics(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "metrics-collect-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test scenario with requirements
	scenarioDir := filepath.Join(tmpDir, "scenarios", "test-scenario")
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	if err := os.MkdirAll(requirementsDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	requirementsJSON := `{
		"modules": [
			{
				"id": "module-1",
				"operationalTargets": [
					{"id": "target-1", "status": "passing"},
					{"id": "target-2", "status": "passing"}
				]
			}
		]
	}`

	requirementsPath := filepath.Join(requirementsDir, "index.json")
	if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0644); err != nil {
		t.Fatalf("Failed to write requirements file: %v", err)
	}

	t.Run("collect metrics successfully", func(t *testing.T) {
		collector := NewMetricsCollector(tmpDir)
		snapshot, err := collector.CollectMetrics("test-scenario", 5)
		if err != nil {
			t.Fatalf("CollectMetrics() error = %v", err)
		}

		if snapshot.Loops != 5 {
			t.Errorf("Expected 5 loops, got %d", snapshot.Loops)
		}
		if snapshot.OperationalTargetsTotal != 2 {
			t.Errorf("Expected 2 total targets, got %d", snapshot.OperationalTargetsTotal)
		}
		if snapshot.OperationalTargetsPassing != 2 {
			t.Errorf("Expected 2 passing targets, got %d", snapshot.OperationalTargetsPassing)
		}
		if snapshot.OperationalTargetsPercentage != 100.0 {
			t.Errorf("Expected 100%% completion, got %f", snapshot.OperationalTargetsPercentage)
		}
	})

	t.Run("error on empty scenario name", func(t *testing.T) {
		collector := NewMetricsCollector(tmpDir)
		_, err := collector.CollectMetrics("", 5)
		if err == nil {
			t.Error("Expected error for empty scenario name")
		}
	})

	t.Run("error on nonexistent scenario", func(t *testing.T) {
		collector := NewMetricsCollector(tmpDir)
		_, err := collector.CollectMetrics("nonexistent", 5)
		if err == nil {
			t.Error("Expected error for nonexistent scenario")
		}
	})
}

func TestMetricsCollector_CollectUniversalMetrics(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "metrics-universal-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test scenario with requirements
	scenarioDir := filepath.Join(tmpDir, "scenarios", "test-scenario")
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	if err := os.MkdirAll(requirementsDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	requirementsJSON := `{
		"modules": [
			{
				"id": "module-1",
				"operationalTargets": [
					{"id": "target-1", "status": "passing"},
					{"id": "target-2", "status": "failing"},
					{"id": "target-3", "status": "passing"},
					{"id": "target-4", "status": "passing"}
				]
			}
		]
	}`

	requirementsPath := filepath.Join(requirementsDir, "index.json")
	if err := os.WriteFile(requirementsPath, []byte(requirementsJSON), 0644); err != nil {
		t.Fatalf("Failed to write requirements file: %v", err)
	}

	t.Run("collect universal metrics", func(t *testing.T) {
		collector := NewMetricsCollector(tmpDir)
		snapshot := &MetricsSnapshot{}

		err := collector.collectUniversalMetrics("test-scenario", snapshot)
		if err != nil {
			t.Fatalf("collectUniversalMetrics() error = %v", err)
		}

		if snapshot.OperationalTargetsTotal != 4 {
			t.Errorf("Expected 4 total targets, got %d", snapshot.OperationalTargetsTotal)
		}
		if snapshot.OperationalTargetsPassing != 3 {
			t.Errorf("Expected 3 passing targets, got %d", snapshot.OperationalTargetsPassing)
		}
		if snapshot.OperationalTargetsPercentage != 75.0 {
			t.Errorf("Expected 75%% completion, got %f", snapshot.OperationalTargetsPercentage)
		}
		// Build status will be 0 since we can't actually build in this test
		if snapshot.BuildStatus != 0 {
			t.Logf("Build status: %d (expected 0 since no buildable scenario)", snapshot.BuildStatus)
		}
	})

	t.Run("handle zero targets", func(t *testing.T) {
		emptyJSON := `{"modules": [{"id": "module-1", "operationalTargets": []}]}`
		if err := os.WriteFile(requirementsPath, []byte(emptyJSON), 0644); err != nil {
			t.Fatalf("Failed to write requirements file: %v", err)
		}

		collector := NewMetricsCollector(tmpDir)
		snapshot := &MetricsSnapshot{}

		err := collector.collectUniversalMetrics("test-scenario", snapshot)
		if err != nil {
			t.Fatalf("collectUniversalMetrics() error = %v", err)
		}

		if snapshot.OperationalTargetsTotal != 0 {
			t.Errorf("Expected 0 total targets, got %d", snapshot.OperationalTargetsTotal)
		}
		if snapshot.OperationalTargetsPercentage != 0 {
			t.Errorf("Expected 0%% completion, got %f", snapshot.OperationalTargetsPercentage)
		}
	})
}

func TestMetricsCollector_SetTidinessManagerURL(t *testing.T) {
	collector := NewMetricsCollector("/tmp")

	testURL := "http://localhost:8080"
	collector.SetTidinessManagerURL(testURL)

	if collector.refactorCollector.tidinessManagerURL != testURL {
		t.Errorf("Expected tidiness URL %s, got %s", testURL, collector.refactorCollector.tidinessManagerURL)
	}
}
