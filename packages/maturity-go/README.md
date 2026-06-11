# maturity-go

Shared single source of truth for **scenario maturity**: the canonical
improvement-dimension vocabulary and the R0–R4 maturity-ladder gate
predicates. Extracted verbatim from ecosystem-manager's `pkg/dimensions` and
`pkg/ladder` by the scenario-status-layer plan so both the live control loop
(ecosystem-manager) and cached status readers (scenario-completeness-scoring)
compute rungs with the same predicates.

This is a **pure-logic library**: no service calls, no filesystem state, ever.
Its only dependency is the generated `architecture/v1` proto types
(`packages/proto`) for the `FindingSource` enum.

## Packages

### `github.com/vrooli/maturity-go/dimensions`

The controller's improvement-dimension vocabulary (`standards`, `tests`,
`structure`, …) and the SSOT mapping tables from test-genie finding sources
and phase names into that vocabulary. The data lives in the embedded
`dimensions.json`; edit the JSON, never the accessors.

```go
dim, ok := dimensions.ForSource(architecturev1.FindingSource_FINDING_SOURCE_COVERAGE)
dim, ok = dimensions.ForPhase("unit")
all := dimensions.All()
phases := dimensions.PhasesForDimensions("tests", "coverage")
```

Anti-drift guards: the package tests fail when test-genie's `FindingSource`
proto enum or phase catalog adds an entry that `dimensions.json` does not map
(fixture: `dimensions/testdata/testgenie_audit_fixture.json`).

### `github.com/vrooli/maturity-go/ladder`

The canonical rung ladder (R0 Runnable & green → R4 Capability progression)
and the deterministic gate predicates deciding the lowest unsatisfied rung.
Standalone: imports only the dimension vocabulary; callers derive `Signals`
from their own findings state.

```go
sig := ladder.Signals{ErrorPlusByDimension: errs, CountByDimension: counts, BuildPassing: true /* … */}
rung, ok := ladder.Lowest(sig, ladder.DefaultThresholds(), "")  // "" = whole ladder
clean := ladder.AllHold(sig, ladder.DefaultThresholds(), topRung)
```

## Consumers

- `scenarios/ecosystem-manager/api` — autosteer rung selection/termination,
  findings dimension mapping, effectiveness ledger, skillmap (live findings).
- `scenarios/scenario-completeness-scoring` — rung headline + per-dimension
  breakdown computed from cached test-genie artifacts (planned by the
  scenario-status-layer plan).

Consumers may legitimately disagree per snapshot: ecosystem-manager computes
rungs on live findings; cached readers must label theirs "as of digest td:…".

## Adoption

Governed Go module, `go_module_replace` adoption:

```
require github.com/vrooli/maturity-go v0.0.0
replace github.com/vrooli/maturity-go => ../../../packages/maturity-go
```

(Plus the transitive `packages/proto` replace, which every scenario already
carries.)
