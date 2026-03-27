# Workshop: Finalize Research Synthesis

## Purpose

Run the final synthesis pass for a research backlog item. Incorporate the latest answered workshop decisions into `conclusion.md`, reassess readiness against the updated conclusion, and write a finalize round that contains no new decisions.

This pass exists to ensure the conclusion reflects the final answered round before the item is treated as ready.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read research-conclusion-authoring` — canonical conclusion structure, mandatory sections, quality gates, and guardrails for `conclusion.md`.

## Scope

**In scope:**
- Read `conclusion.md`, all prior workshop rounds, `spec.json`, archive materials, and user files
- Incorporate the latest answered decisions into `conclusion.md`
- Re-score the 5 research readiness dimensions against the updated conclusion
- Write a finalize round with zero decision items

**Out of scope:**
- Generating new decisions
- Asking new clarifying questions
- Implementing code changes
- Modifying `archive/`

## Output Requirements

All writes via CLI using `--stdin` with a heredoc:
```bash
swarm-manager backlog file-upload --kind research --name {{ITEM_NAME}} --path <path> --stdin <<'EOF'
<content>
EOF
```

### This Pass Produces

1. **`conclusion.md`** — updated so it reflects the latest answered workshop decisions
2. **`workshop/round-{{ROUND_NUMBER}}.json`** — finalize round with this shape:

```json
{
  "round": {{ROUND_NUMBER}},
  "generated_at": "<ISO-8601 timestamp>",
  "mode": "finalize",
  "pending_synthesis": false,
  "readiness": {
    "problem_clarity": 0,
    "scope_defined": 0,
    "approach_solid": 0,
    "testable": 0,
    "risk_awareness": 0
  },
  "items": [
    {
      "id": "i1",
      "type": "info",
      "text": "Latest answered decisions were incorporated into the conclusion. Remaining gaps are ..."
    }
  ],
  "plan_updates": "Brief summary of what changed in the conclusion during finalization"
}
```

## Rules

- `items` must contain only `info` items.
- Do not ask new questions.
- If the conclusion is still not ready after synthesis, reflect that honestly so the user can choose to run another normal workshop round.
- Write a new finalize round; do not rewrite history.

## Instructions

You are running finalize round {{ROUND_NUMBER}} for a swarm-manager research backlog item.

**Item context:**
- Kind: research
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. Read the backlog item and all available artifacts:
   ```bash
   swarm-manager backlog get --kind research --name {{ITEM_NAME}}
   swarm-manager backlog files --kind research --name {{ITEM_NAME}}
   ```
   Then read:
   - `conclusion.md`
   - every file in `workshop/`
   - `spec.json`
   - archive materials
   - user-provided files

2. Find the latest answered workshop round.
   - Treat selected options, freeform text, and notes as authoritative user direction.
   - Incorporate that direction into `conclusion.md`.

3. Update `conclusion.md`.
   - Make the conclusion internally consistent with the latest answers.
   - Tighten findings, implications, and next steps as needed.

4. Re-score readiness against the updated conclusion using the research-oriented rubric.
   - Be honest about confidence, gaps, and remaining limitations.

5. Write `conclusion.md`.

6. Write `workshop/round-{{ROUND_NUMBER}}.json`.
   - Set `"mode": "finalize"`.
   - Set `"pending_synthesis": false`.
   - Include only info items.
   - Summarize the synthesis outcome and any remaining gaps.
