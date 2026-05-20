### Skill picked this heartbeat
- `browser-automation-studio` - usage-weighted ladder; skipped `knowledge-observatory-tools`, `visited-tracker-tools`, and `swarm-manager-backlog-tools` due pending skill-optimizer decisions; skipped `architecture-scope` and `screaming-architecture-audit` as recently visited/no fresh drift.

### Disposition
- improve

### Baseline
- Tokens: 590 lines / 2,678 words / 23,432 chars
- Usage: 8 inbound consumers
- Drift age: no prior visit found; fresh drift found 2026-05-20

### Expected delta
- Remove immediately failing singular BAS command examples and one prompt-manager typo. Measure by confirming the skill no longer contains `browser-automation-studio workflow `, `browser-automation-studio session `, or `prompt-manager skills read`, and that documented groups match `browser-automation-studio workflows --help`, `session-profiles --help`, and `schema --help`.

### Artifacts updated
- SKILL_AUDIT.md: unchanged; write surface allowed knowledge/decisions/handoff only
- ACTION_AUDIT.md: unchanged as working file; knowledge snapshot written
- ACTION_CONVERSION_QUEUE.md: unchanged
- DEPRECATION_QUEUE.md: unchanged

### Action check
- Discover: `prompt-manager discover "browser automation studio capture screenshot console logs network workflow audit" --type all` returned existing BAS Actions: `bas.audit`, `bas.screenshot`, `bas.console-logs`, `bas.status`, `bas.network`, `bas.screenshot.mobile`
- Existing Action inspected: `bas.audit`, `bas.screenshot`
- Validation: `prompt-manager action validate bas.audit` and `prompt-manager action validate bas.screenshot` both valid/runnable, with warning that owning scenario lacks `cli/manifest.json` governance

### Decisions raised this heartbeat
- `dec-1779315407263556076` - `skill-improvement` - update `browser-automation-studio` to current BAS plural command groups and fix stale examples

### Knowledge entries written
- `knw-1779315470516975876` - `skill-visited/browser-automation-studio`
- `knw-1779315470516166434` - `skill-audit/2026-05-20`
- `knw-1779315470655486789` - `action-visited/bas.audit`
- `knw-1779315470655279028` - `action-visited/bas.screenshot`
- `knw-1779315470655869099` - `action-audit/2026-05-20`