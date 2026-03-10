# Initialize: Bootstrap Backlog Item

## Purpose

Bootstrap a new backlog item with a solid foundation in one agent pass: clarifying questions, suggestions, a refined summary, and (for idea/execute kinds) PRD + requirements generation via prd-control-tower.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — processing patterns and quality standards.

## Scope

**In scope:**
- Reading all existing item context (spec, archive, any user-added files)
- Generating clarifying questions (3–5, pre-answered where inferable from context)
- Generating suggestions (2–3 high-impact improvements)
- Producing a refined summary (`enhance/summary.md`)
- For idea/execute kinds: generating PRD context and invoking prd-control-tower
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

1. **`clarify/questions.json`** — 3–5 targeted questions
2. **`enhance/summary.md`** — refined plan/summary

### idea and execute Kinds (additionally)

3. **`suggest/suggestions.json`** — 2–3 high-impact suggestions
4. **`enhance/prd-context.md`** — context brief for PRD generation
5. **PRD + requirements** via prd-control-tower CLI:
   ```bash
   prd-control-tower prd generate --path <item-folder>/archive
   ```

### fix Kind (additionally)

3. **`suggest/suggestions.json`** — 2–3 fix approach suggestions
4. **`enhance/summary.md`** should focus on investigation plan and root cause analysis

### research Kind

3. **`enhance/summary.md`** should focus on research plan outline, methodology, and expected deliverables
   - Skip suggestions and PRD for research items

## Instructions

You are initializing a Swarm Manager backlog item. Your goal is to bootstrap it from an empty shell into a well-structured item ready for human review and refinement.

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
   - Existing `clarify/`, `suggest/`, `enhance/` artifacts from a prior run (preserve these)

2. **Generate clarifying questions** (`clarify/questions.json`)

   Identify the 3–5 most important unknowns. For each question:
   - If the answer can be inferred from context (description, archive materials), pre-fill the `answer` field
   - Provide `options` where applicable
   - Categorize as: users, technical, scope, constraints, or integration
   - Classify importance as: critical, important, or nice-to-have

   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path clarify/questions.json --stdin <<'EOF'
   {
     "questions": [
       {
         "id": "q1",
         "question": "...",
         "options": ["..."],
         "answer": "...",
         "category": "...",
         "importance": "critical"
       }
     ],
     "generatedAt": "<ISO timestamp>",
     "updatedAt": "<ISO timestamp>"
   }
   EOF
   ```

3. **Generate suggestions** (skip for research kind)

   Produce 2–3 high-impact suggestions appropriate for the item kind:
   - **idea/execute**: Feature improvements, architecture recommendations, UX enhancements
   - **fix**: Investigation approaches, fix strategies, prevention measures

   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path suggest/suggestions.json --stdin <<'EOF'
   {
     "suggestions": [
       {
         "id": "s1",
         "suggestion": "...",
         "details": "...",
         "status": "pending"
       }
     ],
     "generatedAt": "<ISO timestamp>",
     "updatedAt": "<ISO timestamp>"
   }
   EOF
   ```

4. **Generate refined summary** (`enhance/summary.md`)

   Synthesize all context into a structured plan:
   - **idea/execute**: Vision, scope, architecture outline, key decisions, next steps
   - **fix**: Problem statement, reproduction steps, root cause analysis, proposed fix, verification plan
   - **research**: Research question, methodology, data sources, expected deliverables, timeline

   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path enhance/summary.md --stdin <<'EOF'
   <markdown content>
   EOF
   ```

5. **Generate PRD** (idea/execute only)

   First, create a PRD context brief:
   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path enhance/prd-context.md --stdin <<'EOF'
   <context brief for PRD generation>
   EOF
   ```

   Then invoke prd-control-tower to generate PRD and requirements:
   ```bash
   prd-control-tower prd generate --path {{ITEM_FOLDER}}/archive
   ```

   If prd-control-tower is not available, skip this step and note it in the summary.

6. **Verify all outputs**

   ```bash
   swarm-manager backlog files --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   ```

   Confirm all expected artifacts were created.

### Re-run Handling

```
Do any initialize artifacts already exist?
  → No  → Generate everything fresh
  → Yes → Read existing artifacts
          Preserve all existing answers and decisions
          Only fill gaps (missing files or empty sections)
          Update timestamps on modified files
```

## Anti-Patterns

- **Don't** overwrite existing user answers or decisions on re-run
- **Don't** generate more than 5 questions — focus on the most impactful unknowns
- **Don't** generate more than 3 suggestions — quality over quantity
- **Don't** modify files in `archive/` — these are user-provided
- **Don't** write files directly to disk — always use the backlog CLI
- **Don't** skip reading context before generating — existing materials may answer your questions

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `file-get` returns 404 | Normal on first run — generate fresh content |
| `file-upload` fails | Check kind and name match: `swarm-manager backlog get --kind <kind> --name <name>` |
| prd-control-tower not available | Skip PRD generation, note in summary.md |
| Archive already contains detailed PRD | Use it as context, don't regenerate — focus on gaps |
| Item has rich description | Pre-answer questions where possible, generate fewer |
