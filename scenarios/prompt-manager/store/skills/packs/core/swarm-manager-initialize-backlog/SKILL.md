# Initialize: Bootstrap Backlog Item

## Purpose

Bootstrap a new non-research backlog item with a solid foundation in one agent pass: a canonical plan-manager scaffold and a first workshop round with targeted decisions and informational items.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read implementation-plan-authoring` — canonical plan structure, mandatory sections, convergence patterns, quality gates, and guardrails for the plan-manager implementation plan.

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — processing patterns and quality standards.

## Scope

**In scope:**
- Reading all existing item context (spec, archive, any user-added files)
- Creating or adopting an initial canonical plan-manager scaffold with as much content as possible
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

1. **Canonical plan-manager plan** — implementation plan following the structure defined by the `implementation-plan-authoring` skill (loaded via required reading above). Fill in as much as possible from existing context (spec.json, archive materials). Leave sections as `<!-- TBD -->` when information is insufficient. Ensure the backlog item ends with `spec.json.plan_ref` populated.
2. **`workshop/round-001.json`** — first workshop round with:
   - 4-7 targeted decisions presenting researched alternatives for the most important unknowns
   - 0-2 informational items sharing relevant findings from context
   - Honest readiness scores across 5 dimensions

## Implementation Plan Scaffold

The canonical plan structure, mandatory sections, convergence patterns, quality gates, and guardrails are defined by the `implementation-plan-authoring` skill (loaded via required reading above). Follow that skill exactly when creating the initial scaffold.

### Kind-Specific Plan Focus

While the section structure is universal, emphasize different areas depending on the backlog kind:

| Kind | Plan emphasis |
|------|--------------|
| idea | Vision, scope, architecture outline, key decisions, user value |
| fix | Problem statement, reproduction steps, root cause analysis, proposed fix, verification plan |
| research | Research question, methodology, data sources, expected deliverables, scope boundaries |
| execute | Task decomposition, prerequisites, implementation steps, verification criteria |
| chore | Scope boundaries, approach, completion checklist, acceptance criteria |

## Plan Workshop Handoff

Do not create a workshop round or readiness score. After the initial canonical
plan exists, the operator starts the bounded Plan Workshop review. It returns
typed findings, decision questions, and proposal drafts against the immutable
plan hash; Swarm records the proposal entries and owns any later application.

## Instructions

You are initializing a Swarm Manager backlog item. Your goal is to bootstrap it from an empty shell into a well-structured item with a canonical plan-manager draft and first workshop round ready for human review.

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
   - Existing `plan_ref`, rendered plan, and `workshop/` artifacts from a prior run (preserve these)
   - **Milestone context** — If this item belongs to milestone `{{ITEM_INITIATIVE}}`, check for strategic context:
     ```bash
     swarm-manager milestones get --name {{ITEM_INITIATIVE}}
     swarm-manager milestones files --name {{ITEM_INITIATIVE}}
     ```
     Read any files present (orchestration summaries, decision logs, strategy docs). Use milestone context to align decisions with the broader milestone goals and understand how this item relates to sibling items.

2. **Check acceptance fields**

   Read `acceptance_allow` and `acceptance_deny` from `spec.json`:
   - If `acceptance_allow` is **set**, use the patterns to determine target directories and read their structure (e.g., `ls` the matched directories) to inform the plan's technical context, file layout, and approach sections.
   - If `acceptance_allow` is **empty**, include an acceptance decision in round-001.json asking: "What file paths are expected to change?" with options: A) broad scenario-level globs (e.g., `path:scenarios/<name>/**`), B) targeted subdirectory/file paths, C) Other.

3. **Create the canonical plan scaffold**

   Based on available context, fill in as many plan sections as possible. The description and archive materials should provide enough for at least Purpose, Problem Statement, and partial Scope.

   Submit the plan markdown through the swarm-manager plan-import/finalization path so plan-manager stores it and the backlog item receives `spec.json.plan_ref`. Do not write a local implementation-plan file in the backlog folder.

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

   Confirm `spec.json.plan_ref` and `workshop/round-001.json` were created.

### Re-run Handling

```
Do any workshop artifacts already exist?
  -> No  -> Generate everything fresh
  -> Yes -> Read existing rendered plan and workshop rounds
          Preserve all existing answers and decisions
          Only fill gaps (missing plan sections, unanswered questions)
          If plan_ref exists but no rounds: create round-001.json
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
| Archive already contains detailed plan | Use it as context for the canonical plan, generate fewer questions |
| Item has rich description | Pre-answer questions where possible, focus questions on gaps |
