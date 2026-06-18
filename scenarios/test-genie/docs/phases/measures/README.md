# `measures` phase

The `measures` phase is a **thin delegating phase**: it calls the
`measures-health` scenario's shared `ScenarioValidationService` and maps its
normalized findings into the shared `FINDING_SOURCE_MEASURES` channel.
test-genie itself contains **no coverage logic** — deriving a scenario's
stateful domains, classifying each covered/waived/uncovered, and grading
per-measure extraction tier all live in `measures-health`.

## What it runs

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

The static provider path is used so the target scenario does not need to be
running. Test Genie reads the shared `status` and `assessment.findings` fields.
Each assessment finding is mapped to an `ArchitectureFinding{Source:
FINDING_SOURCE_MEASURES}`, so it carries a deterministic stable ID, normalized
severity, and the per-source effort default.

## Severity contract (load-bearing)

measures-health performs the normalization; this phase routes its severity
strings through the same `normalizeFindingSeverity` table every producer uses:

| measures-health severity | normalized | feeds the `measures` dimension as a gap? |
|---|---|---|
| `SEVERITY_ERROR` (uncovered stateful domain / hollow declaration / malformed) | ERROR | **yes** |
| `SEVERITY_WARNING` (stale waiver / fallback tier) | WARNING | no |
| `SEVERITY_INFO` (partial tier / not-expected context) | INFO | no |

Only ERROR findings fail the phase. They flow into the ecosystem-manager
`measures` dimension, which is a **soft** rung (R4) — a scenario stays
runnable and safe without measures, but cannot reach top maturity while a
stateful domain is left uncovered and unwaived.

## Classification

- **Optional**, auto-joins the `comprehensive` preset (anti-drift guard).
- **180s timeout** — coverage harvest reads manifests + the proto descriptor
  across the target.
- Unreachable `measures-health` API → advisory skip because the phase is
  optional. Start it via `vrooli scenario start measures-health`.
- `TEST_GENIE_SKIP_MEASURES=1` skips the phase entirely (matches the
  `security` escape hatch).

## Degraded producer

When the `measures-health` API is unreachable, the phase
emits a skip observation and stays green — the gate only acts on real findings,
never on the producer's availability.

## See also

- `scenarios/measures-health/` — the producer scenario.
- `packages/measures-go/` — the shared measure contract library.
- `packages/proto/schemas/architecture/v1/findings.proto` —
  `FINDING_SOURCE_MEASURES = 10`.
- `scenarios/ecosystem-manager/api/pkg/dimensions/dimensions.json` — the
  `measures` dimension + `testgenie_source_map` / `testgenie_phase_map` wiring.
