# Initialize: Bootstrap Backlog Item

## Purpose

Bootstrap a new backlog item with a solid foundation in one agent pass: an implementation plan scaffold and a first workshop round with targeted decisions and informational items.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read implementation-plan-authoring` — canonical plan structure, mandatory sections, convergence patterns, quality gates, and guardrails for `plan.md`.

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — processing patterns and quality standards.

## Scope

**In scope:**
- Reading all existing item context (spec, archive, any user-added files)
- Creating an initial `plan.md` scaffold with as much content as possible
- Creating a first workshop round (`workshop/round-001.json`) with decisions, informational items, and readiness scores
- Preserving any existing artifacts on re-run (fill gaps, don't overwrite)

**Out of scope:**
- Processing/implementing the item (see `swarm-manager-process-*` skills)
- Modifying `archive/` — it contains user-provided materials and must not be altered by agents
- Queueing the item for execution

## Output Requirements

All writes via CLI using `--stdin` with a heredoc (avoids shell quoting issues):
```bash
swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path <path> --stdin <<'EOF'
<content>
EOF
```

### All Kinds

1. **`plan.md`** — implementation plan following the structure defined by the `implementation-plan-authoring` skill (loaded via required reading above). Fill in as much as possible from existing context (spec.json, archive materials). Leave sections as `<!-- TBD -->` when information is insufficient.
2. **`workshop/round-001.json`** — first workshop round with:
   - 4-7 targeted decisions presenting researched alternatives for the most important unknowns
   - 0-2 informational items sharing relevant findings from context
   - Honest readiness scores across 5 dimensions

## Implementation Plan Scaffold

The `plan.md` structure, mandatory sections, convergence patterns, quality gates, and guardrails are defined by the `implementation-plan-authoring` skill (loaded via required reading above). Follow that skill exactly when creating the initial scaffold.

### Kind-Specific Plan Focus

While the section structure is universal, emphasize different areas depending on the backlog kind:

| Kind | Plan emphasis |
|------|--------------|
| idea | Vision, scope, architecture outline, key decisions, user value |
| fix | Problem statement, reproduction steps, root cause analysis, proposed fix, verification plan |
| research | Research question, methodology, data sources, expected deliverables, scope boundaries |
| execute | Task decomposition, prerequisites, implementation steps, verification criteria |
| chore | Scope boundaries, approach, completion checklist, acceptance criteria |

## Workshop Round Schema

See `swarm-manager-workshop` skill for the full schema. The round file includes:
- `round`: 1
- `generated_at`: ISO timestamp
- `readiness`: 5 dimension scores (0-3)
- `items`: array of decision/info items
- `plan_updates`: description of what was written to plan.md

### Readiness Dimensions

| Dimension | What It Measures |
|-----------|-----------------|
| `problem_clarity` | Is the problem/goal well understood? |
| `scope_defined` | Are boundaries and non-goals defined? |
| `approach_solid` | Is there a clear implementation strategy? |
| `testable` | Do we know how to verify success? |
| `risk_awareness` | Are blockers and unknowns identified? |

Score each 0 (not started) to 3 (solid). First rounds typically score 0-2 depending on how much context exists.

## Instructions

You are initializing a Swarm Manager backlog item. Your goal is to bootstrap it from an empty shell into a well-structured item with a draft plan and first workshop round ready for human review.

**Context from spec.json:**
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
   - `spec.json` — the item description and metadata
   - `archive/` — user-provided materials (requirements docs, prior scenario artifacts, designs)
   - Any user-added files in the item root
   - Existing `plan.md` and `workshop/` artifacts from a prior run (preserve these)

2. **Check scope and acceptance fields**

   Read `scope`, `acceptance_allow`, and `acceptance_deny` from `spec.json`:
   - If `scope` is **set**, use it to contextualize the plan scaffold. Read the target scenario's directory structure (e.g., `ls scenarios/<scope-target>/`) to inform the plan's technical context, file layout, and approach sections.
   - If `scope` is **NOT set**, include a scope decision in round-001.json asking: "Which scenario does this work target?" with options listing likely candidates based on the item description.
   - If `scope` is set but `acceptance_allow` and `acceptance_deny` are both missing, include an acceptance decision in round-001.json asking: "What file paths are expected to change?" with options based on the target scenario's structure (e.g., A: broad `scenarios/<name>/**`, B: targeted subdirectories, C: Other).

3. **Create plan.md scaffold**

   Based on available context, fill in as many plan sections as possible. The description and archive materials should provide enough for at least Purpose, Problem Statement, and partial Scope.

   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path plan.md --stdin <<'EOF'
   <plan content>
   EOF
   ```

4. **Generate workshop round 1**

   Identify the 4-7 most important unknowns and present them as decisions with researched alternatives:
   - Each decision must have at least 2 options (A, B) and should usually include an "Other" option
   - Each option needs a clear label and rationale explaining tradeoffs
   - Indicate which option you recommend and why in the `context` field
   - If the best option can be inferred from context, pre-select it (set `selected` to the key)
   - For factual questions, present likely ranges as options
   - Include info items for important findings from archive/context review

   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path workshop/round-001.json --stdin <<'EOF'
   <round JSON>
   EOF
   ```

5. **Score readiness**

   Evaluate each dimension honestly based on the plan scaffold you just created. First rounds typically range from 0-2 depending on context richness.

6. **Verify all outputs**

   ```bash
   swarm-manager backlog files --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   ```

   Confirm `plan.md` and `workshop/round-001.json` were created.

### Re-run Handling

```
Do any workshop artifacts already exist?
  -> No  -> Generate everything fresh
  -> Yes -> Read existing plan.md and workshop rounds
          Preserve all existing answers and decisions
          Only fill gaps (missing plan sections, unanswered questions)
          If plan.md exists but no rounds: create round-001.json
          If rounds exist: create the next numbered round
```

## Anti-Patterns

- **Don't** overwrite existing user answers or decisions on re-run
- **Don't** generate more than 7 decisions — focus on the most impactful unknowns
- **Don't** present decisions with fewer than 2 options
- **Don't** modify files in `archive/` — these are user-provided
- **Don't** write files directly to disk — always use the backlog CLI
- **Don't** skip reading context before generating — existing materials may answer your questions
- **Don't** inflate readiness scores — be honest about the plan's current state

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `file-get` returns 404 | Normal on first run — generate fresh content |
| `file-upload` fails | Check kind and name match: `swarm-manager backlog get --kind <kind> --name <name>` |
| Archive already contains detailed plan | Use it as context for plan.md, generate fewer questions |
| Item has rich description | Pre-answer questions where possible, focus questions on gaps |
