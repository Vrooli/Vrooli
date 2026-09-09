# Problems — Program Runtime

## Morning walk interruption and invocation friction — 2026-09-06

Related prior art: the 2026-08-17 EOF incident below was caused by the server
write timeout. Scenario-local fix search for EOF returned no fix rows. This
incident differs: program `prog_3f203ec7-56f0-4f56-a908-b165fc6176a7` was accepted
at 05:20:15Z and lifecycle restart began at 05:20:31Z. Its durable record remained
RUNNING across later restarts. A transport cutoff from process restart explains
this failure; a program Python error would have produced a terminal program
record, and the old 30-second timeout does not explain the 16-second restart.

Repairs:

- `RunDeclaredProgramRequest.async` returns the durable id before the CLI waits.
  The CLI prints exact get/wait recovery commands, distinguishes admission
  rejection from unknown acceptance, and never automatically resubmits.
- Session cleanup follows terminal execution, not the observing request. Canceling
  or losing the wait no longer reclaims a running kernel.
- Startup reconciles all accepted/running records through the repository before
  admitting new work. Records become FAILED with typed RUNTIME_INTERRUPTED and
  retained output. The detail states that downstream effects may have happened.
- Repeated CLI input flags are merged, with duplicate keys rejected. Previously
  channel=test disappeared when another --input followed it.
- Final CLI wait results omit source, avoiding an 18,545-byte source echo in the
  observed walk result. `programs get` remains the deliberate source inspection.

Evidence: SQLite/memory restart regressions, async observer-cancellation test,
interrupted-wait recovery test, real-parser repeated-input test, and focused race
checks pass. Lifecycle restart reconciled the original record at 05:36:02Z;
`programs get` now returns `FAILURE_CAUSE_RUNTIME_INTERRUPTED`. Live repaired prep
`prog_409ec470-44ee-4794-9772-1b9a5c780545` completed; follow-up
`prog_29b14acc-1a5c-464d-8f0d-29ab3a42265f` returned zero source bytes.

Remaining opportunities, in priority order:

1. A disconnect during acceptance can still leave the caller unsure whether the
   submission exists. Add a durable caller request key with replay-safe admission;
   do not solve this with retries or time-window heuristics.
2. Emit bounded per-binding progress and direct causal links from program to
   invocation, owner result, and lifecycle interruption. The current safe id is
   useful, but an agent still assembles the explanation across surfaces.
3. Add program-name/current-artifact filters and newest-first bounded listing.
   The default ascending 200-row corpus obscures the recent incident.
4. Audit provenance: library CLI currently supplies operator even from scheduled
   callers. Preserve actor provenance independently of the domain's test channel.
5. The friction digest found zero program-runtime episodes within a capped 40-run
   sample (850 episodes overall); this cannot establish absence of friction.

Work ladder: W0 is consistent with operator-authorized repair, the active
program-runtime-improve goal, OT-P0-006 telemetry and OT-P0-010 durable async work.
Business and requirement gates pass. These are W3 repairs of existing obligations.
No host remediation, execution replay, or claim of transparent restart resumption
was added. Single-process execution ownership remains an explicit assumption.

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

### 2026-08-07 — Clean doctor counts disappeared from JSON

Proto3 omits scalar zero values by default. That made a clean
`program-runtime bindings doctor --json` response look incomplete even though
the human report showed zero semantic findings. The bindings doctor now uses
cli-core's `ProtoListEmitUnpopulatedJSON` renderer variant, preserving the
renderer-separated operation while emitting all four semantic counters.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-07 — CLI renderer received the wrong protobuf type

**Symptom:** Migrated `ai-gateway` commands failed in human mode because a
renderer expecting a generated response received a dynamic protobuf message.

**Root cause:** The generic dispatcher decoded the wire response dynamically
but did not convert it to the registered concrete generated type before the
renderer override.

**Workaround:** None required after the dispatcher conversion; the real
dispatcher path is covered by the cli-core renderer test and live captures.

**Real fix:** `ProtoBindings` now converts through the global protobuf registry
before invoking a renderer.

**Owner:** cli-core.

**Refs:** `packages/cli-core/cliapp/protobindings.go`,
`packages/cli-core/cliapp/protobindings_render_test.go`.

### 2026-08-07 — Manifest binding resolution was semantically gameable

**Symptom:** A binding could resolve to a proto field while multiple arguments
overwrote one another, control flags were sent as payload, or required audio
payloads remained empty.

**Root cause:** The original gate measured only whether an argument reached any
field, not whether the field matched the argument's meaning.

**Workaround:** None; affected manifests were repaired and the fleet gate now
reports semantic counts.

**Real fix:** cli-health now applies deterministic collision, control-flag,
required-payload, and redundant-bind rules. The fleet ended at 0/0/0 errors
and 3 explicitly waived redundant-bind warnings.

**Owner:** cli-health / program-runtime.

**Refs:** `scenarios/cli-health/api/internal/services/manifestvalidation/semanticcheck.go`,
`scenarios/program-runtime/api/internal/bindings/semantic.go`,
`scenarios/program-runtime/tmp/repair-baseline/semantic_census.py`.

### 2026-08-07 — Async context could suppress binding requests

**Symptom:** A bare binding call in a program containing top-level `await`
returned an un-awaited coroutine and issued no request.

**Root cause:** `BridgeBinding.__call__` switched between synchronous execution
and coroutine creation based on whether an event loop was running.

**Workaround:** Parallel work uses `vrooli.gather` with zero-argument callables.

**Real fix:** Calls are eager and return awaitable `Handle` objects; `gather`
uses worker threads for explicit parallel fan-out.

**Owner:** program-runtime.

**Refs:** `kernel/host/engine.py`, `kernel/tests/test_execution_contract.py`.

### 2026-08-11 — Template example domain removed

The official `template-manager detemplate program-runtime` operation completed
successfully after generic binding fixtures were renamed. The notes API, CLI,
UI, schemas, database tables, and marker-bearing documentation were removed.

**Evidence:** `template-manager detemplate program-runtime` reported
`Detemplated program-runtime`; no notes domain directories remain under the
scenario's API, CLI, UI, or proto schema trees.

### 2026-08-06 — Optional IPython adapter is not host-available

**Symptom:** The current host Python installation has no IPython module.

**Root cause:** IPython is not installed in the current profile and has no
approved dependency entry yet.

**Workaround:** The kernel uses a standard-library JSON-lines engine with the
same session, namespace, and bounded-handle protocol, and reports spawn errors
explicitly rather than selecting an ungoverned fallback.

**Real fix:** Add the approved CPython/IPython host requirement through the
dependency analyzer, then layer the IPython adapter behind the same protocol.

**Owner:** program-runtime.

**Refs:** `kernel/host/engine.py`, `requirements/03-sessions/module.json`.

### 2026-08-06 — Agent-manager fleet workflow catalog has validation drift

**Symptom:** A lifecycle-managed program delegation call reaches agent-manager,
but the fleet `swarm-manager` workflow reconciliation fails because 15 workflow
files still contain the removed `budgets.maxCostUsd` field. The catalog is empty
for those workflows, so no successful delegated run can be demonstrated from
that fixture set.

**Root cause:** Agent-manager's current workflow schema uses
`budgets.maxChargeMicroUsd`; the fleet declarations have not been migrated.

**Workaround:** Delegation fails explicitly with the upstream workflow-not-found
response. The program-runtime bridge and its start/wait/result protocol are
covered by a deterministic integration test and the live failure path.

**Real fix:** Migrate the affected scenario-owned workflow declarations through
agent-manager's supported declaration workflow, then rerun the delegated-run
acceptance test against an active single-node workflow.

**Owner:** agent-manager / owning scenarios.

**Refs:** `api/internal/programs/delegator.go`,
`scenarios/agent-manager/docs/reference/scenario-declarations.md`,
`POST /api/v1/declarations/reconcile-scenario`.

### 2026-08-14 — Pre-phase corpus success rows are not trustworthy

The response projection previously selected the first repeated JSON field by
map iteration order. A stored `PROGRAM_STATUS_SUCCEEDED` row therefore proves
only that the bridge returned a successful transport response; it does not
prove that the program consumed the operation's intended primary rows. Mining
and discovery evaluation must exclude submissions created before the
descriptor-driven `rows_field` projection landed.

**Evidence:** the live `search-hub/query/query` probe now reports
`count() == 108`, matching the direct `ranked` response field, while an
ambiguous response fails explicitly with its candidate fields.

**Owner:** program-runtime.

**Refs:** `api/internal/bindings/registry.go`, `kernel/host/engine.py`,
`api/internal/programs/runner.go`.

### 2026-08-14 — Judged discovery remains model-sensitive on close ties

The governed `judge.default` path now receives only identity-joined candidates,
resolved argument names, effects, rank, and reviewed intent hints. The reviewed
51-case run exceeded the floor (39/43 positive cases met), but four positive
cases still selected a semantically adjacent operation. This is not hidden by
the threshold: the eval records wrong-selection separately and null remains a
valid outcome.

**Workaround:** Use `fast` when deterministic provider ranking is preferred;
use `judged` for the conservative one-or-none contract and inspect its
confidence/method. The provider corpus and suite are versioned evidence for
future prompt/model or reranker improvements.

**Owner:** program-runtime / ai-gateway.

**Refs:** `api/handlers/bindings/module.go`, `cli/domains/discovery/register.go`,
`evals/discovery.primary.json`.

### 2026-08-14 — Immutable regression validation lacks cli-health inventory

The pre-edit plan baseline is intentionally preserved, but its inventory does
not contain `cli-health`, which the generated full-plan validation scope now
requires. Plan validation therefore cannot produce a comparable terminal
verdict; phase transitions record an explicit validation override and name this
missing member rather than claiming a clean regression diff.

**Owner:** test-genie / plan-manager infrastructure.

**Refs:** plan baseline `program-runtime-trustworthy-results-and-a-self-improving-baseline`,
`docs/TESTING.md`.

### 2026-08-17 — RESOLVED: money-ledger bind census entry was stale

**Status:** Resolved during the 2026-08-18 binding-registry repair; the live
doctor now reports zero uncallable bindings.

**Symptom:** An older doctor census reported one uncallable
`money-ledger/ledger/accounts-create` binding. The current live census reports
`uncallable: 0`, `partial: 0` for that scenario, and the binding is callable.

**Root cause:** the manifest argument name does not match any field on the
request message, and no rename is declared.

**Workaround:** none. This entry is retained only to explain the historical
census and must not be used as current fleet evidence.

**Real fix:** The binding-registry repair and the live manifest census now
resolve the argument as `accountKind`; the stale count is corrected in the
current doctor output.

**Owner:** money-ledger.

**Refs:** `scenarios/money-ledger/cli/manifest.json`.

### 2026-08-17 — RESOLVED: prose-studio control-flag census entry was stale

**Status:** Resolved during the 2026-08-18 binding-registry repair; the live
doctor now reports zero uncallable and zero control-flag-bound bindings.

**Symptom:** An older doctor census reported ten uncallable
`prose-studio/prose/*` bindings. The current live census reports
`uncallable: 0`, `control_flags_bound: 0`, and no new findings for that
scenario.

**Root cause:** `--json` is a CLI renderer control flag, not a proto payload
field. cli-health's `binding.control_flag_bound` rule exists for exactly this
shape; the scenario's manifest predates or bypasses it.

**Workaround:** none. This entry is historical context, not an open fleet
problem.

**Real fix:** The binding-registry repair and current manifest census no longer
classify the renderer-only `json` flag as a payload binding.

**Owner:** prose-studio.

**Refs:** `scenarios/prose-studio/cli/manifest.json`,
`scenarios/cli-health/api/internal/services/manifestvalidation/findings.go`
(`CodeBindingControlFlagBound`).

### 2026-08-18 — RESOLVED: test and operator provenance polluted empirical mining

**Symptom:** Deliberately exercised authoring cases and operator probes could
appear in `programs mine`, `mine-refusals`, and `mine-unresolved`, making the
readiness board rank harness behavior as fleet friction.

**Root cause:** The mining paths did not apply the provenance boundary that
distinguishes agent behavior from test and operator evidence.

**Real fix:** All three mining surfaces exclude `PROVENANCE_TEST` and
`PROVENANCE_OPERATOR` by default and expose an explicit opt-in for diagnostics.
The focus board now has no gap derived from test-provenance evidence.

**Owner:** program-runtime.

**Refs:** `api/internal/programs/repository.go`,
`api/internal/programs/service.go`, `api/internal/programs/*_test.go`.

### 2026-08-18 — RESOLVED: unresolved-attempt ledger admitted local names

**Symptom:** Historical unresolved rows included local variable names such as
`handle1`, `handle2`, `handle_one`, `prior_result`, and `data_store`, which are
not capabilities.

**Root cause:** One or more write paths recorded every unresolved identifier
without enforcing the capability-shaped-name boundary.

**Real fix:** All write paths now admit capability-shaped names only, purge the
historical local-name pollution at startup, and cover accepted and rejected
shapes with regression tests. The live ledger reports capability-shaped rows
only (currently `test_geni`).

**Owner:** program-runtime.

**Refs:** `api/internal/programs/repository.go`,
`api/internal/programs/schema.go`, `api/internal/programs/preflight.go`,
`api/internal/programs/*_test.go`.

### 2026-08-18 — RESOLVED: Cross-stamp authoring result is below the reference pair

**Symptom:** The initial post-change authoring pair measured 4/12 and 4/12,
and an additional same-stamp pair measured 4/12 and 3/12, against
`authoring-brief@5(16 rules)`, while the fresh pre-change pair measured 7/12
and 7/12 against `authoring-brief@3(12 rules)`.

**Root cause:** The post-change brief contained additional rules, while strict
result oracles exposed under-specified version-2 tasks, an unavailable-data
assumption, missing session-aware preflight, and two natural kernel call shapes
(`group_by` values and `join(on=)`) that were not supported. The result was
measured, but it was not comparable as a like-for-like score and did not
demonstrate the plan's hoped-for improvement.

**Workaround:** None retained. The version-3 corpus is explicit and the
version-6 brief names the describe argument-row shape. Incomplete and
unavailable evaluator attempts remain excluded from the floor calculation.

**Real fix:** Session-aware preflight, grouped-count compatibility, `join(on=)`,
strict declared-field oracles, explicit corpus tasks, and the describe teaching
rule are implemented. Two complete runs at `authoring-brief@6(18 rules)` now
measure 9/12 and 9/12; the floor is re-derived to 8. The before/after result is
reported as cross-stamp evidence rather than a like-for-like claim.

**Owner:** program-runtime.

**Refs:** `evals/authoring.primary.json`, `internal/harness/contract.json`.

### 2026-08-18 — RESOLVED: project control-plane authoring case

The `project-cli` corpus case correctly authors
`vrooli.scenario.status(name="program-runtime")`. It previously reached a stale
root `vrooli-api` that supported runtime-registry schema 7 after the database
had advanced to schema 8, producing an opaque HTTP 500 while the freshly built
CLI path succeeded.

The root API is now current and the live program succeeds. The forward-schema
guard remains strict because an older writer must not mutate state whose newer
operational semantics it does not understand. The guard is now a typed
`SchemaCompatibilityError`; the Connect status path maps it to
`failed_precondition` with a supported `vrooli develop` rebuild instruction,
and tests pin both the storage and transport contracts. The corpus case remains
as an integration signal.

**Owner:** project control plane / operator.

**Refs:** `evals/authoring.primary.json`, `docs/reference/staleness-and-rebuild.md`,
`internal/scenarioruntime/schema.go`.

### 2026-08-17 — RESOLVED: the 30-second ceiling nobody set

**Symptom:** every synchronous call failed at ~30s. `programs submit` returned
`unavailable: unexpected EOF`; `authoring eval` returned the same after 3m26s
having actually run all twelve cases; judged discovery returned `HTTP 000`; both
inference examples failed with `RemoteDisconnected`.

**Root cause:** `api/main.go` called `apiserver.Run` without `WriteTimeout`, so
it inherited api-core's 30s default (`packages/api-core/server/server.go:382`).
Twenty other scenarios override it — search-hub allows 15 minutes, agent-manager
3, ai-gateway 2 — so the scenario built to run long programs had the shortest
deadline in its own dependency chain, while its sessions advertised a four-hour
wall budget and its kernel client waited 180s. Measured boundary: 28s succeeded,
32s failed.

**Real fix:** `internal/budgets` is now the single authority for every budget on
the execution path, arranged as a strictly nested ladder that `Validate()`
asserts at startup. The kernel's client timeouts are marshalled to it at spawn
(`PROGRAM_RUNTIME_BUDGETS`) rather than declared in Python, so the two languages
cannot drift.

**Owner:** program-runtime. **Refs:** `api/internal/budgets/budgets.go`,
`api/internal/budgets/budgets_test.go`, `kernel/host/engine.py` (`_Budgets`).

### 2026-08-17 — RESOLVED: `validate` and `capture` were broken by a rendered nil

**Symptom:** `validate("<scenario>")` returned `result: {}` for every scenario
while reporting SUCCEEDED. `capture` at the bridge refused any request that
omitted `kind`, with `capture kind "<nil>" is not accepted`.

**Root cause:** `fmt.Sprint` on a missing `map[string]any` key returns the string
`"<nil>"`, which is non-empty and therefore defeats a `!= ""` guard. `validate`
sent it to test-genie as a status filter and matched zero runs; `capture` tested
it against `== ""` so the documented `note` default was unreachable. Seven call
sites had the shape; exactly one carried the `!= "<nil>"` guard. The same defect
wrote 18 rows to `binding_invocations` whose provenance is the literal `<nil>`.

**Real fix:** `ProjectionBridge` decodes into a typed `projectionRequest` whose
`first()` accessor treats a missing key, a JSON null, and whitespace alike as
absent. Nine tests in `handlers/bindings/projection_test.go` cover the class,
including a table-driven guard that walks every declared verb.

**Owner:** program-runtime. **Refs:** `api/handlers/bindings/module.go`,
`api/handlers/bindings/projection_test.go`.

### 2026-08-17 — RESOLVED: per-invocation usage was silently zero

**Symptom:** all 426 successful `ai-gateway/inference/run` rows in
`binding_invocations` recorded 0 input tokens, 0 output tokens, and 0 cost.

**Root cause:** `invocationUsage` read `input_tokens`/`output_tokens`/
`cost_micros`, but the wire format is protojson, which emits camelCase. Session
level spend accounting reads a different path and was correct, which is why the
gap survived: one surface said the calls were free and another said they were
not, and nothing compared them.

**Real fix:** both spellings are accepted, and `served_by_provider` /
`served_by_model` are now recorded so a slow call caused by a dead local
candidate is distinguishable from a slow model. Live evidence after the fix: two
`ollama|qwen3.5:4b` rows at 610ms and 815ms with real token counts, beside a
pre-fix row at 46,048ms with no route recorded at all.

**Owner:** program-runtime. **Refs:** `api/internal/bindings/registry.go`,
`api/internal/bindings/schema.sql`.

### 2026-08-18 — RESOLVED: the inference and describe surfaces used parameter names models did not reach for

**Symptom:** two of twelve authoring-eval cases fail on keyword names, not on logic:

```
TypeError: _InferenceSurface.classify() got an unexpected keyword argument 'text'
TypeError: Namespace.describe() got an unexpected keyword argument 'binding_id'
```

**Root cause:** `ai.classify/extract/judge/write` take `source=`, and `describe`
takes `binding=`. Both are defensible names internally — `source` mirrors the
ai-gateway request field, `binding` mirrors the registry — but a model writing a
program reaches for `text=` and `binding_id=`, and `binding_id` is the exact name
this scenario uses for the same value everywhere else (the corpus, the doctor
output, discovery rows, the CLI). The surface disagrees with its own vocabulary.

**Workaround before the repair:** use `source=` and `binding=`. The failure was
immediate and its message named the offending keyword, so it cost one retry rather
than a wrong result.

**Real fix:** `text=` is now an additive alias for `source=` and `binding_id=` is
an additive alias for `binding=`. Collision attempts raise explicit `TypeError`s,
and the alias behavior is covered by the kernel suite. The repair is measured
under the new `authoring-brief@5(16 rules)` stamp; those post-change authoring
scores are recorded separately below because they are not like-for-like with the
earlier `@3` pair.

**Owner:** program-runtime.

**Refs:** `kernel/host/engine.py` (`_InferenceSurface._invoke`,
`Namespace.describe`), `evals/authoring.primary.json`.

### 2026-08-17 — RESOLVED: `WaitForProgram` shipped ahead of an operational target

**Status:** Resolved on 2026-08-19. `OT-P0-010` now promises bounded
asynchronous execution and `PRT-P0-010` traces it to the notification/deadline
API tests and to the governed `programs wait` CLI binding test.

**Symptom:** `ProgramService.WaitForProgram` and `program-runtime programs wait`
are live, tested, and documented, but no PRD operational target names them and
no requirement in `requirements/04-programs` traces to them.

**Root cause:** the RPC was added to repair a defect — the only way to await an
async program was a 50ms client-side poll loop that lived in the CLI, so no
other consumer could reuse it and it contradicted the project's never-poll
rule. The repair was authorised as engineering work; the contract change it
implies was not, and inventing a mapping onto an existing target would be a
false trace.

**Workaround:** none needed for behaviour. The gap is contractual: the scenario
has a public capability its PRD does not promise, so `business-health` cannot
grade it and the work ladder's W1 rung has nothing to check.

**Real fix:** The P0 operational target and its evidence-linked requirement are
now authored. `vrooli scenario requirements validate program-runtime` and the
business-health producer are the structural and live traceability gates.

**Owner:** program-runtime, with operator approval for the contract change.

**Refs:** `packages/proto/schemas/program-runtime/v1/programs/programs.proto`,
`api/internal/programs/service.go` (`Wait`), `cli/domains/programs/register.go`.

### 2026-08-17 — RESOLVED: the `guide` verb had no governed binding to compose

**Status:** Resolved on 2026-08-19. `guide` composes the callable
`prompt-manager/discover/discover` binding, projects `results`, and accepts the
runtime intent aliases while mapping them to the repeated `queries` request
field. Live program `prog_6f4b6093-d245-4208-87f8-dd855d92fc25` returned ten
rows and binding metadata.

**Symptom:** `guide("...")` fails with `projection "guide" is unavailable:
prompt-manager exposes no governed binding in the live registry`.

**Root cause:** prompt-manager contributes zero bindings to the live registry,
so there is no typed operation for the verb to call. The other three projection
verbs compose `search-hub/query/query`, `test-genie/runs/list`, and
`vrooli-memory/journal/note`.

**Workaround:** None required after the repair.

**Real fix:** Prompt Manager now ships the typed discovery binding and Program
Runtime's projection bridge supplies the schema-aware argument builder.

**Owner:** prompt-manager, with a one-line follow-up here.

**Refs:** `api/handlers/bindings/module.go` (`projectionVerbs`),
`docs/guides/program-construction.md` § Runtime verbs.

### 2026-08-17 — `validate` reads verdicts and cannot start a run

**Symptom:** `validate("program-runtime")` returns the latest recorded
test-genie run verdicts. It does not start a run, which the construction guide
states explicitly. (Separately, it used to return *nothing at all*; that was the
rendered-nil defect resolved above, not this stated boundary.)

**Root cause:** starting a run is a write that test-genie declares
run-ineligible, so no governed binding exists. Composing an ungoverned client
for it would put an unvalidated mutation behind a read-shaped verb.

**Workaround:** Start runs through the lifecycle (`vrooli scenario test <name>`)
and block once on the run id.

**Real fix:** test-genie publishes a run-start binding with an equivalent
governance contract, at which point the verb gains a `wait` mode.

**Owner:** test-genie / program-runtime.

**Refs:** `api/handlers/bindings/module.go` (`projectionVerbs`), `docs/TESTING.md`.

### 2026-08-17 — RESOLVED: the authoring eval was a stub that reported honest degradation

**Symptom:** `program-runtime authoring eval --json` always returned
`{"status": "unavailable", "reason": "no ai-gateway code-authoring route
resolved; tried hosted code.authoring and local code.local fallbacks"}`, and
`evals/authoring.primary.json` carried `floor: 0`.

**Root cause:** `RunAuthoringEval` in `api/handlers/programs/module.go` returned
a fixed literal. It never read the corpus, never resolved a route, and never
submitted a program. The two roles its reason named — `code.authoring` and
`code.local` — do not exist in the ai-gateway catalog, so the reason was
fabricated as well.

This is worse than an unimplemented feature. The response is byte-identical to
an honest degradation, so a reader concludes the dependency is missing rather
than the measurement. It also passes the floor gate silently: a floor
comparison is skipped when nothing was measured, and the floor was 0 because
both "pre-change" captures recorded the same stub output.

**Workaround:** None was needed; the number was never usable.

**Real fix:** Implemented in `api/internal/programs/authoring.go`. The eval
loads the corpus, authors each case through the governed
`ai-gateway/inference/run` binding using the `author.generator` role, submits
the result with `PROVENANCE_TEST` into a fresh session, and evaluates a
result-shaped oracle. `unavailable` is now reachable only on real route loss
and carries the underlying error. Six tests cover measured, wrong-result,
missed, route-loss, missing-corpus, and fence-stripping paths.

**Closing evidence:** The shared budget ladder removed the accidental 30-second
server deadline, the measured evaluator now completes its 12-case corpus, and
two complete `authoring-brief@6(18 rules)` runs scored 9/12 and 9/12 against a
derived floor of 8. Final-plan validation repeats a same-stamp pair so this
entry cannot regress into a single-run claim.

**Owner:** program-runtime.

**Refs:** `api/internal/programs/authoring.go`,
`api/internal/programs/authoring_test.go`, `api/handlers/programs/module.go`.

### 2026-08-19 — Symbol Search is absent from the Act supply chain

**Symptom:** Act cell A10 remains `IN-REACH`; the named `symbol-search`
contributor cannot resolve to a governed binding.

**Root cause:** Repository-wide inspection found no `scenarios/symbol-search`
directory, no `scenarios/symbol-search/cli/manifest.json`, and no
`packages/proto/schemas/symbol-search` package. This is a missing capability,
not a Program Runtime registry defect.

**Workaround:** A10 also names `code-facts`, `go-code-graph`, and
`typescript-code-graph`; programs can use the governed portions of that supply
while the compound cell remains conservatively partial.

**Real fix:** The Symbol Search capability owner must create or nominate the
scenario, then publish a typed proto contract and governed CLI manifest. Program
Runtime can re-audit A10 after those artifacts exist.

**Owner:** Symbol Search capability owner; Program Runtime owns the A10
consumer re-audit.

**Refs:** Scenario QA bug `knw-1787130446936328892`; `docs/spaces/act-space.md`
cell A10; `program-runtime bindings act --json` captured 2026-08-19.

### 2026-09-09 — Learning-verb close-out retains qualified external evidence

**Symptom:** The learning-verb implementation is exercised by deterministic
fixtures and a live BAS browser loop, but the BAS comparison remains partial.
The advice-application setpoint previously appeared unavailable. An earlier stopped-Memory
probe also appeared not to demand-start the dependency.

**Root cause:** BAS retains one invalid and three legacy records in the measured
window, while the setpoint reader previously summed only the first ten returned
comparison cohorts, so valid advice in later cohorts was omitted. An earlier
promotion probe also used an incorrectly shell-escaped step key.
The apparent demand-start failure was a runtime ordering/classification race:
Python performed a stale reachability preflight, while discovery classified the
control plane's stopped `port=0` JSON as `invalid_port`.

**Workaround:** Use the checked-in all-verbs example, live BAS receipts, and
focused workflow tests for implementation evidence. Treat the BAS aggregate as
partial and keep the safe device agent path non-actuating.

**Real fix:** The Memory comparison now exposes bounded all-cohort advice totals
without replacing preserved cohort rows, and the setpoint consumes those totals.
The current setpoint reports `36/36 = 1.0` with the source reliability warning
still visible. The correctly keyed promotion probe now finds the durable
fragment and returns `promoted=true`; clean the BAS legacy/invalid corpus.
Demand-start is repaired and proven by the stopped-Memory all-verb receipt.

**Owner:** browser-automation-studio, vrooli-memory, and program-runtime.

**Refs:** `/home/matthalloran8/.vrooli/plan-artifacts/learn-verbs-in-code-program-learning/baseline/after/README.md`,
`baseline/after/final/34-memory-demand-start.json`,
`baseline/after/final/35-demand-bridge-smoke.json`,
`baseline/after/final/20-bas-agent-task-only.txt`,
`baseline/after/final/21-compare-bas-agent.json`,
`baseline/after/final/27-device-compare-rerun.json`,
`baseline/after/final/28-bas-compare-rerun.json`, and
`baseline/after/final/04-fragment-contract.json`. Filed Plan Manager bug
`69c088a2-c7dc-4e56-965b-bdee2b1ba5cb`; downstream Scenario QA forwarding
returned 404 and remains sync-pending.

### 2026-09-09 — Test Genie programs metadata is not authoritative for these runs

**Symptom:** The structure phase passes, but the programs-phase metadata reports
zero observations or a presentation failure for BAS, Device Control, and Program
Runtime validation runs.

**Root cause:** The server-owned Test Genie presentation/observation metadata is
inconsistent with the terminal suite envelope in this shared workspace.

**Workaround:** Retain the full Test Genie receipts, use focused regressions and
direct program receipts for this change, and do not interpret zero observations
as zero program coverage.

**Real fix:** Test Genie must preserve phase observation records and derive its
presentation verdict from the server-owned terminal receipt.

**Owner:** test-genie.

**Refs:** `/home/matthalloran8/.vrooli/plan-artifacts/learn-verbs-in-code-program-learning/baseline/after/final/11-programs-structure.txt`,
`baseline/after/bas-programs-phase.json`. Filed Plan Manager bug
`2ce09d3d-b9bc-4fbd-92e4-dc2e332d97c7`; downstream Scenario QA forwarding
returned 404 and remains sync-pending.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## Work ladder

### 2026-09-06 — Nested program composition

- Rung: W3, implementation repair of the existing governed composition contract.
- W0: The active `program-runtime-improve` goal and PRD targets PRT-P0-001/002/003 support the change; no new product target is required.
- W1/W2: `business-health validate scenario program-runtime` and `vrooli scenario requirements validate program-runtime` passed before implementation.
- Defect evidence: local paired probes and live program `prog_dff713e1-c931-4bbf-a1fc-f45122f38f3c` reproduced missing `ai` inside a declared child. Report: `knw-1788665072824870467`.
- Repair: one public globals builder, declared nested input admission and copies, one bounded child envelope, child artifact metadata, bounded call nesting, and invocation context preserved through `gather`.
- Live evidence: `prog_55971fb6-9a94-4d3d-8dff-dcf2e74f16b4` asserted a successful nested batch, one schema-validated label, and a 64-character artifact digest. The temporary session was reclaimed.
- Validation: Test Genie unit run `20260906-035232-a0cd409f` completed FAIL on 2026-09-06. All eight new composition tests ran without failures. Its existing scaling test failed because `runtime.Caller` cannot locate the host under `-trimpath`; that test path is repaired without changing its assertions. A subsequent Unit Health API diagnostic reports `ok program-runtime/internal/programs` (6.511 s). The pre-change suite was cancelled while queued; it is not passing baseline evidence. The paired defect reproduction is the behavior baseline.
- Remaining suite findings: control-plane CLI/proto binding drift (`knw-1788670217499139681`); 13 usage-skill conformance probes after repairing the harness's old skill path (`knw-1788670218386465902`); UI QueryClient, locale, and coverage failures (`knw-1788670219241730261`). No finding or assertion was suppressed to make the suite pass.
- Coverage limit: Code Facts omits the existing Python `kernel` workspace; Unit Health confirms that selector is absent. Report `knw-1788667260047533054` owns discovery repair. The new Go tests exercise the real Python host but do not prove the whole Python suite ran.
- Limits: this repair does not provide transitive artifact pinning, child output-schema validation, or independent child budget allocation. Those require additional owner contracts and evidence before a broader composition guarantee.
- Artifacts: `/home/matthalloran8/.vrooli/research/scenario-program-composition-20260906/`.

### Previous ladder assessment

- Rung: W0
- Evidence: On 2026-08-19, `swarm-manager goals list --json` with the required named-mention filter again returned no goal whose name, title, description, or targets contain `program-runtime`. The canonical PRD was compared in both directions with the approved Plan Manager execution contract; the only uncovered shipped capability was bounded asynchronous execution.
- Decision: The operator explicitly authorized full execution of this plan, whose Phase 18 requires the bounded-async operational target. `OT-P0-010` and `PRT-P0-010` close that contract gap without mapping the behavior falsely onto an older target.
- Result: W0 passed for this approved change. No separate `docs/decisions/` record exists for Program Runtime, and no unrelated product scope was added.
- Measured: 2026-08-19

## Work ladder — adaptive learning lifecycle, 2026-09-09

- Rung: W0 → W1 → W3, scoped authorized extension and repairs.
- Authority: Operator approved full implementation and validation of the preceding learning assessment, including portable reviewed baselines and retention of learn.act.
- Contract: OT-P1-012 and OT-P1-013 express those outcomes; existing LV requirements retain their behavioral obligations.
- Evidence: Isolated probes showed three attempts reexecuting one invalid fragment, contradiction counts lost before choose, and an avoided default selected. Compatibility and publication require new executable coverage.
- Implementation: independent per-run verifiers; bounded fresh repairs; compatible, fresh, diverse evidence; separate test/replay cohorts; lost-write-response protection; immutable verifier inputs; durable feedback; reviewed baselines that retain learn.act; ordinary plugin asset packaging.
- Focused evidence: 205 kernel/Memory program regressions (plus 7 subtests) and 42 consumer workflow tests (plus 21 subtests), runtime task/storage/binding/contract/program packages, the full Memory API suite, and the full plugin API suite pass. The real Go-to-Python integration starts fresh kernels without AI, Memory, or Agent Manager, qualifies reuse after three runs, verifies the fourth cache hit, and proves test evidence cannot qualify operator execution.
- Public CLI evidence: baseline `prog_097089d8-5dbc-4999-a85c-0b846eb19515` and final baseline `prog_8c6005ca-f030-488e-b0ce-f1b19392b5e5` returned exact verified output with zero model calls. Adaptive `prog_af668024-c0f0-43b7-88b1-e2ef507327f4` generated and verified correct code in one model call. Runtime fragment writes were delivered; Memory delivery remains a separate receipt.
- Scoped Test Genie: BAS programs `20260909-172143-57ecde41` passed its phase. Runtime `20260909-171648-c4859d26` and programs rerun `20260909-172211-3e519e03`, Memory `20260909-171729-4fed24cc` and programs rerun `20260909-172212-274e1219`, device programs `20260909-171743-d3330263`, and plugin unit `20260909-171648-70f046c4` retain failed phases. Their aggregate PASS output is not accepted as clean evidence.
- Reported external findings: Test Genie aggregate/phase disagreement `knw-1788974375947586203`; Deployment Manager corpus declaration/binding mismatch `knw-1788974515571592865`; Memory/plugin UI coverage command failures `knw-1788974548432408552`; existing runtime program dependency/portfolio gates `knw-1788974830782855822`.
- Limits: verifiers must encode real postconditions; eligibility is not a statistical guarantee. Python fragment restrictions complement the existing process/governance boundary, not a hostile-code sandbox. Memory evidence scans are bounded to 1,000 entries and fail closed on truncation; a maintained owner index is the scaling follow-up. Optional service absence does not remove required runtime/domain dependencies. These findings preclude full scenario certification; requirement completion is not manually asserted.
