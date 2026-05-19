### Skill picked this heartbeat
- `swarm-manager-backlog-tools` - usage-weighted ladder; skipped recently visited/pending `knowledge-observatory-tools` and `visited-tracker-tools`, then selected next popular active skill with no prior visit found.

### Disposition
- improve

### Baseline
- Tokens: 527 lines / 3,399 words / 27,685 chars
- Usage: 13 inbound consumers
- Drift age: unknown/no prior visit found

### Expected delta
- Split/update the oversized canonical reference so the primary skill is materially smaller, no longer triggers the oversized graph warning, and includes current `swarm-manager backlog` / `initiatives` lifecycle surfaces. Measure with `prompt-manager graph node swarm-manager-backlog-tools`, `prompt-manager skill read`, and CLI help parity checks.

### Artifacts updated
- SKILL_AUDIT.md: unchanged; write surface allowed knowledge/decisions/handoff only
- ACTION_AUDIT.md: unchanged
- ACTION_CONVERSION_QUEUE.md: unchanged
- DEPRECATION_QUEUE.md: unchanged

### Action check
- Discover: `prompt-manager discover "swarm manager backlog create item list update notes dependencies" --type all` returned related skills only, no exact Action.
- Existing Action inspected: none
- Validation: not run; no exact Action found, and deterministic owner is already the `swarm-manager` CLI family.

### Decisions raised this heartbeat
- `dec-1779142599530774097` - `skill-improvement` - update/split `swarm-manager-backlog-tools` to match current CLI and reduce oversized catch-all content.

### Knowledge entries written
- `knw-1779142621672207394` - `skill-visited/swarm-manager-backlog-tools`
- `knw-1779142621671485623` - `skill-audit/2026-05-18`
- `action-visited/<action-id>`: not applicable
- `action-audit/YYYY-MM-DD`: not applicable; Action audit unchanged