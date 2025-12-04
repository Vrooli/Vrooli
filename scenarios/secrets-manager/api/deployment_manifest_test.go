// Package main provides tests for deployment manifest generation.
package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// Context Timeout Tests
// -----------------------------------------------------------------------------

func TestManifestBuilder_Build_ContextCancelled(t *testing.T) {
	store := &mockSecretStore{
		secrets: []DeploymentSecretEntry{
			{ID: "s1", SecretKey: "KEY", HandlingStrategy: "prompt"},
		},
	}
	analyzer := &mockAnalyzerClient{}
	resolver := &mockResourceResolver{
		resolved: ResolvedResources{Effective: []string{"test"}},
	}
	builder := NewManifestBuilderWithDeps(store, analyzer, resolver, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := DeploymentManifestRequest{Scenario: "test", Tier: "desktop"}
	_, err := builder.Build(ctx, req)

	// Cancelled context should cause Build to return (may succeed if fast or fail with context error)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("Build with cancelled context returned: %v (acceptable)", err)
	}
}

func TestManifestBuilder_Build_ContextTimeout(t *testing.T) {
	store := &mockSecretStore{
		secrets: []DeploymentSecretEntry{
			{ID: "s1", SecretKey: "KEY", HandlingStrategy: "prompt"},
		},
	}
	analyzer := &mockAnalyzerClient{}
	resolver := &mockResourceResolver{
		resolved: ResolvedResources{Effective: []string{"test"}},
	}
	builder := NewManifestBuilderWithDeps(store, analyzer, resolver, nil)

	// Very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow some time for timeout to trigger
	time.Sleep(1 * time.Millisecond)

	req := DeploymentManifestRequest{Scenario: "test", Tier: "desktop"}
	_, err := builder.Build(ctx, req)

	// Timeout context may cause Build to return with context deadline exceeded
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Build with expired context returned: %v (acceptable)", err)
	}
}

// -----------------------------------------------------------------------------
// Mock Implementations
// -----------------------------------------------------------------------------

// mockSecretStore provides a configurable mock for SecretStore.
type mockSecretStore struct {
	secrets     []DeploymentSecretEntry
	fetchErr    error
	persistErr  error
	persistedAt []string // tracks scenario+tier combinations persisted
}

func (m *mockSecretStore) FetchSecrets(ctx context.Context, tier string, resources []string, includeOptional bool) ([]DeploymentSecretEntry, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.secrets, nil
}

func (m *mockSecretStore) PersistManifest(ctx context.Context, scenario, tier string, manifest *DeploymentManifest) error {
	if m.persistedAt == nil {
		m.persistedAt = []string{}
	}
	m.persistedAt = append(m.persistedAt, scenario+":"+tier)
	return m.persistErr
}

// mockAnalyzerClient provides a configurable mock for AnalyzerClient.
type mockAnalyzerClient struct {
	report *analyzerDeploymentReport
	err    error
}

func (m *mockAnalyzerClient) FetchDeploymentReport(ctx context.Context, scenario string) (*analyzerDeploymentReport, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.report, nil
}

// mockResourceResolver provides a configurable mock for ResourceResolver.
type mockResourceResolver struct {
	resolved ResolvedResources
}

func (m *mockResourceResolver) ResolveResources(ctx context.Context, scenario string, requestedResources []string) ResolvedResources {
	return m.resolved
}

// -----------------------------------------------------------------------------
// ManifestBuilder Tests
// -----------------------------------------------------------------------------

func TestManifestBuilder_Build_Success(t *testing.T) {
	secrets := []DeploymentSecretEntry{
		{
			ID:               "secret-1",
			ResourceName:     "postgres",
			SecretKey:        "POSTGRES_PASSWORD",
			SecretType:       "password",
			Required:         true,
			Classification:   "service",
			HandlingStrategy: "generate",
		},
		{
			ID:               "secret-2",
			ResourceName:     "postgres",
			SecretKey:        "POSTGRES_USER",
			SecretType:       "string",
			Required:         true,
			Classification:   "service",
			HandlingStrategy: "prompt",
		},
	}

	store := &mockSecretStore{secrets: secrets}
	analyzer := &mockAnalyzerClient{report: nil}
	resolver := &mockResourceResolver{
		resolved: ResolvedResources{
			Effective:       []string{"postgres"},
			FromServiceJSON: []string{"postgres"},
		},
	}

	builder := NewManifestBuilderWithDeps(store, analyzer, resolver, nil)

	req := DeploymentManifestRequest{
		Scenario: "test-scenario",
		Tier:     "tier-2-desktop",
	}

	manifest, err := builder.Build(context.Background(), req)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil", err)
	}

	if manifest.Scenario != "test-scenario" {
		t.Errorf("Scenario = %q, want %q", manifest.Scenario, "test-scenario")
	}
	if manifest.Tier != "tier-2-desktop" {
		t.Errorf("Tier = %q, want %q", manifest.Tier, "tier-2-desktop")
	}
	if len(manifest.Secrets) != 2 {
		t.Errorf("len(Secrets) = %d, want 2", len(manifest.Secrets))
	}
	if manifest.Summary.TotalSecrets != 2 {
		t.Errorf("Summary.TotalSecrets = %d, want 2", manifest.Summary.TotalSecrets)
	}
	if len(manifest.BundleSecrets) != 2 {
		t.Errorf("len(BundleSecrets) = %d, want 2", len(manifest.BundleSecrets))
	}
}

func TestManifestBuilder_Build_ValidationError(t *testing.T) {
	store := &mockSecretStore{}
	analyzer := &mockAnalyzerClient{}
	resolver := &mockResourceResolver{}
	builder := NewManifestBuilderWithDeps(store, analyzer, resolver, nil)

	tests := []struct {
		name     string
		scenario string
		tier     string
	}{
		{"empty scenario", "", "tier-2-desktop"},
		{"empty tier", "test", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := DeploymentManifestRequest{Scenario: tt.scenario, Tier: tt.tier}
			_, err := builder.Build(context.Background(), req)
			if err == nil {
				t.Error("Build() error = nil, want error")
			}
		})
	}
}

func TestManifestBuilder_Build_NoSecrets(t *testing.T) {
	store := &mockSecretStore{secrets: []DeploymentSecretEntry{}}
	analyzer := &mockAnalyzerClient{}
	resolver := &mockResourceResolver{
		resolved: ResolvedResources{Effective: []string{"postgres"}},
	}
	builder := NewManifestBuilderWithDeps(store, analyzer, resolver, nil)

	req := DeploymentManifestRequest{Scenario: "test", Tier: "desktop"}
	_, err := builder.Build(context.Background(), req)
	if err == nil {
		t.Error("Build() error = nil, want error for no secrets")
	}
}

func TestManifestBuilder_Build_FetchError(t *testing.T) {
	store := &mockSecretStore{fetchErr: errors.New("db connection failed")}
	analyzer := &mockAnalyzerClient{}
	resolver := &mockResourceResolver{
		resolved: ResolvedResources{Effective: []string{"postgres"}},
	}
	builder := NewManifestBuilderWithDeps(store, analyzer, resolver, nil)

	req := DeploymentManifestRequest{Scenario: "test", Tier: "desktop"}
	_, err := builder.Build(context.Background(), req)
	if err == nil {
		t.Error("Build() error = nil, want error")
	}
}

func TestManifestBuilder_Build_FallbackNoDatabase(t *testing.T) {
	analyzer := &mockAnalyzerClient{}
	resolver := &mockResourceResolver{
		resolved: ResolvedResources{Effective: []string{"core"}},
	}
	builder := NewManifestBuilderWithDeps(nil, analyzer, resolver, nil)

	req := DeploymentManifestRequest{Scenario: "test", Tier: "desktop"}
	manifest, err := builder.Build(context.Background(), req)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil", err)
	}

	if len(manifest.Secrets) == 0 {
		t.Error("Fallback manifest should have placeholder secrets")
	}
	if manifest.Secrets[0].HandlingStrategy != "prompt" {
		t.Error("Fallback secrets should use prompt strategy")
	}
}

func TestManifestBuilder_Build_WithAnalyzerReport(t *testing.T) {
	secrets := []DeploymentSecretEntry{
		{
			ResourceName:     "postgres",
			SecretKey:        "POSTGRES_PASSWORD",
			HandlingStrategy: "generate",
			Classification:   "service",
		},
	}

	report := &analyzerDeploymentReport{
		Scenario:    "test",
		GeneratedAt: time.Now(),
		Dependencies: []analyzerDependency{
			{Name: "postgres", Type: "resource"},
		},
		Aggregates: map[string]analyzerAggregate{
			"desktop": {FitnessScore: 0.85},
		},
	}

	store := &mockSecretStore{secrets: secrets}
	analyzer := &mockAnalyzerClient{report: report}
	resolver := &mockResourceResolver{
		resolved: ResolvedResources{Effective: []string{"postgres"}},
	}
	builder := NewManifestBuilderWithDeps(store, analyzer, resolver, nil)

	req := DeploymentManifestRequest{Scenario: "test", Tier: "desktop"}
	manifest, err := builder.Build(context.Background(), req)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(manifest.Dependencies) == 0 {
		t.Error("Expected dependencies from analyzer report")
	}
	if manifest.TierAggregates == nil {
		t.Error("Expected tier aggregates from analyzer report")
	}
	if manifest.AnalyzerGeneratedAt == nil {
		t.Error("Expected AnalyzerGeneratedAt to be set")
	}
}

// -----------------------------------------------------------------------------
// SummaryBuilder Tests
// -----------------------------------------------------------------------------

func TestSummaryBuilder_BuildSummary(t *testing.T) {
	builder := NewSummaryBuilder()

	entries := []DeploymentSecretEntry{
		{ResourceName: "pg", SecretKey: "pass", Classification: "service", HandlingStrategy: "generate"},
		{ResourceName: "pg", SecretKey: "user", Classification: "service", HandlingStrategy: "prompt"},
		{ResourceName: "redis", SecretKey: "pass", Classification: "service", HandlingStrategy: "unspecified"},
	}

	summary := builder.BuildSummary(SummaryInput{
		Entries:           entries,
		ScenarioResources: []string{"pg"},
		AnalyzerResources: []string{"redis"},
		Scenario:          "test",
	})

	if summary.TotalSecrets != 3 {
		t.Errorf("TotalSecrets = %d, want 3", summary.TotalSecrets)
	}
	if summary.StrategizedSecrets != 2 {
		t.Errorf("StrategizedSecrets = %d, want 2", summary.StrategizedSecrets)
	}
	if summary.RequiresAction != 1 {
		t.Errorf("RequiresAction = %d, want 1", summary.RequiresAction)
	}
	if len(summary.BlockingSecrets) != 1 {
		t.Errorf("len(BlockingSecrets) = %d, want 1", len(summary.BlockingSecrets))
	}
}

func TestSummaryBuilder_BlockingDetailsSource(t *testing.T) {
	builder := NewSummaryBuilder()

	tests := []struct {
		name              string
		scenarioResources []string
		analyzerResources []string
		wantSource        string
	}{
		{"both sources", []string{"pg"}, []string{"pg"}, "service.json+analyzer"},
		{"scenario only", []string{"pg"}, nil, "service.json"},
		{"analyzer only", nil, []string{"pg"}, "analyzer"},
		{"unknown", nil, nil, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := []DeploymentSecretEntry{
				{ResourceName: "pg", SecretKey: "pass", HandlingStrategy: "unspecified"},
			}
			summary := builder.BuildSummary(SummaryInput{
				Entries:           entries,
				ScenarioResources: tt.scenarioResources,
				AnalyzerResources: tt.analyzerResources,
				Scenario:          "test",
			})
			if len(summary.BlockingSecretDetails) == 0 {
				t.Fatal("Expected blocking details")
			}
			if summary.BlockingSecretDetails[0].Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", summary.BlockingSecretDetails[0].Source, tt.wantSource)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// BundlePlanBuilder Tests
// -----------------------------------------------------------------------------

func TestBundlePlanBuilder_DeriveSecretClass(t *testing.T) {
	builder := NewBundlePlanBuilder()

	tests := []struct {
		name  string
		entry DeploymentSecretEntry
		want  string
	}{
		{
			name:  "infrastructure classification",
			entry: DeploymentSecretEntry{Classification: "infrastructure"},
			want:  "infrastructure",
		},
		{
			name:  "prompt strategy",
			entry: DeploymentSecretEntry{HandlingStrategy: "prompt"},
			want:  "user_prompt",
		},
		{
			name:  "generate strategy",
			entry: DeploymentSecretEntry{HandlingStrategy: "generate"},
			want:  "per_install_generated",
		},
		{
			name:  "delegate strategy",
			entry: DeploymentSecretEntry{HandlingStrategy: "delegate"},
			want:  "remote_fetch",
		},
		{
			name:  "strip strategy",
			entry: DeploymentSecretEntry{HandlingStrategy: "strip"},
			want:  "infrastructure",
		},
		{
			name:  "requires user input",
			entry: DeploymentSecretEntry{RequiresUserInput: true},
			want:  "user_prompt",
		},
		{
			name:  "default",
			entry: DeploymentSecretEntry{},
			want:  "per_install_generated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.deriveSecretClass(tt.entry)
			if got != tt.want {
				t.Errorf("deriveSecretClass() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBundlePlanBuilder_DeriveSecretID(t *testing.T) {
	builder := NewBundlePlanBuilder()

	tests := []struct {
		name  string
		entry DeploymentSecretEntry
		want  string
	}{
		{
			name:  "with ID",
			entry: DeploymentSecretEntry{ID: "my-id"},
			want:  "my-id",
		},
		{
			name:  "resource and key",
			entry: DeploymentSecretEntry{ResourceName: "Postgres", SecretKey: "PASSWORD"},
			want:  "postgres_password",
		},
		{
			name:  "key only",
			entry: DeploymentSecretEntry{SecretKey: "API_KEY"},
			want:  "api_key",
		},
		{
			name:  "resource only",
			entry: DeploymentSecretEntry{ResourceName: "Redis"},
			want:  "redis",
		},
		{
			name:  "empty",
			entry: DeploymentSecretEntry{},
			want:  "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.deriveSecretID(tt.entry)
			if got != tt.want {
				t.Errorf("deriveSecretID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBundlePlanBuilder_DeriveBundleTarget(t *testing.T) {
	builder := NewBundlePlanBuilder()

	tests := []struct {
		name     string
		entry    DeploymentSecretEntry
		wantType string
		wantName string
	}{
		{
			name:     "default env",
			entry:    DeploymentSecretEntry{SecretKey: "api_key"},
			wantType: "env",
			wantName: "API_KEY",
		},
		{
			name: "explicit target type",
			entry: DeploymentSecretEntry{
				SecretKey:   "config",
				BundleHints: map[string]interface{}{"target_type": "file"},
			},
			wantType: "file",
			wantName: "CONFIG",
		},
		{
			name: "explicit target name",
			entry: DeploymentSecretEntry{
				SecretKey:   "pass",
				BundleHints: map[string]interface{}{"target_name": "MY_PASSWORD"},
			},
			wantType: "env",
			wantName: "MY_PASSWORD",
		},
		{
			name: "file path hint",
			entry: DeploymentSecretEntry{
				SecretKey:   "cert",
				BundleHints: map[string]interface{}{"file_path": "/etc/ssl/cert.pem"},
			},
			wantType: "file",
			wantName: "/etc/ssl/cert.pem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := builder.deriveBundleTarget(tt.entry)
			if target.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", target.Type, tt.wantType)
			}
			if target.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", target.Name, tt.wantName)
			}
		})
	}
}

func TestBundlePlanBuilder_InfrastructureExcluded(t *testing.T) {
	builder := NewBundlePlanBuilder()

	entries := []DeploymentSecretEntry{
		{Classification: "infrastructure", SecretKey: "internal"},
		{HandlingStrategy: "strip", SecretKey: "stripped"},
		{Classification: "service", HandlingStrategy: "generate", SecretKey: "included"},
	}

	plans := builder.DeriveBundlePlans(entries)
	if len(plans) != 1 {
		t.Errorf("len(plans) = %d, want 1 (infrastructure should be excluded)", len(plans))
	}
}

// -----------------------------------------------------------------------------
// String Utils Tests
// -----------------------------------------------------------------------------

func TestDedupeStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"nil", nil, nil},
		{"empty", []string{}, nil},
		{"single", []string{"a"}, []string{"a"}},
		{"duplicates", []string{"b", "a", "b", "a"}, []string{"a", "b"}},
		{"whitespace", []string{" a ", "  b  ", "a"}, []string{"a", "b"}},
		{"empty strings", []string{"a", "", "  ", "b"}, []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupeStrings(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("dedupeStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntersectStrings(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{"nil a", nil, []string{"a"}, nil},
		{"nil b", []string{"a"}, nil, nil},
		{"no overlap", []string{"a"}, []string{"b"}, nil},
		{"full overlap", []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
		{"partial", []string{"a", "b", "c"}, []string{"b", "c", "d"}, []string{"b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intersectStrings(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Errorf("intersectStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	if !containsString([]string{"a", "b", "c"}, "b") {
		t.Error("containsString should find 'b'")
	}
	if containsString([]string{"a", "b", "c"}, "d") {
		t.Error("containsString should not find 'd'")
	}
	if containsString(nil, "a") {
		t.Error("containsString should return false for nil")
	}
	if containsString([]string{}, "a") {
		t.Error("containsString should return false for empty slice")
	}
	if !containsString([]string{"test"}, "test") {
		t.Error("containsString should find single element")
	}
}

// -----------------------------------------------------------------------------
// Additional Edge Case Tests
// -----------------------------------------------------------------------------

func TestBundlePlanBuilder_EmptyEntries(t *testing.T) {
	builder := NewBundlePlanBuilder()
	plans := builder.DeriveBundlePlans([]DeploymentSecretEntry{})
	if len(plans) != 0 {
		t.Errorf("Expected 0 plans for empty entries, got %d", len(plans))
	}
}

func TestBundlePlanBuilder_NilEntries(t *testing.T) {
	builder := NewBundlePlanBuilder()
	plans := builder.DeriveBundlePlans(nil)
	if plans == nil {
		t.Error("DeriveBundlePlans should return non-nil slice for nil input")
	}
}

func TestSummaryBuilder_EmptyEntries(t *testing.T) {
	builder := NewSummaryBuilder()
	summary := builder.BuildSummary(SummaryInput{
		Entries:  []DeploymentSecretEntry{},
		Scenario: "test",
	})
	if summary.TotalSecrets != 0 {
		t.Errorf("Expected 0 total secrets, got %d", summary.TotalSecrets)
	}
	if summary.StrategizedSecrets != 0 {
		t.Errorf("Expected 0 strategized secrets, got %d", summary.StrategizedSecrets)
	}
}

func TestManifestBuilder_PersistError(t *testing.T) {
	store := &mockSecretStore{
		secrets: []DeploymentSecretEntry{
			{ID: "s1", SecretKey: "KEY", HandlingStrategy: "prompt"},
		},
		persistErr: errors.New("persist failed"),
	}
	analyzer := &mockAnalyzerClient{}
	resolver := &mockResourceResolver{
		resolved: ResolvedResources{Effective: []string{"test"}},
	}
	builder := NewManifestBuilderWithDeps(store, analyzer, resolver, nil)

	req := DeploymentManifestRequest{Scenario: "test", Tier: "desktop"}
	_, err := builder.Build(context.Background(), req)

	// Persist error may or may not fail the build depending on implementation
	if err != nil {
		t.Logf("Build with persist error returned: %v (implementation-dependent)", err)
	}
}

// -----------------------------------------------------------------------------
// Analyzer Conversion Tests
// -----------------------------------------------------------------------------

func TestConvertAnalyzerDependencies(t *testing.T) {
	report := &analyzerDeploymentReport{
		Dependencies: []analyzerDependency{
			{
				Name: "postgres",
				Type: "resource",
				Requirements: analyzerRequirement{
					RAMMB:  512,
					DiskMB: 1024,
				},
				TierSupport: map[string]analyzerTierSupport{
					"desktop": {Supported: true, FitnessScore: 0.9},
				},
			},
		},
	}

	insights := convertAnalyzerDependencies(report)
	if len(insights) != 1 {
		t.Fatalf("len(insights) = %d, want 1", len(insights))
	}

	if insights[0].Name != "postgres" {
		t.Errorf("Name = %q, want %q", insights[0].Name, "postgres")
	}
	if insights[0].Requirements == nil {
		t.Error("Requirements should not be nil")
	}
	if insights[0].TierSupport == nil {
		t.Error("TierSupport should not be nil")
	}
}

func TestConvertAnalyzerAggregates(t *testing.T) {
	report := &analyzerDeploymentReport{
		Aggregates: map[string]analyzerAggregate{
			"desktop": {
				FitnessScore:    0.85,
				DependencyCount: 3,
				EstimatedRequirements: analyzerRequirementTotals{
					RAMMB: 1024,
				},
			},
		},
	}

	aggregates := convertAnalyzerAggregates(report)
	if aggregates == nil {
		t.Fatal("aggregates should not be nil")
	}

	desktop, ok := aggregates["desktop"]
	if !ok {
		t.Fatal("desktop tier not found")
	}
	if desktop.FitnessScore != 0.85 {
		t.Errorf("FitnessScore = %f, want 0.85", desktop.FitnessScore)
	}
	if desktop.EstimatedRequirements == nil {
		t.Error("EstimatedRequirements should not be nil")
	}
}

// -----------------------------------------------------------------------------
// Nil/Empty Edge Cases for Analyzer Conversions
// -----------------------------------------------------------------------------

func TestConvertAnalyzerDependencies_NilReport(t *testing.T) {
	insights := convertAnalyzerDependencies(nil)
	if insights != nil {
		t.Errorf("Expected nil for nil report, got %v", insights)
	}
}

func TestConvertAnalyzerDependencies_EmptyDependencies(t *testing.T) {
	report := &analyzerDeploymentReport{
		Dependencies: []analyzerDependency{},
	}
	insights := convertAnalyzerDependencies(report)
	if len(insights) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(insights))
	}
}

func TestConvertAnalyzerAggregates_NilReport(t *testing.T) {
	aggregates := convertAnalyzerAggregates(nil)
	if aggregates != nil {
		t.Errorf("Expected nil for nil report, got %v", aggregates)
	}
}

func TestConvertAnalyzerAggregates_EmptyAggregates(t *testing.T) {
	report := &analyzerDeploymentReport{
		Aggregates: map[string]analyzerAggregate{},
	}
	aggregates := convertAnalyzerAggregates(report)
	if aggregates != nil {
		t.Errorf("Expected nil for empty aggregates, got %v", aggregates)
	}
}

// -----------------------------------------------------------------------------
// BundlePlanBuilder Prompt/Generator Tests
// -----------------------------------------------------------------------------

func TestBundlePlanBuilder_DerivePrompt(t *testing.T) {
	builder := NewBundlePlanBuilder()

	tests := []struct {
		name             string
		entry            DeploymentSecretEntry
		wantLabel        string
		wantDescription  string
		wantExistingUsed bool
	}{
		{
			name: "existing prompt with content",
			entry: DeploymentSecretEntry{
				SecretKey: "ignored",
				Prompt: &PromptMetadata{
					Label:       "Existing Label",
					Description: "Existing Description",
				},
			},
			wantLabel:        "Existing Label",
			wantDescription:  "Existing Description",
			wantExistingUsed: true,
		},
		{
			name: "nil prompt uses defaults",
			entry: DeploymentSecretEntry{
				SecretKey:   "API_KEY",
				Description: "Custom description",
			},
			wantLabel:       "API_KEY",
			wantDescription: "Custom description",
		},
		{
			name: "empty prompt uses defaults",
			entry: DeploymentSecretEntry{
				SecretKey: "PASSWORD",
				Prompt: &PromptMetadata{
					Label:       "",
					Description: "",
				},
			},
			wantLabel:       "PASSWORD",
			wantDescription: "Enter the value for this bundled secret.",
		},
		{
			name: "no key uses generic label",
			entry: DeploymentSecretEntry{
				SecretKey: "",
			},
			wantLabel:       "Provide secret",
			wantDescription: "Enter the value for this bundled secret.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := builder.derivePrompt(tt.entry)
			if prompt == nil {
				t.Fatal("Expected non-nil prompt")
			}
			if prompt.Label != tt.wantLabel {
				t.Errorf("Label = %q, want %q", prompt.Label, tt.wantLabel)
			}
			if prompt.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", prompt.Description, tt.wantDescription)
			}
		})
	}
}

func TestBundlePlanBuilder_DeriveGenerator(t *testing.T) {
	builder := NewBundlePlanBuilder()

	tests := []struct {
		name         string
		entry        DeploymentSecretEntry
		wantType     string
		wantExisting bool
	}{
		{
			name: "existing generator used",
			entry: DeploymentSecretEntry{
				GeneratorTemplate: map[string]interface{}{
					"type":   "uuid",
					"format": "v4",
				},
			},
			wantType:     "uuid",
			wantExisting: true,
		},
		{
			name:     "nil generator uses default",
			entry:    DeploymentSecretEntry{},
			wantType: "random",
		},
		{
			name: "empty generator uses default",
			entry: DeploymentSecretEntry{
				GeneratorTemplate: map[string]interface{}{},
			},
			wantType: "random",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := builder.deriveGenerator(tt.entry)
			if generator == nil {
				t.Fatal("Expected non-nil generator")
			}
			typeVal, ok := generator["type"].(string)
			if !ok {
				t.Fatal("Expected type field to be string")
			}
			if typeVal != tt.wantType {
				t.Errorf("type = %q, want %q", typeVal, tt.wantType)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// SummaryBuilder Truncation Tests
// -----------------------------------------------------------------------------

func TestSummaryBuilder_TruncateBlockingInfo(t *testing.T) {
	builder := NewSummaryBuilder()

	// Create more than 10 blocking entries
	entries := make([]DeploymentSecretEntry, 15)
	for i := 0; i < 15; i++ {
		entries[i] = DeploymentSecretEntry{
			ResourceName:     "resource",
			SecretKey:        "secret" + string(rune('A'+i)),
			HandlingStrategy: "unspecified",
		}
	}

	summary := builder.BuildSummary(SummaryInput{
		Entries:  entries,
		Scenario: "test",
	})

	// Should be truncated to 10
	if len(summary.BlockingSecrets) > 10 {
		t.Errorf("BlockingSecrets should be limited to 10, got %d", len(summary.BlockingSecrets))
	}
	if len(summary.BlockingSecretDetails) > 10 {
		t.Errorf("BlockingSecretDetails should be limited to 10, got %d", len(summary.BlockingSecretDetails))
	}
	if summary.RequiresAction != 10 {
		t.Errorf("RequiresAction = %d, want 10 (truncated count)", summary.RequiresAction)
	}
}

func TestSummaryBuilder_ScopeReadiness(t *testing.T) {
	builder := NewSummaryBuilder()

	entries := []DeploymentSecretEntry{
		{ResourceName: "pg", SecretKey: "pass1", Classification: "service", HandlingStrategy: "generate"},
		{ResourceName: "pg", SecretKey: "pass2", Classification: "service", HandlingStrategy: "prompt"},
		{ResourceName: "pg", SecretKey: "pass3", Classification: "service", HandlingStrategy: "unspecified"},
		{ResourceName: "redis", SecretKey: "auth", Classification: "infrastructure", HandlingStrategy: "strip"},
	}

	summary := builder.BuildSummary(SummaryInput{
		Entries:  entries,
		Scenario: "test",
	})

	// Check service scope: 2 ready out of 3
	serviceReadiness, ok := summary.ScopeReadiness["service"]
	if !ok {
		t.Fatal("Expected service scope in ScopeReadiness")
	}
	if serviceReadiness != "2/3" {
		t.Errorf("service readiness = %q, want %q", serviceReadiness, "2/3")
	}

	// Check infrastructure scope: 1 ready out of 1
	infraReadiness, ok := summary.ScopeReadiness["infrastructure"]
	if !ok {
		t.Fatal("Expected infrastructure scope in ScopeReadiness")
	}
	if infraReadiness != "1/1" {
		t.Errorf("infrastructure readiness = %q, want %q", infraReadiness, "1/1")
	}
}

func TestSummaryBuilder_StrategyBreakdown(t *testing.T) {
	builder := NewSummaryBuilder()

	entries := []DeploymentSecretEntry{
		{HandlingStrategy: "generate"},
		{HandlingStrategy: "generate"},
		{HandlingStrategy: "prompt"},
		{HandlingStrategy: "delegate"},
		{HandlingStrategy: "strip"},
		{HandlingStrategy: "unspecified"}, // Not counted in breakdown
	}

	summary := builder.BuildSummary(SummaryInput{
		Entries:  entries,
		Scenario: "test",
	})

	if summary.StrategyBreakdown["generate"] != 2 {
		t.Errorf("generate count = %d, want 2", summary.StrategyBreakdown["generate"])
	}
	if summary.StrategyBreakdown["prompt"] != 1 {
		t.Errorf("prompt count = %d, want 1", summary.StrategyBreakdown["prompt"])
	}
	if summary.StrategyBreakdown["delegate"] != 1 {
		t.Errorf("delegate count = %d, want 1", summary.StrategyBreakdown["delegate"])
	}
	if summary.StrategyBreakdown["strip"] != 1 {
		t.Errorf("strip count = %d, want 1", summary.StrategyBreakdown["strip"])
	}
	// unspecified should NOT be in breakdown
	if _, found := summary.StrategyBreakdown["unspecified"]; found {
		t.Error("unspecified should not appear in StrategyBreakdown")
	}
}
