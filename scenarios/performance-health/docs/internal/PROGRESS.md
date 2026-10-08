# Progress — Performance Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-06-21 | Plan P3 agent | done | Documentation-first initialization. Generated from react-vite template; authored PRD.md (three-axis/two-tier model, 12 OTs), 8 requirement modules (26 requirements, all linked, no phantom validation.refs — refs point to future test files), ecosystem-fit recorded in `internal/ECOSYSTEM_FIT.md`, updated DOMAINS.md (11 planned domains) + INTEGRATIONS.md (planned deps). PRD + requirements validation green; docs audit at parity with sibling health scenarios; scaffold starts healthy. NO feature/domain/CLI code yet (P4+). Example `notes` domain intentionally retained as the copy-from reference; detemplate deferred to after the first real domain (P4+). |
| 2026-06-21 | Plan P4–P12 | done | Full backend + CLI + UI buildout (supersedes the "no feature code yet" note above). All 10 domains implemented and unit-tested: readiness (Code-Facts-gated tier detection + filesystem fallback), autofix (shared `packages/maturity-go/autofix`, format-preserving, idempotent), analysis (deterministic ⚛ component table + file:line findings against checked-in fixture traces), capture (BAS perf-capture client + profile-mode orchestration + graceful headless skip), lighthouse (own-Chrome CLI runner, silent-skip), benchmark (go/ui build timing), budgets (declarative config + ratchet + suite-run gating), trend (SQLite store, additive), fleet (structured offender queries), startup (migrated from structure-health's `CLIRunner.Measure`, axis ②). `handlers/validation` dual-mounts the native ReadinessService and the shared `scenario-validation/v1 ScenarioValidationService` so test-genie's Performance phase delegates axes ①/③ here. `notes` example domain detempated. UI: all six core workflows wired to Connect RPCs over real data, a11y + i18n (incl. RTL), vitest ≥85% coverage. Migrations: test-genie native perf phase deleted + delegated (parity-gated); structure-health perf domain removed. |
| 2026-06-21 | Cleanup pass | done | Post-buildout cleanup: removed orphaned lighthouse artifact-path helpers left in test-genie after the native perf phase was deleted; removed stale "SCAFFOLD/lands in PX" doc comments across `internal/{fleet,trend,startup,assessment,readiness}` that described a half-built state; de-duplicated the fix-endpoint descriptors in `handlers/validation/module.go` (cleared both ERROR `DUPLICATED_CODE` tidiness findings — the 3 remaining tidiness errors are react-vite template harness debt, byte-identical to the template). Flipped requirement declared-status `planned`→`complete` (validations `planned`→`implemented`) now that every validation.ref exists and passes. |
| 2026-06-21 | Close-the-loop | done | Closed the test-genie Performance-phase validation loop (was readiness-only with a starved budget gate; could only PASS/SKIP). Honored `include_execution` end-to-end: new `ExecutionOrchestrator` (`handlers/validation/execution.go`) runs the build benchmark (Go+UI) + cheap bundle-size stat + Lighthouse-if-UI, persists ONE combined `perf_samples` row, then folds native build-floor breaches / broken builds / Lighthouse error-threshold breaches into ERROR findings; `SharedHandler.ValidateScenario` routes the flag through `Handler.validate`. test-genie's Performance delegate now sends `IncludeExecution=true` (guard test pins it). Wired every budget axis to a producer: bundle measured during UI build; startup cross-writes `perf_samples.startup_ms` (standalone-fed — NOT run in-phase, see DECISIONS); analysis cross-writes `lcp_ms` + slowest-component avg/max. Added `slowest_component_max_ms` (additive migration), fixed the `SampleToMeasurement` bug (read max from the avg column). Removed the dead `p95` axis end-to-end (proto field reserved + regen, Go core, schema, CLI, UI, maturity.json). Added `X-Content-Type-Options: nosniff` to the REST JSON writers. Consolidated the perf skill surface: deleted `scenario-performance-audit`, rewrote the `performance` steer skill to drive performance-health (+`programmaticHome: performance-health:performance`), repointed all live references (template + ~28 manifests + ~52 ui/perf copies), reactivated the dormant steer-vocabulary ratchet (stale SSOT path → `packages/maturity-go/dimensions/dimensions.json`). |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
