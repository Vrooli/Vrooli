# Known Issues & Technical Debt

## Work ladder

- Rung: W3 implementation — selected System Monitor redesign slices implemented; governed re-measurement remains partially red
- Evidence: focused UI validation passes (Header, Scripts, metric detail variants: 9 tests) and TypeScript type-check passes. BAS desktop/mobile captures confirm the responsive shell, including a stable mobile Menu trigger after the header was restructured into an explicit brand row and control row. The final server-owned run `20260824-143930-f36c6335` passed 20/22 phases; only `docs` and `unit` remain failed. The run also recorded a host-level degraded Ollama dependency (declared CUDA, observed CPU) while System Monitor itself started healthy in best-effort mode.
- Blocker: no missing authority. Remaining red phases are inherited contract/documentation and unit-health debt; broad catalog cleanup and Plan Manager baseline-comparator repair are intentionally deferred under the current user-visible-outcome reprioritization. Linux arm64 hardware evidence remains unavailable.
- Measured: 2026-08-24

## Last Updated
2026-08-21 (UI hardening pass)

## Resolved Since 2026-02-16

- **CLI rewritten in Go** — the old bash CLI (grep/cut JSON parsing, `/api/reports/generate` missing `/v1/` prefix, ignored `--quiet`, separate divergent `vrooli-system-monitor` entry point, broken `simulate` command) has been replaced by a typed Go CLI under `cli/domains/*`. The JSON-parsing fragility, the report-endpoint 404, the `--quiet`/entry-point divergence, and the `simulate`→`/api/test/anomaly/cpu` bug are all gone.
- **Script API implemented** (`api/internal/handlers/investigations.go:516-574`): `ListScripts`, `GetScript`, and `ExecuteScript` now delegate to a real `scriptSvc` that serves the on-disk scripts (no longer empty-array / 404 stubs).
- **Metrics timeline endpoint added** (`GET /api/v1/metrics/timeline`, `metrics.go:54`): the UI sparkline now fetches real timeline data instead of relying solely on client-side accumulation.
- **Four UI lifecycle bugs fixed** (see `COHERENCE-NOTES.md` Round 5): poll-race dedup, `setTimeout` cleanup, unmounted-state-update guards, and spawn dedup.
- **`useSystemMonitor` god hook split** into `useHealthCheck` + `useMetricHistory` + the composition root.
- **Dashboard bundle code-split / lazy-loaded** (recharts detail views, secondary pages, modals, and `react-syntax-highlighter`) to lift the Lighthouse performance score.

## Open Issues

- **Process kill endpoint not implemented**: the UI process monitor has a kill confirmation dialog, but `POST /api/v1/processes/{pid}/kill` is not registered in the router (`api/internal/server/router.go`). The kill action currently no-ops. Tracked as P0 operational target OT-P0-009 (process-monitoring) — see `requirements/09-process-monitoring-and-management/`. Severity: medium.
- **Explicit development memory mode**: SQLite is the runtime default. In-memory storage is available only when `SYSTEM_MONITOR_STORAGE_MODE=memory` is explicitly set in a non-production environment, and history is then intentionally lost on restart. Severity: low (development-only).
- **No authentication**: all API endpoints are publicly accessible; no auth middleware is enabled. Acceptable for the current local-monitoring posture but must be addressed before any networked deployment. Severity: medium (deployment-gated).
- **HTTP polling, no WebSocket**: the UI polls (5s metrics, 60s detailed, 4s agents) rather than streaming. Introduces latency vs real-time; acceptable for V1. Severity: low.

## Performance — residual gated by fleet infrastructure (not scenario code)

The `performance` test-genie phase gates on a Lighthouse dashboard score `>= 0.70`. The scenario currently lands **0.66–0.68**, up from a `0.41` baseline after real, scenario-owned optimization:
- bundle code-splitting + `React.lazy`/`Suspense` (recharts detail views, secondary pages, modals, and `react-syntax-highlighter` all deferred) shrank the main entry from **~1.63 MB to ~420 KB**;
- web fonts made non-render-blocking (removed the `@import`/`<link>` blocking requests);
- below-the-fold dashboard sections deferred to cut first-commit hydration / Total Blocking Time.

The remaining **~0.04** gap is **not** system-monitor's own code. The dashboard UI is served by the shared `@vrooli/api-base` static server (`packages/api-base/src/server/template.ts` `express.static`), which has **no `compression` middleware**, so the ~420 KB JS entry ships **uncompressed** (~3.4× the ~125 KB gzipped size). Lighthouse's "Enable text compression" penalty and the larger transfer depress the score fleet-wide for every react-vite scenario. Fixing that one shared middleware would lift system-monitor (and the whole fleet) past 0.70 with zero per-scenario change. Tracked as a platform bug: `bug-inbox/performance/api-base-static-server-missing-gzip-compression` (knw-1782155812312310344). This residual is therefore **fleet/platform infrastructure debt, unrelated to system-monitor's own code**, and out of scope for the scenario's health pass.

## Test Coverage

- Unit tests exist for collectors, services (monitor, alert, report), and handlers; the `unit` test-genie phase passes. Integration/CLI bats and UI vitest suites run green. Coverage gaps remain in some repository and settings paths (non-blocking).

## Cleanup History

- 2026-07-08: Implemented the proto-owned `MetricsService.GetDiskDetail` path with read-only partition, directory, and largest-file attribution plus storage-manager handoff notes. Removed the disk-detail missing-endpoint issue from the open ledger; cleanup execution remains outside system-monitor.
- 2026-06-22: Phase-5 health pass — reconciled this file with the current codebase (most 2026-02-16 items were already fixed); migrated `docs/manifest.json` to `scenario-docs-manifest/v2`; restructured `PRD.md` to the canonical template and linked all P0/P1 operational targets to requirements; added `bas/registry.json`; added `minimumReleaseAge` to `ui/pnpm-workspace.yaml`; cleared the six gating tidiness errors; fixed the four UI lifecycle bugs; split the god hook; code-split the UI bundle. The lone residual red (`performance` Lighthouse `0.66–0.68 < 0.70`) is documented above as fleet/platform infrastructure debt (api-base no-gzip, bug knw-1782155812312310344), not scenario code.
- 2026-02-16: Previous spec-sync sessions corrected script count (70+ → 30), removed non-existent timeline endpoint from API contract, documented placeholder endpoints, corrected polling interval descriptions.
