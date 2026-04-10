# Research Conclusion: Define Per-Loop Health Snapshot Contract Using Git Control Tower Reviews

## Research Question
How should ecosystem-manager capture per-loop health snapshots using Git Control Tower reviews, and what schema, trigger contract, regression detection, and storage design should be used?

## Summary
<!-- TBD — will be refined as research progresses -->
Initial investigation confirms that swarm-manager already has a mature GCT integration pattern (trigger → poll → store ReviewResult) that ecosystem-manager can adopt. The key challenge is adapting this from a one-shot post-execution review to a per-iteration review within Auto Steer loops, including delta tracking and regression detection across iterations.

## Methodology
- Examined swarm-manager's GCT integration: `review_client.go` (HTTP client), `finalization.go` (trigger/poll logic), `model.go` (ReviewResult types)
- Examined Git Control Tower's review API: `review_handler.go`, `review_model.go`, `review_readiness.go` (5-dimension model, readiness classification, thresholds)
- Examined ecosystem-manager's loop structure: `processor.go` (main loop), `execution_history.go` (metadata tracking), `autosteer/types.go` (phase/iteration model), `autosteer/db_init.go` (PostgreSQL schema)
- Mapped current execution metadata fields against proposed snapshot fields

## Findings

### Finding 1: Swarm-Manager Review Integration Is a Proven Pattern
The swarm-manager integration follows a clean async pattern:
1. **Trigger:** `POST /api/v1/review/run` with `ReviewRequest{ScenarioName, ExpectedPaths, SandboxID, Thresholds}`
2. **Poll:** `GET /api/v1/review/run/{jobID}` every 5 seconds, 10-minute timeout
3. **Result:** `ReviewResult{JobID, Classification, Dimensions[], Summary, ReviewedAt}`

Classification maps from readiness: green→ready, yellow→ready_with_notes, red→needs_work.

Each `ReviewDimension` has `Name`, `Status` (green/yellow/red/skipped), and `Details`. The 5 dimensions are: codeQuality, tests, standards, visual (screenshots), provenance (AI tracing).

Configurable thresholds (`ReviewThresholds`) control green/yellow/red boundaries per dimension.

**Evidence:** `scenarios/swarm-manager/api/internal/execution/review_client.go`, `model.go:92-105`, `finalization.go:462-541`

### Finding 2: GCT Review API Supports Selective Checks and Details
The review API accepts an optional `checks` array (tidiness, tests, rules) and a `details` parameter controlling response verbosity. This means ecosystem-manager can request lightweight reviews (fewer checks) for per-iteration snapshots if full reviews are too slow.

Only one active review per scenario is allowed (409 Conflict on concurrent requests). Jobs are stored in-memory with 1-hour retention.

**Evidence:** `scenarios/git-control-tower/api/review_handler.go`, `review_store.go`

### Finding 3: Ecosystem-Manager Has No Existing Review Integration
The ecosystem-manager tracks extensive execution metadata (duration, prompt size, tokens, exit reason, Auto Steer phase/iteration/steering source) but has zero GCT integration. Current quality gates use `MetricsSnapshot` (build status, operational targets) but not scenario-level code quality or test results from GCT.

Storage is dual: file-based (`logs/task-runs/{taskID}/executions/{executionID}/metadata.json`) and PostgreSQL (`profile_execution_state`, `profile_executions` tables).

**Evidence:** `scenarios/ecosystem-manager/api/pkg/queue/execution_history.go:16-47`, `autosteer/db_init.go`

### Finding 4: Auto Steer Iteration Model Provides Natural Integration Points
Auto Steer already has:
- `IterationEvaluator.Evaluate()` — called after each iteration, increments counters, collects metrics
- `PhaseCoordinator.ShouldAdvancePhase()` — evaluates stop conditions
- `MetricsSnapshot` — captures state at each iteration boundary
- `PhaseExecution.StartMetrics` / `EndMetrics` — tracks deltas per phase

The natural integration point is within or after `IterationEvaluator.Evaluate()`, where a GCT review could be triggered and its result stored alongside the existing MetricsSnapshot.

**Evidence:** `scenarios/ecosystem-manager/api/pkg/autosteer/iteration_evaluator.go`, `phase_coordinator.go`

### Finding 5: Per-Iteration Full Reviews May Be Too Slow
GCT reviews can take up to 5 minutes (3 parallel checks). Auto Steer iterations can be as short as a few minutes. Running a full review after every iteration could double iteration time and hit the one-review-at-a-time constraint.

Options: (a) review after every iteration but accept the time cost, (b) review only at phase boundaries, (c) review every N iterations, (d) run lightweight checks (e.g., tidiness only) per iteration and full reviews at phase boundaries.

<!-- TBD — waiting for user decision on trigger frequency -->

## Limitations
- Have not yet investigated the performance impact of per-iteration reviews in practice
- Have not determined whether ecosystem-manager needs its own ReviewClient or can reuse swarm-manager's
- Storage schema design (enriching existing tables vs. new snapshot table) is still open
- Regression detection algorithm not yet specified

## Actions
<!-- TBD — actions will be defined as decisions are resolved and findings mature -->
