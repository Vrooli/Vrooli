### Skill picked this heartbeat
- `swarm-manager-backlog-tools` — top of priority ladder: 19 inbound consumers (popular) + health 0.51 with "oversized content" warning + never-visited.

### Disposition
- **improve** (trim). Not a conversion candidate — it's a reference doc, not a behavior skill. But the CLI Commands section duplicates `swarm-manager <group> --help` which the CLI already emits.

### Baseline
- Tokens: ~6,900 (527 lines / 27,565 chars)
- CLI Commands section alone: ~3,000 tokens (258 lines / 12,286 chars, 44% of skill)
- Inbound consumers: 19
- Drift age: fresh (no prior audit; first visit)

### Expected delta (if change proposed)
- Replace CLI Commands section with pointer to `swarm-manager <group> --help` + a ~15-20-line "non-obvious usage patterns" block (stdin heredoc, batch --preview, referential-integrity auto-maintenance, initiatives-context single-call loader)
- Target skill size: ~4,400 tokens (~36% total reduction; ~80% reduction of the CLI section)
- Per-read savings: ~2,500 tokens × 19 consumers every load
- Measurement: re-count after edit; re-run `graph health` to confirm oversized-content warning drops; spot-check one downstream skill still has enough surface; 14-heartbeat revisit for drift

### Artifacts updated
- SKILL_AUDIT.md: **created** (did not exist); added swarm-manager-backlog-tools row + revisit queue
- PROGRAMMATIC_CONVERSION_QUEUE.md: **created**; noted as "trim candidate, not thin-wrapper conversion"
- DEPRECATION_QUEUE.md: **created** (empty)

### Decisions raised this heartbeat
- `dec-1776982635141465033` · `skill-improvement` · Trim swarm-manager-backlog-tools CLI Commands section into a --help pointer + non-obvious notes (expected -2,500 tokens/read × 19 consumers)

### Knowledge entries written
- `knw-1776982616473003313` · `skill-visited/swarm-manager-backlog-tools` (first visit; no prior to supersede)
- `knw-1776982619949185628` · `skill-audit-2026-04-23` (first audit snapshot; no prior to supersede)