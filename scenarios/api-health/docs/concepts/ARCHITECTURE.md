# Architecture — API Health

This document defines API Health's system shape, provider boundary, data flow,
extension rules, and cutover responsibilities.

## Purpose Of This Document

This document owns:

- the scenario's provider architecture,
- the role of each surface,
- how validation data flows between target scenario, provider, CLI, UI, and Test Genie,
- the shared infrastructure boundary,
- extension rules for future code,
- architecture maturity and intentional deviations.

This document does not own:

- product capability inventory: [`DOMAINS.md`](DOMAINS.md),
- temporal and user/system workflows: [`FLOWS.md`](FLOWS.md),
- storage details and retention: [`DATA.md`](DATA.md),
- resource and scenario dependencies: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md).

## Scenario Shape

API Health is a meta/provider scenario. Its core loop is:

```text
Target scenario tree + lifecycle metadata
  -> API Health validation engine
      -> capability findings
      -> common.v1.MaturityAssessment
      -> native API Health detail
      -> optional live /health probe evidence
      -> optional deterministic fix candidates
  -> CLI / UI / Test Genie delegated phase
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Provider engine | Target resolution, validation rules, probe execution, assessment mapping, fix registry | CLI formatting, browser state, adjacent provider policy |
| CLI (`cli/`) | Agent/operator wrapper | Arguments, API invocation, human/JSON reports | Validation decisions or duplicated rule logic |
| UI (`ui/`) | Operator inspection console | Capability summary, findings, probe evidence, fix preview | Independent validation model |
| Proto/contracts | Wire shape | Shared validation RPC plus native detail messages | Hand-written DTO mirrors |
| `.vrooli/maturity.json` | Provider maturity source | Capability ladders, finding mappings, fixability declarations | Runtime finding construction |

The load-bearing invariant: **all finding codes are declared in
`.vrooli/maturity.json` before implementation emits them**. The provider can
change rule internals freely, but the maturity contract is stable and reviewable.

## System Boundaries

API Health owns:

- API readiness validation for Vrooli scenarios,
- target resolution and API surface classification,
- static lifecycle checks for `.vrooli/service.json`, `api-core/preflight`, and `api-core/server`,
- static and live `/health` contract checks,
- low-ambiguity HTTP response checks,
- API-runtime hygiene checks,
- deterministic fixes for local, unambiguous repairs,
- migration accounting for legacy scenario-auditor API rules.

API Health does not own:

- lint/type/config quality: `quality-health`,
- security headers/CORS/scanners: `security-health`,
- CLI manifest/runtime conformance: `cli-health`,
- proto breaking changes or generated-code freshness: `proto-health`,
- UI rendering/interop: `ui-health`,
- storage isolation/migrations: `storage-health`,
- performance/load budgets: `performance-health`,
- generic file-handle hygiene outside API-runtime scope.

## Contracts And Data Flow

The provider API uses the shared validation contract:

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

The response carries:

- `status`: canonical pass/fail/error/degraded status for Test Genie.
- `assessment`: common maturity assessment built from `.vrooli/maturity.json`.
- `native_detail`: API Health's own report: target summary, capability summaries, static findings, live probe evidence, migration decisions, and fix preview metadata.
- `metrics`: real execution metrics collected by the provider.

Validation phases:

1. Resolve target scenario and classify API applicability.
2. Load service/lifecycle metadata and API source surfaces.
3. Run static lifecycle, health, HTTP semantics, and runtime hygiene checks.
4. If `include_execution` is true, run exactly one bounded live health probe through lifecycle-discovered URL.
5. Normalize findings, map them to maturity capabilities, and build native detail.
6. Optionally preview deterministic fixes through the shared Fix RPC.

REST endpoints inside API Health itself are limited to the standard health probe
and generated template exceptions. Provider validation, probe reports, and fix
operations should be Connect-RPC.

## Shared Infrastructure

Shared infrastructure is allowed only when it is business-vocabulary-free and
used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not an API Health product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | Provider-local database bootstrap. | Cross-cutting persistence infrastructure. | API boot, health, future probe history. |
| `api/internal/clock/` | Deterministic time seam. | Time is cross-cutting and test-substitutable. | Probe timestamps, metrics, reports. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains. | API tests. |
| `ui/src/test-utils/` | Cross-feature render and a11y helpers. | Used by unrelated UI features. | UI tests. |

## Extension Rules

1. Add or update `.vrooli/maturity.json` before emitting a new finding code.
2. Add proto messages before adding new API/CLI/UI payload shapes.
3. Keep CLI commands thin over API calls.
4. Render UI from native API Health detail; do not invent frontend-only finding categories.
5. Add fixture tests for every validator, including at least one false-positive guard.
6. Auto-fix only local, deterministic repairs; otherwise declare `fix_class: manual` with a reason.
7. Use lifecycle-discovered URLs for live probes; do not run target binaries directly.
8. When migrating scenario-auditor rules, preserve useful contract intent and redesign flawed implementation logic.

## Architecture Maturity

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| Foundation docs | Active | PRD, requirements, domains, architecture, maturity spec. | Keep specs aligned as Test Genie cutover lands. |
| Provider contract | Active | Shared `ScenarioValidationService`, target resolver, maturity assessment, metrics, and native detail. | Test Genie is not yet cut over to consume API Health for API readiness. |
| API lifecycle checks | Active | Requirements `APIH-LIFE-*`, lifecycle AST fixtures, and service metadata validation. | Broader endpoint inventory reconciliation remains planned under `APIH-MIG-002`. |
| Live health probe | Active | Requirements `APIH-HEALTH-*`, bounded execution-mode `/health` probe, and httptest coverage. | Representative endpoint probe packs remain opt-in future work. |
| HTTP/runtime checks | Active | Requirements `APIH-HTTP-*`, `APIH-RUN-*`, HTTP semantics fixtures, and runtime hygiene fixtures. | Generic security/storage/quality concerns stay delegated to neighboring providers. |
| Autofix | Active | Requirements `APIH-FIX-*`, shared fix RPCs, CLI fix commands, and deterministic fixer tests. | Design-bearing repairs stay manual until a safe rewrite is proven. |
| Operator UI | Active | Requirement `APIH-UX-001`, validation workbench route, typed validation API wrapper, and UI tests. | Fleet/API facts reporting remains planned future work. |

## Intentional Deviations

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-07-03 | `requirements/index.json` has `auto_sync_enabled=false`. | Planned validations reference future tests; enabling sync now could fake completion before implementation exists. | Re-enable when validation refs point at real tests. |
| 2026-07-03 | `scenario-auditor` listed disabled in dependencies. | It is a migration source, not a runtime dependency. | Remove after API rule migration ledger is complete. |

## Documentation Architecture

| Concern | Canonical Document |
|---|---|
| Product targets | `PRD.md` |
| Domain ownership | `docs/concepts/DOMAINS.md` |
| System architecture | `docs/concepts/ARCHITECTURE.md` |
| Workflows and states | `docs/concepts/FLOWS.md` |
| Data ownership | `docs/concepts/DATA.md` |
| Dependencies | `docs/concepts/INTEGRATIONS.md` |
| API endpoints | `docs/reference/api-endpoints.md` |
| CLI commands | `docs/reference/cli-commands.md` |
| Test seams | `docs/internal/SEAMS.md` |
| Known gaps | `docs/internal/PROBLEMS.md` |
| Progress log | `docs/internal/PROGRESS.md` |

## Cross-References

- [`START-HERE.md`](../START-HERE.md) — generated scenario orientation
- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — workflow and state-transition map
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues and deferred work
- [`../internal/PROGRESS.md`](../internal/PROGRESS.md) — lifecycle log
