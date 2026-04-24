# Skill Audit — Rolling

Usage-weighted priority audit of the skill library. Maintained by `skill-optimizer`.

## Column legend
- **Skill** — skill id
- **Rating** — rough overall health: `high` (popular + healthy), `mixed` (popular but flags), `low` (low use), `drift` (changed-since-validation)
- **Last visited** — ISO date of most recent `skill-visited/<skill-id>` knowledge entry
- **Disposition** — `convert` | `improve` | `prune` | `no-action`
- **Notes** — one-line context

## Audit rows

| Skill | Inbound | Health | Rating | Last visited | Disposition | Notes |
|-------|---------|--------|--------|--------------|-------------|-------|
| swarm-manager-backlog-tools | 19 | 0.51 (oversized) | mixed | 2026-04-23 | improve | CLI-commands section (~3K tokens / 258 lines) duplicates `swarm-manager <group> --help`. Trim to pointer + non-obvious notes; saves ~3K tokens × 19 consumers per load. |
| documentation-health | 26 (unique) | 0.81 | high | 2026-04-24 | no-action | ~3,500 tokens across judgment prose (1, 8-10) + stable reference schemas (3-5) + thin CLI wrappers (6, 11). Already cites `knowledge-observatory docs audit/templates/template`. Section 4 reference-format examples are mildly redundant (~375 tokens trimmable × 26 consumers), but payoff is small relative to risk of breaking 26 downstream consumers that rely on the format-spec surface. Revisit if health regresses or section 4 drifts. |

## Revisit queue (priority order)
1. skill-principles (popular, not yet visited) — judgment-based; audit for clarity
2. visited-tracker-tools (popular, tools-named; check for CLI parity with tracker)
3. knowledge-observatory-tools (popular, tools-named; similar check)
4. scientific-debugging (popular, methodology skill; likely polish-only)
5. documentation-health — revisit in ~14 heartbeats or on drift
