# Workshop: Finalize Plan Synthesis

## Purpose

Run the final synthesis pass for a non-research backlog item. Incorporate the latest answered workshop decisions into `plan.md`, reassess readiness against the updated plan, and write a finalize round that contains no new decisions.

This skill exists specifically to avoid the stale-plan gap after the user answers the final round. It is not a normal workshop round and must not generate fresh questions.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read implementation-plan-authoring` — canonical plan structure, mandatory sections, quality gates, and guardrails for `plan.md`.

**Required reading:** `prompt-manager skill read plan-skill-discovery` — use the existing required reading already embedded in the plan when synthesizing the final draft.

## Scope

**In scope:**
- Read the current `plan.md`, all prior workshop rounds, `spec.json`, optional research artifacts, and user files
- Incorporate the latest answered decisions into `plan.md`
- Re-score the 5 readiness dimensions based on the updated plan
- Write a finalize round (`workshop/round-NNN.json`) with zero decision items

**Out of scope:**
- Generating new decisions or proposals
- Asking clarifying questions
- Implementing the backlog item
- Writing `handoff/` artifacts directly — swarm-manager derives idea handoff packages later from the finalized backlog state when execution begins
- Modifying `archive/`

## Output Requirements

All writes via CLI using `--stdin` with a heredoc:
```bash
swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path <path> --stdin <<'EOF'
<content>
EOF
```

### This Pass Produces

1. **`plan.md`** — updated so it reflects the latest answered workshop decisions
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
      "text": "Latest answered decisions were incorporated into the plan. Remaining gaps are ..."
    }
  ],
  "plan_updates": "Brief summary of what changed in the plan during finalization"
}
```

## Rules

- `items` must contain **only** `info` items. Zero decisions.
- Do not ask any new questions, even if gaps remain.
- If readiness is still below the threshold after synthesis, reflect that honestly in scores and info items so the user can choose to run another normal workshop round.
- Preserve the audit trail: write a new finalize round instead of mutating prior rounds.

## Instructions

You are running finalize round {{ROUND_NUMBER}} for a swarm-manager backlog item.

**Item context:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. Read the backlog item and all available artifacts:
   ```bash
   swarm-manager backlog get --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   swarm-manager backlog files --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   ```
   Then read:
   - `plan.md`
   - every file in `workshop/`
   - `spec.json`
   - `research/summary.md` if present
   - user-provided files
   - **Initiative context** — If this item belongs to initiative `{{ITEM_INITIATIVE}}`, check for strategic context:
     ```bash
     swarm-manager initiatives get --name {{ITEM_INITIATIVE}}
     swarm-manager initiatives files --name {{ITEM_INITIATIVE}}
     ```
     Read any files present (orchestration summaries, decision logs, strategy docs). Use initiative context to align decisions with the broader initiative goals and understand how this item relates to sibling items.

2. Find the latest answered workshop round.
   - Treat selected options, freeform text, and notes as authoritative user intent.
   - Incorporate those choices into `plan.md`.

3. Update `plan.md`.
   - Make the plan internally consistent with the latest answers.
   - Tighten sections that changed because of those answers.
   - Do not introduce placeholder questions or “TBD” sections unless the information is genuinely missing.

4. Re-score readiness against the updated plan using the normal 0-3 rubric.
   - Be honest.
   - If the plan is still not ready, note the remaining weak dimensions in info items.

5. Write `plan.md`.

6. Write `workshop/round-{{ROUND_NUMBER}}.json`.
   - Set `"mode": "finalize"`.
   - Set `"pending_synthesis": false`.
   - Include only info items.
   - Summarize the synthesis outcome and any remaining gaps.

### Important

Do not create or update `handoff/` files during workshop finalization. The finalized `plan.md` and workshop state are the authoritative sources; swarm-manager regenerates the downstream idea handoff package from those files at process-time.
