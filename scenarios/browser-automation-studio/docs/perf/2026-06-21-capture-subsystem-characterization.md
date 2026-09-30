---
date: 2026-06-21
scenario: browser-automation-studio
purpose: P0 characterization + red-triage anchor for the performance-health buildout
plan: performance-health-scenario-bas-perf-capture-test-genie-perf-phase-migration
---

# BAS capture-subsystem characterization & red triage (P0)

This note anchors the `performance-health` buildout. It records the current
state of BAS's capture subsystem (the surface we refactor in P1 and extend in
P2) and triages the pre-existing failing phases so that, later, our regressions
are distinguishable from inherited red.

## Regression anchor

- **Baseline:** `perf-health-buildout` (run `20260621-035533-73550b93`), pinned
  server-side this session. ⚠ Captured against a **dirty tree** (large
  uncommitted agi work) — BAS diffs are muddled; treat any `diff` exit 1/2 as
  "investigate," cross-checking against the known reds below before blaming our
  changes.
  - Diff: `git-control-tower baseline diff --scenario browser-automation-studio --name perf-health-buildout`
- **packages allowlist anchor sha:** `a655f116de6c3d6dba733088828a42db9fa73d43`
  (`git diff --stat <sha> -- packages/proto packages/maturity-go` must show only
  the declared paths from §6a).

## Capture subsystem inventory (file:line confirmed 2026-06-21)

- **CLI:** `cli/capture/command.go` — one `capture` verb; flag loop ~187-272;
  capture-type parse switch ~335-348 (`screenshot|console-logs|network|video|dom|performance`);
  a second switch ~432-448.
- **Handler:** `api/handlers/capture/service.go` — `Capture()` RPC ~47-134;
  `harvestArtifacts()` ~396, `harvestOne()` ~408 switch; `unavailableArtifact()`
  ~465 (currently returned for video/dom-file/performance); capture-type
  label/ext duplicated across `captureTypeShortName` ~476 + `captureTypeExt` ~495
  + line ~376.
- **Export:** `api/services/workflow/export_folder.go` `ExportToFolder()` —
  hardcodes console/network markdown; no producer registry; no isolated test.
- **Markdown:** `api/services/export/markdown.go` —
  `GenerateConsoleLogsMarkdown`, `GenerateNetworkActivityMarkdown` (untested in
  isolation).
- **Driver:** `playwright-driver/src/` — rebrowser-playwright v2 HTTP server;
  `src/telemetry/collector.ts` console+network via `page.on(...)`; **no CDP
  Tracing today** (perf tracing is new infra in P2); `src/session/manager.ts`
  owns browser lifecycle.
- **Proto:** `packages/proto/schemas/browser-automation-studio/v1/capture/capture.proto`
  — `CAPTURE_TYPE_PERFORMANCE` enum already present, currently stubbed
  `unavailable`.
- **Existing good seams:** `api/handlers/capture/module.go` `Executor` interface
  (mockable; `fakeExecutor` in tests); `api/automation/driver/interface.go`
  Driver/Session abstraction.
- **Missing seams (P1 adds):** pluggable artifact producers; telemetry
  instrumentation hook.

## Red triage (latest run 20260621-035533-73550b93: 3 passed / 15 failed / 18 phases)

Passed: `ui-health`, `architecture`, `integration`.

| Phase | Verdict | Classification |
|-------|---------|----------------|
| `playbooks` | `bas/seeds/seed.go` exit 1 | out-of-scope debt → existing BAS GCT backlog |
| `business` | dozens of `validation.ref` to non-existent test files | out-of-scope debt → existing BAS GCT backlog |
| `tidiness` | ~122 findings | out-of-scope debt; fix only touched-file findings inline |
| `security` | ~12 findings (template security-headers etc.) | known template debt; out-of-scope |
| `measures` | 2 findings | out-of-scope debt |
| `proto` | ~59 findings | out-of-scope debt; fix only files we touch |
| `standards` | template/scaffolding violations | out-of-scope template debt |
| `structure`, `contracts`, `quality`, `docs`, `unit`, `smoke` | inherited red on dirty tree | out-of-scope; cross-check on diff |
| `performance` | native perf phase (axes ①/③) | DELEGATED+DELETED in P9 — expected to change |

**None of the 14/15 reds block the capture-subsystem refactor (P1) or the perf
capture feature (P2).** Out-of-scope BAS rehab is filed against the existing
backlog items `qa-browser-automation-studio-gct-requirement-target-pass-rate-20260517`
and `qa-browser-automation-studio-gct-test-decomposition-20260517` — NOT absorbed
into this plan. Inline fixes are limited to (a) files we touch and (b) cheap
adjacent wins.

## P1 entry criteria (met)

- Baseline pinned; sha recorded; inventory confirmed; reds triaged.
- The "perf is just another capture type" model is already the intended shape
  (`CAPTURE_TYPE_PERFORMANCE` declared) — P1 builds the producer registry it
  plugs into, P2 builds the producer + driver tracing.
