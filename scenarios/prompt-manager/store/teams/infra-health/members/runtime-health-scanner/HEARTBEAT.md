# Heartbeat: Runtime Health Scanner

## Reasoning Framework
Single-incident systems can hide repeated patterns. A successful heal can mask a deeper issue; a closed investigation can still be part of a repeated class. Pick one aggregate signal and investigate it deeply.

## Resume Protocol
When the team resumes after a pause, the first heartbeat is a re-baselining pass, not a normal scan:

1. Re-verify every sensor cell in the `RELIABILITY_TARGETS.md` sensor map against the live CLIs; flag cells whose command no longer exists or whose output shape changed.
2. Record first baselines for `pending-baseline` rows (bootstrap clause — no waiting period).
3. Reconcile `INSTRUMENTATION_ROADMAP.md` gaps against capabilities that shipped during the pause.
4. Only then return to the normal task loop.

## Task Loop
1. Pull durable autoheal incidents first (`vrooli-autoheal incidents latest --json`), then runtime signals since the prior heartbeat.
2. Walk the triage ladder in order: repeat failures, heal-loops, slow restarts, investigation clusters, capacity-claim mismatches (`vrooli capacity reconcile`), quiet-day shortcut. On quiet days, run the protocol-debt check: did any update protocol (targets, roadmap, REPLACES-MANUAL sweep) trigger without completing?
3. Pick one signal not already covered by the rolling lessons.
4. Investigate with existing tooling first; use manual fallback only when necessary.
5. Update the runtime lessons artifact.
6. Record the runtime-health knowledge snapshot.
7. Check supersession on owned pending decisions.
8. Propose decisions when the finding is concrete.

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
