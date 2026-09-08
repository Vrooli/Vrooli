<skills count="10">
  <skill id="improve-skill-authoring" name="improve-skill-authoring"><![CDATA[
## Meta focus: Improve Skill Authoring

Author the improve role for one scenario as a control loop that regulates the scenario against a setpoint it does not own, reads sensors it does not decide with, and leaves evidence a later cycle can compare. The skill this produces is read by an agent whose task is the scenario itself, under `goal-loop` or a heartbeat; it is never loaded for ordinary use of the scenario.

Required reading:
- `path:docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets" and §"Universal quality bars".
- `path:docs/agent-system/TARGET_MODEL.md` §2 — the control chain this skill instantiates at scenario scale. Sensor implies no authority.
- `path:docs/agent-system/FRAMEWORK_HEALTH.md` §"Deadband rule" — a deadband states the target, never the current reading.
- `prompt-manager skill read improvement-do-and-dont` — the anti-gaming patterns D1, D2, D3 the authored skill cites by id.
- `path:scenarios/program-runtime/docs/guides/program-contracts.md` — the standard the rendered `setpoint-read` program follows; its §"The envelope" defines the row shape, the closed `reason` vocabulary, and which reasons lower the program's status.

Inputs from `skill-set-authoring`: the sensor inventory (sensor, command, `measured` or `pending_telemetry`), the golden corpora with floors, the open problems ledger, the dependents count, the program inventory.

### 1. Scope

**In scope:** the improve skill's eight sections; the setpoint table and its rendering into `scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.{py,json}`; the routes from a sensor reading to a curation move or a work-ladder rung.

**Out of scope:** defining sensors (measures live in the manifest, corpora in `evals/`); code work (the authored skill hands off to `scenario-work-ladder`); the usage role.

### 2. The eight sections

The authored skill has exactly these sections in this order. Each has a source the author reads and a rule the author applies.

| # | Section | Source | Rule |
|---|---|---|---|
| 1 | Focus and scope | the scenario's PRD one-liner; the dependents count | Name the plant (what is regulated) and what this skill never touches |
| 2 | Setpoint | PRD `OT-*` targets; the space doc if the scenario owns a projection; the sensor inventory | One row per goal: `row`, `sensor` (a command), `band` (a target), `today` (dated reading, or the row's `reason` when unavailable). A row with no measured sensor is kept with reason `pending_telemetry`; it is not turned into prose |
| 3 | Sensors | the inventory | The exact commands, plus the two fleet sensors every scenario has: `program-runtime bindings condition --scenario <scenario>` for its bindings and the `agent-manager.friction-digest` program (inputs `scenario`, `window_days`) for recurring friction on its commands. External sensors outrank self-reported ones |
| 4 | Golden corpora | `evals/*.json` | Cite the suite and its floor. The floor is re-derived from comparable runs and recorded with its derivation; a run below floor is a stop for every other route |
| 5 | Routes | the problems ledger; the sensor rows | A work table from `row out of band` to one of: curation move (data only), `scenario-work-ladder` rung (W0 to W3), file against another owner. Each route names the sensor that should move next cycle |
| 6 | Anti-gaming | `improvement-do-and-dont` §1 | Cite D1 (loosened or deleted tagged test), D2 (deleted known-issue ledger), D3 (suppressed finding) by id, and §2 (the skeptic test); add the scenario's own gaming moves (a waiver on a domain the sensor needs, editing a requirements registry to match a claim, widening a floor, deleting a failing corpus case) |
| 7 | Evidence | canon memory loop | One `vrooli-memory journal note --kind work-record` per cycle with the before and after reading of the row moved, on the same sensor; a `PROBLEMS.md` entry when a row is `scenario_unreachable` for three consecutive cycles; a `report-bug` filing when the fix belongs to another owner |
| 8 | Stop rules | this skill | One line per stop, in this order. Row `reason` decides the first four (vocabulary: `program-contracts.md` §"The envelope"): `no_governed_binding` or `pending_telemetry` → route on the first read (W1, or the filing recipe in `skill-set-authoring` Phase 2), never wait; `scenario_unreachable` → record, do not estimate, wait one cycle; `read_elsewhere:<program>` → run that program; `unreliable:<why>` or `kernel_invoke_budget` → report the reason, do not band. Then: floor below comparable count → refuse to lower it. Route needs a grant → request through the session path. Two cycles in band → propose close-out, do not close. Budget: one line stating the wall-clock budget per cycle as a duration this author sets; `goal-loop` passes it as `program-runtime sessions create --wall-budget <duration>` (the flag is optional; omitted, the runtime sets 4 h, reported as `wall_budget_millis`) and, when the contract's `budget.async` is true, as `programs submit --async --wait-timeout <duration>`; a synchronous submit is bound at 120 s by the runtime. It is a stop, never a target |

### 3. Writing the setpoint

A setpoint row is a sentence an agent can check:

```
| discovery-floor | program-runtime discovery eval --suite evals/discovery.primary.json | met >= floor | 41/45 (floor 43) 2026-09-02 |
```

Rules:
- **Band states the target.** Never write the current reading as the band. If no target exists, write the direction (`rising`, `falling`) and the reading, and mark the row `pending-baseline`; the rendered program emits `target: null` for it.
- **One sensor per row.** A row that needs two commands is two rows.
- **Readings are dated.** They are observations, not state; the next cycle re-reads them.
- **The recursive row** belongs only to scenarios whose improve loop installs capability elsewhere (program-runtime, prompt-manager): "share of <targets> with a conformant skill set". No sensor computes conformance yet, so the row is `pending_telemetry` with `target: null`. Its interim read is `prompt-manager.skill-set-read` (contract under `scenarios/prompt-manager/.vrooli/program-runtime/`), one target per run: it reports the ids registered under the scenario pack, whether `<scenario>` and `<scenario>-improve` are registered, and the set's token size; it cannot report waiver grading, `programs[]` resolution, sensor reality, or frontmatter dialect, and its `read-counts` row is `unreliable:proto_drift_skill_usage` until the skill-usage binding is repaired. The row's actuator is filing `skill-set-authoring` runs against owners, never editing another scenario.

### 4. Rendering `setpoint-read`

The setpoint table is also a program. Render `scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.py` and `.json` from it:

1. `collect`: one `gather` over every row whose sensor is a governed binding. Every other row is emitted with `unavailable: true` and a `reason` from the closed vocabulary in `program-contracts.md` §"The envelope": `no_governed_binding` for a CLI command with no binding, `pending_telemetry` for a row with no sensor, `read_elsewhere:<program>` for a row another program owns, `kernel_invoke_budget` for a binding that outruns the invoke budget. A binding that raises is classified by the verbatim `classify_transport` from that guide.
2. `classify`: each row → `in_band` by the band rule, computed in the kernel from the handle (`count`, `head`, `group_by`); no materialization. A sensor whose own validity gate failed is `unreliable:<why>` with `reading` kept and `in_band: null`.
3. `report`: the envelope with `signals.rows` as the list of `{row, reading, target, in_band, unavailable, reason}`. `status` is `ok` when every read that was attempted returned; a permanent reason (`no_governed_binding`, `kernel_invoke_budget`, `read_elsewhere`, `pending_telemetry`) does not lower it; only a `scenario_unreachable` row or a failed read makes it `partial`. A `no_governed_binding` row is routed W1 by `goal-loop` on the first read; the program never waits on it, and the skill's stop rules say so.

The contract declares every binding with effect `read`, `budget.inference_calls: 0`, `budget.delegated_runs: 0`, `status.enum` limited to the statuses the program can reach, `errors.classes` as the minimum list in `program-contracts.md` plus domain classes, and one live fixture expecting `["ok"]`: a board whose unavailable rows are all permanent is `ok`, so the fixture accepts `partial` only when it names the row that is transiently unavailable in its `note`. Validate with `program-runtime sessions create --name <scenario>-setpoint --json` (`--json` because the id is read from `.session.id`), then `program-runtime programs submit --session-id <id> --source-file scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.py --provenance operator --explain --json` (`--json` because the diagnostics list is read from the response), then `program-runtime sessions delete <id> --reason "explain done"` before finishing.

### 5. Routes

A route table row:

```
| agent-failure-rate above band with kernel_runtime naming a forbidden import | ladder W3: kernel guard for the name, then retire the prose | agent-failure-rate |
```

Choose the cheapest route that moves the sensor:

| If the fix is | Route | Who does it |
|---|---|---|
| A data or policy change the scenario already exposes (promote, supersede, gc, set-current, a config knob) | curation move, done in-cycle | the agent running the improve skill |
| A missing sensor (`pending_telemetry`) | backlog item by the filing recipe in `skill-set-authoring` Phase 2 | filed |
| A missing binding (`no_governed_binding`) | `scenario-work-ladder` rung W1 | filed as a backlog item on the goal's milestone |
| A contract, obligation, evidence, or implementation defect in this scenario | `scenario-work-ladder` rung named | filed as a backlog item on the goal's milestone |
| A defect in another scenario | `report-bug` against that owner | filed |

### 6. Convergence patterns

Two agents authoring the improve skill for the same scenario from the same inventory must produce the same setpoint rows, the same `reason` marks, and the same routes. The sources that decide it: the PRD's `OT-*` list, the inventory's `measured` column, the corpora's floors, the reason vocabulary in `program-contracts.md`, and the route table in §5.

### 7. Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| A band equal to the last reading | Reads in band while the defect stands; can only detect growth | State the target; if unknown, `pending-baseline` |
| A sensor the skill computes by hand from files | The skill becomes the instrument and can be gamed by editing | Cite a measure, corpus, binding, or digest; file for what is missing |
| Retry logic in the routes | Hides the failure class the next cycle needs | One move per row per cycle; re-read next cycle |
| A route that edits another scenario | The instrument becomes a controller (`TARGET_MODEL.md` §2 "Sensor implies no authority") | File against the owner |
| Prose goals with no sensor | The loop regulates nothing and reports progress anyway | `pending_telemetry` rows and filed backlog items |
| A program status of `partial` because a row has no binding | The caller cannot tell a healthy board from a degraded one | Permanent reasons keep `ok`; only transient rows lower status |

### 8. Output expectations

You may: create `scenarios/<scenario>/skills/<scenario>-improve/SKILL.md`; render `setpoint-read.py` and `setpoint-read.json`; file backlog items and `report-bug` items.

You must: keep every sensor cited real; date every reading; cite anti-gaming by id (D1, D2, D3); emit only the closed `reason` vocabulary; make the setpoint table and the program agree (same rows, same bands); pass `skill-validation` §3.3 (divergence probe) on the routes table.

You must not: write a measure into a manifest; edit `PROBLEMS.md` to close an entry; author a row whose only sensor is an adjective.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every setpoint row is `pending_telemetry` | The scenario has no measures | `measures-health validate scenario <scenario>` | Author the skill anyway with the rows marked; its first route is the filing recipe; the skill is short and honest |
| A corpus has no floor field | Floors were never derived | `jq .floor evals/*.json` | Route: derive from at least two comparable runs and record the derivation; until then the row is `pending-baseline` |
| `setpoint-read` cannot call a sensor because it is a local CLI command | The command has `binding.kind: local` | `program-runtime bindings unbound` | The row carries `unavailable: true`, reason `no_governed_binding`; route W1 on the first read |
| The scenario owns a projection but `space --projection` fails | Space contract unimplemented | `<scenario> space --json` | Coverage row is `pending_telemetry`; route to W1 |
| Routes table has two rows for one reading | A C4 defect the divergence probe will catch | Run the probe | Merge or add a discriminating predicate |

]]></skill>
  <skill id="implementation-plan-authoring" name="implementation-plan-authoring"><![CDATA[
## Practice focus: Implementation Plan Authoring

Author durable implementation plans through Plan Manager's guided authoring
runtime. Plan Manager is the plan-logic authority: it owns the section order
(the ten reader-question clusters), phase shape, validation gates, context
discovery execution, rendered markdown, and persistence. This skill supplies
only the judgment layer — when to plan, and how to decide well inside the
wizard.

Before the first operation, read `prompt-manager skill read plan-manager` and
use its recall and per-attempt learning capture. Do not duplicate the record.

Required reading:
- `scenarios/plan-manager/docs/concepts/PLAN-MODEL.md` — canonical structured
  plan and phase model.
- `scenarios/plan-manager/docs/reference/cli-commands.md` — current
  `plan-manager author`, `plans`, `exec`, `validate`, and `log` command surface.

Companion skill: `prompt-manager skill read implementation-plan-execution` owns
what happens when a finalized plan turns out to be wrong during execution. Read
it before authoring the Boundaries and stakes content below — knowing which
divergences execution is expected to make on its own tells you what the plan
must state and what it can leave to judgment.

---

### 0.0 Before authoring: read prior-plan status correctly

You will find related plans while checking for duplicate work. Plan status is
**computed from the phase-status set**, not from anyone's intent:

- `draft` = no phase has left `todo`. A finalized, validated, never-started plan
  reports `draft` forever. It does **not** mean the plan is half-written.
- `active` = at least one phase left `todo`. A run abandoned mid-phase reports
  `active` indefinitely. It does **not** mean anyone is working on it now.

Neither value carries recency. Judge ownership by **last activity**, which
`plans list` and `plans get` print beside the status.

Do not report an existing plan as a reason not to author. A stale plan covering
the subject is input to supersede, extend, or reuse — say explicitly which parts
you took and which were stale.

---

### 0. Preserve the planning source, not just the task title

A plan is a durable compression of the investigation and discussion that led
to it. Compress repetition and incidental conversation; do **not** compress
away a decision, constraint, visual model, or acceptance expectation that a
fresh execution agent would need in order to make the same implementation
choices.

When a plan coins or narrows a term, fill the optional **Definitions** section
with one `Term — meaning` line per term. Do not duplicate shared ecosystem
vocabulary there; reference `docs/concepts/GLOSSARY.md` instead.

Before authoring, build a private source inventory from the operator request,
conversation, attachments, workshop decisions, research, code inspection, and
durable references available to you. Identify every material item in these
classes:

- intended outcome, user experience, and non-goals;
- current behavior, problem evidence, and relevant constraints;
- selected design, rejected alternatives, and the rationale/tradeoffs;
- diagrams, flows, examples, or interface contracts that explain the design;
- invariants, change boundaries, dependencies, assumptions, risks, and open
  questions;
- discovered implementation facts, code/doc/requirement references, and
  validation evidence or expectations.

Do not copy a transcript into a plan. Instead, place each material item in its
proper durable Plan Manager location:

| Material | Durable destination |
| --- | --- |
| Outcome, user value | Purpose and Outcome |
| What breaks or stays blocked if this is not delivered | Purpose (see §0.1) |
| Current behavior and evidence | Problem |
| Chosen design, alternatives, rationale, diagrams, flows | Approach & Decisions |
| Allowed/forbidden areas and non-goals | Boundaries |
| Assumptions, dependencies, risks, unresolved questions | Assumptions & Risks |
| Skills, docs, code, requirements, commands, research | Relevant Context and references |
| How correctness is demonstrated | Verification and phase validation |
| Ordered implementation work | Phases and phase acceptance |

When a diagram, example, or flow materially explains the intended system,
preserve it in the relevant decision content or as a durable referenced
artifact. Never replace it with “see the prior discussion.”

When a multi-step flow involves three or more actors, components, or
sequential states, author a Mermaid diagram for it (a fenced ```mermaid
block inside Approach & Decisions content) instead of an inline prose arrow
chain ("A -> B -> C"). The fence is the durable format even where a given
viewer renders it as source text; do not downgrade to prose because of the
viewer.

#### 0.1 Record the consequence of failure

Every plan's Purpose must state, in one sentence, **what breaks or stays blocked
if this plan is not delivered, or is delivered wrong**. Write the consequence,
not a priority level: "P1" tells an executing agent nothing, and a number
invites a debate about the number.

Good: "Without this, the desktop build ships without signed receipts, so the
first paid scenario cannot be monetized."
Bad: "This is high priority." / "This is important for quality."

This sentence is not decoration. An executing agent hitting a defect that blocks
a phase uses it to decide whether to fix the defect properly or work around it
and file it (`implementation-plan-execution` §5). With no consequence recorded,
that agent defaults to the conservative posture and works around things you may
have wanted fixed.

Do not inflate it. A plan whose honest consequence is "a rough edge stays rough"
should say that; overstating stakes buys detours you did not want.

There are two valid authoring modes:

- **Plan Manager mode (default):** create, validate, and finalize the
  authoritative Plan Manager plan through the commands below.
- **Candidate mode:** a calling workflow may explicitly forbid external Plan
  Manager writes and request a Plan-Manager-compatible candidate markdown
  artifact. Use the same source inventory, placement map, and quality bar, but
  return the candidate to the caller instead of running `author start`,
  `validate`, or `finalize`. Do not claim that a candidate is valid or has a
  plan reference.

---

### 1. When To Use This Skill

| Situation | Use this skill? | Why |
|---|---|---|
| Context window is tight and work must continue later | Yes | Captures executable state in Plan Manager |
| User asked for an implementation plan | Yes | Produces a structured plan and rendered review artifact |
| Quick one-step fix with no follow-up risk | No | Plan overhead is unnecessary |
| Pure brainstorming with no execution intent | No | Use discussion or idea-workshop flow first |

---

### 2. Entry Point

```bash
plan-manager --auto-start author start --title "<plan title>"
```

Global flags such as `--auto-start` go before the subcommand.

`plan-manager author --help` lists every subcommand. `plan-manager author
<subcommand> --help` lists its flags, choices, and synonyms. Read the command
surface there, not here — the CLI stays accurate through change and a copied
signature does not. This section carries only what `--help` cannot.

**The session is a form, not a stage-gated wizard.** Every response carries a
full-disclosure checklist: all requirements for the touched scope, with
filled/missing/violation status. Read the checklist and submit any or all fields
**in any order**, batched when you already know the content. `author submit`
batches sections. `author phase-add` adds and fills a phase in one call.
`author phase-submit` batches fields on an existing phase.

Each batch item returns an accepted/rejected line naming exactly what was
parsed. A rejected item — unknown field, or acceptance duplicating validation —
is NOT applied while the rest of the batch lands. A complete N-phase plan takes
≤ 3+N mutation calls. The `author continue` loop remains available when you
prefer one recommended action at a time. The artifact renders in the fixed
cluster order (Purpose / Problem / Outcome / Approach & Decisions / Boundaries /
Assumptions & Risks / Verification / Execution Setup / Phases) regardless of
submission order.

**Skill discovery is a low-friction bootstrap, not a curation workflow.** Run
`author skill-pack` with 2-5 decomposed concepts. It runs `prompt-manager
discover --type skill --json`, auto-adds the returned skills as global relevant
context, and prints the read command. Keep most returned skills unless they are
clearly irrelevant; they are there to improve professional execution, not only
to match the task title literally. `--phase` adds the pack to one phase instead
of the plan; an unknown phase reference fails the call and never falls back to
global scope.

**Run search-hub directly** for docs, records, and code references, so you
inspect confidence and attribution yourself:

```bash
search-hub query "<intent>" --type record,doc,skill
```

Submit only durable context or references that will help a resumed agent:
`author context-submit` for setup commands and notes, normal reference fields
for `[CODE:]`, `[DOC:]`, `[REQ:]`, or an honest `NO_CODE_REFS: <reason>` when
there are no useful code references. There is no candidate accept/reject/apply
step.

Then `author preview`, `author validate`, `author finalize`. `author status` is
an alias of `author preview`.

Report back: plan id/slug, `plan-manager plans render <slug>`, any degraded
notes, and the first execution command (`plan-manager exec continue <slug>`).

---

### 3. Writing Standard (ASD-STE100)

Write all procedural plan content in ASD-STE100 Simplified Technical English:
phase steps, validation, acceptance, boundaries, and the Definition of Done.
The core rules:

- One instruction per sentence. Keep procedural sentences to 20 words or fewer.
- Use the active voice and the imperative mood ("Run the migration", not
  "The migration should be run").
- Use one meaning per word, and the same word for the same thing everywhere
  in the plan. Define any term with a non-obvious meaning where it first
  appears.
- Name concrete objects and commands, not categories ("edit
  `resolver.go`", not "update the relevant files").

Banned words in procedural content — each hides an undecided decision.
Replace the word with the specific behavior, path, or check you mean.
The canonical word list lives in `docs/agent-system/SKILL_AUTHORING.md`
§"Universal quality bars" — cite it, do not copy it.

STE-100 applies to procedures only. Rationale content — why this design,
rejected alternatives, tradeoffs, risks — stays in normal explanatory prose;
do not strip nuance from it to satisfy the style rules.

---

### 4. Judgment Rules

- A plan should be executable without this chat history.
- Before finalization, perform a preservation audit: a fresh execution agent
  must be able to make the same material design decisions, respect the same
  constraints, and validate the intended outcome without the source
  conversation. Add the missing rationale, visual, reference, boundary,
  acceptance expectation, or open question before finalizing.
- Reject the lossy plans in §4.1 before finalizing.
- The change boundary must name the paths the work may touch; do not hide scope
  in prose.
- **Author `acceptance_allow` as the full reach of the change, not the folder
  the work is "about".** Trace the change outward before writing the globs: a
  change to an API shape reaches the proto that defines it; a change to a shared
  behavior reaches `packages/**`; a change to a documented contract reaches
  `docs/**`. Include those paths. A boundary listing only
  `scenarios/<name>/**` when the change alters a wire contract is a defect in
  the plan, and execution pays for it — either as a workaround inside the narrow
  boundary or as a mid-run `exec boundary-extend`.
- **`acceptance_allow` is an estimate; `acceptance_deny` is a prohibition.**
  Execution may widen allow when a phase's intent needs it. Execution may never
  overrule deny. Put a path in deny only when you mean "not even if it would
  make the change cleaner" — deny is not a way to express "probably not needed",
  and every deny glob you add is a path execution must escalate to reach.
- Record the consequence of failure in Purpose (§0.1). It is the input an
  executing agent uses to size its response to friction.
- `validation` is the method of checking; `acceptance` is the outcome gate.
  They must not be identical.
- **Phase validation scope is proportionate by default.** A phase with affected
  areas derives a narrow scope from those areas and their scenario code
  references when it has no `validation_scope` declaration. Use
  `validation_scope: narrow:` to provide an explicit boundary when the derived
  scope is incomplete. Use `validation_scope: full_plan: <rationale>` only when
  the phase needs the whole plan boundary for a stated reason. The final
  Definition of Done remains selector-free and validates the whole captured
  collection, so narrowing removes duplicate work and does not remove coverage.
- Every fact lives in exactly one section: Purpose is an abstract (do not
  restate Problem or Outcome there); the Definition of Done carries plan-level
  gates only, never restated phase acceptances.
- Use `author skill-pack` early, then read the compact skill command it returns.
  Remove a discovered skill only when it is clearly irrelevant or harmful.
- Scope a skill to a phase when an agent who read the global pack would still
  work that phase wrongly. That happens when the phase enters a governed surface
  with its own maturity ladder, uses a different working method than the rest of
  the plan, or produces an artifact with its own authoring standard. When the
  global pack already covers the phase, add nothing — a skill list every phase
  carries and no phase reads is attention debt.
- Run search-hub directly when docs/records/code context would help, and submit
  only durable context. Do not recreate a candidate queue by hand.
- Use explicit fallback markers only when honest: `NO_CODE_REFS: <reason>`,
  `NO_SKILL_CONTEXT: <reason>`, `NO_CONTEXT: <reason>`.
- Greenfield/Brownfield posture is derived by Plan Manager. Do not hand-author
  contradictory compatibility language.
- Do not fabricate baseline success. Plan Manager records regression-anchor
  intent during authoring; execution/validation captures and checks the fresh
  baseline.
- Never hand-edit rendered Plan Manager markdown mirrors; re-render through the
  tool (`plan-manager plans render` / `plans reconcile`).
- For out-of-scope defects found while authoring, use Plan Manager log entries
  during execution, or load `prompt-manager skill read report-bug` if the defect
  needs filing before execution begins.

#### 4.1 Anti-patterns — the lossy plan

Each row is a plan that passes validation and still fails the fresh execution
agent. Check the plan against every row before `author finalize`.

| Anti-pattern | Why it fails | Better approach |
|---|---|---|
| Says what to do, not why this design | The executor hits an unforeseen case and has no principle to decide from, so it invents one | Record the chosen design and the rejected alternative in Approach & Decisions |
| Names a component, omits the interaction | The executor knows where to write and not what the data does across the boundary | Add the interaction or data flow; author a Mermaid diagram at 3+ actors or states |
| Lists phases without exact acceptance or validation | "Done" becomes the executor's opinion, so the phase closes on a plausible-looking edit | Give every phase a runnable validation and a distinct outcome gate |
| Records a decision without its tradeoff | A later agent relitigates the decision because the cost of the alternative is invisible | Record the tradeoff and the revisit trigger with the decision |
| Assumes context that exists only in chat | The executor never had the conversation and cannot recover the missing premise | Move the premise into the matching durable section before finalizing |
| Leaves discovered facts behind "investigate as needed" | The executor repeats investigation already paid for, and may reach a different answer | Write the discovered fact and its evidence into the phase |
| Bounds the change to one scenario when it alters a shared contract | The executor needs the proto/shared type the API shape depends on, and either writes an adapter to avoid touching it or stops | Trace the change outward and list every reached path — proto, `packages/**`, `docs/**` — in `acceptance_allow` |
| States importance as a level ("P1", "high priority") instead of a consequence | The executor cannot tell what a detour would protect, so it treats every defect the same way | Write what breaks or stays blocked if the plan fails (§0.1) |
| Uses `acceptance_deny` to express "probably out of scope" | Deny is a hard prohibition execution must escalate to cross, so a soft guess becomes a stop | Leave it out of both lists; an unlisted path is already outside the estimate and extendable |
| Demands the whole baseline collection in every phase without a rationale | Each phase repeats the final gate and validation cost grows with no added coverage | Leave the field undeclared for the affected-area default, or use `full_plan:` with a concrete reason |

---

### 5. Troubleshooting & Edge Cases

| Symptom | Likely cause | First move |
|---|---|---|
| `plan-manager` is unavailable | Scenario is stopped or not installed | Run `plan-manager --auto-start status`, or `vrooli scenario start plan-manager` |
| `author continue` repeats a gate | A required section, reference marker, phase field, or validation distinction is missing | Read `remaining_required_inputs` / human output and submit the requested decision |
| `author skill-pack` degrades | prompt-manager is down or returns no skill pack | Continue authoring with a warning, or record an honest `NO_SKILL_CONTEXT:` only when no useful skill setup exists |
| Reference discovery is needed | Plan Manager no longer runs search-hub for you | Run `search-hub query "<intent>" --type record,doc,skill`, then manually submit `[CODE:]`, `[DOC:]`, `[REQ:]`, context, or honest `NO_CODE_REFS:` |
| Anchor autofill is degraded | Boundary missing or validation dependency unavailable | Submit/repair change boundary, rerun autofill, or record degraded intent only when Plan Manager permits it |
| Need to preserve an existing markdown plan | Legacy import/adoption path | Use `plan-manager plans import --source <path> --workspace <repo-root>` |

---

### **6. Output Expectations**

**Must produce:**
- A finalized Plan Manager plan id/slug, or a clear explanation of why the
  authoring session remains unfinished. Finalize output names the physical
  SQLite store path, the stamped workspace, and a **computed** mirror status
  (`fresh`, or a loud `write_failed` warning with the repair command) —
  treat a `write_failed` mirror or a missing store path as something to
  report, and re-running finalize prints `Already finalized at <ts>`
- A rendered review command/path
- A concise note about degraded dependencies or manual fallbacks
- The next execution command when implementation should continue, paired with
  `prompt-manager skill read implementation-plan-execution` so the executing
  agent starts with the divergence rules rather than inferring them

In Candidate mode, ensure the candidate itself records the material source
context in the appropriate plan sections. Include a concise preservation note
only when the caller's typed result contract has a field for it; do not claim
finalization, validation, a plan id, or a rendered Plan Manager path.

**Must not produce:**
- A standalone hand-formatted markdown plan as the default artifact
- Placeholder-only phases or context entries
- Contradictory constraints or fabricated validation evidence

]]></skill>
  <skill id="skill-authoring-platform" name="Platform Skill Authoring"><![CDATA[
## Meta focus: Platform Skill Authoring

Guide for creating **platform** skills (the authored skill declares `modes[0] = "platform"`). Platform skills steer safe evolution of shared code (for example `path:packages/*`, shared templates, shared contracts) that is consumed by many scenarios.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `docs/agent-system/PROMOTION_LADDER.md`

Optional reading:
- `prompt-manager skill read cli-steer`
- `prompt-manager skill read conversation-friction-analysis`

---

### 1. Category Scope

**In scope:**
- Shared package evolution (`path:packages/*`) and other shared platform code used by multiple scenarios
- Cross-scenario standardization (CLI output contracts, config precedence, error taxonomy, codegen contracts)
- Brownfield-safe improvement paths (additive changes, deprecation cycles, compatibility envelopes)
- Verification discipline proportional to blast radius (tests + downstream smoke/compat checks)

**Out of scope:**
- Scenario-specific feature design or refactors (use Steer skills; Steer must target `scenarios/{{TARGET}}/`)
- One-off operational runbooks (use Tools skills)
- Skill system governance (use Meta skills)

---

### 2. Required Placeholders and Targeting

Platform skills must not use `{{TARGET}}` (reserved for scenario-focused Steer skills).

Use:
- `{{PACKAGE}}` for a shared package identifier (example: `cli-core`, `api-core`, `proto`)
- `{{PACKAGE_PATH}}` when a path needs to be explicit (example: `packages/{{PACKAGE}}/`)

Optional:
- `{{AREA}}` for subdomains (example: `cli`, `api`, `ui`, `proto`, `runtime`)

---

### 3. Recommended Structure (Keep It Small)

Structure follows `docs/agent-system/SKILL_AUTHORING.md` §"Skill structure" — do not restate it. The platform-specific additions are a **compatibility envelope** (template in §7) and **work tables** for change classification and layering (§4). Platform skills are enforcement-heavy, not prose-heavy — avoid turning them into “everything about the package”; prefer linking to package docs.

---

### 4. Compatibility-First Convergence Patterns

#### 4.1 Change classification table

| Proposed change | Default decision | Requires explicit opt-in? | Notes |
|---|---|---|---|
| Bugfix (no contract change) | Ship | No | Add regression test |
| Additive API/CLI capability | Ship | No | Prefer new flags/fields/commands |
| Behavior change (same inputs, different outputs) | Avoid | Yes | Requires migration plan + downstream tests |
| Breaking change (removes/renames) | Avoid | Yes | Deprecate first; document timeline |
| New dependency/toolchain | Avoid | Yes | Must justify; avoid forcing scenarios |

#### 4.2 Layering decision tree (where to fix)

```
Is this friction repeated across multiple scenarios?
  -> YES: Prefer package/tool output contract improvement
  -> NO: Prefer scenario-local fix or documentation

Does the skill propose a prose workaround for a missing tool capability?
  -> YES: Add minimal interim guardrail, but file a promotion candidate to package/tooling
```

---

### 5. Verification & Testing Bars (Blast Radius Aware)

Platform skills must require verification proportional to impact:
- Unit tests inside `{{PACKAGE_PATH}}` for logic changes
- Contract tests for user-facing output/behavior (CLI output, JSON schema, config precedence)
- A downstream “compat set”: 1-3 representative scenarios or integration tests that exercise the changed seam

Rule:
- If you can’t name how to verify a change, you don’t have a safe platform change plan yet.

---

### 6. Output Contract Standards (Human-First)

When Platform skills touch CLI behavior (directly or via shared libraries), apply the human-first CLI bar in `docs/agent-system/SKILL_AUTHORING.md` §"Universal quality bars" and the output-contract standards in `cli-steer`; do not restate them. The platform-specific delta: blocking (`--wait`) flows must be progress-observable and return reliable exit codes, because shared libraries set the contract for every consumer CLI.

---

### 7. Compatibility Envelope Template (Copy-Paste)

```markdown
### Compatibility Envelope

- Package: `packages/{{PACKAGE}}/`
- Consumers (known): [list scenarios/packages]
- Supported environments: [Go/Node versions, OS constraints]
- Stability promise: [no breaking changes unless explicitly requested]
- Compat set (must pass): [commands/tests]
```

---

### 8. Output Expectations

You may update:
- Platform skills to improve clarity, compatibility guidance, or verification coverage
- `skill.json` entries for Platform skills

You must:
- Use `{{PACKAGE}}`/`{{PACKAGE_PATH}}` (and optional `{{AREA}}`) — never `{{TARGET}}`, which is reserved for scenario-focused Steer skills
- State the compatibility envelope (template in §7) and the compat-set verification commands that prove a change is safe for downstream consumers
- Prefer human-first CLI output patterns and avoid parser-dependent workflows by default

Registration follows `docs/agent-system/SKILL_AUTHORING.md` §"Registration and metadata"; the authored skill declares `modes[0] = "platform"` and a description that names the shared package surface and the safety posture (compatibility-first, brownfield-safe).


]]></skill>
  <skill id="documentation-health" name="Documentation Health"><![CDATA[
## Steer focus: Documentation Health

> **Ladder position:** R2 (evolvable architecture — the docs map that keeps the system legible). See `prompt-manager skill read scenario-maturity-ladder` for rung context and `prompt-manager skill read improvement-do-and-dont` for what counts as a real improvement.

Prioritize **documentation quality, consistency, and bidirectional traceability** between code and documentation across this scenario.

Your goal is to ensure documentation remains accurate, discoverable, and tightly coupled to the code it describes, preventing drift, gaps, and duplication.

Do **not** change core business logic or introduce new features. All changes focus on documentation structure, references, and validation infrastructure.

Required reading:
- `prompt-manager skill read visited-tracker-tools`

---

### **1. Why This Skill Exists**

Documentation suffers from predictable failure modes:
- **Drift**: Code changes without corresponding doc updates
- **Gaps**: Features added without documentation
- **Duplication**: Agents can't find existing docs, so they create new ones
- **Orphaning**: Docs become disconnected from the code they describe

The root cause: **No systematic approach to bidirectional code↔docs traceability.**

This skill provides concrete patterns that ensure agents across multiple sessions maintain documentation the same way, preventing entropy over time.

---

### **2. Documentation Hierarchy**

Documentation should align with the mental model hierarchy used in screaming-architecture-audit:

Required reading:
- `prompt-manager skill read screaming-architecture-audit`

```
                      PRD.md
                   (Why does this exist?)
                         │
                         ▼
              Operational Targets
           (What must it accomplish?)
                         │
                         ▼
            Technical Requirements
         (How do we measure success?)
                         │
                         ▼
               Implementation Docs
        (How does the code achieve this?)
                         │
                         ▼
                    Code Files
               (The actual solution)
```

**Genre placement follows the Diátaxis model** (diataxis.fr): every reader-facing doc is one of tutorial, how-to guide, reference, or explanation — decide which *before* writing, and never mix modes in one document. Repo mapping: tutorial → `QUICKSTART.md`, how-to → `docs/guides/`, reference → `docs/reference/`, explanation → `docs/concepts/`. `docs/internal/` is deliberately outside Diátaxis: it is agent memory, not reader documentation.

---

### **3. Standard docs/ Directory Layout**

```
docs/
├── manifest.json          # Navigation & metadata (REQUIRED for UI display)
├── QUICKSTART.md          # First-touch experience
├── concepts/              # Mental model documentation
│   ├── ARCHITECTURE.md    # High-level system design
│   └── GLOSSARY.md        # Domain vocabulary
├── guides/                # Task-oriented walkthroughs
│   ├── getting-started.md
│   └── troubleshooting.md
├── reference/             # API, CLI, config specs
│   ├── api-endpoints.md
│   ├── cli-commands.md
│   └── configuration.md
├── internal/              # Developer-only docs (agent memory)
│   ├── SEAMS.md           # Integration boundaries, responsibility zones, testability
│   ├── PROBLEMS.md        # Known issues, tech debt, deferred work
│   ├── PROGRESS.md        # Development history, what's been completed
│   ├── INVARIANTS.md      # System contracts that must never be violated (optional)
│   ├── ASSUMPTIONS.md     # Implicit beliefs not yet validated (optional)
│   ├── ERROR-SEMANTICS.md # Error categories, recovery paths (optional)
│   ├── SECURITY-POSTURE.md # Security hardening status (optional)
│   ├── TEMPORAL-FLOWS.md  # Async patterns, race conditions, workflow maturity (optional)
│   ├── COHERENCE-NOTES.md # React coherence audit (React UIs only)
│   └── EXPERIENCE-AUDIT.md # UX friction analysis (user-facing only)
└── plans/                 # Architecture decisions, proposals
```

#### Document Placement Decision Table

| Content Type | Primary Location | Secondary Reference |
|--------------|------------------|---------------------|
| Why this exists | PRD.md | docs/concepts/ |
| How to use it | docs/QUICKSTART.md | docs/guides/ |
| API contract | docs/reference/api-endpoints.md | Inline JSDoc |
| CLI usage | docs/reference/cli-commands.md | --help output |
| Config options | docs/reference/configuration.md | Schema files |
| Known issues | docs/internal/PROBLEMS.md | GitHub Issues |
| Architecture decisions | docs/strategy/ or promoted docs/plans/ | ADR format |
| Scratch implementation plans | `plan-manager author start/continue/finalize` | Plan Manager structured record + rendered mirror |
| Code behavior | Inline comments | docs/reference/ |

---

### **4. Bidirectional Reference Format**

Establish consistent, searchable formats for linking code and documentation.

#### Code-to-Doc References (DOC: comments)

Use `// DOC:` comments to link code to its documentation:

**TypeScript/JavaScript:**
```typescript
// DOC: docs/reference/api-endpoints.md#user-authentication
// DOC: PRD.md#OT-P0-003
export function authenticateUser(token: string): Promise<User> {
```

**Go:**
```go
// DOC: docs/reference/api-endpoints.md#health-check
// HealthHandler returns the service health status.
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
```

**Multiple References:**
```typescript
// DOC: docs/guides/getting-started.md#configuration
// DOC: docs/reference/configuration.md#environment-variables
// DOC: PRD.md#OT-P0-003
const config = loadConfiguration();
```

#### Doc-to-Code References ([CODE: ...] syntax)

Use bracketed syntax in markdown that can be validated:

```markdown
## Authentication Flow

The authentication process is implemented in:
- [CODE: src/auth/authenticator.ts#authenticateUser] - Main entry point
- [CODE: src/auth/token.ts#validateToken] - Token validation
- [CODE: api/handlers/auth.go:AuthHandler] - HTTP handler

See also: [REQ: OT-P0-005] for the requirement this implements.
```

#### Reference Format Specification

| Reference Type | Format | Example |
|----------------|--------|---------|
| Code file | `[CODE: path/to/file.ext]` | `[CODE: src/auth/token.ts]` |
| Code function | `[CODE: path/to/file.ext#functionName]` | `[CODE: src/auth/token.ts#validateToken]` |
| Code line | `[CODE: path/to/file.ext:lineNumber]` | `[CODE: api/main.go:142]` |
| Requirement | `[REQ: ID]` | `[REQ: OT-P0-005]` |
| Doc section | `[DOC: path/to/doc.md#section]` | `[DOC: docs/QUICKSTART.md#setup]` |

#### Protective Comments for Critical Documentation

When documentation is critical for understanding, add protective comments:

```typescript
// ╔════════════════════════════════════════════════════════════════╗
// ║  IMPORTANT: This implements the core authentication flow.     ║
// ║  DOC: docs/concepts/ARCHITECTURE.md#authentication            ║
// ║  Before modifying, read the documentation and update it       ║
// ║  if your changes affect the described behavior.               ║
// ╚════════════════════════════════════════════════════════════════╝
```

---

### **5. manifest.json Documentation Pattern**

Every scenario with UI documentation should include a `docs/manifest.json`:

```json
{
  "version": "1.0.0",
  "title": "Scenario Documentation",
  "description": "Brief description of documentation scope",
  "defaultDocument": "QUICKSTART.md",
  "sections": [
    {
      "id": "getting-started",
      "title": "Getting Started",
      "icon": "rocket",
      "documents": [
        {
          "path": "QUICKSTART.md",
          "title": "Quick Start",
          "description": "Get running in 5 minutes",
          "audience": ["users", "developers"]
        }
      ]
    },
    {
      "id": "reference",
      "title": "Reference",
      "icon": "book",
      "documents": [
        {
          "path": "reference/api-endpoints.md",
          "title": "API Reference",
          "description": "REST API documentation"
        }
      ]
    },
    {
      "id": "internal",
      "title": "Internal",
      "icon": "lock",
      "visibility": "developers-only",
      "documents": [
        {
          "path": "internal/PROBLEMS.md",
          "title": "Known Issues",
          "internal": true
        }
      ]
    }
  ],
  "navigation": {
    "primary": ["getting-started", "reference"],
    "secondary": ["internal"]
  }
}
```

**Manifest Rules:**
- All docs in `docs/` should be registered in the manifest
- Use `visibility: "developers-only"` for internal docs
- Group related docs into sections
- Provide descriptions for discoverability

---

### **6. Documentation Health Audit**

Run this audit when joining a project, before major work, or periodically for maintenance.

```bash
knowledge-observatory docs audit {{TARGET}}
```

The audit checks:
- **Infrastructure**: manifest.json, required docs, file counts, misplaced/missing/extra
- **Code coverage**: files with exported symbols but no `// DOC:` references
- **Reference integrity**: broken `[CODE: ...]` references in documentation
- **Manifest registration**: orphaned docs not listed in manifest.json
- **Deduplication**: duplicate heading titles across doc files
- **PRD alignment**: operational targets (OT-*) without corresponding docs
- **Derived counts**: drift-prone hardcoded numbers in prose (the `numbers` check, surfaced by `docs health`)

Use `--json` for machine-readable output.

`docs health` checks are tagged **generic** (apply to any docs) vs **scenario** (need a scenario contract). Targeting works two ways:

```bash
knowledge-observatory docs health {{SCENARIO}}                 # all checks for a scenario
knowledge-observatory docs health --scope=path --path docs/    # generic checks over project-level docs
knowledge-observatory docs health {{SCENARIO}} --checks=numbers # narrow to one check
```

A scenario (or a path inside one) runs every check; a project-level path (`docs/`, `VISION.md`, `docs/<team>/`) runs only the generic checks. The `numbers` check is generic, so it runs in both — and runs automatically in every scenario's test-genie docs phase at warning severity.

#### Red Flags Checklist

- [ ] Missing `docs/manifest.json` → Create manifest for navigation
- [ ] Files with 500+ lines and no DOC: references → Add documentation links
- [ ] Broken `[CODE: ...]` references → Update paths or remove stale references
- [ ] Orphaned docs not in manifest → Add to manifest or delete if obsolete
- [ ] Duplicate documentation titles → Consolidate or differentiate
- [ ] PRD operational targets without docs → Create implementation docs

---

### **7. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `documentation-health`.

---

### **8. Relationship to Other Skills**

| Skill | Focus | When to Use Together |
|-------|-------|---------------------|
| screaming-architecture-audit | Mental model alignment | Documentation-health provides the docs that screaming-architecture reads first |
| ui-health | Code organization | Coherence patterns should be documented; docs should reference coherence decisions |
| refactor | Code cleanup | After refactoring, update DOC: references and [CODE: ...] links |
| code-cleanup | Dead code removal | Remove documentation for deleted code |

**Recommended sequence:**
1. **documentation-health** (audit) → understand documentation state
2. **screaming-architecture-audit** → align architecture with documented mental model
3. **refactor/code changes** → implement improvements
4. **documentation-health** (update) → sync docs with code changes

---

### **9. Scenario Constraints**

* Do **not** change the scenario's core workflows, APIs, or business logic
* Do **not** introduce new features unrelated to documentation
* Do **not** over-document trivial code (getters, setters, obvious utilities)

---

### **10. Output Expectations**

You may update:
* `docs/manifest.json` to register all documentation files
* Code files to add `// DOC:` comments linking to relevant documentation
* Documentation files to add `[CODE: ...]` references linking to implementation
* Documentation structure to follow the standard layout

You **must**:
* Keep the scenario fully functional and non-regressed
* Ensure all `[CODE: ...]` references point to valid files
* Ensure all `// DOC:` comments point to valid documentation
* Register new docs in `manifest.json`
* Document PRD operational targets in implementation docs

**Avoid:**
* Documentation that restates the code without adding context
* Over-documenting trivial functions
* Creating documentation that will immediately become stale — in particular, **don't freeze a derived current-state count or enumeration in prose** ("N teams", "30+ resources", "the four X"). It is a projection of a source of truth that drifts the moment the SoT changes. Omit it, or point at the SoT (a directory / registry / canonical list); reserve an inline number for cases where it is genuinely load-bearing for the reader, and then prefer generating or guarding it. Numbers that have an owner and a reason — targets, thresholds, prices, version pins, design decisions — are fine: they change by decision, they don't silently drift. **The derived-count lint enforces this**: `docs health` runs a `numbers` check that flags untagged counts in prose at warning severity. To keep an owner-backed number, tag it with the `num[<category>]` marker (`num[target]:1000`, `num[threshold]:100`, categories ∈ target/threshold/price/version/decision/sot — see `path:docs/reference/machine-readable-references.md`). Default to rewording out; tagging is the documented exception, and a `num` marker with no category is itself flagged.
* Duplicating information that belongs in a single source of truth

**Known-issue ledgers are tracked gaps, not clutter.** `docs/internal/PROBLEMS.md` and `docs/internal/PROGRESS.md` are core internal docs (the "Always — never skip" rows above). **Never delete one to "clean up" a scenario** — that erases the only record that the gap or the history exists. Entries leave a ledger because the work was *done* (then you update/migrate the entry), never because the file was *removed*. Deleting a ledger reads as metric-gaming and the controller flags it (see `improvement-do-and-dont`).

Focus on **documentation that helps agents quickly understand the scenario** and maintain accurate mental models across sessions.

---

### **11. Internal Document Templates**

The `path:docs/internal/` directory serves as **persistent agent memory** - documents written by agents to share findings with future agents. These are NOT user-facing documentation.

Fetch templates and their purposes on demand via the knowledge-observatory CLI:

```bash
knowledge-observatory docs templates              # List types with purpose descriptions
knowledge-observatory docs template "<type>"         # Get template content
```

Available types: seams, problems, progress, invariants,
error-semantics, security-posture, temporal-flows, coherence-notes, experience-audit

#### When to Create vs. Skip Files

| File | Create When | Skip When |
|------|-------------|-----------|
| SEAMS.md | Always - core internal doc | Never skip |
| PROBLEMS.md | Always - core internal doc | Never skip |
| PROGRESS.md | Always - core internal doc | Never skip |
| INVARIANTS.md | System has critical contracts or cross-cutting rules; also tracks unenforced-but-relied-on rules in Gaps section (see `invariant-discovery-and-enforcement`) | Simple CRUD with no invariants |
| ERROR-SEMANTICS.md | User-facing errors matter | Internal tooling only |
| SECURITY-POSTURE.md | Security is a concern | Internal-only, no auth |
| TEMPORAL-FLOWS.md | Async/concurrent operations, lifecycle flows, workflow maturity/spec status | Purely synchronous code |
| COHERENCE-NOTES.md | React UI exists | No React UI |
| EXPERIENCE-AUDIT.md | User-facing scenario | Backend-only service |

]]></skill>
  <skill id="test" name="Test"><![CDATA[
## Steer focus: Test Suite Strengthening

> **Ladder position:** R0 and R3 (runnable & green, then features hardened — coverage). Green-and-runnable is the floor; real coverage of critical behavior is the hardening. See `prompt-manager skill read scenario-maturity-ladder` for rung context and `prompt-manager skill read improvement-do-and-dont` for what counts as a real improvement.

> **Provider & scorer:** the **unit-health** scenario is the canonical test-maturity provider behind Test Genie's `unit` phase — it owns test execution, coverage, test architecture, test quality, and the local-maturity score. This skill is the *remediation* loop, **not** the scorer. To see a scenario's current test maturity, failing/uncovered surfaces, and the next blocker, run the human report:
>
> ```bash
> unit-health validate scenario {{TARGET}}
> ```
>
> (add `--execution` to actually run the test commands; `--json` is for Test Genie/programmatic consumers, not your workflow). Fix the findings it reports; don't treat this skill's prose as the authority on whether tests are "good enough" — `unit-health` is.

> **Policy profile contract:** for react-vite-derived scenarios, `.vrooli/testing.json`
> `unit.policy_profile` declares the template unit-test contract while Code Facts
> discovers the actual API/CLI/UI surfaces. Treat `UNIT_POLICY_*`,
> `UNIT_REQUIRED_ROLE_MISSING`, `UNIT_SURFACE_UNGOVERNED`, and
> `UNIT_POLICY_PROJECTION_DRIFT` as Unit Health contract findings: fix the
> declared profile or native test projection, not Test Genie orchestration.
> Legacy `unit.languages` is compatibility-only and is not the policy source for
> new generated scenarios.

Prioritize **test quality, coverage, and reliability** across this scenario.
Do **not** break functionality or regress existing tests; all changes must maintain or improve overall completeness.

Focus on producing a **high-signal, trustworthy test suite** that accurately reflects the scenario’s operational targets and technical requirements.

Required reading:
- `prompt-manager skill read visited-tracker-tools knowledge-observatory-tools`

---

### **1. Align Tests With Operational Targets & Requirements**

* Read the `problems` doc for `{{TARGET}}` using `knowledge-observatory-tools` to understand existing test gaps.

* For **UI-level validation**, e2e tests are handled by BAS workflows in `bas/` directories. See the **e2e-testing** skill for strategy and the **browser-automation-studio** skill for CLI usage. This skill focuses on unit and integration tests that complement (not duplicate) e2e coverage.

Optional reading:
- `prompt-manager skill read e2e-testing browser-automation-studio`

* Ensure **each operational target** has clear, meaningful test coverage through its linked technical requirements.
* Where gaps exist, add tests that validate the **actual behavior** users and systems depend on, not just internal implementation details.
* Prefer tests that verify the **full intent** of a requirement (happy path + key edge cases), rather than narrow or trivial assertions.

---

### **2. Increase Coverage Where It Matters Most**

* Identify and strengthen coverage for:

  * **Critical user journeys** and core workflows
  * Error handling and fallback behavior
  * Boundary conditions (empty inputs, large inputs, missing data, timeouts)

* Prioritize **high-impact areas** where a regression would meaningfully harm the user experience or operational reliability.
* Avoid adding low-value tests that simply increase raw counts without improving real protection.

---

### **3. Improve Assertion Quality & Signal Strength**

* Upgrade vague or weak assertions (e.g. “component renders”) to **specific, behavior-focused checks**:

  * correct outputs
  * correct side effects
  * correct UI states and transitions

* Ensure tests would **fail clearly and immediately** if the behavior they protect were broken.
* Avoid loosening tests or weakening assertions just to make them pass; tests should **enforce correctness**, not accommodate bugs.
* **Never weaken or delete a `[REQ:]`-tagged test to get green.** That test is the executable definition of a tracked requirement; removing its assertion un-defines the requirement (and the controller will not count it toward operational targets). Fix the code, or — only if the assertion is genuinely wrong — make it *more accurate* to the requirement, never looser. See `improvement-do-and-dont`.

---

### **4. Reduce Flakiness and Brittleness**

* Identify and fix sources of **flaky or timing-sensitive tests**:

  * brittle selectors
  * arbitrary timeouts
  * unnecessary reliance on network, random data, or global state

* Make tests **deterministic and repeatable** by:

  * controlling randomness
  * isolating side effects
  * mocking external dependencies only where appropriate

* Keep the balance: don’t over-mock to the point that tests no longer reflect real user-visible behavior.

---

### **5. Organize and Simplify the Test Suite**

* Improve **structure, naming, and grouping** so tests are easy to read, navigate, and extend:

  * clear test names describing intent and behavior
  * logical grouping by feature, domain, or workflow
  * shared helpers for repeated setup and assertions

* Remove or refactor **redundant, overlapping, or obsolete tests** only when you’re confident they no longer provide unique value.
* Keep test files focused and approachable so future agents (and humans) can quickly understand what’s covered and what’s missing.

---

### **6. Anti-Patterns: Test the Specification, Not the Diff**

The suite is an executable specification of what the scenario does **now**.
Tests that encode the *history of a change* instead of a *current requirement*
are noise that hardens into obstruction. Avoid these shapes:

* **Tombstone tests.** After removing a feature, do not write tests asserting
  the behavior is gone ("the old endpoint returns 404", "the flag no longer
  exists"). Absences are unenumerable, meaningful only to someone holding the
  removal diff, and can block a legitimate future feature that reuses the
  surface. The removal protocol is: delete the old feature's tests, update the
  requirements registry (delete the requirement or mark it `not_implemented`),
  and **positively test the replacement behavior**. Removal coverage = the
  suite passes without the feature.
* **"Shall not" done as absence.** A genuine ongoing prohibition (auth
  rejection, input validation, rate limiting) is a *positive* claim about an
  observable response — "if an unauthenticated request arrives, the API
  returns 401" — and deserves a permanent test asserting that response. If it
  matters enough to test forever, it belongs in `requirements/` as an
  unwanted-behaviour requirement; that is a judgment call, not a mandate to
  tag every test.
* **Change-detector tests.** Assertions that pin incidental implementation
  detail (exact private call order, snapshot-everything) fail on harmless
  refactors and pass on real regressions. Assert the behavior a requirement
  names.
* **Characterization tests as permanent fixtures.** Pinning current behavior
  wholesale is legitimate *temporarily* while refactoring untested code —
  mark them as scaffolding and delete them when the refactor lands.

---

### **7. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `test`.

---

### **8. Output Expectations**

You may update or add:

* unit, integration, and end-to-end tests (e2e tests handled by the bas/ workflows)
* test utilities, fixtures, and helpers
* test naming, structure, and organization
* coverage of error paths, edge cases, and critical workflows
* mock tools such as testcontainers-go

You **must**:

* keep the scenario fully functional
* avoid regressions
* improve the **trustworthiness and clarity** of the test suite
* raise the **real** protective value of the tests, not just their quantity

Focus this loop on delivering **practical, high-impact test improvements** that make the scenario safer to evolve, easier to reason about, and more accurately measured by its programmatic completeness.

**Avoid superficial tests that increase coverage numbers without meaningfully protecting behavior. Only add or modify tests when they genuinely sharpen the feedback signal and reduce the risk of unnoticed regressions.**

---

### **9. Documentation**

Use `knowledge-observatory-tools` to read the current `problems` doc for `{{TARGET}}`, then update the **Test Gaps** section with your findings (critical flows lacking coverage, flaky tests, assertion quality issues, remaining coverage priorities).

]]></skill>
  <skill id="domain-clarity" name="Domain Clarity"><![CDATA[
## Steer focus: Domain Clarity

Prioritize making the **domain model of `scenarios/{{TARGET}}/` small, consistently named, and explicit about intent** — so any agent can answer "what are the core concepts, what is each called, and why does each exist?" from the code and its `docs/concepts/` alone. This skill governs the conceptual layer: which concepts exist, what they are named, and whether their purpose is visible. It does not move files, own lifecycle, or polish local function shape — those hand off (see §1).

Do not change observable behavior, regress tests, or add product features. Every change must maintain or improve completeness and test health.

### Required reading

- `prompt-manager skill read knowledge-observatory-tools` — read and update `docs/concepts/` through the canonical docs CLI.
- `prompt-manager skill read screaming-architecture-audit` — structural/folder shape; the audit this skill hands structural findings to.
- `prompt-manager skill read cognitive-load-reduction` — local naming and function shape; the audit this skill hands local-readability findings to.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — current domain model (entities, actions, states).
- `scenarios/{{TARGET}}/docs/concepts/GLOSSARY.md` — current canonical vocabulary.
- The scenario's PRD, operational targets, and technical requirements — the authoritative domain language.

---

### 1. Scope Boundaries

This skill owns the **conceptual model**, not its physical layout or its local code-craft. Three lenses, one boundary.

**In scope:**
- **Concept compression** — discover the smallest, clearest set of domain concepts; collapse conceptual duplicates (multiple names for one idea, near-duplicate shapes, parallel flows for the same journey) toward one canonical concept per idea.
- **Vocabulary unification** — one primary term per concept across code, tests, and UI; split overloaded names so each concept gets a distinct term; align terms with the PRD's domain language.
- **Intent documentation** — make purpose visible: names that answer "what is this for / when does it change", and `why`-comments where a non-obvious decision, invariant, or constraint would otherwise be lost.

**Out of scope (hand off):**

| Finding | Hand off to |
|---|---|
| Folders organized by tech bucket instead of domain; files in the wrong domain folder | `screaming-architecture-audit` |
| Lifecycle, retry, cancellation, state-machine ownership complexity | `temporal-flow-audit` |
| Local naming of variables/params, function length/nesting, comment-restates-code | `cognitive-load-reduction` |
| Whether an abstraction earns its keep across domains | `seam-discovery-and-enforcement` |
| Public API surface, transport, contracts | `api-steer` |
| Dead code, stale config, abandoned files | `code-cleanup` |

The distinction from `cognitive-load-reduction`: that skill asks "is this *name* clear and this *function* shaped well?"; this skill asks "is this the *right concept*, is it *named consistently everywhere*, and is its *purpose* visible?" Rename a local loop variable → cog-load. Reconcile that `job`, `run`, and `task` are three names for one concept → here.

---

### 2. Domain Clarity Maturity Model

Assess the scenario against this ladder; each level is gated by a verifiable artifact, not an adjective. Use it to pick the next concrete move, not as a score to inflate.

| Level | Name | What exists | When to stop here |
|---|---|---|---|
| 0 | Untracked | No `docs/concepts/ARCHITECTURE.md`; domain model lives only in code and agent memory. | Never — write the minimal domain model first. |
| 1 | Documented | `ARCHITECTURE.md` names the core entities, primary actions, and key states, verified against code. | The domain is small and the doc is accurate; no duplication or naming drift found. |
| 2 | Compressed | Conceptual duplicates are collapsed: one canonical shape per core concept, one primary flow per journey. `ARCHITECTURE.md` records where duplicates were merged. | `grep` finds no parallel near-duplicate types/flows for the same concept; distinctions that remain are PRD-justified. |
| 3 | Unified vocabulary | One primary term per concept across code, tests, and UI; overloaded names split. `GLOSSARY.md` records canonical terms and old→new mappings. | `grep` for each retired synonym returns only historical/adapter mentions; boundary layers use domain terms, not implementation jargon. |
| 4 | Intent explicit | Names express purpose; non-obvious logic carries `why`-comments referencing the invariant or decision. Names, comments, and tests agree on the same intent. | A new agent can state why each core concept exists without reading call sites. |

A scenario may sit at different levels for different surfaces; advance the weakest core concept first. Do not claim a level whose artifact (`ARCHITECTURE.md` / `GLOSSARY.md` entry, or a clean `grep`) does not exist.

---

### 3. Lens-Routing Table

Classify each finding into one lens and act deterministically. Two agents reading the same scenario must land in the same row.

| Signal | Lens | Action | Durable anchor |
|---|---|---|---|
| Two+ names for the same idea; near-duplicate types/shapes; parallel flows for one journey | Concept compression | Choose one canonical concept; align code where clearly safe and PRD-consistent | `ARCHITECTURE.md` (what was merged, core vs edge) |
| Same concept, different terms across code/tests/UI; one name used for two concepts | Vocabulary unification | Choose one primary term; split overloaded names; update uses cohesively | `GLOSSARY.md` (canonical terms, old→new) |
| Generic name (`data`, `manager`, `doStuff`) where a domain term exists; purpose not visible | Intent documentation | Rename to purpose; add a `why`-comment only where intent is non-obvious | Code + `ARCHITECTURE.md` |
| Structural / lifecycle / local-craft (see §1 table) | — | Stop and hand off | The owning audit's doc |

Rules: only remove code when it is provably unused or superseded and tests still pass meaningfully. Prefer localized, safe changes over repo-wide renames in one step. Avoid over-compression — do not merge genuinely distinct concepts (materially different behavior, or the PRD keeps them separate) just to cut concept count. If a larger merge is beneficial but risky, record it in `ARCHITECTURE.md` rather than partially applying it.

---

### 4. Anti-Gaming

Real domain clarity changes which concepts exist and what they are called throughout, not surface symbols. Avoid: renaming purely stylistic local symbols, reformatting, shuffling files, or adding comments that restate the signature. A change counts only when it removes a conceptual duplicate, retires a synonym across the codebase, or makes a previously hidden purpose visible.

---

### 5. Output Expectations

You may update naming of domain types/entities/operations, canonical data shapes (where safe), core-journey flow structure, UI labels, tests that misrepresent intent, and `docs/concepts/` docs. You must preserve observable behavior and user workflows, keep public contracts stable where consumed externally (update all in-scenario call sites when you must change them), and maintain or improve test coverage.

Update `docs/concepts/ARCHITECTURE.md` and `docs/concepts/GLOSSARY.md` (user-facing) via `knowledge-observatory-tools`. The code is the source of truth: verify existing claims against code before extending, correct inaccuracies, and record new discoveries. Do not create standalone `*_AUDIT.md` files — findings route to these existing docs.

By the end of a loop the scenario should have advanced at least one core concept on the §2 ladder: a duplicate collapsed, a synonym retired across the codebase, or a hidden purpose made explicit — with the matching `docs/concepts/` entry updated.

]]></skill>
  <skill id="requirements-traceability-steer" name="Requirements Traceability"><![CDATA[
## Steer focus: Requirements Traceability

`scenarios/{{TARGET}}/`'s requirements registry is a **claim** about what the scenario does and how you'd know; when behavior changes and the registry doesn't, the claim is a lie. Your job is to **make the registry true, not green** — every requirement tied to real evidence, every PRD operational target tied to requirements, every status earned rather than declared.

The drift signal you are answering comes from the test-genie `business` phase's typed findings (source `BUSINESS`): starter-template registries, requirements with no validation, dangling validation refs, prd_refs that match no PRD operational target. Move `{{TARGET}}` **up the Traceability Maturity Ladder** (§2) until the latest business-phase run is findings-clean — do not invent new requirements to look thorough.

Required reading:
- `path:scenarios/test-genie/docs/requirements/STATUS_MODEL.md` — how declared status, live evidence, and the sync snapshot compose (statuses are *earned* from `[REQ:ID]`-tagged test results, not asserted).
- `path:scenarios/test-genie/docs/requirements/IMPROVING_COVERAGE.md` — the `[REQ:ID]` tagging mechanics per language and the validation-ref formats (`path/file.go::TestName`).
- `path:scenarios/business-health/docs/reference/canonical-prd-template.md` — the PRD shape; operational targets (`OT-P0-001 | Title | …`) are the IDs `prd_ref` must match.

Read first when present (prior findings — continue, don't restart):
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — deferred requirements work or a known-stale module.

> Universal authoring/quality bars (intent statement, convergence patterns, anti-gaming framing, the agent memory loop) are canon in `path:docs/agent-system/SKILL_AUTHORING.md` and are not restated here.

---

### 1. Scope Boundaries

**In scope** (anchored to `scenarios/{{TARGET}}/`):
- truing `requirements/*.json` to actual behavior: descriptions, statuses, criticality, `validation[]` refs.
- `prd_ref` ↔ PRD.md operational-target linkage (both directions: dangling refs and uncovered targets).
- tagging existing tests with `[REQ:ID]` and pointing `validation[].ref` at them.
- flipping PRD operational-target checkboxes **only** when the linked requirements are complete with passing evidence.
- recording deferred registry work in `docs/internal/PROBLEMS.md`.

**Out of scope** (hand off):
- authoring a brand-new PRD or renegotiating product scope → `prompt-manager skill read prd-authoring`, which owns the wizard flow (`vrooli scenario requirements validate` tells you *what's unlinked*, not *what to build*).
- deciding whether the PRD still describes the right product at all → that is the W0 rung of `prompt-manager skill read scenario-work-ladder`, and it is settled before this skill runs. This skill is that ladder's W2 rung: it trues the registry against the contract, and assumes the contract itself is true.
- test quality / writing new test suites → the `test` skill; here you only **link** evidence that exists (write a test only when a P0 requirement has none at all).
- the requirements **sync** machinery itself (test-genie internals) → never hand-edit sync snapshots or force sync to make statuses move.
- CLI/API/manifest contract conformance → `cli-steer` / `api-steer`.

---

### 2. Traceability Maturity Ladder

Grade `{{TARGET}}` against this ladder, then climb only as far as the scenario's reality justifies. Every level is gated by a **runnable command**, so two agents grading the same scenario land on the same rung.

The findings command throughout (the same producer EM scores the `business` dimension with):

```bash
test-genie execute {{TARGET}} business --json   # .phases[] | select(.name=="business") | .findings[]
```

| Level | Name | What exists (verifiable artifact) | When to stop here |
|---|---|---|---|
| **0** | Starter template | `grep -rl template-starter scenarios/{{TARGET}}/requirements/` is non-empty (finding `business_starter_template`). The registry describes the scaffold, not the scenario. | Never. A starter registry is a placeholder, not a claim. |
| **1** | Honest skeleton | Registry parses and is structurally sound: `vrooli scenario requirements validate {{TARGET}}` exits 0; no error-severity business findings (`business_duplicate_req_id`, `business_import_cycle`, `business_orphaned_ref`). Requirements describe *this scenario's* intended behavior. | Only mid-build, when behavior is still moving daily. |
| **2** | Evidence-linked | Every requirement has at least one real validation ref: no `business_req_no_validation` / `business_validation_ref_missing` findings; refs resolve to files that exist. | A scenario with no automated tests yet — but then a P0 with no validation is an ERROR finding you must not waive away; write the missing test. |
| **3** | Live-traced | Tests carry `[REQ:ID]` tags and the sync snapshot shows live evidence: `vrooli scenario requirements report {{TARGET}}` shows `live_passed > 0` and a `critical_gap` of 0; `vrooli scenario requirements snapshot {{TARGET}}` is from a recent full run. | Most scenarios. Statuses now *earn* themselves via sync on comprehensive runs. |
| **4** | Findings-clean | Latest business run emits **zero** BUSINESS findings; `vrooli scenario requirements lint-prd {{TARGET}}` shows every P0 operational target linked; PRD checkboxes match requirement completion. | **The target.** Re-grade after every behavior change (§3). |

The rung is not a vanity score — write the level reached (and what still blocks the next one) into `PROBLEMS.md` so the next agent continues rather than re-discovers.

---

### 3. You changed X → check Y

Walk this table **after any change to `{{TARGET}}`**; it is the per-change reflex that keeps the registry true. Each row is deterministic — no judgment call about *whether* to check, only *what you find*.

| You changed… | Check… |
|---|---|
| Added a new capability / endpoint / command | Does an operational target + requirement for it exist? If not, draft the requirement now and link `prd_ref` (add the OT line to PRD.md, or extend it via the business-health wizard, if the PRD predates the capability). |
| Changed existing behavior | Which requirement described the old behavior? Its description, validation refs, or status is now wrong — fix whichever side is the lie (§4 ban #2 decides how). |
| Removed behavior | The requirement that claimed it: mark `not_implemented` or delete it (and its PRD checkbox), don't leave a passing-looking ghost. |
| Added a test | Tag it `[REQ:ID]` and add/confirm the `validation[]` ref pointing at it — an untagged test is invisible to sync. |
| Renamed/moved a test file | Every `validation[].ref` that pointed at the old path is now a `business_validation_ref_missing` finding — update the refs. |
| Edited PRD.md operational targets | `vrooli scenario requirements lint-prd {{TARGET}}` — every `prd_ref` must still match; every new P0 target needs a requirement. |

---

### 4. Anti-gaming bans (the registry must be true, not green)

The EM `gameguard` zeroes credit for suppression-shaped fixes; these are the suppression shapes for this dimension:

1. **Never flip a status to `complete` without a passing validation ref.** Status is downstream of evidence; `vrooli scenario requirements sync` earns it on full runs. Hand-flipping is lying to the ladder.
2. **Never rewrite a requirement's description to match drifted code without deciding which side is wrong.** Code-follows-spec or spec-follows-code is a *product decision*: cite the operational target it serves. If the OT still wants the old behavior, the code is the bug — surface it (`report-bug`) instead of papering the registry over it.
3. **Never bulk-add boilerplate requirements** (one per file/endpoint, copy-pasted descriptions) to fatten linkage counts. One requirement = one falsifiable behavioral claim someone could test.
4. **Never delete or waive a P0 requirement to silence a `business_req_no_validation` ERROR.** Write the missing test, or downgrade the criticality *with a written reason* if it was never truly P0.
5. **Never point `validation[].ref` at a file that doesn't actually validate the claim** (e.g. an unrelated test that merely exists). The ref's job is to let a human jump from claim to proof.

---

### 5. Requirement wording standard (EARS + RFC 2119)

When you author or true a requirement's `title`/`description`, write it as an
**EARS** (Easy Approach to Requirements Syntax) statement — the falsifiable-
behavioral-claim rule (§4 ban #3) made structural. Pick the template that fits:

| Pattern | Template |
|---|---|
| Ubiquitous | The `<system>` shall `<response>`. |
| Event-driven | When `<trigger>`, the `<system>` shall `<response>`. |
| State-driven | While `<state>`, the `<system>` shall `<response>`. |
| Unwanted behaviour | If `<undesired trigger>`, then the `<system>` shall `<response>`. |
| Optional feature | Where `<feature is present>`, the `<system>` shall `<response>`. |

Use **RFC 2119** keywords with their defined meanings: `shall`/`must` for
P0-linked requirements, `should` for P1, `may` for P2. Do not use those words
loosely elsewhere in the description.

Two consequences worth internalizing:

- A "the system must not X" obligation is an **unwanted-behaviour** claim about
  an observable response ("If an unauthenticated request arrives, then the API
  shall return 401"), never a bare absence ("X no longer exists"). Absence
  claims are untestable and unenumerable.
- This is an on-touch standard, not a migration: rewrite a requirement into
  EARS form when you are already editing it. Do not bulk-rewrite a registry
  just to change wording — that is churn, not truth.

---

### 6. Verification gate

Before claiming the `business` dimension closed for `{{TARGET}}`:

```bash
vrooli scenario requirements validate {{TARGET}}        # structural: exit 0
vrooli scenario requirements report {{TARGET}}          # coverage: no critical_gap
test-genie execute {{TARGET}} business --json           # producer: zero BUSINESS findings
```

All three clean = L4. Anything less, record the rung + blockers in `PROBLEMS.md`.

---

### 7. Output expectations

You **may**: edit `scenarios/{{TARGET}}/requirements/*.json` (descriptions, refs, statuses *with evidence*, criticality with reasons); tag tests with `[REQ:ID]`; fix `prd_ref` values; flip PRD checkboxes whose requirements are complete with passing evidence; update `PROBLEMS.md`.

You **must**: keep every requirement a falsifiable behavioral claim; keep validation refs resolvable; run the §6 gate before claiming done.

You **must NOT**: hand-edit sync snapshots or `coverage/` artifacts; force requirements sync (`TESTING_REQUIREMENTS_SYNC_FORCE`) to move statuses; restructure the PRD outside the operational-targets section (business-health owns the document shape — `canonical-prd-template.md`); create standalone `*_AUDIT.md` reports — findings go in durable docs.

---

### 8. Troubleshooting & Edge Cases

- **`validate` passes but the business phase still emits findings.** `validate` is structural only; the producer also checks registry drift (starter tags, empty `validation[]`, unmatched `prd_ref`). The findings list *is* the work queue.
- **`business_prd_ref_unmatched` but the target looks right.** The producer matches literal `OT-…` tokens in PRD.md; a reformatted or renamed target breaks the match. Fix the ref or the PRD line — exact ID match, no fuzz.
- **Statuses won't move after fixing refs.** Sync only runs on full (comprehensive) suite runs with no skipped required phases — `test-genie execute {{TARGET}}` (no phase filter) and check `vrooli scenario requirements snapshot {{TARGET}}` afterward. Quick/smoke runs validate but never write.
- **A requirement is real but genuinely can't have an automated validation** (e.g. a manual ops procedure). Use a `manual` validation type with `vrooli scenario requirements manual-log` evidence instead of leaving `validation[]` empty.
- **No PRD.md at all.** The prd_ref check skips silently; the scenario needs a PRD first — drive `business-health wizard` — record that in `PROBLEMS.md` and stop at L2.

]]></skill>
  <skill id="seam-discovery-and-enforcement" name="Testing Seam Discovery &amp; Enforcement"><![CDATA[
## Steer focus: Seam Discovery & Enforcement

Prioritize **how variation is substituted** in `scenarios/{{TARGET}}/`. A *seam* is a named interface declared at the point of use whose production implementation is wired once and whose test double lives in a known catalog. This skill governs the seam itself — its shape, its fakes, and the registry that lets a future agent find it — not the directory layout that holds it.

The destination is the React-Vite-template shape: every dependency a domain has on the outside world (time, network, env, log, persistence) is a Go interface in the domain's own package; production wires the concrete in `main.go` / `server.Deps`; tests substitute a fake from `internal/<domain>/mocks/` or `internal/testutil/mocks/`. Drift is gated by a seam-registry test that reconciles `// seam:`-tagged interfaces with `SEAMS.md`.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read boundary-of-responsibility-enforcement` — the directory layout this skill assumes seams live inside.

Read first when present:
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — the seam registry this skill is the source of truth for.
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — domain map; seams are owned by domains.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — unresolved seam drift.

---

> **Template example domain — delete on generation.** The react-vite template ships canonical seam examples: `path:templates/scenarios/react-vite/api/internal/clock/clock.go` (ambient time), `path:templates/scenarios/react-vite/api/internal/httpc/doer.go` (outbound HTTP), and `path:templates/scenarios/react-vite/api/internal/notes/repository.go` (domain persistence). The `notes` domain is starter scaffolding to study, not boilerplate to keep — when a scenario is generated, replace it with real domains and apply the same seam shape. This skill uses `<domain>`, `<Domain>Service`, `<Seam>` (interface name), and `<Fake><Seam>` (test double) as placeholders. Substituting them is the prompt to ask whether a leftover `notes` seam is your scenario's real shape or template residue.

---

### 1. Scope Boundaries

**In scope:**
- the interface itself: name, method surface (narrow, single-purpose), where it is declared
- the production implementation: how it is constructed once and threaded through `server.Deps` / handler constructors
- the test double: where the fake lives, what shape it has, whether it satisfies the same compile-time check as the production impl (`var _ Seam = (*Fake)(nil)`)
- the four canonical ambient seams every scenario needs: clock, outbound HTTP, env reader, logger
- domain-level seams: `Repository`, integration clients, blob stores, scheduler clients
- the registry: keeping `SEAMS.md` and the code in lockstep, and gating drift with a registry test

**Out of scope:**
- *where* the interface file lives in the directory tree — that is the `boundary-of-responsibility-enforcement` concern (zone map, what may import what)
- proto-defined RPC contracts and Connect handler wiring — use `api-steer`
- product capability identification — use `screaming-architecture-audit`
- the contents of the fake's behavior beyond "it satisfies the seam" — domain-specific test fixtures live in `internal/testutil/fixtures/` and are tuned per test

**Decision rule for the reader.** Is the question *where does this code live?* → `boundary-of-responsibility-enforcement` (boundary). Is the question *how is this dependency substituted?* → this skill (seam). The two skills are paired and cite each other; a refactor that crosses the line must cite both. (Boundary-side observations encountered during a seam audit go to `ARCHITECTURE.md`, not into a parallel seam doc.)

---

### 2. Seam Maturity Model

Score each seam independently. A scenario has many seams; some may be L5 while others are L1.

| Level | What exists | Verifiable signal | When to stop |
|---|---|---|---|
| 0 | Side effects inline: `time.Now()`, `http.DefaultClient.Do(...)`, `os.Getenv(...)`, raw SQL in handlers; tests skip or mutate globals. | `rg "time\.Now\(\)\|http\.DefaultClient\|os\.Getenv\(" scenarios/{{TARGET}}/api/internal/<domain>/` returns hits. | Never — L0 is a finding, not a target. |
| 1 | `SEAMS.md` exists with a flat list naming each seam, its production file path, and its fake path. | The file is present and lists at least the four ambient seams (clock, http client, env, log). | Fewer than three seams are documented. |
| 2 | Each seam is a Go interface declared in the package that consumes it; a `// seam:` comment tags the declaration; a compile-time check (`var _ <Seam> = (*<Impl>)(nil)`) anchors production and fake. | `rg "// seam:" --type go` lists every interface declared as a seam; each location has a matching `var _ <Seam>` assertion in the production file and the fake file. | A seam is informal (a function value, an exported global) rather than an interface. |
| 3 | The four ambient seams are replaced: no domain package contains `time.Now()`, `os.Getenv()`, `http.DefaultClient`, or `log.Default()`. Each ambient is injected via `server.Deps`. | `rg "time\.Now\(\)\|os\.Getenv\(\|http\.DefaultClient\|log\.Default\(\)" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!internal/{clock,httpc,server}/**'` returns zero hits. | Not all four ambients have been migrated. |
| 4 | Every seam has at least one production impl AND one test double, both with compile-time `var _` assertions. `SEAMS.md` lists every seam with: declaration site, production impl file, test-double file, why it exists. | `rg "// seam:" --type go \| wc -l` equals the row count in `SEAMS.md`'s seam table. | The registry has not been reconciled this cycle. |
| 5 | Drift-gated. A registry test (analogous to `path:templates/scenarios/react-vite/api/internal/testutil/no_prod_import_test.go`) walks the AST, finds every `// seam:`-tagged interface, and asserts each appears in `SEAMS.md` (and vice versa). New ambient calls in domain code fail CI. | `go test ./internal/testutil/... -run TestSeamRegistry` passes; CI breaks when a new `time.Now()` lands in a domain file. | This is the destination. |

Use the level to pick the next concrete move: name the seam, write the interface, ship the fake, migrate the ambient, register it, gate it.

---

### 3. Seam Archetype Decision Model

| Archetype | Use when | Canonical interface shape | Production impl | Test double |
|---|---|---|---|---|
| Ambient substrate | A platform primitive (time, network, env, log) the domain depends on but does not own | Single-method interface in `internal/<substrate>/` (e.g. `clock.Clock { Now() time.Time }`) | Struct in the same package (`clock.System{}`) | `internal/testutil/mocks/<substrate>.go` |
| Outbound integration | The domain calls an external HTTP/gRPC service | Narrow interface in `internal/<domain>/` exposing only the operations the domain uses | Adapter struct that wraps an `httpc.Doer` | `internal/<domain>/mocks/<integration>.go` |
| Persistence | The domain reads or writes durable state | `Repository` interface in `internal/<domain>/repository.go` | `<domain>_sqlite.go` or `<domain>_postgres.go` in the same package | `internal/<domain>/mocks/repository.go` |
| Process-out side effect | The domain enqueues background work, emits events, or writes blobs | Narrow interface (`EventPublisher`, `BlobStore`) in `internal/<domain>/` | Concrete adapter in `internal/<substrate>/` | Per-domain mock; recorded calls in tests |
| Policy / decision | A decision the scenario wants to vary per environment or per tenant | Pure function-shaped interface (`Authorizer`, `RateLimiter`) | Production rule struct | Programmable fake returning canned decisions |
| Clock-shaped derivative | Tickers, sleeps, jitter — anything `time`-derived beyond `Now()` | Extend the `clock.Clock` interface rather than adding a parallel seam | Extend `clock.System` | Extend `mocks.FakeClock` |

Decision rule:

```text
Does the call cross a process boundary (network, disk, syscall)?
  YES -> it needs a seam.
Does the call read a global (env, default client, default logger, wall clock)?
  YES -> it needs a seam.
Does the dependency vary per environment or per tenant?
  YES -> it needs a seam.
Is the dependency a pure stdlib data transform?
  NO  -> no seam.
Does an existing seam already cover this concern?
  YES -> extend it (add a method) rather than introduce a parallel one.
```

The interface is declared **in the package that consumes it**, not in the package that implements it. The consumer owns the seam; the producer satisfies it. This is the Go idiom and the precondition for narrow surfaces.

---

### 4. Canonical Seam Shape

A healthy seam in the React-Vite template looks like:

```go
// internal/<domain>/repository.go
package <domain>

import "context"

// seam: Repository persists <Resource> rows. Production wires
// <domain>Sqlite from <domain>_sqlite.go; tests wire FakeRepository
// from mocks/repository.go.
type Repository interface {
    Create(ctx context.Context, r <Resource>) (<Resource>, error)
    Get(ctx context.Context, id string) (<Resource>, error)
}
```

```go
// internal/<domain>/<domain>_sqlite.go
package <domain>

type SqliteRepository struct{ db *sql.DB }

func (r *SqliteRepository) Create(...) { ... }
func (r *SqliteRepository) Get(...)    { ... }

var _ Repository = (*SqliteRepository)(nil)
```

```go
// internal/<domain>/mocks/repository.go
package mocks

type FakeRepository struct{ ... }

func (f *FakeRepository) Create(...) { ... }
func (f *FakeRepository) Get(...)    { ... }

var _ <domain>.Repository = (*FakeRepository)(nil)
```

Invariants:
- The interface is declared in the consumer package, not in a separate `interfaces/` bucket.
- Both production and fake carry the `var _ <Seam> = (*Impl)(nil)` compile-time check. Renaming a method on the interface fails the build everywhere it must.
- The fake's package name is `mocks`, lacks the `_test.go` suffix (so sibling `_test.go` files in other packages can import it), and is exempt from `no_prod_import_test.go` via the `mocks/` directory rule.
- The interface surface is narrow — only the methods the consumer actually calls. Resist exposing the full `*sql.DB` surface through a `Repository` interface.
- The four ambient seams (clock, httpc, env, log) live in `internal/<substrate>/`. Domain seams live in `internal/<domain>/`.

---

### 5. The Four Ambient Seams

Every scenario inherits these from the template. Migrating L0→L3 is mostly the work of replacing the ambient call with the injected seam.

| Seam | Interface | Production | Fake |
|---|---|---|---|
| Wall clock | `clock.Clock { Now() time.Time }` | `clock.System{}` in `internal/clock/clock.go` | `mocks.FakeClock` in `internal/testutil/mocks/clock.go` |
| Outbound HTTP | `httpc.Doer { Do(*http.Request) (*http.Response, error) }` | `*http.Client` (satisfies via assertion) | `mocks.FakeDoer` in `internal/testutil/mocks/doer.go` |
| Env reader | `envx.Reader { Get(key string) string }` | `envx.OS{}` | `mocks.FakeEnv` (programmable map) |
| Structured logger | `logx.Logger` (project-specific surface) | `slog.Logger` adapter | `mocks.FakeLogger` (records calls) |

The template ships `clock` and `httpc` as concrete examples and `httpx` middleware as a consumer of both. `envx` and `logx` follow the same shape; if the template has not landed them yet, framing them as the L3 destination for new scenarios is the right move.

---

### 6. Audit Workflow

1. **Read `SEAMS.md`.** Treat each row as a claim: does the interface exist at the listed path? Does the fake satisfy it?
2. **Inventory every interface.** `rg "^type \w+ interface" scenarios/{{TARGET}}/api/internal --type go` — each one is a seam candidate. Tag with `// seam:` or remove if it is internal collaboration.
3. **Check ambient leaks.**
   ```bash
   rg "time\.Now\(\)" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!internal/clock/**'
   rg "http\.DefaultClient\|http\.Get\(\|http\.Post\(" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!internal/httpc/**'
   rg "os\.Getenv\(" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!main\.go'
   rg "log\.Default\(\)\|log\.Print" scenarios/{{TARGET}}/api/internal -g '!*_test.go'
   ```
   Each hit is a concrete L0→L3 task.
4. **Verify compile-time assertions.** `rg "var _ \w+\.\w+ = " --type go` — every seam should have one in both the production and the fake file.
5. **Verify fakes exist.** For each `// seam:` interface, `fd "<seamname>.go" internal/<domain>/mocks internal/testutil/mocks` should hit.
6. **Reconcile with `SEAMS.md`.** Every `// seam:` interface appears in the registry; every registry row has a `// seam:` interface.
7. **Run the enforcement tests.** `go test ./internal/testutil/...` — `no_prod_import_test.go` for production-side cleanliness, plus the seam-registry test if it exists. Where it does not yet, vendor the pattern from the template.
8. **Update `SEAMS.md`.** Land discovered seams; record unresolved drift in `PROBLEMS.md`.

---

### 7. Red Flags

- A test that calls `time.Sleep` to wait for a clock-driven branch (the seam is missing or unused).
- A test that sets an env var to drive production behavior (env reader is not a seam).
- A `*http.Client` constructed inside a domain function.
- An interface in `internal/<domain>/` whose method set mirrors `*sql.DB` verbatim — the seam is leaking the implementation.
- Two seams with overlapping surfaces (e.g. `Clock` and a separate `TimeProvider`) — collapse them.
- A fake in `internal/testutil/mocks/` that does not carry a `var _ <Seam> = ...` assertion.
- `SEAMS.md` rows that point to deleted files, or `// seam:` interfaces missing from `SEAMS.md`.
- A "test mode" boolean threaded through production code in lieu of a seam.

---

### 8. Safe Refactoring Guidelines

You may:
- introduce a new interface to replace an ambient call (`time.Now()` → `clock.Clock.Now()`), wire it through `server.Deps`, and ship a fake
- narrow an existing seam by removing an unused method
- extend an existing seam (add `Sleep` to `clock.Clock`) rather than introduce a parallel one
- move a fake from `internal/testutil/mocks/` to `internal/<domain>/mocks/` when it serves only one domain (coordinate with `boundary-of-responsibility-enforcement` for the zone change)
- add the seam-registry test, or extend it with new tag patterns

You must:
- preserve observable behavior; a seam introduction is a refactor, not a feature change
- update both `SEAMS.md` and the `// seam:` tag in the same loop
- ship the production impl and the fake together — a seam without a fake is L2 at best
- keep compile-time `var _ <Seam>` assertions on every impl
- record any seam that *should* exist but requires broader redesign in `PROBLEMS.md`

Challenge yourself before a move:
- Is this interface narrow enough that the fake is trivial to write?
- Would a second agent, reading `SEAMS.md` alone, find this seam and its fake?
- Is the seam declared in the consumer's package, or did I put it in the implementer's?
- Does the registry test fail if I delete the `// seam:` tag without updating `SEAMS.md`?

---

### **9. Output Expectations**

By the end of this loop, the scenario should:
- have a `Seam Registry` table in `scenarios/{{TARGET}}/docs/internal/SEAMS.md` listing every seam with its declaration site, production impl, fake, and reason for existing
- have all four ambient seams (clock, httpc, env, log) replaced in domain code, or a `PROBLEMS.md` entry naming the remaining offenders
- carry compile-time `var _ <Seam>` assertions on every production impl and every fake
- carry a seam-registry test that reconciles `// seam:` tags with `SEAMS.md` (or a `PROBLEMS.md` entry pointing at it as the next step toward L5)
- record unresolved seam drift in `PROBLEMS.md`, not in a standalone `SEAM_AUDIT.md`

Anchor every finding to those durable docs through `knowledge-observatory-tools`. **Do not create a standalone `SEAM_AUDIT.md` or revive the legacy `UNIT_TEST_ARCHITECTURE.md` pattern** — those formats are retired. The Seam Registry in `SEAMS.md`, the Zone Map in `ARCHITECTURE.md` (owned by `boundary-of-responsibility-enforcement`), and the deferred-drift list in `PROBLEMS.md` are the only durable surfaces. A one-off audit report is acceptable solely for a migration handoff and must carry an explicit retirement path back into those three docs.

Recommended `SEAMS.md` additions:

```markdown
## Seam Registry

| Seam | Declaration | Production Impl | Test Double | Why it exists |
|---|---|---|---|---|
| clock.Clock | internal/clock/clock.go | clock.System | testutil/mocks/clock.go (FakeClock) | Wall-clock primitives for deterministic tests |
| httpc.Doer | internal/httpc/doer.go | *http.Client | testutil/mocks/doer.go (FakeDoer) | Outbound HTTP substitution |
| <domain>.Repository | internal/<domain>/repository.go | <domain>.SqliteRepository | <domain>/mocks/repository.go | Persistence substitution |
| ... | ... | ... | ... | ... |

## Seam Maturity

| Seam | Level | Evidence | Remaining Drift |
|---|---|---|---|
```

For *where the file lives* and the import-graph rules that protect it, see `prompt-manager skill read boundary-of-responsibility-enforcement`.

Last updated: 2026-05-12

]]></skill>
  <skill id="boundary-of-responsibility-enforcement" name="Boundary of Responsibility Enforcement"><![CDATA[
## Steer focus: Boundary-of-Responsibility Enforcement

Prioritize **where code lives** in `scenarios/{{TARGET}}/`. This skill governs the directory and package layout — which folders own which responsibility, what may import what, and how cross-cutting concerns enter and leave each zone. Two agents working independently from the same docs should land code in the same place.

The destination is the React-Vite-template shape: `handlers/<domain>/` translates transport, `internal/<domain>/` owns transport-free domain logic, and `internal/{clock,database,httpc,httpx,middleware,module,server}/` hold business-vocabulary-free substrate. Drift is gated by an import-graph test in the spirit of `path:templates/scenarios/react-vite/api/internal/testutil/no_prod_import_test.go`.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read screaming-architecture-audit` — domain map, surfaces, and archetype vocabulary this skill assumes.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — domain map and zone ownership.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — interface registry (the *how-to-substitute* concern, owned by the seam skill).
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — unresolved boundary drift.

---

> **Template example domain — delete on generation.** The react-vite template ships a `notes` domain (`path:templates/scenarios/react-vite/api/internal/notes/`) as a fully-worked illustration of the canonical layout — transport-free service, repository interface, schema, mocks, attachments sub-flow. It is *not* a domain every scenario inherits. When you generate from the template, delete `notes` and stand up your scenario's real domains in the same shape. This skill uses `<domain>` (lowercase package), `<Domain>Service`, `<Resource>` (singular), and `<substrate>` (e.g. `clock`, `httpc`) as placeholders. Encountering one in the audit is the cue to ask whether a leftover `notes` package is product vocabulary or template residue.

---

### 1. Scope Boundaries

**In scope:**
- top-level zones: `handlers/`, `internal/<domain>/`, `internal/<substrate>/`, `cmd/`, `main.go`
- which directories may import which (e.g. `internal/<domain>/` must not import `connectrpc.com/connect`, `gorilla/mux`, or sibling domains)
- placement of cross-cutting concerns (clock, logger, env reader, http client) at the composition edge — `main.go`, `server.Deps`, handler constructors — never reached for from inside a domain package
- enforcement primitives: import-graph tests, `// boundary:` tags, build tags, or `gen-endpoints`-style drift gates
- documenting the zone inventory in `ARCHITECTURE.md`

**Out of scope:**
- *how* a dependency is substituted in tests — that is the `seam-discovery-and-enforcement` concern (interface design, fakes, registration in `SEAMS.md`)
- API wire shape, proto coverage, REST exceptions — use `api-steer`
- product capability identification, archetype assignment — use `screaming-architecture-audit`
- UI feature layout and CLI command grouping (the analogous boundary question on the consumer surfaces — same principles, different folders)

**Decision rule for the reader.** Is the question *where does this code live?* → this skill (boundary). Is the question *how is this dependency substituted?* → `seam-discovery-and-enforcement` (seam). The two skills are paired and cite each other; a refactor that crosses the line must cite both. (Seam-side observations encountered during a boundary audit go to `SEAMS.md`, not into a parallel boundary doc.)

---

### 2. Boundary Maturity Model

Score each major zone (API, UI, CLI) independently. The level is the lowest verifiable artifact that still holds.

| Level | What exists | Verifiable signal | When to stop |
|---|---|---|---|
| 0 | Mixed concerns; handlers contain SQL or business rules; domain types import transport packages. | `rg "database/sql|sqlx|gorm" handlers/` returns hits, or `rg '"connectrpc.com/connect"' internal/<domain>/` returns hits. | Never — L0 is a finding, not a target. |
| 1 | Zone inventory recorded in `ARCHITECTURE.md`: each top-level folder under `api/` is named and assigned to one of {transport, domain, persistence, cross-cutting substrate, app entrypoint}. | `ARCHITECTURE.md` has a "Zone Map" section listing every directory under `api/`; `ls api/` matches one-to-one. | The scenario has more than three undocumented top-level packages. |
| 2 | Folder shape matches the template: `handlers/<domain>/` + `internal/<domain>/` + at least one of `internal/{clock,database,httpc,httpx,middleware,module,server}/`. Handler files contain no SQL. | `fd -t d . api/handlers \| sort` and `fd -t d . api/internal \| sort` align with the template; `rg "database/sql" api/handlers/` is empty. | The scenario has at most one domain. |
| 3 | Domain layer is transport-free. | `rg '"connectrpc\.com/connect"\|"net/http"\|"github.com/gorilla/mux"' api/internal/<domain>/` returns zero hits, encoded as a Go test (see `path:templates/scenarios/react-vite/api/internal/testutil/no_prod_import_test.go` for the pattern). | A new domain has not yet been ported. |
| 4 | Cross-cutting substrate is injected, not ambient. No `time.Now()`, `os.Getenv()`, `http.DefaultClient`, or `log.Default()` inside `api/internal/<domain>/`. Injection points are `main.go` and `server.Deps`. | An import-graph test asserts the forbidden-symbol set is empty in domain packages; CI fails on a new offender. | The substrate set is unstable (still discovering which seams the scenario needs). |
| 5 | Drift-gated. `ARCHITECTURE.md` declares every owned zone; a reconciliation test fails when a new top-level package appears without a `Zone Map` entry, when a domain imports another domain, or when handlers import persistence drivers directly. | A test analogous to `TestNoProductionImports` runs in `go test ./...`; passing CI is the artifact. | This is the destination. |

Use the level to pick the next concrete move: document the zone map, move SQL out of handlers, write the import-graph test, add the substrate exclusion, register the zone reconciliation.

---

### 3. Zone Decision Model

Given an arbitrary file or symbol, the canonical zone is determined by **what would break if the file disappeared**, not by what it imports today.

| Zone | Belongs here when | Canonical path | May import |
|---|---|---|---|
| Transport edge | Translates wire format ↔ domain types; owns request validation, auth interceptor wiring, error envelope mapping | `api/handlers/<domain>/` | proto generated code, `connect`, `internal/<domain>`, `internal/module`, `internal/httpx` |
| Domain core | Encodes product rules and lifecycle; would have to exist even if the scenario shipped without HTTP | `api/internal/<domain>/` | sibling files in same domain, `internal/clock`, `internal/httpc` (as interfaces), standard library |
| Persistence | Concrete adapter that materializes a `Repository` interface from `internal/<domain>/repository.go` | `api/internal/<domain>/<domain>_sqlite.go` (or `_postgres.go`) | `internal/database`, `database/sql`, driver packages |
| Cross-cutting substrate | Generic mechanics with no product vocabulary; used by unrelated domains | `api/internal/{clock,database,httpc,httpx,middleware,module,server}/` | standard library, other substrate packages |
| Composition root | Wires concrete substrate impls into `server.Deps`, registers `Module`s | `api/main.go`, `api/internal/server/` | every other zone |
| CLI / cmd | Operator tools that should not be in the server binary | `api/cmd/<tool>/` | substrate, domain, generated proto |

Decision sequence:

```text
Does the file translate an HTTP request to a typed call?           -> transport edge
Does it encode rules that would survive a transport change?        -> domain core
Does it speak a driver dialect (SQL, S3, gRPC)?                    -> persistence adapter inside the owning domain
Does it have no product vocabulary and is used by 2+ domains?      -> cross-cutting substrate
Does it construct concretes and pass them into Module/Deps?        -> composition root
```

If a file currently lives in the wrong zone, the move is mechanical: extract the misplaced symbol, place it in the right zone, update imports. Most boundary violations come from a single mis-zoned file, not architectural disagreement.

---

### 4. Canonical File-Shape

For React-Vite-template scenarios, the healthy target is:

```text
api/
  main.go                          # composition root: constructs substrate, calls server.New(deps), wires Modules
  handlers/<domain>/               # Connect service impl + Module constructor + EndpointDescriptors
  internal/<domain>/               # domain types, service, repository interface, schema, workflows
    <domain>_sqlite.go             # persistence adapter (driver dialect lives here, not in handlers)
    mocks/                         # test-double impls of this domain's interfaces (seam concern, paired)
  internal/clock/                  # ambient-seam: wall-clock primitives
  internal/httpc/                  # ambient-seam: outbound HTTP
  internal/database/               # shared DB plumbing (migrations, connection pool)
  internal/httpx/                  # shared HTTP middleware, error envelope, request id
  internal/middleware/             # auth / interceptor plumbing
  internal/module/                 # EndpointDescriptor + RESTException contract
  internal/server/                 # Deps struct, server.New, route registration
  internal/testutil/               # shared test helpers (seam concern, paired)
  cmd/<tool>/                      # operator binaries (gen-endpoints, migrate, …)
```

Invariants:
- `internal/<domain>/` imports no transport package and no sibling domain. Cross-domain coordination happens in `handlers/` or in a deliberate orchestration domain.
- Persistence adapters live *inside* the owning domain, not in a global `internal/persistence/` bucket. The interface is in the domain; the driver-specific file is alongside it.
- `main.go` is the only place that constructs concrete substrate (`clock.System{}`, `&http.Client{}`, `database.Open(...)`). Every other site receives them through `server.Deps` or a handler constructor.
- `internal/testutil/` and `internal/<domain>/mocks/` exist for tests; production code must not import them. Enforce with `no_prod_import_test.go`.

---

### 5. Audit Workflow

1. **Read `ARCHITECTURE.md`.** Treat its Zone Map as a claim. If it is missing, the scenario is at most L0.
2. **List the top-level packages.** Compare `fd -t d . api/internal -d 1` and `fd -t d . api/handlers -d 1` against the Zone Map. Every directory must have a documented zone assignment.
3. **Run the forbidden-import greps.** Each hit is a concrete L2→L3 task:
   ```bash
   rg "database/sql|jmoiron/sqlx|gorm.io" scenarios/{{TARGET}}/api/handlers --type go
   rg '"connectrpc\.com/connect"|"net/http"|"github.com/gorilla/mux"' scenarios/{{TARGET}}/api/internal --type go -g '!*_test.go' -g '!internal/{httpx,httpc,server,middleware,module}/**'
   rg "time\.Now\(\)|os\.Getenv\(|http\.DefaultClient|log\.Default\(\)" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!internal/{clock,httpc,server}/**'
   ```
4. **Check cross-domain imports.** `rg '"{{MODULE}}/internal/[^/"]+"' scenarios/{{TARGET}}/api/internal/<domain>/` — a domain importing a sibling domain is a boundary smell.
5. **Verify the enforcement test runs.** `cd scenarios/{{TARGET}}/api && go test ./internal/testutil/...`. If `no_prod_import_test.go` does not exist yet, vendoring it from the template is the first move toward L5.
6. **Assign maturity per zone.** API, UI, CLI score independently.
7. **Find drift.** Generic buckets named `utils/`, `common/`, `helpers/`; god-files mixing transport and persistence; domains importing each other.
8. **Update docs.** Zone Map changes land in `ARCHITECTURE.md`; unresolved violations land in `PROBLEMS.md`.

---

### 6. Red Flags

- A `handlers/<domain>/<domain>.go` file containing `db.Query(...)` or `sql.Open(...)`.
- An `internal/<domain>/` file importing `connectrpc.com/connect`, `net/http` (outside the `httpc` seam), or `github.com/gorilla/mux`.
- A `time.Now()` / `os.Getenv()` / `http.DefaultClient` call inside a domain package.
- A top-level `internal/utils/`, `internal/common/`, or `internal/helpers/` bucket — these absorb product vocabulary and erode the Zone Map.
- A persistence adapter living under `internal/persistence/<domain>/` instead of `internal/<domain>/<domain>_sqlite.go`.
- Production files importing `internal/testutil/...` or `internal/<domain>/mocks/...`.
- New top-level packages appearing in `api/internal/` without a `Zone Map` entry.

---

### 7. Safe Refactoring Guidelines

You may:
- move a file to its correct zone and update its package declaration and imports
- extract SQL out of a handler into the owning domain's `<domain>_sqlite.go`
- introduce an interface in the domain so a transport-leaking call can be moved behind the substrate boundary (coordinate with `seam-discovery-and-enforcement` for the interface design)
- add the `no_prod_import_test.go` pattern, or extend it with new forbidden prefixes
- rename a generic bucket folder to a substrate or domain name

You must:
- preserve observable behavior; boundary moves are mechanical, not redesigns
- update the Zone Map in `ARCHITECTURE.md` in the same loop as the directory change
- keep cross-cutting injection consistent — if you remove an ambient call, the dependency must arrive through `server.Deps` or a handler constructor
- record any deferred move (e.g. "domain X still imports domain Y") in `PROBLEMS.md` rather than half-applying it

Challenge yourself before a move:
- Would a second agent, given only `ARCHITECTURE.md`, place this file in the same folder I did?
- Does the move tighten an import-graph rule, or does it just relocate the problem?
- Is the substrate package I am creating actually generic, or is it one domain's helpers in disguise?

---

### **8. Output Expectations**

By the end of this loop, the scenario should:
- have a `Zone Map` in `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` listing every directory under `api/` with its zone assignment
- have no transport imports in domain packages (or a recorded `PROBLEMS.md` entry naming the remaining offenders and a removal plan)
- have no ambient `time.Now()` / `os.Getenv()` / `http.DefaultClient` calls in domain packages
- carry the `no_prod_import_test.go` enforcement test (or an equivalent zone-reconciliation test) running in CI
- record unresolved boundary drift in `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`, not in a standalone `BOUNDARY_AUDIT.md`

Anchor every finding to one of those three durable docs through `knowledge-observatory-tools`. **Do not create a standalone `BOUNDARY_AUDIT.md` or revive the legacy `UNIT_TEST_ARCHITECTURE.md` pattern** — those formats are retired. The Zone Map in `ARCHITECTURE.md`, the seam registry in `SEAMS.md` (owned by `seam-discovery-and-enforcement`), and the deferred-drift list in `PROBLEMS.md` are the only durable surfaces. A one-off audit report is acceptable solely for a migration handoff and must carry an explicit retirement path back into those three docs.

Recommended `ARCHITECTURE.md` additions:

```markdown
## Zone Map

| Directory | Zone | May Import | Enforcement |
|---|---|---|---|
| api/handlers/<domain>/ | transport edge | proto gen, connect, internal/<domain>, internal/module, internal/httpx | no_prod_import_test |
| api/internal/<domain>/ | domain core | stdlib, internal/clock, internal/httpc (interfaces) | no_prod_import_test |
| api/internal/clock/ | substrate | stdlib | — |
| ... | ... | ... | ... |

## Boundary Maturity

| Zone | Level | Evidence | Remaining Drift |
|---|---|---|---|
```

For *how* the substituted-in interfaces are designed and registered, see `prompt-manager skill read seam-discovery-and-enforcement`.

Last updated: 2026-05-12

]]></skill>
  <skill id="invariant-discovery-and-enforcement" name="Invariant Discovery &amp; Enforcement"><![CDATA[
## Steer focus: Invariant Discovery & Enforcement (Behavior Definition & Verification)

Make every rule the code relies on **formally declared, anchored to code, and mechanically verified**. Invariants are the load-bearing facts of a scenario; this skill ensures they are stated where they belong (local to code, or additionally in the cross-cutting registry), backed by a real enforcement mechanism, and exercised by tests. The skill succeeds to the extent that intended behaviors and enforced behaviors are the same set.

An invariant is a condition that **must always be true** for the system to behave correctly: "this list is always sortable by date", "this route requires an authenticated user", "this account balance is never negative", "every `EndpointDescriptor.Path` is either a Connect procedure or carries a `RESTException`". Bugs are violations of invariants.

This skill replaces the retired `assumption-mapping-and-hardening` skill. An "assumption" is just an invariant that isn't yet enforced — both states are tracked here.

Do **not** turn current bugs, temporary workarounds, or incomplete behavior into "invariants." Stabilize true rules, do not freeze accidental behavior.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read boundary-of-responsibility-enforcement` — owns directory shape and per-layer responsibility; this skill assumes domain ownership is already clear.
- `prompt-manager skill read seam-discovery-and-enforcement` — owns substitution points; invariants at a seam ("fake and production agree on shape") are seam-skill territory.
- `prompt-manager skill read temporal-flow-audit` — owns ordering/lifecycle invariants; this skill defers temporal rules there and only catalogs their existence.
- `prompt-manager skill read interoperability-steer` — owns proto + `protovalidate` as the canonical boundary-validation mechanism this skill points to.
- `prompt-manager skill read error-semantics-recovery-path-design` — owns failure-mode design when an invariant's violation must degrade gracefully rather than fail.

Read first when present:
- `scenarios/{{TARGET}}/docs/internal/INVARIANTS.md` — prior invariant registry and enforcement status. **Code is the source of truth; verify claims against code before extending.**
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — domain map; cross-cutting invariants may originate here.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — known enforcement gaps deferred for later; soften-resolution decisions also land here.

Optional context:
- `docs/scenario-qa/methods/audit/invariant-discovery-and-enforcement.md` — when this lens applies, when it backfires, what the qa-contrarian challenges.

---

> **Template example domain — delete on generation.** The react-vite template ships a `notes` domain as a worked example. It is *not* a real domain every scenario inherits. Examples below use placeholder identifiers (`<domain>`, `<Resource>`, `<Flow>`); substituting them is the moment to check whether a leftover `notes` folder is template residue.

---

### 1. Scope Boundaries

**In scope:**
- discovering rules the code relies on (whether currently enforced or not)
- triaging each discovered rule into Enforce / Soften / Accept
- choosing the right enforcement mechanism per archetype (type / `protovalidate` / runtime guard / DB constraint / test / registry)
- declaring rules in code with the `// INVARIANT: <name>` tag at the enforcement site
- adding or strengthening violation-exercising tests
- maintaining `INVARIANTS.md` for cross-cutting (Tier 2) invariants only
- recording unenforced invariants as explicit gaps and softened-out rules as accepted-resolutions in `PROBLEMS.md`

**Out of scope (hand off):**
- directory shape and layer ownership → `boundary-of-responsibility-enforcement`
- seam shape and test substitution → `seam-discovery-and-enforcement`
- transition tables and lifecycle ordering → `temporal-flow-audit` (this skill catalogs that the temporal invariant exists)
- proto schema design and codegen workflow → `interoperability-steer`
- failure-mode design for soft-failure paths → `error-semantics-recovery-path-design`
- broad rewrites or new features under the banner of invariant cleanup
- creating standalone `INVARIANT_AUDIT.md`, `ASSUMPTIONS.md`, or per-domain registry files — the single INVARIANTS.md registry is the durable surface

---

### 2. Two-Tier Model: Local vs. Registry

Every invariant lives at one of two tiers. The tag convention and enforcement mechanism are the same at both tiers; the difference is whether the rule is *additionally* indexed in `INVARIANTS.md`.

**Tier 1 — Local invariant.** Scope is one function, one file, or one tightly-bounded module. Declaration and enforcement both live with the code:

- `// INVARIANT: <name>` tag on the guard, type, annotation, constraint, or validator that embodies the rule (this tag is the universal marker)
- Mechanism is one of: type encoding, `protovalidate` annotation, runtime guard, DB constraint, codegen check, or a violation-exercising test
- **No `INVARIANTS.md` entry required.** Adding one would create drift risk with no readership benefit.

**Tier 2 — Registry-worthy invariant.** Scope spans domains, layers, or has system-wide consequences. Declaration in code (same tag, same mechanism) AND a row in `INVARIANTS.md` so cross-cutting agents can discover the rule without reading every file.

**Tiering criterion — register in INVARIANTS.md when one or more is true:**
- Multiple domains depend on the rule (e.g., "every entity ID is a UUIDv7")
- Layers must agree (UI assumes shape X, API enforces shape X, DB constrains shape X — drift here is silent corruption)
- Violation has system-wide impact: security boundary, billing integrity, data corruption, irreversible state change
- Enforcement is itself cross-cutting (interceptor, codegen validator, registry drift check) — the mechanism touches code that doesn't otherwise have a reason to know about the rule

Otherwise the rule is local. Local rules can graduate to Tier 2 when their scope expands; graduation is just adding the row to `INVARIANTS.md` and ensuring the code tag and doc row agree on the name.

---

### 3. Triage Taxonomy

Before choosing a mechanism, classify every discovered rule into exactly one resolution:

| Resolution | When to choose | Where it lands |
|---|---|---|
| **Enforce** | The rule should always hold and the code already relies on it (or should). Pick a mechanism from §6 and encode it. | Code tag at the enforcement site; test exercising violation; Tier-2 rules additionally appear in `INVARIANTS.md`. |
| **Soften** | The rule is fragile or unrealistic; the right move is to change the code so it no longer *depends* on the rule (guard, default, fallback, graceful degradation). | Code change (no tag, since there's no longer an invariant). Record the decision in `PROBLEMS.md` under "Softened-out rules" with rationale. |
| **Accept** | The rule should hold, enforcement would be costly, and the residual risk is judged acceptable (cost-of-enforcement > impact × likelihood). | `INVARIANTS.md` Gaps section with rationale, impact, and trigger condition for revisiting. |

Rules:
- Every triaged rule has exactly one resolution. "TBD" entries are a code smell — either you haven't decided, or the rule isn't ready to be triaged.
- `Accept` is not a synonym for `Defer`. Deferring is just Accept with a written trigger ("revisit when concurrency model changes"). Both live in the same Gaps section.
- A rule resolved as `Soften` does **not** appear in `INVARIANTS.md` (it's no longer an invariant). The decision lives in `PROBLEMS.md` so future agents can see why the dependency was removed.

---

### 4. The `// INVARIANT:` Tag Convention

The tag is the universal marker that declares an invariant in code, at both tiers.

**Format:**

```
// INVARIANT: <camelCaseName>
```

- **`INVARIANT:` in uppercase** — matches the house convention (`KNOWLEDGE-OBSERVATORY` doc-anchor style).
- **`<camelCaseName>`** — scope-prefixed by the owning domain noun when local (e.g., `noteIDIsImmutable`), or no prefix when truly cross-cutting (e.g., `everyEndpointIsConnectOrTaggedRESTException`).
- The name **reads as an assertion of what is true**, not a wish ("should be") and not a mechanism ("hasNoSetter"). It describes the rule, not how it's enforced.

**Placement: on the line that carries the rule.** The tag sits *at the enforcement site* — the type definition, the proto field annotation, the guard `if` statement, the SQL `ALTER TABLE`, the codegen validator function. The rule of thumb: "the line you'd have to bypass to violate this rule."

Right placement (tag on the guard line itself):

```go
func (s *Service) Update(ctx context.Context, id NoteID, in UpdateInput) (*Note, error) {
    // INVARIANT: noteUpdateRequiresOwnership
    if !s.auth.CallerOwns(ctx, id) {
        return nil, ErrNotOwner
    }
    // ...
}
```

Wrong placement (vague tag above the function — a verifier cannot tell what line is "the rule"):

```go
// INVARIANT: noteUpdateRequiresOwnership   ← do not do this
func (s *Service) Update(...) (*Note, error) { ... }
```

**Test linkage:** for each tag, a sibling test file contains a test whose name or comment references the same invariant name. The common pattern:

```go
// enforces invariant: noteUpdateRequiresOwnership
func Test_Update_RejectsNonOwner(t *testing.T) { /* ... */ }
```

For invariants whose enforcement *is* the build (codegen validators, type-system encodings), the build step itself is the evidence — no separate test required.

---

### 5. Invariant Maturity Ladder (Per Invariant)

The ladder applies **per invariant**, not per scenario. A scenario may have one L5 invariant and ten L1 invariants. Move only as far as risk and time justify. L5 differs between tiers.

| Level | Name | Tier 1 (Local) | Tier 2 (Registry-worthy) |
|---|---|---|---|
| L0 | Implicit | Rule lives only in someone's head; rediscovered each loop. | Same. |
| L1 | Tagged | `// INVARIANT: <name>` on the enforcement site with a one-line prose rationale near it. | Tagged in code AND entered in `INVARIANTS.md` with prose statement. |
| L2 | Code-anchored | Tag sits on the guard/type/check that actually embodies the rule (not floating). | Same, plus the doc row carries the `path:file:line` anchor. |
| L3 | Test-enforced | A violation test exists; its name or comment references the invariant name. | Same. |
| L4 | Mechanism-declared | The enforcement mechanism is named (`type-system` / `protovalidate` / `runtime-guard` / `db-constraint` / `test` / `registry`) and that mechanism actually exists in code. | Same; the doc row's `Mechanism` column matches code reality. |
| L5 | Drift-gated | Local convention (lint rule, grep target, or convention-check) catches new code that violates the tag's contract. | Registry drift gate: every `INVARIANT:` tag at Tier-2 scope appears in `INVARIANTS.md` and vice versa, validated by a registry test analogous to `validateTransport` (`path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go`). |

The level is not a vanity score. It tells the next agent what kind of drift is still possible for this specific invariant.

---

### 6. Invariant Archetype → Enforcement Mechanism

Pick the mechanism by the *shape* of the rule, not by what is most convenient. Each row is a decision shortcut; the right answer is usually the leftmost mechanism that can carry the rule.

| Archetype | Example | Preferred mechanism | Fallback | Cross-link |
|---|---|---|---|---|
| Type / shape | "this field is non-empty"; "this enum has exactly N values" | Type system (non-nullable, discriminated union, sealed enum) | `protovalidate` annotation at boundary; test | `interoperability-steer` |
| Range / value | "value is between 0 and 100"; "string matches regex" | `protovalidate` constraint on proto field | Runtime guard at the boundary entrypoint | `interoperability-steer` |
| Ordering / temporal | "X must happen before Y"; "terminal states cannot be escaped" | State machine + transition function in the owning domain | Trace tests over the workflow | `temporal-flow-audit` |
| Reference / referential integrity | "foreign key always resolves"; "owner of resource still exists" | DB foreign-key constraint + cascade rule | Service-level guard with test | — |
| Identity / stability | "ID is stable across runs"; "IDs are globally unique" | Type (opaque ID) + DB uniqueness constraint | Test asserting stability across reload | — |
| Cross-aggregate / invariant sum | "sum of debits = sum of credits"; "count(active children) ≤ parent.cap" | DB constraint or transactional check | Scheduled reconciliation job + test | — |
| Cross-surface parity | "every `EndpointDescriptor.Path` is Connect or has `RESTException`" | Registry validator (codegen-time check) | Generated-artifact diff test | `api-steer` / `interoperability-steer` |
| Authorization / ownership | "caller must own resource"; "admin-only operation" | Interceptor / middleware at the boundary | Per-handler guard with test | — |

Decision rule:

```text
Can the type system make the illegal state unrepresentable?
  YES -> encode in types; tests guard the conversion boundary only.
Can a declarative annotation (protovalidate, DB constraint) carry the rule?
  YES -> declare it there; tests assert the annotation is present and enforced.
Does correctness depend on history or ordering?
  YES -> hand off to temporal-flow-audit; record only the existence here.
Does the rule cross artifacts (code <-> registry <-> generated file)?
  YES -> add a drift test at codegen or boot time (L5 pattern).
Otherwise:
  Encode as a named runtime guard + an enforcement test.
```

A real example of this pattern, lit up end-to-end, is the `RESTReason` enum at `path:templates/scenarios/react-vite/api/internal/module/module.go` paired with the `validateTransport` check at `path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go`. The invariant ("every endpoint is either Connect or a tagged REST exception") is type-encoded (closed enum), registry-enforced (validator iterates all endpoints), and drift-gated (codegen fails on violation). That is what L5 Tier-2 looks like.

---

### 7. Discovery: Finding Rules the Code Already Relies On

Discovery is **archaeology, not invention.** An invariant is a rule the code *currently relies on*. If you can't find code that relies on it, it's not an invariant — it's a PRD requirement (belongs in the PRD) or speculation (don't record).

**Earn-the-entry rule:** include only rules whose violation would produce a bug class the team would want to prevent forever. Micro-preconditions that "just need a fix and a regression test" stay in the test suite, not the registry.

**Per-domain density signal** (sanity check, not a target):

| Domain weight | Expected order of magnitude (Critical invariants) |
|---|---|
| Pure CRUD / display | 2–4 |
| Temporal / orchestration | 6–12 (state-machine ones hand off to TEMPORAL-FLOWS; a one-line existence entry remains) |
| Security / billing / auth | 10+ |
| Pure config / static | 0–2 (sometimes legitimately none) |

Zero in a high-risk domain is a discovery gap. Thirty in a CRUD domain is conflation with implementation details — promote the meta-rules, demote the specifics.

**Static signals in code:**
- repeated guards or assertions at the start of functions ("this value is never null here")
- branches that assume a particular shape and `panic` / `throw` / return error on others
- comments containing "must", "always", "never", "invariant", "assume", "precondition", "postcondition"
- error types that name a rule violation (`ErrOwnerMismatch`, `ErrAlreadyClaimed`)
- type assertions and unchecked casts that work because the producer guarantees the shape
- `// TODO` / `// HACK` near validation paths

**Discovery fingerprints absorbed from the retired `assumption-mapping-and-hardening` skill:**
- **Data shape & nullability assumptions**: properties accessed without prior null checks; lists iterated without empty-check where empty would be ambiguous
- **External-system assumptions**: response fields assumed always present; assumed latency, availability, or error behaviors; webhook ordering assumptions
- **Timing / environment assumptions**: functions assuming call order; components assuming mount order; assumed env vars or runtime modes
- **User-behavior assumptions**: assumed input quality ("they won't paste huge blobs"); assumed permissions; assumed flows
- **Test-fixture assumptions**: tests relying on narrow happy-path fixtures that encode an assumption not enforced in production logic

Each fingerprint is a *candidate* invariant. Triage per §3 before encoding.

**Domain signals:**
- PRD or operational targets that say "must", "never", "exactly", "at most N"
- existing tests with names like `Test_RejectsNegativeBalance`, `Test_DuplicateIDFails`
- finite enums and discriminated unions
- DB unique / check / foreign-key constraints already in migrations
- `protovalidate` annotations already present

**Useful greps (run from `scenarios/{{TARGET}}/`):**

```bash
# named invariant tags already in the codebase
rg -n 'INVARIANT:\s*\w' --type-add 'src:*.{go,ts,tsx,py,rs,sql,proto}' -tsrc

# guards and assertions that imply an unwritten rule
rg -n '\b(assert|invariant|require|must|panic\()' -tsrc

# DB constraints declared in migrations vs claims in INVARIANTS.md
rg -n '(UNIQUE|FOREIGN KEY|CHECK \()' --type sql

# hand-rolled validation that should be protovalidate at the proto boundary
rg -n 'len\([^)]+\)\s*==\s*0|strings\.TrimSpace\([^)]+\)\s*==\s*""' api/internal

# mutators that change state without an explicit guard
rg -n 'func.*\b(Update|Set|Adjust|Apply|Increment|Decrement)\w*\b' api/internal

# claims in INVARIANTS.md that reference no code
rg -n '^\s*-\s' docs/internal/INVARIANTS.md
```

Treat each match as a *candidate*. Promote only if the rule aligns with PRD intent, real usage, and existing tests. Demote incidental quirks into `PROBLEMS.md`.

---

### 8. Programmatic Verification: What Is and Isn't Checkable

This skill produces tag conventions and registry shape designed to support increasingly programmatic verification over time (tidiness-manager smart-scan or a dedicated CLI as the substrate).

**Can be programmatically checked (day-one floor):**
1. **Tag format** — `// INVARIANT: <camelCaseName>` on a comment line. Catches typos like `// Invariant:`, missing colon, inconsistent case.
2. **Name uniqueness** — no two unrelated tags share a name.
3. **Test linkage** — every invariant name appears in at least one test name or test comment. For codegen-validator invariants, the build step substitutes for the test.
4. **Mechanism evidence** — the tagged line is structurally a guard / type / annotation / constraint / validator (not a random comment).
5. **Registry parity (Tier 2 only)** — every Tier-2 tag appears in `INVARIANTS.md` and vice versa, with the doc's `path:` resolving near the tag.

**Cannot be programmatically checked (stays human/AI judgment):**
1. Whether the prose statement is true (matches actual behavior).
2. Whether the mechanism is sufficient (a runtime guard might be in place when a DB constraint is actually needed).
3. Whether the test exercises the *violation* path, not just the happy path.
4. Whether the rule is correctly classified as Tier 1 vs Tier 2.
5. Whether the right mechanism was chosen from §6.

Day-one programmatic value: structural drift, name uniqueness, test linkage, registry parity. Smart-scan can grow heuristics for the harder checks over time. Every recurring manual finding is a candidate for promotion into the programmatic surface.

---

### 9. Audit Workflow

1. **Read the registry.** Open `docs/internal/INVARIANTS.md` if present. List every claimed Tier-2 invariant; for each, the next step is to verify or refute.
2. **Verify against code.** For each entry, follow it to the guard/type/constraint. If the code does not match the claim, the doc is wrong — fix the doc or fix the code, never let the mismatch stand.
3. **Walk discovery per domain.** Use §7 signals and greps. For each domain: existing tags → guards/errors → tests → declarative annotations → PRD cross-reference. Stop when each domain has been walked at the current pass's risk threshold (Critical first, Important next if time permits).
4. **Triage each candidate (§3).** Pick exactly one resolution: Enforce, Soften, or Accept.
5. **For Enforce candidates: classify the archetype (§6) and pick the mechanism.** Prefer making illegal states unrepresentable (types) over runtime guards over tests.
6. **For Enforce candidates: decide the tier.** Use the §2 criterion. Default to Tier 1 unless one of the four registry-worthy conditions is met.
7. **Encode the invariant.**
   - Add `// INVARIANT: <name>` at the enforcement site (§4).
   - Add or strengthen the violation test; the test name or comment references the invariant.
   - For boundary input rules, prefer a `protovalidate` annotation over a hand-rolled handler check.
   - For cross-artifact rules, add a codegen / boot-time validator following the `validateTransport` pattern.
8. **For Soften decisions: change the code so the rule is no longer required.** Record the decision in `PROBLEMS.md` under "Softened-out rules" with rationale.
9. **For Accept decisions: record in `INVARIANTS.md` Gaps section** with risk, rationale, and trigger condition.
10. **Reconcile the registry.** Tier-2 entries each have name, prose statement, `path:file:line` anchor, declared mechanism, and pointer to enforcing test.

Do not introduce new noisy or user-hostile failures while encoding invariants. Protect correctness without degrading UX (see `error-semantics-recovery-path-design` for soft-failure shapes).

---

### 10. Canonical File Shape for `INVARIANTS.md`

The doc is a **registry**, not a deep prose document. Keep it short, structured, and code-anchored. At L3+ it has four sections. For scenarios with more than ~15 Critical entries, group within each section by domain.

```markdown
# Invariants

## Critical Invariants

| Name | Statement | Mechanism | Code Anchor | Enforcing Test | Notes |
|---|---|---|---|---|---|
| ownerMustMatchCaller | A user can only mutate resources they own. | runtime-guard + test | path:api/internal/<domain>/service.go:142 | path:api/internal/<domain>/service_test.go:Test_RejectsNonOwnerUpdate | Pre-protovalidate; candidate to lift to interceptor. |
| balanceNeverNegative | Account balance must never drop below zero. | db-constraint + test | path:api/migrations/0007_balance_check.sql | path:api/internal/ledger/ledger_test.go:Test_RejectsNegativeBalance | — |
| restPathHasReason | Every EndpointDescriptor.Path is Connect or tagged with RESTException. | registry | path:api/cmd/gen-endpoints/main.go:validateTransport | codegen fails | L5 pattern. |

## Enforcement Mechanisms

| Mechanism | Where it lives | What it catches | What it does NOT catch |
|---|---|---|---|
| type-system | proto messages, Go types, TS discriminated unions | shape / nullability / closed enums | range, cross-field, cross-row |
| protovalidate | `path:packages/proto/schemas/{{SCENARIO_ID}}/v1/.../*.proto` | range, regex, required, value constraints at ingress | runtime-only or cross-aggregate rules |
| runtime-guard | service / handler entrypoints | preconditions that types cannot express | rules better moved to protovalidate |
| db-constraint | migrations | uniqueness, FK, range CHECK, transactional invariants | application-layer rules |
| test | `*_test.go` / `*.test.ts` | regression of any of the above | invariants no one wrote a test for |
| registry | codegen validators (`validateTransport`-style) | cross-artifact parity drift | invariants within a single file |

## Important Invariants

Same table shape as Critical. Violation is recoverable (degraded UX, retriable) rather than corrupting.

## Gaps

| Invariant | Resolution | Why unenforced | Risk if violated | Trigger to revisit |
|---|---|---|---|---|
| uniqueWorkflowIdPerScenario | Accept | No DB constraint yet; only service-level check. | Duplicate IDs would silently overwrite. | When concurrent writers ship. |
```

Rules:
- Code is the source of truth. If the table claims something the code does not, the table is the bug.
- Anchors use `path:` references (machine-readable).
- Only Tier-2 (registry-worthy) invariants appear here. Tier-1 invariants live entirely in code via the `INVARIANT:` tag.
- Cross-cutting observations that change the mental model belong in `ARCHITECTURE.md`; unresolved drift and soften-resolutions belong in `PROBLEMS.md`. Keep `INVARIANTS.md` focused on the registry.

---

### 11. Migrating Scenarios with Existing `ASSUMPTIONS.md`

Some scenarios have an `ASSUMPTIONS.md` from the retired `assumption-mapping-and-hardening` skill. When you next audit such a scenario:

1. Read each entry in `ASSUMPTIONS.md`.
2. Triage per §3: Enforce, Soften, or Accept.
3. For Enforce: encode per §7 (tag at enforcement site + test); Tier-2 entries move into `INVARIANTS.md` Critical/Important.
4. For Soften: change the code, record in `PROBLEMS.md`.
5. For Accept: move into `INVARIANTS.md` Gaps with trigger.
6. Delete `ASSUMPTIONS.md` once empty.

Do not preserve `ASSUMPTIONS.md` as a parallel doc — the single registry is the durable surface, and the local `INVARIANT:` tag is the durable code marker. If a full migration in one pass is too large, leave `ASSUMPTIONS.md` in place with a header note pointing at this skill and record the migration as an entry in `PROBLEMS.md`.

---

### 12. Documentation

Use `knowledge-observatory-tools` to read and update stable docs.

- `docs/internal/INVARIANTS.md` — primary surface. Four-section schema above. Tier-2 only.
- `docs/concepts/ARCHITECTURE.md` — when the invariant changes the mental model ("the API is the source of truth for X"), record it there too with a back-link.
- `docs/internal/PROBLEMS.md` — unresolved enforcement gaps, softened-out rules with rationale, and the narrative behind risky deferred work.

Do not create one-off `INVARIANT_AUDIT.md` reports, parallel `ASSUMPTIONS.md` files, or per-domain registry files. The single registry is the durable surface.

---

### **13. Output Expectations**

By the end of this loop, the scenario should:
- have a verified `INVARIANTS.md` where every Tier-2 entry is code-anchored and test-enforced (≥ L3)
- have at least one invariant that moved up the maturity ladder
- prefer mechanism strength in this order: type-system > declarative annotation (`protovalidate` / DB constraint) > runtime guard > test
- have `// INVARIANT: <name>` tags in code at the enforcement site for every Critical and Tier-2 invariant
- record softened-out rules in `PROBLEMS.md` with rationale (not in `INVARIANTS.md`)
- record accepted gaps in `INVARIANTS.md` with trigger conditions
- avoid promoting incidental quirks or current bugs into the registry

Avoid superficial edits. The goal is not a longer document; it is a codebase where the rules that must hold are mechanically protected and a future agent cannot violate one by accident.

]]></skill>
</skills>