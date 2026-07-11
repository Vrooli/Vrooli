# Validation Temporal Flows

## Checkpoint Flows

1. `start` resolves scope and persists a queued operation plus every child.
2. A scheduler claims one child per operation before marking it running; global
   capacity is bounded and queued children never allocate goroutines. This gives
   competing operations a share instead of letting a large graph monopolize it.
3. Queue residence ends before the execution deadline begins; transport wait has
   its own budget and cannot cancel execution.
4. After all oracle children are terminal, Plan Manager computes the verdict,
   upserts the stable terminal result, and marks the operation terminal.
5. `show` reads without waiting. `wait` succeeds only after a durable terminal
   reread; detach/EOF/deadline is typed non-success. `resume` restarts missing
   queued/running dispatch after process restart and then uses that same predicate.

## Budget Boundaries

| Budget | Value | Starts when | Outcome on expiry |
|---|---:|---|---|
| Queue residence | 2m advisory | intent commit | never consumes child execution time |
| Validation operation execution | 30m | operation enters running | unfinished children become UNKNOWN |
| Individual command execution | 10m | child dispatch | typed execution timeout/UNKNOWN |
| CLI transport attachment | 15m | wait/resume request | one inspect recovery by operation ID |
| API write timeout | 17m | HTTP request | margin above transport attachment |

## Interruption Handling

- Client EOF/cancel: execution continues; the caller reattaches by operation ID.
- Process restart: boot lists non-terminal operations and resumes only unfinished
  children. Already-terminal checkpoints remain untouched.
- Partial child completion: every completed child is durable immediately, so an
  aggregate failure cannot erase evidence already received.
- Unexpected EOF in the CLI: exactly one non-blocking `show`-equivalent recovery
  read; no loop and no duplicate start.
