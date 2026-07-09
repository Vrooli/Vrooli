# Operating Modes

> **Status:** Canonical Northstar concept — and the spec the operating-mode subsystem is being rebuilt onto. An operating mode is a **reusable, inspectable, testable methodology loop for agentic software engineering**: the repeatable state-machine a human runs when driving coding agents. This document defines what a mode is, why the concept exists, the single vocabulary it uses, and the **target data-driven architecture** — a mode is *data* interpreted by *one generic engine*, not hardcoded logic. Three modes run today (`item-level`, `holistic-loop`, `phased-plan-drain`); the rebuild expresses them purely as data and deletes the hardcoded definitions. Where this doc describes an architecture that is still being landed, it is describing the target the operating-mode rebuild plan implements, not a fiction.

## The Northstar

Swarm Manager is becoming the surface through which this project runs *all* agentic software-engineering work. The core question every such run must answer is not "what code do we write?" but **"what methodology do we run to get an agent to write it correctly?"** — what unit of work an operator-and-agent pair operates on at one time, in what phases, with what checkpoints, and how the loop reacts when a phase doesn't converge.

An **operating mode** makes that methodology a first-class object:

- **Reusable** — the same loop drives many initiatives; you pick a mode, you don't re-derive a workflow.
- **Inspectable** — you can read what a mode does at a glance, as data, without reading source.
- **Testable** — a mode can be simulated and asserted before it is trusted with real work.
- **Robust** — the loop survives unreliable model output instead of failing on the first malformed message.

This is the conceptual move: "how we do software engineering in the agentic world" stops being an implicit implementation detail and becomes an explicit, extensible, data-defined capability.

## Why a mode must be data, not code

A methodology loop is a *specification* — a phase graph, the checkpoints, what each phase reads and emits, how branches are chosen. Specifications belong in data that one interpreter reads, not smeared across many source files. Encoding a mode in code has three failure modes this rebuild removes:

1. **Illegibility.** If the only complete description of a methodology is its source, no operator can understand it, and the concept has to be re-derived every time someone reasons about it. A mode should be legible from its data alone.
2. **Wrong representation.** When a mode is smeared across many structures — typed definitions, prompt catalogs, UI explainers, docs — *no single artifact is the mode*, and authoring a new one means editing many places and redeploying. A mode should be one folder you can scaffold, validate, simulate, and run with **zero code edits and no redeploy**.
3. **Brittleness.** Real model output is nondeterministic: envelopes come back malformed, or an agent emits a "final" message and then a subagent appends more so the chronologically-last message isn't the real answer. A loop that fails the phase on the first imperfect output is not trustworthy. The mode should *declare what good output looks like* and the engine should reliably extract, reconstruct, or honestly abstain — never silently guess.

The pattern we follow is knowledge-observatory's **validation-as-data**: a fixed generic vocabulary declared in data, one interpreter in code (KO: `docs/manifest.json` → `doccontract.Validation` → `docvalidation.validateContent`). Operating modes apply the same shape to methodology loops.

## Vocabulary (single source of truth)

These terms mean exactly this throughout swarm-manager. Defined once here so they never have to be re-derived.

| Term | Definition |
|------|------------|
| **Mode** | A named, reusable methodology loop, expressed as a data folder. It owns the unit of work, the phase graph, the run strategy, and per-phase policy. `item-level`, `holistic-loop`, and `phased-plan-drain` are modes. |
| **Phase** | One node in a mode's graph — a distinct step of the loop (e.g. `investigate`, `execute`, `review`). A phase declares what it reads, what it emits, and the prompt an agent runs. |
| **Round** | One execution of one phase: a single **fresh** agent runs the phase's prompt and returns a structured result. Rounds are durable and append-only; prior rounds are carried forward as context so a later round continues correctly. |
| **Slice** | The **elastic unit of work** an execute round advances. Not a persisted type — a *contract*: an execute round advances the frontier by one comprehensively-completable unit (a whole phase or the remainder of one) and states the true frontier in its handoff, so the next fresh agent continues from the right place. This is how a too-large phase is handled without failure. |
| **Handoff** | The structured result a round produces and the next round consumes — summary, blockers, next step, changed files, tests, and the declared frontier. The handoff is *the* continuity mechanism between fresh agents. |
| **Transition / Guard** | A declared edge from one phase to another, chosen by a **guard**: a generic **field-predicate condition** over the round's declared output (field-path, operator, value). Guards can express any phase graph — a clean pass, a replan loop, a continue-draining branch, a blocked stop, a non-accepting review reloop. There is no closed, mode-specific vocabulary of branch kinds; a branch is just a predicate over declared output. |
| **Declared output schema** | The per-phase contract stating what a round is supposed to emit — field name, type, required/optional, enum, bounds. It is the single artifact that both *validates* a round's result and *steers* the resolution ladder when output is imperfect. |
| **Resolution ladder** | The four-layer mechanism (below) that turns raw, possibly-malformed agent output into the phase's declared structured result — or an honest abstain — instead of failing the phase. |
| **Example-run** | A mode-owned data fixture that seeds phase outputs and asserts the expected phase path through the *real* guards. Example-runs are how a mode is **simulated and tested** before use; they are the data behind the UI's simulation presets. |

## A mode is a data folder

Each mode is a directory interpreted by one generic engine. Adding or changing a mode is a **data edit**, not a code change.

```
scenarios/swarm-manager/modes/<id>/
  mode.json          # identity + decision metadata, scope, run strategy,
                     # phase graph (phases, transitions, guard conditions),
                     # per-phase declared output schema, prompt template refs,
                     # and profile / artifact / backlog-sync / metrics / lock / ui policy
  prompts/           # the phase prompt templates (with {{VARIABLE}} slots)
  example-runs/      # simulation fixtures: seeded outputs + expected phase paths
```

- **Schema (the vocabulary):** `.vrooli/schemas/operating-mode.schema.json` is the JSON Schema every `mode.json` and example-run validates against. It is generic — it can express arbitrary phase graphs, branching guards, and output contracts — with **no vocabulary bespoke to any named mode**.
- **Engine (the interpreter):** one generic engine under `scenarios/swarm-manager/api/internal/operatingmode/` loads mode data into the typed in-memory definition the runtime consumes, validates it (schema + semantics, returning typed actionable errors), evaluates the declared guards, and runs the phases. A **data-backed registry** discovers modes from disk. Shared framework code gains **no mode-specific behavior branches** — new methodology is new data, not new code.

This is a hard, greenfield cutover: the three modes become data, and the hardcoded mode definitions and static registry are deleted.

## The resolution ladder

The ladder is the robustness upgrade that lets modes survive real nondeterministic model output. **Data says WHAT good output is; code does HOW to obtain it.** Every phase result passes through four layers, steered by the phase's declared output schema:

- **L0 — True-final-message detection (built-in, framework-level; a mode may opt out).** Agents sometimes emit a "final" message and then a subagent appends more, so the chronologically-last message is often not the real answer. L0 scans the last *N* agent messages for the declared output shape to identify the *true* final message, falling back to a built-in classifier when none matches.
- **L1 — Deterministic extraction.** Extract the phase's declared output schema from the identified message — field name / type / required / enum / bounds — with no model call. This is the fast, cheap, exact path and unifies what used to be scattered output-contract flags into one declared schema.
- **L2 — AI-classifier fallback (steered by the declared schema).** When L1 cannot cleanly extract, route the raw output *plus the declared schema* through an LLM classification call that reconstructs the fields, and **abstain rather than guess** when it cannot. The mechanism reuses `measures-go` (`measures.Completer` / `NewLLMExtractor`) over `resource-ollama gateway generate --role classify.routing`; output is prompt-constrained and Go-validated.
- **L3 — Contract validation.** Validate the resolved result against the phase's declared output contract. A violation is a typed, honest failure — not a silently-accepted wrong value.

The old behavior — *require a clean envelope; fail the phase on anything malformed or missing, with no recovery* — is replaced. A phase with malformed or trailing-subagent output now resolves to the correct structured result or an honest abstain, driven entirely by the mode's declared output schema.

## The three modes today

Modes are chosen by **how the work is shaped**, not by how much work there is or how present the operator is. All three are async-reviewable in the same shape as the backlog-item flow; what differs is the *unit of execution and validation*.

| Mode | Unit of execution | Work shape it fits |
|------|-------------------|--------------------|
| **item-level** (default) | One backlog item per agent run | Items are right-sized, independent, reviewable in isolation, and stable through execution |
| **holistic-loop** | The whole initiative through `investigate → plan → execute → review → replan` | Items are coupled, likely to shift mid-flight, mis-scoped, or only validatable as a system |
| **phased-plan-drain** | The whole initiative, drained by a stable sequential plan and accumulated handoffs | A large multi-phase plan is prepared once and drained by agents completing the earliest contiguous slice they can fully finish |

### item-level (default)

- **Unit:** one backlog item per agent run. **Lifecycle:** `backlog → researching → ready → queued → in_progress → completed/failed`.
- **Refinement:** the workshop loop (rounds with 5-dimension readiness scoring; converges to a canonical plan-manager plan bound through the item `plan_ref`). **Execution:** an agent receives the rendered plan-manager projection, executes, hands back a result. **Review:** a review round ratifies completion.
- **Strengths:** parallelism (many items in flight), per-item provenance, bounded blast radius, per-item legibility.
- **Failure mode:** items coupled by a shared substrate produce broken intermediate states; work whose item shape shifts mid-execution thrashes the item graph; over-fragmented items waste cycles on per-item ceremony.

### holistic-loop

- **Unit:** the initiative, taken holistically. **Phases:** `investigate → plan → execute → review → reconcile`, with execute looping back to investigate when its result declares `replan_needed=true`, and review looping back to execute when its `verdict=changes_requested`.

```
investigate ──▶ plan ──▶ execute ──▶ review ──▶ reconcile
     ▲                    ▲   │          │
     │                    │   └─ replan ─┘  (replan_needed=true)
     └── replan (findings)┘   │
                          └── changes_requested ──┘  (review loops back to execute)
```

- **Refinement:** durable mode rounds over the full member-item graph, producing initiative-level artifacts (`findings.md`, `initiative-plan.md`). **Execution:** an agent works against the initiative-level plan, touching whichever items the work covers; items get marked done through audited reconciliation, not as independent execution events. **Review:** validate the initiative as a whole against its acceptance criteria — an `accept`/`accepted` verdict advances to reconcile, a `changes_requested` verdict loops back to execute to close the specific gaps, and any other non-accepting verdict records the gap and reconciles.
- **Strengths:** correct unit of validation for coupled work; lower replanning cost (one plan, not N); investigation is a first-class step instead of being smuggled into per-item workshop; items can shift scope as the work reveals what they are.
- **Failure mode:** loses parallelism; higher per-run cost; wrong for genuinely independent, stable items.

The investigation phase has no analog at item scale: at item scale the workshop *is* the investigation, scoped to one item; at initiative scale the *cross-item ground truth* (what's in code now vs. what items still describe vs. what prior handoffs assumed) drifts and only a deliberate investigation pass surfaces it.

### phased-plan-drain

- **Unit:** the initiative, drained by a stable sequential plan and accumulated handoffs. **Phases:** `prepare_plan → execute_next → classify_progress`, then either `execute_next`, `prepare_plan`, or `review` based on the classifier's progress decision.
- **Refinement:** a canonical plan-manager `plan_ref` plus round handoffs; the plan is prepared once in plan-manager and revised only when `classify_progress` decides `replan`. **Execution:** each execute round receives plan-manager phase context, completes the earliest contiguous **slice** it can safely finish, records important state in the plan-manager log, and hands off the true **frontier** for the next round. **Review:** validate the completed initiative against acceptance criteria once progress is classified complete; a `changes_requested` verdict loops back to `execute_next` for one more gap-closing slice before acceptance.
- **Strengths:** continuity across long sequential work, less planning churn than a holistic loop, explicit progress classification between slices.
- **Failure mode:** unsuitable when the plan is not stable enough to drain, or when parallel independent execution is more valuable.

### Backlog items in initiative-scoped modes: tracking, not execution

When an initiative runs in `holistic-loop` or `phased-plan-drain`, its member items don't disappear — they survive as **tracking and scope markers** (what the initiative claims to deliver, progress reporting, partial cancellation, cross-initiative dependency targets). What changes is that they are no longer *independent execution units*: their `plan_ref` points to item-level scope context, their workshop rounds are historical context, and they are marked `completed` as the initiative-level plan covers them. This is the load-bearing distinction — items remain the unit of *visibility* and *scope* while the unit of *execution* and *validation* moves up to the initiative.

## Why the distinction exists (the empirical cue)

Swarm Manager's original flow assumes **backlog items are the right unit of execution and validation** — atomic enough that one agent run picks up one item, executes it, and hands back a discrete outcome. That assumption is load-bearing: it is what lets an operator review N initiatives in parallel and gives execution bounded blast radius. It holds when items are correctly scoped, independent, and stable. It breaks when items are coupled by the very thing being changed, likely to shift mid-flight, or mis-scoped.

The **2026-04-27 sandboxing trap** surfaced this. Two initiatives (`agent-sandbox-audit-foundation`, `protected-agent-sandboxing`) both restructured how agent runs flow through the workspace sandbox. Completing them item-by-item left the system in an inconsistent intermediate state for the items still in flight — changes to sandbox routing affected every other run, including swarm-manager's own runs — so swarm-manager stopped working. The work succeeded only when the operator stepped *outside* the harness and ran: investigate current state in code → generate a consolidated plan → stage execution against it → re-investigate after each wave and revise. The work didn't fail because the items were big; it failed because **the items couldn't be validated in isolation** — only the system as a whole could be, after the coupled work reached a coherent state. Rather than treat that recovery as "an escape hatch when swarm-manager fails," we recognize it as a first-class operating mode. The operator's continuous presence during recovery was an artifact of having no in-harness support, not a requirement of the work — a structured loop running *inside* swarm-manager absorbs the same work at the operator's normal async cadence.

## Choosing and switching modes

The right mode is a property of how the work is shaped, not its size. A 10-item initiative of independent SKU-level changes is appropriately `item-level`; a 3-item initiative whose items all touch the same auth middleware is appropriately initiative-scoped.

- **Use `item-level`** when items are right-sized for one run, loosely coupled, reviewable in isolation, stable, and parallelism is valuable.
- **Use an initiative-scoped mode** when items are coupled by a shared substrate, intermediate states leave the system inconsistent, item shape will shift mid-execution, items are mis-scoped, the natural unit of validation is "does the system as a whole work," the right plan can only be authored after investigating cross-item ground truth, or replanning is expected.

A mode is a property of an initiative *at a point in time*, not immutable. An initiative may begin `item-level`, hit a shape mismatch, and be promoted to an initiative-scoped mode; after an initiative-level plan converges, residual independent work may be drained back to `item-level` for parallel finish-out. The switch is a first-class, operator-chosen lifecycle action — generic initiative create/update does not mutate mode; switching into a non-default mode handles active item-level executions explicitly, and switching out is blocked while a mode round is active.

## Understanding a mode: the Flow tab

Every mode is inspectable from the UI without reading source or raw JSON first. The operating-mode detail page has a **Flow** tab that renders **one shared phase viewer** with four concern tabs — **Instructions / Reads / Emits / Transition** — and a **data-source control** that swaps the fill without changing the view. The same viewer backs the **Phases** tab, so a phase is understood through one surface everywhere. Three sources:

- **Contract** — the phase's static contract: the agent prompt template with `{{VARIABLE}}` slots still unfilled, and what the phase is *defined* to read, emit, and route to. No initiative data substituted.
- **Simulation preset** — deterministic, in-memory walks of the phase graph against illustrative data (from the mode's **example-runs**). Presets seed phase outputs to exercise real transition branches (clean pass, replan, continue, blocked, non-accepting review, backlog reconcile) but never spawn agents, acquire locks, or persist state.
- **Live round** — the actual rounds recorded for a linked initiative.

For Simulation and Live, the prompt is rendered server-side through the *same* prompt-render seam the real spawn path uses, and a parity test asserts the preview is byte-identical to what an agent receives. When that seam is unavailable the tab degrades to the resolved variable map rather than erroring. Because presets, prompts, and transitions are sourced from the mode's data, adding a mode or a branch surfaces in Flow with no mode-specific UI. A top-level "what is an operating mode" entry point (this concept, surfaced in-app) means an operator who has never seen operating modes can understand what one is before drilling into any specific mode.

## Authoring a mode

Authoring is self-serve and writes **data**: scaffold a mode folder from a template, validate it against the schema and semantics, simulate it against its example-runs, and run it — with **zero Go edits and no redeploy**. The `operating_mode_authoring` session drives this flow. See [DOC: ../internal/OPERATING-MODE-AUTHORING.md] for the authoring procedure.

## References

- [`docs/concepts/ARCHITECTURE.md`](./ARCHITECTURE.md) — the staging-and-review framing operating modes build on.
- [`docs/guides/workshop-workflow.md`](../guides/workshop-workflow.md) — the item-level workshop loop, readiness model, and `plan_ref` handoff.
- [`docs/guides/holistic-loop-mode.md`](../guides/holistic-loop-mode.md) — operator workflow for holistic-loop mode.
- [`docs/guides/phased-plan-drain-mode.md`](../guides/phased-plan-drain-mode.md) — operator workflow for phased-plan-drain mode.
- [DOC: ../internal/OPERATING-MODE-AUTHORING.md] — how to author a mode as data.
- `.vrooli/schemas/operating-mode.schema.json` — the generic mode schema (the vocabulary).
- `scenarios/swarm-manager/modes/<id>/` — the per-mode data folders (`mode.json`, `prompts/`, `example-runs/`).
- [`docs/concepts/GLOSSARY.md`](./GLOSSARY.md) — scenario term definitions.

## Changelog

- **2026-07-08** — Closed the two behavior gaps the spec had declared but the engine omitted: the **elastic-slice contract** is now encoded in the execute-phase prompt (a shared `ELASTIC_SLICE_SNIPPET`) and the handoff schema (a `frontier` field on `handoff`, carried through Proto + Connect), and the **review reloop** is now real data — a `changes_requested` verdict routes review back to execute/`execute_next` (with the reconcile auto-start gated on the guard actually routing there), covered by mode-owned example-runs.
- **2026-07-08** — Rewritten as the canonical Northstar concept and the spec for the data-driven rebuild: operating modes are reusable, inspectable, testable methodology loops; a mode is a data folder interpreted by one generic engine; added the single-source vocabulary, the resolution ladder (L0–L3), the elastic-slice contract, and example-runs-as-data. Reframed the architecture from a hardcoded Go registry to the data SSOT target.
- **2026-05-01** — Documented the registry-owned authoring architecture (superseded by the data-driven model above).
- **2026-04-30** — Added mode-aware prompt routing, stable phase activity purposes, typed event/stat contracts, and the first Modes stats surface.
- **2026-04-28** — Initial document, authored after the 2026-04-27 sandboxing trap surfaced "items are the right unit of execution and validation" as a load-bearing assumption that doesn't always hold.
