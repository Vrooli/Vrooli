package adapters

import (
	"testing"

	"image-tools/internal/models"
	"image-tools/internal/safety"
)

func TestSeedLoadsAndUpholdsInvariants(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("load seed: %v", err)
	}
	if len(r.Adapters()) == 0 {
		t.Fatal("seed has no adapters")
	}
	for _, a := range r.Adapters() {
		if a.Ready {
			t.Fatalf("seed adapter %q ships Ready=true (no vaporware)", a.ID)
		}
		if a.Pending == "" {
			t.Fatalf("not-ready adapter %q has no pending reason", a.ID)
		}
		if a.CapabilityLabels.CommercialUse == models.CommercialUseConditional && a.Enabled {
			t.Fatalf("conditional-commercial adapter %q must not be enabled", a.ID)
		}
		if !a.Architecture.Valid() || a.Architecture == models.ArchNone {
			t.Fatalf("adapter %q has non-concrete architecture %q", a.ID, a.Architecture)
		}
	}
}

func TestSeedLicenseDiscipline(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("load seed: %v", err)
	}
	for _, a := range r.Adapters() {
		if a.CapabilityLabels.CommercialUse == models.CommercialUseNo {
			t.Fatalf("adapter %q is commercial_use=no (commercial-clean gate)", a.ID)
		}
		if a.CapabilityLabels.License == "" {
			t.Fatalf("adapter %q declares no license", a.ID)
		}
	}
}

func TestByKindAndCompatible(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(r.ByKind(KindControlNet)) == 0 {
		t.Fatal("expected controlnet adapters in seed")
	}
	if len(r.ByKind(KindLoRA)) == 0 {
		t.Fatal("expected lora adapters in seed")
	}
	sd15 := r.Compatible(models.ArchSD15)
	if len(sd15) == 0 {
		t.Fatal("expected sd15-compatible adapters")
	}
	for _, a := range sd15 {
		if a.Architecture != models.ArchSD15 {
			t.Fatalf("Compatible(sd15) returned %q with arch %q", a.ID, a.Architecture)
		}
		if a.CompatibleWith(models.ArchSDXL) {
			t.Fatalf("adapter %q should not be compatible with sdxl", a.ID)
		}
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	cases := map[string]string{
		"bad kind":             `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"nope","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"arch none":            `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"none","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"preprocessor on lora": `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","preprocessor":"canny","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"bad scale range":      `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":2,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"default out of range": `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"bad weight":           `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"extreme","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"both assets and repo": `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"},"source":{"assets":[{"url":"u","filename":"f","kind":"safetensors","min_bytes":1}],"repo":{"repo_id":"r"}}}]}`,
		"duplicate id":         `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}},{"id":"x","name":"Y","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(doc)); err == nil {
				t.Fatalf("expected Parse to reject %s", name)
			}
		})
	}
}

func TestScaleRangeClamp(t *testing.T) {
	r := ScaleRange{Min: 0, Max: 1, Default: 0.6}
	if got := r.Clamp(2); got != 1 {
		t.Fatalf("clamp above max: got %v want 1", got)
	}
	if got := r.Clamp(-1); got != 0 {
		t.Fatalf("clamp below min: got %v want 0", got)
	}
	if got := r.Clamp(0.5); got != 0.5 {
		t.Fatalf("clamp in range: got %v want 0.5", got)
	}
	// A zero range is treated as no-clamp.
	if got := (ScaleRange{}).Clamp(42); got != 42 {
		t.Fatalf("zero range should not clamp: got %v want 42", got)
	}
}

func TestRequiresReferenceImage(t *testing.T) {
	if (Adapter{Kind: KindLoRA}).RequiresReferenceImage() {
		t.Fatal("lora must not require a reference image")
	}
	if !(Adapter{Kind: KindIPAdapter}).RequiresReferenceImage() {
		t.Fatal("ip-adapter must require a reference image")
	}
	if !(Adapter{Kind: KindControlNet}).RequiresReferenceImage() {
		t.Fatal("controlnet must require a conditioning image")
	}
}

func TestMaxWeightElevation(t *testing.T) {
	// Decision C4: effective weight = max(op, adapters...).
	if got := safety.MaxWeight(safety.WeightNone, safety.WeightHigh); got != safety.WeightHigh {
		t.Fatalf("max(none,high)=%q want high", got)
	}
	if got := safety.MaxWeight(safety.WeightLow, safety.WeightNone); got != safety.WeightLow {
		t.Fatalf("max(low,none)=%q want low", got)
	}
	if got := safety.MaxWeight(); got != safety.WeightNone {
		t.Fatalf("max()=%q want none", got)
	}
}
