# Agent Audit

Rolling audit of agent files (AGENTS.md, SOUL.md, TOOLS.md, agent.json) maintained by `team-agent-optimizer`. Complements `TEAM_AUDIT.md`.

## How to read this file
- Rows are ordered by most-recent visit (newest at top).
- `Rating` is a rough 1–5 (5 = sharp, scoped, evidence-rich; 1 = drifted or misaligned).
- `Disposition` is the heartbeat's conclusion: `no-action`, `improve`, `prune`, or `restructure-implication`.

## Rotation state
- Last visited: `run-introspector` (2026-04-24)
- Coverage: 2/21 agents visited since audit start.

## Recent entries

| Visited    | Agent            | Health | Rating | Disposition                    | Notes |
|------------|------------------|--------|--------|--------------------------------|-------|
| 2026-04-24 | run-introspector | 0.72   | 3      | improve (HEARTBEAT.md step 3)  | RUN_LESSONS.md 2026-04-23 entry (run `60116710`) explicitly handed off to team-agent-optimizer: add tier-1 false-positive gate. Triage ladder picks `exit_code=429` errored runs whose error_msg is substantive task output (run report mentioning "rate limit"), wasting heartbeats. Baseline 2/22 (~9%) of FAILED runs match. Concrete prose insert into step 3. |
| 2026-04-23 | quality-auditor  | 0.56   | 2      | improve (add TOOLS.md)         | Only skillless agent; AGENTS.md names 11 steer skills but TOOLS.md absent; agent.json fileOrder omits TOOLS.md. Peer programmatic-qa-runner (same team) has triad. Decision dec-1776983541260124317 accepted 2026-04-24 vision walk; awaits implementer. |
