# Problems — Program Runtime

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

### 2026-08-17 — `WaitForProgram` shipped ahead of an operational target

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

**Real fix:** author an operational target for bounded asynchronous execution
through `prompt-manager skill read prd-authoring`, then add the requirement with
validation refs to `internal/programs/wait_test.go` and the CLI evidence.

**Owner:** program-runtime, with operator approval for the contract change.

**Refs:** `packages/proto/schemas/program-runtime/v1/programs/programs.proto`,
`api/internal/programs/service.go` (`Wait`), `cli/domains/programs/register.go`.

### 2026-08-17 — the `guide` verb has no governed binding to compose

**Symptom:** `guide("...")` fails with `projection "guide" is unavailable:
prompt-manager exposes no governed binding in the live registry`.

**Root cause:** prompt-manager contributes zero bindings to the live registry,
so there is no typed operation for the verb to call. The other three projection
verbs compose `search-hub/query/query`, `test-genie/runs/list`, and
`vrooli-memory/journal/note`.

**Workaround:** Read skills with `prompt-manager skill read <name>` outside a
program.

**Real fix:** prompt-manager ships a `cli/manifest.json` binding for skill and
action lookup; the verb then needs only a binding id and an argument builder.

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

### 2026-08-17 — the authoring eval was a stub that reported honest degradation

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

**Remaining — the eval needs an asynchronous shape.** A full 12-case run authors
and executes twelve programs, so its wall time is twelve model round-trips plus
twelve kernel spawns. That exceeds the API server's response deadline and the
run ends as `unavailable: unexpected EOF` before it can return counts. The
measurement path itself is proven: a single authoring call against
`author.generator` returns a correct program (`z-ai/glm-5.2` via openrouter,
`validated: true`) using the object schema
`{"type":"object","properties":{"source":{"type":"string"}},"required":["source"]}`,
and the corpus/oracle/submission path is covered by six unit tests. What is
missing is the submission shape: `RunAuthoringEval` should accept and return
like `programs submit --async`, with a run id the caller blocks on once, rather
than holding one long request open.

Until that lands, the floor in `evals/authoring.primary.json` stays `0` and must
not be treated as a gate. It has never been derived from a measured run.

**Owner:** program-runtime.

**Refs:** `api/internal/programs/authoring.go`,
`api/internal/programs/authoring_test.go`, `api/handlers/programs/module.go`.

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

- Rung: W0
- Evidence: `swarm-manager goals list --json` with the required named-mention filter returned no goal whose name, title, or description contains `program-runtime`; the plan-manager objective is a separate execution artifact and is not a swarm-manager goal.
- Blocker: The contract cannot be compared against an approved named scenario goal, so W0 is unverifiable under the Scenario Work Ladder.
- Measured: 2026-08-07
