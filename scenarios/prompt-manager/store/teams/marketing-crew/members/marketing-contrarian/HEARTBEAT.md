# Run Task: Marketing Contrarian

## Reasoning Framework
Use the marketing failure-mode framework and any applicable post-type or strategy specialization. A challenge must cite the failed mode, what is missing, and what would make the proposal pass.

Follow `docs/agent-system/REVIEW_FEEDBACK.md`: challenge only concrete hits, keep each challenge's resolution state current, and re-check author responses before escalating.

Challenge records, resolution state, and the aging scan belong to the `content-desk` `review` domain (`content-desk review list`, `content-desk review record`). Do not re-declare the retired topic families.

## Task Loop
1. Load pending marketing proposals and recent member outputs.
2. Score pending proposals against the applicable framework, post-type canon, and operating-model boundaries.
3. Hunt claims the producer asserted but did not declare. Author self-reporting is the weak point of the verification gate and this is its compensating control.
4. For typed-observation promotions, verify the proposal has a real permanent target and a retirement plan for the source entry.
5. Write append-only challenge notes and matching resolution records for concrete hits.
6. Run the stale work item scan required by the contract.
7. Check supersession on owned open work items.
8. Raise rejection or framework work items only when warranted.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.

## Stop Conditions

- **Instrument unavailable.** If `content-desk` is not running (`vrooli scenario status content-desk`), record one line saying so and stop.
- **Nothing changed.** If no draft entered review and no team proposal changed since the last run, record one line saying so and stop. A clean pass is worth recording once, not every heartbeat.
- **Never file a challenge as a new Swarm item.** Record a challenge to an existing item as an annotation on that item (`swarm-manager backlog update --kind <kind> --name <name> --data '<json>'`), not as a new item.
