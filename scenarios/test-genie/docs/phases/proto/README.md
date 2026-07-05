# `proto` phase

The `proto` phase is a **thin delegating phase**: it calls the `proto-health`
scenario's shared `ScenarioValidationService` and maps scenario-scoped Protocol
Buffer contract findings into the shared `FINDING_SOURCE_PROTO` channel.
test-genie does not parse descriptors or validate proto organization itself;
those checks live in `proto-health`, alongside the proto-surface fact RPC.

## What it runs

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

Test Genie reads the shared `status`, `assessment.local`, and
`assessment.findings` fields. Each assessment finding is mapped to an
`ArchitectureFinding{Source: FINDING_SOURCE_PROTO}`, so it carries a
deterministic stable ID, normalized severity, and the per-source effort
default. The phase summary carries proto-health's `current_level`, `next_level`,
`clean`, and `unknown_count` convergence signals; phase pass/fail still comes
only from `status`.

## Severity contract

proto-health owns the rule semantics and severity tier. This phase only
normalizes the emitted severity string:

| proto-health severity | normalized | feeds the `proto-health` dimension as a gap? |
|---|---|---|
| `SEVERITY_ERROR` | ERROR | **yes** |
| `SEVERITY_WARNING` | WARNING | no |
| `SEVERITY_INFO` | INFO | no |

Only ERROR findings fail the phase. They flow into the ecosystem-manager
`proto-health` dimension, which is a soft R2 ladder input for evolvable
architecture and contract health.

proto-health also marks each finding with a `clean_requirement`. REQUIRED
findings gate proto-health's local rung even when they are WARNING-level debt;
UNCHECKABLE findings are excluded from the rung and surfaced as unknown. This
lets agents continue fixing a scenario toward a clean proto surface without
turning warning-level debt into a Test Genie failure.

Generated-artifact drift for the target scenario is still an ERROR
(`proto.gen_out_of_sync`). A proto-bearing scenario with no committed
generation lockfile emits `proto.gen_manifest_missing` as an ERROR.
Toolchain pin drift emits `proto.gen_toolchain_drift` as a WARNING, so it is
visible but does not fail the phase.

Evidence checks from code-facts are advisory unless they prove a contradiction:
`proto.proto_adoption_missing`, `proto.endpoint_proof_missing`, and
`*_unsupported` findings are WARNING. `*_contradicted` findings remain ERROR.
Absence of proof therefore does not make the phase outcome depend on whether
code-facts happens to be running.

## Classification

- **Optional**, included in every built-in preset. Curated presets reference it
  explicitly; `comprehensive` auto-joins it through the default catalog.
- **120s timeout** — descriptor and generated-artifact checks are file reads and
  hashing; the budget mainly protects RPC startup and slow worktrees.
- Unreachable `proto-health` API -> skip observation (never a hard failure).
- `proto.gen_toolchain_drift` -> advisory finding (never a hard failure).
- `TEST_GENIE_SKIP_PROTO=1` skips the phase entirely.

## See also

- `scenarios/proto-health/` — the producer scenario.
- `packages/proto/STYLE_GUIDE.md` — the proto organization standard.
- `packages/proto/schemas/architecture/v1/findings.proto` —
  `FINDING_SOURCE_PROTO = 12`.
- `scenarios/proto-health/.vrooli/test-genie.json` — the descriptor-owned
  `proto-health` dimension and phase coverage metadata.
