# Progress — CLI Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-19 | Generator Agent | done | Phase 0 complete: scenario generated from react-vite + vrooli-default; PRD authored via prd-control-tower; 14 requirements generated covering 9/9 OTs; DOMAINS.md updated with validation/search/reindex domains; service.json declares ollama + qdrant resource dependencies; notes example domain removed; orientation finalized. Fixed 7 substrate bugs in templates/scenarios/react-vite (also patched in cli-health): (1) SCENARIO_ID→SCENARIO_ID_SNAKE in proto-symbol references, (2) cliapp.Register signature drift, (3) stale UI flow contractSha256, (4) stale API flow contractSha256, (5) missing RESTException on /health endpoint, (6) Makefile missing canonical .PHONY targets (clean/build/fmt/fmt-go/fmt-ui/lint/lint-go/lint-ui/check), (7) iframe guard + appId missing from ui/src/main.tsx, (bonus) manifest's omitted[] referenced nonexistent NotesService.AttachFile. Also relaxed TestProtoConnectParity to skip on empty AllProtoFiles() for greenfield transitional state. |
| 2026-06-16 | Codex | done | Adopted the shared health maturity assessment contract for CLI Health. Added `.vrooli/maturity.json`, emitted `common.v1.MaturityAssessment` from `ValidateScenario`, rendered local maturity in CLI human output, and preserved local maturity in Test Genie contracts phase summaries. |
| 2026-07-02 | Codex | done | Added the `entrypoint_structure` validator that supersedes scenario-auditor's lightweight-main heuristic inside cli-health. The implementation parses `cli/main.go` with Go AST, accepts the repo-standard `NewApp` + `app.Run` shape (including lifecycle guards/custom exit handling), and warns with `cli.main_heavy` only when `main()` directly owns infrastructure setup or non-delegating command/business work. Maturity maps unreadable entrypoints and heavy main bodies; `cli.main_heavy` is manual/no-autofix because safe extraction changes function boundaries, defers, imports, and exit semantics. Validation: focused entrypoint tests green, `go test -short ./...` in API green, CLI `go test ./...` green, public `cli-health validate scenario cli-health --json` passed with `entrypoint_structure` L3/clean. Full scenario run `20260702-235736-6bd86084` still fails on existing non-contract gates; contracts passed and emitted no `cli.main_*` findings. |
| 2026-08-07 | Codex | validated-with-limit | Added deterministic manifest semantic validation beside the existing resolution checks: field collisions, control flags bound as request data, required payload fields left empty, and redundant binds. Seeded violations and corrected fixtures cover all four rules; `bind_waiver` is schema-validated for genuine control-named request data. The fleet repair reduced the independent census from 124/78/2/9 to 0/0/0/3, with the three warnings explicitly waived. Full API tests and live audio-tools/development-toolchain-validator contract probes pass; broader server-owned suite validation remains limited by test-genie infrastructure and is recorded in the active plan. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
