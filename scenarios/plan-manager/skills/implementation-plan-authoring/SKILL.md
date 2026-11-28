---
name: "implementation-plan-authoring"
description: "Author durable implementation plans through Plan Manager's guided structured-plan runtime, preserving context, constraints, phases, validation strategy, and execution handoff"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["planning","handoff","implementation","documentation","execution"]
  icon: "file-text"
  status: "active"
  revision: 14
  createdAt: "2026-02-09T00:00:00Z"
  updatedAt: "2026-09-30T00:00:00Z"
  requires:
    scenarios: ["plan-manager", "prompt-manager", "vrooli"]
    commands: ["plan-manager", "prompt-manager discover", "prompt-manager skill read", "vrooli scenario"]
  origin:
    kind: "authored"
---
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

**Choose the plan shape first.** A plan has a shape: `phased` or `mandate`.
This skill authors phased plans: ordered phases, each with steps, acceptance
and validation. An adaptive mandate (a target pointer, sensors, bands, scope,
stop rules and a suggested arc, with no step list) is authored with
`prompt-manager skill read adaptive-mandate-authoring`. The rule for choosing
between them lives in `path:docs/agent-system/SCENARIO_DEVELOPMENT.md#work-shapes` §"Work
shapes"; read it before starting a session, and do not author a phased plan
over a scenario whose route is not yet clear.

---

### 0.0 Read prior-plan status correctly

Plan status is computed from phase state, not intent. For bounded inventory, run `program-runtime library run plan-manager.plan-inventory`; use `program-runtime library run plan-manager.plan-read --input id=<id-or-slug>` for its bounded status/phase summary; use owner `plans get` only when authored prose is needed. A stale plan is input to supersede, extend or reuse, not a reason not to author.

---

### 0. Preserve the planning source, not just the task title

For changes to documentation, architecture, requirements, experiences, skills or programs, read `docs/agent-system/SCENARIO_DEVELOPMENT.md` "Documentation-first authoring and application", author the concrete proposed artifacts, and use the plan for rationale, dependencies, authority and rollout. Discover Tech Tree Designer's actual draft capabilities before choosing it; availability or proto materialization does not establish support for arbitrary artifacts. Link supported draft identities and exact revisions through Plan Manager's existing context and reference fields (invent no schema fields); otherwise use the returned artifact directory. Edit canonical docs or skills directly only with explicit operator permission. When Plan Manager writes are forbidden, use the caller's artifact destination or return the content through its response contract; if neither can retain the artifacts, report the missing destination and do not start a session to obtain one. Before review, identify changed target artifacts, source bases, protected acceptance changes, draft validation and unresolved application conflicts, keeping desired behavior distinct from implemented behavior; executable prototypes need their own effects grant. For a scenario-development mandate, preserve the approved destination and delegated discretion rather than predicting every improvement as a phase.

Author for concurrent agents in one shared worktree using the completion policy in `PLAN-MODEL.md`: ordinary plans use advisory broad validation and optional baseline context, describe evidence for the actual outcome, and select certification only for an explicit requirement with a reason. No blanket green-suite, no-regression, isolation or quiescence requirements; missing baseline evidence stays unknown and historical reconstruction must not replace or stash current shared-worktree contents.

A plan is a durable compression of the investigation behind it: compress repetition and chatter, never a decision, constraint, visual model or acceptance expectation a fresh execution agent needs to make the same choices. Fill the optional **Definitions** section with `Term - meaning` lines for terms the plan coins or narrows (shared vocabulary goes to `docs/concepts/GLOSSARY.md`).

Before authoring, build a private inventory of the intended outcome, experience and non-goals; current behavior, evidence and constraints; the selected design, rejected alternatives and tradeoffs; diagrams, flows and interface contracts; invariants, boundaries, dependencies, assumptions, risks and open questions; and discovered facts, code, doc and requirement references and validation expectations. Optional inventories, submissions, snapshots and execution evidence go in the artifact directory the authoring step returns (the runtime-home `plan_artifacts` entry, normally `~/.vrooli/plan-artifacts/<session-slug>`), outside scenario sources; the canonical mirror stays in `~/.vrooli/plans`. Reference source directly and copy only for immutable evidence. Do not copy a transcript; place each item: outcome and user value in Purpose and Outcome, what breaks if undelivered in Purpose (§0.1), current behavior and evidence in Problem, design, alternatives, rationale, diagrams and flows in Approach & Decisions, allowed and forbidden areas and non-goals in Boundaries, assumptions, dependencies, risks and open questions in Assumptions & Risks, skills, docs, code, requirements, commands and research in Relevant Context and references, correctness in Verification and phase validation, and ordered work in Phases with acceptance. Preserve a materially explanatory diagram, example or flow in the decision content or a referenced artifact, never "see the prior discussion". Author a fenced Mermaid diagram for any flow with three or more actors, components or states instead of an inline arrow chain.

#### 0.1 Record the consequence of failure

Purpose states in one sentence what breaks or stays blocked if the plan is not delivered, or is delivered wrong: a consequence, not a level ("Without this, the desktop build ships without signed receipts, so the first paid scenario cannot be monetized", not "high priority"). An executing agent uses it to decide whether to fix a blocking defect properly or work around it and file it (`implementation-plan-execution` §5); with none it defaults to the conservative posture. Do not inflate it: an honest "a rough edge stays rough" avoids detours you did not want.

Two modes: **Plan Manager mode** (default) creates, validates and finalizes the authoritative plan through the commands below. **Candidate mode**: when a caller forbids external Plan Manager writes and asks for a compatible candidate markdown artifact, use the same inventory, placement and quality bar and return it to the caller without running `author start`, `validate` or `finalize`, and never claim it is valid or has a plan reference.

---

### 1. When To Use This Skill

Use it when the context window is tight and work must continue later, when asked for an implementation plan, or when a Swarm development grant needs its reviewed boundary (the grant binds one canonical plan; sub-assignments need none). Do not use it for a quick one-step fix, for work that fits one session with a route recoverable from docs and code (write a harness goal: `harness-goal-authoring`; shapes are chosen per `SCENARIO_DEVELOPMENT.md#work-shapes`), or for pure brainstorming (use discussion or idea-workshop first).

---

### 2. Entry Point

`plan-manager --auto-start author start --title "<plan title>"` (global flags go before the subcommand). `plan-manager author --help` and each subcommand's `--help` are the command surface; this section carries only what `--help` cannot.

**The session is a form, not a wizard.** Every response carries a full checklist (filled, missing, violation) for the touched scope. Submit any fields in any order, batched when you know the content: `author submit` batches sections, `author phase-add` adds and fills a phase in one call, `author phase-submit` batches fields on a phase. Each item returns accepted or rejected; a rejected item (unknown field, acceptance duplicating validation) is not applied while the rest lands. A complete N-phase plan takes at most 3+N mutation calls; `author continue` offers one recommended action at a time. The artifact renders in fixed cluster order (Purpose, Problem, Outcome, Approach & Decisions, Boundaries, Assumptions & Risks, Verification, Execution Setup, Phases) regardless of submission order.

**Skill discovery is a bootstrap, not curation.** `author skill-pack` with 2 to 5 decomposed concepts runs `prompt-manager discover --type skill --json`, adds the returned skills as global relevant context and prints the read command; keep most of them unless clearly irrelevant. `--phase` scopes the pack to one phase and an unknown phase reference fails rather than falling back to global. Run `search-hub query "<intent>" --type record,doc,skill` directly for docs, records and code so you inspect confidence and attribution. Submit only durable context for a resumed agent: `author context-submit` for setup commands and notes, normal reference fields for `[CODE:]`, `[DOC:]`, `[REQ:]` or an honest `NO_CODE_REFS: <reason>`. There is no candidate accept, reject or apply step. Then `author preview` (alias `author status`), `author validate`, `author finalize`, and report the plan id or slug, `plan-manager plans render <slug>`, degraded notes and the first execution command (`plan-manager exec continue <slug>`).

---

### 3. Writing Standard (ASD-STE100)

Write procedural content (steps, validation, acceptance, boundaries, Definition of Done) in Simplified Technical English: one instruction per sentence of 20 words or fewer, active voice and imperative mood, one meaning per word and the same word for the same thing throughout (define non-obvious terms where first used), and concrete objects and commands ("edit `resolver.go`", not "update the relevant files"). The banned vague words each hide an undecided decision; the canonical list is in `docs/agent-system/SKILL_AUTHORING.md` "Universal quality bars" (cite it, do not copy). STE applies to procedures only; rationale, rejected alternatives, tradeoffs and risks stay in normal prose with their nuance.

---

### 4. Judgment Rules

- A plan is executable without this chat. Before finalizing, run a preservation audit: a fresh execution agent can make the same material design decisions, respect the same constraints and validate the outcome without the conversation. Add any missing rationale, visual, reference, boundary, acceptance expectation or open question, and reject the lossy plans of §4.1.
- The change boundary names the paths the work may touch. **Author `acceptance_allow` as the full reach of the change**, not the folder the work is about: an API shape change reaches its proto, a shared behavior change reaches `packages/**`, a documented contract change reaches `docs/**`. A boundary of only `scenarios/<name>/**` for a wire-contract change is a plan defect that execution pays for with a workaround or a mid-run `exec boundary-extend`. `acceptance_allow` is an estimate execution may widen; `acceptance_deny` is a prohibition execution may never overrule, so put a path there only for "not even if it would make the change cleaner", never "probably not needed".
- `validation` is the method of checking and `acceptance` the outcome gate; they must not be identical. **Phase validation scope is proportionate by default**: a phase derives a narrow scope from its affected areas and code references; use `validation_scope: narrow:` for an explicit boundary or `validation_scope: full_plan: <rationale>` only for a stated need. The final Definition of Done is selector-free and validates the whole collection.
- Every fact lives in one section: Purpose is an abstract (no restating Problem or Outcome) and the Definition of Done carries plan-level gates only, never restated phase acceptances.
- Run `author skill-pack` early and remove a discovered skill only when clearly irrelevant or harmful. Scope a skill to a phase only when a reader of the global pack would still work that phase wrongly (a governed surface with its own maturity ladder, a different working method, an artifact with its own authoring standard); a skill list every phase carries and none reads is attention debt.
- Use explicit fallback markers only when honest: `NO_CODE_REFS`, `NO_SKILL_CONTEXT`, `NO_CONTEXT` with a reason. Plan Manager derives greenfield or brownfield posture, so do not hand-author contradictory compatibility language, and do not fabricate baseline success (authoring records regression-anchor intent; execution captures the fresh baseline). Never hand-edit rendered markdown mirrors; re-render with `plan-manager plans render` or `plans reconcile`. File out-of-scope defects as Plan Manager log entries during execution, or `prompt-manager skill read report-bug` if needed before.

#### 4.1 Anti-patterns: the lossy plan

Each passes validation and still fails a fresh executor; check every one before `author finalize`.
- Says what to do, not why this design: record the chosen design and the rejected alternative in Approach & Decisions.
- Names a component but omits the interaction: add the data flow, with a Mermaid diagram at three or more actors or states.
- Lists phases without exact acceptance or validation: give each a runnable validation and a distinct outcome gate.
- Records a decision without its tradeoff: add the tradeoff and revisit trigger.
- Assumes context that exists only in chat, or leaves discovered facts behind "investigate as needed": write the premise or fact and its evidence into the matching section or phase.
- Bounds the change to one scenario when it alters a shared contract: trace outward and list every reached path in `acceptance_allow`.
- States importance as a level instead of a consequence (§0.1).
- Uses `acceptance_deny` for "probably out of scope": leave it out of both lists.
- Accepts a retirement or cleanup phase on "the new path works": list deleted paths and removed symbols in the acceptance, each with a search that must return nothing (`improvement-do-and-dont` D7).
- Demands the whole baseline collection in every phase without a rationale: leave the field undeclared, or use `full_plan:` with a concrete reason.

---

### 5. Troubleshooting

`plan-manager` unavailable: `plan-manager --auto-start status` or `vrooli scenario start plan-manager`. `author continue` repeats a gate: a required section, reference marker, phase field or validation distinction is missing; read `remaining_required_inputs` and submit it. `author skill-pack` degrades: continue with a warning, and record `NO_SKILL_CONTEXT:` only when no useful skill setup exists. Reference discovery is manual: run search-hub, then submit `[CODE:]`, `[DOC:]`, `[REQ:]`, context or `NO_CODE_REFS:`. Anchor autofill degraded: repair the change boundary and rerun autofill. Existing markdown plan to preserve: `plan-manager plans import --source <path> --workspace <repo-root>`.

---

### **6. Output Expectations**

Produce a finalized plan id or slug (or why the session is unfinished); finalize output names the SQLite store path, the stamped workspace and a computed mirror status (`fresh` or a loud `write_failed` warning with the repair command, which you report; re-running prints `Already finalized at <ts>`); a rendered review command or path; a note on degraded dependencies or manual fallbacks; and the next execution command paired with `prompt-manager skill read implementation-plan-execution`. In Candidate mode the candidate records the material source context in the right sections, with a preservation note only if the caller's result contract has a field for it, and claims no finalization, validation, plan id or rendered path. Produce no hand-formatted markdown plan as the default artifact, no placeholder-only phases or context, and no contradictory constraints or fabricated validation evidence.
