# Scaffold vrooli-emulator Scenario (API + Headless Sessions + VNC Proxy)

## 1. Purpose

Create the standalone `scenarios/vrooli-emulator/` scenario as the Phase 1 backbone of the emulator-platform initiative. This lands the API, minimal operator CLI, placeholder UI, and scenario lifecycle wiring so that downstream work (Phase 1 acceptance tests, chore to remove livedesktop from scenario-to-desktop, Phase 2 standalone UI + iframe, Phase 3 smoketest migration) can execute in parallel or sequence.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read scenario-generation refactor
```

Source materials:
```bash
swarm-manager backlog file-get --kind research --name emulator-extraction-and-service-plan --path conclusion.md
swarm-manager initiatives get --name emulator-platform
```

## 3. Problem Statement

scenario-to-desktop currently owns livedesktop (virtual display + VNC sessions) and procmetrics (process/window monitoring). The emulator-platform initiative requires these capabilities to live in a dedicated scenario so that:
- Multiple consumers (smoketest, deployment-manager visual validation, future remote backends) share one session service.
- scenario-to-desktop shrinks to its domain (deployment automation) and stops hosting display infrastructure.
- Headless session allocation becomes a first-class primitive — smoketest no longer duplicates `xvfb-run` logic.

Phase 1 uses a **hard-cut** strategy (Finding 15): the emulator ships concurrent with livedesktop removal from scenario-to-desktop; scenario-to-desktop's desktop-viewer UI is intentionally broken between Phase 1 and Phase 2. This item lands only the emulator side; companion chore `scenario-to-desktop-remove-livedesktop` handles deletion.

## 4. Scope

### In scope
- Scaffold `scenarios/vrooli-emulator/` using the `react-vite` template: `vrooli scenario generate react-vite --id vrooli-emulator`. The template emits `api/`, `ui/`, `cli/`, `.vrooli/service.json`, `Makefile`, `PRD.md`, `README.md`, and `cli/install.sh`.
- Copy `api/livedesktop/` and `api/procmetrics/` from scenario-to-desktop into `scenarios/vrooli-emulator/api/` as internal packages. Update module import paths.
- Rename route prefix from `/api/v1/livedesktop/` → `/api/v1/sessions/`.
- Drop `GET /sessions/{id}/artifact`. Remove `findArtifact`, `FindArtifact`, `currentPlatform`, `packaging` import, and `vrooliRoot` field from `Service`.
- Require explicit `app_path` on `POST /sessions/{id}/launch`. Return `400 {"error":"app_path is required"}` if missing.
- Add `headless` boolean field to session creation. Headless sessions allocate Xvfb only — skip x11vnc/websockify. `VNCPort=0`, `WSPort=0`, `RemoteAccess=nil`. `/ws` proxy returns HTTP 400 on headless sessions.
- Ship `cli/` with operator-surface commands: `session list`, `session create`, `session destroy`, `session exec`, `session logs`, `metrics tail` — thin HTTP-client wrappers. Port from `scenarios/scenario-to-desktop/cli/domains/livedesktop/` with `livedesktop` → `session` rename; drop obsolete flags (auto-artifact, `vrooliRoot`-dependent lookup); preserve existing flag parsing and human-readable output formatting.
- Minimal placeholder UI (react-vite default + `/health` page + "Sessions" stub) — richer UI work is tracked in sibling item `execute/vrooli-emulator-standalone-ui`, which (despite its name) lives entirely inside `scenarios/vrooli-emulator/ui/` — there is no separate standalone UI app.
- `.vrooli/service.json` with ports (template defaults), host tools (Xvfb, x11vnc, websockify, openbox, xdotool), lifecycle (setup/develop/stop/test), no scenario dependencies.
- `Makefile` with `start`, `stop`, `restart`, `test`, `logs` targets matching scenario conventions.
- Move unit tests from livedesktop and procmetrics; delete tests that cover removed behavior (findArtifact, `/artifact` endpoint).
- `go build ./...` and `go test ./...` pass in the new scenario's api/ and cli/.

### Out of scope (handled by sibling items)
- Deletion of `api/livedesktop/` and `api/procmetrics/` from scenario-to-desktop → `chore/scenario-to-desktop-remove-livedesktop`.
- Phase 1 integration/acceptance tests → `execute/emulator-acceptance-tests-phase-1`.
- Full session-management UI (drawer, VNC canvas, controls) → `execute/vrooli-emulator-standalone-ui` (inside `scenarios/vrooli-emulator/ui/`).
- External-url endpoint and iframe embed → `execute/vrooli-emulator-external-url-endpoint`, `execute/scenario-to-desktop-emulator-iframe-embed`.
- Smoketest migration → `execute/smoketest-delegate-display-to-emulator`.
- Documentation → `chore/vrooli-emulator-documentation`.

## 5. Non-goals / Prohibited Patterns

- NO dual-run or compatibility alias from `/api/v1/livedesktop/` to `/api/v1/sessions/`. Greenfield — hard cut.
- NO gRPC. HTTP REST + WebSocket only.
- NO shared Go module for procmetrics. Internal package only.
- NO cross-scenario artifact discovery. Callers resolve `app_path`.
- NO database at Phase 1. `InMemoryStore` only.
- NO `lib/` folder in the scenario — use `.vrooli/service.json` lifecycle config (per CLAUDE.md).
- NO changes outside `scenarios/vrooli-emulator/**`.
- NO separate "standalone UI" scenario or app. All UI lives inside `scenarios/vrooli-emulator/ui/`.
- NO restart of the scenario inside which the executing agent is running. Restarting the **target** scenario (`vrooli-emulator`) is expected and encouraged once scaffolding lands — it is newly created and not the agent's own sandbox.

## 6. Current Technical Context

### Source (scenario-to-desktop → vrooli-emulator)
- `scenarios/scenario-to-desktop/api/livedesktop/` — 17 files (action*, handler, janitor, platform*, proxy, service, store, types).
- `scenarios/scenario-to-desktop/api/procmetrics/` — 9 files (factory, interfaces, monitor, proc_reader, types, window_detector).

### Critical call sites
- `handler.go:22-35` — route registration (rename prefix on all 11 routes).
- `handler.go:30` — `GET artifact` route (delete).
- `handler.go:92-97` — `launchApp` reads `app_path` already; add non-empty validation.
- `service.go:170-229` — `LaunchApp`: strip auto-discovery branch (lines ~180-186).
- `service.go:284-310` — delete `FindArtifact`, `findArtifact`, `currentPlatform`, `packaging` import.
- `service.go` — `NewService(store, backend, logger, vrooliRoot)` → `NewService(store, backend, logger)` (drop field).
- `main.go:205-220` — reference implementation of service wiring.
- `types.go:23-54` — `Session` struct (add `Headless bool`).
- `platform.go` / `platform_linux.go` — `PlatformBackend`: thread `headless` through `StartDisplay` options.

### Scaffold reference
- `templates/scenarios/react-vite/` — template source (generates `api/`, `ui/`, `cli/`, `.vrooli/service.json`, `Makefile`, `PRD.md`, `README.md`, `cli/install.sh` with default port ranges API `15000-19999`, UI `35000-39999`, WS `25000-29999`).
- `scenarios/scenario-to-desktop/.vrooli/service.json` — host tools, lifecycle entries for livedesktop.
- `scenarios/scenario-to-desktop/cli/` — CLI layout (main.go, domains/, internal/).
- `scenarios/landing-page-business-suite/cli/` — modern Go CLI pattern reference.
- `scenarios/scenario-to-desktop/api/go.mod` — replace directives for `api-core`, `repo-contract-go`, `proto`. Paths resolve identically from `scenarios/vrooli-emulator/api/` (same depth).

### Module naming
- API module: `vrooli-emulator-api` (mirrors `scenario-to-desktop-api`).
- CLI module: per template output.
- CLI binary name: `vrooli-emulator` (produced by template; consumed by Makefile/Service lifecycle).

## 7. Target End State

```
scenarios/vrooli-emulator/
├── .vrooli/
│   ├── service.json           # template default ports (API 15000-19999, UI 35000-39999, WS 25000-29999), host tools, lifecycle, no scenario deps
│   └── testing.json           # template default
├── api/
│   ├── go.mod                 # module vrooli-emulator-api
│   ├── main.go                # router, /health, session service, janitor
│   ├── livedesktop/           # moved; prefix /api/v1/sessions/, no findArtifact, headless-aware
│   └── procmetrics/           # moved; window_detector degrades on headless
├── cli/
│   ├── go.mod
│   ├── main.go
│   ├── domains/sessions/      # list/create/destroy/exec/logs
│   ├── domains/metrics/       # tail
│   └── install.sh             # template default, installs `vrooli-emulator` binary
├── ui/                        # react-vite defaults + /health + "Sessions" stub (placeholder)
├── Makefile                   # start/stop/restart/test/logs
├── PRD.md
└── README.md
```

### API surface
- `GET /health` → 200.
- `POST /api/v1/sessions` — body `{scenario_name, headless?, width?, height?, platform?}`; 201 with session JSON.
- `GET /api/v1/sessions` — list.
- `GET /api/v1/sessions/{id}` — detail.
- `DELETE /api/v1/sessions/{id}` — stop.
- `POST /api/v1/sessions/{id}/heartbeat`.
- `POST /api/v1/sessions/{id}/launch` — body `{app_path}` (required).
- `POST /api/v1/sessions/{id}/control` — 13 existing actions.
- `GET /api/v1/sessions/{id}/metrics`.
- `GET /api/v1/sessions/{id}/files/{filename}`.
- `WS /api/v1/sessions/{id}/ws` — VNC proxy (400 on headless).

### CLI surface (human-readable default output)
- `vrooli-emulator session list`
- `vrooli-emulator session create --scenario <name> [--headless] [--width X --height Y]`
- `vrooli-emulator session destroy <id>`
- `vrooli-emulator session exec <id> <action> [--app-path P] [--env K=V]...`
- `vrooli-emulator session logs <id> [-f]`
- `vrooli-emulator metrics tail <id>`

## 8. Implementation Strategy

### Phase A — Scaffold (~15 min)
1. `vrooli scenario generate react-vite --id vrooli-emulator --display-name "Vrooli Emulator" --description "Virtual display + session management for scenario validation"`.
2. Confirm `.vrooli/service.json`, `Makefile`, `api/go.mod`, `api/main.go`, `ui/package.json`, `cli/main.go`, `cli/install.sh` are present.
3. `cd scenarios/vrooli-emulator/api && go mod tidy` to seed replace directives; inspect go.mod and copy additional replace directives (`api-core`, `repo-contract-go`, `proto`) from `scenarios/scenario-to-desktop/api/go.mod` as needed.

### Phase B — Move livedesktop + procmetrics (1-2h)
1. Copy `scenarios/scenario-to-desktop/api/livedesktop/` → `scenarios/vrooli-emulator/api/livedesktop/` verbatim.
2. Copy `scenarios/scenario-to-desktop/api/procmetrics/` → `scenarios/vrooli-emulator/api/procmetrics/` verbatim.
3. Rewrite imports: `scenario-to-desktop-api/procmetrics` → `vrooli-emulator-api/procmetrics`; `scenario-to-desktop-api/livedesktop` → `vrooli-emulator-api/livedesktop`.
4. `cd scenarios/vrooli-emulator/api && go mod tidy`.

### Phase C — API contract changes (~1h)
1. `handler.go`: rename route prefix on 11 routes (`/api/v1/livedesktop/` → `/api/v1/sessions/`).
2. `handler.go`: delete `GET /sessions/{id}/artifact` route + `findArtifact` handler.
3. `handler.go`: `launchApp` rejects empty `app_path` → `400 {"error":"app_path is required"}`.
4. `service.go`: delete `FindArtifact`, `findArtifact`, `currentPlatform`, and `packaging` import; strip auto-discovery branch in `LaunchApp`; change `NewService` signature to drop `vrooliRoot`.
5. Delete tests covering removed behavior.

### Phase D — Headless support (1-2h)
1. `types.go`: add `Headless bool` to `Session` (JSON `headless`).
2. `platform.go`: add `Headless bool` to display-start options.
3. `handler.go`: `startSession` reads `headless` and passes through.
4. `service.go`: `StartSession` threads `headless` to backend.
5. `platform_linux.go`: when `Headless`, allocate Xvfb only; skip `startRemoteAccess`; leave `RemoteAccess=nil`, `VNCPort=0`, `WSPort=0`.
6. `proxy.go`: `handleVNCProxy` returns `400 {"error":"no VNC proxy on headless session"}` if `session.Headless || session.RemoteAccess == nil`.
7. `procmetrics/window_detector.go`: graceful degrade — return empty window list without error if `DISPLAY` is unset or xdotool fails to connect; PID-based metrics keep flowing.
8. New tests (see §10).

### Phase E — main.go wiring (~30 min)
1. Replace template's stub `api/main.go` with a router (gorilla/mux), `/health`, session service construction, janitor (30s interval, 30m TTL). Use scenario-to-desktop's main.go `livedesktop` block as the template, strip unrelated domains.
2. Port resolution: `API_PORT` env var; fallback to template `ports.go` pattern.

### Phase F — CLI (2-3h)
1. Replace template's stub `cli/main.go` with a cobra root `vrooli-emulator`.
2. Port scenario-to-desktop's livedesktop CLI from `scenarios/scenario-to-desktop/cli/domains/livedesktop/` → `scenarios/vrooli-emulator/cli/domains/sessions/`. Rename the domain/command from `livedesktop` to `session` throughout. Preserve the tested flag parsing and human-readable output formatting; drop obsolete flags tied to removed behavior (auto-artifact discovery, `vrooliRoot`-dependent lookup).
3. Add `metrics tail` subcommand (`cli/domains/metrics/tail.go`) as a thin HTTP client against `GET /api/v1/sessions/{id}/metrics` with streaming/poll semantics matching the existing CLI pattern.
4. Human-readable default output; `--json` flag opt-in.
5. Migrate any existing CLI tests from scenario-to-desktop's livedesktop domain into `cli/domains/sessions/` with renamed expectations. Delete tests covering removed flags.
6. Reuse template's `install.sh` unchanged (installs the `vrooli-emulator` binary to the standard location).

### Phase G — UI placeholder (~30 min)
1. Ensure template-generated UI builds.
2. Add `/health` page and "Sessions" stub list (no API wiring — placeholder only; the full UI is tracked in `execute/vrooli-emulator-standalone-ui`).
3. `pnpm install --ignore-workspace && pnpm run build` produces `ui/dist/index.html`.

### Phase H — service.json + Makefile (~30 min)
1. Keep template default ports (API `15000-19999`, UI `35000-39999`, WS `25000-29999`).
2. Add hostTools to `.vrooli/service.json`: Xvfb (required on linux), x11vnc, websockify, openbox, xdotool (required on linux).
3. Lifecycle: `lifecycle.health` = `/health` on api + ui; `lifecycle.setup` builds api binary, installs + builds UI, installs CLI; `lifecycle.develop` runs api binary + UI server; `lifecycle.stop` uses pkill by binary name.
4. No `dependencies.scenarios`.
5. Keep template's `Makefile` (start/stop/restart/test/logs targets delegate to `vrooli scenario <action> vrooli-emulator`).

### Phase I — Verify (~30 min)
1. `cd scenarios/vrooli-emulator/api && go build ./... && go test ./...` passes.
2. `cd scenarios/vrooli-emulator/cli && go build ./...` passes.
3. `cd scenarios/vrooli-emulator/ui && pnpm install --ignore-workspace && pnpm run build` passes.
4. `git diff --name-only` shows only paths under `scenarios/vrooli-emulator/**`.
5. Scenario lifecycle spot-check: attempt `vrooli scenario restart vrooli-emulator` (requires Xvfb/xdotool on host). On success, probe `GET /health` against the allocated API port. If the sandbox blocks host lifecycle, emit a structured skip note in the completion summary (command attempted + observed error) and rely on build/test passes — do NOT fail the item for environment-only reasons.

## 9. Contract Decisions

### API
- Route prefix `/api/v1/sessions/` (generic, not livedesktop-branded).
- Request bodies: JSON, `snake_case` field names.
- `POST /api/v1/sessions` accepts optional `headless: true`; response `vnc_port` and `ws_port` are `0` when headless.
- `POST /api/v1/sessions/{id}/launch` requires `app_path`; empty returns 400.
- `WS /api/v1/sessions/{id}/ws` returns 400 on headless.
- Error shape preserved from existing handler.

### CLI
- Human-readable default output (per `feedback_cli_default_human_output`).
- Optional `--json` for scripted consumers.
- `session create` prints ID, DISPLAY, VNC URL (when non-headless), and state.
- Binary name `vrooli-emulator` (template default).

### Data model
- `Session.Headless bool` (JSON `headless`).
- `VNCPort`/`WSPort` remain `int`; `0` = unused.
- No backwards-compat — greenfield scenario.

### Ports (from template defaults)
- API_PORT range `15000-19999`.
- UI_PORT range `35000-39999`.
- WS_PORT range `25000-29999` (allocated by template; not all scenarios use it; the emulator's WebSocket proxy runs on the API port under `/api/v1/sessions/{id}/ws`, while per-session websockify procs use dynamic ports from this range).

## 10. Testing Plan (automated; no manual checklists per user feedback)

### Moved unit tests
- All existing tests from `livedesktop/*_test.go` and `procmetrics/*_test.go` compile and pass under the new module path.
- Delete tests whose subject is removed (`findArtifact`, `GET artifact`, `vrooliRoot`).

### New tests added in this item
- `service_test.go`:
  - `StartSession` with `headless=true` yields session with `RemoteAccess == nil`, `VNCPort == 0`, display allocated via a fake `PlatformBackend` (tests should not require a real Xvfb).
- `handler_test.go`:
  - `POST /api/v1/sessions` body `{headless:true}` → 201 with `"vnc_port": 0`, `"ws_port": 0`.
  - `POST /api/v1/sessions/{id}/launch` body `{}` → 400 `{"error":"app_path is required"}`.
  - `WS /api/v1/sessions/{id}/ws` on headless session → 400 `{"error":"no VNC proxy on headless session"}`.
  - `GET /api/v1/livedesktop/sessions` returns 404 (prefix moved, not aliased).
- `procmetrics/window_detector_test.go`: with `DISPLAY` unset (via `t.Setenv("DISPLAY", "")`), `enumerate()` returns `([]Window{}, nil)` — no error, empty slice.
- Build gate: `go build ./...` and `go test ./...` exit 0.

### Test isolation
- Platform behaviors (Xvfb allocation, x11vnc start) are already abstracted behind `PlatformBackend`. Tests use a fake implementation — they do NOT require `xvfb` or `x11vnc` on the test host.
- `window_detector` tests rely on `DISPLAY` environment manipulation via `t.Setenv`; they do NOT require `xdotool`.

### Test helpers (visibility)
- All test helpers inside `api/livedesktop/` and `api/procmetrics/` remain **unexported** (package-private `*_test.go` scope). Do NOT introduce an exported `NewTestServer` or similar cross-package harness in this item.
- The acceptance-tests sibling (`execute/emulator-acceptance-tests-phase-1`) will construct its own end-to-end harness against the public HTTP surface. This keeps this item tightly scoped and lets the sibling shape its own fixtures.

### Out of scope
- End-to-end launch with real Electron → `execute/emulator-acceptance-tests-phase-1`.
- Live-server CLI integration tests → sibling items.
- Smoketest migration → sibling item.
- Exported test harness for cross-package consumers → sibling item owns this if it ever becomes needed.

## 11. Rollout / Validation Checklist

- [ ] `scenarios/vrooli-emulator/` exists with `api/`, `cli/`, `ui/`, `.vrooli/`, `Makefile`, `PRD.md`, `README.md`.
- [ ] `cd scenarios/vrooli-emulator/api && go build ./...` exits 0.
- [ ] `cd scenarios/vrooli-emulator/api && go test ./...` exits 0.
- [ ] `cd scenarios/vrooli-emulator/cli && go build ./...` exits 0.
- [ ] `cd scenarios/vrooli-emulator/ui && pnpm install --ignore-workspace && pnpm run build` exits 0.
- [ ] `.vrooli/service.json` validates against the scenarios schema.
- [ ] `grep -r "scenario-to-desktop-api" scenarios/vrooli-emulator/` returns zero matches.
- [ ] `grep -r "findArtifact\|vrooliRoot\|packaging.FindBuiltPackage" scenarios/vrooli-emulator/api/livedesktop/` returns zero matches.
- [ ] `grep -r "/api/v1/livedesktop" scenarios/vrooli-emulator/` returns zero matches.
- [ ] `git diff --name-only` shows only `scenarios/vrooli-emulator/**` paths.
- [ ] Executing agent attempted `vrooli scenario restart vrooli-emulator`. On success, health check confirmed. If the sandbox blocked host lifecycle access, the agent emitted a structured skip note in the completion summary (command attempted, exit code or error) so the reviewer can run the restart + health check manually after merge.
- [ ] Executing agent did not restart the scenario it runs inside.

## 12. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| procmetrics' `xdotool` window detection requires `DISPLAY` — breaks on headless sessions | Medium | Medium | Graceful degrade in `window_detector.go`: return empty windows without error when display unavailable; PID metrics still flow. |
| Headless session response with `vnc_port: 0`/`ws_port: 0` confuses UI clients | Low | Low | UI clients are out of scope for this item. Phase 2 UI hides VNC controls. Unit tests assert response shape. |
| `go.mod` replace directives break when paths move | Low | High | `scenarios/vrooli-emulator/api/` sits at the same depth as `scenarios/scenario-to-desktop/api/` — `../../../packages/<name>` resolves identically. Validate with `go mod tidy` + `go build ./...` at Phase B end. |
| Tests reference deleted symbols after move | Medium | Low | Phase C removes `findArtifact` tests concurrently with code deletion; `go test ./...` surfaces any stragglers. |
| CI/test environment lacks Xvfb, x11vnc, or xdotool | Medium | Medium | All platform and window-detector tests use fakes or `t.Setenv`-driven degradation paths. No test binary invokes a real X11 binary. `.vrooli/service.json` `hostTools` declaration surfaces the runtime dependency for scenario start. |
| Port allocation conflicts with scenario-to-desktop during hard-cut window | Low | Low | Both scenarios declare overlapping ranges (template default); the allocator picks distinct free ports per scenario at start time. |
| Executing agent starts scenarios and damages in-flight state | Low | High | Per feedback memory: do not run `vrooli scenario *` commands against the scenario Claude Code is running inside. Restarting the **target** `vrooli-emulator` (which the agent is NOT inside) is allowed and recommended if the sandbox permits host lifecycle access; otherwise defer to reviewer. |
| Template's `api/main.go` stub conflicts with the full wiring we need | Low | Low | Phase E replaces the stub entirely with scenario-to-desktop's proven main.go pattern adapted to this scenario's domain. |

## 13. Definition of Done

- All rollout checklist items pass.
- New scenario builds and tests pass locally (`go build ./...`, `go test ./...`, `pnpm run build`).
- No changes outside `scenarios/vrooli-emulator/**` (verified by `git diff --name-only`).
- Scenario lifecycle verification: `vrooli scenario restart vrooli-emulator` succeeds and `GET /health` returns 200, OR the agent explicitly flags sandbox restrictions (command attempted + error captured) so the reviewer can run it after merge.
- Sibling items `execute/emulator-acceptance-tests-phase-1` and `chore/scenario-to-desktop-remove-livedesktop` are still pending (confirm via `swarm-manager backlog list --initiative emulator-platform`) — Phase 1 hard-cut bundle releases when all three land together.
