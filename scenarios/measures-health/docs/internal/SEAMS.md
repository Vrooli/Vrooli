# Seams — Measures Health

A seam is a dependency boundary used by production wiring and tests. This register describes the boundaries that exist in this scenario today; proto messages and endpoint paths are contracts, not seams.

## API boundaries

| Boundary | Source and production wiring | Test substitution / evidence |
|---|---|---|
| Database reachability | `api/internal/database/pinger.go::Pinger`; the health handler receives the pinger through its module dependencies. Production's routed database and `*sql.DB` satisfy it. | `api-core/databasetest.FakePinger`; `api/handlers/health/handler_test.go`. |
| Scenario validation inputs | `api/internal/validation/service.go` defines manifest, domain, schema, scenario-list, and probe contracts. `handlers/validation/module.go` wires `NewFilesystemValidator` rooted at the repository and the live HTTP prober. | Validation service tests inject deterministic sources and a probe fake. See `api/internal/validation/service_test.go`, `domains_test.go`, and `substrate_test.go`. |
| Measure corpus inputs | `api/internal/measureindex/harvest.go` separates manifest reads, scenario listing, and schema lookup. Production uses `NewFilesystemHarvester(repoRoot)`. | `api/internal/measureindex/harvest_test.go` supplies fake sources and a schema source. |
| Measure execution and status | `api/internal/measureindex/provider.go::Config` accepts an executor, extractor/completer, clock, and Ollama-availability probe. Production resolves owner URLs through api-core discovery and uses the configured measure executor. | `api/internal/measureindex/provider_test.go` injects deterministic executor, extractor, clock, and availability functions. No external service is required by these tests. |
| Search matching | `api/internal/measureindex/matcher.go::LexicalMatcher` implements the matcher used by the shared measures engine. | `api/internal/measureindex/matcher_test.go` exercises ranking, ties, and empty queries. |
| Outbound HTTP interface | `api/internal/httpc/doer.go::Doer` is declared for a future scenario consumer. The source explicitly says no current production path uses it. | `api/internal/httpc/doer_test.go` verifies the interface contract. Do not report it as a live production dependency until a consumer is wired. |
| API module mounting | `api/internal/module/module.go` defines the module and endpoint metadata; `api/internal/modules/registry.go` collects schemas/endpoints and `api/internal/server/server.go` mounts modules. | `api/internal/modules/registry_test.go` and `api/internal/server/server_test.go` cover registration/mounting. |

## CLI boundary

`cli/manifest.json` declares commands, arguments, Connect bindings, and governance. `cli/domains/<domain>/register.go` binds generated RPC methods to handlers; `cliapp.RunContext` is supplied by cli-core. The manifest is validated by `cli/domains/domains_test.go`. Use cli-core's test context and an in-process API server for handler behavior tests; do not add a second local command registry.

## Wire contracts

Proto contracts are under `packages/proto/schemas/measures-health/v1/`. Generated Connect clients and handlers carry proto-typed requests and responses. `GET /health` is the operational REST probe; the current HTTP exception is `GET /health`; proto operations use Connect-RPC.

## Maintenance rule

When production substitutes an interface in a test, update this page with the owner, wiring point, and test fake. Remove an entry when its source boundary or production use is gone. Keep feature workflows in `FLOWS.md` and durable data ownership in `DATA.md`.
