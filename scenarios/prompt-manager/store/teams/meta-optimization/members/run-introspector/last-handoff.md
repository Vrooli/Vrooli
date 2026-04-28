### Runs in window
- Errored: 6 (3 silent-stall cluster — picked by twin heartbeat; 1 max-turns — picked here; 1 5xx Overloaded; 1 swarm-manager workshop-decision-triage)
- Retried: 0
- Slow: 0 (with the pending tier-3 work-duration gate; raw wall-clock not re-investigated)
- User-flagged: 0
- Successful: 116 complete + 9 needs-review + 2 running

### Run picked this heartbeat
- Run ID: `56398acb-2aec-4b2c-bfc4-eafc8dc28c3e`
- Agent: `heartbeat-meta-optimization-run-introspector-2026-04-26T22-45-00Z` (run-introspector's own prior heartbeat)
- Triage tier: **errored** (tier-1)

### What happened
The prior run-introspector heartbeat ran 51 turns / 7m36s and was killed by `subtype=error_max_turns`. It had drafted the 2026-04-26 silent-stall lesson (visible uncommitted in `git diff`) and was mid-supersession of `dec-1777157323547139809` when it hit the cap. Its sequence-127 message identified the actionable insight: the 5xx gate's `turns_used <= 1` predicate is too narrow — confirmed by run `13ac79cb` (34-turn `overloaded_error`).

### Implicated
- **Primary (in lane):** `dec-1777157323547139809` predicate `summary.turns_used <= 1` — wrong filter; 5xx-pattern alone is the right signal.
- **Secondary (in lane):** run-introspector `HEARTBEAT.md` Required Loop is bumping the 50-turn ceiling on dual-finding heartbeats (new lesson + supersession). Surfaced; not raised this HB (cap discipline).
- **Out of lane:** Two run-introspector heartbeats fired simultaneously at 2026-04-27T22:45 (runs `096b1dee` + `937bcb50`, 55ms apart, same tag). Scheduler concern; flagged in lesson only.

### Proposed lesson
- Drop `turns_used <= 1` from the pending 5xx environmental-failure predicate; collapse 429-FP / 5xx-pattern / silent-stall into one "tier-1 environmental-failure exclusions" bullet list when team-agent-optimizer implements.
- Handoff to: **team-agent-optimizer** (consolidated tier-1 gate edit; coordinates with `dec-1777069916962818847` and the 2026-04-27 silent-stall lesson).

### Measurement plan
- 2026-05-04 (7 HBs): grep RUN_LESSONS.md → 0 tier-1 lessons opened on `5\d\d.*Overloaded|overloaded_error` runs regardless of turn count. Concrete miss this window: run `13ac79cb` (would slip the original predicate, caught by the broader one).
- Heartbeat turn-budget watch: any further `error_max_turns` on a run-introspector heartbeat within 7 HBs escalates the loop-tightening / max-turns-raise option from secondary to primary.

### Decisions raised this heartbeat
- `dec-1777330324477920142` · `run-lesson` · supersedes `dec-1777157323547139809`; broader 5xx predicate (drops turn-count gate) → team-agent-optimizer
- `dec-1777157323547139809` · marked `superseded` · notes link to `dec-1777330324477920142`
- (1 of ≤2 cap used; secondary HEARTBEAT.md tightening + duplicate-firing observation deferred — both noted in lesson, neither in run-introspector's owned contexts as primary lever)

### Knowledge entries written
- `knw-1777330346419212523` · topic `run-lessons-2026-04-27` (supersedes `run-lessons-2026-04-25`)