# Testing — Proto Health

How to verify the scenario's current behavior. Tests should exercise the
owning API, CLI, or UI boundary and use deterministic fixtures for external
repo, descriptor, Code Facts, and baseline inputs.

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

## Current owner test surfaces

| Area | Owner tests | Behavior covered |
|---|---|---|
| Proto surface | `api/internal/protosurface/loader_test.go` | Descriptor image parsing, service/message/field traversal, scenario scoping, and deterministic ordering. |
| Validation | `api/internal/validation/service_test.go`, `gen_sync_test.go`, `metrics_test.go` | Stable findings, generated artifact drift, Code Facts degraded evidence, and bounded checks. |
| API boundary | `api/handlers/validation/connect_handler_test.go`, `handlers/health/handler_test.go` | Generated Connect contracts, request validation, and dependency health. |
| Impact | `api/internal/impact/service_test.go` | Baseline resolution, compatibility parsing, and consumer reconciliation. |
| CLI | `cli/domains/validate/handlers_test.go`, `cli/domains/domains_test.go` | Manifest dispatch, local owner validation, and command group coverage. |
| UI | `ui/src/features/proto-health/ProtoHealthPanel.test.tsx` | Loading, error, empty, and populated inspection states. |

The generated `notes` sample and its tests were removed. Historical tutorial
material is not evidence of a live endpoint or active test surface.

## Running focused suites

```bash
(cd api && GOWORK=off go test ./... -count=1)
(cd cli && GOWORK=off go test ./... -count=1)
(cd ui && pnpm exec vitest run src/features/proto-health/ProtoHealthPanel.test.tsx)
```

For scenario-wide readiness, use the supported Test Genie phases through
`vrooli scenario test proto-health`; a passing focused suite is not a full
scenario readiness receipt.
