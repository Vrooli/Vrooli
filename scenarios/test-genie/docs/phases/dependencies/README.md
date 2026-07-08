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

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

Every declared dependency is **real, governed, and healthy**: dependency surfaces are fully inventoried, required runtime resources and scenario dependencies are present, running, and healthy, package metadata and local install state are clean, new package versions are held behind the default release-age cooldown with governed exceptions, observed direct dependencies all carry valid governance decisions, and the declared dependency graph matches observed import/interface evidence. At maximum maturity the scenario's dependency posture is provably accurate and safe.

## The rungs and their gates

SDA reports a ladder per capability. The rungs are monotone — each implies the one below.

| Capability | Rungs | Top rung (aspiration) | Key next unlock |
|---|---|---|---|
| Surface Inventory | L0–L3 | Dependency surface inventory is clean and ready for downstream checks. | Readable target → classified surfaces → clean inventory. |
| Runtime Dependencies | L0–L4 | Runtime dependencies are present, running, and healthy. | Manifest readable → present → running → healthy. |
| Package Readiness | L0–L4 | Package readiness is clean across discovered dependency surfaces. | Toolchain → metadata present → coherent (lockfiles/replaces/install) → clean. |
| Release Age Policy | L0–L4 | Release-age policy and exceptions are governed. | Policy readable → minimum configured → meets cooldown → exclusions governed. |
| Dependency Governance | L0–L3 | Dependency governance is clean for the target scenario. | Registry readable → observed deps governed → no blocking findings. |
| Graph Accuracy | L0–L2 | Declared dependency graph matches observed evidence. | Graph comparable → no undeclared usage or stale declarations. |

## What each finding means

Each finding caps the capability it names at a rung; only ERROR/BLOCKER severities fail the phase, so WARNING/INFO findings are honest, non-failing debt. Representative codes (full inventory in the descriptor):

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `dependency.runtime.manifest_readable` | runtime_dependencies | L0 | ERROR | Yes |
| `dependency.runtime.resource_healthy` | runtime_dependencies | L4 | ERROR | Yes |
| `dependency.runtime.scenario_healthy` | runtime_dependencies | L4 | ERROR | Yes |
| `dependency.go.build` | package_readiness | L3 | ERROR | Yes |
| `dependency.node.lockfile_present` | package_readiness | L3 | ERROR | Yes |
| `dependency.release_age.minimum_value` | release_age_policy | L3 | ERROR | Yes |
| `dependency.release_age.exclude_governed` | release_age_policy | L4 | ERROR | Yes |
| `dependency.governance.registry_readable` | dependency_governance | L0 | ERROR | Yes |
| `dependency.governance.approved_dependency` | dependency_governance | L2 | WARNING | No |
| `dependency.graph.undeclared` | graph_accuracy | L2 | WARNING | No |

## The canonical fix

- **Surface-inventory findings** (`dependency.surfaces.none`, `dependency.surface.unsupported*`) → make dependency surfaces discoverable, or accept the explicit unsupported classification.
- **Runtime findings** (`dependency.runtime.*`) → make the `.vrooli/service.json` manifest readable, then start/restart the required resource or scenario dependency until it reports present, running, and healthy.
- **Package-readiness findings** (`dependency.go.*`, `dependency.node.*`, `dependency.command.available`, `dependency.gomod.replace.missing`) → install missing toolchain commands, add the package metadata (`go.mod`/`package.json`), fix Go module tidy/build and local replaces, and ensure a single Node lockfile with coherent install state. `dependency.gomod.replace.missing` is auto-fixable.
- **Release-age findings** (`dependency.release_age.*`) → make the pnpm policy readable, configure a `minimumReleaseAge` that meets Vrooli's default cooldown, and record approved governance for any exclusions.
- **Governance findings** (`dependency.governance.*`) → never hand-edit the approved-dependency JSON; use `scenario-dependency-analyzer deps approved explain <ecosystem>/<pkg>` then `approve-observed --apply` / `widen-range` / `deny-vulnerable` as appropriate.
- **Graph-accuracy findings** (`dependency.graph.*`, `dependency.undeclared-but-used`, `dependency.declared-without-import-evidence`) → reconcile `.vrooli/service.json` declarations with observed import/interface evidence: declare undeclared usage or remove stale declarations.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
scenario-dependency-analyzer validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases dependencies
test-genie runs findings --scenario <scenario>
```

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
