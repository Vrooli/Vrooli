### Runs in window
- Errored: 1
- Retried: 0
- Slow: 1 (`e15d9639` work-duration 15,865s, swarm-manager backlog execute brand-manager-rule-detail-ux — not picked; tier-1 wins)
- User-flagged: 0
- Successful: 72 complete + 3 needs-review + 2 still running

### Run picked this heartbeat
- Run ID: `cab1c399-9a3e-4b12-99e4-6a21dcb69ddb`
- Agent: `swarm-manager:backlog:execute:prompt-manager-decision-deferral-primitive:process` (sandboxed, opus, fallback_runner_types=[CLAUDE_CODE, CODEX])
- Triage tier: errored (tier-1)

### What happened
Run failed after 1 turn / 203,961ms with `API Error: 529 Overloaded. This is a server-side issue, usually temporary`. claude-code returned `is_error=true subtype=success`; declared codex fallback never engaged. Sole `RUN_STATUS_FAILED` in a 78-run window.

### Implicated
- **Meta (in lane):** `run-introspector/HEARTBEAT.md` tier-1 — currently-pending gate (`dec-1777069916962818847`) covers only 429-false-positives, not 5xx-transient failures with `turns_used<=1`.
- **Scenario-qa (out of lane, noted in RUN_LESSONS.md only):** claude-code runner's terminate-on-error path does not engage `fallback_runner_types` on transient 5xx; `subtype=success` masks the upstream API failure as non-retryable.

### Proposed lesson
- Extend the already-pending tier-1 gate to also reclassify `API Error: 5xx Overloaded/...` 1-turn runs as tier-5.
- Handoff to: **team-agent-optimizer** (HEARTBEAT.md edit; coordinate with `dec-1777069916962818847`).

### Measurement plan
- Baseline: 1/78 (~1.3%) of completed runs in this window are pure-transient-5xx failures.
- 2026-05-02 (7 HBs): grep RUN_LESSONS.md → 0 tier-1 lessons opened on `API Error: 5xx`-only runs.
- **Standing pattern.** Third tier-contamination class in three heartbeats (tier-1 detectRateLimit, tier-3 approval-lag, tier-1 5xx-transient). All same shape: triage fires per definition; signal is environmental, not agent-behavioral. Per `dec-1777070860432410408`'s standing-pattern note, contrarian should now evaluate `framework-update` for "tier-signal-contamination" as a standing failure mode. Not raised by me (out of lane).

### Decisions raised this heartbeat
- `dec-1777157323547139809` · `run-lesson` · tier-1 gate extension to cover 5xx-transient-API failures → team-agent-optimizer
- (1 of ≤2 cap used; no second decision — pure-scenario-qa observation noted in RUN_LESSONS.md without a decision since it falls outside run-introspector's owned contexts)

### Knowledge entries written
- `knw-1777157297255514121` · topic `run-lessons-2026-04-25` (supersedes `run-lessons-2026-04-24`)

### Supersession check
- Prior `dec-1777070860432410408` (tier-3 work-duration) and `dec-1777069916962818847` (tier-1 429-FP) both pending; this decision is **additive** (different sub-class within tier-1 / different tier). No supersession.
- Team queue: 6 → 7 pending; under 12-ceiling. Read-only mode not triggered.