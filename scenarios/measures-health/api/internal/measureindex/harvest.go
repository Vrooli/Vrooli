// Package measureindex is the central measures index + the federated measures
// provider's brain wiring. It harvests every scenario's manifest `measure`
// blocks into scenario-attributed MeasureDeclarations (the corpus), matches a
// natural-language analytical question to the best measure, and drives the
// shared packages/measures-go Engine (resolve -> gate -> execute) to answer it.
//
// Layering: this package OWNS the corpus + the Matcher seam; the
// match/resolve/gate/execute brain lives in measures-go (NewEngine). The Matcher
// is intentionally a seam — today it is a deterministic lexical index over the
// curated questions[] (offline, infra-free, the graceful-degradation path); the
// plan's aisearch-go hybrid index (embed questions via measures.MeasureComposer,
// qdrant retrieval) drops into the SAME seam once a corpus exists (no scenario
// declares a measure until Phase 5/6, so a vector index would have nothing to
// retrieve yet — see DECISIONS.md).
package measureindex

import (
	"context"

	measures "github.com/vrooli/measures-go"
	"measures-health/internal/measurescan"
	"measures-health/internal/validation"
)

// ManifestSource reads a scenario's raw cli/manifest.json bytes. A scenario with
// no manifest yields (nil, nil) — it simply declares no measures. Satisfied
// structurally by validation.FilesystemManifestSource; tests inject bytes.
type ManifestSource interface {
	Manifest(scenario string) ([]byte, error)
}

// ScenarioLister lists the scenario ids the index spans. Satisfied structurally
// by validation.FilesystemScenarioLister; tests inject a fixed list.
type ScenarioLister interface {
	Scenarios() ([]string, error)
}

// Harvester assembles the central corpus: every scenario's manifest `measure`
// blocks, joined against the proto-derived param schema and attributed to their
// owning scenario. All I/O is behind the seams above so Harvest stays testable.
type Harvester struct {
	manifests ManifestSource
	scenarios ScenarioLister
	schema    measurescan.SchemaSource
}

// NewHarvester constructs a Harvester over its seams.
func NewHarvester(m ManifestSource, l ScenarioLister, s measurescan.SchemaSource) *Harvester {
	return &Harvester{manifests: m, scenarios: l, schema: s}
}

// NewFilesystemHarvester wires the production seams rooted at repoRoot, reusing
// the validation domain's filesystem seams (single source of "what is a
// scenario" + the manifest path) and the committed proto descriptor reader.
func NewFilesystemHarvester(repoRoot string) *Harvester {
	return NewHarvester(
		validation.FilesystemManifestSource{RepoRoot: repoRoot},
		validation.FilesystemScenarioLister{RepoRoot: repoRoot},
		measurescan.NewDescriptorSchemaReader(repoRoot),
	)
}

// Harvest enumerates every scenario's declared measures as assembled,
// scenario-attributed declarations. A scenario whose manifest is missing,
// unreadable, or malformed contributes nothing (the measures-health *validator*
// is the surface that raises those as findings — the index degrades silently so
// one bad manifest never sinks the whole corpus). A measure whose assembly drifts
// (a manifest param absent from the proto request) is likewise skipped here.
func (h *Harvester) Harvest(ctx context.Context) ([]measures.MeasureDeclaration, error) {
	scenarios, err := h.scenarios.Scenarios()
	if err != nil {
		return nil, err
	}
	out := make([]measures.MeasureDeclaration, 0, 32)
	for _, scenario := range scenarios {
		raw, err := h.manifests.Manifest(scenario)
		if err != nil || len(raw) == 0 {
			continue
		}
		mm, err := measurescan.Parse(raw)
		if err != nil {
			continue
		}
		for _, cm := range mm.Commands {
			decl, err := cm.Assemble(h.schema)
			if err != nil {
				continue
			}
			decl.Scenario = scenario
			out = append(out, decl)
		}
	}
	return out, nil
}
