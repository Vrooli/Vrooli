# Progress — CLI Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-19 | Generator Agent | done | Phase 0 complete: scenario generated from react-vite + vrooli-default; PRD authored via prd-control-tower; 14 requirements generated covering 9/9 OTs; DOMAINS.md updated with validation/search/reindex domains; service.json declares ollama + qdrant resource dependencies; notes example domain removed; orientation finalized. Fixed 7 substrate bugs in templates/scenarios/react-vite (also patched in cli-health): (1) SCENARIO_ID→SCENARIO_ID_SNAKE in proto-symbol references, (2) cliapp.Register signature drift, (3) stale UI flow contractSha256, (4) stale API flow contractSha256, (5) missing RESTException on /health endpoint, (6) Makefile missing canonical .PHONY targets (clean/build/fmt/fmt-go/fmt-ui/lint/lint-go/lint-ui/check), (7) iframe guard + appId missing from ui/src/main.tsx, (bonus) manifest's omitted[] referenced nonexistent NotesService.AttachFile. Also relaxed TestProtoConnectParity to skip on empty AllProtoFiles() for greenfield transitional state. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
