# Control Surface & Tunable Levers

Scenario-level control surface for browser-automation-studio across the Go API, Playwright driver, and test runner. Use this as the steering dashboard; `docs/ENVIRONMENT.md` remains the canonical variable list and `playwright-driver/CONTROL-SURFACE.md` covers driver-only detail.

## Priority levers (what to steer)

| Lever | Set via | Default | When to move |
|-------|---------|---------|--------------|
| Execution budget & per-step limits | Workflow metadata `executionTimeoutMs`/`defaultTimeoutMs`; API `BAS_EXECUTION_*`; driver `EXECUTION_*` | API dynamic ≈ `30s + steps*10s` (subflows 15s/step, clamp 90s–4.5m); driver 30s default / 45s navigation / 30s waits / 15s assertions | Raise for flaky/slow sites; lower for CI fail fast. Keep driver < API < HTTP client (5m). |
| Selector probes & heartbeats | `BAS_EXECUTION_DEFAULT_ENTRY_TIMEOUT_MS`, `BAS_EXECUTION_MIN_ENTRY_TIMEOUT_MS`, `BAS_EXECUTION_HEARTBEAT_INTERVAL_MS` (`BROWSERLESS_HEARTBEAT_INTERVAL` alias) | 3000ms entry / 250ms floor / 2000ms heartbeats (0 = off) | Increase entry timeout for slow rendering; shorten for fixtures to catch missing selectors quickly. Lengthen or disable heartbeats if clients are bandwidth-constrained; tighten for closer liveness tracking. |
| Session concurrency/pooling | Driver `MAX_SESSIONS`, `SESSION_POOL_SIZE`, `SESSION_IDLE_TIMEOUT_MS`; engine capabilities mirror driver | 10 concurrent / pool 5 / idle 5m | Raise pool/idle for reuse-heavy runs; cap lower for resource-constrained nodes. Keep `maxConcurrent >= poolSize`. |
| Telemetry backpressure | `BAS_EVENTS_PER_EXECUTION_BUFFER`, `BAS_EVENTS_PER_ATTEMPT_BUFFER`, `BAS_WS_CLIENT_*` | 200 per execution / 50 per attempt; WS send 256 / binary 120 / read 512 | Raise for verbose console/network/heartbeat streams or UX metrics; lower on memory pressure. Droppable events are heartbeat/telemetry/screenshot; completion/failure stay guaranteed. |
| Recording quality & archive bounds | `BAS_RECORDING_DEFAULT_*`, `BAS_RECORDING_MAX_ARCHIVE_BYTES`, `BAS_RECORDING_MAX_FRAMES`, driver recording toggles | 1280x720, 6 FPS, JPEG 55; 200MB archive; 400 frames; 90s idle timeout | Raise quality/FPS for demos; lower for CI to reduce bandwidth and processing. Trim archive/frame caps when running many parallel recordings. |
| Replay/export workload | `BAS_REPLAY_CAPTURE_INTERVAL_MS`, `BAS_REPLAY_MAX_CAPTURE_FRAMES`, `BAS_REPLAY_RENDER_TIMEOUT_MS`, `BAS_EXPORT_FRAME_INTERVAL_MS`, `FFMPEG_BIN` | 40ms interval, 720 frames, 16m render timeout | Tighten for CI to avoid long ffmpeg runs; loosen for customer-ready exports. |
| DOM/AI extraction limits | `BAS_AI_DOM_*`, `BAS_AI_PREVIEW_*` | Depth 6, nodes 800, text 120 chars, waits 750/1200ms; preview 1920x1080 | Raise for complex SPAs; lower to keep payloads lean and control inference cost. |
| Database resilience | `BAS_DB_BACKEND`, `DATABASE_URL`, `BAS_DB_MAX_OPEN_CONNS`, `BAS_DB_MAX_IDLE_CONNS`, retry envs | Postgres; 25 open / 5 idle; retries 10 with 1s base and jitter 0.25 | Raise pool for high concurrency; trim for laptops. Increase retry delay/jitter for noisy networks. |
| Entitlement & gating | `BAS_ENTITLEMENT_*` | Disabled; cache 5m; offline grace 24h; default tier `free` | Enable in staged/prod to enforce tier limits and watermarks; align cache TTL with billing cadence. |
| Observability & perf diagnostics | `LOG_LEVEL`, `LOG_FORMAT`, driver `METRICS_ENABLED`; API `BAS_PERF_*`; health timeouts `BAS_TIMEOUT_*` | `info` / `json`; perf disabled; metrics on | Bump to `debug` when diagnosing dragDrop/selector issues or stream stalls. Enable perf buffers temporarily when profiling frame streaming. |

## Profiles (swap quickly)

- **Fast local iteration:** `BAS_TIMEOUT_DEFAULT_REQUEST_MS=2500`, `BAS_EXECUTION_PER_STEP_TIMEOUT_MS=6000`, `BAS_EXECUTION_HEARTBEAT_INTERVAL_MS=1000`, `SCREENSHOT_ENABLED=false`, `DOM_ENABLED=false`, `BAS_EVENTS_PER_EXECUTION_BUFFER=120`, `BAS_EVENTS_PER_ATTEMPT_BUFFER=30`.
- **Flaky/slow CI:** `EXECUTION_NAVIGATION_TIMEOUT_MS=90000`, `EXECUTION_WAIT_TIMEOUT_MS=60000`, `BAS_EXECUTION_PER_STEP_SUBFLOW_TIMEOUT_MS=25000`, `BAS_DB_MAX_OPEN_CONNS=40`, `BAS_WS_CLIENT_BINARY_BUFFER_SIZE=256`, `BAS_EVENTS_PER_EXECUTION_BUFFER=400`; keep HAR/video off.
- **High-fidelity demos/exports:** `BAS_RECORDING_DEFAULT_STREAM_QUALITY=75`, `BAS_RECORDING_DEFAULT_STREAM_FPS=12`, `SCREENSHOT_FULL_PAGE=true`, `BAS_AI_PREVIEW_WAIT_MS=1800`, `BAS_REPLAY_CAPTURE_INTERVAL_MS=25`, `BAS_REPLAY_PRESENTATION_WIDTH=1920`, `BAS_REPLAY_PRESENTATION_HEIGHT=1080`.

## Hierarchy & bounds to respect

- Timeout chain: API global (`BAS_TIMEOUT_GLOBAL_REQUEST_MS` 15m) > HTTP client to driver (5m) > execution budget (`executionTimeoutMs`/dynamic calc) > driver instruction timeouts (`EXECUTION_*`) > Playwright-level waits.
- Telemetry buffers: event buffers (`BAS_EVENTS_*`) cap what the sequencer holds; WebSocket buffers (`BAS_WS_CLIENT_*`) should comfortably carry the expected FPS/heartbeat volume.
- Storage/telemetry caps: keep screenshot/DOM/network caps below driver `MAX_REQUEST_SIZE` and API `BAS_HTTP_MAX_BODY_BYTES` to avoid rejection.
- Ensure archive caps and replay frame counts stay within storage quotas when running parallel executions.

## Recently wired levers (now active)

- Execution timeout envs honored via `config.Execution` (base/per-step/subflow/min/max) with safe defaults (`BAS_EXECUTION_*`).
- Heartbeat cadence now comes from `BAS_EXECUTION_HEARTBEAT_INTERVAL_MS` (alias `BROWSERLESS_HEARTBEAT_INTERVAL`), defaulting to 2s with 0 to disable.
- Event buffer limits now respect `BAS_EVENTS_PER_EXECUTION_BUFFER` / `BAS_EVENTS_PER_ATTEMPT_BUFFER` for WebSocket sinks and collectors.
- Adhoc cleanup cadence/retention configurable via `BAS_EXECUTION_ADHOC_CLEANUP_INTERVAL_MS` / `BAS_EXECUTION_ADHOC_RETENTION_PERIOD_MS`.
- Engine concurrency cap mirrors driver `MAX_SESSIONS` (default 10) for consistent pool limits.

## Non-levers (keep internal)

- Driver safety guards (`MIN_TIMEOUT_MS`, `MAX_TIMEOUT_MS`, allowed URL protocols, drag/swipe step counts) stay fixed to prevent unsafe automation behavior.
- Selector duplication stripping and retry/backoff constants inside the compiler/executor stay internal; tune wait/timeout levers instead.
- Event drop policy remains bounded/droppable only for heartbeat/telemetry/screenshot to preserve completion/failure guarantees.

## References

- `docs/ENVIRONMENT.md` – canonical variable list and timeout hierarchy.
- `api/config/config.go` – API lever definitions and validation.
- `api/services/workflow/automation.go` – heartbeat wiring into executor requests.
- `api/services/workflow/service.go` / `api/handlers/handler.go` – event buffer limits applied to WebSocket sinks.
- `playwright-driver/src/config.ts` and `playwright-driver/CONTROL-SURFACE.md` – driver-side levers.
