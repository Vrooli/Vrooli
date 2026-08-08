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

### 2026-08-06 — Template Manager detemplate service unavailable

**Symptom:** `template-manager detemplate program-runtime` cannot run because
Template Manager fails lifecycle setup with `go: updates to go.mod needed`.

**Root cause:** The unrelated Template Manager API module is out of sync with
its dependency graph, so its server-owned detemplate endpoint never starts.

**Workaround:** Real program-runtime API, CLI, routes, and measures no longer
register the notes example. Isolated generated notes packages remain pending
the official safety-checked cleanup.

**Real fix:** Repair Template Manager through Scenario Dependency Analyzer,
start it through lifecycle, run the official detemplate operation, and remove
remaining example artifacts and marker-bearing docs.

**Owner:** template-manager.

**Refs:** `template-manager detemplate program-runtime`,
`vrooli scenario logs template-manager`.

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
