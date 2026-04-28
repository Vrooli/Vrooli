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
| skill-principles | 29 (unique) | 0.65 | high | 2026-04-25 | improve | ~1,960 tokens / 171 lines. Judgment-heavy and mostly irreducible (categories, sprawl, lifecycle, architecture heuristics) — not a conversion candidate. Section 3 contains a category-→-decision table AND an isomorphic decision-tree code block immediately below it (~145 tokens of literal duplication). Removing the tree (keeping the table — more scannable) saves ~145 tokens × 29 consumers per full-fanout load. Health 0.65 flag is the generic "external tooling dependency" warning over `cli:jq` + `cli:prompt-manager`, both legitimate — no action there. |
| visited-tracker-tools | 25 (unique) | 0.72 | drift | 2026-04-26 | improve | ~560 tokens / 71 lines / 2,237 chars. Already a thin wrapper over `visited-tracker` CLI — NOT a conversion candidate. **Drift confirmed**: §1 documents `visited-tracker status` (lines 18-23) but no such command exists in the CLI (`Error: Unknown command: status`). Canonical replacement is `visited-tracker coverage` (Analytics group; surfaces Total/Visited/Unvisited/Average visits/Average staleness — exactly what the broken example was trying to show). All other documented commands (`least-visited`, `visit`, `exclude`, `campaigns note`) verified working. Secondary minor: §2 cites `coverage_percent` as a field name, but the `coverage` command's human/JSON output presents it as `Visited: N (X%)` — interpretive guidance still works, not bundling into this decision. |
| knowledge-observatory-tools | 18 | 0.72 | high | 2026-04-27 | no-action | ~654 tokens / 77 lines / 2,617 chars. Already a thin wrapper over `knowledge-observatory docs` CLI — NOT a conversion candidate. **CLI parity verified**: all 7 documented commands (`read`, `add`, `view`, `search-files`, `search-text`, `health`, `reset`) exist in the live CLI top-level help line. Undocumented-but-available commands (`search-deep`, `scenarios`, `tree`, `heal`, `heal-status`, `stats`) are discoverable via `--help` and don't represent drift — the skill is intentionally curated, not exhaustive. Quality Guidelines §4 are concise and judgment-based (irreducible). No drift, no decision warranted. Note: scenario was DOWN at heartbeat (auto-start API base resolution failure); CLI parity verified via top-level help banner only. Revisit if health drops or new commands become canonical. |
| scientific-debugging | 17 | 0.74 | high | 2026-04-27 | no-action | ~3,600 tokens / 396 lines / 14,372 chars. Methodology skill (hypothesis-driven debugging) — judgment-heavy, mostly irreducible (Phase 0–6 process, hypothesis/RCA templates, anti-patterns, convergence patterns). NOT a conversion candidate. **CLI parity verified**: Phase 0 cites `swarm-manager scenarios fixes --name … --all --search …` and `swarm-manager ai-search query "…" --kind fix [--target-scenario …]`; both confirmed live (smoke-tested `scenarios fixes` returned real LPBS fixes; `--target-scenario` is a real filter at `cli/cmd_aisearch.go:320` despite the CLI's own usage banner omitting it — that's a CLI-side gap, not a skill defect). Possible polish targets (Section 2 ASCII diagram ~80 tokens, structural repetition across Phase 1–6 entry/exit blocks ~200 tokens) trip failure-mode-1 (polishing) and failure-mode-4 (churn-without-benefit) at 17×consumer fanout — risk to working consumers > token savings. No decision warranted. Revisit on drift / if `swarm-manager` CLI surface shifts under Phase 0. |

## Revisit queue (priority order)
1. architecture-scope (popular per `graph popular`, never visited)
2. systematic-exploration (popular per `graph popular`, never visited)
3. scientific-debugging — revisit in ~14 heartbeats or on Phase-0 CLI drift
4. knowledge-observatory-tools — revisit in ~14 heartbeats or on drift / new canonical commands
5. visited-tracker-tools — revisit in ~14 heartbeats or on drift / once status→coverage fix resolves
6. skill-principles — revisit in ~14 heartbeats or on drift / once trim decision resolves
7. documentation-health — revisit in ~14 heartbeats or on drift
