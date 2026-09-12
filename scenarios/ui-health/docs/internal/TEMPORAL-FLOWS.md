# Temporal Flows

## Flow Index

| Flow ID | Domain | Risk | Model Status | Source of Truth | Tests | Remaining Gaps |
|---|---|---|---|---|---|---|
| runtime-experience-surface-lifecycle | `api/internal/uiruntime` | Medium | Event-observed terminal gate | path:scenarios/ui-health/api/internal/uiruntime/workflow.go | path:scenarios/ui-health/api/internal/uiruntime/workflow_test.go | Required declared surfaces are watched through a same-origin `MutationObserver` and BAS waits for terminal lifecycle evidence before artifact capture. The current trace does not retain a distinct pre-settlement artifact. |

## Audit Notes

- 2026-07-17: The runtime workflow has a bounded bridge-ready assertion and one artifact capture. A fixed-delay second capture would be arbitrary and is intentionally not treated as lifecycle evidence.
- 2026-07-17: Declared required regions now install a same-origin `MutationObserver` in the iframe host. It mirrors settled terminal states to `[data-smoke-experience-settled]`, which BAS asserts before artifact capture. Loading is excluded from terminal state expectations and undeclared pages retain handshake-only behavior. A future enhancement may retain a separate pre-settlement artifact when an application actually exposes loading.
