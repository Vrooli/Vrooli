# Historical progress

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/test-genie/docs/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
The table retains selected dated milestones. Other milestones and their limitations
remain in the archive; consult it before resuming historical work.

| Date | Author | Status Snapshot | Notes |
|------|--------|-----------------|-------|
| 2026-07-23 | Codex | Pre-phase failure diagnostics clarified | `runs findings` now distinguishes a known run that failed before phase execution from an unknown run. |
| 2026-07-11 | Codex Agent | Execution-backed remediation shipped | Replaced the legacy generation queue and separate fix/requirements-improve spawners with one evidence-bound remediation job lifecycle: immutable execution findings, Agent Manager role selection, server-owned rerun verification, and stable-ID deltas. |
| 2026-07-02 | Codex Agent | Workflow self-smoke repaired | Updated the delegated workflow smoke case to validate Test Genie's own dashboard instead of Browser Automation Studio, linked it to `TESTGENIE-ORCH-P0`, added the selector manifest entry needed by BAS runtime resolution, refreshed the BAS registry entry, and verified `test-genie execute test-genie --phases workflow --json` passes. |
| 2026-07-02 | Codex Agent | Workflow phase provider cutover hardened | Removed the old native playbooks phase runner and its tests, kept only legacy seed/claim compatibility, asserted the default catalog is fully provider-delegated, updated CLI/API/self-health wording, and converted playbooks docs into deprecated aliases that point to the workflow-health-backed `workflow` phase. |
| 2026-06-26 | Codex Agent | Agent wait guidance hardened | Made durable run output more agent-proof without naming specific providers: background/busy runs now print an **Agent wait protocol** block with an ETA-derived `runs wait --json --timeout=<seconds>` command, expected duration, recommended timeout, and interrupted-wait recovery via `pgrep` + `tail --pid`. |
