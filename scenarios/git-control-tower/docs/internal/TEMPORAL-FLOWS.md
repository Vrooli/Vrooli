# Temporal Flows & Async Patterns

## Last Updated
[Date]

## Async Flows Identified

| Flow | Entry Point | Async Operations | Completion Signal |
|------|-------------|------------------|-------------------|
| [name] | [where it starts] | [what's async] | [how we know it's done] |

## Race Conditions
[Identified race conditions and their status]
- **Location**: description, mitigation status

## Timing Assumptions
[Implicit ordering or delay assumptions]

- Snapshot/diff detached tails have a 30m attachment ceiling. Expiry leaves the
  durable intent pending; it is not an execution verdict.
- The baseline CLI transport ceiling is 30m and the API write timeout is 31m,
  preventing the server from cutting off the client first.
- Test Genie owns queue and execution truth. A later status/resume attachment
  gets a fresh transport budget rather than subtracting prior queue residence.

## Checkpoint Flows
[From progress-continuity-interruption-resilience]
- **Snapshot:** `StartCapture` persists a pending intent before the handler
  dispatches its detached finalizer. Terminal Test Genie truth → one pin + V2
  manifest + ready intent. Resume through `snapshot status --run R [--wait]` or
  startup reattachment. Caller cancellation/deadline leaves the intent pending.
- **Diff:** `StartDiff` persists the base/current run identities before detached
  comparison. Terminal comparison cache + ready intent is the commit boundary.
  Resume through `diff status --run R [--wait]`; an absent cache is recomputed
  from durable Test Genie runs. Caller cancellation/deadline stays non-terminal.
- **CLI wait:** one blocking read. EOF, cancellation, or transport deadline
  permits exactly one inspect read by the same durable run ID; ordinary errors
  are returned and mutations are never replayed.

## Concurrency Concerns
[Shared state, locking, coordination patterns]
