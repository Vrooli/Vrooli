# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **scientific-debugging** — isolating specific causes
- **conversation-friction-analysis** — extracting interaction-flow lessons
- **capability-extraction** — distilling reusable patterns from a successful run
- **documentation-health** — durable lesson writeups

## Primary Surfaces
- Agent-manager run list + investigation command (preferred)
- Run manifest / transcript / artifact reads (fallback)
- Agent AGENTS.md / SOUL.md / TOOLS.md files
- The skills referenced during the run
- `shared/RUN_LESSONS.md`
- `prompt-manager team decision-list meta-optimization --status=pending --context=run-lesson | capability-gap`
- `prompt-manager team knowledge-list meta-optimization --topic-prefix=run-lessons-`

## Usage Rules
- One run per heartbeat. No exceptions.
- Never edit skills, agents, or team configs. Lessons only.
- Every lesson names a run ID, an implicated skill/agent/prompt passage, a proposed change target, and a measurement plan.
- Cap decisions at 2 per heartbeat.
