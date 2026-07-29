# API Phase

The `api` phase delegates API readiness validation to **api-health** through the shared `ScenarioValidationService`.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton, so a doc-search topic emitted in a run's scorecard resolves to the exact remediation section.

## North Star

The scenario's API boots cleanly under the Vrooli lifecycle: `.vrooli/service.json` declares standard health metadata, preflight runs in the right order, and startup is on `api-core/server.Run`. It exposes a schema-valid `/health` surface that passes a live readiness probe. Its HTTP responses are unambiguous — correct status codes, explicit content types, versioned feature endpoints, and no error path that implicitly reports success. Its runtime is disciplined — bounded outbound HTTP, closed response bodies, propagated request context, cancellable long-running work, and structured logging. And every unambiguous finding has a deterministic fixer that previews safely and applies idempotently. At maximum maturity every capability ladder is at **L3 Clean / Complete**, so API readiness has no required provider-owned findings.

## The rungs and their gates

api-health reports a ladder per capability (`target_resolution`, `api_lifecycle_contract`, `health_runtime_contract`, `http_response_contract`, `runtime_hygiene`, `autofix_readiness`). The rungs are monotone — each implies the one below — and the phase-level ladder (L0 unavailable → L1 inspectable → L2 baseline enforced → L3 clean) is the shared shape of these capability ladders.

| Rung | Gate (what it means) | Next unlock |
|---|---|---|
| L0 Unavailable | The target scenario, its lifecycle metadata, health surface, routes, or runtime source cannot be inspected. | Resolve a readable target and classify API applicability (or return an honest no-API result). |
| L1 Foundation | Inputs are inspectable and applicability is known, but findings remain and no unevaluated target is reported clean. | Satisfy the provider-owned lifecycle, health, HTTP, runtime, and fixability checks. |
| L2 Ready | The provider-owned baseline is enforced (lifecycle compatible, health valid, HTTP consistent, runtime hygienic, fixers preview safely). | Resolve all remaining required provider-owned findings. |
| L3 Complete | No required provider-owned findings remain for the capability. | Maximum maturity reached. |

## What each finding means

Each finding caps the capability it names at the rung shown; only ERROR/BLOCKER severities fail the phase, so `WARNING`/`INFO` findings are honest, non-failing debt. Many are auto-fixable (`fix_class: auto`).

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `api_health.target_unresolved` | target_resolution | L1 | ERROR | Yes |
| `api_health.api_surface_absent` | target_resolution | L1 | INFO | No |
| `api_health.service_health_missing` | api_lifecycle_contract | L2 | ERROR (auto-fix) | Yes |
| `api_health.preflight_missing_or_late` / `api_health.server_runner_missing` | api_lifecycle_contract | L2 | ERROR | Yes |
| `api_health.health_endpoint_missing` / `api_health.health_probe_failed` | health_runtime_contract | L1 | ERROR | Yes |
| `api_health.health_schema_invalid` | health_runtime_contract | L2 | ERROR | Yes |
| `api_health.error_response_implicit_success` | http_response_contract | L2 | ERROR | Yes |
| `api_health.raw_status_code` / `api_health.content_type_missing` / `api_health.unversioned_feature_endpoint` | http_response_contract | L2 | WARNING | No |
| `api_health.http_client_unbounded` / `api_health.response_body_unclosed` | runtime_hygiene | L2 | ERROR | Yes |
| `api_health.request_context_dropped` / `api_health.goroutine_not_cancellable` / `api_health.unstructured_api_logging` | runtime_hygiene | L2 | WARNING | No |
| `api_health.autofix_contract_incomplete` | autofix_readiness | L1 | WARNING | No |

## The canonical fix

- **Target-resolution findings** → point Test Genie at a resolvable scenario; `api_health.api_surface_absent` is informational for scenarios that intentionally have no API.
- **Lifecycle findings** (`service_health_missing`, `preflight_missing_or_late`, `server_runner_missing`) → add standard `.vrooli/service.json` health metadata (auto-fixable), order preflight before serving, and migrate startup to `api-core/server.Run`. A `ListenAndServe` inside `server.Run`'s documented `StartServer` callback is sanctioned and must not be reported as a bypass.
- **Health-runtime findings** (`health_endpoint_missing`, `health_probe_failed`, `health_schema_invalid`) → declare a `/health` endpoint (auto-fixable), make it reachable under the lifecycle for the live probe, and return a schema-valid readiness payload with dependency status.
- **HTTP-response findings** (`raw_status_code`, `content_type_missing`, `error_response_implicit_success`, `unversioned_feature_endpoint`) → use named status helpers instead of raw codes (auto-fixable), set explicit content types (auto-fixable), return an error status+envelope on error paths, and version feature endpoints. The bare `/` Connect dispatcher mount and `/debug/pprof/` operations routes are transport/ops mounts, not unversioned feature endpoints.
- **Runtime-hygiene findings** (`http_client_unbounded`, `response_body_unclosed`, `request_context_dropped`, `goroutine_not_cancellable`, `unstructured_api_logging`) → bound outbound HTTP timeouts, close response bodies (auto-fixable), propagate the request context, make long-running goroutines cancellable, and use structured logging.
- **Autofix-readiness findings** (`autofix_contract_incomplete`) → declare fixability on every finding and implement deterministic fixers (provider implementation work).

Preview auto-fixes before applying with `api-health validate fix-preview <scenario>`. Load `prompt-manager skill read api-health` for the deep remediation guidance behind most of these codes.

## How to verify

```bash
# See the current rung, gaps, and next move for every API capability:
api-health validate scenario <scenario>

# Preview deterministic fixes without applying them:
api-health validate fix-preview <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases api
test-genie runs findings --scenario <scenario>
```

The `api` line in the scorecard shows the current rung, the single highest-unlock next move, and a runnable doc-search topic that resolves back to the sections above.

## What It Owns

API Health owns API-specific readiness:

- `.vrooli/service.json` API and health metadata
- API startup lifecycle and preflight wiring
- `/health` schema and optional live probe evidence
- route-aware HTTP response semantics
- API-runtime hygiene such as outbound HTTP timeouts, response body closure, request-context propagation, and cancellable long-running work
- deterministic API fix preview/apply metadata

Other standards-like checks live with their focused health providers rather than a catch-all Test Genie phase.

## How It Runs

Test Genie resolves `api-health` and calls its provider contract:

```bash
api-health validate scenario <scenario>
```

Execution mode is not requested by Test Genie by default, so the phase stays bounded and static unless API Health is run directly with execution enabled.

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip api
```

Disable per scenario via `.vrooli/testing.json`:

```json
{
  "phases": {
    "api": { "enabled": false }
  }
}
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json`:

```json
{
  "phases": {
    "api": { "timeout": "120s" }
  }
}
```

## Provider Detail

Run the provider directly for native API Health evidence and fix previews:

```bash
api-health validate scenario <scenario>
api-health validate fix-preview <scenario>
```
