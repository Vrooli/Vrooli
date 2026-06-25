package spacedoc

import "testing"

const answerFixture = `# Answer Space

## This Space

| | |
|---|---|
| Projection | Answer |
| Owner | ` + "`search-hub`" + ` (holds the registry) |
| Denominator confidence | ` + "`PARTIAL`" + ` — cells enumerated from the model, not swept. |

## Coverage Grid

### G0 — Project (whole repo)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 23 | Repo contract · Conformance — "Does it conform?" | ` + "`contract-registry.contracts`" + ` | IN-REACH (gap stub) | DERIVED / VALIDATED | contract-registry is the home. |
| 24 | ⭐ Control-plane CLI · Anatomy — "How is it structured?" | _(none)_ | MISSING | DERIVED | Needs a registry. |

### G1 — Scenario (whole)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 1 | Surfaces · Inventory | ` + "`ui-health.surfaces`" + ` | NOW (UI, CLI) / IN-REACH (API) | DERIVED | UI + CLI live. |

## Known Gaps

Prose that must not be parsed as a cell.
`

const validateFixture = `# Validate Space

## This Space

| | |
|---|---|
| Projection | Validate |
| Owner | ` + "`test-genie`" + ` (owns the phase catalog) |
| Denominator confidence | Covered set: ` + "`AUTHORITATIVE`" + `. Candidate delta: ` + "`SKETCH`" + `. |

## Coverage Grid

### Current phases (the covered set — ` + "`AUTHORITATIVE`" + `)

| # | Concern (phase) | Owner (health scenario) | Status | Autofix | Notes |
|---|---|---|---|---|---|
| V1 | Structure | ` + "`structure-health`" + ` | NOW | mechanical | Skeleton wiring. |

### Candidate concerns (the denominator delta — ` + "`SKETCH`" + `)

| # | Candidate concern | Status | Autofix | Notes |
|---|---|---|---|---|
| V18 | Observability | MISSING | partial | No dedicated phase. |
`

const guideFixture = `# Guide Space

## This Space

| | |
|---|---|
| Projection | Guide |
| Owner | ` + "`prompt-manager`" + ` (owns the skill graph) |
| Denominator confidence | ` + "`SKETCH`" + ` — first cut. |

## Coverage Grid

| # | SWE task | Guiding skill(s) | Status | Gate | Notes |
|---|---|---|---|---|---|
| **Understand** | | | | | |
| G1 | Explore a codebase | ` + "`explore`" + ` | COVERED | — | Answer providers help. |
| **Dependencies / deploy** | | | | | |
| G26 | Dependency work | ` + "`platform-package-hardening`" + ` | PARTIAL | — | + analyzer flows. |
`

func TestParseAnswer(t *testing.T) {
	def, err := Parse(ProjectionAnswer, []byte(answerFixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if def.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %q", def.SchemaVersion)
	}
	if def.Owner != "search-hub" {
		t.Errorf("owner = %q, want search-hub", def.Owner)
	}
	if def.DenominatorConfidence != ConfidencePartial {
		t.Errorf("confidence = %q, want partial", def.DenominatorConfidence)
	}
	if len(def.Cells) != 3 {
		t.Fatalf("cells = %d, want 3 (prose must not parse): %+v", len(def.Cells), def.Cells)
	}
	c0 := def.Cells[0]
	if c0.ID != "23" || c0.Status != StatusInReach || c0.Basis != BasisDerived {
		t.Errorf("cell0 = %+v", c0)
	}
	if c0.Group != "G0 - Project (whole repo)" {
		t.Errorf("cell0.group = %q", c0.Group)
	}
	if c0.Owner != "contract-registry.contracts" {
		t.Errorf("cell0.owner = %q", c0.Owner)
	}
	if len(c0.Notes) != 1 || c0.Notes[0] != "contract-registry is the home." {
		t.Errorf("cell0.notes = %+v", c0.Notes)
	}
	// Star + combined status on the G1 cell.
	c2 := def.Cells[2]
	if c2.ID != "1" || c2.Status != StatusNow {
		t.Errorf("cell2 status: %+v", c2)
	}
	if got := c2.Question; got != "Surfaces · Inventory" {
		t.Errorf("cell2.question = %q", got)
	}
	if def.Cells[1].Question != "Control-plane CLI · Anatomy — \"How is it structured?\"" {
		t.Errorf("star not stripped: %q", def.Cells[1].Question)
	}
}

func TestParseValidate(t *testing.T) {
	def, err := Parse(ProjectionValidate, []byte(validateFixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if def.Owner != "test-genie" {
		t.Errorf("owner = %q", def.Owner)
	}
	if def.DenominatorConfidence != ConfidenceAuthoritative {
		t.Errorf("confidence = %q, want authoritative (first token)", def.DenominatorConfidence)
	}
	if len(def.Cells) != 2 {
		t.Fatalf("cells = %d, want 2: %+v", len(def.Cells), def.Cells)
	}
	if def.Cells[0].ID != "V1" || def.Cells[0].Status != StatusNow {
		t.Errorf("V1 = %+v", def.Cells[0])
	}
	if def.Cells[0].Basis != "" {
		t.Errorf("validate must not carry basis: %q", def.Cells[0].Basis)
	}
	if def.Cells[1].ID != "V18" || def.Cells[1].Status != StatusMissing {
		t.Errorf("V18 = %+v", def.Cells[1])
	}
}

func TestParseGuide(t *testing.T) {
	def, err := Parse(ProjectionGuide, []byte(guideFixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if def.Owner != "prompt-manager" {
		t.Errorf("owner = %q", def.Owner)
	}
	if def.DenominatorConfidence != ConfidenceSketch {
		t.Errorf("confidence = %q", def.DenominatorConfidence)
	}
	if len(def.Cells) != 2 {
		t.Fatalf("cells = %d, want 2 (category rows excluded): %+v", len(def.Cells), def.Cells)
	}
	g1 := def.Cells[0]
	if g1.ID != "G1" || g1.Status != StatusNow || g1.Group != "Understand" {
		t.Errorf("G1 = %+v", g1)
	}
	if g1.Owner != "explore" {
		t.Errorf("G1.owner = %q", g1.Owner)
	}
	g26 := def.Cells[1]
	if g26.ID != "G26" || g26.Status != StatusInReach || g26.Group != "Dependencies / deploy" {
		t.Errorf("G26 = %+v", g26)
	}
}

func TestParseProjectionMismatch(t *testing.T) {
	if _, err := Parse(ProjectionGuide, []byte(answerFixture)); err == nil {
		t.Fatal("expected projection-mismatch error")
	}
}

func TestParseUnknownProjection(t *testing.T) {
	if _, err := Parse(Projection("nope"), []byte(answerFixture)); err == nil {
		t.Fatal("expected unknown-projection error")
	}
}

func TestNormalizeStatus(t *testing.T) {
	cases := map[string]CellStatus{
		"NOW":                            StatusNow,
		"COVERED":                        StatusNow,
		"IN-REACH (gap stub)":            StatusInReach,
		"PARTIAL":                        StatusInReach,
		"MISSING":                        StatusMissing,
		"NOW (UI, CLI) / IN-REACH (API)": StatusNow,
		"":                               StatusMissing,
		"garbage":                        StatusMissing,
	}
	for in, want := range cases {
		if got := normalizeStatus(in); got != want {
			t.Errorf("normalizeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeBasis(t *testing.T) {
	cases := map[string]Basis{
		"DERIVED":                       BasisDerived,
		"DERIVED / VALIDATED":           BasisDerived,
		"DECLARED_UNVERIFIED → DERIVED": BasisDeclaredUnverified,
		"HEURISTIC → DECLARED":          BasisDeclaredUnverified,
		"PARTIAL (reconstructed)":       BasisDeclaredUnverified,
		"ABSENT":                        BasisAbsent,
		"":                              Basis(""),
	}
	for in, want := range cases {
		if got := normalizeBasis(in); got != want {
			t.Errorf("normalizeBasis(%q) = %q, want %q", in, got, want)
		}
	}
}
