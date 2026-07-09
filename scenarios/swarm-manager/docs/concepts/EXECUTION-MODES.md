# Operating Modes

> **Status:** Canonical Northstar concept — and the spec for **operating modes v2**, now **shipped**. An operating mode is a **generic, composable, plan-first, data-defined state machine for agentic software engineering**: the repeatable methodology loop a human runs when driving coding agents, expressed as data and interpreted by one generic engine. This document is the single source of truth for the whole model: mode-as-data, the **target** (unit of work), the **Reads/Emits contracts**, **classification-on-transition**, and **phase-delegation composition**. All of these are implemented: the mode-as-data engine, resolution ladder, guards, and example-runs, plus the v2 generalizations (targets, composed Reads, classification-on-transition, `executed_by`). This document describes the model; the code implements it. Where a nuance is still being refined, the Changelog records it.

## The Northstar

Swarm Manager is becoming the surface through which this project runs *all* agentic software-engineering work. The core question every such run must answer is not "what code do we write?" but **"what methodology do we run to get an agent to write it correctly?"** — what unit of work an operator-and-agent pair operates on at one time, in what phases, with what checkpoints, and how the loop reacts when a phase doesn't converge.

An **operating mode** makes that methodology a first-class object:

- **Reusable** — the same loop drives many units of work; you pick a mode, you don't re-derive a workflow.
- **Inspectable** — you can read what a mode does at a glance, as data, without reading source.
- **Testable** — a mode can be simulated and asserted before it is trusted with real work.
- **Robust** — the loop survives unreliable model output instead of failing on the first malformed message.
- **Generic** — a mode is not welded to any one kind of work. It declares a **target** (its unit of work); driving a swarm-manager initiative is *one* target among several, not the substrate everything else is bolted onto.
- **Composable** — a mode can delegate a phase to another mode (`executed_by`), so specialized loops are built by composing generic ones instead of duplicating them.

This is the conceptual move: "how we do software engineering in the agentic world" stops being an implicit implementation detail and becomes an explicit, extensible, data-defined capability — and the simplest useful mode is *plan-first*: point a loop at a plan and drain it.

## Why a mode must be data, not code

A methodology loop is a *specification* — a phase graph, the checkpoints, what each phase reads and emits, how branches are chosen. Specifications belong in data that one interpreter reads, not smeared across many source files. Encoding a mode in code has three failure modes the mode-as-data engine removes:

1. **Illegibility.** If the only complete description of a methodology is its source, no operator can understand it, and the concept has to be re-derived every time someone reasons about it. A mode should be legible from its data alone.
2. **Wrong representation.** When a mode is smeared across many structures — typed definitions, prompt catalogs, UI explainers, docs — *no single artifact is the mode*, and authoring a new one means editing many places and redeploying. A mode should be one folder you can scaffold, validate, simulate, and run with **zero code edits and no redeploy**.
3. **Brittleness.** Real model output is nondeterministic: envelopes come back malformed, or an agent emits a "final" message and then a subagent appends more so the chronologically-last message isn't the real answer. A loop that fails the phase on the first imperfect output is not trustworthy. The mode should *declare what good output looks like* and the engine should reliably extract, reconstruct, or honestly abstain — never silently guess.

The pattern we follow is knowledge-observatory's **validation-as-data**: a fixed generic vocabulary declared in data, one interpreter in code. V2 extends the same discipline to the *inputs* (Reads), the *unit of work* (targets), and the *branch decisions* (classification-on-transition): everything a mode does is declared; the engine only interprets.

## Vocabulary (single source of truth)

These terms mean exactly this throughout swarm-manager. Defined once here so they never have to be re-derived.

| Term | Definition |
|------|------------|
| **Mode** | A named, reusable methodology loop, expressed as a data folder. It owns the target kind, the phase graph, the run strategy, and per-phase policy. |
| **Target** | The mode's **unit of work** — the thing a run operates on. Declared in mode data as one of: `plan-manager-plan` (a canonical plan-manager plan), `plan-ref` (a plan file/reference not imported into swarm-manager), or `initiative` (a swarm-manager initiative with member items). Each target kind has an **adapter** that supplies target-specific reads, artifact roots, and locking identity. Initiative is one adapter among several, not the base case. |
| **Run context** | The generic per-run identity and state the engine threads through a mode run: which mode, which target instance, the durable rounds, the artifact roots, and the lock owner. It is target-agnostic; the target adapter fills in the target-specific parts. |
| **Phase** | One node in a mode's graph — a distinct step of the loop (e.g. `investigate`, `execute`, `review`). A phase declares what it **reads**, what it **emits** (`declared_output`), the prompt an agent runs, and — optionally — a sub-mode that executes it (`executed_by`). |
| **Round** | One execution of one phase: a single **fresh** agent runs the phase's prompt and returns a structured result. Rounds are durable and append-only; prior rounds are carried forward as context so a later round continues correctly. |
| **Slice** | The **elastic unit of work** an execute round advances. Not a persisted type — a *contract*: an execute round advances the frontier by one comprehensively-completable unit (a whole plan phase or the remainder of one) and states the true frontier in its handoff, so the next fresh agent continues from the right place. This is how a too-large phase is handled without failure. |
| **Handoff** | The structured result a round produces and the next round consumes — summary, blockers, next step, changed files, tests, and the declared frontier. The handoff is *the* continuity mechanism between fresh agents. |
| **Reads** | The declared input contract of a phase: the named variables its prompt template may reference. Composed as **generic-base provider ∪ target-adapter provider** — a union, never a conditional. The loader validates that every template slot is satisfiable by the composed set (read-side validation, symmetric with emit-side validation). |
| **Emits / Declared output** | The per-phase contract stating what a round is supposed to emit — field name, type, required/optional, enum, bounds (`declared_output.fields`). It is the single artifact that *validates* a round's result, *steers* the resolution ladder when output is imperfect, and *anchors* guard field-paths. |
| **Transition / Guard** | A declared edge from one phase to another, chosen by a **guard**: a generic **field-predicate condition** over the round's resolved output (field-path, operator, value). Guards can express any phase graph. There is no closed, mode-specific vocabulary of branch kinds; a branch is just a predicate over declared output. |
| **Classification-on-transition** | A transition-owned classification contract: when the routing field a guard needs is not directly emitted by the completed round, the transition declares how to **derive** it from the round's handoff — the same resolution ladder (deterministic extraction → schema-steered classifier → contract validation) applied at the edge. There is no dedicated classifier *phase*; classification is a property of the transition. Abstention routes the round to `needs_attention`, never a crash. |
| **`executed_by` (phase delegation)** | A phase-level declaration naming a sub-mode that executes the phase: the engine runs the sub-mode's loop *as* that phase, exactly one level deep. This is the sole composition mechanism — no inheritance, no deep nesting. |
| **Resolution ladder** | The four-rung mechanism (below) that turns raw, possibly-malformed agent output into the phase's declared structured result — or an honest abstain — instead of failing the phase. |
| **Example-run** | A mode-owned data fixture that seeds phase outputs and asserts the expected phase path through the *real* guards. Example-runs are how a mode is **simulated and tested** before use; they are the data behind the UI's simulation presets, and the loader replays them at startup. |

## A mode is a data folder

Each mode is a directory interpreted by one generic engine. Adding or changing a mode is a **data edit**, not a code change.

```
scenarios/swarm-manager/modes/<id>/
  mode.json          # identity + decision metadata, target, run strategy,
                     # phase graph (phases, transitions, guards, classification
                     # contracts, per-phase reads + declared output, executed_by),
                     # prompt skill refs, and profile / artifact / metrics /
                     # lock / ui policy (+ backlog-sync policy on the
                     # initiative adapter)
  example-runs/      # simulation fixtures: seeded outputs + expected phase paths
```

- **Schema (the vocabulary):** `.vrooli/schemas/operating-mode.schema.json` is the JSON Schema every `mode.json` and example-run validates against. It is generic — arbitrary phase graphs, branching guards, output contracts, targets, and delegation — with **no vocabulary bespoke to any named mode**.
- **Prompts (the executable instructions):** prompt bodies stay in prompt-manager. Mode data declares `prompt.catalog_prefix` and per-phase prompt metadata; the generic engine resolves those to phase SkillIDs and renders the skill through the same prompt seam the live spawn path uses. A mode folder is still legible end-to-end because the data names the resolved skill pointer without duplicating prompt text.
- **Engine (the interpreter):** one generic engine under `scenarios/swarm-manager/api/internal/operatingmode/` loads mode data into the typed in-memory definition the runtime consumes, validates it (schema + semantics, returning typed actionable errors), evaluates the declared guards, and runs the phases. A **data-backed registry** discovers modes from disk. Shared framework code gains **no mode-specific behavior branches** — new methodology is new data, not new code.

## Targets: the unit of work

**The v2 decoupling (D1).** A mode declares a `target` — what one run of the loop operates on. The engine's run substrate (round persistence, locking, artifact roots, prompt context) is built around the generic **run context**; a **target adapter** per kind supplies the target-specific parts. Three target kinds:

| Target kind | Unit of work | Adapter supplies |
|-------------|--------------|------------------|
| `plan-manager-plan` | A canonical plan-manager plan (execution id / slug) | Plan phase context, plan log access, plan-derived artifact root, plan-scoped lock identity |
| `plan-ref` | A plan file or reference **not** imported into swarm-manager (e.g. a repo-relative plan path) | The plan path/content read, a ref-derived artifact root and lock identity |
| `initiative` | A swarm-manager initiative and its member items | Initiative metadata, member items, acceptance criteria, backlog-sync capabilities, initiative-exclusive locking |

What this replaced: the pre-v2 run substrate was initiative-shaped end to end — phase resolution keyed off the initiative name, the run lock owner was the initiative, and every prompt was built from initiative data. Those are now adapter concerns behind the generic run context. **Initiative is one adapter among several**; nothing in the shared engine assumes an initiative exists.

The payoff is the **plan-first** entry point: the simplest useful run is "point the generic drain at a plan" — no initiative created, no member items, no backlog ceremony. Work that *does* warrant initiative tracking gets it by choosing the initiative target, which layers on member-item context, acceptance criteria, and backlog reconciliation.

## Reads: the input contract

**The v2 symmetry (D1b).** Emits have been declarative from the start: `declared_output.fields` drives extraction, validation, guards, and the UI Emits tab. Reads are the matching contract on the input side.

- **Composition, not branching.** The variables available to a phase's prompt template are the **union** of two providers: the **generic-base provider** (mode identity and label, phase id, run strategy, round number, profile key, operator note, prior rounds/handoffs, mode artifacts) and the **target-adapter provider** (initiative adapter: initiative name/title/description, member items, acceptance criteria; plan adapters: plan path and plan phase context). There is **no scope switch and no conditional emptiness** — a variable either exists in the composed set or it does not. A phase never receives an always-on initiative-shaped variable that happens to be empty because the target isn't an initiative.
- **Read-side validation.** The loader validates, at load time, that (a) a phase's prompt template references only reads its mode's composed provider set supplies, and (b) every template slot is satisfiable. This is the input-side parallel of the emit-side checks (deterministic extraction and contract validation against `declared_output`): a mode that references an unavailable read fails to load with a typed error, exactly as a guard referencing an undeclared output field does.
- **UI from data.** The phase viewer's **Reads** tab derives from this declared, composed contract — not from a hardcoded category list. Adding a target adapter or a read surfaces in the UI with no UI change.

The symmetry is the point: **a phase is fully specified by its Reads, its prompt, and its Emits — all three declared, all three validated, all three rendered from data.**

## The resolution ladder

The ladder is the robustness mechanism that lets modes survive real nondeterministic model output. **Data says WHAT good output is; code does HOW to obtain it.** Every phase result passes through four rungs, steered by the phase's declared output schema:

- **L0 — True-final-message detection (built-in, framework-level; a mode may opt out).** Agents sometimes emit a "final" message and then a subagent appends more, so the chronologically-last message is often not the real answer. L0 scans the last *N* agent messages (newest→older) for the declared output shape to identify the *true* final message.
- **L1 — Deterministic extraction.** Extract the phase's declared output schema from the identified message — field name / type / required / enum / bounds — with no model call. This is the fast, cheap, exact path.
- **L2 — AI-classifier fallback (steered by the declared schema).** When L1 cannot cleanly extract, route the raw output *plus the declared schema* through an LLM classification call that reconstructs the scalar/enum fields, and **abstain rather than guess** when it cannot. Object fields (handoff, backlog_sync) are never fabricated. The mechanism reuses `measures-go` over the `resource-ollama` gateway (`--role classify.routing`); output is prompt-constrained and Go-validated.
- **L3 — Contract validation.** Validate the resolved result against the phase's declared output contract. A violation is a typed, honest failure — not a silently-accepted wrong value.

A phase with malformed or trailing-subagent output resolves to the correct structured result or an honest abstain, driven entirely by the mode's declared output schema. An abstain routes the round to **`needs_attention`** for operator resolution — never a crash, never a fabricated field.

## Classification-on-transition

**The v2 generalization (D2).** Guards route on fields of the completed round's resolved output. But the most natural agent output for an execute round is a *handoff* — prose-plus-structure about what happened — not a routing enum. Pre-v2, a dedicated classifier **phase** (`classify_progress`) bridged that gap with a whole extra agent round whose only job was to read the previous handoff and emit `progress.decision`. That was a phase doing a transition's job.

V2 moves classification to the edge:

- A **transition declares a classification contract**: the routing field's name, type, and enum, plus how to derive it from the just-completed round's handoff when it was not directly emitted.
- The engine derives the field using the **same resolution ladder mechanics** it already owns — deterministic extraction from the handoff first, schema-steered classifier fallback second, contract validation last. No new machinery; the ladder is applied at the edge instead of only at the phase result.
- If the round *did* emit the field directly, the declaration is satisfied deterministically and no classifier runs — classification is a fallback, not a toll.
- **Abstention routes to `needs_attention`.** If the ladder cannot derive the routing field honestly, the round parks for operator attention. A routing decision is never fabricated and never crashes the loop.
- **The dedicated `classify_progress` phase is deleted.** Progress classification stops costing an agent round, stops appearing as a phase in the graph, and stops needing its own prompt skill. What remains in the graph is the real loop: execute → (classified transition) → execute / review / stop.

Guards stay exactly what they are — field-predicates over declared output. Classification-on-transition only changes *where the field can come from*: emitted by the round, or derived at the edge under a declared contract.

## Composition: `executed_by`

**The v2 composition model (D3).** A phase may declare `executed_by: <sub-mode-id>`. The engine then runs the sub-mode's loop *as* that phase: the sub-mode's rounds execute under the parent's run context and target, the sub-mode's terminal outcome becomes the phase's resolved output, and the parent's guards route on it.

Deliberate limits, chosen for legibility:

- **One level deep.** A sub-mode's phases may not themselves declare `executed_by`. The loader rejects nesting. A mode remains readable as at most a two-layer structure.
- **Explicit delegation, no inheritance.** A mode never "extends" or "overrides" another mode. Composition is a phase saying "this step *is* that loop," visible in the data at the exact point it applies.
- **Inline rendering, backend SSOT.** CLI and UI render the composed flow inline — the operator sees the sub-mode's phases expanded within the parent's graph — while the backend engine remains the single source of truth for routing. The UI never re-derives composition semantics.

The motivating use: **initiative modes compose the generic drain instead of duplicating it.** An initiative-target mode keeps what is genuinely initiative-shaped — investigation over member items, acceptance review, backlog reconciliation — and declares its execute phase `executed_by: phased-plan-drain`. Execution mechanics (slices, handoffs, frontier, continue/complete/blocked classification) exist once, in the generic mode, and every composing mode inherits fixes and improvements to them by reference, not by copy.

## Worked example: the generic phased-plan-drain

The minimal, load-bearing v2 mode — and the plan-first entry point. One phase, one loop, no initiative required.

- **Target:** `plan-manager-plan` or `plan-ref` — a plan that already says what to do.
- **Phase graph:** a single `execute` phase that loops on itself.
- **Prompt:** two sentences, plus the composed reads. Essentially: *"Read the plan at `{{PLAN_PATH}}`. Execute the next drainable slice — the earliest contiguous unit you can complete comprehensively — then emit your handoff stating the true frontier."* No MEMBER_ITEMS, no ACCEPTANCE_CRITERIA — those reads don't exist for a plan target, and the composed Reads contract means they are absent, not empty.
- **Reads:** base (round number, operator note, **accumulated prior handoffs** carried forward) ∪ plan adapter (plan path / plan phase context).
- **Emits:** the standard `handoff` object — summary, blockers, next_step, changed_files, tests, **frontier**.
- **Transitions (classification-on-transition):** one classified edge deriving `progress` ∈ {`continue`, `complete`, `blocked`} from the handoff:
  - `continue` → `execute` (loop: a fresh agent picks up from the declared frontier with all prior handoffs as context),
  - `complete` → stop (terminal; a composing parent's guards take over from here),
  - `blocked` → guarded stop, round parked `needs_attention`.
  - Derivation abstains honestly → `needs_attention`, same as blocked, with the abstain recorded.

Illustrative mode data (the `classify`/`routes` spelling below is the implemented v2 schema shape — `classify` declares `field`/`enum`/`from` (+ optional `description` to steer the L2 classifier), and `routes` maps every enum value to its target phase(s), empty = guarded stop):

```jsonc
{
  "kind": "operating-mode",
  "id": "phased-plan-drain",
  "target": { "kind": "plan-manager-plan" },
  "run_strategy": { "kind": "sequential_handoff" },
  "phase_graph": {
    "start_phase": "execute",
    "phases": [
      {
        "id": "execute",
        "kind": "execute",
        "writes_repo": true,
        "reads": ["PLAN_PATH", "PLAN_CONTEXT_JSON", "PRIOR_HANDOFFS_JSON", "OPERATOR_NOTE"],
        "declared_output": { "fields": [ { "name": "handoff", "type": "object", "required": true,
          "fields": [ /* summary, blockers, next_step, changed_files, tests, frontier */ ] } ] },
        "transitions": [
          {
            "classify": {
              "field": "progress",
              "enum": ["continue", "complete", "blocked"],
              "from": "handoff"
            },
            "routes": {
              "continue": ["execute"],
              "complete": [],
              "blocked": []
            }
          }
        ]
      }
    ]
  }
}
```

This one mode demonstrates every v2 concept: a non-initiative **target**, composed **Reads** with no initiative variables, declared **Emits**, **classification-on-transition** with no classifier phase, a self-loop expressed as an ordinary guarded route, and a shape small enough for other modes to **compose** via `executed_by`.

## Modes today, and their v2 shape

Modes are chosen by **how the work is shaped**, not by how much work there is or how present the operator is. What differs between modes is the *target* and the *unit of execution and validation*.

| Mode | Target | Unit of execution | Work shape it fits |
|------|--------|-------------------|--------------------|
| **item-level** (default for initiatives) | initiative | One backlog item per agent run | Items are right-sized, independent, reviewable in isolation, and stable through execution |
| **holistic-loop** | initiative | The whole initiative through `investigate → plan → execute → review → reconcile` | Items are coupled, likely to shift mid-flight, mis-scoped, or only validatable as a system |
| **phased-plan-drain** | plan (v2) | The plan, drained slice-by-slice with accumulated handoffs | A stable multi-phase plan exists (or is prepared once) and is drained by agents completing the earliest contiguous slice they can fully finish |

- **item-level** — one backlog item per agent run through the existing item pipeline (`backlog → researching → ready → queued → in_progress → completed/failed`); refinement via the workshop loop converging to a `plan_ref`. Strengths: parallelism, per-item provenance, bounded blast radius. Failure mode: items coupled by a shared substrate produce broken intermediate states.
- **holistic-loop** — durable mode rounds over the whole initiative: `investigate → plan → execute → review → reconcile`. Its `plan` phase authors and binds the canonical plan-manager plan, and its `execute` phase is `executed_by: phased-plan-drain` — the shipped first `executed_by` composition. The composed replan loop is the drain's `blocked` outcome routing back to investigate (`progress = blocked → investigate`, `progress = complete → review`); review loops back to execute on `verdict=changes_requested`. Strengths: correct unit of validation for coupled work; investigation is a first-class step. Failure mode: loses parallelism; wrong for genuinely independent, stable items.
- **phased-plan-drain** — the **generic plan drain** described above, shipped in its v2 form: a single `execute` phase on a `plan-manager-plan` target, looping on itself through one classified edge (`progress` ∈ continue/complete/blocked derived from the handoff), with complete and blocked as guarded stops. The pre-v2 initiative-coupled shape (`prepare_plan → execute_next → classify_progress → review → reconcile`) is gone: the classifier phase dissolved into classification-on-transition, and plan-preparation / review / reconcile concerns move to the composing initiative-target modes (`executed_by`). Initiative-keyed surfaces (mode switch, phase start) reject plan-target modes with a typed error; the dedicated start-on-plan surface (`OperatingModeService.StartTargetPhase`, CLI `swarm-manager operating-mode start --mode phased-plan-drain --target <plan-id>`) runs the drain live on a bare plan with no initiative.

### Backlog items in initiative-target modes: tracking, not execution

When an initiative runs in `holistic-loop` (or composes the drain), its member items don't disappear — they survive as **tracking and scope markers** (what the initiative claims to deliver, progress reporting, partial cancellation, cross-initiative dependency targets). What changes is that they are no longer *independent execution units*: their `plan_ref` points to item-level scope context, their workshop rounds are historical context, and they are marked `completed` as the initiative-level plan covers them. Items remain the unit of *visibility* and *scope* while the unit of *execution* and *validation* moves up to the initiative. This whole section is an **initiative-adapter concern** — plan-target runs have no member items at all.

## Why the distinction exists (the empirical cue)

Swarm Manager's original flow assumes **backlog items are the right unit of execution and validation** — atomic enough that one agent run picks up one item, executes it, and hands back a discrete outcome. That assumption is load-bearing: it is what lets an operator review N initiatives in parallel and gives execution bounded blast radius. It holds when items are correctly scoped, independent, and stable. It breaks when items are coupled by the very thing being changed, likely to shift mid-flight, or mis-scoped.

The **2026-04-27 sandboxing trap** surfaced this. Two initiatives (`agent-sandbox-audit-foundation`, `protected-agent-sandboxing`) both restructured how agent runs flow through the workspace sandbox. Completing them item-by-item left the system in an inconsistent intermediate state for the items still in flight — changes to sandbox routing affected every other run, including swarm-manager's own runs — so swarm-manager stopped working. The work succeeded only when the operator stepped *outside* the harness and ran: investigate current state in code → generate a consolidated plan → stage execution against it → re-investigate after each wave and revise. The work didn't fail because the items were big; it failed because **the items couldn't be validated in isolation** — only the system as a whole could be, after the coupled work reached a coherent state. Rather than treat that recovery as "an escape hatch when swarm-manager fails," we recognize it as a first-class operating mode. The operator's continuous presence during recovery was an artifact of having no in-harness support, not a requirement of the work — a structured loop running *inside* swarm-manager absorbs the same work at the operator's normal async cadence.

The v2 target decoupling is the same lesson one step further: sometimes the right unit of work isn't an initiative *at all* — it's just a plan. The methodology loop shouldn't require importing the work into swarm-manager's initiative machinery before it can run.

## Choosing and switching modes

The right mode is a property of how the work is shaped, not its size.

- **Use the plan drain directly** when a stable plan already exists and the work is "execute this plan": no initiative ceremony, accumulated handoffs carry continuity, built-in classification loops or stops.
- **Use `item-level`** when items are right-sized for one run, loosely coupled, reviewable in isolation, stable, and parallelism is valuable.
- **Use an initiative-target loop mode** when items are coupled by a shared substrate, intermediate states leave the system inconsistent, item shape will shift mid-execution, the natural unit of validation is "does the system as a whole work," the right plan can only be authored after investigating cross-item ground truth, or replanning is expected.

For initiatives, a mode is a property of the initiative *at a point in time*, not immutable. An initiative may begin `item-level`, hit a shape mismatch, and be promoted to a loop mode; after an initiative-level plan converges, residual independent work may be drained back to `item-level` for parallel finish-out. The switch is a first-class, operator-chosen lifecycle action — generic initiative create/update does not mutate mode; switching into a non-default mode handles active item-level executions explicitly, and switching out is blocked while a mode round is active.

## Understanding a mode: the Flow tab

Every mode is inspectable from the UI without reading source or raw JSON first. The operating-mode detail page has a **Flow** tab that renders **one shared phase viewer** with four concern tabs — **Instructions / Reads / Emits / Transition** — and a **data-source control** that swaps the fill without changing the view. The same viewer backs the **Phases** tab, so a phase is understood through one surface everywhere. Three sources:

- **Contract** — the phase's static contract: the agent prompt template with `{{VARIABLE}}` slots still unfilled, and what the phase is *defined* to read, emit, and route to. In v2 the **Reads** tab renders the declared composed contract (base ∪ target adapter) instead of a hardcoded category list, and the **Transition** tab shows classification contracts on classified edges.
- **Simulation preset** — deterministic, in-memory walks of the phase graph against illustrative data (from the mode's **example-runs**). Presets seed phase outputs to exercise real transition branches but never spawn agents, acquire locks, or persist state.
- **Live round** — the actual rounds recorded for a linked run.

For Simulation and Live, the prompt is rendered server-side through the *same* prompt-render seam the real spawn path uses, and a parity test asserts the preview is byte-identical to what an agent receives. When that seam is unavailable the tab degrades to the resolved variable map rather than erroring. A phase delegated via `executed_by` renders its sub-mode's flow inline. Because presets, resolved SkillIDs, prompt rendering, reads, and transitions are sourced from the mode's data, adding a mode, a target, or a branch surfaces in Flow with no mode-specific UI.

## Authoring a mode

Authoring is self-serve, **agent-driven**, and writes **data**: scaffold a mode folder from a template, validate it against the schema and semantics (including read-side validation and example-run replay), simulate it against its example-runs, and run it — with **zero Go edits and no redeploy**. The `operating_mode_authoring` session drives this flow, including the describe→mode path where an operator describes a methodology and an agent authors the mode data and walks the operator through a simulation before it is trusted. See [DOC: ../internal/OPERATING-MODE-AUTHORING.md] for the authoring procedure.

## Decisions (v2)

The three pinned architecture decisions, with the alternatives that were considered and rejected:

- **D1 — Unit-of-work decoupling (targets).** A mode declares its `target` (`plan-manager-plan` | `plan-ref` | `initiative`); the engine runs on a generic run context and target adapters supply the specifics; the generic plan drain runs plan-first with no initiative. *Rejected:* keeping initiative as the universal substrate and importing every plan into an initiative first — that taxes the simplest case with ceremony it doesn't need and welds a generic engine to one product concept.
- **D1b — Reads/Emits contract symmetry.** Phase reads are declared and composed (generic-base ∪ target-adapter, by union), read-validated at load time, and rendered from data. *Rejected:* the always-on flat variable map with conditionally-empty initiative fields — conditional emptiness is an undeclared contract, invisible to validation and misleading in the UI.
- **D2 — Classification-on-transition.** Transitions declare classification contracts; routing fields are derived from the completed handoff via the existing resolution-ladder mechanics; abstention → `needs_attention`; the dedicated `classify_progress` agent phase is deleted. *Rejected:* keeping classification as a phase — it spends an agent round on an edge decision and pollutes the phase graph with machinery instead of methodology.
- **D3 — One-level `executed_by` composition.** A phase may delegate to exactly one sub-mode, one level deep, rendered inline, with the backend as routing SSOT; initiative modes compose the generic drain instead of duplicating execution+classification. *Rejected:* runtime inheritance between modes (implicit, illegible — you can no longer read a mode from its folder) and deep/recursive nesting (destroys at-a-glance legibility for marginal expressive gain).
- **Also rejected:** a deterministic natural-language→mode generator. Authoring stays agent-driven — an agent writes mode data and proves it via simulation; a template-stamping generator would produce modes nobody validated and nobody understands.

### D3 implementation decisions (composition, as shipped)

Where the composition spec under-determined behavior, the implementation pinned the simplest legible semantics:

- **Data shape.** A delegated phase is `{ "id", "kind", "executed_by": <sub-mode-id>, "transitions": [...] }` and NOTHING else: reads, prompt, declared output, artifacts, result bindings, purposes, profile, `writes_repo`, and metrics are all schema-forbidden on it — the sub-mode's phases own the entire execution surface. Its `transitions` are the parent's routing over the sub-mode's terminal outcome; their guard fields are load-validated against the union of the sub-mode's declared output fields and edge-classified routing fields (`delegatedOutcomeFields`).
- **Runtime.** Each round of a delegated phase is one sub-mode round persisted under the PARENT run — parent mode, parent scope id, parent phase id, parent ownership lock — with `delegated_mode` / `delegated_phase` payload markers recording the effective execution contract (`effectiveRoundExecution` resolves every downstream surface — resolution ladder, edge classification, prompt render — through them). After a round completes, the SUB-mode's guards evaluate first: an onward sub-route (the drain's `continue`) keeps the delegated phase in progress (the next startable phase is the delegated phase again, resuming the sub-loop where it left off); a sub-mode stop (guarded stop or terminal sub-phase) ends the delegation and the parent's transitions route on the same resolved output. Re-entry after a stop (e.g. review `changes_requested` → execute) starts the sub-loop fresh at the sub-mode's start phase.
- **Target-context flow.** The delegating mode supplies the sub-mode's target from its own resolved target instance: same target kind passes through unchanged; an initiative-target mode may delegate to a `plan-manager-plan`-target sub-mode **only** when it declares `target.plan_ref.required` — the initiative's bound plan becomes the sub-mode's unit of work. Every other combination is rejected at load. Correspondingly, a missing bound plan is tolerated (surfaced as missing context, not an error) for the start phase *and* for any phase whose own output contract emits `plan_ref` — the phase that authors the plan must be able to run before the binding exists.
- **Replan, composed.** holistic-loop's `replan_needed` flag (and its replan-rate metric sample) is retired: the composed replan loop is the drain's `blocked` outcome routing back to `investigate` via an ordinary parent guard (`progress = blocked → investigate`, `progress = complete → review`).
- **Plan-first start surface.** `OperatingModeService.StartTargetPhase` (CLI: `swarm-manager operating-mode start --mode <id> --target <plan-id|slug|path>`) starts a round of a non-initiative-target mode directly on its target. Plan-target rounds store under `<dataRoot>/mode-targets/<kind>/<sanitized-id>/` and lock on the plan ownership key (`plan--<id>` / `plan-ref--<id>`); round follow-up reuses the ordinary round actions with the resolved scope id plus an explicit `--mode`. Initiative-target modes are rejected on this surface — initiatives keep the initiative-keyed `StartPhase`.

V2 is a **greenfield hard cutover**: old initiative-coupled concepts (initiative-keyed run substrate, always-on initiative reads, the classifier phase) are replaced, not shimmed.

## References

- [`docs/concepts/ARCHITECTURE.md`](./ARCHITECTURE.md) — the staging-and-review framing operating modes build on.
- [`docs/guides/workshop-workflow.md`](../guides/workshop-workflow.md) — the item-level workshop loop, readiness model, and `plan_ref` handoff.
- [`docs/guides/holistic-loop-mode.md`](../guides/holistic-loop-mode.md) — operator workflow for holistic-loop mode.
- [`docs/guides/phased-plan-drain-mode.md`](../guides/phased-plan-drain-mode.md) — operator workflow for the shipped (pre-v2) phased-plan-drain mode.
- [DOC: ../internal/OPERATING-MODE-AUTHORING.md] — how to author a mode as data.
- `.vrooli/schemas/operating-mode.schema.json` — the generic mode schema (the vocabulary).
- `scenarios/swarm-manager/modes/<id>/` — the per-mode data folders (`mode.json`, `example-runs/`).
- [`docs/concepts/GLOSSARY.md`](./GLOSSARY.md) — scenario term definitions.

## Changelog

- **2026-07-09** — **Phase-delegation composition (`executed_by`) SHIPPED** (v2 rebuild phase 5): the schema/loader/engine implement one-level delegation per the D3 implementation decisions above; **holistic-loop is the first composed mode** — its plan phase now authors and binds the canonical plan-manager plan (required `plan_ref` output; `target.plan_ref.required`), its execute phase is `executed_by: phased-plan-drain` (inline continue loop, `complete → review`, `blocked → investigate` as the composed replan), and its four example-runs walk the composed guards (guard-replay green at load). The **start-on-plan surface landed** (`StartTargetPhase` RPC + `operating-mode start` CLI) with plan-scoped round storage and locking, so the drain runs live on a bare plan with no initiative. Retired: the duplicated inline execution/classification machinery and the `swarm-manager-phased-plan-{prepare,classify-progress,review,reconcile}` and `swarm-manager-holistic-loop-execute` skills; `planimport` now rejects plan-target modes server-side.
- **2026-07-09** — **Generic phased-plan-drain SHIPPED** (v2 rebuild phase 4): `modes/phased-plan-drain/` is now the worked example above as real data — `plan-manager-plan` target, one self-looping `execute` phase, one classified edge deriving `progress` from the handoff, complete/blocked as guarded stops, three L1-derivable example-runs (happy-path / complete-first-slice / blocked). The engine now allows a mode with no terminal phase when it has at least one guarded stop, the simulation envelope preserves seeded fields the typed result does not model (matching the runtime's raw-envelope fidelity), and initiative-keyed surfaces (mode switch, phase start, live render) reject plan-target modes with typed errors. At this phase checkpoint the start-on-plan API was still missing; the phase 5 entry above records its closure.
- **2026-07-09** — **V2 spec pinned** ahead of implementation: operating modes become generic, composable, plan-first. Added the **target** (unit-of-work) concept with three target kinds and initiative-as-adapter (D1); the composed, load-validated **Reads** contract symmetric with Emits (D1b); **classification-on-transition** replacing the dedicated `classify_progress` phase, with abstain→`needs_attention` (D2); one-level **`executed_by`** phase delegation with inline rendering and backend routing SSOT (D3); the generic phased-plan-drain worked example; and the Decisions section recording the rejected alternatives (inheritance, NL→mode generator, always-on flat reads).
- **2026-07-08** — Closed the two behavior gaps the spec had declared but the engine omitted: the **elastic-slice contract** is now encoded in the execute-phase prompt (a shared `ELASTIC_SLICE_SNIPPET`) and the handoff schema (a `frontier` field on `handoff`, carried through Proto + Connect), and the **review reloop** is now real data — a `changes_requested` verdict routes review back to execute/`execute_next` (with the reconcile auto-start gated on the guard actually routing there), covered by mode-owned example-runs.
- **2026-07-08** — Rewritten as the canonical Northstar concept and the spec for the data-driven rebuild: operating modes are reusable, inspectable, testable methodology loops; a mode is a data folder interpreted by one generic engine; added the single-source vocabulary, the resolution ladder (L0–L3), the elastic-slice contract, and example-runs-as-data. Reframed the architecture from a hardcoded Go registry to the data SSOT target.
- **2026-05-01** — Documented the registry-owned authoring architecture (superseded by the data-driven model above).
- **2026-04-30** — Added mode-aware prompt routing, stable phase activity purposes, typed event/stat contracts, and the first Modes stats surface.
- **2026-04-28** — Initial document, authored after the 2026-04-27 sandboxing trap surfaced "items are the right unit of execution and validation" as a load-bearing assumption that doesn't always hold.
