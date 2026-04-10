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
- `prompt-manager graph orphaned-skills [--limit N]` — Unreferenced skills.
- `prompt-manager graph cliless-skills [--limit N]` — Skills without CLI promotion.
- `prompt-manager graph popular --type skill [--limit N]` — Most-referenced skills.
- `prompt-manager graph node <id> [--json]` — Inspect a skill's connections and health breakdown.

## Development Toolchain Validation

*Available when development-toolchain-validator ships.*

- `development-toolchain-validator report --conflicts` — Cross-skill contradictions.
- `development-toolchain-validator report --maturity` — Skill configurability and maturity scores.
- `development-toolchain-validator report --drift [--skill <id>]` — Skills changed since last validation.

## Usage Rules

- Follow the appropriate authoring guide for each skill type.
- Validate every change against skill-validation criteria.
- Resolve cross-skill conflicts before optimizing individual skill quality.
- Track health scores before and after changes.
