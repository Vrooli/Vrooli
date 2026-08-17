package binaryfetch

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAcquisitionResolveUsesOrderedPredicatesAndFallback(t *testing.T) {
	acquisition := Acquisition{Targets: []AcquisitionTarget{
		{When: map[string]string{"os": "linux", "gpu.cuda_compute": ">=8.9"}, URL: "gpu"},
		{When: map[string]string{"os": "linux"}, URL: "cpu"},
	}}
	target, err := acquisition.Resolve(Facts{"os": "linux", "gpu.cuda_compute": "8.0"})
	if err != nil || target.URL != "cpu" {
		t.Fatalf("Resolve fallback = %#v, %v", target, err)
	}
	target, err = acquisition.Resolve(Facts{"os": "linux", "gpu.cuda_compute": "8.9"})
	if err != nil || target.URL != "gpu" {
		t.Fatalf("Resolve preferred = %#v, %v", target, err)
	}
}

func TestAcquisitionResolveAbsentFactDoesNotMatch(t *testing.T) {
	acquisition := Acquisition{Targets: []AcquisitionTarget{
		{When: map[string]string{"gpu.cuda_compute": ">=8.0"}, URL: "gpu"},
		{URL: "portable"},
	}}
	target, err := acquisition.Resolve(Facts{})
	if err != nil || target.URL != "portable" {
		t.Fatalf("Resolve absent fact = %#v, %v", target, err)
	}
}

func TestAcquisitionResolveReportsUnsupportedTarget(t *testing.T) {
	_, err := (Acquisition{Targets: []AcquisitionTarget{{When: map[string]string{"os": "windows"}, Unsupported: "no upstream build"}}}).Resolve(Facts{"os": "windows"})
	var unsupported *UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Reason != "no upstream build" {
		t.Fatalf("Resolve error = %v", err)
	}
}

func TestAcquisitionValidateRejectsAmbiguousDeclarations(t *testing.T) {
	validDigest := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	valid := Acquisition{Kind: "url", Targets: []AcquisitionTarget{{URL: "https://example.test/tool", SHA256: validDigest}}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid acquisition rejected: %v", err)
	}
	withArtifactDigest := valid
	withArtifactDigest.Targets = []AcquisitionTarget{{URL: "https://example.test/tool", SHA256: validDigest, ArtifactSHA256: validDigest}}
	if err := withArtifactDigest.Validate(); err != nil {
		t.Fatalf("artifact digest declaration rejected: %v", err)
	}
	invalidArtifactDigest := withArtifactDigest
	invalidArtifactDigest.Targets[0].ArtifactSHA256 = "not-a-digest"
	if err := invalidArtifactDigest.Validate(); err == nil {
		t.Fatal("invalid artifact digest declaration accepted")
	}
	for name, acquisition := range map[string]Acquisition{
		"missing-kind":              {Targets: valid.Targets},
		"missing-digest":            {Kind: "url", Targets: []AcquisitionTarget{{URL: "https://example.test/tool"}}},
		"unsupported-with-artifact": {Kind: "url", Targets: []AcquisitionTarget{{URL: "https://example.test/tool", Unsupported: "not published"}}},
		"none-without-unsupported":  {Kind: "none", Targets: []AcquisitionTarget{{}}},
	} {
		if err := acquisition.Validate(); err == nil {
			t.Errorf("%s: Validate succeeded", name)
		}
	}
}

func TestAcquisitionExplainNamesRejectedCandidates(t *testing.T) {
	explanation := (Acquisition{Targets: []AcquisitionTarget{
		{When: map[string]string{"os": "linux", "arch": "arm64"}, URL: "arm"},
		{When: map[string]string{"os": "linux"}, URL: "amd"},
	}}).Explain(Facts{"os": "linux", "arch": "amd64"})
	if explanation.Selected != 1 || explanation.Candidates[0].Reason == "" || !explanation.Candidates[1].Selected {
		t.Fatalf("unexpected explanation: %#v", explanation)
	}
}

func TestAcquisitionSchemaDrift(t *testing.T) {
	root := filepath.Join("..", "..")
	if err := ValidateAcquisitionSchema(root); err != nil {
		t.Fatal(err)
	}
}
