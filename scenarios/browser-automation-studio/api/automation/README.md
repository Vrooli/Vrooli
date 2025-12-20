# Automation Engine

This folder hosts the engine-agnostic automation stack. It provides clear seams (engine / executor / recorder / events / contracts) so automation engines can plug in without changing API or replay contracts.

## Architecture at a Glance
```mermaid
flowchart LR
    subgraph Request["WorkflowService / handlers"]
        WF["Workflow execution request\n(flow + params)"]
    end
    subgraph Planning["automation/executor"]
        PB["plan_builder.go\ncompile workflow → ExecutionPlan + CompiledInstructions"]
        EX["SimpleExecutor\n(retries, loops, heartbeats, event emission)"]
    end
    subgraph Engine["automation/engine"]
        SEL["selection.go\nENV: ENGINE/ENGINE_OVERRIDE"]
        ENG["AutomationEngine (PlaywrightEngine)\nStartSession/Run/Reset/Close"]
    end
    subgraph Writer["automation/execution-writer"]
        REC["FileWriter\nstep rows, artifacts, screenshots, DOM\ntruncation/dedupe"]
    end
    subgraph Events["automation/events"]
        SEQ["Sequencer\nper-execution ordering + backpressure limits"]
        WS["WSHubSink → websocket hub"]
        MEM["MemorySink\n(tests)"]
    end

    WF --> PB --> EX --> ENG
    ENG -->|Run -> StepOutcome| EX
    EX -->|RecordStepOutcome| REC
    EX -->|EventEnvelope| SEQ --> WS
    EX -->|Telemetry| SEQ
    REC -->|Artifacts/IDs| EX
    SEQ --> MEM
```

Key invariants:
- Contracts own schema/versioning and size limits; engines cannot add vendor-specific fields.
- Executors own orchestration/heartbeats/telemetry; engines run a single instruction and return a normalized `StepOutcome`.
- Recorder generates durable IDs/dedupe keys and shapes artifacts so UI/replay stay stable.
- Event sinks enforce ordering/backpressure; completion/failure events must never drop.

## Layout
- `contracts/` — Stable payload shapes (`StepOutcome`, telemetry, capabilities, event envelopes) plus size limits and schema versions. Keep these backward-compatible; bump versions when shapes change.
- `engine/` — `AutomationEngine` interface, env-based selection, and static factory. The `PlaywrightEngine` drives a local Playwright driver over HTTP, supporting downloads/uploads/tracing/video for desktop bundles.
- `executor/` — Orchestration (`SimpleExecutor`) that drives engines, emits heartbeats/telemetry, and enforces capability checks. `plan_builder.go` compiles workflows into the contract plan shape.
- `execution-writer/` — Persists normalized outcomes/telemetry (`FileWriter`) while owning IDs/dedupe and storage uploads.
- `events/` — Event sinks and sequencing/backpressure helpers. `ws_sink` bridges to the websocket hub; `memory_sink` supports tests.
- See also:
  - [contracts/README.md](contracts/README.md)
  - [engine/README.md](engine/README.md)
  - [executor/README.md](executor/README.md)
  - [recorder/README.md](recorder/README.md)
  - [events/README.md](events/README.md)

### Engine Configuration
- Runtime execution flows through `automation/executor` + `PlaywrightEngine` + `FileWriter` + `WSHubSink`.
- Only `subflow` nodes are supported for child workflow execution.
- Playwright (local driver) is the only supported engine; select via `ENGINE` / `ENGINE_OVERRIDE` env vars (defaults to `playwright`).

### Flow navigation (where complex orchestration lives)
- Planning/compilation: `executor/plan_builder.go`
- Graph + branching + loop execution: `executor/flow_executor.go` (helpers in `executor/flow_utils.go`)
- Loop coverage: repeat + forEach + while (basic variable condition); built-in `set_variable` node for executor-scoped vars; `${var}` interpolation on instruction params.
- Capability preflight (tabs/iframe/upload/HAR/video/download/viewport): `executor/preflight.go`
- Retry/heartbeat/normalization shell: `executor/simple_executor.go`
- Future variable interpolation/cancellation/session policy should also land in `executor/` alongside these files so flow logic stays discoverable.

## Environment Variables
- `ENGINE` sets the default engine (defaults to `playwright`).
- `ENGINE_OVERRIDE` forces all executions to use a specific engine.
- `PLAYWRIGHT_DRIVER_URL` points at the local driver (defaults to `http://127.0.0.1:39400` for dev).

## Current Coverage
- Covered: linear + graph execution (repeat/forEach/while loops with executor-owned `set_variable`), `${var}` interpolation, heartbeats, retries, capability preflight, DB persistence of step outcomes/console/network/assert/assertion/screenshot artifacts, websocket event emission, clean reuse mode (session reset between steps).
- Not yet covered (needs implementation/parity work):
  - Rich variable expressions/interpolation beyond simple replacements.
  - Session reuse policies: true `fresh` (per-step session spin-up) and policy-driven reset/failure recovery.
  - Cancellation/timeout taxonomy and propagation (executor now enforces `timeoutMs` per step and `executionTimeoutMs` at the plan level; still need workflow-level status/artifact/event semantics).
  - Retry/failure taxonomy alignment (failure kinds/messages, retryable flags) and telemetry parity/drop metrics.
  - Capability enforcement matrix (tabs/iframes/HAR/tracing/uploads/downloads/video/viewport) with clear fail-fast gaps.
  - Artifact shaping: DOM truncation/dedupe, cursor trails/timeline framing payloads, screenshot handling parity, backpressure/drop counters.
  - Crash handling/recovery markers.

## Testing
- Unit tests live alongside packages (`contracts_test.go`, `selection_test.go`, `simple_executor_test.go`, etc.).
- Integration tests should use `testcontainers-go` + `FileWriter` + `MemorySink` to exercise real persistence and event sequencing.
- Grow executor/recorder contract tests (artifact shaping, capability gaps, telemetry/drop counters).
- Codify capability matrix + artifact shaping and add targeted contract tests for truncation/dedupe/backpressure.
## What to Tackle Next
1) Executor parity: fill gaps for variable interpolation, retry taxonomy, reuse/clean/fresh, cancellation/timeout, cursor trails/timeline framing, DOM truncation/dedupe.
2) Capability matrix: codify workflow feature → capability requirements (HAR, multi-tab, upload/download) and fail fast when unsupported.
3) Artifact shaping/backpressure: add truncation/dedupe/drop-counter coverage so recorder + events stand independently.
4) Playwright: keep expanding `playwright-driver` instruction coverage (downloads, tracing, assertions beyond text) and enable replay capture client when bundling for desktop.
5) Desktop artifact storage: recorder now stores execution-level video/trace/HAR paths (inlines small files) for Playwright runs via session close metadata, but there is no generic storage API for large non-screenshot artifacts. To fully support desktop exports/replays, add storage/upload for HAR/trace/video artifacts (MinIO/local), persist URLs in artifacts, and surface them in export/UI/CLI.
