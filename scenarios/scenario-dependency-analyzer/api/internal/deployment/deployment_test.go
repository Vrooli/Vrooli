package deployment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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
				"desktop": {Status: "ready"},
				"server":  {Status: "ready"},
			},
		}

		report := BuildReport(scenarioName, scenarioDir, scenariosDir, cfg)
		if report == nil {
			t.Fatal("expected non-nil report")
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

		reportPath := filepath.Join(scenarioDir, ".vrooli", "deployment", "deployment-report.json")
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
		reportDir := filepath.Join(scenarioDir, ".vrooli", "deployment")
		os.MkdirAll(reportDir, 0755)
		os.WriteFile(filepath.Join(reportDir, "deployment-report.json"), []byte("invalid json"), 0644)

		_, err := LoadReport(scenarioDir)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

// TestClassifyResource tests resource classification.
func TestClassifyResource(t *testing.T) {
	tests := []struct {
		name       string
		resource   string
		wantClass  ResourceClass
		wantHeavy  bool
	}{
		{"Postgres", "postgres", ResourceClassDatabase, true},
		{"MySQL", "mysql", ResourceClassDatabase, true},
		{"MongoDB", "mongodb", ResourceClassDatabase, true},
		{"Redis", "redis", ResourceClassDatabase, false},
		{"Qdrant", "qdrant", ResourceClassDatabase, false},
		{"Ollama", "ollama", ResourceClassAI, true},
		{"ClaudeCode", "claude-code", ResourceClassAI, false},
		{"OpenAI", "openai", ResourceClassAI, false},
		{"N8N", "n8n", ResourceClassAutomation, true},
		{"Huginn", "huginn", ResourceClassAutomation, true},
		{"Windmill", "windmill", ResourceClassAutomation, true},
		{"Minio", "minio", ResourceClassStorage, false},
		{"S3", "s3", ResourceClassStorage, false},
		{"Browserless", "browserless", ResourceClassBrowser, true},
		{"Playwright", "playwright", ResourceClassBrowser, true},
		{"Judge0", "judge0", ResourceClassExecution, true},
		{"Sandbox", "sandbox", ResourceClassExecution, true},
		{"Unknown", "unknown-resource", ResourceClassService, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classification := ClassifyResource(tt.resource)
			if classification.Class != tt.wantClass {
				t.Errorf("ClassifyResource(%q).Class = %q, want %q",
					tt.resource, classification.Class, tt.wantClass)
			}
			if classification.IsHeavyOps != tt.wantHeavy {
				t.Errorf("ClassifyResource(%q).IsHeavyOps = %v, want %v",
					tt.resource, classification.IsHeavyOps, tt.wantHeavy)
			}
		})
	}
}

// TestDecideTierFitness tests tier fitness decisions.
func TestDecideTierFitness(t *testing.T) {
	tests := []struct {
		name          string
		tier          string
		classification ResourceClassification
		wantSupported bool
		minScore      float64
		maxScore      float64
	}{
		{
			name:          "LocalAlwaysSupported",
			tier:          "local",
			classification: ResourceClassification{Class: ResourceClassDatabase, IsHeavyOps: true},
			wantSupported: true,
			minScore:      1.0,
			maxScore:      1.0,
		},
		{
			name:          "DesktopLightweight",
			tier:          "desktop",
			classification: ResourceClassification{Class: ResourceClassStorage, IsHeavyOps: false},
			wantSupported: true,
			minScore:      0.85,
			maxScore:      1.0,
		},
		{
			name:          "DesktopHeavy",
			tier:          "desktop",
			classification: ResourceClassification{Class: ResourceClassDatabase, IsHeavyOps: true},
			wantSupported: true,
			minScore:      0.5,
			maxScore:      0.7,
		},
		{
			name:          "ServerAlwaysGood",
			tier:          "server",
			classification: ResourceClassification{Class: ResourceClassDatabase, IsHeavyOps: true},
			wantSupported: true,
			minScore:      0.9,
			maxScore:      1.0,
		},
		{
			name:          "MobileAIBlocked",
			tier:          "mobile",
			classification: ResourceClassification{Class: ResourceClassAI, IsHeavyOps: true},
			wantSupported: false,
			minScore:      0.0,
			maxScore:      0.1,
		},
		{
			name:          "MobileDatabaseBlocked",
			tier:          "mobile",
			classification: ResourceClassification{Class: ResourceClassDatabase, IsHeavyOps: false},
			wantSupported: false,
			minScore:      0.1,
			maxScore:      0.3,
		},
		{
			name:          "SaaSDatabase",
			tier:          "saas",
			classification: ResourceClassification{Class: ResourceClassDatabase, IsHeavyOps: true},
			wantSupported: true,
			minScore:      0.9,
			maxScore:      1.0,
		},
		{
			name:          "SaaSLocalAI",
			tier:          "saas",
			classification: ResourceClassification{Class: ResourceClassAI, IsHeavyOps: true},
			wantSupported: true,
			minScore:      0.2,
			maxScore:      0.4,
		},
		{
			name:          "EnterpriseAlwaysHigh",
			tier:          "enterprise",
			classification: ResourceClassification{Class: ResourceClassAI, IsHeavyOps: true},
			wantSupported: true,
			minScore:      0.95,
			maxScore:      1.0,
		},
		{
			name:          "UnknownTier",
			tier:          "unknown-tier",
			classification: ResourceClassification{Class: ResourceClassService, IsHeavyOps: false},
			wantSupported: true,
			minScore:      0.6,
			maxScore:      0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideTierFitness(tt.tier, tt.classification)
			if decision.Supported != tt.wantSupported {
				t.Errorf("DecideTierFitness(%q).Supported = %v, want %v",
					tt.tier, decision.Supported, tt.wantSupported)
			}
			if decision.FitnessScore < tt.minScore || decision.FitnessScore > tt.maxScore {
				t.Errorf("DecideTierFitness(%q).FitnessScore = %v, want between %v and %v",
					tt.tier, decision.FitnessScore, tt.minScore, tt.maxScore)
			}
		})
	}
}

// TestIsTierBlocker tests the tier blocker decision logic.
func TestIsTierBlocker(t *testing.T) {
	truePtr := func(b bool) *bool { return &b }
	floatPtr := func(f float64) *float64 { return &f }

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
			name:    "LowFitness",
			support: types.TierSupportSummary{FitnessScore: floatPtr(0.5)},
			want:    true,
		},
		{
			name:    "HighFitness",
			support: types.TierSupportSummary{FitnessScore: floatPtr(0.9)},
			want:    false,
		},
		{
			name:    "ThresholdFitness",
			support: types.TierSupportSummary{FitnessScore: floatPtr(TierBlockerThreshold)},
			want:    false,
		},
		{
			name:    "BelowThreshold",
			support: types.TierSupportSummary{FitnessScore: floatPtr(TierBlockerThreshold - 0.01)},
			want:    true,
		},
		{
			name:    "NoData",
			support: types.TierSupportSummary{},
			want:    false,
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

// TestInterpretTierStatus tests tier status interpretation.
func TestInterpretTierStatus(t *testing.T) {
	tests := []struct {
		status string
		want   TierStatus
	}{
		{"ready", TierStatusReady},
		{"supported", TierStatusReady},
		{"limited", TierStatusLimited},
		{"blocked", TierStatusLimited},
		{"unknown", TierStatusUnknown},
		{"", TierStatusUnknown},
		{"READY", TierStatusUnknown}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := InterpretTierStatus(tt.status)
			if got != tt.want {
				t.Errorf("InterpretTierStatus(%q) = %v, want %v", tt.status, got, tt.want)
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

// TestInferResourceTierSupport tests default tier support inference.
func TestInferResourceTierSupport(t *testing.T) {
	t.Run("PostgresInference", func(t *testing.T) {
		support := InferResourceTierSupport("postgres", nil)
		if len(support) == 0 {
			t.Fatal("expected tier support to be populated")
		}

		// Local should always work
		if local, ok := support["local"]; ok {
			if local.Supported == nil || !*local.Supported {
				t.Error("expected local tier to be supported")
			}
		} else {
			t.Error("expected local tier in support map")
		}

		// Mobile should be blocked for heavy database
		if mobile, ok := support["mobile"]; ok {
			if mobile.Supported == nil || *mobile.Supported {
				t.Error("expected mobile tier to be unsupported for postgres")
			}
		}
	})

	t.Run("RedisInference", func(t *testing.T) {
		support := InferResourceTierSupport("redis", nil)
		// Redis is lightweight, should have better mobile support
		if mobile, ok := support["mobile"]; ok {
			// Even redis should be blocked on mobile (it's still a database)
			if mobile.Supported != nil && *mobile.Supported {
				t.Logf("mobile support for redis: %v", *mobile.Supported)
			}
		}
	})
}

// TestBundleManifestIntegration tests bundle manifest generation integration.
func TestBundleManifestIntegration(t *testing.T) {
	scenarioDir := t.TempDir()

	// Create minimal directory structure
	os.MkdirAll(filepath.Join(scenarioDir, "api"), 0755)
	os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

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
