# Progress — Program Runtime

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/program-runtime/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
Copied July 7 template history was removed from this scenario’s log.

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/program-runtime/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append concise milestones when work lands. Keep detailed execution receipts
with the owning plan.

## Progress Log

### 2026-09-09 — Memory-not-installed degradation contract pinned

- Corrected task checkpoint admission so a missing `vrooli-memory.finish-attempt`
  artifact is tolerated when the caller has no manifest edge, or has a
  non-`must_start` edge. Completion preserves the domain result and records
  `delivery=blocked` with `last_error=memory_not_installed`.
- A focused Go bridge regression proves the absent-edge degraded path and the
  declared `must_start` refusal path. This closes the remaining P14 matrix gap;
  the stopped-and-installed demand-start path remains covered by the live
  receipt below.

### 2026-09-09 — Learning verbs final validation

- Completed the P14 live close-out for the in-code learning verbs. The
  all-verb fixture ran twice with `status=ok`, cached fragment reuse, zero
  model calls, derived choice evidence, and delivered learning attempts.
- Device Control's seven-day comparison is reliable and reports one delivered
  `agent` cohort. BAS reports four delivered `agent` cohorts and records the
  live browser navigation, authored workflow, verified smoke flow, preference
  capture, and verified workflow replay with reuse.
- Final focused checks passed: 101 kernel tests, the 28-test Memory
  shared-program suite, the five touched Program Runtime Go package groups,
  13 BAS workflow tests, and 16 Device Control workflow tests. The setpoint
  reports zero aged blocked deliveries, advice ratio `1.0`, a `0.5079`
  fragment-cache hit rate, and zero unexercised contracts.
- Qualifications remain explicit: BAS comparison is partial because of one
  invalid and three legacy records; the advice setpoint now reports a non-null
  `1.0` ratio while preserving Memory's invalid-record reliability warning; the
  durable fragment promotion probe now resolves the persisted row and returns
  `promoted=true`; and Test Genie programs metadata is non-authoritative for the
  affected shared-workspace runs. The follow-up stopped-Memory learning run
  now proves demand-start, asynchronous startup retry, matched recall, and
  delivered learning evidence. Receipts and filed defects are in the plan
  artifact after directory and PROBLEMS.md.
- Plan Manager's plan-id continuation still resolves the active P14 execution,
  but execution-addressed `transition` and `complete` return `not_found`; the
  assessment is retained in the after-measurement README and work record.

### 2026-09-09 — Demand-start lifecycle race repaired

- Corrected the learning dependency path so Python does not reject a stopped
  demand-managed target from a stale `/reachability` projection before the Go
  governance bridge can acquire its lease and start the scenario.
- Corrected discovery classification for the control plane's successful JSON
  response with `port=0` and `no running runtime ports`, which is a stopped
  scenario rather than an invalid-port configuration.
- Added bounded post-start discovery retry for asynchronous lifecycle startup.
  Focused regressions pass in `api-core/discovery`, Program Runtime bindings,
  and the kernel. The stopped-Memory all-verb receipt records matched recall
  with three hits, derived advice, verified outcome, and delivered learning.

### 2026-09-06 — Shared-program friction repair (scoped implementation evidence)

- Corrected learning exposure semantics: runtime and Memory distinguish unassessed advice from rejection, forward explicit observations and measured effort, and block permanent capture failures without replaying domain actions.
- Added `vrooli-memory.choose-option`; Device Control and BAS consume scoped recommendations within currently allowed workflows. A deterministic repeated-task fixture reduces reads from three to one while preserving the same desired-result assertion. This is fixture evidence, not a measured operator or TV-control improvement.
- Added `program-runtime.prepare-operation` and Search Hub's concrete combined-read hint. Live automatic search for `Turn down the volume of my tv 50%` ranked `device-control.do-task` first (2,399 ms); typed library search took 4 ms. Live preparation returned the complete 10,152-character owner skill. Contract-derived routing profiles refresh after registration success as well as service failure.
- Added `test-genie.iterate`, composing the existing validation broker and history-informed quick planner. Explicit phases remain required. Admission output retains bounded receipt identity and one wait command, not the owner's potentially huge file inventory. No terminal pass is inferred from admission.
- Updated AGENTS.md, testing policy, and affected usage/debugging skills. Focused validation: 67 Python tests across shared helpers, runtime tasks, Device Control, BAS and new entry points; targeted Memory/Runtime/Search Hub Go packages and runtime capture race tests passed. Live no-action device lookup `attempt-9031b119-888d-42f9-9c30-73c6a838d750` retained unknown outcome and delivered unassessed advice on its first capture attempt.
- Scoped server-owned validation was admitted; one receipt correctly rejected concurrent changes to shared process files before execution. Remaining runs were queued behind other work at this checkpoint. This record is not a complete scenario maturity or certification claim.
- Final live owner measurement (`prog_349e779f-287e-4357-b4f1-401750ce199d`) reports one eligible attempt, five unassessed exposures, zero applied/rejected/supported/contradicted claims, and reliable untruncated evidence. Canonical debugging and work-ladder changes were verified in the generated Codex projections after Prompt Manager's owner refresh.
- Follow-up handles: Runtime receipt `6c14ec68-0749-4b79-b948-01414e6e9f4f` / run `program-runtime/20260906-183948-c2e7cdec`; Memory run `vrooli-memory/20260906-184239-af670fe4`; Search Hub run `search-hub/20260906-184345-bf62fa78`; Test Genie receipt `d56689c1-4092-43e3-9bbf-7aea58618dec`. Search Hub's 600-second wait expired with queued status, not a test verdict. No other agents' work was aborted and no test-capacity policy was bypassed. The Runtime receipt was admitted before final discovery refinements, so a terminal receipt must also be checked for source-identity validity before reuse.

### 2026-09-09 — In-code learning verbs and durable fragment reuse

- Delivered the nine in-code learning verbs (`task`, `step`, `recall`, `choose`,
  `note`, `outcome`, `infer`, `act`, and `delegate`) as a runtime-owned learning
  surface. The legacy JSON `learning_task` declaration remains a migration shim;
  new programs express learning in source code with an explicit stable key.
- Added durable fragment storage, verified-run cache selection, bounded model
  fragment preflight, cache telemetry, a promotion route, and setpoint readings.
  The checked-in all-verbs example completed twice with `fragment_source=cache`,
  zero model calls, `choice.derived=true`, and delivered learning records.
- Migrated BAS and Device Control learning callers, including explicit operation
  metadata required by the checkpoint bridge. Final focused regressions pass:
  BAS 12 tests plus 19 subtests; Device Control 16 tests; migrated Python files
  compile cleanly. Targeted Program Runtime and Memory Go/Python validation also
  passed earlier in the execution.
- Declared the learning dependency edges and updated contracts, requirements,
  skills, CLI reference, glossary, promotion ladder, and scenario architecture
  docs. Complete before/after evidence is retained in the plan artifact directory.
- Qualification: a real BAS browser E2E receipt was unavailable; Test Genie
  programs metadata reported zero observations despite a passing structure phase;
  live fragment promotion did not resolve the matching persisted fragment through
  the task bridge; and Prompt Manager refused core-skill writes. These are recorded
  in the plan after-measurement README and the problem ledger.

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-09-06 | Codex | implemented; full suite pending | Archived |
| 2026-08-06 | codex | shipped | Archived |
| 2026-08-06 | codex | shipped | Archived |
| 2026-08-06 | codex | validated | Archived |
| 2026-08-06 | codex | shipped | Archived |
| 2026-08-06 | codex | corrected | Archived |
| 2026-08-07 | codex | validated | Archived |
| 2026-08-07 | codex | corrected | Archived |
| 2026-08-07 | codex | validated-with-limit | Archived |
| 2026-08-07 | codex | Added the manifest/proto semantic binding gate and independent runtime-doctor counters. | Archived |
| 2026-08-07 | codex | validated-with-limit | Archived |
| 2026-08-07 | codex | shipped | Archived |
| 2026-08-07 | codex | validated-with-limit | Archived |
| 2026-08-07 | codex | validated-with-limit | Archived |
| 2026-08-07 | codex | corrected | Archived |
| 2026-08-07 | codex | validated-with-limit | Archived |
| 2026-08-11 | claude | corrected | Archived |
| 2026-08-11 | codex | validated | Archived |
| 2026-08-11 | codex | validated-with-limit | Archived |
| 2026-08-11 | codex | validated-with-limit | Archived |
| 2026-08-11 | codex | validated | Archived |
| 2026-08-11 | codex | validated-with-limit | Archived |
| 2026-08-11 | codex | corrected | Archived |
| 2026-08-11 | codex | validated-with-limit | Archived |
| 2026-08-12 | codex | validated | Archived |
| 2026-08-12 | codex | validated | Archived |
| 2026-08-12 | codex | validated | Archived |
| 2026-08-12 | codex | validated | Archived |
| 2026-08-12 | codex | validated-with-limit | Archived |
| 2026-08-12 | codex | validated | Archived |
| 2026-08-12 | codex | validated-with-limit | Archived |
| 2026-08-12 | codex | validated | Archived |
| 2026-08-12 | codex | validated | Archived |
| 2026-08-13 | codex | validated-with-limit | Archived |
| 2026-08-13 | codex | validated-with-limit | Archived |
| 2026-08-14 | codex | baseline-recorded | Archived |
| 2026-08-14 | codex | validated | Archived |
| 2026-08-14 | codex | validated-with-limit | Archived |
| 2026-08-14 | codex | validated | Archived |
| 2026-08-14 | codex | validated-with-limit | Archived |
| 2026-08-14 | codex | validated-with-limit | Archived |
| 2026-08-16 | codex | implemented-with-limit | Archived |
| 2026-08-17 | claude | validated | Archived |
| 2026-08-17 | claude | validated | Archived |
| 2026-08-17 | claude | validated | Archived |
| 2026-08-17 | claude | validated-with-limit | Archived |
| 2026-08-18 | codex | baseline-recorded | Archived |
| 2026-08-18 | codex | validated-with-limit | Archived |
| 2026-08-18 | codex | validated | Archived |
| 2026-08-18 | codex | validated | Archived |
| 2026-08-18 | codex | validated | Archived |
| 2026-08-18 | codex | validated | Archived |
| 2026-08-19 | codex | validated | Archived |
| 2026-08-19 | codex | validated-with-limit | Archived |
## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-09-06 — implemented; full suite pending**: Remaining diagnostic and attribution opportunities are in PROBLEMS.


- **2026-08-06 — validated**: Template Manager remains the sole skipped phase because its own setup is blocked by go.mod tidy drift.

- **2026-08-06 — corrected**: Follow-up validation closed the previously recorded inference waiver: ai-gateway now accepts native JSON Schema constraints for Ollama/OpenRouter, the live program-runtime kernel successfully completed `vrooli.ai.classify`, and the authoritative program-runtime suite passed 20/20 phases. Session spend accounting remains separately waived because its upstream Usage contracts are not yet available.

- **2026-08-07 — validated-with-limit**: The required multi-scenario baseline could not reach a comparable terminal capture: generation 19 had 5/7 ready; generation 20 failed with Test Genie `unexpected EOF` for six admitted members plus admission saturation for `vrooli-events`; after Test Genie recovered, generation 21 produced only 1/7 ready, with the other admitted runs again failing `unexpected EOF` and the remaining members blocked by an in-progress BAS run or admission saturation.

- **2026-08-07 — completed**: The independent pre-repair census was exact at 124 collisions, 78 control binds, 2 unpopulated required payloads, and 9 redundant binds; fleet repair converged the live error counts to zero, with three documented redundant-bind waivers remaining.

- **2026-08-07 — validated-with-limit**: Follow-up validation corrected explicit zero-valued semantic counters in `bindings doctor --json`, cleaned program-runtime Go module metadata, and captured live audio transcribe/transcode probes.

- **2026-08-07 — validated-with-limit**: The three remaining warnings are intentional file/JSON decoding binds with non-empty `bind_waiver` reasons. cli-health and program-runtime tests pass, live audio-tools/development-toolchain-validator validation has no binding findings, and the live runtime doctor agrees.

- **2026-08-11 — corrected**: The 19 remaining `complete` requirements were re-audited and every one resolves to a validation target that exists.

- **2026-08-11 — validated-with-limit**: Durable failure, refusal, and unresolved-binding mining surfaces are consumed by focus ranking; focused consumer tests and live mining commands pass.

- **2026-08-16 — implemented-with-limit**: Implemented the flat namespace, protected runtime names with `__vrooli__` escape hatch, Go preflight diagnostics and unresolved capture, closed failure causes, per-event timestamps/sequences, caller-selected response rows, Go-owned projection verbs, promoted-only library with `lib.list`, scenario condition rollups, authoring corpus/RPC/CLI, construction docs, routing, and device reconnect manifest repair.

- **2026-08-17 — validated**: The regex-based resolver was replaced by `kernel/host/analyze.py` (`symtable` + `ast`): it had treated every name bound by a `def`, `lambda`, comprehension, `for`, `with`, or `except` as an unresolved global and refused ordinary programs before execution, including four of eight shipped examples, and it carried a hardcoded special case for the literal string `test_geni`. The unresolved-attempt ledger now admits only capability-shaped names, with historical pollution purged at startup.

- **2026-08-18 — validated-with-limit**: Operator/test provenance is filtered from mining, capability-shaped unresolved names are purged and regression-tested, and the live ledger now reports only `test_geni`.

- **2026-08-18 — validated**: The only repeated non-surface miss is `vrooli.scenario.status`, blocked by the running root `vrooli-api` binary being older than its runtime registry schema; a lifecycle refresh was attempted but setup stopped on an unrelated sudo-owned host requirement.

- **2026-08-18 — validated**: Final post-restart Program Runtime run `20260818-221821-fb47b7c6` passed 22/22 after tightening unresolved-name filtering for `left_handle` and `right_handle`. The live unresolved ledger now contains only `test_geni`;

- **2026-08-19 — validated**: Reliability and Act supply-chain repair: projection results and aliases were corrected, imports and unresolved-name telemetry were hardened, authoring evaluation reached 9/12 twice at `authoring-brief@6` with floor 8, Prompt Manager was re-platformed to generated Connect bindings and measures, `guide` became live, nested control-plane manifest groups were restored, and the Act join reached 25 NOW / 2 IN-REACH / 1 MISSING.

### 2026-09-09 — Verified adaptive sections and portable baselines

Implemented the operator-approved learning lifecycle assessment. `learn.act` now
requires an independent verifier and revision, requests new code after a failed
candidate, and runs compatible qualified code without AI review. Every output is
still checked. Reuse requires fresh verified root attempts and distinct inputs;
test/replay evidence is isolated from live use. Generated fragments cannot mutate
verifier inputs or tool transport/audit objects, and a potentially committed write
with a lost response stops adaptation.

Later feedback survives outages in the finish outbox. Memory recall preserves
contradictions and origin context; choose excludes avoided/contradicted defaults
and can return no eligible option. BAS handles that result, and Device Control
accepts the earlier recommendation's attempt ID for evidence-backed feedback.
Reviewed baselines stay in Git alongside programs; publication preserves source,
replays curated fixtures, and checks review and source/declaration freshness.
Plugin composition transports these ordinary assets without exporting Memory.

The canonical construction guide and registry skill describe the new behavior.
Focused regressions and public CLI generation/baseline smokes passed. The problem
ledger records exact receipts and failed broader gates; no whole-scenario or
arbitrary self-healing reliability claim is made.
