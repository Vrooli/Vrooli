# `security` phase

The `security` phase is a **thin delegating phase**: it shells the
`security-health` scenario's public CLI and maps its normalized findings into
the shared `FINDING_SOURCE_SECURITY` channel. test-genie itself contains **no
security scanners** — all stack-specific scanning (secrets, Go SAST, Go
vuln-DB, JS dependency CVEs) lives in `security-health`, which is the only
component that knows how to scan each substrate. This mirrors the
`contracts` (cli-health) and `ui-health` delegation phases.

## What it runs

```
security-health validate scenario <name> --json
```

The `--json` payload is the proto wire shape of `ValidateScenarioResponse`
(snake_case field names, `SEVERITY_*` enum strings). Each finding is mapped to
an `ArchitectureFinding{Source: FINDING_SOURCE_SECURITY}` via the shared
`newFinding` helper, so it carries a deterministic stable ID, normalized
severity, and the per-source effort default (`MEDIUM`).

## Severity contract (load-bearing)

security-health performs the normalization; this phase routes its severity
strings through the same `normalizeFindingSeverity` table every producer uses:

| security-health severity | normalized | gates R1? |
|---|---|---|
| `SEVERITY_ERROR` (critical/high) | ERROR | **yes** |
| `SEVERITY_WARNING` (moderate/medium) | WARNING | no |
| `SEVERITY_INFO` (low/info/degraded) | INFO | no |

Only ERROR findings fail the phase. They flow into the ecosystem-manager
`security` dimension, which hard-gates the **R1 ("Safe")** ladder rung — so a
leaked credential or a reachable CVE holds a scenario at R1 until it is
resolved.

## Classification

- **Optional**, auto-joins the `comprehensive` preset (anti-drift guard).
- **180s timeout** — scanners (govulncheck/osv-scanner) hit network vuln DBs
  and are slower than the other delegating phases.
- Missing `security-health` CLI → `FailureClassMissingDependency` (skip-class,
  never a hard failure). Install via `vrooli scenario start security-health`.
- `TEST_GENIE_SKIP_SECURITY=1` skips the phase entirely (matches the
  `ui-health` escape hatch).

## Degraded scanners

When an applicable scanner's binary is absent (e.g. `osv-scanner` not
installed) or cannot complete, security-health emits an **INFO** observation
rather than a failure. The phase stays green; the gap is visible but advisory.

## See also

- `scenarios/security-health/` — the producer scenario.
- `packages/proto/schemas/architecture/v1/findings.proto` —
  `FINDING_SOURCE_SECURITY = 9`.
- `scenarios/ecosystem-manager/api/pkg/dimensions/dimensions.json` — the
  `security` dimension + `testgenie_source_map` / `testgenie_phase_map` wiring.
