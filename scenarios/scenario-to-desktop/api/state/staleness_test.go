package state

import (
	"context"
	"testing"
	"time"
)

// testStore creates a temporary store for testing
func testStore(t *testing.T) *Store {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return store
}

func TestStalenessDetector_DetectChanges_NoStoredState(t *testing.T) {
	store := testStore(t)
	detector := NewStalenessDetector(store)
	ctx := context.Background()

	current := &InputFingerprint{
		ManifestPath: "/path/to/manifest.json",
		ManifestHash: "newhash",
	}

	changes, err := detector.DetectChanges(ctx, "nonexistent-scenario", current)
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}
	if changes != nil {
		t.Errorf("expected nil changes for nonexistent scenario, got %v", changes)
	}
}

func TestStalenessDetector_DetectChanges_ManifestPathChanged(t *testing.T) {
	store := testStore(t)
	detector := NewStalenessDetector(store)
	ctx := context.Background()

	// Store state with bundle stage
	state := &ScenarioState{
		ScenarioName: "test-scenario",
		Stages: map[string]StageState{
			StageBundle: {
				Stage:  StageBundle,
				Status: StatusValid,
				InputFingerprint: InputFingerprint{
					ManifestPath: "/old/path/manifest.json",
					ManifestHash: "oldhash",
				},
			},
		},
	}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Check with different manifest path
	current := &InputFingerprint{
		ManifestPath: "/new/path/manifest.json",
		ManifestHash: "newhash",
	}

	changes, err := detector.DetectChanges(ctx, "test-scenario", current)
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].ChangeType != "manifest_path" {
		t.Errorf("change type = %s, want manifest_path", changes[0].ChangeType)
	}
	if changes[0].AffectedStage != StageBundle {
		t.Errorf("affected stage = %s, want %s", changes[0].AffectedStage, StageBundle)
	}
}

func TestStalenessDetector_DetectChanges_ManifestContentChanged(t *testing.T) {
	store := testStore(t)
	detector := NewStalenessDetector(store)
	ctx := context.Background()

	// Store state with bundle stage
	state := &ScenarioState{
		ScenarioName: "test-scenario",
		Stages: map[string]StageState{
			StageBundle: {
				Stage:  StageBundle,
				Status: StatusValid,
				InputFingerprint: InputFingerprint{
					ManifestPath: "/path/manifest.json",
					ManifestHash: "oldhash123",
				},
			},
		},
	}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Check with same path but different hash
	current := &InputFingerprint{
		ManifestPath: "/path/manifest.json",
		ManifestHash: "newhash456",
	}

	changes, err := detector.DetectChanges(ctx, "test-scenario", current)
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].ChangeType != "manifest_content" {
		t.Errorf("change type = %s, want manifest_content", changes[0].ChangeType)
	}
	// Content change affects preflight, not bundle
	if changes[0].AffectedStage != StagePreflight {
		t.Errorf("affected stage = %s, want %s", changes[0].AffectedStage, StagePreflight)
	}
}

func TestStalenessDetector_DetectChanges_PreflightSecretsChanged(t *testing.T) {
	store := testStore(t)
	detector := NewStalenessDetector(store)
	ctx := context.Background()

	state := &ScenarioState{
		ScenarioName: "test-scenario",
		Stages: map[string]StageState{
			StagePreflight: {
				Stage:  StagePreflight,
				Status: StatusValid,
				InputFingerprint: InputFingerprint{
					PreflightSecretKeys: []string{"SECRET_A", "SECRET_B"},
				},
			},
		},
	}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Check with different secrets
	current := &InputFingerprint{
		PreflightSecretKeys: []string{"SECRET_A", "SECRET_C"}, // B changed to C
	}

	changes, err := detector.DetectChanges(ctx, "test-scenario", current)
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].ChangeType != "preflight_secrets" {
		t.Errorf("change type = %s, want preflight_secrets", changes[0].ChangeType)
	}
}

func TestStalenessDetector_DetectChanges_GenerateInputsChanged(t *testing.T) {
	store := testStore(t)
	detector := NewStalenessDetector(store)
	ctx := context.Background()

	state := &ScenarioState{
		ScenarioName: "test-scenario",
		Stages: map[string]StageState{
			StageGenerate: {
				Stage:  StageGenerate,
				Status: StatusValid,
				InputFingerprint: InputFingerprint{
					TemplateType:   "spa",
					Framework:      "electron",
					DeploymentMode: "bundled",
					AppDisplayName: "Old Name",
				},
			},
		},
	}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Check with different template and name
	current := &InputFingerprint{
		TemplateType:   "basic",        // Changed
		Framework:      "electron",     // Same
		DeploymentMode: "bundled",      // Same
		AppDisplayName: "New App Name", // Changed
	}

	changes, err := detector.DetectChanges(ctx, "test-scenario", current)
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}
	// Should have 2 changes: template_type and app_display_name
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d: %+v", len(changes), changes)
	}

	changeTypes := make(map[string]bool)
	for _, c := range changes {
		changeTypes[c.ChangeType] = true
	}
	if !changeTypes["template_type"] {
		t.Error("expected template_type change")
	}
	if !changeTypes["app_display_name"] {
		t.Error("expected app_display_name change")
	}
}

func TestStalenessDetector_DetectChanges_BuildInputsChanged(t *testing.T) {
	store := testStore(t)
	detector := NewStalenessDetector(store)
	ctx := context.Background()

	state := &ScenarioState{
		ScenarioName: "test-scenario",
		Stages: map[string]StageState{
			StageBuild: {
				Stage:  StageBuild,
				Status: StatusValid,
				InputFingerprint: InputFingerprint{
					Platforms:      []string{"linux", "win"},
					SigningEnabled: false,
					OutputLocation: "proper",
				},
			},
		},
	}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Check with different platforms and signing
	current := &InputFingerprint{
		Platforms:      []string{"linux", "mac"}, // win -> mac
		SigningEnabled: true,                     // toggled
		OutputLocation: "proper",
	}

	changes, err := detector.DetectChanges(ctx, "test-scenario", current)
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d: %+v", len(changes), changes)
	}

	changeTypes := make(map[string]bool)
	for _, c := range changes {
		changeTypes[c.ChangeType] = true
	}
	if !changeTypes["platforms"] {
		t.Error("expected platforms change")
	}
	if !changeTypes["signing_enabled"] {
		t.Error("expected signing_enabled change")
	}
}

func TestStalenessDetector_ComputeAffectedStages(t *testing.T) {
	detector := &StalenessDetector{}

	tests := []struct {
		name    string
		changes []StateChange
		want    []string
	}{
		{
			name:    "no changes",
			changes: nil,
			want:    nil,
		},
		{
			name: "bundle affected - all stages",
			changes: []StateChange{
				{AffectedStage: StageBundle},
			},
			want: StageOrder,
		},
		{
			name: "preflight affected",
			changes: []StateChange{
				{AffectedStage: StagePreflight},
			},
			want: []string{StagePreflight, StageGenerate, StageBuild, StageSmokeTest},
		},
		{
			name: "generate affected",
			changes: []StateChange{
				{AffectedStage: StageGenerate},
			},
			want: []string{StageGenerate, StageBuild, StageSmokeTest},
		},
		{
			name: "build affected",
			changes: []StateChange{
				{AffectedStage: StageBuild},
			},
			want: []string{StageBuild, StageSmokeTest},
		},
		{
			name: "smoke test affected only",
			changes: []StateChange{
				{AffectedStage: StageSmokeTest},
			},
			want: []string{StageSmokeTest},
		},
		{
			name: "multiple changes - earliest wins",
			changes: []StateChange{
				{AffectedStage: StageBuild},
				{AffectedStage: StagePreflight},
				{AffectedStage: StageGenerate},
			},
			want: []string{StagePreflight, StageGenerate, StageBuild, StageSmokeTest},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.ComputeAffectedStages(tt.changes)
			if len(got) != len(tt.want) {
				t.Errorf("ComputeAffectedStages() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ComputeAffectedStages()[%d] = %s, want %s", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStalenessDetector_BuildValidationStatus(t *testing.T) {
	detector := &StalenessDetector{}

	t.Run("nil stored state", func(t *testing.T) {
		status := detector.BuildValidationStatus("test", nil, nil)

		if status.ScenarioName != "test" {
			t.Errorf("ScenarioName = %s, want test", status.ScenarioName)
		}
		if status.OverallStatus != StatusNone {
			t.Errorf("OverallStatus = %s, want %s", status.OverallStatus, StatusNone)
		}
		// All stages should be "none"
		for _, stage := range StageOrder {
			if status.Stages[stage].Status != StatusNone {
				t.Errorf("stage %s status = %s, want none", stage, status.Stages[stage].Status)
			}
		}
	})

	t.Run("all valid, no changes", func(t *testing.T) {
		stored := &ScenarioState{
			ScenarioName: "test",
			Stages: map[string]StageState{
				StageBundle:    {Stage: StageBundle, Status: StatusValid, ValidatedAt: time.Now()},
				StagePreflight: {Stage: StagePreflight, Status: StatusValid, ValidatedAt: time.Now()},
				StageGenerate:  {Stage: StageGenerate, Status: StatusValid, ValidatedAt: time.Now()},
				StageBuild:     {Stage: StageBuild, Status: StatusValid, ValidatedAt: time.Now()},
				StageSmokeTest: {Stage: StageSmokeTest, Status: StatusValid, ValidatedAt: time.Now()},
			},
		}

		status := detector.BuildValidationStatus("test", stored, nil)

		if status.OverallStatus != StatusValid {
			t.Errorf("OverallStatus = %s, want valid", status.OverallStatus)
		}
		for _, stage := range StageOrder {
			if status.Stages[stage].Status != StatusValid {
				t.Errorf("stage %s status = %s, want valid", stage, status.Stages[stage].Status)
			}
			if !status.Stages[stage].CanReuse {
				t.Errorf("stage %s CanReuse should be true", stage)
			}
		}
	})

	t.Run("with changes - partial status", func(t *testing.T) {
		stored := &ScenarioState{
			ScenarioName: "test",
			Stages: map[string]StageState{
				StageBundle:    {Stage: StageBundle, Status: StatusValid, ValidatedAt: time.Now()},
				StagePreflight: {Stage: StagePreflight, Status: StatusValid, ValidatedAt: time.Now()},
				StageGenerate:  {Stage: StageGenerate, Status: StatusValid, ValidatedAt: time.Now()},
				StageBuild:     {Stage: StageBuild, Status: StatusValid, ValidatedAt: time.Now()},
				StageSmokeTest: {Stage: StageSmokeTest, Status: StatusValid, ValidatedAt: time.Now()},
			},
		}

		changes := []StateChange{
			{ChangeType: "template_type", AffectedStage: StageGenerate, Reason: "Template changed"},
		}

		status := detector.BuildValidationStatus("test", stored, changes)

		// Overall should be partial (some valid, some stale)
		if status.OverallStatus != "partial" {
			t.Errorf("OverallStatus = %s, want partial", status.OverallStatus)
		}

		// Bundle and preflight should still be valid
		if status.Stages[StageBundle].Status != StatusValid {
			t.Errorf("bundle status = %s, want valid", status.Stages[StageBundle].Status)
		}
		if status.Stages[StagePreflight].Status != StatusValid {
			t.Errorf("preflight status = %s, want valid", status.Stages[StagePreflight].Status)
		}

		// Generate, build, smoke_test should be stale
		if status.Stages[StageGenerate].Status != StatusStale {
			t.Errorf("generate status = %s, want stale", status.Stages[StageGenerate].Status)
		}
		if status.Stages[StageBuild].Status != StatusStale {
			t.Errorf("build status = %s, want stale", status.Stages[StageBuild].Status)
		}
		if status.Stages[StageSmokeTest].Status != StatusStale {
			t.Errorf("smoketest status = %s, want stale", status.Stages[StageSmokeTest].Status)
		}

		// Generate should have staleness reason
		if status.Stages[StageGenerate].StalenessReason != "Template changed" {
			t.Errorf("generate staleness reason = %s, want 'Template changed'",
				status.Stages[StageGenerate].StalenessReason)
		}
	})
}

func TestTruncateHash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"short", "short"},
		{"exactly12ch", "exactly12ch"},
		{"0123456789abcdef", "0123456789ab"},
		{"verylonghashabcdef0123456789", "verylonghash"}, // truncate at 12 chars
	}

	for _, tt := range tests {
		got := truncateHash(tt.input)
		if got != tt.want {
			t.Errorf("truncateHash(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStringSlicesEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    []string{},
			b:    []string{},
			want: true,
		},
		{
			name: "equal slices same order",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "c"},
			want: true,
		},
		{
			name: "equal slices different order",
			a:    []string{"c", "a", "b"},
			b:    []string{"a", "b", "c"},
			want: true,
		},
		{
			name: "different lengths",
			a:    []string{"a", "b"},
			b:    []string{"a", "b", "c"},
			want: false,
		},
		{
			name: "different content",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "d"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringSlicesEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("stringSlicesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
