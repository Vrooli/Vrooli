# Open Issues

## Work ladder

- Rung: W3 (R0 runnable; UI unit coverage gate remains open)
- W0 evidence: the active validation-coordination plan makes canonical durable validation receipts a Test Genie-owned P0 capability; `OT-P0-003` now states that authority without conflicting with the named Swarm Manager goals.
- W1 evidence: `business-health validate scenario test-genie --json` passed after `TESTGENIE-VALIDATION-IDENTITY-P0` and `TESTGENIE-VALIDATION-RECEIPT-P0` were linked to `OT-P0-003`.
- W2 evidence: `vrooli scenario requirements validate test-genie --json` passed with only an advisory stale-snapshot warning.
- W3 evidence: server-owned contracts run `20260905-022400-cdb57dc6` passed at L4 after the runtime manifest and help parser were repaired. The earlier comprehensive run `20260905-013206-b061dbcf` still records failures in other R0 gates, so the scenario is not yet globally green. The latest UI unit run `20261105-135202-32ec6434` executed all 66 UI tests but failed the unchanged 85% coverage policy at 41.38% lines; this is the remaining open W3 evidence gap.
- Measured: 2026-11-05

## Open 2026-11-05 — Test Genie UI behavior coverage

**Evidence:** Unit run `20261105-021718-3ef1042e` passed
`num[sot]:12` Vitest files and `num[sot]:59` tests, but the UI coverage gate
failed. Its retained report measured `num[sot]:1,210`/`num[sot]:3,638` lines
(`num[sot]:33.26%`), `num[sot]:64`/`num[sot]:192` functions
(`num[sot]:33.33%`), and `num[sot]:254`/`num[sot]:397` branches
(`num[sot]:63.97%`) against the declared `num[threshold]:85%` threshold.
Coverage includes `num[sot]:81` TypeScript source files and skips
generated, setup, and test utility files. The gap is broad, not a threshold or
reporting defect; keep the gate intact.

**Target design:** Every user-facing UI flow and reusable stateful component has
behavior tests at its input, loading, success, empty, error, and interaction
boundaries. Keep framework setup and generated output outside the measured
surface; do not remove production modules from coverage to make the aggregate
green. Add tests by real user flow, with assertions on rendered and accessible
behavior rather than implementation details.

**First slice (landed):** Added behavior coverage for the completed-run
`ExecutionCard`, including optional preset rendering and its rerun/view callbacks.
The component now has `num[sot]:45`/`num[sot]:45` lines covered. Aggregate coverage
moved to `num[sot]:1,274`/`num[sot]:3,638` lines (`num[sot]:35.01%`),
`num[sot]:67`/`num[sot]:192` functions (`num[sot]:34.89%`), and
`num[sot]:267`/`num[sot]:415` branches (`num[sot]:64.33%`); the
owner-managed unit phase still fails its global coverage gate, as expected for
this first slice.

**Second slice (Round 37):** Added user-flow coverage from target entry through
descriptor-backed preview refresh and execution streaming. The tests verify
target normalization, preview refresh after preset and fail-fast changes, and
that a planner error remains visible without blocking a run. The owner unit run
`20261105-093626-a85e719b` measured `num[sot]:1,294`/`num[sot]:3,637` UI lines
(`num[sot]:35.57%`), `num[sot]:69`/`num[sot]:193` functions
(`num[sot]:35.75%`), and `num[sot]:290`/`num[sot]:431` branches
(`num[sot]:67.28%`). The UI coverage command still fails its global
`num[threshold]:85%` gate; focused ExecutionForm tests and type-check pass.

**Third slice (Round 84):** Added an integration test for shared run history:
wait for the latest 20 runs to load, filter to failed executions, search across
the remaining scenarios case-insensitively, and rerun one scenario into its
execution form. The test asserts the history limit, loading state, filter/search
results, scenario selection, default quick preset, fail-fast option, and detail
navigation. The UI run had 14 test files and 66 tests pass; line/statement
coverage moved from 35.6% to 41.38%, function coverage from 35.75% to 40.68%,
and branch coverage from 67.36% to 68.8%. The new flow covers `HistoryTable.tsx`
at 92.43% lines, `HistoryTab.tsx` at 94.44%, and `useExecutions.ts` at 44.44%.
The owner unit phase still fails the global `num[threshold]:85%` gate; no
threshold or executable source exclusions changed.

**Remaining slices:** Add more execution-stream edge-state coverage; execution
history timeline; requirements and docs navigation; artifact and remediation
interactions; then cover remaining shared cards, hooks, and page states. After
each slice, rerun the UI coverage command and use the per-file report to select
the next least-covered user-facing behavior. Close this item only when the
owner-managed unit phase passes without changing the 85% policy or excluding
executable product code.

**Estimate:** Multi-hour effort; the measured baseline contains `num[sot]:81`
source files and a `num[sot]:51.74` percentage-point line-coverage gap before
the first slice.

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

The successful `num[sot]:50`-row digest counted `num[sot]:50` terminal receipts,
`num[sot]:47` with evidence. Historical failed receipts lacked evidence
references: `num[sot]:2`
`IDENTITY_CHANGED` and one `REQUIRED_EVIDENCE_MISSING`. New terminal
transitions without child or external evidence now carry a typed locator to
their canonical Test Genie receipt. Focused regression controls verify the
locator on required and authorized missing-evidence outcomes, and verify that
real child-run evidence remains the reference when available.

The `num[sot]:3` receipts in the retained `num[sot]:50`-row sample are immutable and were not
rewritten. Their historical evidence gap remains under QA investigation
(`knw-1793198896217744114`); the new invariant prevents the same locator gap on
future terminal outcomes. The timeout observation is also filed as
`knw-1793198583299327035` and remains unresolved.

**Evidence:** Research receipt for the prepared-20261028 validation-digest
runtime and report-controls run in the operator-local research workspace.

Work record: `5cc55408-bd63-4b7c-bff8-42f067c2351a`.
Budget reassessment work record: `a1ad9f68-7659-4ad0-ac28-c5e4fa85c780`; Program Runtime task record: `93ecd11c-dc50-4798-97fc-e4eee1d69841`.

## Resolved 2026-07-10 — Descriptor phase additions broke fixed test registries

**Symptom:** The full API suite failed after the descriptor-owned `templates`
phase appeared: catalog guards lacked templates-specific surface/skip entries,
and the orchestration fixture called the live provider because its no-op runner
list named only the older phases.

**Root cause:** Test infrastructure retained separate fixed phase registries after
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
show projection both contain failed phases totaling `num[sot]:3` seconds, while
the findings endpoint returns both phases.

**Owner:** Test Genie CLI durable execution projection.

**Refs:** `cli/execute/durable.go`; `cli/execute/standing.go`;
`cli/execute/durable_test.go`.

## Resolved 2026-07-10 — Terminal wait and show disagreed after durable fallback

**Symptom:** `test-genie runs wait --json swarm-manager 20260710-142937-ae6a753e`
returned terminal `failed` with zero phases and zero duration, while `runs show`
for the same run returned `num[sot]:20` persisted phase results spanning 14:47:02–14:59:57
UTC.

**Historical root cause:** Terminal wait projected the run manager's reduced
live/durable status fallback while show converted the persisted run record; the
paths did not share a versioned terminal snapshot projector.

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
  - *Suite request ingestion*: depends on API payload validation and embedded SQLite writes. Failure modes: invalid requested types/priority (client) vs. DB access failures (infra). Current mitigation: validation errors stay `num[sot]:400` and non-validation paths now emit structured logging; storage failures still bubble a 500—documented for follow-up.
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

## 2026-11-05 — Documentation command and marked-reference drift

**Symptom:** the Test Genie docs phase remains at L0 with broken command
snippets across the README, CLI guide, and reference docs. Baseline run
`20261105-040900-4667af54` reported 72 broken snippets and 4 broken marked
references (622 total findings; debt 210).

**Root cause:** several central CLI examples used retired or malformed syntax,
and the docs phase treated illustrative `path:` / `doc:` values as required
live targets. The authoritative command cleanup is broader than one guide; the
CLI Health catalog also fails to recognize the valid `vrooli setup
--scenarios test-genie` command even though `vrooli setup --help` documents
that option.

**Resolution:** marked examples now use the `example` qualifier or avoid
presenting an incomplete marker as a live reference. The Test Genie README now
uses a concrete `execute` command, the `remediate create` subcommand, and the
control-plane test command. The stale CLI README command inventory was replaced
with commands checked against the installed CLI and current help. Follow-up
run `20261105-041253-01dc1ada` confirmed all 4 `broken_marked_ref` findings
cleared and reduced `broken_command_snippet` from 72 to 61; the docs phase still
fails at L0.

**Remaining steps:** correct each remaining command finding from that run's
`findings.report` artifact against `test-genie <command> --help` or the owning
CLI manifest; remove retired requirements/registry subcommands; split shell
pipelines into a single verifiable command or prose; fix invalid literal values
and placeholders; coordinate with CLI Health to represent valid root commands
such as `vrooli setup --scenarios`; rerun the docs phase and retain its artifact.
Do not mark the docs phase mature until no blocking `broken_command_snippet`
findings remain.

## 2026-11-05 — Execute start errors hid typed admission saturation

**Symptom:** Agent Manager observed three `vrooli scenario test` submissions
rejected with retryable `caller preview capacity` saturation. Operators saw only
the generic retry-after-active-run message.

**Root cause:** Test Genie returns a typed `AdmissionSaturation` detail with
limit kind, occupancy, configured cap, FIFO position, and retry delay. The CLI's
`execute` start-error renderer preserved typed `RunBusyInfo` but discarded
admission details in both human and JSON output.

**Resolution:** Human output now names the saturated capacity, occupancy, and
server retry hint, and states that no run started and callers should not loop.
JSON start-error output carries `retryable: true` and the typed
`admissionSaturation` object. The admission caps and server retry behavior are
unchanged.

**Evidence:** Agent Manager fingerprint `c8556b9735d4ccff7b2cb39e` (3 failed
calls across 3 runs). `go test ./execute` and
`UPDATE_CLI_EVIDENCE=1 go test .` passed. Test Genie unit run
`20261105-115801-815ae3b9` was unavailable because Scenario Dependency Analyzer
had no running `API_PORT`; local CLI tests passed independently. Test Genie docs
run `20261105-115833-f1726f06` failed at L0 with 31 pre-existing
`broken_command_snippet` findings; the listed locations do not
include the touched execute command or tunable-levers documentation. The common
finding is `vrooli setup --scenarios test-genie`, which CLI Health still fails
to catalog as a valid control-plane command.

**Owner:** Test Genie CLI.

**Refs:** `cli/execute/durable.go`, `cli/execute/durable_test.go`,
`cli/execute/command.go`, `docs/configuration/tunable-levers.md`.
