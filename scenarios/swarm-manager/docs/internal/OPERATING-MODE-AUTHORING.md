# Operating Mode Authoring Guide

> **Status:** Active. Authoring an operating mode is a **data task** — no Go edits, no rebuild. A mode is a data folder interpreted by one generic engine. This guide is the how-to for scaffolding, validating, simulating, and shipping a mode. For *what a mode is and why the concept exists*, read the Northstar: [`docs/concepts/EXECUTION-MODES.md`](../concepts/EXECUTION-MODES.md).

## The one-sentence model

A mode is a folder under `scenarios/swarm-manager/modes/<id>/` — a `mode.json` plus `example-runs/*.json` — validated by [`.vrooli/schemas/operating-mode.schema.json`](../../../../.vrooli/schemas/operating-mode.schema.json) and loaded into the runtime by a single generic engine. There is **no** hardcoded Go mode definition and **no** static registry map. Adding or changing a mode is a data edit plus a restart, never a code change.

## The authoring loop

```
describe → propose a graph (reuse first) → scaffold → edit → validate → simulate walkthrough → restart → execute
                                               │         │        │             │
                                               │         │        │             └ walk every branch with the operator (real guards)
                                               │         │        └ loads from disk, reports errors + uncovered branches
                                               │         └ shape the phase graph, add an example-run per branch
                                               └ clone the closest existing mode, or start blank
```

Every step but the last is served by the typed `OperatingModeService` Connect contract; the CLI is a thin client over it.

### 1. Scaffold (reuse first)

A mode is generic, composable, and plan-first. Before scaffolding, propose the phase graph from the described workflow and **reuse** what exists: clone the closest existing mode instead of a blank template, and compose the generic `phased-plan-drain` via `executed_by` instead of re-authoring an execute loop.

```bash
# reuse-first: clone an existing registered mode as the head start
swarm-manager operating-mode scaffold --id my-mode --start-from holistic-loop \
  --label "My Mode" --description "One line on when to use it."

# blank: only when nothing existing is close
swarm-manager operating-mode scaffold --id my-mode --label "My Mode" \
  --description "One line on when to use it."
```

`--start-from` re-homes the source mode's phase graph, reads, transitions, and example-runs under the new id (regenerating its prompt catalog prefix, artifact root, and event sources), so you edit a working methodology — including any `executed_by` delegation it already carries — instead of assembling one. The blank template writes a minimal initiative-scoped mode (`execute → review → reconcile`) with a verdict-guarded branch and a passing happy-path example-run. Both are self-validated before anything is written, so a scaffold always produces a folder that loads and simulates. `TODO:` markers flag the prose you must fill in; `--force` overwrites a folder you are re-scaffolding.

### 2. Edit

Open `modes/my-mode/mode.json` and shape it to your methodology:

- **Identity & decision metadata** — `label`, `description`, and the `best_for` / `not_for` / `tradeoffs` lists (each must be non-empty) plus optional `when_in_doubt_pick_instead`. These drive the how-to-choose UI and the authoring startup brief.
- **Target & run strategy** — `target.kind` (the unit of work: `backlog-item` | `initiative` | `plan-execution`; an initiative-target mode may nest `target.plan_ref` for its bound-plan contract, which is the domain plan_ref field, not a target kind) and `run_strategy.kind`.
- **Inputs** — every phase mode declares one top-level `input_contract` with three deliberately separate sets: logical `specs` (namespaced id, JSON type/bounds, sensitivity, retention, description), exactly one `sources` binding per spec (`generic_provider`, `target_adapter`, `caller`, `derived`, or `default`), and prompt `aliases`. A phase's `reads` selects those aliases. The transitive compiler rejects competing/missing sources, unavailable capabilities, target/type mismatches, derived cycles, unsafe sensitive-value retention, unknown aliases, and unused bindings before an execution manifest is created.
- **Runtime parity** — prompt rendering resolves only the compiled phase bindings, not the old global read vocabulary. Provider descriptors declare their supported target kinds, value type, freshness, and failure policy; provider code only performs behavioral retrieval. New executions persist the canonical compiled contract and its SHA-256 digest, so later mode edits affect new executions only.
- **Phase graph** — `phase_graph.start_phase`, `phase_graph.terminal`, and the `phases` list. Each phase declares a `kind` (`investigate` / `execute` / `review` / `reconcile`), an `activity_purpose` (lowercase snake-case token), a `profile_key` (must start with `swarm-manager/`), optional `output_artifacts`, an optional `declared_output` schema, and its `transitions`.
- **Transitions & guards** — each transition is `{ "when": <guard>, "to": ["<phase>"] }`. A guard is a generic field-predicate: `{ "op": "always" }`, `{ "op": "eq", "field": "verdict", "value": "accepted" }`, or the composite forms `all` / `any` / `not`. A guard's `field` must be a declared output field of the phase, so a mode can never branch on output it doesn't promise to emit. This vocabulary can express any DAG — there is no branch type bespoke to a named mode.
- **Classification-on-transition** — when the routing field is not directly emitted (the natural execute output is a *handoff*, not a routing enum), the transition declares `{ "classify": { "field": "progress", "enum": ["continue","complete","blocked"], "from": "handoff" }, "routes": { "continue": ["execute"], "complete": [], "blocked": [] } }`. The engine derives `field` from the handoff via the resolution ladder at the edge, then routes; an empty route is a guarded stop, and an honest abstain parks the round in `needs_attention`. There is no `classify_progress` phase — classification costs no agent round. Example-run routing for a classified edge must be L1-derivable (the value emitted directly or carried inline on the handoff).
- **Composition (`executed_by`)** — a phase may delegate to exactly one sub-mode, one level deep: `{ "id": "execute", "kind": "execute", "executed_by": "phased-plan-drain", "transitions": [...] }` and nothing else (the sub-mode owns reads/prompt/emits/artifacts). The parent's transitions route on the sub-mode's terminal output. Initiative modes compose the generic drain instead of duplicating it (an initiative mode that delegates to a `plan-execution` sub-mode declares `target.plan_ref.required`; its bound plan becomes the sub-mode's unit of work).
- **Declared output** — `declared_output.fields` names the structured fields a phase must emit (`name`, `type`, `required`, `enum`, numeric/length bounds, nested `fields`). This one schema is what makes the phase robust to imperfect model output: the resolution ladder extracts, reconstructs, or honestly abstains against it (see the Northstar's resolution-ladder section). A required field named `progress` / `verdict` / `handoff` / `backlog_sync` sets the matching contract flag.
- **Policy blocks** — `prompt.catalog_prefix`, `artifact.root`, `profile.default_profile_key`, `backlog_sync`, `metrics`, `lock.initiative_exclusive`, and `ui.workspace_tab_id`.

Example-runs live in `modes/my-mode/example-runs/*.json`. Each seeds the
declared target fixture and an ordered list of per-phase outputs, then declares
the `expected_path` the real guards should walk. Initiative modes may supply
items and criteria; a plan-target mode instead receives the deterministic plan
fixture through its target adapter. The reserved `happy-path` id is the
simulator's default preset; a phase mode that ships any example-runs must own
one. Add a file per branch you want to prove (a replan, a non-accepting review,
a blocked stop).

### 3. Validate

```bash
swarm-manager operating-mode validate --mode my-mode
```

Validate loads the whole `modes/` directory fresh from disk (so on-disk edits and cross-mode references both resolve) and runs the real loader + semantic validator: schema structure, phase-graph integrity, guard fields, read-side validation, metric/profile invariants, and a replay of every example-run through the real guards (a fixture whose walked path differs from its `expected_path` fails). An invalid mode reports typed errors and a non-zero exit. A valid mode reports its phase and example-run counts **and its branch coverage**: the guarded/classified branches no example-run walks. Author an example-run per uncovered branch until coverage is clean — that is the checklist for the walkthrough. No restart required.

### 4. Simulate: the branch walkthrough

```bash
swarm-manager operating-mode simulate --mode my-mode          # on-disk draft (default)
swarm-manager operating-mode simulate --mode my-mode --preset review-not-accepted
swarm-manager operating-mode simulate --mode my-mode --registered   # the loaded copy
```

Simulate walks the mode's phase graph against an example-run through the **real** generic guards — the same mechanics a live round uses — with no agents, locks, or persistence. It prints the phase path and the guard-derived transition at each step. By default it loads the mode fresh from disk so it reflects unsaved edits; `--registered` walks the process registry's copy instead.

The simulation walkthrough is the **final trust step** before registration: with branch coverage clean, simulate **every** guarded/classified branch (one preset per example-run) and walk the operator through each simulated flow — which phase runs, what it reads and emits, how the transition routed, where it stops — confirming it matches their mental model. Register only after the operator agrees the walkthrough is correct.

### 5. Restart & execute

A newly scaffolded or edited mode is picked up by restarting swarm-manager — the registry reloads every mode from disk at startup. This is a data reload, **not a rebuild**:

```bash
vrooli scenario restart swarm-manager
```

After the restart the mode appears in `operating-mode list`, its prompt-catalog entries are generated automatically from the data, and it can be selected and executed on an initiative like any built-in mode.

## What you never touch

Authoring a mode is *only* the data folder. Do **not**:

- write or edit any `api/internal/operatingmode/mode_*.go` file (they no longer exist — modes are data);
- add a registry entry, a `TransitionRule`, or a branch to `state.go`, `artifact_applier.go`, `simulation.go`, the phase runner, the prompt catalog, stats, lock/activity packages, the UI, or the CLI;
- run a raw package manager or regenerate proto (a mode is not a contract change).

If a new mode appears to need one of those, it means the engine is missing a *generic* mechanism — raise that as its own change against the operating-mode subsystem, not as a per-mode branch.

## A mode implements an operation (the agent-operations layer)

A mode is the *how*; an **operation contract** is the *what*. The declarative
agent-operations layer ([`docs/concepts/AGENT-OPERATIONS.md`](../concepts/AGENT-OPERATIONS.md))
binds each operation to the mode that implements it. When you author a mode that
a backlog item or initiative will run autonomously, you also wire it into that
layer — all as data:

- **Operation contract** (`operation-contracts/<id>.json`): the provider-neutral
  behavior — required target capabilities, typed caller inputs, the typed result
  + closed outcomes, evidence expectations, and cancellation/retry. Materialized
  from `agentops.SeedOperationContracts()` via `go run ./api/cmd/genopscatalog .`;
  edit the Go SSOT, not the JSON by hand.
- **System-default binding** (`bindings/<operation>.json`): picks the mode (at a
  pinned `mode_revision`) that implements the operation. **The bound mode's
  `target.kind` must match the target the operation runs against** — the resolver
  fails closed (`ErrIncompatibleMode`) otherwise. This is why, e.g.,
  `execution-run`/`execution-retry` bind to a `plan-execution`-target mode
  (`execution-drain`, which delegates to `phased-plan-drain`), `execution-fixup`
  binds to a `backlog-item` mode (only backlog-item provides *both*
  execution-workspace and review-artifacts), and `initiative-review` binds to an
  `initiative` mode.
- **Transition policy** (`policy/<domain>-default.json`): selects the closed
  domain action that fires on each operation outcome. Policy names only registered
  actions — never code.

Inspect the whole thing without reading Go:

```bash
swarm-manager operating-mode list                 # every mode + its target
swarm-manager operating-mode get --mode <id>      # phases, reads, classification, delegation
swarm-manager agent-operations validate --operation <id> --target <sel>   # contract + binding resolves
swarm-manager agent-operations resolve-binding --operation <id> --target <sel>
```

### Where a mode is *not* a selectable methodology

`item-level` is **not** a real operating mode — it has no phase graph. It is
member-item strategy configuration (run each member item through its own
operation), retained as a compatibility placeholder until its cutover removes it.
Do not present it as a selectable mode, and do not point a mode's
`when_in_doubt_pick_instead` at it.

## Reference

- Concept & vocabulary (Northstar): [`docs/concepts/EXECUTION-MODES.md`](../concepts/EXECUTION-MODES.md)
- Agent-operations layer (contracts, bindings, policies): [`docs/concepts/AGENT-OPERATIONS.md`](../concepts/AGENT-OPERATIONS.md)
- Cutover ledger (the 14 target-bound behaviors): [`docs/internal/AGENT-CUTOVER-LEDGER.md`](./AGENT-CUTOVER-LEDGER.md)
- Schema: [`.vrooli/schemas/operating-mode.schema.json`](../../../../.vrooli/schemas/operating-mode.schema.json)
- Working examples: `scenarios/swarm-manager/modes/{holistic-loop,phased-plan-drain,backlog-research,backlog-review,execution-drain,initiative-review-loop}/`
- Engine: `scenarios/swarm-manager/api/internal/operatingmode/` (loader, validator, guard evaluator, resolution ladder, simulation, authoring)
