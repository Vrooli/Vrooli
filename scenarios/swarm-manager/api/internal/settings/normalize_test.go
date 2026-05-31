package settings

import (
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

func TestNormalizeSettingsDefaultsTheme(t *testing.T) {
	normalized := normalizeSettings(Settings{Theme: ""})
	if normalized.Theme != "dark" {
		t.Fatalf("expected default theme dark, got %q", normalized.Theme)
	}
}

func TestNormalizeFixBeforeFeature(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", FixBeforeFeatureSuggest},
		{"off", FixBeforeFeatureOff},
		{"suggest", FixBeforeFeatureSuggest},
		{"block", FixBeforeFeatureBlock},
		{"bogus", FixBeforeFeatureSuggest},
	}
	for _, tt := range tests {
		got := normalizeSettings(Settings{FixBeforeFeature: tt.in}).FixBeforeFeature
		if got != tt.want {
			t.Errorf("normalize fix_before_feature %q = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDefaultSettingsFixBeforeFeature(t *testing.T) {
	d := DefaultSettings()
	if d.FixBeforeFeature != FixBeforeFeatureSuggest {
		t.Errorf("default fix_before_feature = %q, want suggest", d.FixBeforeFeature)
	}
	if d.FixBeforeFeatureDiscovery {
		t.Errorf("default fix_before_feature_discovery = true, want false")
	}
}

func TestApplyPatchFixBeforeFeature(t *testing.T) {
	block := FixBeforeFeatureBlock
	enabled := true
	patched := applyPatch(DefaultSettings(), SettingsPatch{
		FixBeforeFeature:          &block,
		FixBeforeFeatureDiscovery: &enabled,
	})
	if patched.FixBeforeFeature != FixBeforeFeatureBlock {
		t.Errorf("patched fix_before_feature = %q, want block", patched.FixBeforeFeature)
	}
	if !patched.FixBeforeFeatureDiscovery {
		t.Errorf("patched fix_before_feature_discovery = false, want true")
	}
}

func TestDeleteConfirmLevelProtoDefaulting(t *testing.T) {
	tests := []struct {
		name  string
		proto domainpb.DeleteConfirmLevel
		want  DeleteConfirmLevel
	}{
		{
			name:  "unspecified defaults to simple",
			proto: domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_UNSPECIFIED,
			want:  DeleteConfirmSimple,
		},
		{
			name:  "simple round-trips",
			proto: domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_SIMPLE,
			want:  DeleteConfirmSimple,
		},
		{
			name:  "none round-trips",
			proto: domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_NONE,
			want:  DeleteConfirmNone,
		},
		{
			name:  "strong round-trips",
			proto: domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_STRONG,
			want:  DeleteConfirmStrong,
		},
		{
			name:  "unknown defaults to simple",
			proto: domainpb.DeleteConfirmLevel(99),
			want:  DeleteConfirmSimple,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := deleteConfirmLevelFromProto(tc.proto); got != tc.want {
				t.Fatalf("deleteConfirmLevelFromProto(%v) = %q, want %q", tc.proto, got, tc.want)
			}
		})
	}
}

func TestDeleteConfirmLevelToProtoUsesExplicitSimple(t *testing.T) {
	if got := deleteConfirmLevelToProto(DeleteConfirmSimple); got != domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_SIMPLE {
		t.Fatalf("deleteConfirmLevelToProto(simple) = %v, want SIMPLE", got)
	}
	if got := deleteConfirmLevelToProto(DeleteConfirmLevel("")); got != domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_SIMPLE {
		t.Fatalf("deleteConfirmLevelToProto(empty) = %v, want SIMPLE", got)
	}
}

func TestNormalizeDeleteConfirmationLevels_FillsAndPreserves(t *testing.T) {
	// Provide one valid known override, one invalid known value, omit the
	// rest, and include an unknown forward-compat key.
	raw := map[string]DeleteConfirmLevel{
		"session":     DeleteConfirmStrong, // valid override
		"capture":     DeleteConfirmLevel("bogus"),
		"futureThing": DeleteConfirmNone, // unknown key, valid value
		"futureBad":   DeleteConfirmLevel("x"),
	}
	out := normalizeDeleteConfirmationLevels(raw)

	// Every known registry key must be present.
	for key, def := range deletableEntityDefaults {
		got, ok := out[key]
		if !ok {
			t.Fatalf("missing known key %q", key)
		}
		switch key {
		case "session":
			if got != DeleteConfirmStrong {
				t.Errorf("session = %q, want strong (override honored)", got)
			}
		case "capture":
			if got != def {
				t.Errorf("capture = %q, want default %q (invalid coerced)", got, def)
			}
		default:
			if got != def {
				t.Errorf("%s = %q, want default %q", key, got, def)
			}
		}
	}
	// Unknown key with a valid value is preserved.
	if out["futureThing"] != DeleteConfirmNone {
		t.Errorf("futureThing = %q, want none (preserved)", out["futureThing"])
	}
	// Unknown key with an invalid value is coerced to simple, not dropped.
	if out["futureBad"] != DeleteConfirmSimple {
		t.Errorf("futureBad = %q, want simple (coerced, preserved)", out["futureBad"])
	}
}

func TestApplyPatchDeleteConfirmationLevels_Merges(t *testing.T) {
	current := DefaultSettings()
	current.DeleteConfirmationLevels["futureThing"] = DeleteConfirmStrong

	level := DeleteConfirmNone
	patched := applyPatch(current, SettingsPatch{
		DeleteConfirmationLevels: map[string]DeleteConfirmLevel{"session": level},
	})

	if patched.DeleteConfirmationLevels["session"] != DeleteConfirmNone {
		t.Errorf("session = %q, want none (patched)", patched.DeleteConfirmationLevels["session"])
	}
	// Unknown existing key not in the patch survives the merge.
	if patched.DeleteConfirmationLevels["futureThing"] != DeleteConfirmStrong {
		t.Errorf("futureThing = %q, want strong (preserved across patch)", patched.DeleteConfirmationLevels["futureThing"])
	}
}

func TestNormalizeIntFields(t *testing.T) {
	type intCase struct {
		name  string
		input int
		want  int
	}

	fields := []struct {
		field string
		cases []intCase
		build func(int) Settings
		get   func(Settings) int
	}{
		{
			field: "MaxQueueDepth",
			cases: []intCase{
				{"negative clamped to 0", -1, 0},
				{"zero allowed (unlimited)", 0, 0},
				{"mid range", 50, 50},
				{"max boundary", 100, 100},
				{"over max clamped to 100", 200, 100},
			},
			build: func(v int) Settings { return Settings{MaxQueueDepth: v} },
			get:   func(s Settings) int { return s.MaxQueueDepth },
		},
		{
			field: "CircuitBreakerThreshold",
			cases: []intCase{
				{"zero clamped to 1", 0, 1},
				{"min boundary", 1, 1},
				{"mid range", 5, 5},
				{"max boundary", 10, 10},
				{"over max clamped to 10", 15, 10},
			},
			build: func(v int) Settings { return Settings{CircuitBreakerThreshold: v} },
			get:   func(s Settings) int { return s.CircuitBreakerThreshold },
		},
		{
			field: "CircuitBreakerCooldownMinutes",
			cases: []intCase{
				{"zero clamped to 5", 0, 5},
				{"below min clamped to 5", 3, 5},
				{"min boundary", 5, 5},
				{"mid range", 60, 60},
				{"max boundary", 1440, 1440},
				{"over max clamped to 1440", 2000, 1440},
			},
			build: func(v int) Settings { return Settings{CircuitBreakerCooldownMinutes: v} },
			get:   func(s Settings) int { return s.CircuitBreakerCooldownMinutes },
		},
	}

	for _, f := range fields {
		t.Run(f.field, func(t *testing.T) {
			for _, tc := range f.cases {
				t.Run(tc.name, func(t *testing.T) {
					got := f.get(normalizeSettings(f.build(tc.input)))
					if got != tc.want {
						t.Fatalf("%s: got %d, want %d", f.field, got, tc.want)
					}
				})
			}
		})
	}
}

func TestNormalizeLaneConcurrencyLimits_FillsMissingKeys(t *testing.T) {
	// Empty / nil input should resolve to the canonical defaults so
	// settings stored before P2 (no lane_concurrency_limits) load cleanly.
	got := normalizeLaneConcurrencyLimits(nil)
	want := defaultLaneConcurrencyLimits()
	if len(got) != len(want) {
		t.Fatalf("len(normalized) = %d, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("default %q = %d, want %d", k, got[k], v)
		}
	}
}

func TestNormalizeLaneConcurrencyLimits_ClampsAndDropsUnknownKeys(t *testing.T) {
	in := map[string]int{
		"investigate": 0,    // <= 0 → default
		"execute":     -3,   // negative → default
		"review":      999,  // over max → clamped to 50
		"reconcile":   1,    // valid pass-through
		"unknown":     1234, // dropped (not a canonical lane)
	}
	got := normalizeLaneConcurrencyLimits(in)

	defaults := defaultLaneConcurrencyLimits()
	if got["investigate"] != defaults["investigate"] {
		t.Errorf("investigate <= 0 should fall back to default %d, got %d", defaults["investigate"], got["investigate"])
	}
	if got["execute"] != defaults["execute"] {
		t.Errorf("execute negative should fall back to default %d, got %d", defaults["execute"], got["execute"])
	}
	if got["review"] != 50 {
		t.Errorf("review over max should clamp to 50, got %d", got["review"])
	}
	if got["reconcile"] != 1 {
		t.Errorf("reconcile valid should pass through, got %d", got["reconcile"])
	}
	if _, present := got["unknown"]; present {
		t.Errorf("unknown key should be dropped, found %d", got["unknown"])
	}
	if len(got) != 4 {
		t.Errorf("expected exactly 4 canonical keys after normalize, got %d", len(got))
	}
}

func TestNormalizeFloatFields(t *testing.T) {
	type floatCase struct {
		name  string
		input float64
		want  float64
	}

	fields := []struct {
		field string
		cases []floatCase
		build func(float64) Settings
		get   func(Settings) float64
	}{
		{
			field: "ExecutionCostCapPerRun",
			cases: []floatCase{
				{"negative clamped to 0", -5.0, 0.0},
				{"zero allowed (unlimited)", 0.0, 0.0},
				{"positive value preserved", 10.0, 10.0},
			},
			build: func(v float64) Settings { return Settings{ExecutionCostCapPerRun: v} },
			get:   func(s Settings) float64 { return s.ExecutionCostCapPerRun },
		},
		{
			field: "CostPerTurnEstimate",
			cases: []floatCase{
				{"negative clamped to 0", -1.0, 0.0},
				{"zero allowed", 0.0, 0.0},
				{"mid range", 0.10, 0.10},
				{"max boundary", 5.0, 5.0},
				{"over max clamped to 5", 10.0, 5.0},
			},
			build: func(v float64) Settings { return Settings{CostPerTurnEstimate: v} },
			get:   func(s Settings) float64 { return s.CostPerTurnEstimate },
		},
	}

	for _, f := range fields {
		t.Run(f.field, func(t *testing.T) {
			for _, tc := range f.cases {
				t.Run(tc.name, func(t *testing.T) {
					got := f.get(normalizeSettings(f.build(tc.input)))
					if got != tc.want {
						t.Fatalf("%s: got %f, want %f", f.field, got, tc.want)
					}
				})
			}
		})
	}
}
