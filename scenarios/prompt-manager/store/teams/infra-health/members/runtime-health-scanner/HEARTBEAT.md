# Heartbeat: Runtime Health Scanner

## Reasoning Framework
Single-incident systems can hide repeated patterns. A successful heal can mask a deeper issue; a closed investigation can still be part of a repeated class. Pick one aggregate signal and investigate it deeply.

## Resume Protocol
When the team resumes after a pause, the first heartbeat is a re-baselining pass, not a normal scan:

1. Re-verify every sensor cell in the `RELIABILITY_TARGETS.md` sensor map against the live CLIs; flag cells whose command no longer exists or whose output shape changed.
2. Clean the alarm channel before recording anything: reconcile the check registry against the installed scenario/resource set — both directions: ghost checks (→ retire finding) and unsupervised members of the derived should-be-supervised set (→ coverage finding) — and sweep for saturated checks (single status ≥24h → one durable finding, then shelve with expiry or retire). The `vrooli-autoheal check shelve` / `check reconcile` verbs are Gap 10 and not shipped yet — until then, record shelves and the reconcile diff manually per `RELIABILITY_TARGETS.md` § Sensor integrity. Baselines recorded over a polluted channel are dishonest.
3. Record first baselines for `pending-baseline` rows (bootstrap clause — no waiting period).
4. Reconcile `INSTRUMENTATION_ROADMAP.md` gaps against capabilities that shipped during the pause.
5. Only then return to the normal task loop.

## Task Loop
1. Pull durable autoheal incidents first (`vrooli-autoheal incidents latest --json`), then runtime signals since the prior heartbeat.
2. Walk the triage ladder in order: repeat failures, heal-loops, slow restarts, investigation clusters, capacity-claim mismatches (`vrooli capacity reconcile`), alarm-channel integrity (ghost or saturated checks, expired shelves, flood target out of band — see sensor-integrity rule), supervised-set coverage (unsupervised members of the derived should-be-supervised set — manual diff until the Gap 10 reconcile extension ships), capability availability (owner-derived aggregates out of band, once Gap 11 ships persistence; repeated unabsorbed degradation escalates per operating-model rule 1), validation cost and cache reliability (`test-genie runs cost --window 7d --json`, including reliable-sample composition, calibration freshness, cache-hit rate, audit/demotion counts, and net saving), quiet-day shortcut. Respect cascade discipline: skip an outer tier while an inner tier holds an unresolved excursion. On quiet days, run the protocol-debt check: did any update protocol (targets, roadmap, REPLACES-MANUAL sweep, actuation-efficacy re-reads) trigger without completing?
3. Pick one signal not already covered by the rolling lessons.
4. Investigate with existing tooling first; use manual fallback only when necessary.
5. Update the runtime lessons artifact.
6. Record the runtime-health knowledge snapshot.
7. Check supersession on owned open work items.
8. Propose work items when the finding is concrete.

For validation-cost scans, treat the Test Genie report as the source of truth. Do not rerun a comprehensive suite merely to obtain a cost sample when the report already has a recent reliable sample. If calibration is due, record the scheduler work item and wait for the server-owned run once; do not start a duplicate run because the cost command is slow.

## Handoff Shape
### Window inspected
### Signal counts
### Signal picked
### Pattern observed
### Hypothesized root cause
### Proposed action
### Measurement plan
### Missing CLI or telemetry surfaces
### Work items filed
### Knowledge entries written
