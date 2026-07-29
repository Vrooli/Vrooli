# Domains — Vrooli Onboarding

## Purpose Of This Document

This inventory names the product capabilities that own onboarding behavior and
the evidence used by the requirements registry. It is intentionally scenario-
first: resources, credentials, and operating choices are derived from manifests
and persisted operator state rather than treated as separate configuration
products.

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| onboarding-read-model | Derive selectable scenarios, resources, host requirements, credential metadata, and readiness from manifests. | reporting | No durable data; assembles manifest and credential-authority metadata. | API, UI | REQ-P0-001, REQ-P0-002, REQ-P0-003, REQ-P1-002, REQ-P2-001, REQ-P2-003 | `api/v2_read_model.go`, `api/v2_read_model_test.go`, `api/v2_readiness.go`, `ui/src/components/wizard/StepSelectScenarios.tsx`, `ui/src/components/wizard/StepDerivedResources.tsx`, `ui/src/components/wizard/StepReadiness.tsx` |
| operator-state | Commit and re-enter operator choices atomically without using disposable wizard navigation or generated configuration as authority. | mutation | `.vrooli/operator-state.json` through the API-owned file boundary. | API, UI | REQ-P0-004, REQ-P0-005, REQ-P0-006, REQ-P1-001 | `api/operator_state.go`, `api/operator_state_test.go`, `ui/src/hooks/useWizardState.ts`, `ui/src/hooks/useWizardState.test.tsx` |
| onboarding-experience | Guide the operator through scenario-first setup, explicit deferred integrations, operating choices, validation, and accessible completion. | orchestration | Browser-local navigation only; it is not operator authority. | UI | REQ-P1-003, REQ-P1-004, REQ-P2-002 | `ui/src/App.tsx`, `ui/src/App.accessibility.test.tsx`, `ui/src/components/wizard/` |

## Domain Details

The read model is a pure manifest-plus-authority projection. Operator state is
the only durable configuration written by this scenario. The experience domain
renders and commits those decisions, but its browser navigation is disposable.

## Shared Concepts

Logical credential identities and deployment-tier provider policy are shared
control-plane concepts. They are surfaced as status metadata only; values cross
the boundary solely through stdin provisioning and scoped runtime injection.

## Non-Domains

- `api/main.go` and `api/helpers_test.go` are HTTP composition and test support.
- `ui/src/lib/` is transport support; it does not own onboarding decisions.
- `ui/src/components/ui/` and `ui/src/test-utils.tsx` are shared presentation
  and testing primitives.
- `cli/` is a thin operator surface and does not own onboarding state.

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| integrations | Integration Hub, connection binding, and OAuth are explicitly out of scope for V2. | Integration Hub is available as a real dependency. |

## Cross-References

- [`../WIZARD_FLOW.md`](../WIZARD_FLOW.md)
- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`FLOWS.md`](FLOWS.md)
- [`DATA.md`](DATA.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
- [`../../PRD.md`](../../PRD.md)
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md)
