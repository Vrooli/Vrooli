## Practice focus: Implementation Plan Authoring

Author durable implementation plans through Plan Manager's guided authoring
runtime. Plan Manager is the plan-logic authority: it owns the section order
(the nine reader-question clusters), phase shape, validation gates, context
discovery execution, rendered markdown, and persistence. This skill supplies
only the judgment layer — when to plan, and how to decide well inside the
wizard.

Required reading:
- `scenarios/plan-manager/docs/concepts/PLAN-MODEL.md` — canonical structured
  plan and phase model.
- `scenarios/plan-manager/docs/reference/cli-commands.md` — current
  `plan-manager author`, `plans`, `exec`, `validate`, and `log` command surface.

---

### 0. Preserve the planning source, not just the task title

A plan is a durable compression of the investigation and discussion that led
to it. Compress repetition and incidental conversation; do **not** compress
away a decision, constraint, visual model, or acceptance expectation that a
fresh execution agent would need in order to make the same implementation
choices.

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

Then follow the API-owned next action until finalize:

```bash
plan-manager author continue "<session>"
```

The session is a form, not a stage-gated wizard: every response carries a
full-disclosure **checklist** (all requirements for the touched scope with
filled/missing/violation status). Read the checklist and submit any or all
fields **in any order**, batched when you already know the content:

```bash
plan-manager author submit "<session>" --set purpose="..." --set scope="..."   # sections in one call
plan-manager author phase-add "<session>" --title "..." --intent "..."   --set steps="..." --set validation="..." --set acceptance="..."            # add+fill a phase in one call
plan-manager author phase-submit "<session>" "<phase>" --set "<field>"="..." ...   # batch fields on an existing phase
```

Each batch item returns an accepted/rejected line naming exactly what was
parsed; a rejected item (unknown field, acceptance duplicating validation) is
NOT applied while the rest of the batch lands. A complete N-phase plan takes
≤ 3+N mutation calls. The `continue` loop remains available when you prefer
one recommended action at a time. The artifact renders in the fixed cluster
order (Purpose / Problem / Outcome / Approach & Decisions / Boundaries /
Assumptions & Risks / Verification / Execution Setup / Phases) regardless of
submission order. Global flags such as `--auto-start` go before the
subcommand. `author status <session>` is an alias of `author preview`.

Skill discovery is a low-friction bootstrap, not a curation workflow. Ask Plan
Manager to add the prompt-manager pack:

```bash
plan-manager author skill-pack "<session>" --concepts "<c1>,<c2>" --complexity "<minor|moderate|major|architectural>"
```

That command runs `prompt-manager discover --type skill --json`, auto-adds the
returned skills as global relevant context, and prints the read command (plus a
recommended read command when prompt-manager budgets the pack down). Keep most
returned skills unless they are clearly irrelevant; they are there to improve
professional execution, not only to match the task title literally.

For docs, records, code references, and other context, run search-hub directly
so you can inspect confidence and attribution yourself:

```bash
search-hub query "<intent>" --type record,doc,skill
```

Submit only durable context or references that will help a resumed agent:
`plan-manager author context-submit ...` for setup commands/notes, normal
reference fields for `[CODE:]`, `[DOC:]`, `[REQ:]`, or an honest
`NO_CODE_REFS: <reason>` when there are no useful code references. There is no
candidate accept/reject/apply step.

Review, validate, finalize:

```bash
plan-manager author preview "<session>"
plan-manager author validate "<session>"
plan-manager author finalize "<session>"
```

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
Replace the word with the specific behavior, path, or check you mean:
"robust", "comprehensive", "appropriately", "properly", "seamless",
"handle", "improve", "enhance", "leverage", "as needed".

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
- Specifically reject these lossy plans: a plan that says what to do but not
  why this design; names a component but omits the relevant interaction/data
  flow; lists phases without exact acceptance or validation; records a decision
  without its tradeoff; assumes context that exists only in chat; or leaves
  already-discovered facts behind an opaque “investigate as needed” step.
- The change boundary must name the paths the work may touch; do not hide scope
  in prose.
- `validation` is the method of checking; `acceptance` is the outcome gate.
  They must not be identical.
- Every fact lives in exactly one section: Purpose is an abstract (do not
  restate Problem or Outcome there); the Definition of Done carries plan-level
  gates only, never restated phase acceptances.
- Use `author skill-pack` early, then read the compact skill command it returns.
  Remove a discovered skill only when it is clearly irrelevant or harmful.
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

### 6. Output Expectations

**Must produce:**
- A finalized Plan Manager plan id/slug, or a clear explanation of why the
  authoring session remains unfinished. Finalize output names the physical
  SQLite store path, the stamped workspace, and a **computed** mirror status
  (`fresh`, or a loud `write_failed` warning with the repair command) —
  treat a `write_failed` mirror or a missing store path as something to
  report, and re-running finalize prints `Already finalized at <ts>`
- A rendered review command/path
- A concise note about degraded dependencies or manual fallbacks
- The next execution command when implementation should continue

In Candidate mode, ensure the candidate itself records the material source
context in the appropriate plan sections. Include a concise preservation note
only when the caller's typed result contract has a field for it; do not claim
finalization, validation, a plan id, or a rendered Plan Manager path.

**Must not produce:**
- A standalone hand-formatted markdown plan as the default artifact
- Placeholder-only phases or context entries
- Contradictory constraints or fabricated validation evidence
