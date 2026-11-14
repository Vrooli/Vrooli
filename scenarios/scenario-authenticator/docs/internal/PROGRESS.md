# Progress — Scenario Authenticator

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

The first entry below is historical orientation work from 2026-06-18. It
describes the repository at that date; it is not the current implementation
status. The current auth foundation and this plan's identity/delegation work
are recorded in the later entry.

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-06-18 | agi | done | **Documentation-first regeneration (orientation Gates 0–5b).** Regenerated scenario-authenticator from the `react-vite` template (`vrooli-default` design) after preserving the old scenario read-only at `/tmp/scenario-authenticator-OLD-reference`. Gate 0: scaffold builds green (API/CLI/UI). Gate 1: authored + validated `PRD.md` (IdP model, realm primitive, 12 P0 / 9 P1 / 7 P2 operational targets, carried-over crypto invariants; prd-control-tower validate = healthy). Gate 2: generated the requirements registry — 28 modules, one per target, 28/28 linked, validate = healthy. Gates 3–5b: filled the full docs folder (concepts, internal, operations, business, reference) + DOMAINS domain map + DECISIONS + this handoff; set `dependencies.resources.redis` (required) and fixed the category in `.vrooli/service.json`. **STOPPED for review before Gate 6 (implementation).** Nothing beyond the template scaffold (`health` + fenced `notes` example) is implemented yet. |
| 2026-08-09 | codex | done | Implemented and validated the local IdP foundation and the identity/delegation plan seams: refresh/password change, scope catalog and authorization, trust posture, machine binding, agent delegation, scope-derived Bridge dispatch, artifacts, and truthful documentation. Remaining plan evidence is the live Mac onboarding/screenshot and final credential-store teardown. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
