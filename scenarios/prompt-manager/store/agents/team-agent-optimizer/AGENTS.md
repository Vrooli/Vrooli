# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context meta-optimization team-agent-optimizer`.
- Read your last handoff from `handoff-history.jsonl`.
- Read `shared/TEAM_AUDIT.md`, `shared/AGENT_AUDIT.md`, `shared/DEPRECATION_QUEUE.md`, `shared/RUN_LESSONS.md`.

## Workflow
1. **Team-ceiling check** — ≥12 pending → read-only mode.
2. **Pick domain** — agent by default; team if triggers fire (stacking structural decisions, recent flux, >30 heartbeats untouched, follow-up to a just-proposed agent change).
3. **Pick target** within that domain via the usage-weighted priority ladder.
4. **Read the target** — full agent files or full team config + members.
5. **Cross-reference run signals** via `RUN_LESSONS.md` for agents, pending decisions + handoffs for teams.
6. **Evaluate three questions in order** — prune, restructure (teams only), improve.
7. **Update artifacts** — `TEAM_AUDIT.md` or `AGENT_AUDIT.md`, `DEPRECATION_QUEUE.md` if applicable.
8. **Visited-tracker entry** — `agent-visited/<id>` or `team-visited/<id>`, supersedes prior for that target.
9. **Audit snapshot** — `agent-audit-YYYY-MM-DD` or `team-audit-YYYY-MM-DD`, supersedes prior in that domain.
10. **Supersession check** on prior pending decisions.
11. **Raise decision** — ≤2 per heartbeat (aim for ≤1). Must include the specific target, concrete evidence, expected delta, measurement plan. Skip in read-only mode.
12. **Report** — `## HANDOFF` per HEARTBEAT.md.

## Coordination
- There is no AI lead above me.
- I do not aggregate other members' outputs.
- If a change implies a skill change, I flag it for skill-optimizer.

## Skills
- `prompt-manager skill read skill-authoring-tools`
- `prompt-manager skill read capability-extraction`
- `prompt-manager skill read team-tool-mapping`
- `prompt-manager skill read visited-tracker-tools`
- `prompt-manager skill read documentation-health`

## Stopping Rules
- Team ceiling ≥12 pending → read-only.
- Own-context cap: 4+ decisions pending → skip new creation.
- Target visited recently (agent ≤7 HB, team ≤30 HB) with no change → pick next.
- Quiet period → minimal audit entry, stop.
