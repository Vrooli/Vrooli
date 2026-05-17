# Usage Reporting

This document is the canonical architecture reference for usage
accounting inside audio-tools: how every chain dispatch produces a
`UsageRow`, how rows are persisted asynchronously, and how the
`UsageService` aggregates them for the operator UI.

Read this first when:

- adding a new capability or operation that should be metered,
- changing the bounded queue depth or retry policy,
- introducing a new aggregation in `GetSummary`,
- debugging "why is my usage row missing from the last 24h?".

Usage reporting is intentionally non-blocking: the request path
enqueues and returns. A drained queue and a successful summary query
are unrelated guarantees.

## Purpose

`internal/usagereport` (`api/internal/usagereport/recorder.go:6`)
owns the async pipeline that persists usage rows from the provider
chains. It is the single home for:

- the `Recorder` interface (`Enqueue` non-blocking; `Record`
  synchronous; `Close` flushes),
- the bounded buffered channel (`QueueCapacity = 1024`),
- the drop-newest policy when the queue is full,
- the 500 ms / 1 s / 2 s retry ladder per row,
- the observability counters (`DroppedTotal`, `EnqueuedTotal`,
  `QueueDepth`).

`handlers/usage` (`api/handlers/usage/handler.go:21`) is the
Connect-RPC read surface: `ListRecent` paginates raw rows;
`GetSummary` returns aggregated counts.

`internal/store.UsageStore` (`api/internal/store/usage.go:53`) is the
SQLite persistence layer with idempotent inserts on `operation_id`.

## Inputs

Producers populate a `store.UsageRow`
(`api/internal/store/usage.go:11`) and call `Recorder.Enqueue`:

| Field | Notes |
|---|---|
| `OperationID` | Required; usually `X-Audio-Operation-ID` header value or a fresh UUID. Insert is idempotent on this id, so retries cannot double-count. |
| `EmittedAt` | UTC timestamp; `UsageStore.Insert` defaults to `time.Now().UTC()` when zero. |
| `Capability` | `stt` \| `tts` \| `summarize` \| `audio`. |
| `Operation` | `transcribe` \| `synthesize` \| `summarize` \| `transcode` \| etc. |
| `ProviderTier`, `ProviderID`, `ModelID` | Trace fields from the chain `Result`. Blank on the error path. |
| `LatencyMs` | Wall-clock latency the handler measured around `Chain.Execute`. |
| `CreditsCharged` | Currently always 0 for non-Vrooli tiers; Vrooli credit accounting is wired through this field. |
| `PromptTokens`, `OutputTokens` | Populated by summarize / chat-style operations. |
| `AudioDurationSeconds` | Populated by STT / audio operations. |
| `Error` | Error string; blank on success. |
| `FallbackReason` | Why the chain demoted to a lower tier (e.g., `"byok_timeout"`). Optional. |
| `UserIdentity` | From the BYOK envelope; opaque string the operator uses to scope rows. |

Today the summarize handler is the only handler wiring usage
(`api/handlers/summarize/handler.go:67`). TTS, STT, and audio
handlers are expected to follow the same shape; the recorder is
already wired through `Deps`.

## Outputs

`UsageService.ListRecent`
(`api/handlers/usage/handler.go:56`) returns up to `limit` rows
newer than `since_seconds` ago (default 24h, limit clamps to 100;
ceiling 1000 enforced in `UsageStore.ListRecent`), optionally
filtered by `capability` and `provider_tier`. Rows are
newest-first.

`UsageService.GetSummary`
(`api/handlers/usage/handler.go:73`) returns an aggregated
`Summary`:

| Field | Source |
|---|---|
| `Since`, `Until` | Window boundaries (RFC3339). |
| `OperationsTotal` | `COUNT(*)` over the window. |
| `CreditsTotal` | `SUM(credits_charged)`. |
| `ErrorCount` | `SUM(CASE WHEN error<>'' THEN 1 ELSE 0 END)`. |
| `Distribution` | `GROUP BY provider_tier, provider_id` with count, credits, avg latency. |
| `FallbackReasons` | `GROUP BY fallback_reason` (excluding blank). |

Both methods return `CodeFailedPrecondition` when the store is not
wired (`api/handlers/usage/handler.go:58`).

## Internal Chain

```
Chain dispatch (e.g. summarize.Summarize)
        │
        ▼
build store.UsageRow from chain Result + error path
        │
        ▼
Recorder.Enqueue(row)         (api/internal/usagereport/recorder.go:75)
        │
        ├── queue has space  → enqueuedTotal++, return immediately
        │                       │
        │                       ▼
        │                   drain goroutine (single)
        │                       │
        │                       ▼
        │                   flush(row)  (api/internal/usagereport/recorder.go:111)
        │                       │
        │                       ├── Insert OK → done
        │                       │
        │                       └── Insert fail
        │                             │
        │                             ▼
        │                       retry after 500ms, 1s, 2s
        │                             │
        │                             ├── any retry OK → done
        │                             └── all fail → log drop
        │
        └── queue is full   → droppedTotal++, log "queue full, dropping op=…", return

Read path:
ListRecent / GetSummary  →  UsageStore  →  SQLite usage_rows table
```

Insert is idempotent on `operation_id`
(`api/internal/store/usage.go:64`): `INSERT OR IGNORE`. The retry
ladder is therefore safe — a row that landed on attempt 1 and a
retry on attempt 2 collapse to a single stored row.

Close-time semantics (`api/internal/usagereport/recorder.go:99`):
`Close` closes the queue channel and waits for the drain goroutine
to finish, so a graceful shutdown flushes whatever's in flight. A
hard kill drops the queue contents.

The drop policy is **newest-rejected**: a full queue rejects the
incoming `Enqueue`, not the oldest queued row. This biases toward
fairness for older operations (which already paid the chain cost) at
the expense of brand-new operations during a queue saturation event.

## Seams

| Seam | Interface | Production | Test fake |
|---|---|---|---|
| Usage recorder | `usagereport.Recorder` (`api/internal/usagereport/recorder.go:37`) | `AsyncRecorder` backed by `store.UsageStore` | Synchronous recorder in handler tests |
| Usage persistence | `*store.UsageStore` (`api/internal/store/usage.go:53`) | SQLite `usage_rows` table | In-memory SQLite |
| Recorder logger | `*log.Logger` | `log.Default()` | Per-test `bytes.Buffer` to assert drop / retry messages |

The `Recorder` interface is the seam handlers depend on; the async
implementation is hidden behind it. Handlers that need synchronous
persistence (e.g., a CLI ingest path) call `Record` instead of
`Enqueue`.

## Failure Modes

| Cause | Behavior |
|---|---|
| `Enqueue` while drain is closed | Sends on closed channel — panics; this path requires explicit operator action (`Close` then `Enqueue`). Production wiring closes the recorder during shutdown only. |
| Queue full at `Enqueue` time | Row dropped, `droppedTotal++`, log line emitted. The caller does NOT see an error — usage drop is invisible to the request path. |
| Single `Insert` failure | Retried at 500 ms / 1 s / 2 s; each retry has its own 5 s context. |
| All retries failed | Row dropped, log line emitted. No DLQ; the row is gone. |
| `UsageStore` not wired in handler | `ListRecent` / `GetSummary` return `CodeFailedPrecondition`. |
| `since_seconds <= 0` | `resolveSince` defaults to 86400 (24h) (`api/handlers/usage/handler.go:101`). |
| `limit <= 0` or `> 1000` | `UsageStore.ListRecent` clamps to 100 (`api/internal/store/usage.go:81`). |
| Insert OK then a duplicate Insert arrives | `INSERT OR IGNORE` — second insert silently skipped. |
| Capability/operation typo in producer | Row stored as-is — `GROUP BY` will surface a stray bucket in the summary, visible to operators. |

The recorder logs but does not metric drops/retries to a Prometheus
or OTLP sink today. `Stats()` (`api/internal/usagereport/recorder.go:86`)
exposes the counters for any future scraping endpoint; the natural
place to surface them is the `/health` metrics block.

## Capacity Notes

The queue is fixed at 1024 rows. At a steady-state of one row per
chain dispatch, 1024 rows buys ~17 minutes of headroom at 1 op/s, or
~1 second at 1000 op/s. Operators expecting bursty workloads with
slow SQLite (network filesystem, full disk) should size their
infrastructure to keep `flush` latency well under the production op
rate × (1024 / op rate) seconds — otherwise sustained drops are the
expected behavior.

The drain is single-goroutine; there is no parallelism. SQLite
inserts are serialised inside the database anyway, so adding more
drain workers would not help write throughput.

`Summary` is one `COUNT`/`SUM` query plus one `GROUP BY` query plus
one fallback-reason query — three round-trips. There is no result
cache; every `GetSummary` re-queries. For the expected operator UI
poll cadence (seconds to minutes) this is fine; for a busy
dashboard, fronting `GetSummary` with a short-TTL cache is the
right next step.

`UsageRow` carries a `UserIdentity` field but there is no automatic
retention or PII scrubbing. Operators expecting GDPR-shaped retention
must add the prune job; the natural place is a sibling `Prune(ctx,
older_than)` method on `UsageStore`.

## Cross-References

- [`../../internal/SEAMS.md`](../../internal/SEAMS.md) — full seam registry
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — async-drop-newest decision
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — missing metrics / retention gap
- [`../../reference/configuration.md`](../../reference/configuration.md) — operator-tunable levers
- [`../summarize/chain.md`](../summarize/chain.md) — example producer (the only handler currently wired)
- [`../tts/synthesis-pipeline.md`](../tts/synthesis-pipeline.md) — future producer
- [`../stt/streaming-pipeline.md`](../stt/streaming-pipeline.md) — future producer
- `packages/proto/schemas/audio-tools/v1/usage/usage.proto` — wire shape
