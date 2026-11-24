# Flows — Measures Health

This document records ordered workflows with meaningful state or side effects. Plain request/response reads are not modeled as state machines.

## Flow inventory

| Flow | Trigger | Outcome | Statefulness | Evidence |
|---|---|---|---|---|
| Validate a scenario | CLI/UI/API validation request, optionally with `--probe` | Coverage assessment and findings; the optional probe calls declared measure endpoints | A completed validation is recorded in `validation_runs` for the measures owned by this scenario | `api/internal/validation/service_test.go`, `api/internal/runhistory/repository.go`, `cli/domains/validate/handlers.go` |
| Search declared measures | Search Hub or CLI query | A matching measure is executed and its result returned with source information | Request-scoped; no query history is persisted here | `api/internal/measureindex/provider_test.go`, `api/handlers/search/connect_handler_test.go` |

## Validation and probe

The validation service discovers scenario manifests, grades declared measure coverage, and can optionally make behavioral calls to the declared endpoints. The CLI's `validate scenario <name> --probe` sets `IncludeExecution`; without the flag, validation does not run those calls. `MeasuresService` stores this scenario's own validation result in `validation_runs`, which backs the `measures failed` and `measures coverage` queries.

Probe failure is reported as an assessment result; it does not change the owner's manifest or the target scenario. Focused source tests live beside the validation service and CLI handlers.

## Measure search

The search provider harvests measures from scenario manifests, matches a natural-language question to a declaration, then executes the selected measure through its endpoint. Search status reports provider/index availability. The request does not create durable state.

## Cross-references

- [`DOMAINS.md`](DOMAINS.md) — domain responsibilities
- [`DATA.md`](DATA.md) — persisted validation run history
- [`../internal/TESTING.md`](../internal/TESTING.md) — owner test entry points
