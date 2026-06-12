# Progress

## 2026-06-12 — Phase 5 Documentation First

- Generated `code-facts` from the `react-vite` template with `vrooli-default`.
- Added REST exception `proto_payloads` metadata required by the current endpoint schema.
- Fixed generated UI test/build issues caused by React Router future warnings and duplicate navigation landmarks.
- Fixed generated Go formatting issues and waived the template notes measure while keeping the template command binding.
- Replaced placeholder PRD and starter requirements with Code Facts P0/P1 traceability.
- Authored initial architecture, domain, data, integration, reference, seam, invariant, problem, decision, and progress docs.

Next phase: Phase 6 proto/API/CLI core.

## 2026-06-12 — Phase 6 Proto/API/CLI Core

- Added `packages/proto/schemas/code-facts/v1/facts/facts.proto` with `CodeFactsService`, target/fact-family enums, evidence status, surfaces, parse units, proof reports, and cache metadata.
- Added `packages/proto/schemas/code-facts/v1/shared/errors.proto` for REST-exception error envelopes.
- Replaced the generated notes API/CLI/proto surface with the real `facts` and `cache` command/API surface.
- Added a minimal `/facts` UI workbench backed by the generated Code Facts client.
- Removed the template notes API, CLI, UI, and proto files.

Next phase: Phase 7 target resolver, surface inventory, and parse unit discovery.

## 2026-06-12 — Phase 7 Target Resolver And Surface Inventory

- Replaced Phase 6 placeholder surface/parse-unit output with filesystem-backed target resolution.
- Added `scenario:<name>` and repo-root-aware scenario resolution, plus scenario detection for paths inside `scenarios/<name>`.
- Added bounded parse-unit discovery for Go modules, TypeScript projects, unsupported Node packages, and unknown code directories.
- Added scenario surface inventory for API, CLI, UI, optional sidecar/worker/job directories, service CLI adapter evidence, endpoint metadata, and CLI manifest metadata.
- Added API unit coverage for generic Go, generic TypeScript, generated API/CLI/UI scenario surfaces, unknown sidecar status, and paths inside scenario surfaces.

Next phase: Phase 8 analyzer brokering and generic fact normalization.

## 2026-06-12 — Phase 8 Analyzer Broker And Generic Fact Normalization

- Added graph-provider seams for Go and TypeScript parse units with production Connect clients for `go-code-graph` and `typescript-code-graph`.
- Added broker routing from discovered parse units to language providers, with provider-unavailable conditions reported as typed `unknown` warnings unless the caller requests strict mode.
- Added generic graph normalization for provider nodes into Code Facts facts: imports, symbols, references, calls, and parse-unit file/module/package facts.
- Extended `GenericFact` with a generic `attributes` map so provider metadata such as import paths, callees, source modules, enclosing declarations, and source ranges is preserved without Go/TS-specific Code Facts fields.
- Added API unit coverage for Go fact normalization, TypeScript fact normalization, provider unavailable handling, strict provider errors, unsupported parse units, and include filtering.

Next phase: Phase 9 cache and performance layer.

## 2026-06-12 — Phase 9 Cache And Performance Layer

- Added cache metadata fields for state, reason, scope, analyzer/schema/provider versions, source/config hashes, timestamps, and hit counts.
- Added `InspectCache` plus CLI `cache inspect` alongside status and clear.
- Added SQLite-backed cache storage for production and an in-memory repository for unit tests.
- Cached graph-provider payloads and synthesized describe reports separately.
- Added source/config fingerprinting so source, config, provider-version, schema-version, and include-option changes deterministically miss old cache entries.
- Added API unit coverage for repeated describe reuse, source-hash invalidation, cache status entries, and dry-run clear behavior.
- Updated the UI workbench to show cache state, key, source hash, and config hash.

Next phase: Phase 10 proto adoption and endpoint proof facts.

## 2026-06-12 — Phase 10 Proto Adoption And Endpoint Proof Facts

- Added proof synthesis for generated proto adoption across API, CLI, and UI surfaces from normalized import facts.
- Added static REST exception endpoint proof synthesis from `.vrooli/endpoints.json`, route metadata when present, and typed helper/call evidence such as `WriteProto` and `WriteError`.
- Added conservative status handling for `proven`, `missing`, `contradicted`, `unsupported`, and `unknown` proof outcomes.
- Wired `DescribeCodeFacts`, `CheckProtoAdoption`, and `CheckEndpointProof` through the same analyzer-backed proof path without making graph providers policy-aware.
- Added API unit coverage for generated proto import adoption, missing adoption, valid REST payload proof, and wrong response type contradiction.

Next phase: Phase 11 operator UI.
