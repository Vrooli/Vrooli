# Problems

## Remaining Contract-Only Evidence

Target resolution, surface inventory, parse-unit discovery, analyzer brokering, generic fact normalization, cache persistence, proto adoption proof, and REST endpoint proof are implemented. `cli_proofs` and `ui_widget_proofs` still return typed `unsupported` evidence until later phases.

Future consumers should treat these families as explicit non-goals for the current Code Facts launch. `test-genie`, `cli-health`, `ui-health`, and `ecosystem-manager` can adopt the existing target/surface/import/reference/call/proto/endpoint facts now, but command-level and widget-level proof semantics still need their own product requirements and fixtures.

## Full Scaffold Health

Initial `make test` failed on generated scaffold drift: missing REST `proto_payloads`, React Router future warnings, duplicate navigation landmarks, Go formatting, and malformed template measure metadata. Phase 5 and Phase 6 removed those blockers for the active Code Facts surfaces.

## Provider Availability

Code Facts does not require live graph providers in unit tests; Phase 8 added fake-provider broker coverage and maps unavailable live providers to typed `unknown` warnings unless strict mode is requested. Explicit integration tests against running `go-code-graph` and `typescript-code-graph` are still needed before consumers depend on live provider availability.

Phase 13 manually validated the live provider path against `proto-health` with `code-facts facts describe scenario:proto-health --include all --no-cache --json`. The TypeScript provider currently returns typed `typescript-code-graph.type_check_failure` warnings with `[object Object]` messages for many generated UI files; Code Facts preserves those as `unknown` warnings and `proto-health` filters noisy non-proof diagnostics. A future TypeScript Code Graph hardening pass should make those diagnostics human-readable.

## Proof Synthesis Limits

Phase 10 proof synthesis is deliberately static and conservative. Endpoint route proof becomes `proven` only when graph facts expose route path/method attributes or typed helper usage; otherwise Code Facts reports `unknown` or `missing` rather than inferring success from endpoint metadata alone.

The first consumer integration still reports `unknown` route implementation proof for `proto-health` REST exceptions `health` and `notes_attach`; payload writer and error-envelope evidence is present, but route registration proof lacks route path/method attributes in the current graph facts. This is acceptable for launch because `proto-health` maps it to warnings, not silent success.

## Deferred Consumer Adoption

Phase 13 intentionally did not migrate additional consumers. Follow-up candidates:

- `architecture-cartographer`: reuse Code Facts target discovery and provider brokering rather than duplicating graph-provider routing.
- `test-genie`: use Code Facts for declared endpoint, CLI, and UI implementation evidence once command/widget proof families are designed.
- `cli-health` and `ui-health`: consume Code Facts for discoverability and generated-client usage proof.
- `ecosystem-manager`: consume Code Facts surface and capability inventory for higher-level ecosystem planning.
