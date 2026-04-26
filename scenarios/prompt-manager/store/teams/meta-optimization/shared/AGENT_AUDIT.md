# Agent Audit

Rolling audit of agent files (AGENTS.md, SOUL.md, TOOLS.md, agent.json) maintained by `team-agent-optimizer`. Complements `TEAM_AUDIT.md`.

## How to read this file
- Rows are ordered by most-recent visit (newest at top).
- `Rating` is a rough 1–5 (5 = sharp, scoped, evidence-rich; 1 = drifted or misaligned).
- `Disposition` is the heartbeat's conclusion: `no-action`, `improve`, `prune`, or `restructure-implication`.

## Rotation state
- Last visited: `run-introspector` (2026-04-25, revisit — fresh handoff)
- Coverage: 2/21 unique agents visited since audit start.

## Recent entries

| Visited    | Agent            | Health | Rating | Disposition                    | Notes |
|------------|------------------|--------|--------|--------------------------------|-------|
| 2026-04-25 | run-introspector | 0.72   | 3      | improve (HEARTBEAT.md tier-3)  | Revisit driven by RUN_LESSONS.md 2026-04-24 lesson on run `564191ef` — tier-3 "Slow" picks contaminated by approval-blocked wall-clock (25/98 successful runs in window were 1-turn $0.09 swarm-manager review runs whose `ended_at == approved_at` after operator batch-cleared 25 approvals at 21:09:5X UTC). Replace tier-3 line in Reasoning Framework: define duration as work-duration (`last_heartbeat - started_at`); exclude `requires_approval=true` and 1-turn cheap runs. Decision dec-1777156591536785033. Combined with prior tier-1 gate (dec-1777069916962818847), clears two of five triage tiers of known contamination. |
| 2026-04-24 | run-introspector | 0.72   | 3      | improve (HEARTBEAT.md step 3)  | RUN_LESSONS.md 2026-04-23 entry (run `60116710`) explicitly handed off to team-agent-optimizer: add tier-1 false-positive gate. Triage ladder picks `exit_code=429` errored runs whose error_msg is substantive task output (run report mentioning "rate limit"), wasting heartbeats. Baseline 2/22 (~9%) of FAILED runs match. Concrete prose insert into step 3. Decision dec-1777069916962818847. |
| 2026-04-23 | quality-auditor  | 0.56   | 2      | improve (add TOOLS.md)         | Only skillless agent; AGENTS.md names 11 steer skills but TOOLS.md absent; agent.json fileOrder omits TOOLS.md. Peer programmatic-qa-runner (same team) has triad. Decision dec-1776983541260124317 accepted 2026-04-24 vision walk; awaits implementer. |
