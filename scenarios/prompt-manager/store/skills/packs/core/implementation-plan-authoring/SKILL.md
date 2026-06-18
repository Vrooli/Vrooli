## Practice focus: Implementation Plan Authoring

Create a durable implementation plan through `vrooli plans add --stdin` by default, preserving full execution context when conversation history is about to compact. This skill standardizes how to capture the problem, constraints, execution approach, and acceptance criteria so any future agent can continue without prior chat context.

Required reading:
- `prompt-manager skill read plan-skill-discovery` — the canonical method for finding domain-relevant skills for this specific plan. Run its discovery process before authoring.

Conditional reading (load only when the plan touches the matching surface — `plan-skill-discovery` will usually surface these automatically):
- `cli-steer` — when the plan adds or modifies a scenario CLI command.
- `api-steer` — when the plan adds or modifies a proto / Connect-RPC contract.
- `utils-unification` — when the plan introduces new helpers or touches shared utilities.
- `seam-discovery-and-enforcement` — when the plan introduces new testability seams or changes existing ones.
- `ecosystem-fit` — when the plan creates a new scenario or significantly changes an existing one's role or interface surface. Places it in Vrooli's interfaces, functional role, and compound-value design (`path:docs/concepts/ECOSYSTEM.md`).

Skip these reads entirely for plans that don't touch the corresponding surface (e.g., pure docs, UI-only, research conclusions). Reading skills that don't apply wastes context.

When the plan touches a scenario, also skim:
- `git-control-tower baseline help` — flag shape for the regression anchor captured in Step A.5.

Optional reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `prompt-manager skill read skill-validation`

---

### 1. When to Use This Skill

| Situation | Use this skill? | Why |
|---|---|---|
| Context window is getting tight and work must continue later | Yes | Prevents loss of critical implementation reasoning |
| User asked for a "plan file" or "implementation plan document" | Yes | Produces a reusable execution artifact |
| Quick one-step fix with no follow-up risk | No | Plan overhead is unnecessary |
| Pure brainstorming with no execution intent | No | Use normal discussion flow |

---

### 2. Scope Boundaries

**In scope:**
- Create/update a single implementation plan through `vrooli plans add --stdin` unless an in-repo durable artifact is explicitly requested
- Preserve problem statement, root causes, constraints, and action plan
- Include explicit required reading commands for future agents
- Include Action commands as evidence or validation steps when relevant, while keeping required-reading discovery focused on skills and methodologies
- Define objective acceptance criteria and test/validation gates
- Document strict constraints such as greenfield/no-compatibility when requested

**Out of scope:**
- Implementing the planned code changes
- Replacing design docs, PRDs, or user-facing product docs
- Writing migration guides unless explicitly requested

---

### 3. Inputs

Required:
- The concrete problem being solved
- The target code areas/systems involved
- Any hard constraints (for example: greenfield only, no compatibility layers)

Optional but recommended:
- Live command evidence showing failures or friction
- Adjacent skills/tools that should be treated as source-of-truth
- Blast radius: which scenario(s), if any. Drives the regression-anchor strategy in Step A.5.

---

### 4. Convergence Pattern

Use this decision flow to keep outputs consistent:

1. **Was implementation already started?**
   - Yes: document current state, known blockers, and exact resume point.
   - No: document baseline, risks, and first executable phase.

2. **Did the user request strict architecture constraints?**
   - Yes: add a hard-rule section (for example: "Greenfield Constraint") and repeat it in Definition of Done.
   - No: include standard guardrails only.

3. **Is there command-level evidence?**
   - Yes: include exact command snippets and observed outputs/errors.
   - No: label assumptions clearly and provide verification commands to replace assumptions.

4. **Is work multi-component?**
   - Yes: phase by component with explicit dependencies and ordering.
   - No: keep phases minimal and linear.

---

### 5. Plan Authoring Workflow

#### Step 0: Recall prior work (before discovery)

Before authoring, run the AGENTS.md §4 "Recall prior work first" beat, scoped to planning:

```bash
search-hub query "<one-sentence plan intent>" --type record,backlog,initiative,skill,doc
```

Read the top hits. A `record` hit shows how prior work was actually *executed* — its trigger and approach — which is exactly the decision context a plan wants (not just code). Three exits: no prior art → author fresh; related prior work → cite it in §5 Current Technical Context and build on it; a near-duplicate of an already-shipped plan → stop and reconcile (supersede or extend it) rather than re-authoring. If search-hub is unavailable, fall back to `swarm-manager records search "<intent>"`.

#### Step A: Establish canonical context

Discover the skills that apply to *this* plan rather than reading a fixed list. Follow `plan-skill-discovery`:

```bash
prompt-manager skill read plan-skill-discovery
```

Then run its discovery process — typically:

```bash
prompt-manager discover "<concept-1>" "<concept-2>" "<concept-3>" --complexity moderate
```

Keep the default **skill mode** and pass `--complexity`: skill mode is curated — discover returns the relevant topic packs (each topic's skills plus its folder's and the root's) alongside strong direct matches, which is the required-reading set a plan wants. Do not switch to `--type all` for plan authoring (that is best-match relevance for "find an existing tool", not curated packs).

If the discovery output surfaces any of the conditional reads above (`cli-steer`, `api-steer`, `utils-unification`, `seam-discovery-and-enforcement`), or your plan obviously touches their surface, load them now. Otherwise skip them. Then gather implementation evidence (commands, files, observed failures) before writing.

#### Step A.5: Anchor the regression surface

Capture a "before" anchor *before any code changes* so future agents (or you, post-implementation) can answer "did this plan introduce that failure?" without ambiguity. Pick exactly one strategy based on the plan's blast radius and record the choice in mandatory section 6a:

| Plan touches… | Strategy | Command |
|---|---|---|
| **One scenario** | `git-control-tower baseline` snapshot | `git-control-tower baseline snapshot --scenario <name> --name <plan-slug> --reason "<plan title>"` |
| **Multiple scenarios** | One baseline per touched scenario | Loop the command above per scenario; record every `(scenario, plan-slug)` pair |
| **Outside any scenario** (root tooling, `internal/`, `packages/proto`, docs-only, `.vrooli/`) | **Skip baseline; record a sha + file allowlist** | `git rev-parse HEAD` — copy the sha into section 6a along with the file paths the plan is allowed to touch |

Notes:
- Baseline captures 5 surfaces (workflows, tests, structure, visuals, rules). Use `--fast` or `--include <surfaces>` to scope; full captures can be slow.
- Do **not** pass an arbitrary `--scenario` for outside-scenario plans — only the `visuals` surface would capture and the report misleads.
- Diff exit codes: `0` safe, `1` regression, `2` not-comparable. Section 10's regression check should treat `1`/`2` as actionable.

#### Step B: Create the scratch plan through the CLI

Default behavior:
- Use `vrooli plans add --title "<topic>" --stdin` and write the plan content to stdin.
- Report the saved path and plan id printed by the command.
- Do not hard-code the plan storage directory; the CLI owns the location.

In-repo exceptions:
- Use an in-repo plan only when the user explicitly asks for it, the plan is being promoted to durable documentation, or a scenario-specific workflow requires it.
- Swarm Manager backlog items keep using their item-local `plan.md` / `conclusion.md` artifacts.

#### Step C: Write the plan with mandatory sections

Every plan must include:
1. Purpose
2. Required Reading (explicit commands — see Step D for what to embed)
3. Problem Statement
4. Scope (in/out)
5. Current Technical Context (key files/components)
6. Target End State
6a. Regression Anchor — strategy chosen in Step A.5; for the scenario / multi-scenario cases, list each `(scenario, plan-slug)` pair and the matching `baseline diff` command; for the outside-scenario case, list the `HEAD` sha + the file-path allowlist and the matching `git diff --stat <sha> -- <paths>` command. This section is mandatory and cannot be empty.
7. Implementation Strategy (phased)
8. Contract Decisions (API/CLI/data model behavior)
9. Testing Plan
10. Rollout/Validation Checklist — must contain a "Regression check" line matching section 6a's strategy: `git-control-tower baseline diff --scenario <s> --name <plan-slug>` (per pair; exit 0 required, exit 1/2 must be triaged), or `git diff --stat <sha> -- <paths>` showing only declared files.
11. Risks + Mitigations
12. Non-goals / Prohibited Patterns
13. Definition of Done — must include "Regression check from section 10 passes" alongside the other pass criteria. For plans that create a new scenario or change an existing one's role or interface surface, also include an "Ecosystem-fit considered" criterion: the scenario's served/enabled interfaces, functional role, and compound-value seams are reflected in Target End State (per `ecosystem-fit` / `path:docs/concepts/ECOSYSTEM.md`).

If user requests strict constraints (for example greenfield):
- Add a dedicated hard-rule section near the top.

#### Step D: Validate quality before finishing

Quality checks:
- Another agent could execute without this chat history
- All major claims are tied to files/commands or marked assumptions
- Constraints are explicit, repeated where needed, and testable
- Acceptance criteria are objective (pass/fail), not narrative
- The plan's Required Reading block lists only the skills that actually apply to the plan's surface — not a generic fixed list

---

### 5b. Out-of-scope defects discovered during authoring

While authoring, you will spot defects unrelated to the plan's surface. Use this three-part heuristic:

1. **Cheap + confident + adjacent** → fix in the plan with a one-line note in §11 Risks or §5 Current Technical Context. Criteria: ≤1 file, no API change, no new tests beyond an obvious assertion, and you can name the root cause without further investigation.
2. **Any of**: needs investigation, unrelated files, new tests, API change, unknown ripple → load `prompt-manager skill read report-bug` and file through the workflow it provides.
3. **Defect is the actual plan trigger** → don't sidebar it; fold it into §3 Problem Statement.

When the plan itself reaches Definition of Done (executor finishes §10 checklist), write a `kind: execute` record describing the substantive decisions:

```bash
swarm-manager records create --kind execute --scenario <name> \
  --trigger "<plan slug>" \
  --approach "<key decisions: contract shapes, seam choices, deferred risks>" \
  --ruled-out "<rejected alternatives>" \
  --commit <merge-sha> --outcome shipped
```

This is the write-side of the recursive-learning loop — future plan-authoring sessions query records to learn from prior decisions, not just code.

---

### 6. Guardrails

- Do not write vague plans ("improve X", "refactor Y") without concrete deliverables.
- Do not hide assumptions; mark unknowns and how to resolve them.
- Do not mix implementation and plan-authoring in the same step unless explicitly asked.
- Do not include legacy/migration/compatibility guidance when user requested greenfield-only.
- Do not pad the plan's Required Reading with skills that don't match the plan's surface — load the conditional steers only when they apply.
- Do not use `git stash` for "is this failure mine?" diagnosis. Concurrent agents share the working tree and stash is process-global; use the regression anchor from section 6a instead. `git-control-tower baseline` is branch-scoped and `flock`-guarded for exactly this reason.
- Do not fake a baseline for outside-scenario plans by passing an arbitrary `--scenario`. Only the `visuals` surface would capture and the resulting report misleads — use the sha + file-allowlist strategy from Step A.5 instead.

---

### 7. Output Expectations

**Must produce:**
- A saved implementation plan file path
- A plan detailed enough for handoff to a future agent with no chat context
- A Required-Reading command block tailored to the plan's actual surface. Compose it from:
  - The conditional steers above, included only when the plan touches their surface
  - Any domain-specific skills surfaced by `plan-skill-discovery`
- Action validation or discovery commands only when they are directly relevant to the plan's execution path

**May include:**
- Copy-pastable command snippets
- File-level references for targeted implementation
- Prioritized phase ordering with dependency notes

**Must not include:**
- Placeholder-only sections with no actionable content
- Contradictory constraints (for example greenfield + migration bridge)
- Implicit assumptions presented as facts
- A generic fixed required-reading list that doesn't match the plan's surface
