# Problems

## Remaining Contract-Only Evidence

Target resolution, surface inventory, parse-unit discovery, analyzer brokering, generic fact normalization, cache persistence, proto adoption proof, and REST endpoint proof are implemented. `cli_proofs` and `ui_widget_proofs` still return typed `unsupported` evidence until later phases.

## Full Scaffold Health

Initial `make test` failed on generated scaffold drift: missing REST `proto_payloads`, React Router future warnings, duplicate navigation landmarks, Go formatting, and malformed template measure metadata. Phase 5 and Phase 6 removed those blockers for the active Code Facts surfaces.

## Provider Availability

Code Facts does not require live graph providers in unit tests; Phase 8 added fake-provider broker coverage and maps unavailable live providers to typed `unknown` warnings unless strict mode is requested. Explicit integration tests against running `go-code-graph` and `typescript-code-graph` are still needed before consumers depend on live provider availability.

## Proof Synthesis Limits

Phase 10 proof synthesis is deliberately static and conservative. Endpoint route proof becomes `proven` only when graph facts expose route path/method attributes or typed helper usage; otherwise Code Facts reports `unknown` or `missing` rather than inferring success from endpoint metadata alone.
