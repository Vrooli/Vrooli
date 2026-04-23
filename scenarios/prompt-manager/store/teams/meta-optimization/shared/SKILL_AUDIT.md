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

## Revisit queue (priority order)
1. documentation-health (popular, health 0.81, not yet visited) — probably irreducibly judgment-based; audit for clarity only
2. skill-principles (popular, not yet visited) — judgment-based; audit for clarity
3. visited-tracker-tools (popular, tools-named; check for CLI parity with tracker)
4. knowledge-observatory-tools (popular, tools-named; similar check)
5. scientific-debugging (popular, methodology skill; likely polish-only)
