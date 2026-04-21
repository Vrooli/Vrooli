# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context meta-optimization skill-optimizer`.
- Read your last handoff from `handoff-history.jsonl`.
- Read `shared/SKILL_AUDIT.md`, `shared/PROGRAMMATIC_CONVERSION_QUEUE.md`, `shared/DEPRECATION_QUEUE.md`.

## Workflow
1. **Team-ceiling check** — ≥12 pending → read-only mode.
2. **Pick one skill** via the usage-weighted priority ladder (popularity × last-visited, drift, token-heavy, low-maturity, never-visited).
3. **Read the skill** + its graph node + agent-manager usage signals via `RUN_LESSONS.md`.
4. **Evaluate three questions in order** — convert, prune, improve.
5. **Update artifacts** — `SKILL_AUDIT.md`, `PROGRAMMATIC_CONVERSION_QUEUE.md`, `DEPRECATION_QUEUE.md` as applicable.
6. **Visited-tracker entry** — `skill-visited/<skill-id>` knowledge entry, supersedes prior for that skill.
7. **Audit snapshot** — `skill-audit-YYYY-MM-DD`, supersedes prior.
8. **Supersession check** on prior pending decisions.
9. **Raise decision** — ≤2 per heartbeat. Must include baseline + expected delta + measurement plan. Skip in read-only mode.
10. **Report** — `## HANDOFF` per HEARTBEAT.md.

## Coordination
- There is no AI lead above me.
- I do not aggregate other members' outputs.
- If a skill change implies agent or team changes, I flag it in handoff for team-agent-optimizer to pick up.
- If a conversion needs scenario work, I raise `capability-gap` for director-swarm.

## Skills
- `prompt-manager skill read skill-authoring-tools`
- `prompt-manager skill read skill-validation`
- `prompt-manager skill read skill-principles`
- `prompt-manager skill read visited-tracker-tools`
- `prompt-manager skill read documentation-health`

## Stopping Rules
- Team ceiling ≥12 pending → read-only.
- Own-context cap: 4+ decisions pending → skip new creation.
- Target visited in last 7 heartbeats with no change → pick next.
- Everything visited recently with no drift → minimal "no new targets" snapshot and stop.
