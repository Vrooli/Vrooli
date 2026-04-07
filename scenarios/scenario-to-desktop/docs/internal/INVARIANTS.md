# System Invariants

## Last Updated
2026-04-06

## Critical Invariants

| Invariant | Domain Concept | Enforcement | Test Coverage |
|-----------|----------------|-------------|---------------|
| Pipeline stages execute sequentially: Bundle → Preflight → Generate → Build → SmokeTest → Deploy | Pipeline state machine | `DefaultOrchestrator` in [CODE: api/pipeline/orchestrator.go] enforces stage ordering | [CODE: api/pipeline/orchestrator_test.go] |
| Pipeline status transitions are one-directional: Running → Completed/Failed | Pipeline lifecycle | `InMemoryStore.Update()` callback with `RWMutex` | [CODE: api/pipeline/store_test.go] |
| Electron main window always uses `contextIsolation: true, nodeIntegration: false` | IPC security boundary | Hardcoded in [CODE: templates/vanilla/main.ts] window creation | Template tests |
| Secret prompt modals use inline data URIs and are destroyed after input | Credential isolation | Modal window created with `nodeIntegration: true` only for ephemeral secret collection, destroyed on response | [CODE: templates/vanilla/main.ts] |
| All domain errors are `DomainError` type with semantic `ErrorCode` | Error contract | Type system in [CODE: api/shared/errors/errors.go] | All handler tests |
| Bundle cleanup paths must contain `platforms/{framework}` | Path traversal prevention | Validation in build handler before `os.RemoveAll` | [CODE: api/build/handler_test.go] |
| All HTTP responses include `X-Request-ID` header | Request tracing | `RequestIDMiddleware` in [CODE: api/shared/http/middleware.go] | Middleware tests |

## Important Invariants

| Invariant | Domain Concept | Enforcement |
|-----------|----------------|-------------|
| CORS origins default to localhost-only if misconfigured | Security default-deny | `CORSMiddleware` falls back to `http://localhost:{UI_PORT}` |
| Template generation requires `scenario_name` in config | Input validation | `ValidateConfig()` in generation package |
| Pipeline blocking mode extends HTTP write deadline | Timeout safety | `SetWriteDeadline` called in handler before orchestrator invocation |
| Async pipelines use `context.Background()` to survive HTTP disconnects | Pipeline independence | Orchestrator creates fresh context for async runs |

## Replay/Idempotency Invariants

- **Pipeline execution**: `IdempotencyKey` field on `Config`; `GetByIdempotencyKey()` deduplicates running/completed pipelines — safe to retry the same generation request
- **UI submission guard**: `isSubmitting` flag + `currentIdempotencyKey` in `pipelineStore` prevents double-clicks; `resetForRetry()` resets session ID for explicit retries
- **Template generation**: Same inputs produce same output files (deterministic template compilation)
