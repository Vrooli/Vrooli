# Performance — Switchboard

## Purpose Of This Document

The performance contract: what must be fast, what is allowed to be slow, what
has actually been measured, and how a regression is caught. Use it to avoid
optimising the wrong thing — most of this scenario is conversational and
latency-insensitive, and exactly three paths are not.

## Budgets

Budgets are stated as intent. **Nothing here has been measured**, because no
domain code exists; see Current Measurements.

| Path | Budget | Why this number |
|---|---|---|
| Ingress accept → thread append → transport ack | **< 150 ms p95** | The transport is waiting. Exceeding a provider's webhook timeout causes a redelivery, which is why de-duplication is P0 rather than a nicety |
| Ingress de-duplication lookup | **< 5 ms p95** | A single indexed lookup on `(channel_id, remote_message_id)`. If this is slow the index is wrong |
| Trust resolution (F2) | **< 20 ms p95** | Pure computation over a contact row, a roster, and a grant. It runs on every turn and it must never be worth caching, because a cached permission is a stale permission |
| Turn dispatch → first reply token | **not budgeted** | Dominated by model inference in `agent-manager`. Budgeting it here would be measuring somebody else's system |
| Boot: descriptor load and validation | **< 500 ms** for 20 descriptors | File I/O plus schema validation, once per boot. Never on the message path |
| Host availability probe | **< 2 s**, cached with a stated age | Reaches `vrooli-bridge` and `tunnel-manager`. Never on the message path; the console shows the cache age rather than blocking |
| Console: roster and thread list first paint | **< 1 s p95** on a warm API | Operational surfaces are scanned repeatedly; a slow roster is felt every time |
| Voice: capture → transcript partial | **< 300 ms p95** | The only genuinely hard real-time constraint in the scenario, and it is P2. See the caveat below |

**The three paths that actually matter** are ingress acknowledgement, trust
resolution, and — at P2 — voice turn latency. Everything else is conversational:
a human is reading, and 200 ms versus 400 ms is invisible.

## Current Measurements

**None.** No domain code exists. This section is deliberately empty rather than
populated with template defaults, because a fabricated baseline is worse than no
baseline: it makes the first real regression invisible.

| Path | Measured | Date | Method |
|---|---|---|---|
| — | — | — | — |

The first slice that lands must record its own baseline here, with the command
that produced it.

## Known Constraints

| Constraint | Consequence | Mitigation |
|---|---|---|
| **SQLite, single writer** | Concurrent inbound bursts across channels serialise on write | Accepted. Traffic is conversational, not bulk. If this ever binds, the answer is batching appends, not a server engine — neither is `supported` on macOS and `postgres` is explicitly unproven there, so either would put the Mac node at risk |
| **No cross-process queue** | Adapter work, delivery, and budget accounting share the API process | Accepted at P0 by the no-resource decision. Revisit only if a channel's throughput genuinely exceeds one process |
| **Cross-node dispatch is a round trip** | iMessage delivery through a Mac node adds bridge latency, and a briefly offline node adds much more | Durable dispatch, so a delay is not a loss. Availability is cached per machine with a stated lifetime and refreshed after a failure |
| **Per-channel rate limits are external** | Every provider throttles differently, and exceeding a limit can cost an account rather than a retry | Limits are declared per descriptor and enforced locally before send, rather than discovered by being throttled |
| **Model inference dominates end-to-end latency** | The user-perceived number is mostly not this scenario's | Do not optimise this scenario against an end-to-end figure. Measure the seams it owns |
| **Speech currently runs on CPU** | Any voice latency budget written today would be wrong by a wide margin | `SWBD-PROB-002`. Dictation never sends `engine_id`, so transcription silently falls back to CPU while every signal reads green. Fix before measuring anything voice-related |
| **Clock-derived windows** | Hourly turn budgets and approval expiry depend on the clock seam | Use `api/internal/clock/` so both are provable in tests without sleeping. A test that sleeps to prove a budget is a test that will flake |

## Regression Procedure

1. Establish the baseline **in the slice that introduces the path**, not
   afterwards. Record it in Current Measurements with the command used.
2. Measure the seams this scenario owns, with the model stubbed. An end-to-end
   number that includes inference measures `agent-manager` and a provider, and
   moves for reasons unrelated to any change here.
3. Ingress acknowledgement and trust resolution get a benchmark in the API test
   suite, because both are on every message and both are cheap to guard.
4. On a suspected regression: reproduce with the recorded command, compare
   against the recorded baseline, and bisect by seam — adapter, de-duplication,
   append, resolution, dispatch — rather than by commit.
5. A budget that is missed twice in a row is either a real regression or a wrong
   budget. Change the number deliberately and record why, rather than letting it
   sit red.

## Cross-References

- `docs/concepts/FLOWS.md` — the paths these budgets cover
- `docs/concepts/DATA.md` — the indexes the fast paths depend on
- `docs/internal/PROBLEMS.md` — `SWBD-PROB-002`, the speech constraint above
- `docs/internal/TESTING.md` — where benchmarks live
- `docs/operations/OBSERVABILITY.md` — the signals that would show a regression in production
