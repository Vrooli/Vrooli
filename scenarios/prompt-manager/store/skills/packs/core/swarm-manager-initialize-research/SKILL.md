# Initialize: Bootstrap Research Backlog Item

## Purpose

Bootstrap a new research backlog item with a solid foundation in one agent pass: a `conclusion.md` scaffold that captures the research question, methodology, expected deliverables, and known constraints, plus a first workshop round with targeted decisions and informational items.

Research items use `conclusion.md` as their canonical deliverable, not `plan.md`. Use this skill (not `swarm-manager-initialize-backlog`) whenever the item kind is `research`.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read research-conclusion-authoring` — canonical `conclusion.md` structure, mandatory sections, quality gates, and guardrails.

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — processing patterns and quality standards.

## Scope

**In scope:**
- Reading all existing item context (spec, archive, any user-added files)
- Creating an initial `conclusion.md` scaffold with as much content as possible
- Creating a first workshop round (`workshop/round-001.json`) with decisions, informational items, and readiness scores
- Preserving any existing artifacts on re-run (fill gaps, don't overwrite)

**Out of scope:**
- Writing `plan.md` — research items do not use plan.md; if one exists from a prior buggy run, leave it alone (the operator will clean it up) and write `conclusion.md` as the authoritative deliverable
- Modifying `archive/` — it contains user-provided materials and must not be altered by agents
- Queueing the item for execution

## Output Requirements

All writes via CLI using `--stdin` with a heredoc (avoids shell quoting issues):
```bash
swarm-manager backlog file-upload --kind research --name {{ITEM_NAME}} --path <path> --stdin <<'EOF'
<content>
EOF
```

### Required Outputs

1. **`conclusion.md`** — research conclusion scaffold following the structure defined by the `research-conclusion-authoring` skill (loaded via required reading). Fill in as much as possible from existing context (spec.json, archive materials). Leave sections as `<!-- TBD -->` when information is insufficient.
2. **`workshop/round-001.json`** — first workshop round with:
   - 4-7 targeted decisions presenting researched alternatives for the most important unknowns
   - 0-2 informational items sharing relevant findings from context
   - Honest readiness scores across 5 dimensions

## Conclusion Scaffold

The `conclusion.md` structure, mandatory sections, and quality gates are defined by the `research-conclusion-authoring` skill (loaded via required reading above). Follow that skill exactly when creating the initial scaffold.

For research items the conclusion should emphasize:
- Research question and motivation
- Methodology (what sources, what comparisons, what experiments if any)
- Expected deliverables (the actual artifacts the research will produce)
- Scope boundaries (what is and is not in scope)
- Known constraints, assumptions, and open questions

## Workshop Round Schema

See `swarm-manager-workshop` skill for the full schema. The round file includes:
- `round`: 1
- `generated_at`: ISO timestamp
- `readiness`: 5 dimension scores (0-3)
- `items`: array of decision/info items
- `plan_updates`: description of what was written to `conclusion.md`

### Readiness Dimensions

| Dimension | What It Measures |
|-----------|-----------------|
| `problem_clarity` | Is the research question well understood? |
| `scope_defined` | Are boundaries and expected deliverables defined? |
| `approach_solid` | Is there a clear methodology? |
| `testable` | Do we know how to verify the conclusion is sound? |
| `risk_awareness` | Are blockers, biases, and unknowns identified? |

Score each 0 (not started) to 3 (solid). First rounds typically score 0-2 depending on how much context exists.

## Instructions

You are initializing a Swarm Manager research backlog item. Your goal is to bootstrap it from an empty shell into a well-structured item with a draft `conclusion.md` and first workshop round ready for human review.

**Context from spec.json:**
- Kind: research
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Read all available context**

   ```bash
   swarm-manager backlog get --kind research --name {{ITEM_NAME}}
   swarm-manager backlog files --kind research --name {{ITEM_NAME}}
   ```

   Then read each available artifact:
   - `spec.json` — the item description and metadata
   - `archive/` — user-provided materials (requirements docs, prior scenario artifacts, designs)
   - Any user-added files in the item root
   - Existing `conclusion.md` and `workshop/` artifacts from a prior run (preserve these)
   - Any stale `plan.md` in the root — IGNORE for content authority; do NOT delete it (operator-managed), but treat `conclusion.md` as the only deliverable you author
   - **Initiative context** — If this item belongs to initiative `{{ITEM_INITIATIVE}}`, check for strategic context:
     ```bash
     swarm-manager initiatives get --name {{ITEM_INITIATIVE}}
     swarm-manager initiatives files --name {{ITEM_INITIATIVE}}
     ```
     Read any files present (orchestration summaries, decision logs, strategy docs). Use initiative context to align decisions with the broader initiative goals and understand how this item relates to sibling items.

2. **Check acceptance fields**

   Read `acceptance_allow`, `acceptance_deny`, and `creates` from `spec.json`:
   - If `acceptance_allow` is **set**, validate every glob against the current repo. For globs whose literal-prefix path does not exist on disk, decide whether the path is genuinely missing (stale — remove from `acceptance_allow`) or whether the research will create it (add to `creates` instead). The artifact `acceptance-validation.json` (if present in the item directory) lists exactly which globs are problematic.
   - If `acceptance_allow` is **empty**, include an acceptance decision in round-001.json asking: "What file paths will this research touch or produce?" with options: A) broad scenario-level globs (e.g., `path:scenarios/<name>/**`), B) targeted subdirectory/file paths, C) Other.

3. **Create conclusion.md scaffold**

   Based on available context, fill in as many conclusion sections as possible. The description and archive materials should provide enough for at least the research question, motivation, and partial methodology.

   ```bash
   swarm-manager backlog file-upload --kind research --name {{ITEM_NAME}} --path conclusion.md --stdin <<'EOF'
   <conclusion content>
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
   swarm-manager backlog file-upload --kind research --name {{ITEM_NAME}} --path workshop/round-001.json --stdin <<'EOF'
   <round JSON>
   EOF
   ```

5. **Score readiness**

   Evaluate each dimension honestly based on the conclusion scaffold you just created. First rounds typically range from 0-2 depending on context richness.

6. **Verify all outputs**

   ```bash
   swarm-manager backlog files --kind research --name {{ITEM_NAME}}
   ```

   Confirm `conclusion.md` and `workshop/round-001.json` were created.

### Re-run Handling

```
Do any workshop artifacts already exist?
  -> No  -> Generate everything fresh
  -> Yes -> Read existing conclusion.md and workshop rounds
          Preserve all existing answers and decisions
          Only fill gaps (missing conclusion sections, unanswered questions)
          If conclusion.md exists but no rounds: create round-001.json
          If rounds exist: create the next numbered round
```

## Anti-Patterns

- **Don't** write `plan.md` — research items use `conclusion.md` exclusively
- **Don't** overwrite existing user answers or decisions on re-run
- **Don't** generate more than 7 decisions — focus on the most impactful unknowns
- **Don't** present decisions with fewer than 2 options
- **Don't** modify files in `archive/` — these are user-provided
- **Don't** write files directly to disk — always use the backlog CLI
- **Don't** skip reading context before generating — existing materials may answer your questions
- **Don't** inflate readiness scores — be honest about the conclusion's current state

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `file-get` returns 404 | Normal on first run — generate fresh content |
| `file-upload` fails | Check that kind is `research` and name matches: `swarm-manager backlog get --kind research --name <name>` |
| Archive already contains detailed research notes | Use them as context for `conclusion.md`, generate fewer questions |
| Item has rich description | Pre-answer questions where possible, focus questions on gaps |
| `acceptance-validation.json` present with problems | Round 1 should explicitly address each missing path: remove or move to `creates` based on the research's actual touchpoints |
