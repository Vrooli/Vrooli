### Skill picked this heartbeat
- `knowledge-observatory-tools` - usage-weighted ladder: highest popular skill, active, CLI-backed, 25 inbound consumers, last visited 2026-04-27.

### Disposition
- improve

### Baseline
- Tokens: ~650 tokens / 338 words / 77 lines / 2,617 chars
- Usage: 25 inbound consumers
- Drift age: previous validation 2026-04-27; content unchanged since 2026-02-06, but CLI/doc-contract drift found today

### Expected delta
- Replace the stale universal `Supported types` list with manifest-driven guidance. Measured by verifying the skill no longer claims `readme`/`prd`/`manifest`/`assumptions` are universally supported, and directs users to `docs health` or `search-files` when doc types vary by scenario contract.

### Artifacts updated
- SKILL_AUDIT.md: unchanged; write surface allowed knowledge/decisions/handoff only
- ACTION_AUDIT.md: unchanged
- ACTION_CONVERSION_QUEUE.md: unchanged
- DEPRECATION_QUEUE.md: unchanged

### Action check
- Discover: `prompt-manager discover "knowledge observatory docs read add search health reset" --type all` returned related skills only, no exact Action
- Existing Action inspected: none
- Validation: no Action validation run. Live CLI smoke checks showed `knowledge-observatory docs read knowledge-observatory readme/prd/manifest` succeeds, `knowledge-observatory docs read knowledge-observatory assumptions` fails, `reference-react-vite quickstart` succeeds, and `reference-react-vite readme/prd/manifest` fail due scenario contract issues.

### Decisions raised this heartbeat
- `dec-1778969859880763739` - `skill-improvement` - update `knowledge-observatory-tools` so doc-type guidance is manifest-driven instead of a universal hard-coded list.

### Knowledge entries written
- `knw-1778969876811109951` - `skill-visited/knowledge-observatory-tools`
- `knw-1778969876811409001` - `skill-audit/2026-05-16`
- `action-visited/<action-id>`: not applicable
- `action-audit/YYYY-MM-DD`: not applicable; Action audit unchanged