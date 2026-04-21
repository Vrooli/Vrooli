# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **skill-authoring-tools** — standards for thin-wrapper skills
- **skill-validation** — post-edit validation
- **skill-principles** — universal quality criteria
- **visited-tracker-tools** — rotation pattern
- **documentation-health** — durable snapshots

## Primary Surfaces
- `prompt-manager graph health --type skill`
- `prompt-manager graph popular --type skill`
- `prompt-manager graph orphaned-skills`
- `prompt-manager graph cliless-skills`
- `prompt-manager graph circular-refs`
- `prompt-manager graph node <skill-id>`
- `prompt-manager skill read <skill-id>`
- `prompt-manager skill update <skill-id>` (for direct edits via decisions)
- `vrooli help` and `scenarios/<name>/cli/` for conversion targets
- `shared/SKILL_AUDIT.md`, `PROGRAMMATIC_CONVERSION_QUEUE.md`, `DEPRECATION_QUEUE.md`
- `shared/RUN_LESSONS.md` (usage signals from run-introspector)
- `prompt-manager team decision-list meta-optimization --status=pending --context=skill-*`
- `prompt-manager team knowledge-list meta-optimization --topic-prefix=skill-visited/`

## Usage Rules
- Every proposal includes a baseline + expected delta + measurement plan. No exceptions.
- Conversion > polishing > pruning choice is in that order *only when* usage justifies it. If usage is zero, pruning jumps to the front.
- Do not edit agent files, team configs, or scenario code. Cross-lane proposals are rejected by the contrarian (failure mode 6).
- Cap decisions at 2 per heartbeat.
