# Workflow executor

The executor owns plan execution and flow control. See the [execution architecture](../../../docs/architecture/execution.md) for the request path and its boundaries.

## Code map

- `plan_builder.go` — compiler interface and contract-plan construction.
- `simple_executor.go` — validation, capability preflight, execution context, retries, heartbeats, normalized outcomes, and event/writer coordination.
- `flow_executor.go` — graph traversal, branch routing, loops, variable mutation/interpolation, and subflows.
- `flow_utils.go` — graph and flow-state helpers.
- `loop_registry.go` — the supported loop handler registry.
- `preflight.go` — derive required engine capabilities from a plan.
- `session_lifecycle.go` — resolve reuse policy and track execution navigation boundaries.

Extend these owners with focused tests beside the behavior. Engine adapters should execute one compiled instruction; they must not acquire graph policy or persist workflow status.
