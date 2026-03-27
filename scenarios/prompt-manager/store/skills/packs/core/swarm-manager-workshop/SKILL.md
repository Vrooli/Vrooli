# Workshop: Iterative Plan Refinement

## Purpose

Run one workshop round for a backlog item of any kind. Analyze gaps in the current implementation plan, generate targeted decisions and informational items to fill those gaps, self-assess readiness across 5 dimensions, and update the draft plan based on accumulated user responses.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read implementation-plan-authoring` — canonical plan structure, mandatory sections, convergence patterns, quality gates, and guardrails for `plan.md`.

**Required reading:** `prompt-manager skill read plan-skill-discovery` — methodology for discovering and embedding relevant skills into the plan

## Scope

**In scope:**
- Reading all existing item context (spec, plan, prior workshop rounds, research, archive, user files)
- Generating a mix of decisions and informational items tailored to the item kind and current plan state
- Self-assessing readiness across 5 standardized dimensions
- Updating `plan.md` with refined/new sections based on accumulated answers and decisions
- Writing the round file (`workshop/round-NNN.json`)

**Out of scope:**
- Processing/implementing the item (see `swarm-manager-process-*` skills)
- Modifying `archive/` — it contains user-provided materials and must not be altered
- Queueing the item for execution
- Research items (see `swarm-manager-workshop-research` for that)

## Output Requirements

All writes via CLI using `--stdin` with a heredoc:
```bash
swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path <path> --stdin <<'EOF'
<content>
EOF
```

### Every Round Produces

1. **`workshop/round-{{ROUND_NUMBER}}.json`** — the round file (see schema below)
2. **`plan.md`** — updated implementation plan (create if first round, update if subsequent)

## Workshop Round Schema

```json
{
  "round": {{ROUND_NUMBER}},
  "generated_at": "<ISO-8601 timestamp>",
  "readiness": {
    "problem_clarity": 0,
    "scope_defined": 0,
    "approach_solid": 0,
    "testable": 0,
    "risk_awareness": 0
  },
  "items": [
    {
      "id": "d1",
      "type": "decision",
      "topic": "Authentication approach",
      "context": "Why this matters and what was found",
      "options": [
        {"key": "A", "label": "OAuth with Google", "rationale": "Lowest effort, covers 90% of users", "recommended": true},
        {"key": "B", "label": "JWT with custom auth", "rationale": "More control, offline support"},
        {"key": "C", "label": "Other", "rationale": "Provide your own approach"}
      ],
      "selected": null,
      "freeform": null,
      "notes": null
    },
    {
      "id": "i1",
      "type": "info",
      "text": "Found that the middleware already handles session management"
    }
  ],
  "plan_updates": "Brief description of what plan sections were created or updated this round"
}
```

### Item Types

| Type | Purpose | User Action |
|------|---------|-------------|
| `decision` | A decision point with multiple lettered options for the user to choose from. The agent presents researched alternatives with rationale for each. | Select an option (A, B, C...) or choose "Other" and provide freeform input. Optional notes. |
| `info` | Share a finding, observation, or important context | Read-only — no action needed |

### Decision Item Guidelines

- Every decision MUST have at least 2 options (A, B) and should usually include an "Other" option as the last choice
- Options are lettered A, B, C, D... (not numbered)
- Each option needs a clear `label` AND `rationale` explaining tradeoffs
- Set `"recommended": true` on exactly one option per decision to indicate the agent's pick. The rationale for that option should explain why it's recommended
- Decisions replace both questions (where the agent presents answer options) and proposals (where the agent presents approaches)
- For factual questions (like "what's the target user count?"), present likely ranges as options (e.g., A: "Under 100", B: "100-10k", C: "10k+", D: "Other")
- Use `id` prefixes: `d1`, `d2`... for decisions; `i1`, `i2`... for info items

### Readiness Dimensions

Score each dimension honestly from 0-3 based on the CURRENT state of the plan:

| Dimension | What It Measures | 0 | 1 | 2 | 3 |
|-----------|-----------------|---|---|---|---|
| `problem_clarity` | Is the problem/goal well understood? | No information | Vague idea | Clear problem, some unknowns | Fully understood, well-articulated |
| `scope_defined` | Are boundaries and non-goals defined? | No acceptance criteria set | Target area identified in description/plan but no acceptance_allow patterns | acceptance_allow defined, covers planned changes | Both acceptance_allow and acceptance_deny defined, plan changes align with globs |
| `approach_solid` | Is there a clear implementation strategy? | No approach | General direction | Concrete strategy, some details TBD | Detailed phased plan with dependencies |
| `testable` | Do we know how to verify success? | No test plan | Vague success criteria | Specific test cases identified | Complete test plan with acceptance criteria |
| `risk_awareness` | Are blockers and unknowns identified? | Not considered | Some risks noted | Key risks with mitigations | Comprehensive risk matrix |

**Scoring rules:**
- Be honest — do not inflate scores to appear further along than reality
- Score based on plan content, not assumptions
- A dimension at 0 means no relevant content exists in the plan for that area
- A dimension at 3 means someone could execute that aspect without further clarification

## Implementation Plan Format

The `plan.md` file structure, mandatory sections, convergence patterns, quality gates, and guardrails are defined by the `implementation-plan-authoring` skill (loaded via required reading above). Follow that skill exactly when creating or updating `plan.md`.

**Workshop-specific notes:**
- Not every section needs content from round 1. Fill what you can and leave sections as `<!-- TBD -->` when information is insufficient. Each subsequent round should fill more sections.
- On the first round, create a scaffold with as much content as possible from existing context.
- On subsequent rounds, refine existing sections and fill gaps based on accumulated user responses.

## Instructions

You are running workshop round {{ROUND_NUMBER}} for a swarm-manager backlog item.

**Item context:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Read all available context**

   ```bash
   swarm-manager backlog get --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   swarm-manager backlog files --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   ```

   Then read each available artifact:
   - `plan.md` — current implementation plan draft (if exists)
   - `workshop/` — all prior round files (to understand what's been asked, answered, decided)
   - `spec.json` — original item description and metadata
   - `research/summary.md` — deep research findings (if exists)
   - `archive/` — user-provided materials
   - Any user-uploaded files

2. **Analyze prior rounds** (if ROUND_NUMBER > 1)

   Review all prior workshop rounds. For each:
   - Note decisions with a `selected` value — these are settled, incorporate into the plan
   - Note decisions with `selected: null` — still pending, do not re-ask unless context has materially changed
   - Note any freeform responses on "Other" selections — incorporate these as user intent

3. **Auto-populate acceptance fields**

   Read `acceptance_allow` and `acceptance_deny` from `spec.json`:
   - If `acceptance_allow` is **empty**, infer the target scenario(s) from the item's title, description, tags, and plan content. Then auto-set `acceptance_allow` using `swarm-manager backlog update`:
     - Default to broad scenario-level globs: `scenarios/<scenario-name>/**`
     - If the plan is specific enough to identify subdirectories (e.g., only `api/` or `ui/`), use targeted globs instead
     - Example: `swarm-manager backlog update --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --data '{"acceptance_allow":["scenarios/<scenario-name>/**"]}'`
     - Do NOT generate a decision for this — determine the patterns autonomously from context
     - `backlog update` is sparse: omitted fields stay unchanged, and empty arrays clear list fields
   - If `acceptance_allow` is **set**, validate that the plan's described changes align with the patterns. Flag any planned changes that fall outside acceptance_allow as an info item.
   - For `acceptance_deny`: if the plan identifies paths that should be protected (e.g., secrets, config, generated files), auto-set `acceptance_deny` the same way. Otherwise leave it empty — most items don't need deny patterns.

4. **Identify plan gaps**

   Compare the current plan against the 5 readiness dimensions:
   - What's missing or weak in problem clarity?
   - What scope boundaries are undefined?
   - What implementation details need to be worked out?
   - What test/verification strategy is missing?
   - What risks haven't been identified?

5. **Discover relevant skills**

   Apply the plan-skill-discovery methodology to find domain-relevant skills:

   **Kind-specific required skills (always include in plan's Required Reading):**

   | Item Kind | Required Skill | Why |
   |-----------|---------------|-----|
   | `idea` | `scenario-generation` | Scenario scaffolding, PRD/requirements tooling, ecosystem-manager integration |

   For other kinds (`fix`, `execute`, `chore`), no kind-specific skill is required — discovery is sufficient. Research items use `swarm-manager-workshop-research` and are not handled by this skill.

   When the item kind matches a row above, embed that skill in plan.md's Required Reading **in addition to** whatever discovery finds. This is not optional — it ensures operational knowledge is always available to the executing agent.

   **On round 1 (full discovery):**
   a. Classify the work using the item's kind, title, description, and tags
   b. Decompose into 2-5 focused concepts. For example, for a fix item titled "SQLite migration fails on large tables":
      - "SQLite migration"
      - "database schema changes"
      - "Go error handling"
   c. Run unified discovery with all concepts:
      ```bash
      prompt-manager discover "<concept-1>" "<concept-2>" "<concept-3>" --complexity moderate
      ```
   d. Read top candidates from the discover output:
      ```bash
      prompt-manager skill read <id-1> <id-2> <id-3> -output combined
      ```
   e. Assess relevance autonomously — include only skills that will materially improve the plan
   f. Embed discovered skills as Required Reading entries in plan.md (alongside any kind-specific required skills from above)

   **On subsequent rounds (conditional re-discovery):**
   - Skip if the approach and domain have not changed materially since the last discovery
   - Re-run discovery ONLY if:
     - A user decision shifted the technical approach (e.g., changed from SQLite to PostgreSQL)
     - New context reveals a domain not covered by existing Required Reading
   - When re-running, check for skills already in Required Reading and only search for the new domain

   **Use discovered knowledge:**
   - Let discovered skill content inform the decisions and options you generate in step 6
   - Reference specific skill guidance when it supports a recommended option

6. **Generate workshop items**

   Based on the gaps identified, produce a focused set of items:

   **Target counts by item kind:**

   | Kind | Decisions | Info | Focus |
   |------|-----------|------|-------|
   | idea | 4-7 | 0-2 | Scope, feasibility, architecture, user value |
   | fix | 2-4 | 1-2 | Reproduction, root cause, fix strategy, regression risk |
   | execute | 3-5 | 0-1 | Requirements clarity, decomposition, verification |
   | chore | 2-4 | 0-1 | Scope boundaries, approach, completion criteria |

   > **Note:** Research items use `swarm-manager-workshop-research` instead of this skill.

   **Quality rules:**
   - Decisions should target specific plan gaps, not general curiosity
   - Each decision option should have a clear label and rationale explaining tradeoffs
   - Mark exactly one option per decision with `"recommended": true` and explain why in its rationale
   - Info items should share genuinely useful findings (codebase observations, dependency discoveries, etc.)
   - Do not repeat decisions from prior rounds unless the context has materially changed
   - Do not re-present decisions that the user has already resolved
   - Pre-select options when inferable from existing context (set `selected` to the key)
   - Use IDs like `d1`, `d2`... for decisions, `i1`, `i2`... for info items (unique within the round)

7. **Score readiness**

   Evaluate each dimension honestly based on the current state of the plan AFTER incorporating answers from prior rounds. Use the scoring rubric above.

8. **Update plan.md**

   Incorporate all settled information into the plan:
   - Resolved decisions (with a `selected` value) become facts/commitments in relevant sections
   - Freeform responses on "Other" selections become user-specified approaches
   - Research findings inform the technical context and risk sections
   - If this is round 1, create the scaffold with as much content as possible
   - If subsequent round, refine existing sections and fill gaps

   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path plan.md --stdin <<'EOF'
   <updated plan content>
   EOF
   ```

9. **Write the round file**

   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path workshop/round-{{ROUND_NUMBER}}.json --stdin <<'EOF'
   <round JSON>
   EOF
   ```

   Use zero-padded 3-digit round numbers: `round-001.json`, `round-002.json`, etc.

10. **Verify outputs**

   ```bash
   swarm-manager backlog files --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   ```

   Confirm both `plan.md` and `workshop/round-{{ROUND_NUMBER}}.json` were created.

### Readiness Progression Guidance

As rounds progress, your focus should shift:

| Rounds Completed | Primary Focus |
|-----------------|---------------|
| 0 (first round) | Problem clarity, initial scope, gather requirements, skill discovery |
| 1-2 | Approach selection, technical context, scope refinement |
| 3-4 | Testing strategy, risk identification, implementation details |
| 5+ | Polish, edge cases, final validation criteria |

**When to suggest readiness:** If all dimensions are at 2+ and you believe the plan is solid enough for execution, include an info item noting: "This plan appears ready for execution. Consider reviewing and queuing." Do not inflate scores to reach this point — the system applies a boost formula that accounts for thoroughness over multiple rounds.

## Anti-Patterns

- **Don't** inflate readiness scores — be honest about what's missing
- **Don't** repeat decisions from prior rounds that were already resolved
- **Don't** present decisions with fewer than 2 options
- **Don't** generate more items than the target counts — focus on highest-impact gaps
- **Don't** modify files in `archive/` — these are user-provided
- **Don't** write files directly to disk — always use the backlog CLI
- **Don't** skip reading prior rounds — context accumulates across rounds
- **Don't** leave plan.md unchanged — every round should advance the plan
- **Don't** present decisions that could be resolved by reading existing context
- **Don't** generate decisions for `acceptance_allow` or `acceptance_deny` — infer and auto-set them
- **Don't** omit the "Other" option unless the choices are truly exhaustive (e.g., yes/no)

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `file-get` returns 404 for plan.md | Normal on first round — create the scaffold |
| `file-get` returns 404 for workshop/ | Normal on first round — create the directory with round-001.json |
| Prior round has unresolved decisions | Still pending — don't re-present, they're waiting for user input |
| All readiness dimensions already at 3 | Unusual but possible — generate minimal items focused on edge cases, or note readiness in an info item |
| Conflicting information between sources | Apply source authority: user answers > accepted proposals > plan.md > research > spec.json > archive |
| Very large workshop history | Focus on the latest 2-3 rounds and the settled decisions from earlier rounds |
