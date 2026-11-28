# Automation engine

This package owns typed workflow execution behind the API. Its live request path and component boundaries are documented in the [execution architecture](../../docs/architecture/execution.md).

## Package map

- `contracts/` — versioned plans, instructions, outcomes, capabilities, telemetry, and event envelopes.
- `compiler/` — converts saved workflows into the shared action contract.
- `executor/` — validates plans, checks capabilities, runs instructions and graphs, applies retries/timeouts, and records events and outcomes.
- `engine/` — engine/session interfaces and the Playwright adapter for the local driver.
- `session/` — browser session admission, reuse, isolation, and lifecycle management.
- `execution-writer/` — result/timeline persistence and bounded artifact shaping.
- `events/` — per-execution sequencing, queue limits, WebSocket delivery, and an in-memory test sink.

The current engine selection is Playwright. Keep action semantics in the shared contracts and executor; the engine and driver implement browser operations without owning workflow status or persistence.

## Where to work

- [Executor guide](executor/README.md) for plan construction and flow-control code locations.
- [Engine guide](engine/README.md) for the driver adapter and session contract.
- [Event guide](events/README.md) for ordering and backpressure behavior.
- [Execution writer guide](execution-writer/README.md) for durable results and artifacts.

Tests are colocated with each owner. The execution package's contract and integration tests cover normalized outcomes, persistence, and event behavior; scenario-level phases are defined in the scenario's Test Genie configuration.
