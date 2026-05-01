# Heartbeat: Runtime Health Scanner

Apply the resolved operating contract above.

## Reasoning Framework
Single-incident systems can hide repeated patterns. A successful heal can mask a deeper issue; a closed investigation can still be part of a repeated class. Pick one aggregate signal and investigate it deeply.

## Task Loop
1. Pull runtime signals since the prior heartbeat.
2. Walk the triage ladder in order: repeat failures, heal-loops, slow restarts, investigation clusters, quiet-day shortcut.
3. Pick one signal not already covered by the rolling lessons.
4. Investigate with existing tooling first; use manual fallback only when necessary.
5. Update the runtime lessons artifact.
6. Write the required knowledge snapshot.
7. Check supersession on owned pending decisions.
8. Raise decisions only when allowed by the contract and the finding is concrete.
9. End with HANDOFF.

## Handoff Shape
### Window inspected
### Signal counts
### Signal picked
### Pattern observed
### Hypothesized root cause
### Proposed action
### Measurement plan
### Missing CLI or telemetry surfaces
### Decisions raised
### Knowledge entries written
