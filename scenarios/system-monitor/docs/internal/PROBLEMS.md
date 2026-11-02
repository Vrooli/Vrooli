# Known Issues & Technical Debt

## Work ladder

- Rung: W3 implementation — focused node-axis changes pass, but the governed scenario run remains partially red
- Evidence: W0 comparison found the active `system-monitor-production-readiness` goal aligned with the P0 operational targets; `business-health validate scenario system-monitor` and `vrooli scenario requirements validate system-monitor` passed with four pre-existing P2 orphan warnings. The server-owned run `20260826-232942-a034eeef` finished 18/22 phases; dependency, unit, workflow, and experience phases failed, with no nodeclient or qualified-verb contract failure. Focused UI/type-check and system-monitor API tests pass, including the remote metrics qualified-verb regression.
- Blocker: no missing authority. Remaining red phases are inherited dependency, unit, workflow, and experience debt; Linux arm64 hardware evidence remains unavailable. Ollama remains CPU-backed until the NVIDIA driver/device-node issue is repaired.
- Measured: 2026-08-26

## Last Updated
2026-08-21 (UI hardening pass)

## Resolved Since 2026-02-16

- **CLI rewritten in Go** — the old bash CLI (grep/cut JSON parsing, `/api/reports/generate` missing `/v1/` prefix, ignored `--quiet`, separate divergent `vrooli-system-monitor` entry point, broken `simulate` command) has been replaced by a typed Go CLI under `cli/domains/*`. The JSON-parsing fragility, the report-endpoint 404, the `--quiet`/entry-point divergence, and the `simulate`→`/api/test/anomaly/cpu` bug are all gone.
- **Script API implemented** (`api/internal/handlers/investigations.go:516-574`): `ListScripts`, `GetScript`, and `ExecuteScript` now delegate to a real `scriptSvc` that serves the on-disk scripts (no longer empty-array / 404 stubs).
- **Metrics timeline endpoint added** (`GET /api/v1/metrics/timeline`, `metrics.go:54`): the UI sparkline now fetches real timeline data instead of relying solely on client-side accumulation.
- **Four UI lifecycle bugs fixed**: poll-race dedup, `setTimeout` cleanup,
  unmounted-state-update guards, and spawn dedup. The focused hook boundaries
  are recorded in [`SEAMS.md`](SEAMS.md#ui-lifecycle-and-shared-utility-seams).
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

## Host-memory investigation — 2026-10-25

- Rung: W3 localized pressure presentation defect. The existing contract requires the backend's platform-neutral pressure verdict and truthful unavailable states; no contract change is needed.
- Evidence: the memory detail view previously declared thrashing whenever combined swap traffic exceeded 128 pages/s, regardless of swap direction, stalls or the server pressure level. Three focused regressions reproduced false thrashing, hidden critical PSI pressure, and a lost paging-failure reason before the correction.
- Repair: present the existing server pressure verdict/reason/source/time, swap-in/out rates and PSI stalls; preserve unsupported/failure reasons and keep occupancy distinct.
- Follow-up repair: shared host inventory now retains optional Linux Slab/SReclaimable/SUnreclaim bytes. Monitor exposes separate stateful kernel accounting in current/detail/timeline responses and the memory detail chart, retaining source observation time and historical/platform absence. No pressure or leak threshold was invented. Privileged kernel-cache attribution and non-slab kernel allocation remain unmeasured; a large SUnreclaim observation alone is insufficient to establish a leak.
- 2026-10-30 W3 follow-up: live `memory-pressure` investigations previously omitted the existing scheduler-owned kernel accounting. They now include it with field states and original observation times, and explicitly report a missing or failed source without inventing an allocation or leak verdict. The metrics timeline also ignored its requested sample interval and silently returned the newest 1,000 cycles; it now selects the newest cycle per interval using bounded SQLite index seeks, matches that behavior in memory mode, and reports the effective window when the bucket cap shortens it. Empty buckets remain unknown coverage.
- Evidence: Test Genie unit `20261030-184334-58587686` passed with 53 warnings; focused Go race tests covered sampled SQLite and memory timelines, timestamp ties, batching, old history, and unavailable kernel observations. After a normal staged lifecycle restart, live investigation `558e47ed-2537-4241-9610-47baca97d6fc` included measured kernel accounting. A 24-hour, ten-minute timeline returned 144 samples from 2026-10-29 18:54:52 to 2026-10-30 18:44:49 UTC; the same request at five-second intervals reported its bounded 5,000-second effective window. Historical endpoints do not prove behavior between samples or identify kernel allocations.
- 2026-10-30 process-I/O follow-up: the repeated managed-artifact scans were originally identified using `/proc/<pid>/io` outside System Monitor. The on-demand `process-read-activity` investigation now compares bounded process-identity-fenced snapshots, distinguishes bytes returned by reads from storage-layer bytes, ranks each separately, and reports permission denials, churn and truncation. It does not run in the steady collection cycle; other platforms explicitly return unsupported. Linux counters include waited-for children, so a reaped child's lifetime I/O can appear as an apparent short spike and process rows can overlap. Observed child exits are flagged where available but do not prove reaping; short-lived processes may be missed. Live run `b5c35c2e-bf15-41f9-8c60-409efe8cb284` completed in 2.15 seconds with 1,052 matched processes and 1,050 permission-denied observations across the two passes; coverage was correctly partial. Test Genie unit `20261030-192023-df61dd48` passed with 53 warnings and stable source. No process or scan is inferred from the aggregate slab gauge.
- Validation: all seven focused MemoryDetailView tests pass; UI type-check and static Unit Health pass; server-owned Test Genie unit run `20261025-235358-9ebb6dc2` passed (29 seconds). Follow-up slab accounting passed focused parser/API/projection/persistence/UI checks, governed proto generation and verification, consumer-materialized UI type-check, and final Test Genie `20261026-001810-2a3f9c76` (unit+dependencies, 2/2 passed, 27 seconds). A stale installed proto copy and module metadata were refreshed through their owners; an existing asynchronous UI test now waits for its accepted enabled state. Receipts, including the failed intermediate run, are retained in the host-memory-review evidence folder. This entry does not claim full W0–W2 or scenario certification.


## Expanded log message containment — 2026-10-28

- Rung: W3 localized implementation defect; existing disclosure contract/test is unchanged.
- Evidence: BAS execution `197f08b3-60b2-44a8-8315-e0150c751f38` is terminal `COMPLETED`; its isolated context closed. Exact-hash matched 445-character content measured 209px message height inside a 150px expanded virtual row (row scrollHeight 247px); no row-local scroll/copy path was present. Two earlier BAS target-selection attempts (`d64d8985-757b-48cc-af40-01e1958d80e2`, `52ca486c-e1b6-4ba6-acbb-cb31275e3336`) failed before clicking. Full retained receipt: `~/.vrooli/research/topics/validation-quality/runs/prepared-20261028/evidence/system-monitor-logs-expanded-exposure-followup.json`.
- Cause: the virtual list assigns fixed collapsed/expanded heights; expanded message CSS allowed natural content to exceed the item box, while `overflow:hidden` hid the overflow. A 6em-bounded, keyboard-focusable message region with vertical scrolling now exposes the full text without changing item-size/virtualization policy. The short 75-character control remains without a toggle.
- Validation: regression failed before the fix because the expanded message had no named region; focused `LogTable.test.tsx` passes 3/3 after the change; UI `tsc -b` passes; Test Genie run `20261028-110202-24bf88ae` passes its `unit` phase (22.4s, zero failed/skipped phases, with 57 Unit Health warnings). Durable owner status is retained at `~/.vrooli/research/topics/validation-quality/runs/prepared-20261028/evidence/system-monitor-unit-testgenie-20261028-110202.json`. No isolated owner capture can render this changed scenario source; the live service remains on its existing prebuilt artifact. No rebuild/deploy or BAS recapture was done; preserve prior live evidence and treat this working-tree patch as not yet live-verified. Work records: `814026f0-9d8f-4d8b-b7b0-a2763fd630ea`, `fde61ebc-ce0d-4ff5-b2f3-bb565964948f`.

- Restart safety review: `vrooli scenario deploy-report system-monitor --json` identifies 347 API input deltas (49 scenario, 298 repo-shared) and three authorized UI log-row changes with shared package identities. Normal restart builds both API and UI; UI-only setup still provisions dependencies declared by the full manifest. Api-base and iframe-bridge have 69 running consumers each, proto-types 62. No restart was attempted; the API/shared package scope is broader than this UI fix. Full report and consumer inventory: `~/.vrooli/research/topics/validation-quality/runs/prepared-20261028/evidence/system-monitor-deploy-report-before-restart.json` and `system-monitor-restart-safety-assessment.json`.
