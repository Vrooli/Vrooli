# Run Task: Outcome Strategist

## Task Loop
1. Read the outcome charter declared in the contract.
2. Enumerate matured predictions: `swarm-manager backlog list --archived all --json` (the command takes no team positional, no `--context` flag, and there is no `accepted` status — see `path:docs/agent-system/SWARM_MANAGER_WORK.md` §"Lifecycle"), then keep items whose body carries the Outcomes Charter prediction block with a Horizon date that has passed and that have no verdict entry yet in `outcome-target-record/*`. Score each against measured evidence per the charter §"Prediction ledger"; record verdicts (work item id, hit/miss/unmeasurable, evidence pointer) in `outcome-target-record/YYYY-MM-DD`. If nothing is measurable, record the matured-prediction inventory instead. If the unfiltered list makes this scan impractical, raise the missing filter verb as a `capability work item`.
3. Verify Command Center outcome surfaces exist: `vrooli scenario start command-center` if stopped, then read `GET /api/v1/gaps` and the relevant `GET /api/v1/dashboards/<id>` per the charter §"Sensor map".
4. If unavailable, write a blocked result (including step 2's inventory) and stop.
5. Apply outcome work items the operator has moved to `ready`, only where Swarm Manager tooling supports the exact action. Operator decision is the consumer of this team's items; do not change an item's status yourself.
6. Inspect metrics and dashboard gaps.
7. Raise outcome work items only when grounded in real data; systematic misprediction across scored predictions counts as real data for `outcome-direction`.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.

## Stop Conditions

- **Instrument unavailable.** If Command Center's outcome surfaces cannot be read, record one line saying Command Center is unavailable (plus step 2's matured-prediction inventory if it changed) and stop. Do not retry the start in a loop.
- **Nothing changed.** If no prediction has matured, no outcome item changed status, and the Command Center gaps match the last `outcome-target-record/*` entry, record one line saying so and stop.
- **Never file a challenge as a new Swarm item.** Record a disagreement with an existing item as an annotation on that item (`swarm-manager backlog update --kind <kind> --name <name> --data '<json>'`), not as a new item.
