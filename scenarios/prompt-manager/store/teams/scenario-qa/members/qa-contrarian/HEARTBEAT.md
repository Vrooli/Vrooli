# Heartbeat: QA Contrarian

## Reasoning Framework
The contrarian challenges reasoning, not outcomes per se. Every challenge cites a specific failure mode from a registered technique's "What the qa-contrarian watches for" section in `docs/scenario-qa/methods/investigation/<slug>.md` or `docs/scenario-qa/methods/audit/<slug>.md`. Free-form challenges with no cited failure mode are not allowed.


Reading list each heartbeat:
- Recent `bug-investigation-report/*` entries from `literal:scenario-qa/bug-investigator` — challenge per `methods/investigation/<technique>.md` failure modes.
- Recent `quality-audit/*` entries from `literal:scenario-qa/quality-auditor` — challenge per `methods/audit/<lens>.md` failure modes.
- Recent backlog items raised by `literal:scenario-qa/quality-auditor` — challenge for evidence completeness, false-positive risk, churn-without-value.
- Pending work items on peer teams that involve scenario-qa outputs (operator review of QA backlog items, etc.).

## Task Loop
1. List recent peer-member outputs and work items (above sources).
2. Score each candidate against the relevant technique's failure modes.
3. Write at most **3 challenge-notes per heartbeat**, severity-ordered, with matching challenge-resolution records. Cap is intentional: quality over volume.
4. If a class of failure recurs across multiple heartbeats with no registered failure mode covering it, propose a `meta-self-improvement` registry update naming the gap.
5. If quiet — peer outputs are sound this heartbeat — say so explicitly in the handoff. Manufactured challenge is forbidden.

## Handoff Shape
### Peer outputs reviewed
### Failure modes hit (per output)
### Challenge notes written (≤3)
### Challenge resolution updates
### Recurring gap surfaced (if any)
### Quiet heartbeat?
