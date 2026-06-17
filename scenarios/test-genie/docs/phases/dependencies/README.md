# Dependencies Phase

**ID**: `dependencies`
**Timeout**: 30 seconds
**Optional**: No
**Requires Runtime**: No

The dependencies phase is a read-only producer-backed preflight. Test Genie calls exactly one public producer RPC:

```bash
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

Scenario Dependency Analyzer owns dependency interpretation. Test Genie only orchestrates the phase, maps returned assessment findings into `FINDING_SOURCE_DEPENDENCY`, and stores the phase pointer. SDA's own CLI still exposes the native detail with `scenario-dependency-analyzer health <scenario>`.

## What Gets Checked

SDA health currently reports these machine-readable sections:

| Section | Owner | Purpose |
|---------|-------|---------|
| `surfaces` | Code Facts via SDA | Discovers scenario surfaces by evidence, not by hard-coded `api` / `ui` names |
| `readiness` | SDA | Checks commands, runtimes, package managers, Go modules, and JS/TS package state across discovered surfaces |
| `runtime` | SDA | Checks required resources and required scenario dependencies from `.vrooli/service.json` |
| `governance` | SDA | Validates dependencies against recorded approved-dependency governance without treating it as a hard allowlist |
| `release-age` | SDA | Validates pnpm `minimumReleaseAge` policy and governed exclusions |
| `security-index` | Security Health via SDA | Reports dependency-index availability without running vulnerability scanners or emitting vulnerability findings |
| `graph` | SDA | Reports declared-vs-actual scenario dependency graph drift |

## Failure Semantics

- SDA `ERROR` or `BLOCKER` findings fail the phase.
- SDA `WARNING` and `INFO` findings are surfaced as observations and architecture findings but do not fail the phase by themselves.
- If `scenario-dependency-analyzer` is unavailable or returns a malformed shared validation response, the phase fails with the appropriate provider contract failure. Test Genie does not fall back to native dependency checks.

## Common Commands

```bash
# Inspect the producer output directly
scenario-dependency-analyzer health <scenario> --json

# Human-readable SDA health summary
scenario-dependency-analyzer health <scenario>

# Search approved dependency governance records
scenario-dependency-analyzer deps approved search "React graph library" --json
```

## Common Failures

| Finding family | Cause | Typical next step |
|----------------|-------|-------------------|
| `dependency.readiness.*` | Missing command/runtime/package state or module/package drift | Run the remediation command reported by SDA |
| `dependency.runtime.*` | Required resource or scenario dependency stopped/unhealthy | Start or restart the reported dependency |
| `dependency.graph.*` | Actual import/interface evidence differs from `.vrooli/service.json` declarations | Update the declaration or remove stale usage |
| `dependency.governance.*` | Dependency is unrecorded, out of approved range, deprecated, or blocked | Use the SDA governance verbs (never hand-edit the JSON): `scenario-dependency-analyzer deps approved explain <ecosystem>/<pkg>` to see the verdict, then `approve-observed --apply` / `widen-range` / `deny-vulnerable` as appropriate |
| `dependency.release_age.*` | pnpm release-age policy missing/too low or exclusion lacks approval | Add/raise `minimumReleaseAge` or record an approved exception |
| Security index degradation | Security Health dependency index unavailable/degraded | Check `security-health deps status --json`; vulnerability findings remain owned by the `security` phase |

## See Also

- [Scenario Dependency Analyzer CLI](../../../../scenario-dependency-analyzer/docs/cli.md)
- [Phases Overview](../README.md)
- [Quality Phase](../quality/README.md)
