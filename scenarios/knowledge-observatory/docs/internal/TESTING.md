# Testing

## Test Layers

| Layer | Command |
|---|---|
| API unit/integration | `cd api && go test ./...` |
| CLI | `cd cli && go test ./...` |
| Scenario | `vrooli scenario test knowledge-observatory` |

## Critical Coverage

Contract parsing, alias resolution, health validation, append/reset log
operations, template endpoints, viewer behavior, explorer annotations, and
healing/autofix prompts must stay covered.

## Fixtures

Prefer temp scenarios with explicit manifests for package tests. Do not mutate
real scenarios from unit tests.
