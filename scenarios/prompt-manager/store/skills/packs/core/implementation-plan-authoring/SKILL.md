## Practice focus: Implementation Plan Authoring

Author durable implementation plans through Plan Manager's guided authoring
runtime. Plan Manager is the plan-logic authority: it owns the section order
(the ten reader-question clusters), phase shape, validation gates, context
discovery execution, rendered markdown, and persistence. This skill supplies
only the judgment layer — when to plan, and how to decide well inside the
wizard.

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
