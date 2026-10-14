# Program Runtime — Maturation & Cleanup Plan

Authored 2026-10-13. Owner: operator-directed, executed without Plan Manager and without
the finite-delivery / Swarm Manager infrastructure (that infra is mid-rework). This plan is
driven by the control agent's own tooling: scoped sub-agents, a `/goal` loop, focused
`test-genie` phases, and the scenario's own self-measurement (`portfolio-audit`,
`authoring eval`, `discovery eval`).

This is a durable strategy document. It extends the existing debt ledgers
(`PROBLEMS.md`, `SEAMS.md`, `DECISIONS.md`, `ERROR-HANDLING.md`); it does not replace them.
Dated window snapshots belong under `docs/internal/delivery-operations/<date>/`.

---

## 1. Why this plan exists (measured state, 2026-10-13)

The scenario is healthy as infrastructure and is one of the most self-aware in the fleet.
Two September adoption blockers are now **fixed**: discovery works
(`search-hub query "...tv volume..." --type library` → 8 programs in 47ms, was circuit-open
at 4s) and provenance attribution is populated (agent/operator/test split is real).

The problem is **not supply**. It is (a) a collapsed autonomous consumer, (b) a few
reliability sharp edges, (c) thin authoring tooling, and (d) corpus debt (copied helpers,
near-absent typed output). Live numbers:

| Signal | Value | Source |
|---|---|---|
| `portfolio-audit` score | **0.69**, 65/117 below band | live run |
| `exercised` | **38%** | the consumption gap, quantified |
| `typed_output` | **1.7%** (2/117 have `output_schema`) | `portfolio-audit` |
| `binding_resilience` | 26% (multi-binding contracts only) | `portfolio-audit` |
| source/learning/live_evidence/capability_tag/budget_realism | 96–100% | `portfolio-audit` |
| Programs on disk | 131 (118 with run history) | `find` + `programs portfolio` |
| Lifetime runs | 4,422 (agent 1,377 / operator 2,271 / test 774) | `programs portfolio` |
| Programs ever run by an agent | **19 of 118** | `programs portfolio` |
| Top failure shape | `kernel_runtime` = **186, last seen today** | `programs mine` |

**The consumption story, corrected.** `agent-manager.supervision-evaluate` is 1,075 of the
1,377 agent runs. Its last *successful* run was **2026-09-07T18:22** — the ~20k decisions on
09-06/07 are essentially the entire agent-run population. It broke at the program-runtime
redeploy that day (commit `63fae23f5b6` rewrote
`api/internal/contracts/index.go`: new `ContentDigest`, contract schema added to
`schemaPaths`); the runtime then began rejecting the evaluator's request at the contract gate
(`invalid_argument: input "watch_id" must have type string`). Since then: 4 zombie
`validation-coordination-*` probe watches fail hourly forever (failure counter 830 at
2026-10-13T16:29; scheduler is alive), and the only source of *real* cohort watches —
plan-family workflows (`plan_family_workflow.go:328` `familySupervision.Create`) — has been
idle since 09-11. Net: the layer has had ~zero healthy autonomous consumption for a month,
and `exercised=38%` is measured almost entirely from a historical burst.

**Correction to an earlier claim:** the "target 5xx misclassified as `kernel_runtime`" bug is
**already fixed** in current code (the `remoteStatusPattern` path in
`api/internal/bindings/failure.go:85-92` classifies 502/503/504 as `unavailable`, with unit
coverage in `failure_test.go`). So the live `kernel_runtime=186` is mostly **real uncaught
exceptions in programs**, not mislabeled infra errors. That shifts weight toward program-quality
hardening (WS-D) over kernel re-classification (WS-B is correspondingly smaller).

---

## 2. Guiding principles

1. **Cleanup and subtraction beat addition.** Delete copied boilerplate, retire zombies, and
   remove canon drift before authoring anything new. Every new program must justify itself
   against reuse.
2. **Every workstream has a measurable Definition of Done tied to a command**, not a vibe —
   a `portfolio-audit` dimension moving, a `test-genie` phase passing, a `go test` green, a
   failure shape dropping in `programs mine`.
3. **No dependency on the reworked infra.** Nothing here requires Plan Manager, Swarm Manager,
   or finite-delivery teams. Where the ideal fix is coupled to that infra (WS-A), we take the
   decoupled path and surface the coupling as an explicit decision.
4. **Work around non-ideal state as a first-class requirement.** Programs and the kernel must
   degrade honestly when a dependency is down (correct status, no false `kernel_runtime`), and
   there must be a canary that catches silent consumer death within a day, not a month.
5. **Respect operator hard rules:** no git usage by the agent; deletions verifiable at a
   glance; no commit/GCT gating; ship decisions with switch costs.

---

## 3. Workstreams

Six workstreams (A–F). Each lists the problem, exact edit sites, concrete tasks, a measurable
DoD, rough effort, and risks. Dependency order and parallelism are in §4.

### WS-A — Restore healthy autonomous consumption (the whole game)

**Problem.** The dominant consumer is dead by infrastructure drift, and its real supply is
coupled to plan-family/Swarm — the infra under rework. `exercised=38%` and
"19/118 programs ever agent-run" are the direct measures.

**Decision (resolved 2026-10-13 with operator).** The agent that was reworking the
plan-family / Swarm area has been **stopped** — there is no active owner of that state. So both
paths are in scope and A1 is a **first-class deliverable, not a gated option**; the "retire the
watch feature" fallback is **off the table** — we fix it properly. Do A2 first for immediate
liveness, then the full A1 repair.

- **A1 — Repair the existing supervision consumer (full, in scope).** Reconcile the request
  contract against the current runtime, rebuild/redeploy agent-manager from source (the
  `watch_id` the runtime rejects is produced by *no* current source version per
  `git log -S '"watch_id"'` → stale-binary hazard), clear the 4 zombie watches, and add the
  canary (WS-F).
  *Honest residual (not a blocker):* real cohort supply still depends on plan-family workflows
  running again, which is the paused infra. We therefore **verify A1 end-to-end with a synthetic
  test watch** (exercise scheduler → evaluator → program contract directly), not by spinning up
  the Swarm pipeline. Outcome: a repaired consumer that is **healthy-and-ready and no longer
  red/zombie**, exercised on demand by the synthetic watch and by A2's heartbeat, and ready to
  pick up real cohorts whenever plan families resume. No feature retirement.

- **A2 — Add an infra-independent heartbeat consumer.** Stand up a scheduled agent-attributed
  loop that runs the read-only supervision suite that already exists
  (`program-runtime.supervision-sweep`, `portfolio-audit`, `validation-sweep`,
  `failure-triage`, `scenario-triage`) on a cadence, tagged `PROVENANCE_AGENT`. This exercises
  the layer and raises `exercised` **without** plan families. agent-manager already owns a
  scheduler (`api/internal/supervision/scheduler.go`) and already runs daily heartbeats
  (meta-optimization skill-optimizer/run-introspector/friction-curator), so the cheapest form
  is a new heartbeat entry there, or a host cron invoking `program-runtime library run …`.
  *Switch cost:* a heartbeat that only *reads* doesn't close the self-improvement loop by
  itself; it proves liveness and lights up `exercised`/typed output, and its envelopes feed
  WS-E and `usage-triage`.

**Edit sites / tasks.**
- A1: `scenarios/agent-manager/api/internal/supervision/program_evaluator.go:138,281-293`
  (request contract), `scenarios/agent-manager/.vrooli/program-runtime/supervision-evaluate.json`
  (digest reconcile), `scheduler.go` + `api/main.go:478` (scheduler is live — verify),
  zombie watches `306b3ae2 / 952c4c06 / d002daa8 / 341a8167` (cancel). Rebuild via
  `make stop agent-manager && make start agent-manager`.
- A2: new heartbeat registration in agent-manager's scheduler/daemon (model on the existing
  meta-optimization heartbeats), or a host cron. Each tick runs 3–5 read-only programs and
  records provenance `agent`.

**DoD.** `programs portfolio` shows a program with `agent_runs` increasing **daily** and a
`last_seen` within 24h; `exercised` ≥ 60% within two weeks of the heartbeat running;
zero ACTIVE zombie watches (`supervision` repo shows no permanently-parked probe watches).

**Effort.** A2: 1–2 days. A1: 2–4 days (contract/binary reconciliation is the unknown).

**Risk.** A1's stale-binary reconciliation may expose the "supervisor runs stale control-plane
code" hazard broadly; keep A1 behind A2 so liveness doesn't block on it.

---

### WS-B — Kernel/binding reliability & honest degradation

**Problem.** A few real misclassifications make healthy programs look broken or hide author
mistakes; and nothing warns when a consumer silently dies.

**Edit sites / tasks (verified present in current code).**
1. **`invalid_port` not classified as unreachable.** A stopped target surfaces as
   `invalid VROOLI_API_PORT` (`api/internal/bindings/registry.go:381-398`) and falls through
   to the `binding_error` catch-all. Add the `invalid_port` / "cannot resolve API port" needle
   to `failureRules[0]` in `api/internal/bindings/failure.go:42-52` (and mirror in
   `kernel/host/program_helper.py:155-168`) so it becomes `scenario_unreachable` / `unavailable`.
   Add cases to `failure_test.go` and `kernel/tests/test_program_helper.py`. **LOW.**
2. **`--explain` skips binding-argument validation.** List-kwarg mistakes only surface at
   runtime as `proto: syntax error` (`registry.go:669`). The preflight
   (`api/internal/programs/preflight.go:75-127`, gated in
   `api/internal/programs/service.go:214`) resolves free names but never checks binding arg
   shapes. Extend preflight to validate declared binding args against the proto schema, or at
   minimum emit a targeted hint for the list-kwarg pattern. **MEDIUM.**
3. **`vrooli/scenario/logs` 500 on scenarios with no lifecycle log** (seen in field use by
   `scenario-triage`; the kernel audit could not fully confirm it as a governed binding — so
   **first verify**: `program-runtime library run program-runtime.scenario-triage` against a
   registry-runtime scenario and watch the classification). If the 500 is real, make the owning
   handler return empty/NotFound not 500 (`internal/cli/scenariohandlers/logs_runtime.go:153-181`
   / the controlplane GetScenarioLogs handler). **LOW–MEDIUM, verify first.**
4. **Document the failure table as the contract.** The closed set is 11 causes with ordered
   substring rules (`failure.go:24-52`, mirrored in `program_helper.py`). Promote this into
   `docs/internal/ERROR-HANDLING.md` (currently a 2-line stub) so authors and future kernel
   edits treat it as a spec, and add the ordering-hazard note (unreachable first, deadline last).

**DoD.** New `failure_test.go` / python cases green; a stopped-target run returns
`unavailable`/`scenario_unreachable` (not `binding_error`); `--explain` on a list-kwarg program
reports the error pre-execution (or a clear hint); `ERROR-HANDLING.md` documents all 11 causes
and the ordering invariant.

**Effort.** 2–4 days total.

---

### WS-C — Authoring tooling: `scaffold`, `validate`, `dry-run`

**Problem.** Authoring is two hand-written files with ~25 lines of boilerplate, no generator,
no offline check; `--explain` only resolves names. This is the "easier to create" ask.

**Architecture (confirmed).** Commands are declared in `cli/manifest.json` and handled in
`cli/domains/<group>/register.go` via `cliapp.LoadFromManifestPrimitives(...)`; local-only
commands use `"binding": {"kind":"local"}` (pattern: `library search`). Add all three to the
existing `programs` domain — no new domain module.

**Tasks.**
- **`programs scaffold --name <s>.<p> --purpose "…" [--binding id]`** — emit a minimal valid
  contract + `.py` state-machine skeleton (required fields enumerated in
  `schemas/program-contract.schema.json:7-18`; model `.py` on `concurrent-fanout.py`). Local
  command, template only. **LOW (~1–2 days).**
- **`programs validate <contract.json> <program.py>`** — offline: (a) schema-validate the
  contract reusing the already-vendored `github.com/santhosh-tekuri/jsonschema/v5`
  (`api/internal/contracts/index.go:20`); (b) check declared bindings exist in the registry
  (`api/internal/bindings/registry.go`); (c) cross-check that `.py` `lib.<s>.<name>` /
  `<scenario>_<group>.<cmd>()` calls match declared `bindings`; (d) reuse `ResolveSource()` /
  `analyze.py` for free-name resolution; (e) reuse `Contract.ResolveInputs()`
  (`contracts/index.go:122-150`) for fixture input validation. **MEDIUM (~3–5 days).**
- **`programs dry-run --source-file <py> [--contract <json>]`** — submit to a scratch session
  with synthetic inputs from contract defaults and a tight budget (`wall_ms≈5000`,
  `inference_calls=0`), report terminal status + phase + error class, reclaim the session.
  Reuses the submit path. **MEDIUM–HIGH (~5–7 days).**

**DoD.** `authoring eval` first-attempt score improves against the versioned corpus; a new
author can go scaffold → validate → dry-run → `library run` without reading the schema by
hand; `validate` catches a deliberately broken contract (bad enum, missing binding,
contract/source mismatch) offline.

**Effort.** ~9–14 days across the three; scaffold+validate first (highest leverage per day).

---

### WS-D — Corpus de-dup, helper injection, and contract honesty

**Problem.** Copied helpers and misleading contracts. 33 programs hand-roll a local `fail()`
that just reimplements `program.fail()`; `transport_failure`/`parse_label`/`as_int`/`as_float`/
`value` are copied across programs; 4 deterministic read programs omit `verbs:["gather"]`.

**Mechanism (confirmed feasible).** De-dup via **kernel-injected `program.*` methods** — the
public surface is drift-guarded by a Python↔Go test
(`kernel/host/program_helper.py:38-57` `PUBLIC_MEMBERS` ↔
`api/internal/programs/surface.go:14-19`, validated by `surface_test.go:38-43`). Shared-module
import is *not* viable (`lib` is for whole promoted programs; each call gets a fresh helper —
`engine.py:969-1059`).

**Tasks.**
1. Add 5 helpers to `ProgramHelper` + `PUBLIC_MEMBERS` + `surface.go`, run
   `go test ./api/internal/programs -run TestProgramHelperMembersMatchKernel`:
   `program.coerce_int`, `program.coerce_float`, `program.select_field`,
   `program.parse_json_field`, `program.classify_transport`. **~1–2 days.**
2. Migrate the duplicated copies to the new methods and **delete** the local defs (deletions
   verifiable at a glance — one helper per diff hunk). Audit the 33 local `fail()` wrappers;
   most collapse to `program.fail()` directly.
3. Add missing `verbs:["gather"]` to the deterministic read programs (supervision-sweep,
   validation-sweep, scenario-triage, efficiency-report) so contracts stop understating them.
4. Make `binding_resilience` apply to single-binding contracts too (it only scores
   multi-binding today), and harden the worst offenders to per-binding try/except → degrade to
   `unavailable` rather than abort. Worst programs to fix first:
   `performance-health.packaged-experiment` (0.375), `program-runtime.fragment-promote` (0.5),
   the `agent-manager.*` cluster (0.625).

**DoD.** Zero local re-definitions of the 5 helpers remain (`grep` clean); `fail()` local count
drops materially; `binding_resilience` ≥ 60%; the three worst programs ≥ 0.75 in
`portfolio-audit`.

**Effort.** ~3–5 days.

---

### WS-E — Typed output adoption

**Problem.** `typed_output=1.7%` — only `portfolio-audit` and `supervision-evaluate` declare an
`output_schema`. Callers must infer signal shape; no runtime validation.

**Tasks.** Add `output_schema` to the high-traffic programs first (the 19 with agent runs, then
the operator-heavy ones), generating schemas from their actual envelopes. Verify the dimension
isn't a measurement artifact: it currently "only penalizes >8-signal programs" — confirm the
`portfolio-audit` scoring (`.vrooli/program-runtime/portfolio-audit.py`) and, if the threshold
hides real gaps, fix the measure so the number means what it says (cross-ref memory:
"validation shapes that lie green").

**DoD.** `typed_output` ≥ 50% among the agent-run programs; the dimension's definition
documented and honest.

**Effort.** ~2–4 days (mechanical once the first few establish the pattern).

---

### WS-F — Docs, debt consolidation, and the liveness canary

**Problem.** Canon drift and a stub `ERROR-HANDLING.md`; and the reason a dead consumer went
unnoticed for a month is that **nothing watches for it**.

**Tasks.**
1. **Liveness canary (do this early — it protects every other workstream).** A health signal on
   "days since last successful supervision decision" / "consecutive contract rejections > N"
   (`program_evaluator.go:378` currently warns-once then drops to debug). Surface it where the
   operator already looks. This is what would have caught the 09-08 break on 09-08.
2. Fill `ERROR-HANDLING.md` with the WS-B failure table (shared contract + scenario specifics).
3. Reconcile canon drift noted in prior audits: the three-speed ladder paragraph, the
   Skill/Program glossary entries, and the README boilerplate; confirm `scenario-work-ladder`
   references programs. (Scope-limit to program-runtime's own docs; cross-scenario doc fixes are
   a separate effort.)
4. Append a milestone to `PROGRESS.md` per workstream as it lands; log new/updated debt in
   `PROBLEMS.md`.

**DoD.** Canary emits a signal when the heartbeat (WS-A) is stopped for >1 day (test by
stopping it); `ERROR-HANDLING.md` is no longer a stub.

**Effort.** ~2–3 days.

---

## 4. Sequencing & parallelism

```
Phase 0 (serial, days 1–3):   WS-F canary  +  WS-A2 heartbeat   ← prove liveness first, so every
                                                                   later change is measured live
Phase 1 (parallel, days 3–10): WS-B (kernel) │ WS-C scaffold+validate │ WS-D helpers+migrate
Phase 2 (parallel, days 10–16):WS-C dry-run  │ WS-E typed output      │ WS-D resilience hardening
Phase 3 (serial, days 16–20):  WS-A1 full consumer repair (contract + binary + synthetic-watch
                               verify) + WS-F docs/canon consolidation + final re-measure
```

Rationale: Phase 0 makes `portfolio-audit` move for real so progress is observable. WS-B/C/D
touch mostly disjoint trees (Go bindings/CLI vs kernel helpers vs program `.py` files) — the one
overlap is `kernel/host/program_helper.py` (WS-B mirrors the failure table there, WS-D adds
helpers there), so those two edits are serialized or done by the same agent. WS-A1 is last
because its outcome (repair vs retire) depends on whether plan-family supply returns — a
decision, not a mechanical task.

---

## 5. Overall Definition of Done

- `portfolio-audit` score **≥ 0.85** (from 0.69), with `exercised ≥ 60%`,
  `binding_resilience ≥ 60%`, `typed_output ≥ 50%` among agent-run programs.
- A program shows `agent_runs` incrementing **daily** with `last_seen` < 24h (liveness restored).
- `programs mine` top `kernel_runtime` count trends **down** after WS-D (real exceptions fixed).
- A new author completes scaffold → validate → dry-run → `library run` for a toy program with no
  schema reading and no runtime surprises.
- Zero ACTIVE zombie watches; a canary fires within a day if the heartbeat stops.
- `go test ./...` green in program-runtime and agent-manager; relevant `test-genie` phases
  (`programs`, `bindings`, `authoring`, `discovery`) pass.
- Deletions (copied helpers) verifiable at a glance in the diff.

---

## 6. Execution strategy — using the control agent's own tooling

No Plan Manager, no Swarm, no finite-delivery team. Driven by the control agent directly.

**Driver.** A single `/goal` loop owns the plan and advances it phase by phase, re-reading this
document and the live `portfolio-audit` each iteration. The goal's `until` is the §5 DoD
(score ≥ 0.85 + liveness + green tests). The loop:
1. Pick the next unblocked task per §4.
2. Make the change (or dispatch a scoped sub-agent for a self-contained one).
3. Verify: `go test` on the touched package, the relevant `test-genie` phase, and a
   `portfolio-audit` / `authoring eval` re-measure.
4. Append a `PROGRESS.md` milestone; log residual debt in `PROBLEMS.md`.

**Sub-agents.** Use scoped sub-agents for the parallelizable Phase-1/2 workstreams, partitioned
by file ownership to avoid the shared-tree hazard (memory: multi-agent shared-tree hazards):
one agent owns Go bindings/CLI (WS-B + WS-C), one owns kernel + program `.py` files
(WS-D + WS-E), one owns docs/canary (WS-F). They edit **disjoint** paths.
*Decision — isolation (switch costs):*
- **Default: sequential same-tree edits by the driver**, dispatching sub-agents only for
  read/analysis and for self-contained single-file edits the driver reviews. Safest under the
  "never use git" rule and the shared-tree hazard; slower.
- **Option: `isolation:"worktree"` sub-agents** for true parallel code edits. Faster, but each
  spins a git worktree — surface to the operator before using, given the no-git rule. Recommend
  only if the operator OKs worktrees for isolation.

**Verification without the test infra.** Everything is checkable with scenario-native commands:
`program-runtime programs portfolio/mine/governance-share`, `authoring eval`, `discovery eval`,
`go test ./...`, and focused `vrooli scenario test program-runtime --phases programs,bindings,
authoring,discovery` (server-owned run; block once with `test-genie runs wait`, never poll).

**What this plan deliberately does NOT do.** It does not author a pile of new programs
(supply is done); it does not depend on plan-family/Swarm supply (WS-A2 decouples liveness);
it does not hand-edit generated artifacts or add compat shims for the contract/digest drift
(WS-A1 reconciles through the governed path).

---

## 7. Decisions ledger (for operator)

| # | Decision | Recommendation | Switch cost if reversed |
|---|---|---|---|
| D1 | Revive existing consumer (A1) vs. new heartbeat (A2) vs. both | **Both — resolved 2026-10-13.** A2 now for liveness; full A1 repair in scope (no active owner of the Swarm rework to conflict with) | A2-only leaves the rich decision kernel broken + zombies red; A1-only doesn't exercise the layer day-to-day |
| D2 | If plan-family supply stays dormant, repair A1 anyway or retire/re-scope the watch feature | **Repair properly — resolved 2026-10-13.** Retirement off the table. Verify A1 with a synthetic watch; leave it healthy-and-ready for when plan families resume | If supply never resumes, A1 stays idle (but healthy, not zombie); cost is maintaining an unexercised consumer vs. the A2 heartbeat that does exercise the layer |
| D3 | Parallel execution via worktrees vs. sequential same-tree | **Sequential default; worktrees only with operator OK** (no-git rule) | Worktrees are faster but touch git; sequential is slower but safe |
| D4 | De-dup via kernel `program.*` methods vs. leave copies | **Kernel methods** (drift-guarded surface) | Leaving copies keeps drift risk; kernel methods add 5 to the public surface forever |
| D5 | Fix `typed_output` measure vs. just add schemas | **Both — add schemas and make the measure honest** | Adding schemas without fixing the threshold leaves the number misleading |

---

## 8. Cross-references

- Live snapshot & traps: memory `program-runtime-supervision-investment-2026-10-12`,
  `program-layer-is-built-and-unused`, `program-runtime-kernel-facts-for-programs`.
- Debt ledgers: `PROBLEMS.md`, `SEAMS.md`, `DECISIONS.md`.
- Supervision loops guide: `docs/guides/supervision-loops.md`.
- Authoring: `docs/guides/program-construction.md`, `docs/construction/*`.
