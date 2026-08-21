# Problems

## Remaining Contract-Only Evidence

Target resolution, surface inventory, parse-unit discovery, analyzer brokering, generic fact normalization, cache persistence, proto adoption proof, and REST endpoint proof are implemented. `cli_proofs` and `ui_widget_proofs` still return typed `unsupported` evidence until later phases.

Future consumers should treat these families as explicit non-goals for the current Code Facts launch. `test-genie`, `cli-health`, `ui-health`, and `swarm-manager` can adopt the existing target/surface/import/reference/call/proto/endpoint facts now, but command-level and widget-level proof semantics still need their own product requirements and fixtures.

## Full Scaffold Health

Initial `make test` failed on generated scaffold drift: missing REST `proto_payloads`, React Router future warnings, duplicate navigation landmarks, Go formatting, and malformed template measure metadata. Phase 5 and Phase 6 removed those blockers for the active Code Facts surfaces.

## Provider Availability

Code Facts does not require live graph providers in unit tests; Phase 8 added fake-provider broker coverage and maps unavailable live providers to typed `unknown` warnings unless strict mode is requested. As of 2026-06-13, provider `unimplemented` and `workspace_unsupported` failures are also downgraded to typed `unsupported` warnings in non-strict mode. Explicit integration tests against running `go-code-graph` and `typescript-code-graph` are still needed before consumers depend on live provider availability.

Phase 13 manually validated the live provider path against `proto-health` with `code-facts facts describe scenario:proto-health --include all --no-cache --json`. The TypeScript provider currently returns typed `typescript-code-graph.type_check_failure` warnings with `[object Object]` messages for many generated UI files; Code Facts preserves those as `unknown` warnings and `proto-health` filters noisy non-proof diagnostics. A future TypeScript Code Graph hardening pass should make those diagnostics human-readable.

## Proof Synthesis Limits

Endpoint proof synthesis is deliberately static and conservative. Endpoint route proof becomes `proven` only when graph facts expose route path/method attributes supported by a Code Facts framework adapter; otherwise Code Facts reports `unknown` or `missing` rather than inferring success from endpoint metadata alone.

As of 2026-06-13, `go-code-graph` emits `go_route_registration` facts for supported static Go route registrations, and `typescript-code-graph` emits generic Express route-registration facts for literal Express `app.METHOD` and `router.METHOD` calls. Code Facts maps both into `FACT_FAMILY_CALLS` and proves endpoints through the `go.http` and `ts.express` adapters. Remaining route-proof unknowns should now mean either the provider was unavailable, the route shape is dynamic/unsupported, no adapter supports the framework, or the endpoint declaration does not match static route evidence.

Express payload proof is intentionally partial. Import-only evidence is `unknown`, wrong generated payload usage is `contradicted`, and missing handler-local payload usage is `missing` for supported static routes.

## Deferred Consumer Adoption

Phase 13 intentionally did not migrate additional consumers. Follow-up candidates:

- `architecture-cartographer`: reuse Code Facts target discovery and provider brokering rather than duplicating graph-provider routing.
- `test-genie`: use Code Facts for declared endpoint, CLI, and UI implementation evidence once command/widget proof families are designed.
- `cli-health` and `ui-health`: consume Code Facts for discoverability and generated-client usage proof.
- `swarm-manager`: consume Code Facts surface and capability inventory for higher-level ecosystem planning.

## Work ladder

- Rung: W3
- Evidence: The public Search handler now serves active generation `phase11-compact-v5` exclusively through the persistent SQLite catalog and FTS index. A source-free regression proves queries continue after the source tree is removed. The generation contains 53,409 classified files and 231,719 search documents in 1,588,555,776 bytes; build RSS high-water was 99.97 MiB. Exact and hybrid current/three-times performance gates pass. Both registered Search Hub suites passed 15/15 reviewed cases, and federation status reports both leaves reachable, ready, fresh, and on the active generation.
- Freshness evidence: ordinary file changes are polled every five seconds, deduplicated against catalog size/modification metadata and then content hash, and atomically replace catalog plus FTS rows. The regression measurement completes below the 15-second budget and proves cache invalidation, stale suppression, and deletion. A five-minute revision-delta and path-inventory audit repairs clean committed changes, untracked deletions, and dirty-path reverts without re-reading the complete corpus; a missing checkpoint retains a full correctness fallback. Policy/model/schema changes retain isolated shadow promotion and rollback.
- Measured: 2026-08-21
