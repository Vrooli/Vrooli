package adapters

import (
	"crypto/sha256"
	"fmt"
	"strings"
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
		// No-vaporware is Ready⇔Pending consistency: an unproven adapter ships
		// Ready=false WITH a pending reason; a proven one ships Ready=true with NO
		// pending reason (flipped only after an attended GPU run — TESTING.md).
		if a.Ready && a.Pending != "" {
			t.Fatalf("Ready adapter %q still declares a pending reason %q", a.ID, a.Pending)
		}
		if !a.Ready && a.Pending == "" {
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

func TestOpenPosePinAndLicenseReceiptAreComplete(t *testing.T) {
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	adapter, ok := registry.ByID("controlnet-openpose-sd15")
	if !ok {
		t.Fatal("OpenPose ControlNet is missing from the catalog")
	}
	if adapter.Enabled || adapter.Ready || adapter.LicenseTermsVersion != OpenRAILMTermsVersion {
		t.Fatalf("adapter rights/readiness state is not fail-closed: %+v", adapter)
	}
	if adapter.Source.Repo.Revision != "df796456519d1ac5dbe674caa0653fcc9673bca8" {
		t.Fatalf("adapter revision = %q", adapter.Source.Repo.Revision)
	}
	if got := strings.Join(adapter.Source.Repo.AllowPatterns, ","); got != "config.json,diffusion_pytorch_model.safetensors" {
		t.Fatalf("repo allow patterns = %q", got)
	}
	terms, err := LicenseTerms(OpenRAILMTermsVersion)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(terms)); got != "be351ebe7ac01bcdbb018639aadcfd38f136b7dc3f2a3d4d3a24db51d1b210ef" {
		t.Fatalf("embedded OpenRAIL-M license hash = %s", got)
	}
	if !strings.Contains(string(terms), "Attachment A") || !strings.Contains(string(terms), "MUST be included as an enforceable provision") {
		t.Fatal("embedded license omits its use restrictions or enforceable pass-through requirement")
	}
}

func TestLargeAdaptersAcquireOnlyRuntimeSafetensors(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("load seed: %v", err)
	}
	want := map[string][]string{
		"ip-adapter-sd15": {
			"models/ip-adapter_sd15.safetensors",
			"models/image_encoder/config.json",
			"models/image_encoder/model.safetensors",
		},
		"controlnet-canny-sd15": {"config.json", "diffusion_pytorch_model.safetensors"},
	}
	for id, expected := range want {
		adapter, ok := r.ByID(id)
		if !ok || adapter.Source.Repo.RepoID == "" {
			t.Fatalf("adapter %q has no pinned repository source", id)
		}
		got := adapter.Source.Repo.AllowPatterns
		if len(got) != len(expected) {
			t.Fatalf("adapter %q allow patterns = %v, want %v", id, got, expected)
		}
		for _, pattern := range expected {
			found := false
			for _, actual := range got {
				if actual == pattern {
					found = true
				}
			}
			if !found {
				t.Errorf("adapter %q does not fetch required runtime file %q: %v", id, pattern, got)
			}
		}
		for _, pattern := range got {
			if strings.HasSuffix(pattern, ".bin") || strings.HasSuffix(pattern, "/*") {
				t.Errorf("adapter %q acquisition includes duplicate/ambiguous format %q", id, pattern)
			}
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
		"bad kind":                 `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"nope","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"arch none":                `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"none","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"preprocessor on lora":     `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","preprocessor":"canny","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"bad scale range":          `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":2,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"default out of range":     `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"bad weight":               `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"extreme","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
		"combined source checksum": `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"},"source":{"assets":[{"url":"https://x/f","filename":"f","kind":"safetensors","min_bytes":1}],"repo":{"repo_id":"r","revision":"abcdef"},"checksum":{"algo":"sha256","value":"abcd","status":"pinned"}}}]}`,
		"duplicate id":             `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}},{"id":"x","name":"Y","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"}}]}`,
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
