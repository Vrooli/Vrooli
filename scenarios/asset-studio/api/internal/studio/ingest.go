package studio

import (
	"errors"
	"fmt"
	"strings"
)

// External byte ingress.
//
// Every other asset in this store was produced by a render this scenario
// dispatched, so its provenance was known here by construction. An asset whose
// bytes arrive from outside inverts that: the producer knows how the image was
// made, and Asset Studio knows what a releasable record requires. The split
// this file encodes is that the caller supplies the facts and Asset Studio owns
// the record — because a producer that writes its own disclosure is a second
// authority on disclosure, and two authorities on a compliance record is how
// one of them ends up wrong.
//
// It exists because Backdrop Studio had no third option. Its model-backed
// styles could either be released with a fabricated or absent disclosure, or
// not released at all; it chose not at all, which was correct and also meant a
// working capability shipped disabled.

// ExternalProvenance is what a producing scenario knows about its own bytes.
//
// It carries no identity record. The composition path elsewhere in this
// scenario is built around binding identities — characters, products, scenes —
// into a prompt, and requiring one here would have made the door usable only by
// producers that already fit that model. Backdrop Studio binds a scaffold and a
// palette; neither is an identity, and neither should have to pretend to be.
type ExternalProvenance struct {
	ProducingScenario string
	Strategy          string
	ModelBacked       bool
	Model             string
	Prompt            string
	NegativePrompt    string
	Seed              string
	Conditioning      ConditioningReference
	Parameters        string
}

// IngestRequest is one external asset arriving with its provenance.
type IngestRequest struct {
	BlobKey       string
	MediaType     string
	AltText       string
	Decorative    bool
	Width, Height int
	Provenance    ExternalProvenance
}

// Ingest records an externally produced asset and returns it in `in_review`.
//
// The status is deliberate. An ingested asset is not released: it still needs
// the operator verdict and the release checks every other asset needs, and a
// door that admitted bytes straight to `released` would be a way around them
// rather than a way in. `in_review` is the state a selected candidate reaches,
// which is exactly what an ingested asset is — a candidate somebody already
// chose.
func (s *Studio) Ingest(id string, r IngestRequest) (*Asset, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("ingest requires an asset id")
	}
	if strings.TrimSpace(r.BlobKey) == "" {
		return nil, errors.New("ingest requires stored bytes")
	}
	if strings.TrimSpace(r.MediaType) == "" {
		return nil, errors.New("ingest requires a media type")
	}
	if r.Width <= 0 || r.Height <= 0 {
		return nil, errors.New("ingest requires positive pixel dimensions")
	}
	if strings.TrimSpace(r.Provenance.ProducingScenario) == "" {
		return nil, errors.New("ingest requires the producing scenario")
	}
	// Alt text or an explicit decorative claim, checked here rather than only at
	// release: an asset admitted without either is a record that cannot ever be
	// released, and failing at the door names the missing field while the caller
	// still has it.
	if !r.Decorative && strings.TrimSpace(r.AltText) == "" {
		return nil, errors.New("ingest requires alt_text, or decorative=true")
	}
	disclosure, err := disclosureFor(r.Provenance)
	if err != nil {
		return nil, err
	}
	if _, exists := s.Assets[id]; exists {
		return nil, fmt.Errorf("asset %q already exists", id)
	}
	asset := &Asset{
		ID:                  id,
		BlobKey:             r.BlobKey,
		AltText:             r.AltText,
		Disclosure:          disclosure,
		Status:              InReview,
		AIgenerated:         r.Provenance.ModelBacked,
		MediaType:           r.MediaType,
		Width:               r.Width,
		Height:              r.Height,
		DerivationOperation: "external-ingest:" + r.Provenance.ProducingScenario,
	}
	s.Assets[id] = asset
	return asset, nil
}

// disclosureFor writes the disclosure record from the producer's facts.
//
// The refusal below is the load-bearing part of this whole path. Synthetic
// media released without the model and prompt that made it cannot be
// reproduced, audited, or honestly labelled — so a request that claims a model
// drew the image and then names no model is refused rather than recorded with a
// gap. A non-model-backed asset needs no such record and gets a short one
// naming its producer, because "how was this made" still has an answer.
func disclosureFor(p ExternalProvenance) (string, error) {
	if !p.ModelBacked {
		strategy := strings.TrimSpace(p.Strategy)
		if strategy == "" {
			strategy = "unspecified"
		}
		return fmt.Sprintf("Produced by %s (%s); no generative model involved.", p.ProducingScenario, strategy), nil
	}
	if strings.TrimSpace(p.Model) == "" {
		return "", errors.New("a model-backed asset must name the model that produced it")
	}
	if strings.TrimSpace(p.Prompt) == "" {
		return "", errors.New("a model-backed asset must carry the prompt that produced it")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "AI-generated by %s using model %s", p.ProducingScenario, p.Model)
	if strategy := strings.TrimSpace(p.Strategy); strategy != "" {
		fmt.Fprintf(&b, " (strategy %s)", strategy)
	}
	fmt.Fprintf(&b, ". Prompt: %s", p.Prompt)
	if negative := strings.TrimSpace(p.NegativePrompt); negative != "" {
		fmt.Fprintf(&b, ". Negative: %s", negative)
	}
	if seed := strings.TrimSpace(p.Seed); seed != "" {
		fmt.Fprintf(&b, ". Seed: %s", seed)
	}
	if kind := strings.TrimSpace(p.Conditioning.Kind); kind != "" {
		// Conditioning is recorded generically. It is the field that decides
		// whether this door is usable by a producer that conditions on
		// something other than an identity — a scaffold, a reference image set,
		// a trained adapter or a look — and narrowing it to identities would
		// close the door this file exists to open.
		fmt.Fprintf(&b, ". Conditioned on %s", kind)
		if id := strings.TrimSpace(p.Conditioning.ID); id != "" {
			fmt.Fprintf(&b, " %s", id)
		}
		if version := strings.TrimSpace(p.Conditioning.Version); version != "" {
			fmt.Fprintf(&b, "@%s", version)
		}
	}
	if params := strings.TrimSpace(p.Parameters); params != "" {
		fmt.Fprintf(&b, ". Parameters: %s", params)
	}
	return b.String() + ".", nil
}
