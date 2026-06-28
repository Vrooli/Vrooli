package adapters

import (
	"testing"

	"image-tools/internal/models"
	"image-tools/internal/safety"
)

// readyAdapter is a test helper producing a Ready, installed-by-default adapter
// (the seed ships everything Ready=false, so the happy path is constructed here).
func readyAdapter(id string, kind Kind, arch models.Architecture, weight safety.Weight) Adapter {
	a := Adapter{
		ID: id, Name: id, Kind: kind, Architecture: arch, Weight: weight,
		ScaleRange: ScaleRange{Min: 0, Max: 2, Default: 1}, Ready: true,
	}
	if kind == KindControlNet {
		a.Preprocessor = PreprocessorCanny
	}
	if kind == KindIPAdapter {
		a.ScaleRange = ScaleRange{Min: 0, Max: 1, Default: 0.6}
	}
	return a
}

func lookup(as ...Adapter) func(string) (Adapter, bool) {
	m := make(map[string]Adapter, len(as))
	for _, a := range as {
		m[a.ID] = a
	}
	return func(id string) (Adapter, bool) { a, ok := m[id]; return a, ok }
}

func alwaysInstalled(string) bool { return true }

func TestResolveConditioningHappyPathOrdersAndClamps(t *testing.T) {
	lora := readyAdapter("lora1", KindLoRA, models.ArchSD15, safety.WeightNone)
	cn := readyAdapter("cn1", KindControlNet, models.ArchSD15, safety.WeightNone)
	ipa := readyAdapter("ipa1", KindIPAdapter, models.ArchSD15, safety.WeightHigh)
	byID := lookup(lora, cn, ipa)

	// Request out of canonical order; resolver must reorder LoRA→ControlNet→IP-Adapter.
	reqs := []AdapterRequest{
		{ID: "ipa1", Scale: 5, ConditioningImageKey: "ref.png"}, // scale clamps to 1.0 (ip max)
		{ID: "cn1", ConditioningImageKey: "edges.png"},
		{ID: "lora1", Scale: 0}, // 0 → default 1.0
	}
	resolved, _, err := ResolveConditioning(models.ArchSD15, reqs, byID, nil, alwaysInstalled)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 3 {
		t.Fatalf("expected 3 resolved, got %d", len(resolved))
	}
	if resolved[0].Kind != KindLoRA || resolved[1].Kind != KindControlNet || resolved[2].Kind != KindIPAdapter {
		t.Fatalf("wrong order: %v %v %v", resolved[0].Kind, resolved[1].Kind, resolved[2].Kind)
	}
	if resolved[0].Scale != 1.0 {
		t.Fatalf("lora default scale: got %v want 1.0", resolved[0].Scale)
	}
	if resolved[2].Scale != 1.0 {
		t.Fatalf("ip-adapter scale should clamp to 1.0, got %v", resolved[2].Scale)
	}
}

func TestResolveConditioningWeightElevation(t *testing.T) {
	ipa := readyAdapter("ipa1", KindIPAdapter, models.ArchSD15, safety.WeightHigh)
	resolved, _, err := ResolveConditioning(models.ArchSD15,
		[]AdapterRequest{{ID: "ipa1", ConditioningImageKey: "ref.png"}},
		lookup(ipa), nil, alwaysInstalled)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// A none-weight op elevated by a high-weight IP-Adapter ⇒ high (decision C4).
	if got := EffectiveWeight(safety.WeightNone, resolved); got != safety.WeightHigh {
		t.Fatalf("effective weight=%q want high", got)
	}
}

func TestResolveConditioningFailsClosed(t *testing.T) {
	ready := readyAdapter("ok", KindLoRA, models.ArchSD15, safety.WeightNone)
	notReady := Adapter{ID: "nr", Name: "nr", Kind: KindLoRA, Architecture: models.ArchSD15, Weight: safety.WeightNone, ScaleRange: ScaleRange{Min: 0, Max: 2, Default: 1}, Ready: false, Pending: "phase 4"}
	wrongArch := readyAdapter("xarch", KindLoRA, models.ArchSDXL, safety.WeightNone)
	ipaNoImg := readyAdapter("ipa", KindIPAdapter, models.ArchSD15, safety.WeightHigh)
	byID := lookup(ready, notReady, wrongArch, ipaNoImg)

	cases := []struct {
		name      string
		reqs      []AdapterRequest
		enabled   func(string) bool
		installed func(string) bool
	}{
		{"unknown id", []AdapterRequest{{ID: "missing"}}, nil, alwaysInstalled},
		{"wrong architecture", []AdapterRequest{{ID: "xarch"}}, nil, alwaysInstalled},
		{"not ready", []AdapterRequest{{ID: "nr"}}, nil, alwaysInstalled},
		{"disabled", []AdapterRequest{{ID: "ok"}}, func(string) bool { return false }, alwaysInstalled},
		{"not installed", []AdapterRequest{{ID: "ok"}}, nil, func(string) bool { return false }},
		{"ip-adapter without reference", []AdapterRequest{{ID: "ipa"}}, nil, alwaysInstalled},
		{"duplicate", []AdapterRequest{{ID: "ok"}, {ID: "ok"}}, nil, alwaysInstalled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := ResolveConditioning(models.ArchSD15, tc.reqs, byID, tc.enabled, tc.installed); err == nil {
				t.Fatalf("expected fail-closed for %s", tc.name)
			}
		})
	}
}

func TestResolveConditioningControlNetAutoPreprocessWarns(t *testing.T) {
	cn := readyAdapter("cn1", KindControlNet, models.ArchSD15, safety.WeightNone) // preprocessor canny
	resolved, warnings, err := ResolveConditioning(models.ArchSD15,
		[]AdapterRequest{{ID: "cn1"}}, // no conditioning image → auto-derive via canny
		lookup(cn), nil, alwaysInstalled)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 1 || resolved[0].Preprocessor != PreprocessorCanny {
		t.Fatalf("expected canny preprocessor, got %+v", resolved)
	}
	if len(warnings) == 0 {
		t.Fatal("expected an auto-preprocess warning")
	}
}

func TestResolveConditioningNilCatalogRejects(t *testing.T) {
	if _, _, err := ResolveConditioning(models.ArchSD15, []AdapterRequest{{ID: "x"}}, nil, nil, alwaysInstalled); err == nil {
		t.Fatal("expected rejection when no catalog lookup is supplied")
	}
}
