package policy

import (
	"testing"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	internalpolicy "lighting-control/internal/policy"
)

func TestDesiredForZoneUsesOffModeWithoutClaimingFallback(t *testing.T) {
	effective := &policyv1.EffectivePolicy{Preset: policyv1.PresetKind_PRESET_KIND_OFF}
	desired, fallback := desiredForZone("controller-1", &controlv1.ZoneCapability{ZoneId: "zone-1"}, internalpolicy.DefaultConfig(), effective)
	if _, ok := desired.DesiredMode.(*controlv1.ZoneState_Off); !ok {
		t.Fatalf("desired mode = %T, want explicit Off", desired.DesiredMode)
	}
	if fallback {
		t.Fatal("Off must not be reported as a static fallback")
	}
	if desired.ControllerId != "controller-1" || desired.ZoneId != "zone-1" {
		t.Fatalf("zone identity = %s/%s", desired.ControllerId, desired.ZoneId)
	}
}

func TestDesiredForZoneFallsBackWhenEffectIsNotAdvertised(t *testing.T) {
	config := internalpolicy.DefaultConfig()
	effective := &policyv1.EffectivePolicy{
		Preset:            policyv1.PresetKind_PRESET_KIND_COMBINED,
		Color:             &controlv1.Color{Red: 1, Green: 2, Blue: 3},
		BrightnessPercent: 25,
		EffectId:          "Breathing",
	}
	desired, fallback := desiredForZone("c", &controlv1.ZoneCapability{ZoneId: "z", SupportedEffects: []string{"Spectrum"}}, config, effective)
	static, ok := desired.DesiredMode.(*controlv1.ZoneState_StaticColor)
	if !ok {
		t.Fatalf("desired mode = %T, want static fallback", desired.DesiredMode)
	}
	if !fallback {
		t.Fatal("unsupported effect must be marked as static fallback")
	}
	if static.StaticColor.Color.Red != 1 || static.StaticColor.BrightnessPercent != 25 {
		t.Fatalf("fallback did not preserve configured effective color/brightness: %+v", static.StaticColor)
	}
}

func TestDesiredForZoneUsesConfiguredStaticFallbackForStaleEvidence(t *testing.T) {
	config := internalpolicy.DefaultConfig()
	effective := &policyv1.EffectivePolicy{Preset: policyv1.PresetKind_PRESET_KIND_HEALTH, StaticFallback: true}
	desired, fallback := desiredForZone("c", &controlv1.ZoneCapability{ZoneId: "z"}, config, effective)
	static, ok := desired.DesiredMode.(*controlv1.ZoneState_StaticColor)
	if !ok || !fallback {
		t.Fatalf("desired=%T fallback=%t; want explicit static fallback", desired.DesiredMode, fallback)
	}
	if static.StaticColor.Color.Red != config.StaleFallbackColor.Red || static.StaticColor.BrightnessPercent != config.StaleFallbackBrightnessPercent {
		t.Fatalf("fallback = %+v, want configured %+v at %d%%", static.StaticColor, config.StaleFallbackColor, config.StaleFallbackBrightnessPercent)
	}
}

func TestDesiredForZoneOnlyEmitsEffectWhenCapabilityAdvertisesIt(t *testing.T) {
	effective := &policyv1.EffectivePolicy{Preset: policyv1.PresetKind_PRESET_KIND_COMBINED, Color: &controlv1.Color{Green: 2}, BrightnessPercent: 12, EffectId: "Breathing"}
	desired, fallback := desiredForZone("c", &controlv1.ZoneCapability{ZoneId: "z", SupportedEffects: []string{"breathing"}}, internalpolicy.DefaultConfig(), effective)
	if _, ok := desired.DesiredMode.(*controlv1.ZoneState_Effect); !ok || fallback {
		t.Fatalf("desired mode=%T fallback=%t; advertised effect should remain an effect", desired.DesiredMode, fallback)
	}
}
