// Package extraction is business-health's ONLY doorway to a scenario's
// business-contract artifacts. It is a thin composition over
// packages/intent-go — the repo-wide single-parser ratchet forbids any
// PRD.md / requirements/ parsing outside that package, and this file is
// where business-health honors it: no regexes, no JSON decoding of
// registry files, only intent-go extractors composed into one Contract
// value the rest of the scenario consumes.
package extraction

import (
	"fmt"

	intent "intent-go"
)

// Contract is the extracted business contract for one scenario: the
// outcome claims (PRD operational targets) and requirement claims
// (requirements/ registry entries) for the alignment checks, plus the
// full structural models (PRD document, registry) the template and
// registry checks need, with enough provenance for actionable locations.
type Contract struct {
	// Scenario slug (directory name under scenarios/).
	Scenario string
	// Absolute scenario root the artifacts were read from.
	ScenarioDir string
	// True when PRD.md exists at the scenario root.
	PRDPresent bool
	// True when requirements/index.json exists.
	RegistryPresent bool
	// Outcome-altitude claims (one per operational target).
	Outcomes []intent.CapabilityClaim
	// Requirement-altitude claims (one per registry requirement).
	Requirements []intent.CapabilityClaim
	// PRDDoc is the structural PRD model (sections + operational targets).
	PRDDoc intent.PRDDocument
	// Registry is the structural requirements model (modules + records).
	Registry intent.Registry
}

// Extractor loads business contracts. The interface exists so checks and
// handlers can be tested against fixture contracts without touching disk.
type Extractor interface {
	Load(scenario, scenarioDir string) (Contract, error)
}

// FileExtractor is the production Extractor: intent-go file extractors
// over a real scenario tree.
type FileExtractor struct {
	prd  intent.PRDExtractor
	reqs intent.RequirementsExtractor
}

// NewFileExtractor wires the intent-go file extractors.
func NewFileExtractor() FileExtractor {
	return FileExtractor{
		prd:  intent.FilePRDExtractor{},
		reqs: intent.FileRequirementsExtractor{},
	}
}

// Load extracts the contract from scenarioDir. A missing PRD or registry
// is not an error — presence is reported on the Contract so checks can
// emit the corresponding findings (prd_missing_prd, prd_missing_requirements)
// rather than the load aborting.
func (e FileExtractor) Load(scenario, scenarioDir string) (Contract, error) {
	c := Contract{Scenario: scenario, ScenarioDir: scenarioDir}

	doc, err := intent.ExtractPRDDocument(scenarioDir)
	if err != nil {
		return c, fmt.Errorf("extract PRD document: %w", err)
	}
	c.PRDDoc = doc
	c.PRDPresent = doc.Present
	if doc.Present {
		outcomes, err := e.prd.ExtractPRDClaims(scenarioDir)
		if err != nil {
			return c, fmt.Errorf("extract PRD claims: %w", err)
		}
		c.Outcomes = outcomes
	}

	registry, err := intent.ExtractRequirementsRegistry(scenarioDir)
	if err != nil {
		return c, fmt.Errorf("extract requirements registry: %w", err)
	}
	c.Registry = registry
	c.RegistryPresent = registry.Present
	if registry.Present {
		// The claim extractor aborts on any unparseable module; the registry
		// extractor above tolerates them (recording ParseErrors, which the
		// checks turn into business_registry_unparseable findings). When the
		// tree is partially broken, run the linkage checks over whatever
		// parsed rather than failing the whole validation.
		reqs, err := e.reqs.ExtractRequirementClaims(scenarioDir)
		if err != nil {
			if len(registry.ParseErrors) == 0 {
				return c, fmt.Errorf("extract requirement claims: %w", err)
			}
			reqs = claimsFromRegistry(registry)
		}
		c.Requirements = reqs
	}
	return c, nil
}

// claimsFromRegistry rebuilds requirement claims from the parseable part of
// the registry model, mirroring the claim extractor's shape (used only when
// broken modules made the strict extractor abort).
func claimsFromRegistry(reg intent.Registry) []intent.CapabilityClaim {
	var claims []intent.CapabilityClaim
	for _, r := range reg.Requirements() {
		if r.ID == "" {
			continue
		}
		refs := make([]intent.Ref, 0, len(r.Validations)+1)
		if r.PRDRef != "" {
			refs = append(refs, intent.Ref{Raw: r.PRDRef, Path: r.PRDRef, Kind: intent.RefDoc})
		}
		for _, v := range r.Validations {
			if raw := v.Ref; raw != "" {
				refs = append(refs, intent.NormalizeRef(raw, v.Type))
			}
		}
		claims = append(claims, intent.CapabilityClaim{
			ID:         r.ID,
			Altitude:   intent.Requirement,
			Text:       r.Title + " " + r.Description,
			Anchor:     r.Module,
			Refs:       refs,
			Provenance: "requirements",
		})
	}
	return claims
}
