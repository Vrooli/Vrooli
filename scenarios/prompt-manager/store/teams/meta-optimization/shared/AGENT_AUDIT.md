# Agent Audit

Rolling audit of agent files (AGENTS.md, SOUL.md, TOOLS.md, agent.json) maintained by `team-agent-optimizer`. Complements `TEAM_AUDIT.md`.

## How to read this file
- Rows are ordered by most-recent visit (newest at top).
- `Rating` is a rough 1–5 (5 = sharp, scoped, evidence-rich; 1 = drifted or misaligned).
- `Disposition` is the heartbeat's conclusion: `no-action`, `improve`, `prune`, or `restructure-implication`.

## Rotation state
- Last visited: `quality-auditor` (2026-04-23)
- Coverage: 1/21 agents visited since audit start.

## Recent entries

| Visited    | Agent            | Health | Rating | Disposition                    | Notes |
|------------|------------------|--------|--------|--------------------------------|-------|
| 2026-04-23 | quality-auditor  | 0.56   | 2      | improve (add TOOLS.md)         | Only skillless agent; AGENTS.md names 11 steer skills but TOOLS.md absent; agent.json fileOrder omits TOOLS.md. Peer programmatic-qa-runner (same team) has triad. Decision dec-1776983541260124317 raised. |
