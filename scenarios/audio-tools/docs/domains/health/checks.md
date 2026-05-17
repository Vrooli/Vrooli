# Health Checks and Capability Matrix

This document is the canonical architecture reference for
audio-tools' health surface: the `/health` and `/api/v1/health`
endpoints, the capability registry that downstream scenarios consume
to discover what audio-tools can do, and the split between full and
liveness-only checks.

Read this first when:

- adding a new local resource that needs a health gate (e.g., a new
  speech model),
- registering a new capability for downstream scenarios to discover,
- changing the cache TTL or splitting full / liveness checkers,
- debugging "why does `/health` say healthy but Whisper requests
  fail?".

Health is distinct from capability: `/health` answers "can this
process serve requests at all?" (single dependency: the database).
Capabilities answer "which optional providers are reachable right
now?" — that's a richer surface used by consumer scenarios.

## Purpose

`handlers/health/handler.go:31` owns the `/health` HTTP endpoint.
It is built on `api-core/health` for the standardized response
envelope (status / dependencies / metrics) but plumbed through the
local `database.Pinger` seam so handler tests can substitute a fake
without opening the on-disk SQLite file.

`internal/capabilities/registry.go:121` owns the in-process capability
state. It is the single home for:

- the canonical `Known` capability catalogue
  (`api/internal/capabilities/registry.go:53`),
- the `Checker` interface and its production implementations
  (`api/internal/capabilities/checkers.go`),
- per-capability status caching with TTL,
- the full-vs-liveness checker split.

The `/health` endpoint is for load balancers and `curl`. The
capability registry is for downstream scenarios (e.g. web-console)
deciding whether to enable voice features.

## Inputs

`GET /health` and `GET /api/v1/health` (same handler, two mounts —
`api/handlers/health/module.go:23`) take no inputs. They run the
database `PingContext` and roll the result into the standardized
envelope.

`Capability` checks consume:

- the `Def` catalogue (`api/internal/capabilities/registry.go:30`)
  — each entry declares an `ID`, human `Name`, `DependencyKind`
  (`resource` or `scenario`), `DependencySlug`, and the `Features`
  it would unlock.
- per-capability `Checker` implementations passed in via the
  `checkers map[string]Checker` argument to `NewRegistry`.
- optional `livenessCheckers` for fast probes that skip expensive
  verification (e.g., the Whisper full checker sends a silent WAV
  to `/asr`; the liveness checker just hits the HTTP root).

## Outputs

`/health` returns the `api-core/health` envelope (JSON):

| Field | Notes |
|---|---|
| `status` | `"healthy"` or `"unhealthy"`. Flipped to unhealthy when any `Critical` check fails. |
| `service` | From `Deps.Service`. |
| `version` | From `Deps.Version`. |
| `dependencies.database` | Result of `Pinger.PingContext(ctx)`; reported as `Critical`. |
| HTTP status | `200` healthy / `503` unhealthy. |

The capability registry's read methods return `[]State`
(`api/internal/capabilities/registry.go:39`):

| Field | Notes |
|---|---|
| `Def` (embedded) | ID, Name, Description, DependencyKind, DependencySlug, Features. |
| `Status` | `available` \| `unavailable` \| `unknown`. |
| `Message` | Human-readable check message ("resource is healthy", "resource is not responding", etc.). |
| `CheckedAt` | RFC3339 UTC. |

`Resolve(ctx)` returns the cached, full-check view (refreshed when
the cache is older than `cacheTTL`). `ResolveLiveness(ctx)` returns
a cheap-check view that does NOT update the main cache — so a
service that is broken-but-live cannot trick the full cache into
showing it as healthy.

## Internal Chain

```
GET /health
    │
    ▼
apihealth.New(service).
    Version(version).
    Check(apihealth.Func("database", Pinger.PingContext), Critical).
    Handler()
    │
    ▼
PingContext → SQLite ping
    │
    ▼
envelope { status, dependencies: { database: ... } }


Capability check (called by handlers/capabilities, downstream scenarios):
    │
    ▼
Registry.Resolve(ctx)
    │
    ├── cached && time.Since(cachedAt) < cacheTTL → return copy
    │
    └── refresh:
          for each Def in defs:
              if checker := checkers[Def.ID]; ok:
                  state.Status, state.Message = checker.Check(ctx)
              else:
                  Status = StatusUnknown
          write to cache
          return copy


Capability check (liveness mode):
    │
    ▼
Registry.ResolveLiveness(ctx)
    │
    ├── cached && time.Since(cachedAt) < cacheTTL → return copy (same as Resolve)
    │
    └── liveness-only:
          for each Def in defs:
              if checker := livenessCheckers[Def.ID]; ok:
                  state.Status, state.Message = checker.Check(ctx)
              else if checker := checkers[Def.ID]; ok:
                  fall back to full checker
              (does NOT write to cache)
          return states
```

### The capability catalogue

`Known` (`api/internal/capabilities/registry.go:53`) is the source of
truth for what audio-tools advertises:

| ID | Kind | Features |
|---|---|---|
| `whisper-stt` | resource | `voice-input`, `voice-streaming` |
| `speaker-verification` | resource | `voice-speaker-verification`, `voice-enrollment` |
| `kokoro-tts` | resource | `voice-output` |
| `ollama` | resource | `ai-command-generation` |
| `openrouter` | resource | `ai-command-generation` |
| `audio-tools` | scenario | All of the above plus `tts-summarization`, `tts-cache`, `tts-paragraph-split`, `audio-provider-routing` |

The `audio-tools` self-entry is intentional: downstream scenarios
that adopt audio-tools wholesale resolve features via this one entry
instead of having to depend on five resources. The individual
resource entries remain for consumers that depend on a single
resource directly.

### Checker implementations

Two production checker shapes
(`api/internal/capabilities/checkers.go`):

- **`ResourceChecker`** — GET against the resource URL with a 5 s
  timeout. `200` or `307` → `StatusAvailable`; other codes →
  `StatusUnavailable`; transport error → `"resource is not
  responding"`.
- **`WhisperChecker`** — sends a 0.1 s silent WAV to the Whisper
  `/asr` endpoint. Catches the case where the bare health endpoint
  responds but transcription is broken (model not loaded, ffmpeg
  missing inside the container). This is the canonical example of why
  full and liveness checks are split.

## Seams

| Seam | Interface | Production | Test fake |
|---|---|---|---|
| Database pinger | `database.Pinger` | SQLite handle | In-memory fake in handler tests |
| Capability checker | `capabilities.Checker` (`api/internal/capabilities/registry.go:46`) | `ResourceChecker`, `WhisperChecker`, others | Function-literal checkers returning canned Status/message |
| Capability registry | `*capabilities.Registry` | Single in-process instance | Tests construct registries directly |
| Cache TTL | `cacheTTL` constructor arg | Currently 30 s in production wiring | Per-test override (often 0 to disable caching) |

The `Pinger` seam is deliberately narrow: `/health` should never
hold a connection-pool-blocking ping. The interface is
`PingContext(ctx) error` so a fast SQLite ping (sub-millisecond) is
the actual cost.

## Failure Modes

| Cause | Behavior |
|---|---|
| Database ping fails | Envelope flips to `status="unhealthy"`, HTTP `503`. Load balancers should fail the pod out of rotation. |
| Resource URL unreachable | `ResourceChecker.Check` returns `(StatusUnavailable, "resource is not responding")`. Capability status surfaces as `unavailable` in the next `Resolve`. |
| Resource returns 5xx | `ResourceChecker.Check` returns `(StatusUnavailable, "resource returned unexpected status")`. |
| Whisper bare endpoint live but transcription broken | `ResourceChecker` would say healthy; `WhisperChecker` catches it by attempting a real transcription. Only the full checker (`Resolve`) catches this; `ResolveLiveness` may report healthy until cache refresh. |
| Checker missing for a registered `Def` | State returned with `Status = StatusUnknown` and blank message. |
| Cache stale during `ResolveLiveness` with no liveness checker | Falls back to the full checker for that Def. |
| Concurrent `Resolve` calls during refresh | Both grab the write lock; the second sees a fresh cache and returns immediately. |
| `livenessCheckers` not set | `ResolveLiveness` returns `Resolve(ctx)` (full check). |

There is no automatic retry inside checkers — a 5 s HTTP timeout
either succeeds or fails. The TTL is the implicit retry cadence.

## Capacity Notes

`/health` is sub-millisecond on the happy path (SQLite ping). It is
intentionally cheap so a load balancer can poll at 1 Hz without
adding meaningful load.

Capability checks are at most `len(Known)` HTTP round-trips per
refresh (currently 6 entries: 5 resources + the audio-tools
self-entry which has no checker). With a 30 s TTL and the typical
operator-UI poll cadence (seconds to minutes), the refresh cost is
amortised. The cache copy returned to callers is a fresh slice so
mutating it cannot corrupt the cached state.

The Whisper full checker sends 3.2 KiB of audio per refresh — a
30 s TTL means ~6.4 KiB/min of liveness traffic against Whisper.
This is negligible but worth knowing when sizing the Whisper
container's request budget.

`ResolveLiveness` was introduced specifically so the operator UI can
poll quickly (1 Hz) without paying the Whisper-transcription cost
every poll. The full check still runs on `Resolve` requests every
TTL window so a broken-but-live service does eventually surface as
unavailable.

There is no metrics export today; `/health` does not include
counters of past check failures. Adding `metrics` to the envelope
would let operators see "Whisper has failed 3 of the last 10 checks"
without sampling the log stream — known future need.

## Cross-References

- [`../../internal/SEAMS.md`](../../internal/SEAMS.md) — full seam registry
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — full-vs-liveness split decision
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — missing metrics
- [`../../reference/configuration.md`](../../reference/configuration.md) — resource URLs that drive checkers
- [`../../operations/OBSERVABILITY.md`](../../operations/OBSERVABILITY.md) — operator-facing health signals
- [`../tts/synthesis-pipeline.md`](../tts/synthesis-pipeline.md) — provider availability cache parallels capability checks
- `packages/proto/schemas/audio-tools/v1/` — no health proto; `/health` is REST by deliberate exception (`api/handlers/health/endpoints.go:23`)
