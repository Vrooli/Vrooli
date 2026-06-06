# Problems

## Open Follow-Up

| Severity | Component | Issue | Evidence | Next Step |
| --- | --- | --- | --- | --- |
| 3 | Requirements | Completeness scoring reports 100% one-to-one operational target mapping. | `scenario-completeness-scoring score get fall-foliage-explorer --json` reports `ungrouped_operational_targets`. | Decompose requirements under cohesive PRD target folders without changing PRD checkboxes manually. |
| 3 | Tests | Unit phase can fail coverage threshold even when Go tests pass. | Historical `test/artifacts/unit-*.log` shows 40.2% coverage below 50%. | Add meaningful tests for uncovered handlers or adjust threshold only with evidence. |
| 2 | Resources | Redis is declared but not currently used directly by API code. | [CODE: .vrooli/service.json] and [CODE: api/main.go]. | Either implement caching in a future non-doc pass or mark the resource optional with product approval. |
| 2 | Standards | UI type-check currently falls back to syntax-only when `typescript` has not been installed locally. | `pnpm run type-check` reports the compiler is unavailable until `pnpm install`; lifecycle setup now uses `pnpm install --ignore-workspace`. | Run lifecycle setup or approved dependency install before relying on strict TS checks locally. |

## Tech Debt

| Severity | Component | Issue | Evidence | Next Step |
| --- | --- | --- | --- | --- |
| 2 | API structure | API handlers still live in a single large `api/main.go` module even after response writing and startup helpers were extracted. | `tidiness-manager recommend-refactors fall-foliage-explorer --limit 5 --sort-by priority --min-lines 50` continues to rank `api/main.go` as a high-priority refactor target. | In a future refactor pass, split cohesive handler groups into focused files while preserving current route contracts and tests. |
| 2 | Standards | Standards scan still reports 5 medium/error findings after this pass. | `coverage/runs/20260531-221642-33564304/phase-results/standards.json` reports custom UI server interop warnings, PRD Tech Direction Snapshot wording, and two type-safety scanner findings. CLI domain test gaps, env-validation warnings, hardcoded URL warnings, and focus-visible CSS findings were cleared in this pass. | Evaluate migration to `@vrooli/api-base/server`, update PRD Tech Direction Snapshot only with explicit PRD edit approval, and re-run standards after scanner-sensitive ESLint comments/type-pattern cleanup. |

## Resolved Or Historical

Historical issues resolved before this pass included the missing API `fmt` import, UI server syntax error, database connection failure, missing database tables, Ollama integration gap, export functionality gap, and photo gallery gap.
