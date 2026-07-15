# Swarm Manager Operating Mode Authoring

## Purpose

Turn an operator-described agent methodology into a Swarm Manager operating mode
— a generic, composable, plan-first state machine expressed as **data** — and
prove it with a simulation walkthrough before it is trusted. Authoring is a data
task: no Go edits, no rebuild. A mode is a folder `scenarios/swarm-manager/modes/<id>/`
(`mode.json` + `example-runs/*.json`) loaded by one generic engine after a restart.

Use this skill when a repeated agent workflow should become a reusable, inspectable,
testable methodology loop with its own phase graph, reads/emits contracts, and
transitions.

Required reading (read the concept SSOT before proposing a graph):

- `path:scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md` — the model: target, Reads=base∪adapter, classification-on-transition, executed_by.
- `path:scenarios/swarm-manager/docs/internal/OPERATING-MODE-AUTHORING.md` — the how-to (scaffold → validate → simulate → restart).

## Scope Boundaries

**In scope:**

- deciding whether a repeated agent workflow needs a distinct operating mode;
- authoring or revising mode data, example runs, and non-delegated phase skills;
- validating and simulating every guarded or classified branch;
- restarting Swarm Manager so an operator-approved mode becomes registered.

**Out of scope:**

- adding per-mode runtime, API, CLI, UI, persistence, or statistics branches;
- implementing missing generic engine primitives as part of a data-only mode;
- converting deterministic service actions or operator/event waits into agent phases;
- changing dependencies, protobufs, or the operating-mode schema for one mode.

## The v2 primitives (the whole vocabulary)

- **Target** — the mode's unit of work, one of `plan-manager-plan` | `plan-ref` | `initiative`. The engine runs on a generic run context; a target adapter supplies target-specific reads, artifact root, and lock identity. Initiative is one adapter, not the base case. The simplest useful mode points the generic drain at a plan — no initiative.
- **Reads** — each phase declares `reads`: the named template variables it may reference, composed as **generic-base ∪ target-adapter** (a union, never a conditional). The loader rejects a read the target does not provide. There is no always-on initiative variable that is empty for a plan target — it simply does not exist.
- **Emits** — `declared_output.fields` names what a phase must emit; the resolution ladder extracts / reconstructs / honestly abstains against it. Guard fields must be declared outputs.
- **Transitions** — a guard is a generic field-predicate over declared output (`always`, `eq`, `in`, `all`/`any`/`not`, …). Any DAG; no branch vocabulary bespoke to a mode.
- **Classification-on-transition** — when the routing field is not directly emitted (the natural execute output is a *handoff*, not a routing enum), a transition declares `classify: {field, enum, from}` + `routes: {<value>: [...], ...}`. The engine derives the field from the handoff via the same ladder at the edge; an empty route is a guarded stop; abstain → `needs_attention`. **There is no `classify_progress` phase** — classification is a property of the edge, costing no agent round.
- **`executed_by` (composition)** — a phase may delegate to exactly one sub-mode, one level deep: `{ "id", "kind", "executed_by": "<sub-mode>", "transitions": [...] }` and nothing else (the sub-mode owns reads/prompt/emits/artifacts). Initiative modes **compose the generic `phased-plan-drain`** instead of duplicating an execute loop; the parent's guards route on the sub-mode's terminal output.

Do not use pre-v2 vocabulary: there is no `scope` (use `target`), no `classify_progress` phase (use classification-on-transition), no always-on flat initiative reads (reads are composed and declared), and no universal "exactly one reconcile phase" rule — reconcile is an initiative-adapter concern, absent from plan-target modes.

## Startup Routine

1. Read your operating guide in full (this skill) and the required docs above.
2. Inspect the attached `operating_mode_authoring` startup brief — it lists registered modes, their targets, phase counts, and decision metadata. It is live state, not the procedure.
3. If no brief is attached or the operator wants the latest catalog:

   ```bash
   swarm-manager operating-mode list
   swarm-manager operating-mode get --mode "<mode>"
   ```

## Workflow

### 1. Describe → classify → propose a graph (reuse first)

- Restate the workflow: name the **unit of work** (which becomes the `target`), what an agent does in each step, and where the loop branches or stops.
- Compare against existing modes. **Recommend an existing mode** unless the workflow needs a distinct phase graph, target, reads/emits contract, or governance policy. Explain any "do not build a new mode" conclusion concretely.
- When a new mode is warranted, propose the phase graph as data — target, phases (each with reads + declared output), transitions (with classification-on-transition where a route is derived from a handoff), and any `executed_by` delegation — **reusing existing modes as the head start**:
  - clone the closest existing mode (`--start-from`) rather than starting blank;
  - compose the generic `phased-plan-drain` via `executed_by` rather than re-authoring an execute/classify loop.

### 2. Scaffold (reuse-first)

Clone the closest existing mode, or scaffold blank only when nothing fits:

```bash
swarm-manager operating-mode scaffold --id "<mode-id>" --start-from "<existing-mode>" --label "<Label>" --description "<When to use it>"
swarm-manager operating-mode scaffold --id "<mode-id>" --label "<Label>" --description "<When to use it>"   # blank
```

`--start-from` re-homes the source mode's phase graph, reads, transitions, and
example-runs under the new id (and its prompt catalog prefix / artifact root /
event sources), so you edit a working methodology instead of assembling one.

### 3. Edit the data

Shape `modes/<id>/mode.json` to the methodology:

- identity + decision metadata (`label`, `description`, non-empty `best_for` / `not_for` / `tradeoffs`, optional `when_in_doubt_pick_instead`);
- `target.kind` (and `target.plan_ref.required` only on an initiative mode that delegates to a plan-target sub-mode);
- per-phase `reads`, `declared_output`, and `transitions` (use `classify`/`routes` for handoff-derived routing);
- `executed_by` on any phase that reuses another mode's loop.

Author one `example-runs/*.json` **per branch** you must prove — each seeds phase
outputs and declares the `expected_path` the real guards should walk. An
example-run's routing must be L1-derivable (the routing value emitted directly or
carried inline on the handoff); load-time derivation rejects a fixture that would
need the live L2 classifier.

Create or update one prompt-manager skill per non-delegated phase (SkillID =
`prompt.catalog_prefix` + phase suffix); repo-writing phases must not rely on
fallback prompts.

### 4. Validate + cover every branch

```bash
swarm-manager operating-mode validate --mode "<mode-id>"
```

Validate loads the mode fresh from disk through the real loader/validator (schema,
phase-graph integrity, guard fields, read-side validation, example-run replay) and
reports **branch coverage**: the guarded/classified branches no example-run walks.
Author an example-run for each uncovered branch until coverage is clean.

### 5. Simulation walkthrough (the trust step)

Before registration, walk the operator through the simulated flow of **every**
guarded/classified branch:

```bash
swarm-manager operating-mode simulate --mode "<mode-id>" --preset "<example-run-id>"
```

Simulate walks the phase graph through the **real** generic guards — no agents,
locks, or persistence. For each branch, narrate the path (which phase runs, what
it reads/emits, how the transition routed, where it stops) and confirm it matches
the operator's mental model. Registration comes only after the operator agrees the
walkthrough is correct. This is the final authoring step.

### 6. Restart + execute

```bash
vrooli scenario restart swarm-manager
swarm-manager operating-mode list
prompt-manager skill sync
```

The registry reloads every mode from disk at startup (a data reload, not a
rebuild). A `plan-manager-plan` / `plan-ref` mode then runs plan-first:

```bash
swarm-manager operating-mode start --mode "<mode-id>" --target "<plan-id|slug|path>"
```

## What you never touch for a data-only mode

- no per-mode branches in shared engine, lifecycle, stats, UI, CLI, prompt catalog, artifact, activity, lock, or simulation code;
- no direct backlog `spec.json` mutation for reconciliation;
- no raw package manager, no proto/schema regeneration, no phase-prompt bodies copied into the mode folder.

If a new mode appears to need one of those, the generic engine is missing a
reusable mechanism — raise that as a separate engine change, not a per-mode branch.

## Output Expectations

Produce exactly one of these outcomes:

- a concrete recommendation to reuse an existing mode, with the mismatched and matching workflow properties named;
- an operator-reviewable mode graph plus branch walkthrough, before registration;
- an implemented data-only mode whose validation and every example simulation pass;
- a bounded generic-engine capability gap when the workflow cannot be represented honestly by current primitives.

Always report the mode id, target kind, delegated modes, validation result, simulated branch paths, and whether the registry was restarted. Never claim registration or branch coverage from file presence alone.

## Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Recovery |
|---|---|---|---|
| `scaffold --start-from` rejects the source | The source mode is missing on disk, invalid, or has the same id as the destination | `swarm-manager operating-mode get --mode <source>` | Choose a valid distinct source or scaffold blank when no mode is structurally close. |
| Validation reports uncovered branches | No example run walks one or more guarded/classified routes | Read the branch list from `operating-mode validate` | Add the smallest example run that emits an L1-derivable routing value and proves that path. |
| Validation reports a missing prompt skill | A non-delegated phase lacks the catalog-derived skill id or metadata/source files | Compare the phase id with `prompt.catalog_prefix` and inspect `prompt-manager skill show <skill-id>` | Add or repair the phase skill, sync it, and validate again. Do not add a fallback prompt for a repo-writing phase. |
| Simulation needs the live classifier | The fixture supplies prose that only L2 classification could interpret | Inspect the transition's `classify.from` field and fixture output | Put the routing value directly in the emitted field or inline handoff for the fixture; keep live classifier behavior for runtime. |
| The mode validates on disk but is absent from the catalog | Swarm Manager still holds the pre-edit registry | `swarm-manager operating-mode list` | Restart Swarm Manager through the scenario lifecycle, then re-check the catalog. |
| A phase needs deterministic service work or an operator/event wait | Current agent-phase primitives cannot represent the workflow honestly | Compare the requested step with the current vocabulary in `EXECUTION-MODES.md` | Stop authoring and raise a generic executor/gate capability gap instead of disguising it as an agent phase. |
