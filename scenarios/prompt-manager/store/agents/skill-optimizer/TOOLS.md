# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Skill Management Commands
- `prompt-manager skill list` — List all skills.
- `prompt-manager skill show <id>` — View skill details.
- `prompt-manager skill update <id>` — Update a skill.
- `prompt-manager skill add <name>` — Create a new skill.
- `prompt-manager skill rate <id>` — Rate effectiveness.
- `prompt-manager skill versions <id>` — View version history.
- `prompt-manager skill revert <id> <version>` — Revert if needed.

## Graph Analysis Commands
- `prompt-manager graph health --type skill` — Skill health scores (sorted, lowest = most attention needed).
- `prompt-manager graph orphaned-skills [--limit N]` — Skills not referenced by any agent.
- `prompt-manager graph cliless-skills [--limit N]` — Skills with no CLI code references.
- `prompt-manager graph popular --type skill [--limit N]` — Most-referenced skills.
- `prompt-manager graph node <id> [--json]` — Inspect a specific skill's connections and health breakdown.

## Usage Rules
- Follow the appropriate authoring guide for each skill type.
- Validate every change against skill-validation criteria.
- Track health scores before and after changes.
