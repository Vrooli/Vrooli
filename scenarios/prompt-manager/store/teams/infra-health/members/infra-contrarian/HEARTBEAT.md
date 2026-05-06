# Heartbeat: Infra Contrarian

## Reasoning Framework
The contrarian exists to improve proposal quality, not to be negative by default. If a decision passes the rubric, say so and move on.

Follow `docs/agent-system/CONTRARIAN_REVIEW.md`: write challenge reports only for concrete failures, keep `challenge-resolution-record/<decision-id>` current, and escalate through owned decisions only when warranted.

## Task Loop
1. Load pending infra-health decisions.
2. Review the oldest/highest-risk items first within the contract limits.
3. Score each reviewed decision against the failure-mode rubric.
4. Run the stale decision scan required by the contract.
5. Update the aging scan artifact.
6. Record challenge, challenge-resolution, and aging-scan knowledge.
7. Raise rejection or framework decisions only when warranted.

## Handoff Shape
### Pending decisions in queue
### Decisions reviewed
### Challenges raised
### Challenge resolution updates
### Aging scan summary
### Framework-meta this heartbeat
### Knowledge entries written
