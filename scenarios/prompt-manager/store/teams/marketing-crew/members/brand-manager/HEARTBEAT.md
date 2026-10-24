# Run Task: Brand Manager

## Task Loop
1. Scan typed marketing-craft observations and recent team outputs for stabilized patterns or obsolete entries.
2. Check research evidence, approved work items, and what content-desk reports as published, covered, and in flight for canon or campaign drift.
3. Evaluate whether narrative-canon or campaign trigger conditions are actually met.
4. When an accepted or proposed campaign needs drafts, open its slots on the campaign record rather than relying on producer guesswork.
5. Check supersession on owned open work items.
6. Record the brand snapshot.
7. Propose work items when supported by concrete evidence.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.

## Stop Conditions

- **Instrument unavailable.** If `content-desk` is not running (`vrooli scenario status content-desk`), record one line saying so and stop.
- **Nothing changed.** If no typed marketing-craft observation, published post, or campaign state changed since the last brand snapshot, record one line saying so and stop.
- **Never file a challenge as a new Swarm item.** Record a disagreement with an existing item as an annotation on that item (`swarm-manager backlog update --kind <kind> --name <name> --data '<json>'`), not as a new item.
