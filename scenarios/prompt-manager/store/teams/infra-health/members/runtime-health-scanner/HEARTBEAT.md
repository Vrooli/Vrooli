# Run Task: Runtime Health Scanner

## Reasoning Framework
Single-incident systems can hide repeated patterns. A successful heal can mask a deeper issue; a closed investigation can still be part of a repeated class. Pick one aggregate signal and investigate it deeply.

## Resume Protocol
When the team resumes after a pause, the first heartbeat is a re-baselining pass, not a normal scan:

1. Read the instrument's own integrity first: `infrastructure-manager coverage validate --json` and `coverage drift --json`. A setpoint-integrity or drift finding means the board itself is wrong; fix that before reading anything off it.
2. Clean the alarm channel before recording anything: `vrooli-autoheal check reconcile --json` gives all three directions at once — ghost checks (target gone → retire finding), out-of-scope checks (target exists, outside the core set → supervision-scope finding, and their readings still count), and unsupervised plant (core-set member with no check → coverage finding). Then `vrooli-autoheal check saturation --json` for the whole registry in one read. If `ghostDetectionAvailable` is false, no check may be called a ghost that heartbeat. Baselines recorded over a polluted channel are dishonest.
3. Read `infrastructure-manager condition trust --json`. If most readings are not `VALID`, the channel is still dirty and step 2 is not finished.
4. Record first baselines for cells whose band verdict is `NEEDS_BASELINE` (bootstrap clause — no waiting period). A `NOT_GRADEABLE` verdict is not a missing baseline: it means the bar authors no threshold, which is a setpoint review item for the operator, not a reading to take.
5. Only then return to the normal task loop.

## Task Loop
1. Read the control-plane failure population before the ranked surface: `vrooli scenario list --json` and retain every `start-failed` item with its reason and timestamp. Read dependency drift with `scenario-dependency-analyzer drift --json`; a lockfile drift entry is a triage item even when its scenario is currently off. Do not start a scenario as a repair.
2. Read the ranked surface: `infrastructure-manager focus next --json`. It merges out-of-band readings, untrusted readings, open-loop cells, coverage drift and source unavailability into one queue already ordered by cascade stage, and states which stage it applied. Check `sources` in the same response: a source reporting `available: false` means that whole finding class was not looked at, so an empty queue is not a quiet day.
3. Take the top-ranked finding whose stage is the innermost unresolved tier. Do not skip down the list to a more interesting outer-tier item — that is the cascade violation the rubric fails mechanically. Pull the evidence with `infrastructure-manager condition explain <cell-ref> --json`, and the durable incident behind it with `vrooli-autoheal incidents latest --json`.
4. A lockfile drift is trivially repairable only with `scenario-dependency-analyzer deps resync --scenario <name> --surface <surface>` followed by explicit `--apply`. A start failure, code defect, or missing sensor is report-only. Never edit code or run a raw package manager from this lane.
5. Only when `focus next` is genuinely empty and every source is available, fall back to the wider sweep: capacity-claim mismatches (`vrooli capacity reconcile`), validation cost and cache reliability (`test-genie runs cost --window 7d --json`, including reliable-sample composition, calibration freshness, cache-hit rate, audit/demotion counts, and net saving), and capability availability aggregates. On quiet days, run the protocol-debt check: did any update protocol (setpoint review, REPLACES-MANUAL sweep, actuation-efficacy re-reads) trigger without completing?
6. Pick one signal not already covered by the rolling lessons.
7. Investigate with existing tooling first; use manual fallback only when necessary.
8. Update the runtime lessons artifact.
9. Record the runtime-health knowledge snapshot.
10. Check supersession on owned open work items.
11. Propose work items when the finding is concrete. Name the sensor the fix must move (`sensor_ref` on the finding) so actuation efficacy can be re-read afterwards.

For validation-cost scans, treat the Test Genie report as the source of truth. Do not rerun a comprehensive suite merely to obtain a cost sample when the report already has a recent reliable sample. If calibration is due, record the scheduler work item and wait for the server-owned run once; do not start a duplicate run because the cost command is slow.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.
