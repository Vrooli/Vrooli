# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **scientific-debugging** — isolating the specific flaw in a proposal
- **documentation-health** — concrete, durable challenge notes

## Primary Surfaces
- `prompt-manager team decision-list meta-optimization --status=pending --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=<each>`
- `shared/TEAM.md` (framework)
- `shared/SKILL_AUDIT.md`, `PROGRAMMATIC_CONVERSION_QUEUE.md`, `DEPRECATION_QUEUE.md`
- `shared/TEAM_AUDIT.md`, `AGENT_AUDIT.md`
- `shared/RUN_LESSONS.md`, `TOOLCHAIN_SCAN.md`
- `prompt-manager team knowledge-list meta-optimization --topic-prefix=challenge-note/`
- `prompt-manager team knowledge-create` (for challenge notes)

## Usage Rules
- Challenge notes are append-only — never include a `supersedes` field on them.
- Every challenge is `flagged` or `cleared` per failure mode. No in-between.
- Specific revisions only: "Revision that would pass: ..." should name the exact missing element.
- Cap rejection decisions at 2 per heartbeat; framework-update at 1 per heartbeat.
- Aging scan runs every heartbeat, including read-only mode. No exceptions.
- Never propose positive actions.
