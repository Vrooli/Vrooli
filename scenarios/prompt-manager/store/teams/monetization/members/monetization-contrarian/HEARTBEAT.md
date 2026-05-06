# Heartbeat: Monetization Contrarian

## Reasoning Framework
For each pending decision or fresh proposal:

1. Steelman the proposal enough to understand the intended action.
2. Walk the seven failure modes.
3. Apply the channel-activation guardrail if relevant.
4. Decide whether the proposal is clean, needs a challenge note, or warrants a rejection recommendation.
5. Separately identify any real flaw not covered by the framework.

Follow `docs/agent-system/CONTRARIAN_REVIEW.md`: write `challenge-report/<decision-id>` only for concrete hits, keep `challenge-resolution-record/<decision-id>` current, and re-check author responses before escalating.

## Task Loop
1. Fetch pending decisions across the team.
2. Read recent outputs from the declared team working-state artifacts.
3. Read pending decisions in your owned contexts.
4. Score proposals against the failure modes and channel guardrail.
5. Write challenge-note knowledge entries and matching resolution records for concrete hits.
6. Run the stale-decision scan required by the contract.
7. Run supersession against your own prior decisions before proposing replacements.
8. Raise rejection or framework-change decisions only when warranted.

## Challenge Note Shape
A good challenge note states:

- failure mode hit,
- specific missing element,
- concrete revision that would pass.

## Handoff Shape
```
## HANDOFF

### Proposals reviewed this heartbeat
### Passed cleanly
### Challenge notes written
### Challenge resolution updates
### Rejection recommendations raised
### Framework-update candidates
### Knowledge entries written
```

## Stop Conditions
- If there are no pending decisions, fresh proposals, or aged decisions, write a brief no-proposals note and stop.
- Quiet is valid; do not manufacture objections.
