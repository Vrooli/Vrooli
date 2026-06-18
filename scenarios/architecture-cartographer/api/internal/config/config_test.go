package config

import (
	"reflect"
	"testing"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad_DefaultsWhenEmpty(t *testing.T) {
	cfg, diags := Load(envFrom(nil))
	if len(diags) != 0 {
		t.Fatalf("no env → no diagnostics, got %+v", diags)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("empty env must reproduce defaults; got %+v", cfg)
	}
}

func TestLoad_ValidOverrides(t *testing.T) {
	cfg, diags := Load(envFrom(map[string]string{
		EnvGodDomainFanOut:     "0.75",
		EnvAutoPlaceMin:        "0.9",
		EnvSuggestMin:          "0.6",
		EnvQuorumHigh:          "0.5",
		EnvQuorumLow:           "0.35",
		EnvArchetypeExemptions: "composition-root, infrastructure, glue",
		EnvNonDomainFolders:    "recipes,embeddings",
		EnvLadderOrder:         "api_folders,domains_doc",
		EnvBannedVocab:         "bucket,drawer",
		EnvLayeringStrict:      "false",
	}))
	if len(diags) != 0 {
		t.Fatalf("valid overrides → no diagnostics, got %+v", diags)
	}
	if cfg.GodDomainFanOut != 0.75 || cfg.AutoPlaceMin != 0.9 || cfg.SuggestMin != 0.6 {
		t.Fatalf("float overrides not applied: %+v", cfg)
	}
	if cfg.QuorumHigh != 0.5 || cfg.QuorumLow != 0.35 {
		t.Fatalf("quorum overrides not applied: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.ArchetypeExemptions, []string{"composition-root", "infrastructure", "glue"}) {
		t.Fatalf("archetypes = %v", cfg.ArchetypeExemptions)
	}
	if !reflect.DeepEqual(cfg.ExtraNonDomainFolders, []string{"recipes", "embeddings"}) {
		t.Fatalf("non-domain folders = %v", cfg.ExtraNonDomainFolders)
	}
	if !reflect.DeepEqual(cfg.LadderOrder, []string{"api_folders", "domains_doc"}) {
		t.Fatalf("ladder order = %v", cfg.LadderOrder)
	}
	if !reflect.DeepEqual(cfg.BannedVocabulary, []string{"bucket", "drawer"}) {
		t.Fatalf("banned vocab = %v", cfg.BannedVocabulary)
	}
	if cfg.LayeringStrict {
		t.Fatal("layering strict override not applied")
	}
}

func TestLoad_ClampsOutOfRange(t *testing.T) {
	cfg, diags := Load(envFrom(map[string]string{
		EnvInstabilityWarnBand: "1.5",
		EnvSuggestMin:          "-0.2",
	}))
	if cfg.InstabilityWarnBand != 1.0 {
		t.Fatalf("instability should clamp to 1.0, got %v", cfg.InstabilityWarnBand)
	}
	if cfg.SuggestMin != 0.0 {
		t.Fatalf("suggest min should clamp to 0.0, got %v", cfg.SuggestMin)
	}
	if len(diags) < 2 {
		t.Fatalf("expected clamp diagnostics, got %+v", diags)
	}
}

func TestLoad_RejectsExclusiveLowerBound(t *testing.T) {
	// god-domain fan-out must be > 0; 0 is rejected to the default.
	cfg, diags := Load(envFrom(map[string]string{EnvGodDomainFanOut: "0"}))
	if cfg.GodDomainFanOut != Default().GodDomainFanOut {
		t.Fatalf("fan-out 0 must revert to default, got %v", cfg.GodDomainFanOut)
	}
	if len(diags) == 0 {
		t.Fatal("expected a diagnostic for fan-out=0")
	}
}

func TestLoad_UnparseableRevertsToDefault(t *testing.T) {
	cfg, diags := Load(envFrom(map[string]string{EnvTieDelta: "notanumber"}))
	if cfg.TieDelta != Default().TieDelta {
		t.Fatalf("unparseable tie delta must revert to default, got %v", cfg.TieDelta)
	}
	if len(diags) == 0 {
		t.Fatal("expected a diagnostic for unparseable value")
	}
}

func TestLoad_AutoPlaceBelowSuggestRaised(t *testing.T) {
	cfg, diags := Load(envFrom(map[string]string{EnvAutoPlaceMin: "0.4", EnvSuggestMin: "0.6"}))
	if cfg.AutoPlaceMin < cfg.SuggestMin {
		t.Fatalf("auto_place must be raised to >= suggest, got %+v", cfg)
	}
	if len(diags) == 0 {
		t.Fatal("expected a diagnostic for auto_place < suggest")
	}
}

func TestLoad_QuorumHighBelowLowRaised(t *testing.T) {
	cfg, diags := Load(envFrom(map[string]string{EnvQuorumHigh: "0.2", EnvQuorumLow: "0.4"}))
	if cfg.QuorumHigh < cfg.QuorumLow {
		t.Fatalf("quorum_high must be raised to >= quorum_low, got %+v", cfg)
	}
	if len(diags) == 0 {
		t.Fatal("expected a diagnostic for quorum_high < quorum_low")
	}
}

func TestLoad_InvalidLadderSourceIgnored(t *testing.T) {
	cfg, diags := Load(envFrom(map[string]string{EnvLadderOrder: "bogus,domains_doc"}))
	if !reflect.DeepEqual(cfg.LadderOrder, []string{"domains_doc"}) {
		t.Fatalf("invalid source must be dropped, got %v", cfg.LadderOrder)
	}
	if len(diags) == 0 {
		t.Fatal("expected a diagnostic for the unknown source")
	}
}

func TestLoad_AllInvalidLadderKeepsDefault(t *testing.T) {
	cfg, _ := Load(envFrom(map[string]string{EnvLadderOrder: "nope,nada"}))
	if !reflect.DeepEqual(cfg.LadderOrder, Default().LadderOrder) {
		t.Fatalf("all-invalid ladder must keep default, got %v", cfg.LadderOrder)
	}
}

func TestLoad_InvalidBoolRevertsToDefault(t *testing.T) {
	cfg, diags := Load(envFrom(map[string]string{EnvLayeringStrict: "maybe"}))
	if cfg.LayeringStrict != Default().LayeringStrict {
		t.Fatalf("invalid bool must keep default, got %v", cfg.LayeringStrict)
	}
	if len(diags) == 0 {
		t.Fatal("expected a diagnostic for invalid bool")
	}
}
