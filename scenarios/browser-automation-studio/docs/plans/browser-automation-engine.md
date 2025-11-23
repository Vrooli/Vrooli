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
- Add explicit schema/version fields so replay/export clients can reject or adapt to changes instead of silently drifting.

## Proposed Architecture
- `AutomationEngine` interface (engine-agnostic):
  - `StartSession(opts) -> EngineSession`
  - `EngineSession.Run(instruction) -> StepOutcome` where `StepOutcome` is a normalized payload (ids, attempt, screenshot/meta, console logs, network entries, DOM snapshot/html, assertions, timing, failure kind) with explicit nullability so engines cannot add vendor-specific fields.
  - `EngineSession.Reset()` (clean state) and `EngineSession.Close()`
  - Capability descriptor (requiresDocker, requiresXvfb, maxConcurrentSessions, supportsHar/video/iframe, allowsParallelTabs) for analyzer/packaging; validate at boot that implementation matches declared capability.
- `Executor` (orchestration only):
  - Compile plan → instructions, manage retries/backoff, variable interpolation, branching/loops, progress, and cancellation.
  - Depends on `AutomationEngine`, `ArtifactRecorder`, and `EventSink`—no Browserless/storage code inside.
  - Owns cancellation/timeout policy and mapping from failure taxonomy → retry/abort semantics (infra vs engine vs user/workflow error); owns interrupt protocol (engine cancel vs page.close) and ensures idempotent close.
- Streaming + cancellation: executor drives heartbeats and per-attempt telemetry emissions (console/network/assertion/retry metadata) via `EventSink` even while a step is inflight; engine only exposes step-level output and optional in-flight progress hooks, not storage or WebSocket details.
- `ArtifactRecorder`:
  - Persist ExecutionStep, ExecutionArtifact, screenshots (MinIO/local fallback), logs, timeline frames, cursor trails, low-information screenshot dedupe, DOM snapshots, assertions, extracted data.
  - Owns mapping from `StepOutcome` → DB records to keep engines compatible; recorder generates IDs/dedupe keys (not the engine) so storage layout stays stable; recorder owns truncation/size limits and timestamp/clock normalization.
  - Holds cross-step state needed for parity (cursor trail accumulation, last replay screenshot reuse, DOM snapshot previews) so engines stay stateless and replaceable; executor only decides *when* to persist vs skip partials, recorder decides *how* to store.
- `EventSink`:
  - Emits WebSocket events (`execution.*`, `step.*`, `step.telemetry`, `step.screenshot`, `step.heartbeat`) using recorder IDs so payloads remain stable.
  - Defines ordering/backpressure rules: per-step attempt ordering = heartbeat → telemetry → screenshot → completion; queue/drop/backoff strategy so slow clients cannot block engines and completion/failure events are never dropped.
- Engine implementations:
  - `BrowserlessEngine`: thin adapter around existing CDP/session logic and session strategy (`reuse/clean/fresh`), implements `AutomationEngine`.
  - `DesktopEngine` (next): uses Playwright/puppeteer-core with bundled Chromium, concurrency=1, produces identical `StepOutcome` artifacts (screenshots, console, network, DOM snapshot, assertions).
- Engine registry/factory:
  - Select via config/env (`ENGINE=browserless|desktop`) and allow per-execution override (workflow flag or trigger payload) with safe default to Browserless.
  - Expose capability metadata for scenario-dependency-analyzer and scenario-to-desktop swaps without leaking engine choice into handlers (inject via `WorkflowService` constructor, not global state).

## Contracts to Codify
- **Failure taxonomy**: engine (navigation/selector), infra (provider unavailable), orchestration (bad workflow), cancellation/timeout. Executor maps each to retry/backoff vs hard failure; recorder persists failure kind consistently.
- **Schema/versioning**: StepOutcome/Event/Artifact include `schemaVersion`/`payloadVersion`; define migration/backfill policy and rejection behavior for unknown versions to avoid silent drift.
- **Session lifecycle**: define when sessions open/close on retries, mapping of reuse/clean/fresh → `StartSession`/`Reset`, idle TTL cleanup, and crash recovery if engine dies mid-step (detect, rehydrate, or mark as failed with clear taxonomy).
- **StepOutcome schema**: stable fields with types/nullability, clock source for timing, deterministic ID generation rules (per execution/step/attempt), and guarantees on screenshot/DOM/console/network shapes so DesktopEngine parity is testable; include truncation limits for DOM/log payloads; explicitly model downloads/uploads/dialog handling/redirects/CSP errors/video/trace as nullable/unsupported so gaps are visible.
- **Event ordering/backpressure**: guarantee deterministic ordering per attempt, max queue depth/drop policy; completion/failure events must never drop, telemetry/heartbeat may drop with counters; heartbeat cadence independent of WebSocket speed; fairness across executions so one noisy run cannot starve others.
- **Event backpressure limits**: per-execution queue depth target (e.g., 200 events) and per-step attempt sub-queue (e.g., 50); drop policy = drop oldest telemetry/heartbeat first, never drop completion/failure; emit drop counters; circuit-break when sustained pressure persists; decide durable vs ephemeral queues and audit what is lost on restart.
- **Capability schema + validation matrix**: structured metadata (concurrency, requiresDocker/Xvfb, supportsHar/video/iframe, maxViewport, supportsFileUploads) with boot-time verification; map workflow directives/features to required capabilities and reject unsupported workflows before execution with a clear failure kind (e.g., multi-tab → allowsParallelTabs, HAR → supportsHar, file upload → supportsFileUploads); preflight analyzer that inspects compiled plan to derive required capabilities before any engine is started.
- **Cancellation semantics**: executor interrupt protocol (engine cancel vs page.close), idempotent cleanup, and rules for partial artifacts (recorded vs discarded) when cancellation/timeout occurs; annotate partial artifacts with `cancelled` markers so replay consumers know why data is missing.
- **Session lifecycle**: define when sessions open/close on retries, mapping of reuse/clean/fresh → `StartSession`/`Reset`, idle TTL cleanup, and crash recovery if engine dies mid-step (detect, rehydrate, or mark as failed with clear taxonomy); define sticky state policy (cookies/localStorage/download dirs) per reuse mode and across branches.
- **Recorder/executor boundaries**: executor decides when to persist/skip partials; recorder normalizes timestamps/timezones, applies size limits, deduping rules, and produces deterministic IDs; recorder owns crash reconciliation (tombstone incomplete attempts with specific failure kind and note which artifacts are missing).
- **Observability/IDs**: correlation IDs across executor events and recorder artifacts; metrics for queue depth/drops, attempt latency, retries by taxonomy, session lifecycle, payload sizes, and crash-recovery outcomes.
- **Payload size limits**: set explicit limits with truncation markers and hashes for DOM snapshots (e.g., 512KB), console/log entries (e.g., 16KB per entry), network payload previews (e.g., 64KB), and screenshot metadata; recorder enforces and annotates truncation; specify ordering/normalization for network/console (header casing, body truncation) so DesktopEngine parity is testable.

## Migration Steps
1) **Define shared contracts**: Extract neutral types for ExecutionPlan/Instruction/StepOutcome/artifact payloads into an engine-agnostic package; include heartbeat/telemetry payload contract; add golden tests for WebSocket/timeline/artifact shapes.
2) **Introduce interfaces + injection**: Add `AutomationEngine`, `EngineSession`, `ArtifactRecorder`, `EventSink` interfaces; wire `WorkflowService` constructor to accept engine factory + executor + recorder; add config/env selection and per-execution override without touching handlers.
3) **Extract executor**: Move orchestration logic (retries, branching, loops, variable interpolation, session reuse policy, cancellation) out of `browserless/client.go` into a new executor package; executor owns heartbeats/per-attempt telemetry emission via `EventSink` and enforces failure taxonomy.
4) **Extract recorder**: Move persistence/artifact shaping (screenshots, telemetry, DOM snapshot, assertion, retry metadata, timeline frames, cursor trails, last-screenshot reuse, DOM preview) into a recorder module reused by all engines; recorder owns ID/dedupe generation and crash reconciliation (tombstone incomplete attempts with failure kind + missing artifact markers).
5) **Wrap Browserless**: Implement `BrowserlessEngine` using existing CDP adapter; keep session strategy envs; ensure parity with current outputs via tests, including mid-step heartbeats/telemetry.
6) **Wire events**: Keep payload shapes unchanged; add golden tests for `execution.*`, `step.*`, `step.telemetry`, `step.screenshot`, `step.heartbeat` to prevent drift; define fair per-execution queues with drop counters and restart-loss auditing.
7) **DesktopEngine spike**: Stub Playwright-based engine that returns matching `StepOutcome` structure; prove single-workflow run with screenshots/console/network/DOM snapshot and download/upload/dialog coverage; guard dependency with build tags or optional module so it is unused unless explicitly enabled.
8) **Capabilities surfaced**: Add capability metadata to config for scenario-dependency-analyzer to choose `browserless@ubuntu` vs `desktop-engine@electron`; include max concurrency and requiresDocker flags; expose through the registry without widening REST/WS contracts; enforce capability validation matrix on workflow features via compiled-plan preflight.
9) **Rollout/guardrails**: Feature-flag new executor/recorder path with per-execution opt-in; add shadow/dual-run mode that executes both old/new paths for a capped subset (percent + max concurrent) and diffs events/artifacts with size-limited logging; collect metrics (step latency, event counts, artifact sizes, queue drops) to compare old vs new before flipping defaults.
10) **Config guardrails**: Add explicit allowlist per environment (dev/test only for Desktop by default), clear errors when requested engine is unavailable, concurrency/queueing behavior when capacity is exceeded, and budget controls for shadow runs (max additional Browserless sessions/events per minute).
11) **Schema/migration**: Add DB/schema migrations for new IDs/dedupe keys/payload versions; ensure old artifacts remain readable/writable; add backfill/replay compatibility path and version bump strategy.

## Immediate Next Steps (design work)
- Draft concrete types: `StepOutcome`, `StepFailure` (taxonomized), `StepTelemetry`, `EventEnvelope`, and `EngineCapabilities` structs with fields/nullability/defaults and ID/timestamp rules; include schemaVersion fields and event ordering guarantees in comments/tests.
- Package layout sketch: propose `automation/engine` (interfaces + registry), `automation/executor` (orchestration, failure policy), `automation/recorder` (artifacts/IDs/dedupe), `automation/events` (sink + ordering/backpressure), and `automation/contracts` (shared types); outline how `WorkflowService` injects engine factory/executor/recorder without handler churn.
- Rollout/config plan: add config keys (`ENGINE`, `ENGINE_OVERRIDE`, `ENGINE_FEATURE_FLAG`) and metrics/tracing plan (step latency, event count, artifact size, queue depth/drop count, failures by taxonomy); define how to run old vs new side-by-side for comparison; specify size limits/truncation rules for snapshots/log payloads.
- Deterministic test harness: seeded clock/ID generator, fixture DOM/network payloads, and deterministic ordering for console/network to keep golden tests stable across engines.
- Decide concrete queue depths/drop policy (per execution/per attempt) and codify drop counters + circuit-break rules in tests.
- Sketch capability-validation matrix (workflow features → required capabilities) to drive executor preflight checks and clear failure messaging.
- Crash recovery path: define reconciliation algorithm (resume from last persisted attempt vs tombstone) and add fixtures for mid-step engine death with partial artifacts.
- Sticky state policy: codify what is reset vs retained (cookies/localStorage/download dir) per reuse mode and across branches/attempts.
- Parity envelope: document network/console ordering rules, header normalization, body truncation, and download/upload/dialog handling so DesktopEngine targets are testable.
- Fairness: design per-execution queue isolation (and per-client if multiple WS consumers) and decide durable vs ephemeral queues with explicit audit of drops on restart.

## Testing & Validation
- Unit: executor branching/retries/loop directives against a fake engine; recorder golden tests for DB artifacts/timeline/event payloads; per-attempt telemetry/heartbeat emission under cancellation with deterministic seeds/IDs.
- Integration: smoke run with Browserless engine to confirm parity with current behavior; shadow/dual-run comparisons that diff artifacts/events between old/new paths with capped sample size/budget.
- Contract: define and lock replay/export JSON schemas and API responses for timeline/artifact endpoints so DesktopEngine must satisfy the same contract; assert schemaVersion handling and rejection of unknown versions; include downloads/uploads/dialogs/HAR parity.
- Backpressure/cancel: tests for slow WebSocket consumers, queue drop/backoff policy, crash/timeout/cancel scenarios, session reuse matrix across reuse/clean/fresh, and idempotent close; fairness across executions and restart-loss auditing.
- Recovery: simulate executor/engine crashes mid-step, ensure reconciliation marks attempts with failure taxonomy and partial artifact annotations, and verify resumability rules.

## Risks & Mitigations
- Risk: Silent drift in artifact/event payloads → Mitigate with golden/contract tests and recorder centralization.
- Risk: Session strategy regressions → Keep current env flags, test reuse/clean/fresh paths in BrowserlessEngine adapter.
- Risk: DesktopEngine feature gaps → Prioritize matching telemetry (console/network/DOM snapshot) and screenshot parity before advanced motion features; keep concurrency=1 explicitly.
- Risk: Backpressure causing engine stalls → Mitigate with bounded queues/drop policy and heartbeat independent of WebSocket throughput.
