# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **capability-extraction** — distilling reusable patterns from doc entries
- **scientific-debugging** — tracing recurring friction to root cause
- **documentation-health** — durable scan snapshots
- **team-shared-docs-design** — reference for promotion-path decisions (notebook → plan-of-record vs notebook → permanent structure)

## Primary Surfaces
- `docs/meta-optimization/README.md`, `CONVERSION_PLAYBOOK.md`, `DEPRECATION_POLICY.md`, `REFERENCE_SCENARIOS.md` (+ any future files)
- `shared/RUN_LESSONS.md`, `SKILL_AUDIT.md`, `PROGRAMMATIC_CONVERSION_QUEUE.md`, `DEPRECATION_QUEUE.md`, `TEAM_AUDIT.md`, `AGENT_AUDIT.md`, `TOOLCHAIN_SCAN.md`
- `prompt-manager team show meta-optimization` (current team surface)
- `prompt-manager skill list` (what already exists)
- `vrooli help` and `scenarios/<name>/cli/` (what tooling already exists)
- `prompt-manager team decision-list meta-optimization --status=pending --context=meta-self-improvement`
- `prompt-manager team knowledge-list meta-optimization --topic-prefix=debt-scan-`

## Usage Rules
- Never write to `docs/meta-optimization/` directly. Retirements go through decisions.
- Never edit skills, agents, team configs, or scenarios. Route through the owning implementer.
- Every proposal cites source entries + promotion direction + owning implementer + measurement plan. No exceptions.
- Cap decisions at 1 per heartbeat — this is intentionally tighter than other members.
- Quiet heartbeats are correct behavior when docs haven't accumulated ripe debt yet.
