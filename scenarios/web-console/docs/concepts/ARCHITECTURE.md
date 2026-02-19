# Architecture

Web Console is a browser-based terminal that connects to PTY processes on the host via WebSocket. It is designed for single-operator use on a personal Vrooli server.

## System Layers

```
┌─────────────────────────────────────────────────┐
│  Browser UI (Vite + React + xterm.js)           │
│  [CODE: ui/src/App.tsx]                         │
│                                                 │
│  ┌──────────┐  ┌───────────┐  ┌──────────────┐ │
│  │Workspace │  │ Terminal   │  │ Mobile       │ │
│  │  Layout   │  │  Launcher  │  │  Toolbar     │ │
│  └──────────┘  └───────────┘  └──────────────┘ │
│       │                                         │
│  ┌──────────────────────────────────────────┐   │
│  │  useSessionManager (session lifecycle)    │   │
│  │  [CODE: ui/src/hooks/useSessionManager.ts]│   │
│  └──────────────────────────────────────────┘   │
│       │ HTTP (REST)          │ WebSocket         │
├───────┼──────────────────────┼───────────────────┤
│  Go API                                         │
│  [CODE: api/main.go]                            │
│                                                 │
│  ┌──────────────┐  ┌────────────────────────┐   │
│  │ session_     │  │ terminal_ws.go         │   │
│  │ handlers.go  │  │ (WS I/O bridge)        │   │
│  │ (REST CRUD)  │  │                        │   │
│  └──────────────┘  └────────────────────────┘   │
│       │                    │                     │
│  ┌──────────────────────────────────────────┐   │
│  │  SessionManager + Session                 │   │
│  │  [CODE: api/session.go]                   │   │
│  │  PTY interface [CODE: api/pty.go]         │   │
│  └──────────────────────────────────────────┘   │
│       │                                         │
│  ┌──────────┐                                   │
│  │  Config   │  [CODE: api/config.go]           │
│  └──────────┘                                   │
└─────────────────────────────────────────────────┘
```

## Data Flow

### Session Creation

1. User clicks "New Terminal" in [CODE: ui/src/components/Workspace.tsx]
2. Launcher presents options in [CODE: ui/src/components/TerminalLauncher.tsx]
3. `useSessionManager` calls `POST /api/v1/sessions` via [CODE: ui/src/lib/api.ts#createSession]
4. API handler in [CODE: api/session_handlers.go#handleCreateSession] delegates to `SessionManager.Create()`
5. `SessionManager` uses `PTYFactory` to spawn a shell process via [CODE: api/pty.go#defaultPTYFactory]
6. Session starts `readLoop()` goroutine for PTY output fan-out

### Terminal I/O

1. UI opens WebSocket to `/api/v1/sessions/{id}/ws`
2. Server upgrades connection in [CODE: api/terminal_ws.go#handleTerminalWS]
3. Two concurrent loops bridge browser ↔ PTY:
   - **Output forwarder**: PTY → `Session.broadcast()` → WebSocket client
   - **Input loop**: WebSocket → `Session.Write()` → PTY stdin
4. Client-side hook [CODE: ui/src/hooks/useTerminalSocket.ts#useTerminalSocket] handles message dispatch

### Error Handling

All API errors return structured JSON with `code`, `category`, `recovery`, and `retry` fields. See [Error Semantics](../internal/ERROR_SEMANTICS.md) for the full contract.

Client-side [CODE: ui/src/lib/api.ts#APIError] parses these into typed errors. The [CODE: ui/src/components/ErrorBanner.tsx] component renders recovery hints and retry buttons based on error metadata.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| In-memory sessions (no SQLite yet) | Simplifies MVP; persistence is a P1 target |
| PTY interface + factory pattern | Enables testing without real shell processes |
| Single `exitCh` channel per session | Session signals exit; SessionManager owns cleanup |
| `@vrooli/api-base` for networking | Proxy-correct HTTP/WS routing under parent iframe |
| Config via env vars with clamping | Safe defaults, graceful degradation on bad input |

## Integration Points

- **Parent scenarios** embed web-console via iframe; `@vrooli/iframe-bridge` handles `postMessage` coordination
- **`@vrooli/api-base`** resolves API/WS URLs for proxy-correct networking
- **Postgres** is the configured resource dependency (for future session persistence)

## Code Organization Pattern

The API uses a **hybrid organization** strategy:

- **Cross-cutting infrastructure** (`errors.go`) owns the error catalog, structured error types, and HTTP error-writing helpers. All handler files import from here rather than from each other, making the dependency explicit.
- **Core domain (sessions)** is split by layer: `session.go` (domain logic) + `session_handlers.go` (HTTP handlers). This split is justified by the size and centrality of the session concept — it's the single largest feature.
- **Feature modules (AI, shortcuts, metrics)** are organized by feature: each file owns its domain types, validation, store, and HTTP handlers together. This keeps related code co-located and makes each feature self-contained.
- **AI generation** (`ai_generate.go`) owns the full generation pipeline — providers, prompt building, extraction, and the config-aware orchestrator (`generateWithConfig`). The companion `ai_provider_config.go` owns only config storage, health tracking, and config HTTP endpoints.
- **Policy handlers** are co-located with session handlers in `session_handlers.go` because they operate on session sub-resource endpoints (`/sessions/{id}/policy`). Policy domain logic (types, validation, TTL, sweeper) lives in `session_policy.go`.

The UI uses **component-per-file** with hooks extracted into `hooks/`, constants into `consts/`, and utilities into `lib/`. Pages (`pages/`) represent top-level routes; components (`components/`) are reusable building blocks. Shared domain constants (policy options, shortcuts, toolbar keys) live in `consts/` and are imported by multiple components to avoid duplication.

## File Map

| File | Responsibility |
|------|---------------|
| `api/main.go` | Server wiring, routes, middleware, health checks |
| `api/session.go` | Session + SessionManager domain logic |
| `api/pty.go` | PTY interface and factory (testability seam) |
| `api/errors.go` | Error catalog, structured error types, HTTP error-writing helpers |
| `api/session_handlers.go` | All session HTTP handlers (CRUD + policy) |
| `api/session_policy.go` | Expiration policy domain logic: validation, TTL, sweeper |
| `api/terminal_ws.go` | WebSocket upgrade + bidirectional I/O bridge |
| `api/config.go` | Environment-based configuration with validation |
| `api/ai_generate.go` | AI command generation: provider chain, prompt building, extraction, config-aware orchestration |
| `api/ai_provider_config.go` | AI provider config store, health tracking, config HTTP handlers |
| `api/shortcut_profiles.go` | Shortcut profile store, validation, profile HTTP handlers |
| `api/events.go` | Structured event logging (session lifecycle, AI) |
| `api/metrics.go` | Operational metrics collection + metrics HTTP handler |
| `ui/src/App.tsx` | Entry point — health check gate + hash routing |
| `ui/src/pages/SessionsPage.tsx` | Session list with policy controls |
| `ui/src/pages/SettingsPage.tsx` | Shortcut profiles + AI provider settings |
| `ui/src/components/Workspace.tsx` | Pane grid layout |
| `ui/src/hooks/useSessionManager.ts` | Session lifecycle orchestration |
| `ui/src/hooks/useTerminalSocket.ts` | WebSocket protocol handling |
| `ui/src/hooks/useHashRoute.ts` | Hash-based page routing |
| `ui/src/hooks/useCountdown.ts` | Policy countdown timer (shared by SessionDrawer + SessionsPage) |
| `ui/src/components/TerminalPane.tsx` | xterm.js rendering |
| `ui/src/components/TerminalLauncher.tsx` | New-terminal modal with shortcuts |
| `ui/src/components/MobileToolbar.tsx` | Floating keyboard toolbar |
| `ui/src/components/SessionDrawer.tsx` | Session list sidebar |
| `ui/src/components/AiInput.tsx` | AI command input with generate/execute flow |
| `ui/src/components/ProviderHealthPanel.tsx` | AI provider health status display |
| `ui/src/components/ErrorBanner.tsx` | Structured error display |
| `ui/src/components/ErrorBoundary.tsx` | React error boundary with region labels |
| `ui/src/lib/api.ts` | HTTP/WS client functions |
| `ui/src/lib/format.ts` | Display formatting utilities |
| `ui/src/lib/ansi.ts` | ANSI escape sequence constants |
| `ui/src/consts/config.ts` | UI tunable constants |
| `ui/src/consts/shortcuts.ts` | Launch shortcut definitions |
| `ui/src/consts/toolbar-keys.ts` | Mobile toolbar key definitions |
| `ui/src/consts/policy-options.ts` | Expiration policy option definitions (shared by SessionDrawer + SessionsPage) |
| `ui/src/consts/selectors.ts` | Test automation selector registry |
