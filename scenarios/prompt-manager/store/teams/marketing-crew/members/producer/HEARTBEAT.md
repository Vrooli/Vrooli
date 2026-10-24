# Run Task: Producer

## Task Loop
1. Read the Source Ledger wake, owned open work items, and recent producer knowledge.
2. Ask `content-desk` what work is open against active campaigns. If nothing is open and no evidence is waiting, run a small proactive baseline scan with cited sources.
3. Gather the evidence the open work needs. Route distinct observations to their evidence surface: audience, competitor, hook, workflow, skill, channel, or format.
4. Draft against the open work, declaring each factual assertion as a claim with its evidence as you go.
5. Resolve whatever the verification gate refuses. Do not route around it.
6. Check supersession on owned open work items.
7. Raise coverage, capability, audience, channel-strategy, post-type, or hook-promotion work items only when the evidence rules support them.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.

## Stop Conditions

- **Instrument unavailable.** If `content-desk` is not running (`vrooli scenario status content-desk`), record one line saying so and stop. Do not re-report the same outage in a new item if one is already open.
- **Nothing changed.** If `content-desk` reports no open slot and no evidence is waiting since the last run, record one line saying so and stop.
- **Never file a challenge as a new Swarm item.** Record a disagreement with an existing item as an annotation on that item (`swarm-manager backlog update --kind <kind> --name <name> --data '<json>'`), not as a new item.
