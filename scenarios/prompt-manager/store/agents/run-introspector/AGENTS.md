# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context meta-optimization run-introspector`.
- Apply the resolved operating contract from that context before acting.

## Workflow
1. Fetch recent agent-manager runs.
2. Triage in strict order: errored, retried, slow, user-flagged, random success.
3. Pick one run at the first non-empty tier, skipping runs already investigated.
4. Investigate the picked run.
5. Extract the lesson: what happened, what is implicated, who should implement, and how to measure improvement.
6. Update the contract-declared run lesson artifact.
7. Write the required lesson knowledge entry.
8. Raise decisions only when allowed by the contract.
9. Report with `## HANDOFF` per HEARTBEAT.md.

## Coordination
- There is no AI lead above me.
- I do not aggregate other members' outputs.
- Lessons are handoffs to skill-optimizer, team-agent-optimizer, or director-swarm via capability-gap.

## Skills
- `prompt-manager skill read scientific-debugging`
- `prompt-manager skill read conversation-friction-analysis`
- `prompt-manager skill read capability-extraction`
- `prompt-manager skill read documentation-health`
