package studio

import (
	"strings"
	"testing"
)

func newStudio() *Studio {
	return &Studio{
		Identities:      map[string]Identity{},
		Specs:           map[string]Spec{},
		Renders:         map[string]*Render{},
		Assets:          map[string]*Asset{},
		ImportHashes:    map[string]string{},
		CampaignBudgets: map[string]*CampaignBudget{},
	}
}

func validIngest() IngestRequest {
	return IngestRequest{
		BlobKey:   "external/one",
		MediaType: "image/png",
		AltText:   "A blue duotone halftone of a colonnade above a bay",
		Width:     1440,
		Height:    720,
		Provenance: ExternalProvenance{
			ProducingScenario: "backdrop-studio",
			Strategy:          "procedural-treated",
		},
	}
}

// TestIngestAcceptsANonCharacterConditioningKind is the conformance test
// PROBLEMS.md asked for.
//
// The recorded worry was that this scenario's composition path is built around
// binding identity records — characters, products, scenes — into a prompt, and
// that a producer conditioning on something else would not fit. Backdrop Studio
// binds a scaffold and a palette. This proves the ingress does not require an
// identity and records the scaffold as what steered generation.
func TestIngestAcceptsANonCharacterConditioningKind(t *testing.T) {
	s := newStudio()
	req := validIngest()
	req.Provenance = ExternalProvenance{
		ProducingScenario: "backdrop-studio",
		Strategy:          "guided",
		ModelBacked:       true,
		Model:             "sd-1.5/local-gpu",
		Prompt:            "sunlit modernist interior, tall windows, long shadows",
		Seed:              "7",
		Conditioning:      ConditioningReference{Kind: "scaffold", ID: "arcade", Version: "edge"},
	}
	asset, err := s.Ingest("a1", req)
	if err != nil {
		t.Fatalf("ingest with a scaffold conditioning kind was refused: %v", err)
	}
	if len(asset.IdentityVersionIDs) != 0 {
		t.Fatalf("ingest invented identity bindings: %v", asset.IdentityVersionIDs)
	}
	if !strings.Contains(asset.Disclosure, "scaffold arcade@edge") {
		t.Fatalf("the disclosure does not record what conditioned the render: %q", asset.Disclosure)
	}
	if !asset.AIgenerated {
		t.Fatal("a model-backed asset must be marked AI-generated")
	}
}

// TestModelBackedIngestWithoutModelOrPromptIsRefused is the rule that makes the
// disclosure worth having. A synthetic image whose model and prompt are unknown
// cannot be reproduced or audited, so it is refused at the door rather than
// recorded with a gap that reads as complete.
func TestModelBackedIngestWithoutModelOrPromptIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*ExternalProvenance)
		want string
	}{
		{"no model", func(p *ExternalProvenance) { p.Model = "" }, "model"},
		{"no prompt", func(p *ExternalProvenance) { p.Prompt = "" }, "prompt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStudio()
			req := validIngest()
			req.Provenance = ExternalProvenance{
				ProducingScenario: "backdrop-studio",
				Strategy:          "synthesized",
				ModelBacked:       true,
				Model:             "sd-1.5/local-gpu",
				Prompt:            "art nouveau celestial chart",
			}
			tc.edit(&req.Provenance)
			if _, err := s.Ingest("a1", req); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected refusal naming %q, got %v", tc.want, err)
			}
			if len(s.Assets) != 0 {
				t.Fatal("a refused ingest left a record behind")
			}
		})
	}
}

// TestIngestedAssetIsNotReleased proves the door is a way in rather than a way
// around: an ingested asset lands in review and still runs the release checks.
func TestIngestedAssetIsNotReleased(t *testing.T) {
	s := newStudio()
	asset, err := s.Ingest("a1", validIngest())
	if err != nil {
		t.Fatal(err)
	}
	if asset.Status != InReview {
		t.Fatalf("ingested asset status is %q, want %q", asset.Status, InReview)
	}
	if err := s.Release("a1"); err != nil {
		t.Fatalf("a procedural ingest with alt text and a disclosure should release: %v", err)
	}
	if s.Assets["a1"].Status != Released {
		t.Fatal("release did not transition the asset")
	}
}

// TestIngestRequiresAltTextOrADecorativeClaim moves the accessibility failure to
// the door, where the caller still has the alt text to supply.
func TestIngestRequiresAltTextOrADecorativeClaim(t *testing.T) {
	s := newStudio()
	req := validIngest()
	req.AltText = ""
	if _, err := s.Ingest("a1", req); err == nil {
		t.Fatal("ingest accepted an asset with neither alt text nor a decorative claim")
	}
	req.Decorative = true
	if _, err := s.Ingest("a1", req); err != nil {
		t.Fatalf("a decorative asset with no alt text must be accepted: %v", err)
	}
}

// TestNonModelBackedDisclosureNamesItsProducer keeps the record meaningful for
// the procedural lane too: "how was this made" has an answer either way.
func TestNonModelBackedDisclosureNamesItsProducer(t *testing.T) {
	s := newStudio()
	asset, err := s.Ingest("a1", validIngest())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(asset.Disclosure, "backdrop-studio") {
		t.Fatalf("disclosure does not name the producer: %q", asset.Disclosure)
	}
	if !strings.Contains(asset.Disclosure, "no generative model") {
		t.Fatalf("disclosure does not state that no model was involved: %q", asset.Disclosure)
	}
	if asset.AIgenerated {
		t.Fatal("a procedural asset must not be marked AI-generated")
	}
}
