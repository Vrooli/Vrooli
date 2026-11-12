# Progress — Research Desk

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-11-11 | design owner (agent) | done | Generated from react-vite; `make setup`/`make start` healthy; structure+unit phases pass (run 20261111-213838-9d3ab725). PRD and requirements authored through the business-health wizard (11 P0, 4 P1, 3 P2 targets, prefix RD); requirements validate. Design written: DOMAINS, FLOWS, DATA, INTEGRATIONS, EXPERIENCE, ARCHITECTURE, GOALS, SKILLS, reference/FILE-CONTRACT, docs/DECISIONS. No product code yet: template `notes`, SQLite and placeholder UI still present by design; they are removed in the API phase. |
| 2026-11-11 | design owner (agent) | done | Design revised to review 1 rulings (`.vrooli/work/research-desk/REVIEW-1-RULINGS.md`): one design doc `concepts/SYSTEM.md` (three domains, 13 RPCs, 14 verbs; states, control G12, since-last-look G13, proofs), `reference/FILE-CONTRACT.md` (packet.py, LAUNCH.json, OPERATOR.md and needs-you grammars), `guides/skills-move.md` for the skills owner; DECISIONS rewritten; README and SEAMS cut to budget; ARCHITECTURE, DOMAINS, INTEGRATIONS, API and CLI references now pointers; FLOWS, DATA, GOALS, SKILLS removed; PRD gains OT-P0-012/013, requirements RD-P0-012/013; requirements validate PASSED; docs manifest validates against its schema with all required headings and links present. No product code. |
| 2026-11-12 | API owner (agent) | done | Review 3 carried out (`.vrooli/work/research-desk/REVIEW-3-RULINGS.md`): unknown never empty (E1), read sets from limits.json (E2), one error mapper (E4), one capability table and peer timeouts (E5), surface (E6), txtar fixtures and `make real-home-check` (E7), template leftovers removed (E8), package names, one version and courier shutdown (E9), docs truth (E10); F1–F15, with the F5 measurement on the last throwaway session (pause and resume act during a running turn and are now allowed then; `evidence/launch-proof.txt` session 3). Contract changes in `.vrooli/work/research-desk/CONTRACT-NOTES.md`. |
| 2026-11-12 | API owner (agent) | done | Review 4 API items P1–P8 (`.vrooli/work/research-desk/REVIEW-4-RULINGS.md`): operator assertion for unknown commissions (P1), blockers resolved in the request with supersede/run alongside and a preselected recommendation (P2), suggested stance quotes and workstream titles (P3), `GetLaunchMessages` read replacing manual `Launch` (P4), dry runs, delivery reason codes and per-commission entry reading (P5), global-item instruction and links (P6), 24-hour default window and `MarkSeen` `all` (P7), research write separate from mission and review with paused/ended notes (P8). Real home: both packets reach the launch confirmation (`internal/realhome`). Decisions D22–D24. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/SYSTEM.md`](../concepts/SYSTEM.md) — the design
