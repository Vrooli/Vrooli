# Progress — Source Ledger

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-07 | codex | done | Generated `source-ledger` from the React/Vite lifecycle template, completed the scenario-specific charter and requirements registry, then mechanically detemplated the notes example without moving any memory-engine code. |
| 2026-08-07 | codex | done | Established the contract-first Source Ledger vocabulary and boundaries: journal, forest, facets, recall, policy, federation, and health, with durable architecture, data, flow, integration, decision, and L0 experience documentation. |
| 2026-08-07 | codex | done | Phase 12 contract foundation: generated the Source Ledger scenario, drove a scenario-specific PRD and requirements registry through business-health, mapped journal/forest/facets/recall/policy/federation plus health, authored data/architecture/flow/integration/decision and L0 experience contracts, removed the notes example domain, and aligned the UI routes with the contract. Validation: `make setup`, lifecycle start/status, API and CLI Go tests, UI type-check/build/coverage (141 tests; 85.45% branch coverage), endpoint generation, business-health, experience-manager, dependency analysis, and comprehensive Test Genie run `20260807-230918-45c5fccb` (20/20 phases passed). No engine code moved. |
| 2026-08-07 | codex | done | Phase 13 service contract: authored source-ledger journal, recall, forest, facets, scopes, shared error, and health protos with scope on every request; regenerated Go/Connect/TypeScript/Python/descriptor/manifest outputs; documented the two-scenario migration boundary, typed degraded mode, startup discovery, and live pre-extraction latency ceilings. Validation: `make generate`, scoped Buf lint, source-ledger API `go test ./...` and `go build ./...`, and proto-health validation passed. |
| 2026-08-07 | codex | done | Phase 14 pure engine move: copied journal, forest, facets, recall, policy, vector, inference, federation, formal flow artifacts, purity test, test seams, and matching Connect handlers into source-ledger; added the rules contract; wired migrations, policy, inference, and all moved RPC modules into the source-ledger composition root; retained the vrooli-memory copies. Validation: source-ledger API tests/build/vet, byte-identical purity test, lifecycle restart/health, real Facets RPC, scoped Buf lint, proto-health passed, source-ledger Test Genie run `20260807-233434-b491199c` passed 20/20, and vrooli-memory Test Genie run `20260807-234144-f8294b16` passed 20/20 after fixing pre-existing native-hook manifest, dependency-replace, bounded-conversion, and rows-close defects. |
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
