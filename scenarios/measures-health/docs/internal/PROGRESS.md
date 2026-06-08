# Progress — Measures Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-08 | measures plan Ph4 | done | Scaffold gates 0–3: generated from react-vite + vrooli-default; PRD published+validated (healthy); requirements generated+validated (8 reqs / 6 targets, healthy); DOMAINS/INTEGRATIONS/service.json deps (qdrant, ollama, search-hub) authored. |
| 2026-06-08 | measures plan Ph4 | done | **`validation` domain (Deliverables 1+2)**: proto `ValidationService.{ValidateScenario,ListFleetCoverage}`; pure `Classify` (expected/covered/waived + tier + findings) + filesystem seams (manifest/proto-domain/descriptor/prober/lister); behavioral probe (gate-safe, never probes write/destructive); Connect handler; CLI `validate scenario`/`validate coverage`; 24 unit tests green; gofumpt + golangci-lint clean. |
| 2026-06-08 | measures plan Ph4 | done | **Producer wiring (Deliverable 4)**: `FINDING_SOURCE_MEASURES=10` (architecture proto, regenerated); test-genie `measures` phase (phase_measures.go + catalog + `Measures` const + phase doc + tests); EM `measures` dimension (dimensions.json source/phase maps + anti-drift fixture) + soft R4 ladder rung; DIMENSIONS.md synced. test-genie + EM tests green (also fixed a pre-existing `stubBAS` compile break in test-genie phases tests). |
| 2026-06-08 | measures plan Ph4 | done | **`index` domain (Deliverable 3)**: proto `SearchService.{Search,Status}` (snake_case `MeasureHit` carrier via `json_name` so the wire matches search-hub's adapter); `internal/measureindex` (harvest all manifest measure blocks → `LexicalMatcher` → `measures-go` Engine with HTTPExecutor + ollama LLMExtractor + θ); `handlers/search`; CLI `search query`/`search status`; `.vrooli/search.json` self-registered via `searchregister.Register` at boot. 33 tests (matcher/harvest/provider-gate/handler-**wire-shape**/search.json-contract) + gofumpt + golangci-lint clean; endpoints.json regenerated. Deliberate deviation: lexical matcher (corpus empty until Phase 5/6) behind the `measures.Matcher` seam — aisearch hybrid index is the documented drop-in follow-up (DECISIONS/PROBLEMS). |
| 2026-06-08 | measures plan Ph4 | done | **Fleet-view UI (Deliverable 5)**: `ui/src/features/fleet/` — `FleetTable` (cross-scenario coverage grid over `ListFleetCoverage`: verdict badge, expected/covered/waived/uncovered counts, worst-tier chip, row-select), `ScenarioCoverageCard` (per-scenario drill-down over `ValidateScenario`: domain rows ordered UNCOVERED-first, status/tier chips, waiver reason, nested measures), `FleetView` composer, `FleetPage` at `/fleet` (+ nav/route/selectors/i18n in en+ja+ar). Enum-driven presentation (`coverage.ts`); polished loading/error/empty + a11y states. **Built BESIDE notes** (scaffold-prescribed intermediate state). 16 new fleet tests; full UI suite **175/175 green**, tsc clean, new code adds **0 lint violations**. Also fixed pre-existing template debt unmasked along the way: RR v6 future-flag-vs-strict-console (router opt-in), duplicate nav-landmark a11y (distinct `bottomNavLabel`), AppShell hardcoded `aria-label`→i18n. |
| 2026-06-08 | measures plan Ph4 | in-progress | **Remaining (Phase 4 close-out)**: Gate 7 (remove notes example — api/cli/proto/ui/seed/i18n + `make endpoints`/`pnpm strings:gen`), then close-out baselines (`baseline diff` measures-health/test-genie/ecosystem-manager) + restart/health smoke. See PROBLEMS.md handoff. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
