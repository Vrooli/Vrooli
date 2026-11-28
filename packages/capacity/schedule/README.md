# Capacity-aware deferred-work scheduler

`packages/capacity/schedule` is the shared scheduler for compute-heavy work that
does not need to run now: conversation embeddings, run annotations, backfills,
and any other batch plane that would otherwise build a private sleep loop.

The capacity broker decides **whether** an operation may run. It deliberately
owns no queue. This package owns the queue, the waiting, the batching, the
fairness, and the durable record of what happened — and still asks the broker
before every batch. There is exactly one admission policy in the system.

```
producer ──Submit──> Store (SQLite: jobs, runs, model dwell)
                        │
                   Scheduler.Tick
                        │  ready → group → rank → trim
                        ▼
                   Admission ──> packages/capacity.Broker
                        │             (RAM/CPU/VRAM, priorities, leases)
                        ▼
                   Executor (embeddings, annotations, …)
                        │
                   Checkpoint / Complete / Defer
```

## Why it exists

- **Waiting is durable.** A job survives a restart; a lease that expires is
  requeued with its attempt count and checkpoint intact.
- **Models are loaded once.** Jobs are grouped by kind, model affinity, and batch
  key. A group whose model is still resident inside `ModelDwell` is preferred, so
  a 200-item embedding backfill does not reload the model 200 times.
- **Compute is bounded.** Modes, windows, rolling-hour budgets, and a hard
  `MaxSharePerOwner` keep background work from crowding out interactive use.
- **Admission is not reimplemented.** Denials are deferrals carrying the broker's
  own reason. Deferred work always asks without reclaim, so it never evicts a
  resident interactive model.
- **Operators can see why.** `Status` reports depth by kind/model/owner, the
  oldest due job, counts per deferral reason, remaining budget, the window and
  idle gate, and the last run's actual versus estimated cost.

## Public API

```go
store, err := schedule.NewStore(ctx, schedule.Config{Path: "/path/schedule.db"})
defer store.Close()

// Producers submit idempotently by job ID.
job, err := store.Submit(ctx, schedule.Job{
    Kind:          "embed",
    Owner:         "agent-manager",
    Priority:      schedule.PriorityBatch,
    ResourceClass: schedule.ResourceClassCompute,
    ModelAffinity: "nomic-embed-text",
    Estimate:      schedule.Estimate{Seconds: 2, RAMBytes: 512 << 20, Items: 32, Tokens: 4096},
    Value:         1,
    PayloadRef:    "runs:backfill:2026-11",
})

broker, err := capacity.NewBroker(ctx, "")            // existing admission seam
admission, err := schedule.NewBrokerAdmission(broker, schedule.BrokerAdmissionOptions{
    OwnerPrefix: "schedule:agent-manager:embeddings",
})

scheduler, err := schedule.New(schedule.SchedulerConfig{
    Store:     store,
    Policy:    policy,
    Admission: admission,
    Executor:  executor,                               // your batch worker
    Observer:  schedule.BrokerObserver{Source: broker}, // idle gating
})
go scheduler.Run(ctx)                                 // ticks every Policy.Interval
// or scheduler.Tick(ctx) for one pass, which is what tests use.

status, err := scheduler.Status(ctx)
```

`Store` exposes `Submit`, `SubmitBatch`, `Lease`, `Complete`, `Defer`, `Cancel`,
`List`, `Get`, `Checkpoint`, `ReapExpiredLeases`, `ExpireOverdue`, `Budget`,
`HotModels`, `Ready`, and `Status`.

Executor writes use the `Checkpointer` passed to `CheckpointExecutor`; it is
fenced to that job attempt. Direct `Store.Checkpoint` writes are limited to
waiting jobs, so an expired worker cannot replace a successor's progress.

`Executor` runs one job. Implement `CheckpointExecutor` to save progress
mid-batch:

```go
type Executor interface {
    Execute(ctx context.Context, job schedule.Job) (schedule.Result, error)
}

type CheckpointExecutor interface {
    ExecuteWithCheckpoint(ctx context.Context, job schedule.Job, cp schedule.Checkpointer) (schedule.Result, error)
}
```

`Job.Checkpoint` is passed back on the next attempt, after a failure or after a
crashed runner's lease expires.

## Modes

| Mode | Background fill | Windows | Budgets | Idle required |
|---|---|---|---|---|
| `off` | no | — | — | — |
| `passive` | no; only interactive-priority jobs | ignored | ignored (operator asked now) | no |
| `idle` | yes | yes | yes | yes |
| `balanced` | yes | yes | yes | only if `RequireIdle` |
| `eager` | yes | ignored | yes | only if `RequireIdle` |

An empty window set means "always open" for balanced/idle. Configure a window to
actually shift work to off-hours. A window crosses midnight when its end is
earlier than its start, and is attributed to the day it starts on, so a
Saturday 22:00–02:00 window includes Sunday 01:00.

Budgets are rolling-hour caps (`MaxSecondsPerHour`, `MaxJobsPerHour`,
`MaxModelLoadsPerHour`). Zero means unlimited. `MaxSharePerOwner` caps one
owner's fraction of seconds started in that hour: inside a tick, a group whose
owner is at its share yields to a remaining group still below it. If every
remaining group is at its share, the gate stands down rather than deadlocking
the queue.

## Agent Manager integration

Agent Manager runs two planes with different compute shapes, both compiled into
the same job envelope (ADR-007 in the work folder). Each plane gets its own
store namespace by `Kind` and its own admission namespace by `OwnerPrefix`; they
may share one scheduler database or use two.

### Retrieval (conversation embeddings)

```go
policy := schedule.Policy{
    Mode:                 schedule.ModeBalanced,
    Location:             time.Local,
    Windows:              []schedule.Window{offHours}, // e.g. 22:00–07:00
    MaxSecondsPerHour:    900,   // 15 minutes of model time per hour
    MaxModelLoadsPerHour: 2,     // nomic-embed-text is cheap; avoid cold churn
    MaxJobsPerHour:       60,
    ModelDwell:           30 * time.Minute,
    MaxSharePerOwner:     1.0,   // one owner: no cross-owner contention
}

job := schedule.Job{
    Kind:          "embed",
    Owner:         "agent-manager",
    Priority:      schedule.PriorityBatch,
    ResourceClass: schedule.ResourceClassCompute,
    ModelAffinity: "nomic-embed-text",
    BatchKey:      runID,          // batch the chunks of one run together
    MaxBatch:      64,
    Estimate:      schedule.Estimate{Seconds: 3, RAMBytes: 768 << 20, Items: 64, Tokens: 8192},
    PayloadRef:    "run:" + runID,
    Value:         1,
}
```

Backfill horizon: the producer decides how far back to submit, and that decision
is data, not scheduler policy. A practical retrieval horizon is **30 days or the
last N runs, whichever is smaller**, because semantic search is evaluated on
recent work; anything older is still reachable by lexical search. Submit in
run-sized jobs with `BatchKey=runID` so a crash resumes one run, not the whole
backfill. Re-embedding after a model change uses the same envelope with a larger
`Value` and `PriorityService`, so it moves ahead of cold history inside the same
budget.

### Annotation (decision-model labelling)

```go
policy := schedule.Policy{
    Mode:                 schedule.ModeIdle,
    Location:             time.Local,
    Windows:              []schedule.Window{overnight, weekend},
    RequireIdle:          true,
    IdleLoadBelow:        2.0,
    IdleRAMFreeBytes:     8 << 30,
    MaxSecondsPerHour:    600,
    MaxModelLoadsPerHour: 1,      // Winnow is expensive to load and must fit VRAM
    MaxJobsPerHour:       120,
    ModelDwell:           30 * time.Minute,
}

job := schedule.Job{
    Kind:          "annotate",
    Owner:         "agent-manager",
    Priority:      schedule.PriorityBatch,
    ResourceClass: schedule.ResourceClassVRAM, // broker GPU admission + identity
    ModelAffinity: "winnow:e4b",
    BatchKey:      packID,                     // one label pack per batch
    MaxBatch:      32,
    Estimate:      schedule.Estimate{Seconds: 8, VRAMBytes: 6 << 30, Items: 32, Tokens: 16384},
    PayloadRef:    "annotation:run:" + runID,
    Value:         3,                          // richer signal than an embedding
}
```

Backfill horizon: annotations are candidate-filtered and sampled (ADR-009), so
the horizon is **the candidate census plus the sampling window** — typically the
last 14 days, plus any run a friction report explicitly asks about. Annotation
jobs carry a `Deadline` when a report needs them; the scheduler fails overdue
work loudly instead of silently producing a stale report.

`ResourceClassVRAM` requires `BrokerAdmissionOptions.GPUUUID`. The scheduler will
not guess a device, and a missing identity is a typed denial rather than a
silent CPU run.

### Operational reading

`Status` is the integration surface for `agent-manager` CLI/health:

```json
{
  "mode": "balanced",
  "queue": {"depth": 412, "byKind": {"embed": 400, "annotate": 12},
             "byModel": {"nomic-embed-text": 400, "winnow:e4b": 12},
             "oldestDue": "2026-11-18T09:00:00Z"},
  "deferrals": [{"reason": "admission queue: VRAM contention", "count": 12}],
  "budget": {"secondsUsed": 120, "secondsRemaining": 780,
              "jobsUsed": 4, "jobsRemaining": 56,
              "modelLoads": 1, "modelLoadsRemaining": 1},
  "window": {"configured": true, "open": true},
  "lastRun": {"estimatedSeconds": 3, "actualSeconds": 2.4, "outcome": "completed"}
}
```

## Testing

Tests use a temporary SQLite database and a fake admission/executor; no live
host, GPU, or model is required:

```
go test ./packages/capacity/schedule
```

Coverage includes mode gating, windows, budget exhaustion, model grouping and
dwell, fair share, denial/defer/backoff, checkpoint/resume, crash recovery via
lease expiry, status accuracy, and priority ordering.

## Boundaries

- This package never mutates the broker's claim ledger directly.
- This package never imports a scenario; it is scenario-agnostic.
- Admission policy, priorities, and reclaim semantics stay in
  `internal/capacity` and `packages/capacity`.
- Jobs are claimed by one runner at a time through a durable lease. Two
  schedulers may run against one database; the loser of a lease race records
  `lease: contended` and retries on its next tick.
