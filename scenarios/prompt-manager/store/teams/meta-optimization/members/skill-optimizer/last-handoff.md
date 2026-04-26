### Skill picked this heartbeat
- `skill-principles` — top of revisit queue: 29 unique inbound consumers (largest skill fanout in current audit; central to all leader-*, plan-skill-discovery, every skill-authoring* variant, morning-vision-walk), never previously visited.

### Disposition
- **improve** — propose §3 duplication trim.

### Baseline
- Tokens: ~1,960 (171 lines / 7,831 chars)
- Inbound: 29 unique consumers
- Health: 0.65 (only flag is generic external-tooling warning over legitimate `cli:jq` + `cli:prompt-manager`)
- Drift age: fresh (first visit)
- Trim target: §3 has both a 6-row category table AND an isomorphic decision-tree code block immediately below it — ~580 chars / ~145 tokens / 16 lines of literal duplication.

### Expected delta (if change proposed)
- Per read: ~1,960 → ~1,815 tokens (−7.4%).
- Per full-fanout load (29 consumers): ~4,200 tokens saved.
- Information loss: zero (table strictly more informative — preserves the optimization-axis column the tree omits).
- Measurement: pre/post `wc -c` (7,831 → ~7,250), graph health holds ≥0.65, 14-heartbeat revisit confirms no consumer added a workaround for the missing tree.

### Artifacts updated
- SKILL_AUDIT.md: skill-principles row added (improve); revisit queue reordered (visited-tracker-tools → #1, skill-principles moved to recently-visited slot).
- PROGRAMMATIC_CONVERSION_QUEUE.md: unchanged (rejected — judgment-irreducible).
- DEPRECATION_QUEUE.md: unchanged (rejected — 29 consumers).

### Decisions raised this heartbeat
- `dec-1777155425370344769` · `skill-improvement` · Trim skill-principles §3 duplicate decision-tree (keep the table).

### Knowledge entries written
- `knw-1777155390747515665` · `skill-visited/skill-principles` (first visit; no prior to supersede)
- `knw-1777155399201273952` · `skill-audit-2026-04-25` (supersedes `knw-1777069027821885714` / skill-audit-2026-04-24)