### Runs in window
- Errored: 0
- Retried: 0 (no retry pattern detected in tag groups since last heartbeat)
- Slow: 25 (all wall-clock outliers — investigation revealed all are approval-lag artifacts, not true slow runs)
- User-flagged: 0
- Successful: 98
- Running: 1 (this heartbeat)

### Run picked this heartbeat
- Run ID: `564191ef-e1b4-4328-b21c-c83999676123`
- Agent: `swarm-manager:initiative:rev-trigger:review:round-001` (sandboxed, opus, requires_approval=true)
- Triage tier: slow (tier-3) — but the investigation reclassified the entire cluster as a tier-3 *contamination* class

### What happened
The longest run by wall-clock (81,648s ≈ 22h44m) actually performed only ~15s of agent work: `last_heartbeat - started_at = 15s`, 1 turn, 27,523 tokens, $0.09. `ended_at = approved_at` (2026-04-24T21:09:57Z), and 25 same-tag runs ended within ~7 seconds of each other from start times spanning 22h — the operator batch-cleared the approval queue in one sweep. Cross-checked with `86f4378d` (same tag, identical agent work, 13s wall-clock because approval landed in 2ms): wall-clock spread is ~6,000× from approval-lag variance alone.

### Implicated
- `scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md` — "Reasoning Framework" tier-3 ("Slow") defines "duration" without qualifier; the natural reading (and what triage used) is wall-clock, which is contaminated by operator latency whenever `requires_approval=true`.

### Proposed lesson
- Redefine tier-3 "duration" as work-duration (`last_heartbeat - started_at`), exclude `requires_approval=true` runs and `turns_used<=1 AND cost_estimate<$0.20` from slow triage.
- Handoff to: **team-agent-optimizer** (HEARTBEAT.md edit). Underlying agent-manager `ended_at` semantics are correct as-is; not a code bug.

### Measurement plan
- Baseline: 25/98 (~25%) of successful runs in this window are approval-lag artifacts on a single tag with median work-duration ~15s but median wall-clock 64,462s (~4,300× discrepancy).
- Post-edit: future tier-3 picks should not surface `requires_approval=true` 1-turn runs.
- Revisit: 2026-05-01 (7 heartbeats) — grep RUN_LESSONS.md for offending picks; expected 0.
- Standing-pattern watch: this is the second tier-contamination lesson in two heartbeats (after tier-1 `detectRateLimit`). If a third surfaces, contrarian should consider a `framework-update` formalizing tier-signal-contamination.

### Decisions raised this heartbeat
- `dec-1777070860432410408` · `run-lesson` · tier-3 work-duration redefinition for run-introspector HEARTBEAT.md → team-agent-optimizer
- (1 of ≤2 cap used; no second decision — depth over breadth, no separate capability-gap warranted since the fix is meta-layer prose)

### Knowledge entries written
- `knw-1777070838911919368` · topic `run-lessons-2026-04-24` (supersedes `run-lessons-2026-04-23`)

### Supersession check
- Prior pending `dec-1776984436121140045` (tier-1 detectRateLimit gate) is **additive**, not redundant, with today's decision — different tier, different gate, both edits land in HEARTBEAT.md "Reasoning Framework". No supersession.
- Team queue at 4 pending (well under 12-ceiling); read-only mode not triggered.