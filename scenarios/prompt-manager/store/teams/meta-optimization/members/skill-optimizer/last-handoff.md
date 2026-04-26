### Skill picked this heartbeat
- `visited-tracker-tools` — top of revisit queue, never-visited, popular (25 unique inbound consumers; bold-listed by both skill-optimizer and team-agent-optimizer; cli-read by every steer-family skill + leader-research-analyze-plan + documentation-health + skill-authoring).

### Disposition
- **improve** — drift fix. Replace the broken `visited-tracker status` example (§1 lines 18-23) with the canonical `visited-tracker coverage`.

### Baseline
- Tokens: ~560 (2,237 chars / 71 lines)
- Inbound: 25 unique consumers
- Health: 0.72
- Drift: CONFIRMED via direct CLI call — `visited-tracker status` returns `Error: Unknown command: status`. CLI top-level groups are `campaigns` + `files`; analytics commands are `least-visited` / `most-stale` / `coverage`. Replacement target verified: `visited-tracker coverage --location . --tag T` returns the Total/Visited (with %)/Unvisited/Average visits/Average staleness report — exactly what the broken example was trying to surface.
- Other 4 documented commands (`least-visited`, `visit`, `exclude`, `campaigns note`) verified working — no drift.

### Expected delta (if change proposed)
- Correctness: 1 of 5 documented commands shifts BROKEN → WORKING for all 25 consumers.
- Token cost: ~560 → ~560 (replacement words are same length — this is a correctness fix, not a trim).
- Trust signal: removes visible drift from a high-fanout tools-named skill.
- Measurement: pre/post CLI invocation (already verified this heartbeat); graph node health holds ≥0.72; 14-HB revisit confirms no consumer copied the broken `status` example into its own guidance.

### Artifacts updated
- SKILL_AUDIT.md: visited-tracker-tools row added (improve, drift); revisit queue reordered (knowledge-observatory-tools → #1, visited-tracker-tools moved to recently-visited slot, added architecture-scope + systematic-exploration as never-visited popular skills).
- PROGRAMMATIC_CONVERSION_QUEUE.md: unchanged (rejected — already a thin wrapper).
- DEPRECATION_QUEUE.md: unchanged (rejected — 25 consumers).

### Decisions raised this heartbeat
- `dec-1777241982993901596` · `skill-improvement` · Replace broken `visited-tracker status` example with `visited-tracker coverage` in visited-tracker-tools §1.

### Knowledge entries written
- `knw-1777241942565284325` · `skill-visited/visited-tracker-tools` (first visit; no prior to supersede)
- `knw-1777241953087310075` · `skill-audit-2026-04-26` (supersedes `knw-1777155399201273952` / skill-audit-2026-04-25)