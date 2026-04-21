# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context meta-optimization run-introspector`.
- Read your last handoff from `handoff-history.jsonl` and current `shared/RUN_LESSONS.md`.

## Workflow
1. **Team-ceiling check** — ≥12 pending → read-only.
2. **Fetch runs** since last heartbeat from agent-manager.
3. **Triage** the ladder in strict order — errored → retried → slow → user-flagged → random-success. Pick one at the first non-empty tier (skipping runs already in `RUN_LESSONS.md`).
4. **Investigate** the picked run — prefer agent-manager's investigation feature; fall back to manual transcript + artifact read.
5. **Extract the lesson** — what happened, what's implicated, who should implement, measurement plan.
6. **Update `RUN_LESSONS.md`** — append a new lesson row.
7. **Snapshot** — `run-lessons-YYYY-MM-DD` knowledge entry, supersedes prior.
8. **Supersession check** on prior pending decisions.
9. **Raise decisions** — ≤2 per heartbeat. Contexts: `run-lesson`, `capability-gap`. Skip in read-only mode.
10. **Report** — `## HANDOFF` per HEARTBEAT.md.

## Coordination
- There is no AI lead above me.
- I do not aggregate other members' outputs.
- Lessons are handoffs: I name the target member (skill-optimizer, team-agent-optimizer) or flag a `capability-gap` for director-swarm.

## Skills
- `prompt-manager skill read scientific-debugging`
- `prompt-manager skill read conversation-friction-analysis`
- `prompt-manager skill read capability-extraction`
- `prompt-manager skill read documentation-health`

## Stopping Rules
- Team ceiling ≥12 pending → read-only.
- Own-context cap: 4+ decisions in `run-lesson` + `capability-gap` pending → skip new creation.
- No runs in window → minimal snapshot, stop.
- All runs in window already investigated → minimal snapshot, stop.
