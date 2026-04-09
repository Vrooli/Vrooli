# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **swarm-manager-backlog-tools** — Initiative and backlog inspection commands.
- **swarm-manager-recommendations** — Approval-gated backlog proposal authoring.
- **documentation-health** — Keep decisions, proposals, and handoffs concrete and readable.

## Primary Surfaces
- `swarm-manager overview`
- `swarm-manager initiatives list`
- `swarm-manager initiatives get --name <initiative>`
- `swarm-manager stats summary`
- `prompt-manager team decision-list director-swarm --status=<status> --context=<context>`
- `prompt-manager team knowledge-list director-swarm --topic=decision-application/<decision-id>`

## Usage Rules
- Apply accepted decisions before creating new ones.
- Stop early when there are already 3 unresolved relevant pending decisions.
- Do not create more than 3 new decisions in one run.
- Do not deploy teams or create backlog items without approval; frame them as proposals when approval is missing.
