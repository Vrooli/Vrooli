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

### 2026-08-17 — prose-studio binds a control flag as a payload argument

**Symptom:** `program-runtime bindings doctor --json` reports `uncallable: 10`.
Every one is a `prose-studio/prose/*` binding whose manifest maps an argument
named `json` onto a request message with no such field.

**Root cause:** `--json` is a CLI renderer control flag, not a proto payload
field. cli-health's `binding.control_flag_bound` rule exists for exactly this
shape; the scenario's manifest predates or bypasses it.

**Workaround:** None needed here. The bindings are refused at generation, so no
program can call them; the count is honest backlog rather than a live defect.

**Real fix:** Remove the `json` argument mapping from those eleven commands in
`scenarios/prose-studio/cli/manifest.json`.

**Owner:** prose-studio.

**Refs:** `scenarios/prose-studio/cli/manifest.json`,
`scenarios/cli-health/api/internal/services/manifestvalidation/findings.go`
(`CodeBindingControlFlagBound`).

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
states explicitly.

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
