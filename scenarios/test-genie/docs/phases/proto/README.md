# `proto` phase

The `proto` phase is a **thin delegating phase**: it shells the
`proto-health` scenario's public CLI and maps scenario-scoped Protocol Buffer
contract findings into the shared `FINDING_SOURCE_PROTO` channel. test-genie
does not parse descriptors or validate proto organization itself; those checks
live in `proto-health`, alongside the proto-surface fact RPC.

## What it runs

```
proto-health validate scenario <name> --json
```

The `--json` payload is the proto-health validation response. Each finding is
mapped to an `ArchitectureFinding{Source: FINDING_SOURCE_PROTO}` via the shared
`newFinding` helper, so it carries a deterministic stable ID, normalized
severity, and the per-source effort default.

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

## Classification

- **Optional**, included in every built-in preset. Curated presets reference it
  explicitly; `comprehensive` auto-joins it through the default catalog.
- **120s timeout** — descriptor and generated-artifact checks should be fast,
  but the budget leaves room for slower worktrees.
- Missing `proto-health` CLI or unreachable producer API -> skip observation
  (never a hard failure).
- `TEST_GENIE_SKIP_PROTO=1` skips the phase entirely.

## See also

- `scenarios/proto-health/` — the producer scenario.
- `packages/proto/STYLE_GUIDE.md` — the proto organization standard.
- `packages/proto/schemas/architecture/v1/findings.proto` —
  `FINDING_SOURCE_PROTO = 12`.
- `packages/maturity-go/dimensions/dimensions.json` — the `proto-health`
  dimension plus test-genie source and phase maps.
