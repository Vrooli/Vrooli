# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **documentation-health** — Keep outcome recommendations and gap writeups concrete.

## Primary Surfaces
- `command-center gaps --json` when available
- future Command Center metrics endpoints
- `prompt-manager team decision-list director-swarm --status=<status> --context=<context>`
- `prompt-manager team knowledge-list director-swarm --topic=decision-application/<decision-id>`

## Usage Rules
- If Command Center is not ready, say so and stop.
- Apply accepted decisions before proposing new ones.
- Do not create more than 3 new decisions in one run.
- Recommend outcome-driven portfolio changes and data-pipeline work, not direct execution.
