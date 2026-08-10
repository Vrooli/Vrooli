package deployment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
	types "scenario-dependency-analyzer/internal/types"
)

// TestBuildReport tests the main report building function.
func TestBuildReport(t *testing.T) {
	t.Run("NilConfig", func(t *testing.T) {
		report := BuildReport("test-scenario", "/tmp/test", "/tmp/scenarios", nil)
		if report != nil {
			t.Error("expected nil report for nil config")
		}
	})

	t.Run("BasicConfig", func(t *testing.T) {
		scenarioDir := t.TempDir()
		scenariosDir := filepath.Dir(scenarioDir)
		scenarioName := filepath.Base(scenarioDir)

		cfg := &types.ServiceConfig{}
		cfg.Service.Name = scenarioName
		cfg.Service.Version = "1.0.0"

		report := BuildReport(scenarioName, scenarioDir, scenariosDir, cfg)
		if report == nil {
			t.Fatal("expected non-nil report")
		}
		if report.Scenario != scenarioName {
			t.Errorf("expected scenario %q, got %q", scenarioName, report.Scenario)
		}
		if report.ReportVersion != ReportVersion {
			t.Errorf("expected report version %d, got %d", ReportVersion, report.ReportVersion)
		}
		if report.GeneratedAt.IsZero() {
			t.Error("expected GeneratedAt to be set")
		}
	})

	t.Run("WithResources", func(t *testing.T) {
		scenarioDir := t.TempDir()
		scenariosDir := filepath.Dir(scenarioDir)
		scenarioName := filepath.Base(scenarioDir)

		cfg := &types.ServiceConfig{}
		cfg.Service.Name = scenarioName
		cfg.Dependencies.Resources = map[string]types.Resource{
			"postgres": {Type: "database", Required: true},
			"redis":    {Type: "cache", Required: false},
		}

		report := BuildReport(scenarioName, scenarioDir, scenariosDir, cfg)
		if report == nil {
			t.Fatal("expected non-nil report")
		}
		if len(report.Dependencies) < 2 {
			t.Errorf("expected at least 2 dependencies, got %d", len(report.Dependencies))
		}
	})

	t.Run("WithDeploymentTiers", func(t *testing.T) {
		scenarioDir := t.TempDir()
		scenariosDir := filepath.Dir(scenarioDir)
		scenarioName := filepath.Base(scenarioDir)

		cfg := &types.ServiceConfig{}
		cfg.Service.Name = scenarioName
		cfg.Deployment = &types.ServiceDeployment{
			Tiers: map[string]types.DeploymentTier{
				"desktop": {},
				"server":  {},
			},
		}

		report := BuildReport(scenarioName, scenarioDir, scenariosDir, cfg)
		if report == nil {
			t.Fatal("expected non-nil report")
		}
	})

	t.Run("IncludesRootScenarioRequirementsInAggregates", func(t *testing.T) {
		scenarioDir := t.TempDir()
		scenariosDir := filepath.Dir(scenarioDir)
		scenarioName := filepath.Base(scenarioDir)

		ram := 1536.0
		disk := 2048.0
		cpu := 1.5

		cfg := &types.ServiceConfig{}
		cfg.Service.Name = scenarioName
		cfg.Deployment = &types.ServiceDeployment{
			Tiers: map[string]types.DeploymentTier{
				"server": {
					Requirements: &types.DeploymentRequirements{RAMMB: &ram, DiskMB: &disk, CPUCores: &cpu},
				},
			},
		}

		report := BuildReport(scenarioName, scenarioDir, scenariosDir, cfg)
		if report == nil {
			t.Fatal("expected non-nil report")
		}

		server, ok := report.Aggregates["server"]
		if !ok {
			t.Fatalf("expected server aggregate, got: %+v", report.Aggregates)
		}
		if server.EstimatedRequirements.RAMMB != ram {
			t.Fatalf("expected server RAM %.0f, got %.0f", ram, server.EstimatedRequirements.RAMMB)
		}
	})

	t.Run("IncludesRootScenarioInMetadataGapAnalysis", func(t *testing.T) {
		scenarioDir := t.TempDir()
		scenariosDir := filepath.Dir(scenarioDir)
		scenarioName := filepath.Base(scenarioDir)

		if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
			t.Fatalf("mkdir .vrooli: %v", err)
		}

		cfg := &types.ServiceConfig{}
		cfg.Service.Name = scenarioName
		cfg.Dependencies.Resources = map[string]types.Resource{
			"postgres": {Type: "postgres", Required: true},
		}
		cfg.Deployment = &types.ServiceDeployment{
			Tiers: map[string]types.DeploymentTier{
				"tier-1-local": {},
			},
		}

		raw, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("marshal config: %v", err)
		}
		if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), raw, 0o644); err != nil {
			t.Fatalf("write service.json: %v", err)
		}

		report := BuildReport(scenarioName, scenarioDir, scenariosDir, cfg)
		if report == nil || report.MetadataGaps == nil {
			t.Fatalf("expected metadata gaps in report, got: %+v", report)
		}

		rootGap, ok := report.MetadataGaps.GapsByScenario[scenarioName]
		if !ok {
			t.Fatalf("expected root scenario gap entry, got %+v", report.MetadataGaps.GapsByScenario)
		}
		if !rootGap.MissingDependencyCatalog {
			t.Fatalf("expected missing dependency catalog for root scenario, got %+v", rootGap)
		}
		if len(rootGap.MissingResourceMetadata) == 0 {
			t.Fatalf("expected missing resource metadata for root scenario, got %+v", rootGap)
		}
	})
}

// TestPersistReport tests report persistence.
func TestPersistReport(t *testing.T) {
	t.Run("NilReport", func(t *testing.T) {
		err := PersistReport("/tmp/test", nil)
		if err != nil {
			t.Errorf("expected nil error for nil report, got %v", err)
		}
	})

	t.Run("ValidReport", func(t *testing.T) {
		scenarioDir := t.TempDir()
		report := &types.DeploymentAnalysisReport{
			Scenario:      "test-scenario",
			ReportVersion: ReportVersion,
			GeneratedAt:   time.Now(),
		}

		err := PersistReport(scenarioDir, report)
		if err != nil {
			t.Fatalf("PersistReport error: %v", err)
		}

		reportPath := mustReportPath(t, scenarioDir)
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			t.Error("expected report file to exist")
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		scenarioDir := t.TempDir()
		original := &types.DeploymentAnalysisReport{
			Scenario:      "test-scenario",
			ReportVersion: ReportVersion,
			GeneratedAt:   time.Now().UTC().Truncate(time.Second),
			Dependencies: []types.DeploymentDependencyNode{
				{Name: "postgres", Type: "resource"},
			},
		}

		if err := PersistReport(scenarioDir, original); err != nil {
			t.Fatalf("PersistReport error: %v", err)
		}

		loaded, err := LoadReport(scenarioDir)
		if err != nil {
			t.Fatalf("LoadReport error: %v", err)
		}

		if loaded.Scenario != original.Scenario {
			t.Errorf("expected scenario %q, got %q", original.Scenario, loaded.Scenario)
		}
		if loaded.ReportVersion != original.ReportVersion {
			t.Errorf("expected version %d, got %d", original.ReportVersion, loaded.ReportVersion)
		}
		if len(loaded.Dependencies) != len(original.Dependencies) {
			t.Errorf("expected %d dependencies, got %d", len(original.Dependencies), len(loaded.Dependencies))
		}
	})

	t.Run("NoRewriteWhenOnlyGeneratedAtChanges", func(t *testing.T) {
		scenarioDir := t.TempDir()
		reportPath := mustReportPath(t, scenarioDir)

		first := &types.DeploymentAnalysisReport{
			Scenario:      "test-scenario",
			ReportVersion: ReportVersion,
			GeneratedAt:   time.Date(2026, 2, 8, 3, 0, 0, 0, time.UTC),
			BundleManifest: types.BundleManifest{
				Scenario:    "test-scenario",
				GeneratedAt: time.Date(2026, 2, 8, 3, 0, 0, 0, time.UTC),
			},
			MetadataGaps: &types.DeploymentMetadataGaps{
				GapsByScenario: map[string]types.ScenarioGapInfo{
					"test-scenario": {
						MissingTierDefinitions: []string{"desktop", "server"},
					},
				},
			},
		}
		if err := PersistReport(scenarioDir, first); err != nil {
			t.Fatalf("PersistReport(first) error: %v", err)
		}

		infoBefore, err := os.Stat(reportPath)
		if err != nil {
			t.Fatalf("stat before: %v", err)
		}
		contentBefore, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read before: %v", err)
		}

		second := &types.DeploymentAnalysisReport{
			Scenario:      "test-scenario",
			ReportVersion: ReportVersion,
			GeneratedAt:   time.Date(2026, 2, 8, 3, 30, 0, 0, time.UTC),
			BundleManifest: types.BundleManifest{
				Scenario:    "test-scenario",
				GeneratedAt: time.Date(2026, 2, 8, 3, 30, 0, 0, time.UTC),
			},
			MetadataGaps: &types.DeploymentMetadataGaps{
				GapsByScenario: map[string]types.ScenarioGapInfo{
					"test-scenario": {
						MissingTierDefinitions: []string{"desktop", "server"},
					},
				},
			},
		}
		if err := PersistReport(scenarioDir, second); err != nil {
			t.Fatalf("PersistReport(second) error: %v", err)
		}

		infoAfter, err := os.Stat(reportPath)
		if err != nil {
			t.Fatalf("stat after: %v", err)
		}
		contentAfter, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read after: %v", err)
		}

		if !infoAfter.ModTime().Equal(infoBefore.ModTime()) {
			t.Fatalf("expected report file mtime unchanged when only timestamps differ")
		}
		if string(contentAfter) != string(contentBefore) {
			t.Fatalf("expected report contents unchanged when only timestamps differ")
		}
	})
}

// TestLoadReport tests report loading.
func TestLoadReport(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := LoadReport("/nonexistent/path")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		scenarioDir := t.TempDir()
		reportDir := filepath.Dir(mustReportPath(t, scenarioDir))
		os.MkdirAll(reportDir, 0o755)
		os.WriteFile(filepath.Join(reportDir, "deployment-report.json"), []byte("invalid json"), 0o644)

		_, err := LoadReport(scenarioDir)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("MarksReportStaleWhenManifestChanges", func(t *testing.T) {
		scenarioDir := t.TempDir()
		manifestDir := filepath.Join(scenarioDir, ".vrooli")
		if err := os.MkdirAll(manifestDir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifestPath := filepath.Join(manifestDir, "service.json")
		if err := os.WriteFile(manifestPath, []byte(`{"service":{"name":"fixture"}}`), 0o644); err != nil {
			t.Fatal(err)
		}

		report := &types.DeploymentAnalysisReport{
			Scenario:      "fixture",
			ReportVersion: ReportVersion,
			GeneratedAt:   time.Now().UTC(),
		}
		if err := PersistReport(scenarioDir, report); err != nil {
			t.Fatalf("PersistReport: %v", err)
		}

		fresh, err := LoadReport(scenarioDir)
		if err != nil {
			t.Fatalf("LoadReport fresh: %v", err)
		}
		if fresh.Stale {
			t.Fatal("fresh report was marked stale")
		}

		if err := os.WriteFile(manifestPath, []byte(`{"service":{"name":"changed"}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		stale, err := LoadReport(scenarioDir)
		if err != nil {
			t.Fatalf("LoadReport stale: %v", err)
		}
		if !stale.Stale {
			t.Fatal("changed manifest did not mark persisted report stale")
		}
		if stale.Provenance.InputDigest == "" {
			t.Fatal("stale report lost its original input digest")
		}
	})
}

func mustReportPath(t *testing.T, scenarioDir string) string {
	t.Helper()
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		t.Fatalf("create storage resolver: %v", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: filepath.Base(filepath.Clean(scenarioDir))},
		storage.ClassData,
		filepath.Join("deployment", "deployment-report.json"),
	)
	if err != nil {
		t.Fatalf("resolve report path: %v", err)
	}
	return path
}

// TestIsTierBlocker tests the tier blocker decision logic.
func TestIsTierBlocker(t *testing.T) {
	truePtr := func(b bool) *bool { return &b }

	tests := []struct {
		name    string
		support types.TierSupportSummary
		want    bool
	}{
		{
			name:    "ExplicitlyUnsupported",
			support: types.TierSupportSummary{Supported: truePtr(false)},
			want:    true,
		},
		{
			name:    "ExplicitlySupported",
			support: types.TierSupportSummary{Supported: truePtr(true)},
			want:    false,
		},
		{
			name:    "NoData",
			support: types.TierSupportSummary{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTierBlocker(tt.support)
			if got != tt.want {
				t.Errorf("IsTierBlocker() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestComputeTierAggregates tests tier aggregate computation.
func TestComputeTierAggregates(t *testing.T) {
	truePtr := func(b bool) *bool { return &b }
	floatPtr := func(f float64) *float64 { return &f }

	t.Run("EmptyNodes", func(t *testing.T) {
		aggregates := ComputeTierAggregates(nil)
		if len(aggregates) != 0 {
			t.Errorf("expected empty aggregates, got %d", len(aggregates))
		}
	})

	t.Run("SingleNode", func(t *testing.T) {
		nodes := []types.DeploymentDependencyNode{
			{
				Name: "postgres",
				Type: "resource",
				TierSupport: map[string]types.TierSupportSummary{
					"desktop": {
						Supported:    truePtr(true),
						FitnessScore: floatPtr(0.9),
					},
				},
			},
		}

		aggregates := ComputeTierAggregates(nodes)
		desktop, ok := aggregates["desktop"]
		if !ok {
			t.Fatal("expected desktop aggregate")
		}
		if desktop.DependencyCount != 1 {
			t.Errorf("expected 1 dependency, got %d", desktop.DependencyCount)
		}
		if desktop.FitnessScore != 0.9 {
			t.Errorf("expected fitness 0.9, got %f", desktop.FitnessScore)
		}
	})

	t.Run("WithBlocker", func(t *testing.T) {
		nodes := []types.DeploymentDependencyNode{
			{
				Name: "postgres",
				Type: "resource",
				TierSupport: map[string]types.TierSupportSummary{
					"mobile": {
						Supported:    truePtr(false),
						FitnessScore: floatPtr(0.2),
					},
				},
			},
		}

		aggregates := ComputeTierAggregates(nodes)
		mobile, ok := aggregates["mobile"]
		if !ok {
			t.Fatal("expected mobile aggregate")
		}
		if len(mobile.BlockingDependencies) != 1 {
			t.Errorf("expected 1 blocker, got %d", len(mobile.BlockingDependencies))
		}
		if mobile.BlockingDependencies[0] != "postgres" {
			t.Errorf("expected postgres as blocker, got %s", mobile.BlockingDependencies[0])
		}
	})

	t.Run("NestedNodes", func(t *testing.T) {
		nodes := []types.DeploymentDependencyNode{
			{
				Name: "auth-service",
				Type: "scenario",
				TierSupport: map[string]types.TierSupportSummary{
					"desktop": {Supported: truePtr(true), FitnessScore: floatPtr(0.8)},
				},
				Children: []types.DeploymentDependencyNode{
					{
						Name: "postgres",
						Type: "resource",
						TierSupport: map[string]types.TierSupportSummary{
							"desktop": {Supported: truePtr(true), FitnessScore: floatPtr(0.6)},
						},
					},
				},
			},
		}

		aggregates := ComputeTierAggregates(nodes)
		desktop, ok := aggregates["desktop"]
		if !ok {
			t.Fatal("expected desktop aggregate")
		}
		if desktop.DependencyCount != 2 {
			t.Errorf("expected 2 dependencies (parent + child), got %d", desktop.DependencyCount)
		}
		// Average of 0.8 and 0.6
		expectedFitness := 0.7
		if desktop.FitnessScore != expectedFitness {
			t.Errorf("expected fitness %f, got %f", expectedFitness, desktop.FitnessScore)
		}
	})

	t.Run("SkipsExplicitNonDeployIntentNodes", func(t *testing.T) {
		falseVal := false
		nodes := []types.DeploymentDependencyNode{
			{
				Name:     "disabled-cache",
				Type:     "resource",
				Required: &falseVal,
				Enabled:  &falseVal,
				TierSupport: map[string]types.TierSupportSummary{
					"server": {Supported: truePtr(true), FitnessScore: floatPtr(0.9)},
				},
			},
		}

		aggregates := ComputeTierAggregates(nodes)
		if len(aggregates) != 0 {
			t.Fatalf("expected no aggregates when all nodes are non-deploy-intent, got: %+v", aggregates)
		}
	})
}

// TestMapKeys tests the MapKeys helper function.
func TestMapKeys(t *testing.T) {
	t.Run("EmptyMap", func(t *testing.T) {
		keys := MapKeys(nil)
		if keys != nil {
			t.Errorf("expected nil for empty map, got %v", keys)
		}
	})

	t.Run("WithValues", func(t *testing.T) {
		set := map[string]struct{}{
			"charlie": {},
			"alpha":   {},
			"bravo":   {},
		}
		keys := MapKeys(set)
		if len(keys) != 3 {
			t.Fatalf("expected 3 keys, got %d", len(keys))
		}
		// Should be sorted
		expected := []string{"alpha", "bravo", "charlie"}
		for i, k := range keys {
			if k != expected[i] {
				t.Errorf("at position %d: expected %q, got %q", i, expected[i], k)
			}
		}
	})
}

// TestBuildDependencyNodeIndex tests the node index builder.
func TestBuildDependencyNodeIndex(t *testing.T) {
	nodes := []types.DeploymentDependencyNode{
		{
			Name: "auth-service",
			Type: "scenario",
			Children: []types.DeploymentDependencyNode{
				{Name: "postgres", Type: "resource"},
				{Name: "redis", Type: "resource"},
			},
		},
		{Name: "minio", Type: "resource"},
	}

	index := BuildDependencyNodeIndex(nodes)
	if len(index) != 4 {
		t.Errorf("expected 4 indexed nodes, got %d", len(index))
	}

	// Keys should be lowercase
	if _, ok := index["auth-service"]; !ok {
		t.Error("expected auth-service in index")
	}
	if _, ok := index["postgres"]; !ok {
		t.Error("expected postgres in index")
	}
}

// TestBundleManifestIntegration tests bundle manifest generation integration.
func TestBundleManifestIntegration(t *testing.T) {
	scenarioDir := t.TempDir()

	// Create minimal directory structure
	os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755)
	os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755)

	cfg := &types.ServiceConfig{}
	cfg.Service.Name = "integration-test"
	cfg.Service.Version = "2.0.0"
	cfg.Service.Description = "Integration test scenario"
	cfg.Dependencies.Resources = map[string]types.Resource{
		"postgres": {Type: "database", Required: true},
	}

	nodes := []types.DeploymentDependencyNode{
		{
			Name:         "postgres",
			Type:         "resource",
			Alternatives: []string{"sqlite"},
		},
	}

	manifest := BuildBundleManifest("integration-test", scenarioDir, time.Now(), nodes, cfg)
	if manifest.Skeleton == nil {
		t.Fatal("expected skeleton to be populated")
	}

	// Verify app info
	if manifest.Skeleton.App.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", manifest.Skeleton.App.Version)
	}

	// Verify swaps are generated
	if len(manifest.Skeleton.Swaps) == 0 {
		t.Error("expected swaps to be generated for postgres->sqlite")
	}

	// Validate the skeleton against schema
	payload, err := json.Marshal(manifest.Skeleton)
	if err != nil {
		t.Fatalf("failed to marshal skeleton: %v", err)
	}
	// The skeleton should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}
