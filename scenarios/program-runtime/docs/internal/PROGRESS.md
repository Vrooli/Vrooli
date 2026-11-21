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

### 2026-10-13 — Maturation plan authored; reachability misclassification fixed (WS-B #1)

- Authored `docs/internal/MATURATION_PLAN.md` (6 workstreams A–F; DoD tied to
  `portfolio-audit` score 0.69→0.85; executed via the control agent's own tooling, not Plan
  Manager / Swarm / finite-delivery). Live re-measure: score 0.69, `exercised` 38%,
  `typed_output` 1.7%, `binding_resilience` 26%; 19/118 programs ever agent-run; the dominant
  consumer `agent-manager.supervision-evaluate` broke 2026-09-07 (contract/digest drift), not
  09-15.
- WS-B #1: discovery `invalid_port` / `vrooli_not_found` kinds and the vrooli-target
  `invalid VROOLI_API_PORT` message now classify as `scenario_unreachable` / `unavailable`
  instead of falling through to `binding_error` (`api/internal/bindings/failure.go:42`,
  mirrored in `kernel/host/program_helper.py:155`). Tests added in `failure_test.go` and
  `kernel/tests/test_program_helper.py`; both green. Not live until redeploy.
- WS-D (kernel helpers + de-dup): added `program.select_field`, `program.coerce_int`,
  `program.coerce_float` to the bound helper (`kernel/host/program_helper.py`), synced to
  `api/internal/programs/surface.go` (drift-guard `TestProgramHelperMembersMatchKernel` green),
  with unit tests. Migrated `usage-triage.py` as the exemplar (local `value`/`as_int`/`as_float`
  → `program.*`; `transport_failure` → `program.classify`, a superset); smoke-run SUCCEEDED and
  degrades to `partial` on a 403 read exactly as designed. The 3 remaining copies
  (`mutation-pilot`, `attention-triage`, `classify-batch`) have DIVERGENT semantics (dict.get
  fallback keeps None/empty; `int(v)` not `int(v or 0)`; return-not-reraise on NameError) and are
  intentionally left — forcing convergence would be a silent behavior change for a 1-copy gain.
  New programs use `program.*` by example + scaffold template.
- WS-A1 (consumer reconciliation): confirmed the current-source `programInput()`
  (`agent-manager/api/internal/supervision/program_evaluator.go:281`) emits exactly the contract's
  declared input keys and correctly omits the optional `watch_id` the OLD running binary was
  sending with a bad type (the cause of the 2026-09-07 break). Redeployed agent-manager from
  source; the request is now contract-valid. The 4 probe watches re-evaluate at their next
  scheduled wake (~18:29) and should resolve/self-close then — verification pending that wake.
- WS-C `scaffold` DONE: `program-runtime programs scaffold --name <s>.<p> --purpose "..."
  [--binding <s>/<g>/<c>] [--out-dir <dir>] [--force]` generates a schema-valid contract + a
  `program.run` state-machine skeleton that uses the kernel helpers. Implemented as a local
  `cliapp.Action` (no proto regen needed — the framework's `Action[Resp any]` renders non-proto
  results), `cli/domains/programs/scaffold.go`, registered in `register.go`, manifest entry added.
  Verified: builds, both binding/no-binding variants validate against
  `schemas/program-contract.schema.json` and the generated `.py` compiles; the `$schema` path is
  owner-aware (own-scenario vs cross-scenario). Unit tests in `scaffold_test.go` (3) + full
  programs-domain suite green. NOTE: live in a freshly built CLI; the installed
  `~/.vrooli/bin/program-runtime` needs a CLI rebuild/reinstall to expose the command. During
  authoring the schema check caught a real generator bug (empty `errors.classes`, schema requires
  minItems:1) — exactly the class of error the planned `validate` command will catch for authors.
- WS-C `validate` DONE: `program-runtime programs validate <contract.json> [program.py]` runs the
  offline authoring checks authors otherwise hit only at submit: authoritative JSON-schema
  validation (schema resolved drift-free from the contract's own `$schema`, compiled with the
  already-present `santhosh-tekuri/jsonschema/v5`, violations reported with JSON-pointer locations),
  name↔file-stem match, and a `python3 -m py_compile` syntax check (skipped-with-warning if no
  interpreter). Exits non-zero on any problem (CI-usable) via `cliapp.ActionWithExit`.
  `cli/domains/programs/validate.go` + `validate_test.go`, manifest entry, registered. Verified
  against a real corpus contract (all 4 checks pass, exit 0) and a deliberately-broken in-repo
  contract (caught missing `version`, empty `errors.classes` with location, name/stem mismatch,
  exit 1). Honest boundary: deep binding-call resolution stays owned by the server preflight
  (`analyze.py`) at submit; validate does not reimplement it. Needs CLI reinstall to be live in
  `~/.vrooli/bin`.
- WS-C `scaffold`/`validate` made live: the installed `~/.vrooli/bin/program-runtime` auto-rebuilds
  from current source on a stale invocation, so both commands are now usable. Dogfooded: `validate`
  passes all 22 of program-runtime's own contracts (0 false positives), including the migrated
  `usage-triage`.
- WS-F liveness canary DONE (in agent-manager): a new Optional `/health` check
  `supervision_liveness` (`scenarios/agent-manager/api/internal/wiring/supervision_health.go` +
  `_test.go`, wired at `routes.go:127`) reads the existing `cohort_watch_evaluation_failures` ⋈
  `cohort_watches` and degrades the scenario (never fails it) when any ACTIVE watch exceeds
  `supervisionStuckFailureThreshold` (24 ≈ a day of hourly retries), naming the watch + the
  `watch get` inspect command. Unit tests (5 cases) green; agent-manager rebuilt. LIVE PROOF: with
  startup-functional cleared and readiness=true, agent-manager's `/health` is **degraded** solely
  from this Optional check — i.e. the canary immediately caught the 4 real stuck zombie watches that
  had been silent since 2026-09-09. This is the signal that would have caught the 2026-09-07 break
  within a day instead of a month.
- A1 HONEST UPDATE: the zombie self-close did NOT happen. Past their 18:29 wake and after both
  redeploys, the 4 probe watches remain ACTIVE at their old revisions (~5145) with lastDecision
  still 2026-09-09 — the scheduler is not re-evaluating them (separate issue: likely the
  `due_family_rank` CTE / next_wake handling). So the clean contract (confirmed) did not by itself
  revive them. Resolution now: the canary makes them VISIBLE (agent-manager degraded), and clearing
  them still needs the operator `watch cancel` (human-gated) OR a follow-up scheduler investigation.
  Once cleared, the canary returns agent-manager to healthy.
- A1 ROOT CAUSE (corrected 2026-10-14) + FIX: the scheduler IS re-evaluating the 4 zombies — hourly,
  failures=842 over 35 days, last_failed_at today. The real defect: a contract rejection is
  classified PERMANENT (`isContractRejection`, "none of them clears by retrying") yet
  `parkedDecision` reschedules it hourly FOREVER — no dead-letter — so a misconfigured watch becomes
  an eternal zombie no scheduler pass can resolve. (The live reason is
  `program_contract_rejected:...watch_id must have type string`; `recordEvaluationFailure` rewrites
  the reason every failure, so it's current, not stale — current-source `programInput` never emits
  `watch_id`, so the rejection comes from the probe watches themselves, which are simply permanently
  misconfigured.) FIX: `program_evaluator.go` now dead-letters to TERMINAL after
  `contractRejectionDeadLetterThreshold` (24 ≈ a day, ruling out a deploy-window blip) consecutive
  contract rejections, with classification `program_contract_rejected_dead_letter`. Unit test +
  full supervision suite green; agent-manager redeployed. EXPECTED: the 4 zombies (streak 842 ≫ 24)
  self-terminate to TERMINAL on their next scheduled wake (~05:29 UTC 2026-10-14) with NO operator
  cancel, and the WS-F canary then returns agent-manager to healthy. (Can't force the wake — it's
  management-gated; verify after 05:29 UTC.)
- Still staged: WS-C `dry-run`, WS-E output_schema, WS-B#2 `--explain` binding-arg validation.
- Operator-gated fallback (likely unnecessary after A1): `watch cancel` on the 4 zombies requires
  a verified operator (`agent-manager:write`); reused the existing watches as the A1 verification
  vehicle instead of force-cancelling, so no human op is needed unless they survive a clean wake.

- Scenarios redeployed this pass: program-runtime (x3), agent-manager (x1). Score steady 0.696
  (no regression); `go build` of touched packages clean.

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
