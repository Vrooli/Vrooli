# Structure Phase

**ID**: `structure`
**Timeout**: 60 seconds
**Optional**: No
**Source**: validation-provider (`structure-health`)

The structure phase validates that a scenario has a well-formed skeleton and
correctly wired lifecycle before any tests run. It runs first and fails fast if
basic requirements aren't met.

Test Genie no longer runs a native structure checker. The phase is **delegated
to the [`structure-health`](../../../../structure-health/) scenario** through the
shared `ScenarioValidationService` contract (the same delegation model used by
unit-health, measures-health, scenario-dependency-analyzer, and the other
provider-backed phases). Test Genie calls
`structure-health validate scenario <scenario>`, maps the returned
`MaturityAssessment` findings into the `FINDING_SOURCE_STRUCTURE` channel, and
gates the phase on finding severity.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

Every declared surface of the scenario sits on a **trusted, profile-conformant skeleton**: `service.json` identity resolves and matches the directory, all required files and per-surface directories are present, declared and actual surfaces reconcile, each surface carries a complete build → start → port + env → health → freshness chain with valid dependency declarations, production serving follows canonical port/binary conventions, and the scenario conforms to every rule of its detected profile. At maximum maturity `structure-health` reports each of its five capabilities at its top rung — the scenario skeleton is clean enough for fleet-level comparison and orchestration with no structural debt.

## The rungs and their gates

structure-health reports a monotone ladder per capability (each rung implies the one below). The five capabilities and their ceilings:

| Capability | Ceiling | L1 → next unlock | Top-rung ("clean") aspiration |
|---|---|---|---|
| Identity Contract | L2 | Readable, self-consistent `service.json` → identity checks clean | Scenario identity is clean. |
| Surface Skeleton | L3 | Required files + surface dirs present → declared/actual surfaces reconcile → skeleton clean | Surface skeleton is clean. |
| Lifecycle Wiring | L3 | Lifecycle declarations inspectable → complete build/start/port/env/health/freshness chain + valid deps → clean | Lifecycle wiring is clean. |
| Production Serving | L3 | Serve commands/ports/binaries inspectable → production bundle + canonical ports + binary naming → clean | Production-serving posture is clean. |
| Profile Conformance | L3 | Profile rules evaluable → required structure/lifecycle/test/port/setup/env/storage conventions met → clean | Profile conformance is clean. |

## What each finding means

Each finding caps the named capability at a rung; only `ERROR`/`BLOCKER` severities fail the phase (`WARNING` findings are honest, non-failing debt).

| Code(s) | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `SERVICE_JSON_MISSING` / `SERVICE_JSON_INVALID` / `SERVICE_NAME_MISMATCH` | identity_contract | L0 | ERROR | Yes |
| `REQUIRED_FILE_MISSING` / `SURFACE_DIR_MISSING` | surface_skeleton | L1 | ERROR | Yes |
| `SURFACE_RECONCILE_MISMATCH` | surface_skeleton | L2 | WARNING | No |
| `LIFECYCLE_STEP_MISSING` / `HEALTH_CHECK_MISSING` / `HEALTH_CHECK_MALFORMED` / `HEALTH_ENDPOINT_MISSING` / `API_PREFLIGHT_MISSING` / `DEPENDENCY_DECLARATION_INVALID` | lifecycle_wiring | L2 | ERROR | Yes |
| `FRESHNESS_CHECK_MISSING` | lifecycle_wiring | L2 | WARNING | No |
| `API_SERVER_RUN_NONCONFORMANT` | production_serving | L2 | ERROR | Yes |
| `PRODUCTION_SERVE_NONCONFORMANT` / `PORT_BAND_NONCONFORMANT` / `API_BINARY_NAME_NONCONFORMANT` | production_serving | L2 | WARNING | No |
| `PROFILE_REQUIRED_STRUCTURE` / `PROFILE_UI_STRUCTURE` / `PROFILE_HEALTH_LIFECYCLE` / `PROFILE_PORTS` / `PROFILE_SETUP_CONDITIONS` / `PROFILE_HARDCODED_VALUES` / `PROFILE_RUNTIME_STORAGE` | profile_conformance | L2 | ERROR | Yes |
| `PROFILE_CONFORMANCE_VIOLATION` / `PROFILE_TEST_COVERAGE` / `PROFILE_SETUP_STEPS` / `PROFILE_ENV_VALIDATION` | profile_conformance | L2 | WARNING | No |

(An unrecognized profile downgrades the profile-conformance pack to advisory so non-Go / non-react-vite scenarios are not falsely failed. See the structure-health rule catalog for the full inventory.)

## The canonical fix

- **Identity findings** → add or repair `service.json`; make `service.name` equal the directory name. Many are auto-fixable with `structure-health fix-config apply <scenario>`.
- **Skeleton findings** → create the missing required files / per-surface directories; reconcile declared surfaces in `service.json` with the actual tree.
- **Lifecycle findings** → complete the build → start → port + env → health → freshness chain for each declared surface; add the `/health` endpoint and API preflight (manual — placement depends on the target router/`main.go`); fix malformed dependency declarations. Missing health-check config is auto-fixable.
- **Production-serving findings** → serve the built production bundle (not a dev server), move ports into the canonical band, name the API binary conventionally, and migrate the server entrypoint to `api-core/server.Run` (manual).
- **Profile-conformance findings** → satisfy the detected profile's required structure, UI lifecycle, ports, setup conditions, and runtime-storage conventions; remove hardcoded values. `structure-health fix-config run <scenario>` previews format-preserving repairs (dry-run by default).

## How to verify

```bash
# Current rung, gaps, and next move for every capability:
structure-health validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases structure
test-genie runs findings --scenario <scenario>
```

## What Gets Validated

structure-health reconciles **code-facts ground truth** (the surfaces,
languages, and frameworks actually present, from the `code-facts` scenario)
against the scenario's **declared `service.json` intent**, profile-aware:

- **Skeleton**: `service.json` present/valid, `service.name == directory`,
  required top-level files, a surface directory per declared surface.
- **Lifecycle wiring**: each declared surface has a build → start →
  port + env_var → health-check chain; freshness checks are present per
  buildable surface (a missing `binaries`/`ui-bundle` check causes silent
  rebuilds and is flagged); the UI develop step serves the built production
  bundle rather than a dev server; dependency declarations are well-formed.
- **Profile-keyed conformance packs** (migrated from scenario-auditor's
  `structure`/`config`/`ui` rule packs): for the recognized default
  `react-vite-go` profile these reproduce the previous verdicts; an
  unrecognized profile downgrades them to advisory so non-Go / non-react-vite
  scenarios are not falsely failed.

Many of these findings are **auto-fixable** —
`structure-health fix-config run|apply <scenario>` previews/applies
format-preserving `service.json` and skeleton repairs (dry-run by default).

See the structure-health scenario's own documentation for the full rule
catalog, profile model, and auto-fix behavior.

## Severity & Gating

Findings carry a severity; `ERROR`/`BLOCKER` findings fail the phase. Because
the default profile reproduces the previous Structure + scenario-auditor
verdicts, already-conformant react-vite/Go scenarios see no new failures.

## Resilience

Structure is the first/fast-fail phase, so it now depends on the
`structure-health` provider being reachable. The provider-contract scan tracks
provider conformance, and Test Genie's delegated-phase path degrades gracefully
(treating an unreachable provider as an environmental skip rather than a hard
failure) — the same contract every other delegated phase relies on.

## Related Documentation

- [structure-health scenario](../../../../structure-health/) - the provider that
  backs this phase (rules, profiles, auto-fix, fleet intelligence)
- [CLI Manifest Contract](cli-approaches.md) - manifest-driven scenario CLI
  adapters (the CLI surface itself is validated by the `contracts` phase via
  cli-health)
- [UI Smoke Tests](ui-smoke.md) - BAS-based UI validation (the `smoke` phase)
- [Playbooks Directory Structure](../playbooks/directory-structure.md)

## See Also

- [Phases Overview](../README.md) - All phases
- [Dependencies Phase](../dependencies/README.md) - Next phase
