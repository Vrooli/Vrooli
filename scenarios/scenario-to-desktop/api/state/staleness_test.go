package state

import (
	"context"
	"reflect"
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

func TestStalenessDetector_DetectChanges(t *testing.T) {
	stateWith := func(stage string, fingerprint InputFingerprint) *ScenarioState {
		return &ScenarioState{
			ScenarioName: "test-scenario",
			Stages: map[string]StageState{
				stage: {Stage: stage, Status: StatusValid, InputFingerprint: fingerprint},
			},
		}
	}

	tests := []struct {
		name    string
		stored  *ScenarioState
		current *InputFingerprint
		want    []StateChange
	}{
		{
			name:    "no stored state",
			current: &InputFingerprint{ManifestPath: "/path/to/manifest.json", ManifestHash: "newhash"},
		},
		{
			name: "manifest path changed",
			stored: stateWith(StageBundle, InputFingerprint{
				ManifestPath: "/old/path/manifest.json", ManifestHash: "oldhash",
			}),
			current: &InputFingerprint{ManifestPath: "/new/path/manifest.json", ManifestHash: "newhash"},
			want: []StateChange{{
				ChangeType: "manifest_path", AffectedStage: StageBundle, Reason: "Manifest path changed",
				OldValue: "/old/path/manifest.json", NewValue: "/new/path/manifest.json",
			}},
		},
		{
			name: "manifest content changed",
			stored: stateWith(StageBundle, InputFingerprint{
				ManifestPath: "/path/manifest.json", ManifestHash: "oldhash123",
			}),
			current: &InputFingerprint{ManifestPath: "/path/manifest.json", ManifestHash: "newhash456"},
			want: []StateChange{{
				ChangeType: "manifest_content", AffectedStage: StagePreflight,
				Reason: "Manifest file content changed", OldValue: "oldhash123", NewValue: "newhash456",
			}},
		},
		{
			name: "preflight secret keys changed",
			stored: stateWith(StagePreflight, InputFingerprint{
				PreflightSecretKeys: []string{"SECRET_A", "SECRET_B"},
			}),
			current: &InputFingerprint{PreflightSecretKeys: []string{"SECRET_A", "SECRET_C"}},
			want: []StateChange{{
				ChangeType: "preflight_secrets", AffectedStage: StagePreflight,
				Reason: "Preflight secrets configuration changed",
			}},
		},
		{
			name: "generate inputs changed",
			stored: stateWith(StageGenerate, InputFingerprint{
				TemplateType: "spa", Framework: "electron", DeploymentMode: "bundled", AppDisplayName: "Old Name",
			}),
			current: &InputFingerprint{
				TemplateType: "basic", Framework: "electron", DeploymentMode: "bundled", AppDisplayName: "New App Name",
			},
			want: []StateChange{
				{ChangeType: "template_type", AffectedStage: StageGenerate, Reason: "Template type changed", OldValue: "spa", NewValue: "basic"},
				{ChangeType: "app_display_name", AffectedStage: StageGenerate, Reason: "App display name changed", OldValue: "Old Name", NewValue: "New App Name"},
			},
		},
		{
			name: "build inputs changed",
			stored: stateWith(StageBuild, InputFingerprint{
				Platforms: []string{"linux", "win"}, SigningEnabled: false, OutputLocation: "proper",
			}),
			current: &InputFingerprint{Platforms: []string{"linux", "mac"}, SigningEnabled: true, OutputLocation: "proper"},
			want: []StateChange{
				{ChangeType: "platforms", AffectedStage: StageBuild, Reason: "Target platforms changed"},
				{ChangeType: "signing_enabled", AffectedStage: StageBuild, Reason: "Signing configuration toggled"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := testStore(t)
			ctx := context.Background()
			if tt.stored != nil {
				if err := store.Save(ctx, tt.stored); err != nil {
					t.Fatalf("save stored state: %v", err)
				}
			}

			changes, err := NewStalenessDetector(store).DetectChanges(ctx, "test-scenario", tt.current)
			if err != nil {
				t.Fatalf("DetectChanges() error = %v", err)
			}
			if !reflect.DeepEqual(changes, tt.want) {
				t.Fatalf("DetectChanges() = %#v, want %#v", changes, tt.want)
			}
		})
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

// allValidStages returns a ScenarioState where every stage is StatusValid.
func allValidStages(name string) *ScenarioState {
	now := time.Now()
	return &ScenarioState{
		ScenarioName: name,
		Stages: map[string]StageState{
			StageBundle:    {Stage: StageBundle, Status: StatusValid, ValidatedAt: now},
			StagePreflight: {Stage: StagePreflight, Status: StatusValid, ValidatedAt: now},
			StageGenerate:  {Stage: StageGenerate, Status: StatusValid, ValidatedAt: now},
			StageBuild:     {Stage: StageBuild, Status: StatusValid, ValidatedAt: now},
			StageSmokeTest: {Stage: StageSmokeTest, Status: StatusValid, ValidatedAt: now},
		},
	}
}

// assertAllStagesStatus checks that every stage in the validation status matches wantStatus.
func assertAllStagesStatus(t *testing.T, status *ValidationStatus, wantStatus string) {
	t.Helper()
	for _, stage := range StageOrder {
		if status.Stages[stage].Status != wantStatus {
			t.Errorf("stage %s status = %s, want %s", stage, status.Stages[stage].Status, wantStatus)
		}
	}
}

// assertStageStatus checks a single stage's status in the validation result.
func assertStageStatus(t *testing.T, status *ValidationStatus, stage, wantStatus string) {
	t.Helper()
	if status.Stages[stage].Status != wantStatus {
		t.Errorf("stage %s status = %s, want %s", stage, status.Stages[stage].Status, wantStatus)
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
		assertAllStagesStatus(t, status, StatusNone)
	})

	t.Run("all valid, no changes", func(t *testing.T) {
		stored := allValidStages("test")
		status := detector.BuildValidationStatus("test", stored, nil)

		if status.OverallStatus != StatusValid {
			t.Errorf("OverallStatus = %s, want valid", status.OverallStatus)
		}
		assertAllStagesStatus(t, status, StatusValid)
		for _, stage := range StageOrder {
			if !status.Stages[stage].CanReuse {
				t.Errorf("stage %s CanReuse should be true", stage)
			}
		}
	})

	t.Run("with changes - partial status", func(t *testing.T) {
		stored := allValidStages("test")
		changes := []StateChange{
			{ChangeType: "template_type", AffectedStage: StageGenerate, Reason: "Template changed"},
		}

		status := detector.BuildValidationStatus("test", stored, changes)

		if status.OverallStatus != "partial" {
			t.Errorf("OverallStatus = %s, want partial", status.OverallStatus)
		}

		assertStageStatus(t, status, StageBundle, StatusValid)
		assertStageStatus(t, status, StagePreflight, StatusValid)
		assertStageStatus(t, status, StageGenerate, StatusStale)
		assertStageStatus(t, status, StageBuild, StatusStale)
		assertStageStatus(t, status, StageSmokeTest, StatusStale)

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
