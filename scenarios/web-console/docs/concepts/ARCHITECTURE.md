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

All API errors return structured JSON with `code`, `category`, `recovery`, and `retry` fields. See [Error Semantics](../internal/ERROR-SEMANTICS.md) for the full contract.

Client-side [CODE: ui/src/lib/api.ts#APIError] parses these into typed errors. The [CODE: ui/src/components/ErrorBanner.tsx] component renders recovery hints and retry buttons based on error metadata.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Repository interfaces (`ShortcutStore`, `AIConfigStore`) | Decouples handlers from storage backend; in-memory for tests, PostgreSQL for production |
| PostgreSQL for shortcuts and AI config | User configuration survives restarts; health metrics stay in-memory (ephemeral, high-frequency) |
| In-memory sessions (PTY process-bound) | PTY state cannot persist across restarts; session metadata persistence is a future target |
| PTY interface + factory pattern | Enables testing without real shell processes |
| Single `exitCh` channel per session | Session signals exit; SessionManager owns cleanup |
| `@vrooli/api-base` for networking | Proxy-correct HTTP/WS routing under parent iframe |
| Config via env vars with clamping | Safe defaults, graceful degradation on bad input |
| Schema initialization on startup | `initSchema()` runs idempotent schema.sql + seed.sql on every boot |

## Integration Points

- **Parent scenarios** embed web-console via iframe; `@vrooli/iframe-bridge` handles `postMessage` coordination
- **`@vrooli/api-base`** resolves API/WS URLs for proxy-correct networking
- **PostgreSQL** persists shortcut profiles and AI provider configuration via `PGShortcutStore` and `PGAIConfigStore`

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
| `api/repository.go` | Storage interfaces: `ShortcutStore`, `AIConfigStore` |
| `api/ai_provider_config.go` | In-memory AI provider config store + health tracking |
| `api/ai_provider_config_pg.go` | PostgreSQL AI provider config store (hybrid: config in PG, health in-memory) |
| `api/shortcut_profiles.go` | In-memory shortcut profile store + domain types |
| `api/shortcut_profiles_pg.go` | PostgreSQL shortcut profile store |
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

## Operational Target Implementation Map

This section maps each PRD operational target to its implementing code and documentation, ensuring bidirectional traceability.

### P0 – Must Ship

| Target | Description | Implementation | Docs |
|--------|-------------|----------------|------|
| OT-P0-001 | Pane-Based Terminal Workspace | [CODE: ui/src/components/Workspace.tsx], [CODE: ui/src/hooks/useSessionManager.ts], [CODE: ui/src/components/TerminalPane.tsx] | [DOC: docs/internal/SEAMS.md#1-entry--presentation] |
| OT-P0-002 | Production-Grade Web Terminal Fidelity | [CODE: api/pty.go], [CODE: api/terminal_ws.go], [CODE: ui/src/hooks/useTerminalSocket.ts] | [DOC: docs/internal/TEMPORAL-FLOWS.md] |
| OT-P0-003 | Durable Session Continuity | [CODE: api/session.go] (offline buffer, reconnect), [CODE: api/session_handlers.go] | [DOC: docs/internal/INVARIANTS.md] |
| OT-P0-004 | Proxy-Correct Networking via api-base | [CODE: ui/src/lib/api.ts], `@vrooli/api-base` integration | [DOC: docs/reference/configuration.md] |
| OT-P0-005 | AI Input with Provider Fallback | [CODE: api/ai_generate.go] (provider chain, fallback), [CODE: ui/src/components/AiInput.tsx] | [DOC: docs/internal/ASSUMPTIONS.md#behavioral-assumptions] |
| OT-P0-006 | New Terminal Launcher with Configurable Shortcuts | [CODE: ui/src/components/TerminalLauncher.tsx], [CODE: ui/src/consts/shortcuts.ts], [CODE: api/shortcut_profiles.go] | [DOC: docs/concepts/GLOSSARY.md#shortcut] |
| OT-P0-007 | Mobile Terminal Usability Toolbar | [CODE: ui/src/components/MobileToolbar.tsx], [CODE: ui/src/consts/toolbar-keys.ts] | [DOC: docs/internal/EXPERIENCE-AUDIT.md] |
| OT-P0-008 | Sidebar/Drawer Controls Surface | [CODE: ui/src/components/SessionDrawer.tsx] | [DOC: docs/internal/SEAMS.md#1-entry--presentation] |

### P1 – Should Have

| Target | Description | Implementation | Docs |
|--------|-------------|----------------|------|
| OT-P1-001 | Session Policy Controls | [CODE: api/session_policy.go], [CODE: ui/src/consts/policy-options.ts], [CODE: ui/src/hooks/useCountdown.ts] | [DOC: docs/concepts/GLOSSARY.md#policy] |
| OT-P1-002 | Shortcut Profile Management | [CODE: api/shortcut_profiles.go], [CODE: api/shortcut_profiles_pg.go], [CODE: ui/src/pages/SettingsPage.tsx] | [DOC: docs/internal/STORAGE_AUDIT.md] |
| OT-P1-003 | AI Provider Policy Controls | [CODE: api/ai_provider_config.go], [CODE: api/ai_provider_config_pg.go], [CODE: ui/src/components/ProviderHealthPanel.tsx] | [DOC: docs/internal/STORAGE_AUDIT.md] |
| OT-P1-004 | Operational Observability Coverage | [CODE: api/metrics.go], [CODE: api/events.go] | [DOC: docs/internal/SEAMS.md#6-cross-cutting] |

### P2 – Future

| Target | Description | Implementation | Docs |
|--------|-------------|----------------|------|
| OT-P2-001 | Collaborative Session Modes | Not yet implemented — requires multi-subscriber session model and auth | [DOC: docs/internal/PROBLEMS.md] |
| OT-P2-002 | Persisted Workspace Presets | Not yet implemented — requires workspace state serialization | [DOC: docs/internal/PROBLEMS.md] |
