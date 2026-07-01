## Practice focus: Implementation Plan Authoring

Author durable implementation plans through Plan Manager's guided authoring
runtime. Plan Manager is the plan-logic authority: the skill supplies the
judgment layer, while `plan-manager author ...` owns the section order, phase
shape, validation gates, rendered markdown, and persistence.

Required reading:
- `prompt-manager skill read plan-skill-discovery` — discover plan-relevant
  skills and operational context before authoring.
- `scenarios/plan-manager/docs/concepts/PLAN-MODEL.md` — canonical structured
  plan and phase model.
- `scenarios/plan-manager/docs/reference/cli-commands.md` — current
  `plan-manager author`, `plans`, `exec`, `validate`, and `log` command surface.

Optional reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `prompt-manager skill read skill-validation`

---

### 1. When To Use This Skill

| Situation | Use this skill? | Why |
|---|---|---|
| Context window is tight and work must continue later | Yes | Captures executable state in Plan Manager |
| User asked for an implementation plan | Yes | Produces a structured plan and rendered review artifact |
| Quick one-step fix with no follow-up risk | No | Plan overhead is unnecessary |
| Pure brainstorming with no execution intent | No | Use discussion or idea-workshop flow first |

---

### 2. Scope Boundaries

**In scope:**
- Author a new implementation plan through `plan-manager author ...`
- Preserve problem, target outcome, constraints, change boundary, execution
  setup, phases, validation strategy, and Definition of Done in Plan Manager's
  structured record
- Curate relevant context and references by accepting/rejecting Plan Manager
  candidates with judgment
- Report the saved plan id/slug and rendered review command/path

**Out of scope:**
- Defining a separate markdown plan format
- Hand-editing rendered Plan Manager markdown mirrors
- Using project-level compatibility commands as the default authoring path
- Implementing the planned code changes during plan authoring unless the user
  explicitly asks for both in the same turn

---

### 3. Convergence Pattern

Use this decision table before authoring:

| Need | Primary path |
|---|---|
| New implementation plan | `plan-manager author start --title "<title>"`, then follow `plan-manager author continue <session>` |
| Review in-progress plan | `plan-manager author preview <session>` |
| Persist completed authoring session | `plan-manager author validate <session>`, then `plan-manager author finalize <session>` |
| Inspect persisted plan | `plan-manager plans render <plan-id-or-slug>` |
| Execute or resume plan work | `plan-manager exec continue <plan-or-execution>` |
| Adopt existing legacy markdown | `plan-manager plans import --source <path> --workspace <repo-root>` |
| Repair/adopt plan mirrors and legacy files | `plan-manager plans reconcile --dry-run --workspace <repo-root>` |

If a root compatibility command appears in older docs or plans, prefer the
matching `plan-manager` command unless the user explicitly asks to test the root
compatibility layer.

---

### 4. Authoring Workflow

#### Phase 0: Recall and discover

Before authoring, recall prior work:

```bash
search-hub query "<one-sentence plan intent>" --type record,backlog,initiative,skill,doc
```

Then use plan skill discovery:

```bash
prompt-manager skill read plan-skill-discovery
prompt-manager discover "<concept-1>" "<concept-2>" "<concept-3>" --type skill --complexity moderate
```

Keep the discovery layers separate:
- `search-hub query ...` is broad recall across records, docs, backlog, and
  registered providers.
- `prompt-manager discover --type skill` is the curated, budget-aware
  Prompt Manager skill bundle for the executor to read.
- `plan-manager author context-discover` turns setup searches into reviewable
  Plan Manager candidates; it does not remove the need to accept/reject with
  judgment.

Accepted discovery output should become Plan Manager relevant context, not a
legacy markdown `Required Reading` section. Prefer accepting useful candidates
through `plan-manager author context-discover` / `context-accept`; when the
curated Prompt Manager output gives a concrete `prompt-manager skill read ...`
command, submit that exact setup command with `context-submit`.

#### Phase 1: Start the authoring session

```bash
plan-manager --auto-start author start --title "<plan title>"
```

Use the returned session id for every subsequent command. Global flags such as
`--auto-start` go before the subcommand.

#### Phase 2: Follow the guided loop

Default to the API-owned next action:

```bash
plan-manager author continue <session>
```

Submit exactly the current requested section or phase field. Do not submit one
large plan blob unless Plan Manager explicitly asks for a field that contains
multi-line content.

When relevant:

```bash
plan-manager author context-discover <session> --concepts "<concepts>" --complexity moderate
plan-manager author context-accept <session> <candidate-id>
plan-manager author suggest-references <session>
plan-manager author reference-accept <session> <candidate-id>
plan-manager author autofill <session> --sources regression_anchor
```

Accept candidates only when they materially improve execution. Reject noisy
candidates with a reason. Use explicit fallback markers only when honest:

- `NO_CODE_REFS: <reason>` for plans or phases with no connected code/doc/req
  references.
- `NO_CONTEXT: <reason>` for global or phase setup that genuinely needs no
  additional context.

#### Phase 3: Review, validate, finalize

```bash
plan-manager author preview <session>
plan-manager author validate <session>
plan-manager author finalize <session>
```

Do not finalize with unresolved structure violations unless Plan Manager returns
a typed degraded path and the user explicitly accepts the risk.

#### Phase 4: Report the artifact

Return:
- plan id and slug
- rendered review command, usually `plan-manager plans render <slug>`
- any degraded discovery, reference, context, or anchor notes
- the first execution command, usually `plan-manager exec continue <slug>`

---

### 5. Judgment Rules

- A plan should be executable without this chat history.
- The change boundary must name the paths the work may touch; do not hide scope
  in prose.
- `validation` is the method of checking; `acceptance` is the outcome gate.
  They must not be identical.
- Greenfield/Brownfield posture is derived by Plan Manager. Do not hand-author
  contradictory compatibility language.
- Do not accept context or reference candidates just to satisfy a gate; they must
  be relevant.
- Do not fabricate baseline success. Plan Manager records regression-anchor
  intent during authoring; execution/validation captures and checks the fresh
  baseline.
- For out-of-scope defects found while authoring, use Plan Manager log entries
  during execution, or load `prompt-manager skill read report-bug` if the defect
  needs filing before execution begins.

---

### 6. Troubleshooting & Edge Cases

| Symptom | Likely cause | First move |
|---|---|---|
| `plan-manager` is unavailable | Scenario is stopped or not installed | Run `plan-manager --auto-start status`, or `vrooli scenario start plan-manager` |
| `author continue` repeats a gate | A required section, context item, reference, phase field, or validation distinction is missing | Read `remaining_required_inputs` / human output and submit the requested field |
| Reference discovery returns nothing | `search-hub` unavailable or no routed locator hits | Manually submit `[CODE:]`, `[DOC:]`, `[REQ:]`, or honest `NO_CODE_REFS:` |
| Context discovery is noisy | Prompt-manager/search-hub returned broad candidates | Accept only relevant candidates; reject the rest |
| Anchor autofill is degraded | Boundary missing or validation dependency unavailable | Submit/repair change boundary, rerun autofill, or record degraded intent only when Plan Manager permits it |
| Need to preserve an existing markdown plan | Legacy import/adoption path | Use `plan-manager plans import --source <path> --workspace <repo-root>` |

Repeated troubleshooting that requires prose here should be promoted into Plan
Manager CLI guidance rather than expanded in this skill.

---

### 7. Output Expectations

**Must produce:**
- A finalized Plan Manager plan id/slug, or a clear explanation of why the
  authoring session remains unfinished
- A rendered review command/path
- A concise note about degraded dependencies or manual fallbacks
- The next execution command when implementation should continue

**Must not produce:**
- A standalone 13-section markdown plan as the default artifact
- A root compatibility command as the default authoring path
- Placeholder-only phases or context entries
- Contradictory constraints or fabricated validation evidence
