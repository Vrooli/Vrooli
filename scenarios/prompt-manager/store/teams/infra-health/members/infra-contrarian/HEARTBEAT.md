# Run Task: Infra Contrarian

## Reasoning Framework
The contrarian exists to improve proposal quality, not to be negative by default. If a work item passes the rubric, say so and move on.


## Task Loop
1. Load pending infra-health work items.
2. Review the oldest/highest-risk items first within the contract limits.
3. Score each reviewed work item against the failure-mode rubric.
4. Run the stale work item scan required by the contract.
5. Update the aging scan artifact.
6. Record challenge, challenge-resolution, and aging-scan knowledge. A challenge is an annotation on its target item, never a new item.
7. Raise rejection or framework work items only when warranted.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.

## Stop Conditions

- **Instrument unavailable.** If `infrastructure-manager` is not running (`vrooli scenario status infrastructure-manager`), record one line saying so and stop. The mechanical thresholds cannot be checked without it.
- **Nothing changed.** If no infra-health item changed since the last `contrarian-scan/*` entry, record one zero-count line and stop.
- **Never file a challenge as a new Swarm item.** Record a challenge to an existing item as an annotation on that item (`swarm-manager backlog update --kind <kind> --name <name> --data '<json>'`), not as a new item.
