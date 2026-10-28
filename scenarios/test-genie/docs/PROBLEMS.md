# Open Issues

## Work ladder

- Rung: W3 (R0 runnable and green)
- W0 evidence: the active validation-coordination plan makes canonical durable validation receipts a Test Genie-owned P0 capability; `OT-P0-003` now states that authority without conflicting with the named Swarm Manager goals.
- W1 evidence: `business-health validate scenario test-genie --json` passed after `TESTGENIE-VALIDATION-IDENTITY-P0` and `TESTGENIE-VALIDATION-RECEIPT-P0` were linked to `OT-P0-003`.
- W2 evidence: `vrooli scenario requirements validate test-genie --json` passed with only an advisory stale-snapshot warning.
- W3 evidence: server-owned contracts run `20260905-022400-cdb57dc6` passed at L4 after the runtime manifest and help parser were repaired. The earlier comprehensive run `20260905-013206-b061dbcf` still records failures in other R0 gates, so the scenario is not yet globally green.
- Measured: 2026-09-05

## Updated 2026-10-28 — Validation digest wall-budget evidence

**Symptom:** Bounded digest runs timed out at 15s twice and at 20s once.

**Confirmed mechanism:** The declared `budget.wall_ms` is enforced by Program
Runtime. The original two runs hit 15000ms; a later 50-receipt run hit 20009ms
under a 20000ms contract. The slow interval inside the Test Genie binding is
unknown: cancelled runs recorded no per-binding invocation events, and owner
runtime logs were unavailable.

**Current setting:** `budget.wall_ms=25000`, selected to leave about 25% headroom
over the latest measured p95 lower bound of 20009ms. The 5-run portfolio sample
was 2 succeeded / 3 failed; its percentile includes deadline-censored failures,
so it does not establish natural completion latency. Two runs under a 30000ms
setting succeeded in 3.242s and 8.920s, both below 15s; the current 25000ms
setting has not been run. This is an evidence-based budget adjustment, not a
confirmed fix for the intermittent delay.

The successful 50-row digest counted 50 terminal receipts, 47 with evidence.
Three historical failed receipts lacked evidence references: two
`IDENTITY_CHANGED` and one `REQUIRED_EVIDENCE_MISSING`. New terminal
transitions without child or external evidence now carry a typed locator to
their canonical Test Genie receipt. Focused regression controls verify the
locator on required and authorized missing-evidence outcomes, and verify that
real child-run evidence remains the reference when available.

The three receipts in the retained 50-row sample are immutable and were not
rewritten. Their historical evidence gap remains under QA investigation
(`knw-1793198896217744114`); the new invariant prevents the same locator gap on
future terminal outcomes. The timeout observation is also filed as
`knw-1793198583299327035` and remains unresolved.

**Evidence:** Research receipt
`/home/matthalloran8/.vrooli/research/topics/validation-quality/runs/prepared-20261028/evidence/test-genie-validation-digest-runtime-and-report-controls.json`.

Work record: `5cc55408-bd63-4b7c-bff8-42f067c2351a`.
Budget reassessment work record: `a1ad9f68-7659-4ad0-ac28-c5e4fa85c780`; Program Runtime task record: `93ecd11c-dc50-4798-97fc-e4eee1d69841`.

## Resolved 2026-07-10 — Descriptor phase additions broke fixed test registries

**Symptom:** The full API suite failed after the descriptor-owned `templates`
phase appeared: catalog guards lacked templates-specific surface/skip entries,
and the orchestration fixture called the live provider because its no-op runner
list named only the older phases.

**Root cause:** Test infrastructure retained three fixed phase registries after
production orchestration became descriptor-driven. The tests therefore required
coordinated Test Genie changes for a provider-owned catalog extension.

**Resolution:** Orchestration fixtures now replace every discovered runner and
detach provider transport generically. Capability guards validate normalized
descriptor ownership without pinning phase surface metadata, and skip-variable
guards derive the stable public name from each machine key. A synthetic future
phase fixture proves planning and execution require no phase registration.

**Owner:** Test Genie descriptor/catalog test architecture.

**Refs:** `api/internal/orchestrator/suite_execution_test.go`;
`api/internal/orchestrator/phases/catalog_guard_test.go`.

## Resolved 2026-07-10 — Foreground execute dropped aggregate duration and unavailable failures

**Symptom:** A lifecycle-managed `vrooli scenario test agent-manager --json`
run published the terminal verdict, phase results, and findings, but its
`phaseSummary.durationSeconds` was zero even though the phase durations were
nonzero. A `provider_unavailable` phase also increased `total` without
increasing any outcome bucket.

**Root cause:** The foreground FollowRun renderer rebuilt its terminal summary
with a private counting loop. Unlike the canonical wait/show summarizer, that
loop did not accumulate phase duration and did not classify
`provider_unavailable` as failed evidence.

**Resolution:** Foreground execute, wait, and show rendering now share the same
phase summarizer for total, outcome counts, duration, and observations. A
regression fixture covers nonzero durations and unavailable providers. Run
`20260711-034608-b6bdcd60` proved the repaired foreground summary and persisted
show projection both contain two failed phases totaling three seconds, while
the findings endpoint returns both phases.

**Owner:** Test Genie CLI durable execution projection.

**Refs:** `cli/execute/durable.go`; `cli/execute/standing.go`;
`cli/execute/durable_test.go`.

## Resolved 2026-07-10 — Terminal wait and show disagreed after durable fallback

**Symptom:** `test-genie runs wait --json swarm-manager 20260710-142937-ae6a753e`
returned terminal `failed` with zero phases and zero duration, while `runs show`
for the same run returned 20 persisted phase results spanning 14:47:02–14:59:57
UTC.

**Historical root cause:** Terminal wait projected the run manager's reduced
live/durable status fallback while show converted the persisted run record; the
two paths did not share a versioned terminal snapshot projector.

**Historical workaround:** Treat terminal wait output with missing phase data as
degraded and inspect the same run with show. Do not relabel the zero-phase
result as a pass or start a replacement run to manufacture baseline evidence.

**Resolution:** New runs atomically persist one schema-versioned terminal
snapshot and both wait and show project it before and after retirement/restart.
Legacy, partial, and corrupt records remain explicitly degraded. The original
run remains a diagnostic fixture; only a fresh exact-SHA lifecycle run may
replace it as authoritative evidence.

**Owner:** test-genie run lifecycle.

**Refs:** `api/internal/app/runs/lifecycle.go`;
`api/internal/app/runs/convert.go`; run `20260710-142937-ae6a753e`.

## Test Gaps — typed artifact catalog live proof

Phase 4 has race-enabled unit and contract coverage for atomic catalogs,
runtime-emitted unknown kinds, legacy discovery, screenshot/video enumeration,
opaque list/detail/byte access, missing files, cross-run IDs, symlink escape,
and active-content headers. Phase 10's copy-first rehearsal also proved repeated
wait/get/catalog projections and opaque resolution over 116 copied historical
Test Genie runs and 2,237 artifacts without changing the copied byte tree; all
116 honestly remained legacy because they predate canonical snapshots and
persisted catalogs. The remaining proof is a clean, lifecycle-managed exact-SHA
run confirming that new ui-health screenshots and workflow-health recordings
are persisted and streamed through opaque routes. Copied historical evidence is
strong migration evidence, but it is not an authoritative post-repair run.

- Closed 2026-08-29: Unit Health's bounded executor used the independently
  resolved managed Go binary while inheriting a foreign host `GOROOT`, allowing
  the selected toolchain to load an incompatible standard library. The executor
  now removes inherited `GOROOT`/`GOTOOLDIR` only for resolved Go commands and
  attributes each waited child to the execution metrics collector. Unit tests
  cover portable Go executable names, environment isolation, preservation for
  non-Go commands, and child RSS attribution. A live execution-backed Unit
  Health validation of `vrooli-autoheal` passed both Go coverage commands and
  the TypeScript coverage command.
- Provider freshness gap observed 2026-08-29: a Test Genie tidiness phase reused
  a running Tidiness Manager whose binary predated the current scenario-seam
  rebasing logic, so repository-only seam budgets were incorrectly applied to
  `vrooli-autoheal`. A lifecycle restart made the same aggregate gate pass with
  an empty applicable-seam audit. Add an exact provider-freshness regression
  proving that source/build-input drift cannot reuse a stale running provider;
  until then, this remains an open Test Genie orchestration gap.

- Detection refactor 2026-05: DB detection lives in `internal/orchestrator/phases/dbdetect`; the silent Postgres+Redis fallback is eliminated. Scenarios with no manifest/godeps/source evidence now provision nothing for workflow seed helpers — fix at the source (declare the resource or add the driver import) rather than reinstating a fallback.
- Historical queue, generation, and separate fix/requirements-improve gaps listed here were retired in July 2026. The supported workflow is now evidence-backed remediation from a completed execution; do not reintroduce those independent paths to address old ledger entries.

# Failure Topography (2025-12-03)
- **Critical flows mapped**
  - *Suite request ingestion*: depends on API payload validation and embedded SQLite writes. Failure modes: invalid requested types/priority (client) vs. DB access failures (infra). Current mitigation: validation errors stay 400 and non-validation paths now emit structured logging; storage failures still bubble a 500—documented for follow-up.
  - *Suite execution orchestrator*: preflight includes provider-backed catalog phases and artifact persistence. Dependencies: filesystem layout, `.vrooli/service.json`, provider availability, language runtimes, package managers, and manifest-declared resources.
- **Observed failure modes**
  - Missing directories or manifest drift silently mapped to 500s before this loop. They are now classified as `misconfiguration` with remediation text so UI/API callers can render contextual actions.
  - Dependency gaps were previously reported one-at-a-time; now the Go phase aggregates all missing commands and surfaces a single actionable error plus per-phase observations to avoid repeated API calls.
  - Optional data (e.g., Node workspaces without lockfiles, manifests without required resources) now degrade gracefully by issuing warnings/observations instead of failing the phase.
- **Remaining risks**
  - Execution persistence is still a single SQL INSERT with no retry or circuit breaker. If the SQLite store cannot be opened or written, the orchestrator returns a 500 even though the phase output is available in memory. Future loop: buffer execution records locally and retry asynchronously.
  - The Go phases emit failure classifications, but the UI/API have not consumed the new integration/perf telemetry yet—wire it into operator dashboards so failure context surfaces to users.
  - The new history endpoints expose per-phase failures to operators, but they still require a manual poll; future work should stream execution status so long-running suites can surface progress before completion.

# Deferred Ideas
- Evaluate which pieces of the archived Go services or CLI should be ported verbatim versus redesigned.
## 2026-09-05 — Default validation wait returned immediately on active receipts

**Symptom:** the Plan Manager-generated command `test-genie validation wait
--wait-id … <receipt> --json` returned an active receipt in under a second even
though its instructions promised one blocking terminal await. This forced
operators toward forbidden polling.

**Root cause:** revision zero was treated as “after revision zero,” so every
persisted receipt satisfied the immediate-return branch. The service also
returned after any nonterminal notification. Explicit revision observers and
the default terminal await had been collapsed into one behavior.

**Resolution:** revision zero now selects a notification-driven terminal await;
an explicit positive `after_revision` retains the bounded next-revision
contract. Nonterminal notifications remain inside the same server request, and
the default/max wait budget is 30 minutes. Cancellation and bounded timeout
still detach only the observer. A focused regression proves a default wait
stays attached through `RUNNING` and wakes on `SUCCEEDED`; the existing revision,
cancel, and timeout tests remain green. Live receipt
`a6d1b16c-b761-4d2b-97ac-b4ef77f67cf2` confirmed the installed command returned
only at terminal state after Test Genie restarted and reattached its producer.
