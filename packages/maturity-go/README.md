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

### `github.com/vrooli/maturity-go/assessment`

The shared health-scenario maturity contract helper. Health providers own their
phase-local maturity ladder in `.vrooli/maturity.json`, emit a shared
`common.v1.MaturityAssessment` object, and keep any richer provider-specific
fields beside it. This package validates the local spec, normalizes finding
maturity metadata, computes the provider-local current/next level, and preserves
fallback behavior when older providers have not emitted maturity metadata yet.

The spec shape is intentionally scenario-local:

```json
{
  "provider": "measures-health",
  "phase": "measures",
  "version": "1",
  "levels": [
    {
      "id": "L0",
      "name": "No measures contract readable",
      "description": "The target has no usable measures declaration.",
      "entry_criteria": [],
      "exit_criteria": ["A measures contract can be parsed."]
    }
  ],
  "findings": {
    "measures.uncovered-domain": {
      "local_level_impact": "L2",
      "global_impact": "capability_gap",
      "dimension": "measures",
      "severity_default": "ERROR",
      "recommended_skill_ids": ["measures-adoption"]
    }
  },
  "fallback": {
    "local_level_impact": "L1",
    "global_impact": "unknown",
    "dimension": "measures",
    "severity_default": "WARNING"
  }
}
```

Global impact vocabulary is semantic and stable:

- `foundation_blocker`: cannot reliably run, build, test, or validate basics.
- `safety_blocker`: severe security, dependency, or safety issue.
- `evolvability_gap`: architecture, contract, docs, proto, or dependency drift.
- `hardening_gap`: coverage, flake, UI/visual/performance, or test-depth gap.
- `capability_gap`: missing operational target, business validation, or measures adoption.
- `advisory`: useful but not maturity-blocking.
- `unknown`: producer cannot classify; fallback rules apply.

`assessment` remains pure logic: callers pass JSON bytes or structs, and the
package never discovers or reads scenario files itself.

For the operator-facing contract, including the human-output-first rule and
JSON automation boundary, see
[`docs/reference/health-maturity-assessments.md`](../../docs/reference/health-maturity-assessments.md).

### `github.com/vrooli/maturity-go/dimensions`

The controller's improvement-dimension vocabulary (`standards`, `tests`,
`structure`, …) and the SSOT mapping tables from test-genie finding sources
and phase names into that vocabulary. The data lives in the embedded
`dimensions.json`; edit the JSON, never the accessors.

The `architecture` phase maps to the `structure` dimension because a clean
cartographer run primarily proves domain-authority and placement coverage.
Provider-local finding metadata can still route specific findings to `cycles`,
so import-cycle drift remains a separate hard evolvability signal while the
phase-level posture stays tied to structure.

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
