# Operating Mode Authoring Guide

> **Status:** Active. Authoring an operating mode is a **data task** — no Go edits, no rebuild. A mode is a data folder interpreted by one generic engine. This guide is the how-to for scaffolding, validating, simulating, and shipping a mode. For *what a mode is and why the concept exists*, read the Northstar: [`docs/concepts/EXECUTION-MODES.md`](../concepts/EXECUTION-MODES.md).

## The one-sentence model

A mode is a folder under `scenarios/swarm-manager/modes/<id>/` — a `mode.json` plus `example-runs/*.json` — validated by [`.vrooli/schemas/operating-mode.schema.json`](../../../../.vrooli/schemas/operating-mode.schema.json) and loaded into the runtime by a single generic engine. There is **no** hardcoded Go mode definition and **no** static registry map. Adding or changing a mode is a data edit plus a restart, never a code change.

## The authoring loop

```
scaffold  →  edit mode.json + example-runs  →  validate  →  simulate  →  restart  →  execute
   │                                              │            │            │
   └ writes a valid starter folder               └ loads from disk, reports errors
                                                               └ walks the real guards, asserts the path
                                                                            └ registry reloads the data (no rebuild)
```

Every step but the last is served by the typed `OperatingModeService` Connect contract; the CLI is a thin client over it.

### 1. Scaffold

```bash
swarm-manager operating-mode scaffold --id my-mode --label "My Mode" \
  --description "One line on when to use it."
```

This writes `modes/my-mode/mode.json` and `modes/my-mode/example-runs/happy-path.json` from a built-in template: a minimal but complete initiative-scoped phase mode (`execute → review → reconcile`) with a verdict-guarded branch and a passing happy-path example-run. The template is self-validated before anything is written, so a scaffold always produces a folder that loads and simulates. The `TODO:` markers flag the prose you must fill in. Pass `--force` to overwrite a folder you are re-scaffolding.

### 2. Edit

Open `modes/my-mode/mode.json` and shape it to your methodology:

- **Identity & decision metadata** — `label`, `description`, and the `best_for` / `not_for` / `tradeoffs` lists (each must be non-empty) plus optional `when_in_doubt_pick_instead`. These drive the how-to-choose UI and the authoring startup brief.
- **Scope & run strategy** — `scope.kind` (`initiative` for a phase mode) and `run_strategy.kind`.
- **Phase graph** — `phase_graph.start_phase`, `phase_graph.terminal`, and the `phases` list. Each phase declares a `kind` (`investigate` / `execute` / `review` / `reconcile`), an `activity_purpose` (lowercase snake-case token), a `profile_key` (must start with `swarm-manager/`), optional `output_artifacts`, an optional `declared_output` schema, and its `transitions`.
- **Transitions & guards** — each transition is `{ "when": <guard>, "to": ["<phase>"] }`. A guard is a generic field-predicate: `{ "op": "always" }`, `{ "op": "eq", "field": "verdict", "value": "accepted" }`, or the composite forms `all` / `any` / `not`. A guard's `field` must be a declared output field of the phase, so a mode can never branch on output it doesn't promise to emit. This vocabulary can express any DAG — there is no branch type bespoke to a named mode.
- **Declared output** — `declared_output.fields` names the structured fields a phase must emit (`name`, `type`, `required`, `enum`, numeric/length bounds, nested `fields`). This one schema is what makes the phase robust to imperfect model output: the resolution ladder extracts, reconstructs, or honestly abstains against it (see the Northstar's resolution-ladder section). A required field named `progress` / `verdict` / `handoff` / `backlog_sync` sets the matching contract flag.
- **Policy blocks** — `prompt.catalog_prefix`, `artifact.root`, `profile.default_profile_key`, `backlog_sync`, `metrics`, `lock.initiative_exclusive`, and `ui.workspace_tab_id`.

Example-runs live in `modes/my-mode/example-runs/*.json`. Each seeds a sandbox initiative and an ordered list of per-phase outputs, then declares the `expected_path` the real guards should walk. The reserved `happy-path` id is the simulator's default preset; a phase mode that ships any example-runs must own one. Add a file per branch you want to prove (a replan, a non-accepting review, a blocked stop).

### 3. Validate

```bash
swarm-manager operating-mode validate --mode my-mode
```

Validate loads the whole `modes/` directory fresh from disk (so on-disk edits and cross-mode references both resolve) and runs the real loader + semantic validator: schema structure, phase-graph integrity, guard fields, metric/profile invariants, and a replay of every example-run through the real guards (a fixture whose walked path differs from its `expected_path` fails). An invalid mode reports typed errors and a non-zero exit; a valid mode reports its phase and example-run counts. No restart required.

### 4. Simulate

```bash
swarm-manager operating-mode simulate --mode my-mode          # on-disk draft (default)
swarm-manager operating-mode simulate --mode my-mode --preset review-not-accepted
swarm-manager operating-mode simulate --mode my-mode --registered   # the loaded copy
```

Simulate walks the mode's phase graph against an example-run through the **real** generic guards — the same mechanics a live round uses — with no agents, locks, or persistence. It prints the phase path and the guard-derived transition at each step. By default it loads the mode fresh from disk so it reflects unsaved edits; `--registered` walks the process registry's copy instead. This is how you test a mode *before* it runs.

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

## Reference

- Concept & vocabulary (Northstar): [`docs/concepts/EXECUTION-MODES.md`](../concepts/EXECUTION-MODES.md)
- Schema: [`.vrooli/schemas/operating-mode.schema.json`](../../../../.vrooli/schemas/operating-mode.schema.json)
- Working examples: `scenarios/swarm-manager/modes/{item-level,holistic-loop,phased-plan-drain}/`
- Engine: `scenarios/swarm-manager/api/internal/operatingmode/` (loader, validator, guard evaluator, resolution ladder, simulation, authoring)
