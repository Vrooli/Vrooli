# Implementation Plan — `execute/emulator-acceptance-tests-phase-1`

## 1. Purpose

Deliver Phase 1 integration tests and a CLI smoke test for `scenarios/vrooli-emulator/`. This plan preserves the full execution context: what to test, how to exercise the real `LinuxBackend` (Xvfb + x11vnc + websockify) without mocks, how to wire into `vrooli scenario test`, and how to absorb the integration-smoke harness and distro-validation matrix rehomed from the retired `execute/vrooli-emulator-linux-first` umbrella.

## 2. Required Reading

Execute before starting implementation:

```bash
# Canonical skills for plan-driven API/CLI work
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement

# Testing + portability skills discovered for this item
prompt-manager skill read test cross-platform-readiness

# Swarm-manager data-model reference (for any ancillary backlog updates)
prompt-manager skill read swarm-manager-backlog-tools implementation-plan-authoring
```

Repo-specific context:
- `swarm-manager backlog get --kind research --name emulator-extraction-and-service-plan` + `file-get --path conclusion.md` — especially Findings 10, 11, 13 (headless DISPLAY contract, full API surface, minimal CLI scope)
- `swarm-manager backlog get --kind execute --name vrooli-emulator-linux-first` + `file-get --path plan.md` — §6 and §8 (absorbed Phase 1 smoke + distro matrix)
- `scenarios/vrooli-emulator/api/livedesktop/` — service, handler, LinuxBackend, existing mock-based unit tests (`handler_test.go`, `service_test.go`, `action_test.go`, `janitor_test.go`, `store_test.go`)
- `scenarios/vrooli-emulator/api/main.go` — how routes, janitor, and data dir are wired for the running binary (read-only — listed in `acceptance_deny`)
- `scenarios/vrooli-emulator/cli/domains/sessions/sessions.go` — the exact CLI surface (`list`, `create`, `destroy`, `exec`, `logs`) this item must smoke-test
- `scenarios/vrooli-emulator/cli/app.go` — note `ExtraAPIEnvVars: ["API_BASE_URL", "VITE_API_BASE_URL"]`; this is the seam by which the CLI smoke test points the binary at the test server
- `scenarios/vrooli-emulator/.vrooli/testing.json` — current scenario-test configuration (unit-only today); §11 defines the integration wiring target

## 3. Greenfield Declaration

This is a **greenfield test authoring** item. There is zero test coverage to migrate — all existing tests in `livedesktop/` are mock-based unit tests; the real-backend path has never been exercised by automated tests. No backwards-compatibility shims, no dual-run of old and new suites, and no deprecated test file preservation. New test code is written fresh against the current `/api/v1/sessions/` contract.

## 4. Problem Statement

The `vrooli-emulator` scenario was scaffolded (`execute/scaffold-vrooli-emulator-scenario` completed 2026-04-18) but has **no automated verification that the real service actually stands up sessions, launches apps, and tears down cleanly**. Every existing test uses `mockPlatformBackend`; no test ever calls the real `LinuxBackend` that runs `Xvfb`, `x11vnc`, and `websockify`. Phase 2 (UI iframe embed) and Phase 3 (smoketest delegation) both depend on this contract holding — without integration tests, regressions in session lifecycle, process cleanup, or VNC/WebSocket wiring will only surface in downstream consumers.

The research decomposition committed to two specific deliverables (conclusion §Actions 2 and research Findings 10–13):

1. **HTTP-level integration tests** covering: create VNC session, create headless session with returned display, launch app from explicit `app_path`, take screenshot, tail metrics, destroy session.
2. **CLI smoke test** exercising `session list`, `session create`, `session destroy`, `session exec`, `session logs`.

Plus two absorbed deliverables from the retired `execute/vrooli-emulator-linux-first` (round 2 `d1=A`):

3. **Phase 1 integration smoke harness** — a scripted end-to-end exercise that stands up the emulator, creates both a VNC and headless session, takes a capture, tails metrics, and tears everything down.
4. **Per-distro validation matrix** — at minimum Ubuntu LTS 22.04 and 24.04.

Deliverables 1 and 3 overlap substantially; the plan structures Phase 1 such that one suite satisfies both (see §8).

The original contract gap from round 1 — that `SessionView` did not expose `display_id` for the Phase 3 smoketest delegation consumer — is **resolved as `d3=A`**: `display_id` is added to `SessionView` in Phase A of this item and surfaced through `Session.View()`. Round 2 `r2-d5=A` extends that decision into the CLI: the local `sessionView` mirror is updated to carry the field and `summarize()` surfaces it for headless sessions.

## 5. Scope

### In scope
- New integration test suite at `scenarios/vrooli-emulator/api/livedesktop/integration_test.go` (same package, build tag `//go:build integration`), exercising the real `LinuxBackend` against an in-process `httptest.Server` (round 1 `d1=A`, `d2=A`).
- CLI smoke test that builds the `vrooli-emulator` binary into `t.TempDir()`, execs it with `API_BASE_URL=<httptest URL>`, and asserts on exit code + key stdout substrings for all five `session` subcommands (round 1 `d5=A`).
- `TestPhase1SmokeHarness` (or equivalent) that chains the six acceptance assertions into a single end-to-end exercise — the absorbed integration-smoke deliverable.
- Add `display_id` field to `SessionView` and populate it in `Session.View()` from `s.Display.DisplayID()` (round 1 `d3=A`); update existing unit-test assertions accordingly. Mirror that field into the CLI's local `sessionView` and surface it via `summarize()` for headless sessions only (round 2 `r2-d5=A`).
- Wire `svc.WithCaptures(...)` in the integration harness with a `t.TempDir()`-backed `FilesystemStore`; assert that the screenshot test writes a file on disk and records metadata in the captures store (round 2 `r2-d1=A`). Production `api/main.go` remains unchanged here — that gap is tracked in info `r2-i1`.
- `.vrooli/testing.json` updates so `vrooli scenario test vrooli-emulator` runs `go test -tags=integration ./...` for the API module (round 1 `d1=A`).
- Documented distro matrix in `scenarios/vrooli-emulator/docs/runbook.md` declaring Ubuntu 22.04 and 24.04 as supported, with a one-time manual verification recorded as part of this item (round 1 `d6=A`). No CI matrix job, no containerized fixture in scope.
- Pre-flight host-dependency check that skips integration tests with a clear message when `Xvfb`/`x11vnc`/`websockify`/`xdotool` are missing.
- xclock-based fixture for `launch_app` assertions (round 1 `d4=A`); test skips with a clear message when `/usr/bin/xclock` is missing.
- Metrics assertion: `require.Eventually` (5s timeout, 200ms tick) until `MetricsView` is non-nil; then assert schema fields `CPUPercent`, `MemoryRSS`, `WindowDetected` are present on the returned view (round 2 `r2-d4=A`).
- Post-teardown stray-process detection: `pgrep -af` then string-match the test scenario name (e.g. prefix `test-phase1-`) in each process's argv; assert no matches remain after teardown (round 2 `r2-d3=A`). Implementable via `os/exec` calling `pgrep -af` and parsing.

### Out of scope
- WebSocket endpoint coverage — deferred entirely to Phase 2 (`execute/scenario-to-desktop-emulator-iframe-embed`), which has stronger motivation (rendering frames) and natural test fixtures (a real headless browser) (round 2 `r2-d2=A`).
- UI tests for `scenarios/vrooli-emulator/ui/` — deferred to `execute/vrooli-emulator-standalone-ui` which owns both UI code and its own coverage.
- iframe-embed contract tests — deferred to `execute/vrooli-emulator-external-url-endpoint` and `execute/scenario-to-desktop-emulator-iframe-embed`.
- Smoketest delegation end-to-end tests — deferred to `execute/smoketest-delegate-display-to-emulator`; once that item lands its suite can consume `display_id` added here.
- Cross-platform backend testing (Windows/QEMU, Android, remote-macOS) — research Finding 3 future work; out of Phase 1.
- Wiring `WithCaptures()` in `api/main.go` (the production binary) — that is a pre-existing gap. This item only wires captures inside the *test harness* (`r2-d1=A`); the production wiring belongs to a follow-up tracked in info `r2-i1`.
- Any changes outside `scenarios/vrooli-emulator/**` — `acceptance_allow` is scoped to that tree.
- Modifications to `api/main.go`, `cli/install.sh`, `cli/install.ps1`, or the scenario `Makefile` — explicitly listed in `acceptance_deny` to preserve current wiring.
- Mutating tests on scenario-to-desktop — this item does not touch that scenario.

## 6. Current Technical Context

**Scenario layout** (`scenarios/vrooli-emulator/`):
- `api/main.go` — wires `livedesktop.LinuxBackend`, `InMemoryStore`, `Service`, optional screen recorder, idle janitor (30s/30m), health handler, and `Handler.RegisterRoutes` onto a `mux.Router`. **Does NOT call `WithCaptures()`** — captures persistence is not wired in production today.
- `api/livedesktop/handler.go` — route surface includes 11 HTTP endpoints + 1 WS upgrade at `/api/v1/sessions/{id}/ws` (`proxy.go` handles the WS upgrade and proxies to `ws://localhost:{WSPort}`).
- `api/livedesktop/service.go` — `StartSession` auto-creates a `Monitor` via `s.backend.NewMonitorFactory()` and calls `session.SetMonitor(monitor)`; `StopSession` clears it. So `MetricsView` is populated for any session that successfully started.
- `api/livedesktop/platform_linux.go` — `LinuxBackend` concrete implementation; requires host `Xvfb`, `x11vnc`, `websockify`, `xdotool`, `xclip`. Spawned helper processes carry the scenario name in argv (display name and websockify args), which is what the `r2-d3=A` `pgrep -af` filter relies on for stray-process detection.
- `api/livedesktop/types.go` — `SessionView` exposes `scenario_name`, `state`, `vnc_port`, `ws_port`, `width`, `height`, `platform`, `headless`, `app_running`, `created_at`, `last_heartbeat`, `error`, and `metrics`. **Does not yet include `display_id`** — `d3=A` adds it in Phase A.
- `api/livedesktop/proxy.go` — `gorilla/websocket.Upgrader` upgrades the browser connection then dials the local websockify on `ws://localhost:{WSPort}`. For headless sessions (WSPort=0), the dial fails — Phase 1 does not exercise this path (`r2-d2=A`).
- `api/livedesktop/*_test.go` — five unit-test files, all using `mockPlatformBackend` via `newMockBackend()` in `service_test.go`.
- `cli/domains/sessions/sessions.go` — five subcommands matching spec (`list`, `create`, `destroy`, `exec`, `logs`); `apiBase = "/sessions"` joined with the core client's base URL; CLI defines its own local `sessionView` struct that does not currently include `DisplayID`. Phase A (`r2-d5=A`) extends this struct and updates `summarize()` to print `Display: <display_id>` for headless sessions only.
- `cli/app.go` — `ExtraAPIEnvVars: ["API_BASE_URL", "VITE_API_BASE_URL"]`; this is the env-var seam for pointing the test binary at the in-process `httptest.Server`.
- `cli/main.go` — entrypoint; trivial.
- `.vrooli/testing.json` — `unit.languages.go` enabled; no `integration` section today; `business.checks` enables endpoints, cli_commands, websockets — but those are business-validation checks, not functional tests.

**Host prerequisites verified on this Linux dev box** (needed for real-backend tests):
```
/usr/bin/Xvfb /usr/bin/xvfb-run /usr/bin/x11vnc /usr/bin/websockify /usr/bin/xdotool /usr/bin/pgrep
```
`xclip` presence is required for clipboard control actions but is not exercised by Phase 1 core assertions. `xclock` (`d4=A` choice) requires `apt install x11-apps`.

**Existing test patterns to reuse:**
- Mock-based helper `newTestHandler()` / `newTestRouter()` in `handler_test.go` — integration tests diverge here by using the real `NewLinuxBackend(logger)` instead of `newMockBackend()`.
- `screenrecording/display_test.go:63` — precedent for skipping when host binaries are missing; integration tests will follow the same pattern.

**Session contract (verified in code):**
- `POST /api/v1/sessions` with `{"headless": true, "scenario_name": "..."}` → returns `SessionView` with `VNCPort == 0` / `WSPort == 0` for headless (service skips `StartRemoteAccess`); after `d3=A`, `DisplayID` matches `^:\d+$`.
- `POST /api/v1/sessions/{id}/control` with `{"action": "launch_app", "params": {"app_path": "/usr/bin/xclock"}}` → sets `AppRunning = true` once the procmetrics window detector observes the new X window.
- `POST /api/v1/sessions/{id}/control` with `{"action": "screenshot"}` → with `WithCaptures(...)` wired in the integration harness (`r2-d1=A`), persists bytes to a file under the `t.TempDir()` store and records metadata in the captures store. Production `api/main.go` still uses the degraded path; that gap is tracked separately (`r2-i1`).
- `GET /api/v1/sessions/{id}/metrics` → returns `MetricsView` populated from the auto-set Monitor; `r2-d4=A` asserts via `require.Eventually` (5s/200ms) until non-nil, then schema-checks `CPUPercent`, `MemoryRSS`, `WindowDetected`.
- `DELETE /api/v1/sessions/{id}` → triggers `StopSession`; backend tears down x11vnc, websockify, Xvfb. Post-teardown verification uses `pgrep -af` filtered by test scenario name (`r2-d3=A`).

## 7. Target End State

- `go test ./... -tags=integration -timeout 300s` in `scenarios/vrooli-emulator/api/` passes on a machine with the X11 deps above, covering at minimum these assertions:
  1. Create headless session → `headless=true`, `vnc_port=0`, `state=running`, `display_id` matches `^:\d+$`.
  2. Create VNC session → `headless=false`, `vnc_port` in `[5900, 5999]`, `ws_port` in `[6080, 6180]`, `state=running`, `display_id` matches `^:\d+$`.
  3. Launch app → `POST /sessions/{id}/control launch_app` with `app_path=/usr/bin/xclock` → `AppRunning=true` within 10s (`require.Eventually`, 200ms tick).
  4. Screenshot capture — captures wired with `t.TempDir()`-backed store and file-on-disk assertion plus captures-store metadata assertion (`r2-d1=A`).
  5. Metrics tail — `require.Eventually` (5s timeout, 200ms tick) until `MetricsView` is non-nil; then assert schema fields `CPUPercent`, `MemoryRSS`, `WindowDetected` are present on the view (`r2-d4=A`).
  6. Destroy session → `DELETE /sessions/{id}` returns `200/204`; subsequent `GET` returns `404`; post-teardown stray-process check via `pgrep -af` filtered by the test scenario name (`r2-d3=A`) confirms zero matches.
- CLI smoke test exercises all five `session` subcommands against the same in-process `httptest.Server`, with default human output (no `--json` flag). The test builds `vrooli-emulator` once via `go build ./cli` into `t.TempDir()`, then execs it five times with `API_BASE_URL=<httptest URL>` in the environment.
- `TestPhase1SmokeHarness` performs the full sequence: start VNC session, start headless session, launch xclock on the headless session, capture screenshot, tail metrics, destroy both sessions. This replaces the retired `make phase1-ready` target.
- `vrooli scenario test vrooli-emulator` exercises the new integration suite alongside existing unit tests by virtue of `.vrooli/testing.json` declaring the `-tags=integration` flag.
- Distro validation matrix is documented in `scenarios/vrooli-emulator/docs/runbook.md` listing Ubuntu 22.04 and 24.04 as supported, with a manual verification timestamp recorded once during this item.
- `SessionView` exposes `display_id`; CLI mirror updated to include `DisplayID` and `summarize()` prints `Display: <display_id>` for headless sessions only (`r2-d5=A`).
- WebSocket coverage explicitly deferred to Phase 2 (`r2-d2=A`); no WS test is added in this item.
- No regressions in existing unit tests in `api/livedesktop/`, `api/screenrecording/`, or `api/captures/`.

## 8. Implementation Strategy

The strategy is phased so that each phase can be validated independently before moving on. Phases A–E are sequential; phases B and C can be parallelized after A is complete.

### Phase A — Add `display_id` to `SessionView` and CLI mirror (round 1 `d3=A`, round 2 `r2-d5=A`)

1. Extend `api/livedesktop/types.go`'s `SessionView` with a `DisplayID string \`json:"display_id"\`` field (alongside the existing `Headless bool` line for readability).
2. Update `Session.View()` (also in `types.go`) to populate `v.DisplayID = ""` initially, then set from `s.Display.DisplayID()` if `s.Display != nil`.
3. Update existing unit tests in `api/livedesktop/handler_test.go` and `api/livedesktop/service_test.go` to assert the field on happy-path creates (mock backend should set a non-empty DisplayID via the mock display).
4. Update the CLI's local `sessionView` struct in `cli/domains/sessions/sessions.go` to include `DisplayID string \`json:"display_id"\``. In `summarize()`, print `Display: <display_id>` on its own line when `Headless == true` and `DisplayID != ""`. Do not surface it for VNC sessions (where the user already has VNC/WS ports as the access path). Update any CLI unit tests that snapshot `summarize()` output for headless sessions.

### Phase B — Integration test suite (round 1 `d1=A`, `d2=A`; round 2 `r2-d1=A`, `r2-d3=A`, `r2-d4=A`)

5. Create `api/livedesktop/integration_test.go` with `//go:build integration` build tag in package `livedesktop`.
6. Add a `TestMain` (in a separate `integration_main_test.go` if needed to avoid build-tag conflicts) that performs the host-dependency precheck once and skips the suite via `os.Exit(0)` with a clear log line if any of `Xvfb`/`x11vnc`/`websockify`/`xdotool`/`pgrep` are missing. Same pattern as `screenrecording/display_test.go:63`.
7. Implement a helper `setupIntegrationServer(t)` that:
   - Constructs `livedesktop.NewLinuxBackend(logger)` with a discard or test-friendly slog handler.
   - Creates `livedesktop.NewInMemoryStore()` and `livedesktop.NewService(store, backend, logger)`.
   - Calls `svc.WithDataDir(t.TempDir())`.
   - Calls `svc.WithCaptures(...)` with a `FilesystemStore` (or equivalent canonical captures store) rooted at a separate `t.TempDir()` (`r2-d1=A`). Returns the captures store handle so tests can verify metadata.
   - Mounts a `mux.NewRouter()`, calls `livedesktop.NewHandler(svc).RegisterRoutes(router)`, and fronts with `httptest.NewServer(router)`.
   - Returns `(serverURL string, svc *Service, capStore CapturesStore, cleanup func())` where cleanup tears down all sessions, then closes the httptest server.
8. Write the six acceptance tests listed in §7 as discrete `Test*` functions:
   - `TestIntegration_CreateHeadlessSession`
   - `TestIntegration_CreateVNCSession`
   - `TestIntegration_LaunchApp`
   - `TestIntegration_Screenshot` — assert HTTP 200, then assert the captures store contains exactly one entry whose path resolves to a non-empty file under the temp captures dir (`r2-d1=A`).
   - `TestIntegration_Metrics` — `require.Eventually` (5s timeout, 200ms tick) until `MetricsView` is non-nil; assert `CPUPercent`, `MemoryRSS`, and `WindowDetected` fields are present on the returned view (`r2-d4=A`).
   - `TestIntegration_DestroySession` — assert delete + `404` on subsequent get; then run a `pgrep -af` filtered by the per-test scenario name and assert zero matches (`r2-d3=A`).
   Each calls `setupIntegrationServer(t)` and uses `require.Eventually` for any state convergence (no `time.Sleep`).
9. Add `TestPhase1SmokeHarness` that chains all six into a single sequence using the same helper.
10. Every test that creates a session must use a unique scenario name prefixed `test-phase1-` (e.g. `test-phase1-<random>`), set in the `POST /sessions` body so it appears in spawned-process argv. Every test must call `DELETE /sessions/{id}` in `t.Cleanup()`, then run the `pgrep -af` post-teardown check filtered by that scenario name. Failures mid-test must still tear down system processes.
11. WebSocket coverage is deferred entirely to Phase 2 (`r2-d2=A`); this item does not add any WS test or `gorilla/websocket.Dial` against the integration server.
12. Tests must not call `t.Parallel()` — port and Xvfb-display allocation are shared global state.

### Phase C — CLI smoke test (round 1 `d5=A`)

13. Create `api/livedesktop/cli_integration_test.go` (or a sibling file in the integration suite) that:
    - Calls `setupIntegrationServer(t)` to get the in-process httptest URL.
    - Builds `vrooli-emulator` once via `go build -o {tempDir}/vrooli-emulator ../../cli` (path resolved relative to the test working dir; use `runtime.Caller` if needed). Cache the build across subtests in the same test function.
    - Execs the binary with `os/exec.CommandContext`, passing `API_BASE_URL=<httptest URL>` in `cmd.Env`.
14. Cover each of the five subcommands in subtests:
    - `session list` (before any creates) — assert exit 0 and stdout contains `0 session(s)`.
    - `session create --scenario test-phase1-cli --headless --width 1024 --height 768` — assert exit 0, capture session ID from stdout via regex on `id=<uuid>`, and assert stdout contains a `Display: :<digits>` line (since `r2-d5=A` surfaces it for headless).
    - `session list` (after create) — assert stdout contains the captured ID.
    - `session exec <id> launch_app --app-path /usr/bin/xclock` — assert exit 0 and stdout contains `Action "launch_app" completed`.
    - `session logs <id>` (single-shot, no `-f`) — assert exit 0 and stdout contains the session ID and `state=running`.
    - `session destroy <id>` — assert exit 0 and stdout contains `Session <id> destroyed`.
15. Assert on exit code + minimum stdout substring matches (session ID, key status keywords, `Display:` line for headless). Do not assert on the full `summarize()` output format — that couples to renderer implementation.

### Phase D — Runner and matrix wiring (round 1 `d1=A`, `d6=A`)

16. Update `.vrooli/testing.json`: add an `integration` section that declares the build tag `integration` for the Go module under `api/`, so `vrooli scenario test vrooli-emulator` invokes `go test -tags=integration ./...` for the API module. Match the schema at `scripts/scenarios/testing/schemas/testing.schema.json`.
17. Update `scenarios/vrooli-emulator/docs/runbook.md` (or create the file) with a "Supported Distros" section listing Ubuntu 22.04 and 24.04, the verified host-package list (`x11-apps`, `xvfb`, `x11vnc`, `websockify`, `xdotool`, `procps` for `pgrep`), and the verification timestamp from this item.
18. Validate end-to-end: run `vrooli scenario test vrooli-emulator` locally and confirm both unit and integration tests execute (or integration tests skip gracefully when X11 deps absent).

### Phase E — Documentation and verification

19. Document how to run the integration suite locally in `scenarios/vrooli-emulator/docs/runbook.md` — the rehomed chore `chore/vrooli-emulator-documentation` will absorb broader runbook content; this item is responsible only for the integration-test section.
20. Run `go test -tags=integration ./... -timeout 300s` in `scenarios/vrooli-emulator/api/` and confirm green.
21. Run `go vet ./...` and `gofumpt -l .` — confirm no lint/format drift.
22. Run the CLI smoke test as part of the same suite.

## 9. Contract Decisions

### Settled (authoritative — do not revisit)
- **Route prefix**: `/api/v1/sessions/` (research Finding 11, confirmed in `handler.go:25`).
- **Headless flag**: `headless: true` on create skips VNC/websockify (research Finding 10).
- **Launch contract**: `app_path` is a required field for `launch_app` control action (research Finding 7).
- **CLI surface under test**: exactly `session list`, `session create`, `session destroy`, `session exec`, `session logs`.
- **Absorbed scope ownership**: Phase 1 integration smoke harness and per-distro validation matrix live in this item.
- **Acceptance glob**: `acceptance_allow=scenarios/vrooli-emulator/**`; `acceptance_deny={api/main.go, cli/install.sh, cli/install.ps1, Makefile}` — preserves current wiring.
- **Round 1 `d1=A`**: New file `api/livedesktop/integration_test.go` with `//go:build integration`; same package; `.vrooli/testing.json` updated to invoke `-tags=integration`.
- **Round 1 `d2=A`**: In-process `httptest.Server` fronting real `LinuxBackend`; no subprocess for the API.
- **Round 1 `d3=A`**: Add `display_id` field to `SessionView`; populate in `Session.View()` from `s.Display.DisplayID()`; update existing unit tests.
- **Round 1 `d4=A`**: `xclock` from `x11-apps` is the `app_path` fixture; tests skip if `/usr/bin/xclock` is missing.
- **Round 1 `d5=A`**: Build CLI binary to `t.TempDir()` once via `go build ./cli`; exec with `os/exec` and `API_BASE_URL=<httptest URL>`.
- **Round 1 `d6=A`**: Distro matrix is documented in `docs/runbook.md` (Ubuntu 22.04 + 24.04); no CI matrix job; one-time manual verification.
- **Round 2 `r2-d1=A`**: Integration harness wires `WithCaptures(...)` with a `t.TempDir()`-backed captures store; `TestIntegration_Screenshot` asserts both file-on-disk persistence and captures-store metadata. Production `api/main.go` is intentionally NOT modified here — the production-wiring gap is tracked in info `r2-i1` as a separate follow-up.
- **Round 2 `r2-d2=A`**: WebSocket endpoint coverage is deferred entirely to Phase 2 (`execute/scenario-to-desktop-emulator-iframe-embed`); no WS test in Phase 1.
- **Round 2 `r2-d3=A`**: Post-teardown stray-process detection uses `pgrep -af` filtered by the per-test scenario name (prefix `test-phase1-`). LinuxBackend already passes the scenario name into spawned-process argv, so this is reliable on a shared dev host.
- **Round 2 `r2-d4=A`**: Metrics test uses `require.Eventually` (5s timeout, 200ms tick) until `MetricsView` is non-nil; then asserts schema fields `CPUPercent`, `MemoryRSS`, `WindowDetected` are present on the view.
- **Round 2 `r2-d5=A`**: CLI's local `sessionView` struct is updated to include `DisplayID`; `summarize()` prints `Display: <display_id>` for headless sessions only. CLI smoke test asserts the line on the headless `create` subtest.

### Open
None. All workshop decisions resolved as of round 003 finalize.

### Prohibited by earlier decisions (do not revisit)
- No shared `packages/procmetrics` library (research Finding 12; procmetrics stays internal to the scenario).
- No subprocess-based API exercise (round 1 `d2=A` chose in-process).
- No env-var-gated integration tests (round 1 `d1=A` chose build-tag gating).
- No custom Go fixture binary for `app_path` (round 1 `d4=A` chose xclock).
- No `--json` CLI output in tests (memory rule `feedback_cli_default_human_output.md`).
- No CI matrix job for distros (round 1 `d6=A` chose documentation-only).
- No mocked `LinuxBackend` in the integration suite — that defeats the purpose; mocked tests stay in the existing `*_test.go` files.
- No `t.Parallel()` in integration tests — shared global Xvfb-display and port-window state.
- No edits to `api/main.go`, `cli/install.sh`, `cli/install.ps1`, or `Makefile` — `acceptance_deny` enforced.
- No WebSocket test in Phase 1 — `r2-d2=A` defers all WS coverage to Phase 2.
- No PID-exposure surface change to `SessionView` for stray-process detection — `r2-d3=A` chose `pgrep -af` argv filtering instead.
- No production wiring of `WithCaptures()` in `api/main.go` from this item — `r2-d1=A` constrains captures wiring to the integration harness; production-wiring is tracked in `r2-i1` as a separate follow-up.

## 10. Testing Plan

This item IS the testing plan; it delivers automated tests rather than consuming them. The verification gates below confirm those tests work as intended:

1. **All six HTTP acceptance tests pass on a host with Xvfb et al.** — `go test -tags=integration -timeout 300s ./api/livedesktop/...` returns exit 0.
2. **All six skip gracefully on a host without Xvfb** — `TestMain`-level precheck fires `os.Exit(0)` with a clear log line; the suite returns exit 0 rather than a flaky failure.
3. **CLI smoke test passes** — exit 0, no panics, all five subcommands emit the expected human output substrings, including `Display: :<digits>` on the headless `create` subtest.
4. **`TestPhase1SmokeHarness` passes** — the single chained test runs end-to-end without orphan processes.
5. **Post-teardown process check passes** — `pgrep -af` filtered by the per-test scenario name returns zero matches after `t.Cleanup()` runs.
6. **Existing unit tests continue to pass** — `go test ./api/...` (no tags) still green; no regression in mock-based coverage. The new `display_id` assertions in `handler_test.go` / `service_test.go` pass against the mock backend.
7. **`vrooli scenario test vrooli-emulator`** runs both unit and integration tests; command exits 0; output shows both suites executed.
8. **Distro matrix verified** — runbook lists 22.04 and 24.04 with a verification timestamp ≥ this item's start date.
9. **Captures behavior asserted** — `TestIntegration_Screenshot` confirms the captures store contains exactly one entry whose path resolves to a non-empty file under the temp captures dir.
10. **Metrics behavior asserted** — `TestIntegration_Metrics` Eventually-polls until `MetricsView` is non-nil and confirms the schema fields (`CPUPercent`, `MemoryRSS`, `WindowDetected`) are present.

All checks are scriptable; the implementation must not rely on manual inspection.

## 11. Rollout/Validation Checklist

- [ ] `display_id` field added to `SessionView`; `Session.View()` populates it; existing unit tests updated.
- [ ] CLI's local `sessionView` mirror updated to include `DisplayID`; `summarize()` prints `Display: <display_id>` for headless sessions only.
- [ ] `api/livedesktop/integration_test.go` created with `//go:build integration` tag.
- [ ] `setupIntegrationServer(t)` helper implemented and reused across all integration tests; helper wires `WithCaptures(...)` with a `t.TempDir()`-backed captures store and returns the store handle.
- [ ] Six acceptance tests written and passing: headless create, VNC create, launch_app, screenshot (with file-on-disk + captures-store metadata assertion), metrics (Eventually-poll + schema-field assertion), destroy (with post-teardown `pgrep -af` zero-match assertion).
- [ ] WebSocket coverage explicitly deferred to Phase 2 — no WS test added in this item.
- [ ] `TestPhase1SmokeHarness` written and passing.
- [ ] Per-test scenario names use the `test-phase1-` prefix so `pgrep -af` argv filtering reliably identifies test-owned processes.
- [ ] CLI smoke test exercises all five session subcommands using a built binary in `t.TempDir()` with `API_BASE_URL` env var; headless `create` subtest asserts the `Display: :<digits>` line.
- [ ] `.vrooli/testing.json` updated so `vrooli scenario test vrooli-emulator` picks up the integration tag.
- [ ] `scenarios/vrooli-emulator/docs/runbook.md` lists Ubuntu 22.04 + 24.04 as supported distros with a verification timestamp.
- [ ] `go vet ./...` and `gofumpt -l .` report no issues in modified files.
- [ ] Existing unit tests in `api/livedesktop/` still pass (`go test ./api/livedesktop/...` with no tags).
- [ ] `vrooli scenario test vrooli-emulator` exits 0 end-to-end.
- [ ] No edits to `api/main.go`, `cli/install.sh`, `cli/install.ps1`, or `Makefile` (verify via `git status`).

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Orphan Xvfb/x11vnc/websockify processes from failed teardown pollute host | Medium | Medium | Every test uses `t.Cleanup` with `DELETE /sessions/{id}`; dedicated post-teardown `pgrep -af` zero-match assertion (`r2-d3=A`); tag sessions with the test scenario-name prefix `test-phase1-<random>` so `pgrep -af` argv filter is reliable on a shared dev host |
| Port conflicts between parallel tests for VNC/websockify (5900-5999 / 6080-6180) | Medium | Medium | `LinuxBackend.findAvailablePort` already serializes via `portMutex`; tests do NOT call `t.Parallel()` |
| Host missing X11 deps causes all tests to fail loudly in CI on minimal images | Medium | Low | `TestMain`-level precheck skips with clear message when binaries missing (pattern from `screenrecording/display_test.go:63`); precheck includes `pgrep` (provided by `procps`) since the post-teardown assertion depends on it |
| `xclock` not present on a given distro | Medium | Medium | Test pre-checks `/usr/bin/xclock` and skips with `apt install x11-apps` hint; runbook (Phase D) documents the dependency |
| `display_id` not populated on the mock backend, breaking unit-test assertions added in Phase A | Low | Medium | Update `mockPlatformBackend` in `service_test.go` to return a non-empty mock display ID (e.g., `:99`) when a display is allocated |
| CLI subprocess test build cost (~5-10s) slows the suite | Low | Low | Build once in a `sync.Once`-guarded helper; share the binary path across all CLI subtests |
| `go build ./cli` fails because the test runs from an unexpected working directory | Low | Medium | Resolve the CLI source path via `runtime.Caller(0)` then `filepath.Join(_, "..", "..", "..", "cli")`; assert the resolved path exists before invoking `go build` |
| Captures wiring decision (`r2-d1=A`) creates test/prod divergence — tests pass with persistence wired while production silently drops captures | Medium | Medium (downstream) | File a follow-up `execute/wire-captures-in-vrooli-emulator-main` per info `r2-i1` to wire `WithCaptures()` in `api/main.go` before Phase 2's iframe-embed consumer needs persisted thumbnails |
| Flaky tests on slow hosts (app launch takes >2s to show `AppRunning=true`) | Medium | Medium | Use `require.Eventually` with 10s timeout and 200ms tick; no arbitrary `time.Sleep` |
| Headless DISPLAY collision with an existing X server on the host (e.g., `:0` is the user's desktop) | Low | High | `LinuxBackend` already chooses an unused display ID > `:0` via its display allocator; the assertion `display_id ~= ^:\d+$` does not pin a specific value |
| WebSocket path uncovered until Phase 2 lands — regressions in router wiring or upgrader config could slip in | Medium | Medium | Phase 2 (`execute/scenario-to-desktop-emulator-iframe-embed`) explicitly owns first WS coverage (`r2-d2=A`); explicit handoff documented in §9 Settled and §14 Non-goals |
| `pgrep -af` post-teardown check yields false positives if another process happens to contain the `test-phase1-` substring in its argv (extremely unlikely on a dev host but possible) | Very Low | Low | Use a sufficiently random suffix in the scenario name (e.g. UUID) so the substring is effectively unique to this test run |
| Metrics `Eventually` poll times out before `MetricsView` populates because of an unrelated Monitor bug | Low | Medium | 5s budget is generous given Monitor is set synchronously in `StartSession`; if the assertion fails, the failure clearly points to a Monitor regression rather than test flake |

## 13. Scenario Restart Note

This plan adds a new field (`display_id`) to `SessionView` (`d3=A`) and surfaces it via the CLI (`r2-d5=A`). In production, that change requires `vrooli scenario restart vrooli-emulator` for the running binary to expose the new field. **Per the user's feedback in `feedback_no_restart_active_scenario.md`, the executing agent must NOT run the restart itself** — it writes the changes to disk and stops. The user (or a subsequent step outside the agent's sandbox) handles the actual restart.

The integration tests themselves do not require a running scenario instance — they construct their own service via `setupIntegrationServer(t)` and front it with `httptest.NewServer`. So the test suite can be developed and run without ever restarting the production scenario binary.

## 14. Non-goals / Prohibited Patterns

- Do not add mock-based unit tests under the integration tag — the point of these tests is to exercise the **real** `LinuxBackend`; mocked tests belong in the existing `*_test.go` files.
- Do not test UI, iframe embed, or smoketest delegation — each has its own execute item with its own acceptance.
- Do not exercise the WebSocket endpoint in Phase 1 — `r2-d2=A` defers all WS coverage to Phase 2 (`execute/scenario-to-desktop-emulator-iframe-embed`).
- Do not assume host has `xclock` / `xterm` / any specific X app without a precheck.
- Do not use `time.Sleep` for synchronization — use `require.Eventually` with bounded timeouts.
- Do not run tests in parallel (`t.Parallel()`) — port and Xvfb-display allocation are shared global state inside the process.
- Do not rely on `--json` CLI output — memory rule `feedback_cli_default_human_output.md` requires default human output.
- Do not `vrooli scenario restart` from inside the agent — memory rule `feedback_no_restart_active_scenario.md`.
- Do not modify `scenarios/scenario-to-desktop/**` — acceptance glob is scoped to `scenarios/vrooli-emulator/**`.
- Do not edit `api/main.go`, `cli/install.sh`, `cli/install.ps1`, or the scenario `Makefile` — `acceptance_deny` forbids it. In particular, do not wire `WithCaptures()` in `api/main.go` from this item — that production gap is tracked separately in info `r2-i1`.
- Do not silently drop the Phase 1 smoke harness or distro matrix — both are absorbed deliverables tracked in §7 and §11.
- Do not change `Session.View()` lock semantics — only ADD the `DisplayID` line; preserve the existing lock acquisition pattern.
- Do not surface `display_id` in CLI `summarize()` for VNC sessions — `r2-d5=A` constrains the print to headless sessions only.

## 15. Definition of Done

1. All round 2 workshop decisions resolved (`r2-d1=A` through `r2-d5=A`); §9 "Open" section is empty.
2. `display_id` exposed in `SessionView`; CLI mirror updated to include `DisplayID` and `summarize()` prints `Display: <display_id>` for headless sessions only.
3. All six HTTP acceptance tests pass on the dev host with X11 deps installed.
4. `TestPhase1SmokeHarness` passes end-to-end.
5. CLI smoke test passes, covering all five session subcommands with default human output; headless `create` subtest asserts the `Display: :<digits>` line.
6. Captures persistence asserted via `t.TempDir()`-backed `WithCaptures()` store: `TestIntegration_Screenshot` confirms file-on-disk and captures-store metadata.
7. Metrics asserted via `require.Eventually` (5s/200ms) until `MetricsView` is non-nil; schema fields `CPUPercent`, `MemoryRSS`, `WindowDetected` confirmed present.
8. WebSocket coverage explicitly deferred to Phase 2 (`r2-d2=A`); no WS test added in this item; deferral is documented in §9, §11, and §14.
9. Post-teardown process-cleanup assertion via `pgrep -af` filtered by the test scenario-name prefix `test-phase1-` confirms zero stray `Xvfb`/`x11vnc`/`websockify` processes belonging to the test runs.
10. `vrooli scenario test vrooli-emulator` exits 0 and visibly runs the new integration suite.
11. Existing unit tests continue to pass — no regressions in `api/livedesktop/*_test.go`, `api/screenrecording/*_test.go`, or `api/captures/*_test.go`.
12. `scenarios/vrooli-emulator/docs/runbook.md` lists Ubuntu 22.04 + 24.04 as supported with a verification timestamp.
13. `go vet ./...` and `gofumpt -l .` report no issues in modified files.
14. No orphan Xvfb/x11vnc/websockify processes on the dev host after full suite run.
15. `git status` confirms no edits to `acceptance_deny` paths.
16. Agent did not run `vrooli scenario restart` — left changes on disk for the user to restart.
