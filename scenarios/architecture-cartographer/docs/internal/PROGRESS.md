# Progress — Architecture Cartographer

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-21 | Claude (Opus 4.7) | done | Scenario charter complete. Published PRD with 10 P0 / 5 P1 / 6 P2 operational targets via `prd-control-tower prd generate --publish`. Generated 21 requirement modules via `prd-control-tower requirements generate`; all linked to operational targets (`requirements validate` reports `healthy`). Filled concept docs (ARCHITECTURE, DOMAINS, DATA, FLOWS, INTEGRATIONS), authored new `SIGNAL_LADDER.md` and registered it in `docs/manifest.json`, filled internal docs (SEAMS additions for Detector/Resolver/Signal/Recipe/CodeGraphAdapter/BuildGuard/AnalyticsRecorder, TESTING patterns for signal/detector/aggregator/adapter tests, SECURITY threat model, PERFORMANCE budgets, DECISIONS for all durable choices, PROBLEMS entries for unimplemented dependencies). Implementation has not started; the scenario is ready for first vertical-slice work (likely the `graph` domain) once `go-code-graph` and `typescript-code-graph` exist. |
| 2026-05-21 | Claude (Opus 4.7) | done | Phase 0 — template `notes` residue removed end-to-end: deleted proto schemas + generated go/ts/python trees, `api/handlers/notes/`, `api/internal/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `ui/src/api/notes.*`, `ui/src/pages/NotesPage.tsx`. Cleared registry wiring (`api/main.go`, `api/internal/modules/registry.go`, `api/cmd/gen-endpoints/cli_commands_seed.json`), CLI wiring (`cli/domains/domains.go`, `cli/manifest.json`), UI wiring (routes, navItems, selectors, i18n locales en/ja/ar, regenerated `strings.generated.ts`). Pre-existing test fixtures referencing notes neutralised (`api/cmd/gen-endpoints/main_test.go`, `api/internal/module/module_test.go`) and the global proto-Connect parity check softened to gracefully accept an empty `AllProtoFiles()` until product domains land. Pre-existing UI defects unblocked Phase 0's "tests green" gate: opted into React Router v7 future flags and differentiated the duplicate `Primary navigation` landmark via a new `bottomNavLabel` value across all locales. Gates: `go build ./... && go test ./...` green for api/ and cli/; `pnpm test` 127/127 passing; `pnpm build` succeeds; `make endpoints` regenerates a clean manifest whose only non-Connect path is `/health` (tagged `RESTReasonOpsProbe`). |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
