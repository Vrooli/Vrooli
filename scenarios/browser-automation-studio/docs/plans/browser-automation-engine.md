# Browser Automation Engine Refactor Plan

Context: The API execution path is tightly coupled to Browserless (see `api/browserless/client.go`), making new features (desktop/Electron bundles, alternative providers) hard to add. This plan introduces an engine abstraction and refactors orchestration/persistence to support multiple implementations (Browserless today, Desktop/Playwright next) without changing workflow JSON or replay consumers.

## Goals
- Isolate engine concerns from orchestration, persistence, and event streaming.
- Maintain current behavior/artifacts while creating a seam for a `DesktopEngine` (Playwright/puppeteer-core) and future providers.
- Keep UI/CLI/replay/timeline contracts stable.
- Surface engine capabilities so scenario-dependency-analyzer and scenario-to-desktop can pick the right implementation.

## Constraints & Compatibility
- No breaking changes to REST/WebSocket contracts, artifact shapes, or timeline/export payloads.
- Preserve screenshot/DOM snapshot handling, console/network telemetry, retry/backoff semantics, branching/loop execution, and MinIO/local storage behavior.
- Avoid new dependencies unless explicitly approved (use existing deps for Browserless; plan DesktopEngine with Playwright/puppeteer-core but keep optional until approved).

## Proposed Architecture
- `AutomationEngine` interface (engine-agnostic):
  - `StartSession(opts) -> EngineSession`
  - `EngineSession.Run(instruction) -> StepOutcome`
  - `EngineSession.Reset()` (clean state) and `EngineSession.Close()`
  - Capability descriptor (requiresDocker, maxConcurrentSessions, supportsHar, etc.) for analyzer/packaging.
- `Executor` (orchestration only):
  - Compile plan → instructions, manage retries/backoff, variable interpolation, branching/loops, progress, and cancellation.
  - Depends on `AutomationEngine`, `ArtifactRecorder`, and `EventSink`—no Browserless/storage code inside.
- `ArtifactRecorder`:
  - Persist ExecutionStep, ExecutionArtifact, screenshots (MinIO/local fallback), logs, timeline frames, cursor trails, low-information screenshot dedupe, DOM snapshots, assertions, extracted data.
  - Owns mapping from `StepOutcome` → DB records to keep engines compatible.
- `EventSink`:
  - Emits WebSocket events (`execution.*`, `step.*`, `step.telemetry`, `step.screenshot`, `step.heartbeat`) using recorder IDs so payloads remain stable.
- Engine implementations:
  - `BrowserlessEngine`: thin adapter around existing CDP/session logic and session strategy (`reuse/clean/fresh`), implements `AutomationEngine`.
  - `DesktopEngine` (next): uses Playwright/puppeteer-core with bundled Chromium, concurrency=1, produces identical `StepOutcome` artifacts (screenshots, console, network, DOM snapshot, assertions).
- Engine registry/factory:
  - Select via config/env (`ENGINE=browserless|desktop`), expose capability metadata for scenario-dependency-analyzer and scenario-to-desktop swaps.

## Migration Steps
1) **Define shared contracts**: Extract neutral types for ExecutionPlan/Instruction/StepOutcome/artifact payloads into an engine-agnostic package; add golden tests for WebSocket/timeline/artifact shapes.
2) **Introduce interfaces**: Add `AutomationEngine`, `EngineSession`, `ArtifactRecorder`, `EventSink` interfaces; wire `WorkflowService` to accept engine + executor instead of concrete Browserless client.
3) **Extract executor**: Move orchestration logic (retries, branching, loops, variable interpolation, session reuse policy) out of `browserless/client.go` into a new executor package.
4) **Extract recorder**: Move persistence/artifact shaping (screenshots, telemetry, DOM snapshot, assertion, retry metadata, timeline frames) into a recorder module reused by all engines.
5) **Wrap Browserless**: Implement `BrowserlessEngine` using existing CDP adapter; keep session strategy envs; ensure parity with current outputs via tests.
6) **Wire events**: Move heartbeat/telemetry emission to executor layer; ensure payloads unchanged (golden tests).
7) **DesktopEngine spike**: Stub Playwright-based engine that returns matching `StepOutcome` structure; prove single-workflow run with screenshots/console/network/DOM snapshot; keep dependency optional until approved.
8) **Capabilities surfaced**: Add capability metadata to config for scenario-dependency-analyzer to choose `browserless@ubuntu` vs `desktop-engine@electron`; include max concurrency and requiresDocker flags.

## Testing & Validation
- Unit: executor branching/retries/loop directives against a fake engine; recorder golden tests for DB artifacts/timeline/event payloads.
- Integration: smoke run with Browserless engine to confirm parity with current behavior.
- Contract: define and lock replay/export JSON schemas so DesktopEngine must satisfy the same contract.

## Risks & Mitigations
- Risk: Silent drift in artifact/event payloads → Mitigate with golden/contract tests and recorder centralization.
- Risk: Session strategy regressions → Keep current env flags, test reuse/clean/fresh paths in BrowserlessEngine adapter.
- Risk: DesktopEngine feature gaps → Prioritize matching telemetry (console/network/DOM snapshot) and screenshot parity before advanced motion features; keep concurrency=1 explicitly.
