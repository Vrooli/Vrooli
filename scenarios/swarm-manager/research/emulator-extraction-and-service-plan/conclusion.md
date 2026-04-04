# Research Conclusion: Plan The Dedicated Emulator Extraction

## Research Question
How should the livedesktop and related validation capabilities be extracted from scenario-to-desktop into a dedicated vrooli-emulator scenario, and what API/UI/CLI boundaries, artifact launch model, and migration path should be used?

## Summary
The extraction is architecturally sound and low-risk. livedesktop is already self-contained with a single weak dependency (procmetrics). The round 1 decisions establish: unified display management covering both VNC and headless sessions (with headless-ready API from day one), HTTP REST + WebSocket protocol, a standalone UI with iframe-embeddable components, and capture primitives owned by the emulator with validation logic in consumers. The remaining open questions center on the artifact launch contract (how callers provide apps to the emulator), captures persistence ownership split, and migration phase boundaries.

## Methodology
- Read the full scenario-to-desktop codebase (API, UI, CLI) with focus on livedesktop package, smoketest, and their consumers
- Read the completed dependency research (desktop-release-control-plane-audit) for release pipeline context
- Mapped all imports and call sites for livedesktop to determine coupling surface
- Identified two separate display management approaches (livedesktop vs smoketest's xvfb-run)
- Analyzed the PlatformBackend abstraction for extraction readiness
- Deep-dived into artifact launch flow: findArtifact → packaging.FindBuiltPackage → dist-electron path convention
- Analyzed captures system coupling: optional WithCaptures() wiring, graceful degradation pattern
- Investigated iframe/embed patterns in deployment-manager (external-url endpoint + iframe) and app-monitor (@vrooli/iframe-bridge with depth protection)
- Reviewed procmetrics interfaces and /proc + X11 dependencies for portability assessment
- Mapped the full livedesktop API surface (11 HTTP + 1 WS endpoint, 13 control actions)

## Findings

### Finding 1: livedesktop is already well-isolated within scenario-to-desktop

The `livedesktop/` package in `scenarios/scenario-to-desktop/api/` is a self-contained domain with its own handler, service, store, types, platform abstraction, actions, proxy, and janitor. It registers routes via `registerLiveDesktopRoutes()` called from `main.go`. Its only inbound dependency from the rest of scenario-to-desktop is the `procmetrics` package (process monitoring). No other scenario-to-desktop domain (pipeline, build, deploy, etc.) imports livedesktop.

**Key files:**
- `api/livedesktop/platform.go` — `PlatformBackend` interface (already designed for local + remote backends)
- `api/livedesktop/platform_linux.go` — Linux/Xvfb implementation
- `api/livedesktop/service.go` — Session lifecycle management
- `api/livedesktop/handler.go` — HTTP endpoints (sessions, heartbeat, launch, control, ws, metrics, files)
- `api/livedesktop/proxy.go` — WebSocket VNC proxy (generic, works for any backend)

### Finding 2: Two independent display management systems exist

scenario-to-desktop has TWO separate ways to create virtual displays:
1. **livedesktop** — Full session management with Xvfb + x11vnc + websockify, used for interactive desktop viewing via noVNC
2. **smoketest** — Uses `xvfb-run` command directly for headless Electron testing, completely independent of livedesktop

This duplication is a natural extraction boundary: vrooli-emulator will unify both under a single display management service, per the round 1 decision to design for both use cases.

### Finding 3: PlatformBackend is designed for multi-platform and remote operation

The `PlatformBackend` interface (`platform.go`) already documents a forward-looking design:
- Local backends: `linux-xvfb` (implemented), future `windows-qemu`, `android-emulator`
- Remote backends: future `remote-macos`, `remote-ios` (for Apple hardware compliance)
- Design constraints already align with service extraction: URLs over byte blobs, stateless callers, backend-owned session state

### Finding 4: UI components are tightly coupled to the livedesktop API client

The UI lives in `scenarios/scenario-to-desktop/ui/src/components/livedesktop/` with:
- `LiveDesktopDrawer.tsx` — Main drawer component
- `VncCanvas.tsx` — noVNC integration
- `DesktopControlsMenu.tsx` — Rich control panel (launch, screenshot, recording, network sim, env vars, clipboard, dark mode, locale, resize)
- `MetricsBar.tsx`, `DesktopToolbar.tsx`, `PlatformSelector.tsx`

State management via Zustand store (`liveDesktopStore.ts`) and API client (`lib/api/livedesktop.ts`). Per the round 1 decision (d3), these move to the emulator as a standalone UI with embeddable iframe components, following the patterns used by app-monitor and deployment-manager.

### Finding 5: No other scenarios currently consume livedesktop

A search across prompt-manager and deployment-manager found zero references to livedesktop. The only consumer is scenario-to-desktop itself. However, deployment-manager's visual validation system could benefit from emulator sessions for screenshot comparison workflows.

### Finding 6: The smoketest system will become a key emulator consumer

Smoketest currently manages its own display (xvfb-run) and process lifecycle. After extraction, smoketest becomes an emulator client: call the emulator to create a headless session, receive a DISPLAY value (e.g., `:100`), pass it as an env var to the test command. The headless session skips VNC/websockify setup — just Xvfb allocation and lifecycle management.

### Finding 7: The artifact launch model has a cross-scenario knowledge problem

`findArtifact()` in `service.go:289-301` discovers built Electron apps by looking at `scenarios/{scenarioName}/platforms/electron/dist-electron` using `packaging.FindBuiltPackage(distPath, platform)`. After extraction, the emulator cannot assume filesystem knowledge of other scenarios' build outputs. The session creation API accepts an optional `app_path` — if provided, it launches directly. If omitted, it calls `findArtifact(scenarioName)`. This discovery logic is scenario-to-desktop specific and should not move to the emulator. The emulator's launch contract should require an explicit path or accept a pluggable resolver.

### Finding 8: Captures integration uses an optional dependency pattern ready for extraction

The captures system (`api/captures/`) integrates via `liveDesktopService.WithCaptures(capturesService)`. Screenshot and recording actions check for captures availability and gracefully degrade to session-scoped temp files if absent. This pattern maps cleanly to the emulator: capture creation (screenshot, record) is a core emulator action; capture persistence is an optional callback or consumer responsibility. The emulator returns capture data (file path or stream); consumers decide where to persist it.

### Finding 9: Iframe embed pattern is well-established in the codebase

Two proven patterns exist:
1. **deployment-manager**: Backend endpoint `GET /embedded/{name}/external-url` returns the URL; UI renders in `<iframe>` with query params for context
2. **app-monitor**: Uses `@vrooli/iframe-bridge` for bidirectional parent-child communication, with recursive embedding protection (max depth = 1)

The emulator UI should combine both: expose `/embedded/emulator/external-url`, support query params (`session_id`, display config), and integrate iframe-bridge for events (session state changes, control commands from parent).

### Finding 10: Smoketest delegation path is straightforward

Smoketest's display management is a thin wrapper: `RequiresHeadlessWrapper()` checks for DISPLAY env var; if absent, prefixes the command with `xvfb-run -a`. To delegate: (1) call emulator API to create a headless session, (2) receive DISPLAY value, (3) set as env var for the test command, (4) destroy session on completion. The emulator API already supports this — `StartSession` creates an Xvfb display. Adding a `headless: true` flag to skip VNC/websockify is the only new parameter needed.

### Finding 11: The full API surface is mature and well-defined

11 HTTP endpoints + 1 WebSocket:
- **Session CRUD**: POST/GET/GET/DELETE `/sessions[/{id}]`
- **Session actions**: POST heartbeat, POST launch, GET artifact, POST control, GET metrics, GET files
- **Real-time**: WS `/sessions/{id}/ws` (VNC proxy)
- **13 control actions**: launch_app, quit_app, screenshot, start_recording, stop_recording, offline_mode, slow_connection, inject_env, resize_display, clipboard_read, clipboard_write, dark_mode, locale

This API can transfer to the emulator with minimal changes — route prefix update from `/api/v1/livedesktop/` to `/api/v1/emulator/` (or similar), and removing `findArtifact` in favor of caller-provided paths.

## Limitations
- No runtime validation of the proposed extraction — all analysis is static code review
- The impact on scenario-to-desktop's binary size (removing livedesktop + procmetrics) has not been measured
- Remote backend design (documented in PlatformBackend comments) is future work — extraction design should not over-engineer for it
- The iframe-bridge integration specifics (which events, what protocol version) need to be defined during implementation
- Smoketest migration timing relative to the livedesktop extraction depends on test suite stability, which hasn't been assessed
- procmetrics' X11 dependency (xdotool for window detection) assumes a display server is available — headless sessions may need a stub or alternate detection

## Actions
<!-- TBD — will be defined after round 2 decisions are resolved (artifact launch model, captures split, migration phasing) -->
