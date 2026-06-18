# Architecture Phase

The `architecture` phase audits a scenario's **structural cohesion** — does the
code's shape scream the product's capabilities, and do the domains hold together
without import cycles, runaway coupling, or misplaced files? It delegates to the
**architecture-cartographer** scenario's shared `ScenarioValidationService` and
normalizes the findings into the shared `ArchitectureFinding` contract that
every test-genie phase now emits.

This is the **cohesion axis** of the audit battery. The per-surface phases
(`contracts`, `ui-health`, `docs`, `standards`) ask "is each surface built
right?"; the `architecture` phase asks "does the whole scenario cohere?".

## How It Runs

Test Genie resolves the architecture-cartographer base URL via service discovery
and calls `ScenarioValidationService.ValidateScenario` for the target scenario.
The shared `assessment.findings` become Observations and normalized
`ArchitectureFinding`s (`source = ARCHITECTURE`). Architecture Cartographer
packs its native `AuditRunResponse` into `native_detail` for its own CLI/UI.

Equivalent operator flow:

```bash
architecture-cartographer audit run <scenario>
```

## Confidence-Gated Enforcement

The phase is **Optional** and graded. Coupling, convergence, naming, and other
heuristic findings remain advisory signals that feed campaign tracking. The
phase hard-fails only when deterministic error/blocker findings are trustworthy
enough to act on. Test Genie reads the native cartographer `AuditRunResponse`
from `native_detail` so it can distinguish `finding_class=deterministic` from
`finding_class=heuristic`:

- default `TEST_GENIE_ARCHITECTURE_GATE=high-confidence`: fail on deterministic
  error/blocker findings only when cartographer reports
  `authority_confidence=high`;
- `TEST_GENIE_ARCHITECTURE_GATE=all`: fail on deterministic error/blocker
  findings regardless of authority confidence;
- `TEST_GENIE_ARCHITECTURE_GATE=off`: never gate architecture findings.

Transport errors (cartographer unreachable) and cartographer `TOOL_ERROR`
outcomes still fail the phase independently of the finding gate. Heuristic
findings still surface in the report as advisory observations, but do not
contribute to the gate.
Together with the overall finding count, they drive the **campaign nudge**:
when findings exceed the single-pass threshold, the suite output steers you to
open a tracked improvement campaign in architecture-cartographer rather than
fixing ad-hoc.

The native audit detail also carries the cartographer score matrix. Test Genie
renders those categories in the phase observations so operators can see
placement legibility, surface alignment, boundary cleanliness, naming clarity,
and authority posture without treating heuristic scores as pass/fail gates.

## Preset

The phase is included in the `architecture-audit` preset, the single command the
screaming-architecture audit skill points at:

```bash
vrooli scenario test <scenario> --preset architecture-audit
# runs: structure, contracts, ui-health, docs, standards, architecture
```

## Summary Metrics

The phase pointer records an `ArchitectureSummary`:

| Field | Meaning |
|---|---|
| `outcome` | cartographer audit outcome (`clean` / `findings` / `tool_error`) |
| `total` | total findings after filters |
| `blockers` / `errors` / `warnings` / `infos` | counts by normalized severity |
| `suppressed` | findings excused by in-repo `// arch:allow` markers (reported, not dropped) |
| `authority_confidence` | domain-derivation authority confidence (`high`, `medium`, `low`, or `missing`) |
| `gate_mode` | effective gate mode (`off`, `high-confidence`, or `all`) |
| `gated_blockers` | deterministic error/blocker findings considered by the gate |
| `categories` | score-matrix category summaries copied from cartographer native detail |

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip architecture
```

Disable for the process via environment:

```bash
TEST_GENIE_SKIP_ARCHITECTURE=1
```

Keep findings advisory during rollout:

```bash
TEST_GENIE_ARCHITECTURE_GATE=off
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json` (default: 120s):

```json
{
  "phases": {
    "architecture": { "timeout": "180s" }
  }
}
```
