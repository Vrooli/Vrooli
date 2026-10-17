# Validation Temporal Flows

## Checkpoint Flows

1. `start` resolves scope and persists a queued producer ticket plus every
   typed evidence child. The ticket never starts or waits for GCT/Test Genie;
   the agent follows its producer argv and later calls nonblocking `sync`.
2. Git Control Tower/Test Genie own producer start, wait, recovery, timeout, and
   parking. An interruption is recovered by re-running the producer's printed
   command, never through a Plan Manager wait loop.
3. `sync` reads producer state once. It leaves a ticket pending until every
   oracle is terminal and comparable, then atomically persists the result and
   marks the ticket terminal.
4. `show` is read-only. Restart recovery leaves nonterminal tickets intact and
   never restarts producer work.

## Budget Boundaries

| Budget | Value | Starts when | Outcome on expiry |
|---|---:|---|---|
| Plan Manager API read/write | ordinary request bound | ticket/sync request | never waits for producer completion |
| Git Control Tower/Test Genie wait | producer-defined | producer native wait | producer-defined recovery/parking route |

## Interruption Handling

- Client EOF/cancel: producer work continues; the caller uses the producer's
  native durable recovery command.
- Process restart: nonterminal tickets remain readable and require only producer
  recovery followed by `sync`; Plan Manager never duplicates work.
- Partial producer completion: `sync` records terminal typed evidence and leaves
  unfinished evidence visibly pending.
