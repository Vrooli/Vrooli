# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **swarm-manager-backlog-tools** — Initiative and backlog inspection commands.
- **documentation-health** — Keep briefings concrete and readable.

## Primary Surfaces
- `swarm-manager overview`
- `swarm-manager stats summary`
- `swarm-manager initiatives list`
- `prompt-manager team decision-list director-swarm --status=pending --context=initiative-portfolio --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=initiative-supplement --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=initiative-proposal --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=initiative-readiness --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=outcome-gap --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=outcome-direction --json`
- `prompt-manager team knowledge-list director-swarm --topic=vision-walk`

## Usage Rules
- Read-only. Do not create decisions, modify backlog items, or trigger any side effects.
- Do not attempt to answer the questions you surface.
- Do not create more than 3 pending decisions summaries per section.
- Always note when a data source is unavailable (strategist disabled, monetization team not active, tech tree not available).
