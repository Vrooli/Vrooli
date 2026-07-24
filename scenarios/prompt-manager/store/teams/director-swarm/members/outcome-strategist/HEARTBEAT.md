# Heartbeat: Outcome Strategist

## Task Loop
1. Read the outcome charter declared in the contract.
2. Enumerate matured predictions: `prompt-manager team decision-list director-swarm --context goal-proposal --status accepted --json` (repeat for `goal-portfolio` and `outcome-direction`), then keep decisions whose prediction-block Horizon date has passed and that have no verdict entry yet in `outcome-target-record/*`. Score each against measured evidence per the charter §"Prediction ledger"; record verdicts (decision id, hit/miss/unmeasurable, evidence pointer) in `outcome-target-record/YYYY-MM-DD`. If nothing is measurable, record the matured-prediction inventory instead. If accepted-decision volume ever makes this scan impractical, raise the missing horizon-filter verb as a `capability-gap`.
3. Verify Command Center outcome surfaces exist: `vrooli scenario start command-center` if stopped, then read `GET /api/v1/gaps` and the relevant `GET /api/v1/dashboards/<id>` per the charter §"Sensor map".
4. If unavailable, write a blocked handoff (including step 2's inventory) and stop.
5. Apply accepted outcome decisions where supported.
6. Inspect metrics and dashboard gaps.
7. Raise outcome decisions only when grounded in real data; systematic misprediction across scored predictions counts as real data for `outcome-direction`.

## Handoff Shape
### Outcome signals
### Prediction scores
### Dashboard gaps
### Applied accepted decisions
### Recommendations needing approval
### Knowledge entries written
