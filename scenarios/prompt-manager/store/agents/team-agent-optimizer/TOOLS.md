# TOOLS

## Tool Access
- `prompt-manager skill read <skill-id>`
- `prompt-manager graph health --type agent`
- `prompt-manager graph health --type team`
- `prompt-manager graph popular --type agent`
- `prompt-manager graph popular --type team`
- `prompt-manager graph skillless-agents`
- `prompt-manager graph empty-teams`
- `prompt-manager graph node <id>`
- `prompt-manager agent show <id>`
- `prompt-manager team show <id>`
- `prompt-manager team decision-list meta-optimization ...`
- `prompt-manager team knowledge-list meta-optimization ...`

## Primary Skills
- **skill-authoring-tools** - reference for agent tool-surface proposals
- **capability-extraction** - distilling methodologies from agent files
- **team-tool-mapping** - when team structure changes touch scenario tool wiring
- **team-shared-docs-design** - choosing plan-of-record vs notebook patterns
- **visited-tracker-tools** - rotation pattern
- **documentation-health** - durable snapshots

## Usage Rules
- Every proposal names the target, evidence, expected delta, and measurement plan.
- Do not touch skills or scenario code.
- Team-structure proposals default to the smallest useful change.
