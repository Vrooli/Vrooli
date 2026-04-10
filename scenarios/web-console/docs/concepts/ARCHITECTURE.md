# Architecture

Web Console is a browser-based terminal that connects to PTY processes on the host via WebSocket. It is designed for single-operator use on a personal Vrooli server.

It now has two distinct data planes:

1. Terminal I/O for raw PTY fidelity.
2. Conversation events for semantic assistant-response features such as auto-TTS, unread counts, and messages view.

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
│  ┌──────────────────────────────────────────┐   │
│  │  useConversationSession                  │   │
│  │  [CODE: ui/src/hooks/useConversationSession.ts] │
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
│  ┌──────────────────────────────────────────┐   │
│  │ conversation_store.go                    │   │
│  │ conversation_router.go                   │   │
│  │ conversation_handlers.go                 │   │
│  └──────────────────────────────────────────┘   │
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
   - **Output forwarder**: PTY → `Session.broadcast()` → WebSocket client (also forwards `sync_warning` when frame coalescing crosses the configured threshold)
   - **Input loop**: WebSocket → `Session.Write()` → PTY stdin
4. Client-side hook [CODE: ui/src/hooks/useTerminalSocket.ts#useTerminalSocket] handles message dispatch
5. `readLoop` splits PTY output at UTF-8 codepoint boundaries so that partial multi-byte sequences are buffered across reads, preventing JSON encoding corruption
6. When a client's output channel is full, frames are **coalesced** (merged into a pending buffer) rather than dropped. The pending buffer is capped at `OfflineBufferMax` and trimmed at ANSI-clean boundaries when exceeded, with an SGR reset prefix to clear dangling color state. The forwarder calls `FlushPending` after each successful WebSocket write to drain coalesced data in 64 KB chunks (matching `Subscribe`'s chunking) to prevent browser UI freezes. After a trimmed buffer is fully flushed, `FlushPending` triggers SIGWINCH (via `pty.SetSize`) so the shell redraws its screen, recovering structural state (cursor position, scroll region, alternate screen) lost during the trim
7. Goroutine lifecycle uses `context.WithCancel`: the input loop's exit cancels the context, which the output forwarder selects on — no goroutine leaks on WebSocket disconnect

### Resize Strategy

The PTY dimensions follow a **last-writer-wins** model: whichever client sends a resize message last sets the PTY size. This keeps the resize path simple and predictable.

- `Subscribe()` registers a client for output broadcast
- `Resize(cols, rows)` sets the PTY size directly
- `Unsubscribe(ch)` removes the client without altering the PTY size

### History Replay Limitations

On reconnect, the client resets the terminal (`terminal.reset()`) before history replay arrives, ensuring a clean slate with no duplicated content. The server replays buffered output history prefixed with an SGR reset (`ESC[0m`) to clear dangling color/attribute state. For large history buffers, the replay is chunked into 64 KB pieces to prevent browser UI freezes — each chunk is sent as a separate WebSocket message so xterm.js can render incrementally.

This does **not** restore cursor position, scroll regions (DECSTBM), alternate screen buffer (smcup/rmcup), or character set state. Full terminal state restoration would require a server-side terminal emulator, which adds significant complexity for marginal benefit in the single-operator use case.

### Terminal History Caching

**Problem**: On page refresh, the server re-sends the entire output history over WebSocket even when the client already has most of it. For long-running sessions this causes noticeable delay and redundant bandwidth.

**Two-layer strategy**:

1. **Client cache**: xterm `SerializeAddon` serializes terminal state to `sessionStorage` on `visibilitychange` (tab hidden) and `beforeunload`.
2. **Server resume**: Client sends `?history_offset=N` on WS connect; server validates the offset against `totalOutputBytes` and sends only the delta.

**Flow**:

```
Page Load → Check sessionStorage
├─ Cache hit → Deserialize to xterm (instant) → Connect WS with ?history_offset=N
│              └─ Server validates offset
│                 ├─ Valid → Send delta + history_end{resumed:true}
│                 └─ Invalid → Send full history + history_end{resumed:false}
│                              └─ Client calls terminal.reset() first
└─ Cache miss → Connect WS without offset → Full history replay (existing behavior)
```

**Cache lifecycle**: Saved on `visibilitychange` (tab hidden) and `beforeunload`. 30-minute TTL. 2 MB max size. Uses `sessionStorage` (per-tab, cleared on tab close).

**Key files**: `ui/src/lib/terminalCache.ts` (save/load/clear), `ui/src/hooks/useTerminalSocket.ts` (historyOffset negotiation), `ui/src/components/TerminalPane.tsx` (SerializeAddon integration), `api/session.go` (totalOutputBytes, Subscribe), `api/terminal_ws.go` (history_offset query param).

### Voice Input

1. User presses Alt+Space (desktop) or taps mic button (mobile toolbar)
2. `useVoiceInput` hook in [CODE: ui/src/hooks/useVoiceInput.ts] calls `getUserMedia()` → starts `MediaRecorder`
3. User releases key / taps stop → recording stops
4. Hook checks transcription backend (determined on mount via `GET /api/v1/capabilities`):
   - **Whisper available**: POST audio blob to `/api/v1/voice/transcribe` → API in [CODE: api/voice_transcribe.go] proxies to Whisper (`localhost:8090/asr`)
   - **Whisper unavailable**: Falls back to browser Web Speech API (Chromium only)
   - **Neither available**: Voice input disabled, mic button hidden
5. Transcribed text injected into terminal via existing `sendInput()` path

### Conversation Events

Conversation events are the semantic record of assistant responses. They are intentionally separate from PTY output history.

1. Claude Stop hook or Codex rollout tailer extracts assistant text.
2. The source adapter appends a `ConversationEvent` into [CODE: api/conversation_store.go].
3. The store assigns a monotonic `sequence` within the owning web-console session.
4. The server fan-outs the event over the existing terminal WebSocket as a `conversation_event` message.
5. The browser appends the event into [CODE: ui/src/stores/useConversationStore.ts].
6. UI features consume that store:
   - auto-TTS for active panes
   - unread counts for inactive tabs/panes
   - messages-mode rendering via [CODE: ui/src/components/MessagesPane.tsx]
7. The browser acknowledges progress (`received`, `seen`, `playback_started`, `playback_succeeded`, `playback_failed`) back over WebSocket.
8. The API updates event/cursor state in [CODE: api/conversation_store.go].

### Cursor Semantics

Each session conversation has two independent cursors:

- `lastSeenSequence`: highest conversation event sequence the user has surfaced in the UI
- `lastListenedSequence`: highest assistant event sequence whose TTS playback completed successfully

These cursors drive unread badges, missed-response replay, and future conversation-centric UI features. They are not inferred from PTY output.

### Error Handling

All API errors return structured JSON with `code`, `category`, `recovery`, and `retry` fields. See [Error Semantics](../internal/ERROR-SEMANTICS.md) for the full contract.

Client-side [CODE: ui/src/lib/api.ts#APIError] parses these into typed errors. The [CODE: ui/src/components/ErrorBanner.tsx] component renders recovery hints and retry buttons based on error metadata.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Repository interfaces (`ShortcutStore`, `AIConfigStore`) | Decouples handlers from storage backend; in-memory for tests, SQLite for production |
| SQLite for shortcuts and AI config | User configuration survives restarts; health metrics stay in-memory (ephemeral, high-frequency) |
| Backend registry + tmux PTY | Sessions can use "standard" (raw PTY, lost on restart) or "persistent" (tmux-backed, survives restart). Registry tracks availability. |
| Session metadata persistence (SQLite) | Session metadata (ID, backend, policy, timestamps) is persisted for restart recovery of tmux sessions |
| Startup recovery | On boot, server discovers surviving tmux sessions, matches against persisted metadata, re-registers them, and cleans up orphans |
| In-memory conversation store | Session conversations are runtime state scoped to the scenario process; enough for unread/messages/TTS within a run without overcommitting to persistence too early |
| PTY interface + factory pattern | Enables testing without real shell processes |
| Conversation events separate from terminal history | Semantic assistant-response features must not depend on raw PTY bytes or terminal rendering heuristics |
| Single `exitCh` channel per session | Session signals exit; SessionManager owns cleanup |
| `@vrooli/api-base` for networking | Proxy-correct HTTP/WS routing under parent iframe |
| Config via env vars with clamping | Safe defaults, graceful degradation on bad input |
| Schema initialization on startup | `initSchema()` runs idempotent schema.sql + seed.sql on every boot |

## Integration Points

- **Parent scenarios** embed web-console via iframe; `@vrooli/iframe-bridge` handles `postMessage` coordination
- **`@vrooli/api-base`** resolves API/WS URLs for proxy-correct networking
- **SQLite persists shortcut profiles, workspace layout, and AI provider configuration via `SQLShortcutStore`, `SQLWorkspaceStore`, and `SQLAIConfigStore`

## Code Organization Pattern

The API uses a **hybrid organization** strategy:

- **Cross-cutting infrastructure** (`errors.go`) owns the error catalog, structured error types, and HTTP error-writing helpers. All handler files import from here rather than from each other, making the dependency explicit.
- **Core domain (sessions)** is split by layer: `session.go` (domain logic) + `session_handlers.go` (HTTP handlers). This split is justified by the size and centrality of the session concept — it's the single largest feature.
- **Feature modules (AI, shortcuts, metrics)** are organized by feature: each file owns its domain types, validation, store, and HTTP handlers together. This keeps related code co-located and makes each feature self-contained.
- **AI generation** (`ai_generate.go`) owns the full generation pipeline — providers, prompt building, extraction, and the config-aware orchestrator (`generateWithConfig`). The companion `ai_provider_config.go` owns only config storage, health tracking, and config HTTP endpoints.
- **Policy handlers** are co-located with session handlers in `session_handlers.go` because they operate on session sub-resource endpoints (`/sessions/{id}/policy`). Policy domain logic (types, validation, TTL, sweeper) lives in `session_policy.go`.

The UI uses **component-per-file** with hooks extracted into `hooks/`, constants into `consts/`, and utilities into `lib/`. The app now ships as a single workspace surface with feature-local section modules under `components/settings/` for the unified settings experience. Shared domain constants (policy options, shortcuts, toolbar keys) live in `consts/` and are imported by multiple components to avoid duplication.

## File Map

| File | Responsibility |
|------|---------------|
| `api/main.go` | Server wiring, routes, middleware, health checks |
| `api/conversation_store.go` | In-memory conversation event store, cursor tracking, playback state |
| `api/conversation_router.go` | Conversation-event append/orchestration entry point for AI response sources |
| `api/conversation_handlers.go` | Conversation REST handlers (`GET /conversation`, `PUT /conversation/cursor`) |
| `api/session.go` | Session + SessionManager domain logic, UTF-8 boundary buffering, frame coalescing |
| `api/pty.go` | PTY interface and factory (testability seam) |
| `api/errors.go` | Error catalog, structured error types, HTTP error-writing helpers |
| `api/session_handlers.go` | All session HTTP handlers (CRUD + policy) |
| `api/session_policy.go` | Expiration policy domain logic: validation, TTL, sweeper |
| `api/terminal_ws.go` | WebSocket upgrade + bidirectional I/O bridge (context-based goroutine lifecycle) |
| `api/config.go` | Environment-based configuration with validation |
| `api/ai_generate.go` | AI command generation: provider chain, prompt building, extraction, config-aware orchestration |
| `api/repository.go` | Storage interfaces: `ShortcutStore`, `AIConfigStore` |
| `api/ai_provider_config.go` | In-memory AI provider config store + health tracking |
| `api/ai_provider_config_sql.go` | SQLite AI provider config store (hybrid: config in SQLite, health in-memory) |
| `api/shortcut_profiles.go` | In-memory shortcut profile store + domain types |
| `api/shortcut_profiles_sql.go` | SQLite shortcut profile store |
| `api/events.go` | Structured event logging (session lifecycle, AI) |
| `api/metrics.go` | Operational metrics collection + metrics HTTP handler |
| `ui/src/App.tsx` | Entry point — health check gate + workspace shell |
| `ui/src/components/Workspace.tsx` | Pane grid layout |
| `ui/src/components/SettingsModal.tsx` | Unified responsive settings shell (desktop modal, mobile drawer) |
| `ui/src/components/settings/SessionManagementSection.tsx` | Sessions tab: policy controls, ordering, focus, terminate |
| `ui/src/components/settings/VoiceInputSection.tsx` | Voice input tab |
| `ui/src/components/settings/TtsSettingsSection.tsx` | Voice output tab |
| `ui/src/components/settings/ShortcutProfilesSection.tsx` | Shortcut profiles tab |
| `ui/src/components/settings/NewPaneDefaultsSection.tsx` | New pane appearance defaults tab |
| `ui/src/components/settings/IntegrationsSection.tsx` | Integrations tab |
| `ui/src/hooks/useSessionManager.ts` | Session lifecycle orchestration |
| `ui/src/hooks/useConversationSession.ts` | Conversation hydration + cursor persistence |
| `ui/src/hooks/useTerminalSocket.ts` | WebSocket protocol handling |
| `ui/src/hooks/useMediaQuery.ts` | Responsive settings shell behavior |
| `ui/src/hooks/useCountdown.ts` | Policy countdown timer (shared by SessionDrawer + SessionsPage) |
| `ui/src/components/TerminalPane.tsx` | xterm.js rendering |
| `ui/src/components/MessagesPane.tsx` | Semantic messages-mode rendering for a single session |
| `ui/src/components/TerminalLauncher.tsx` | New-terminal modal with shortcuts |
| `ui/src/components/MobileToolbar.tsx` | Floating keyboard toolbar |
| `ui/src/components/AiInput.tsx` | AI command input with generate/execute flow |
| `ui/src/components/IntegrationsPanel.tsx` | Dependency health status display (all resources/scenarios) |
| `ui/src/components/ErrorBanner.tsx` | Structured error display |
| `ui/src/components/ErrorBoundary.tsx` | React error boundary with region labels |
| `ui/src/lib/api.ts` | HTTP/WS client functions |
| `ui/src/stores/useConversationStore.ts` | Client-side conversation event/cursor/view-mode state |
| `ui/src/lib/format.ts` | Display formatting utilities |
| `ui/src/lib/localEcho.ts` | Predictive local echo with ANSI-aware graceful degradation |
| `ui/src/lib/ansi.ts` | ANSI escape sequence constants |
| `ui/src/consts/config.ts` | UI tunable constants |
| `ui/src/consts/shortcuts.ts` | Launch shortcut definitions |
| `ui/src/consts/toolbar-keys.ts` | Mobile toolbar key definitions |
| `ui/src/consts/policy-options.ts` | Expiration policy option definitions (shared by settings sections) |
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
| OT-P0-008 | Sidebar/Drawer Controls Surface | [CODE: ui/src/components/SettingsModal.tsx], [CODE: ui/src/components/settings/SessionManagementSection.tsx] | [DOC: docs/internal/SEAMS.md#1-entry--presentation] |

### P1 – Should Have

| Target | Description | Implementation | Docs |
|--------|-------------|----------------|------|
| OT-P1-001 | Session Policy Controls | [CODE: api/session_policy.go], [CODE: ui/src/consts/policy-options.ts], [CODE: ui/src/hooks/useCountdown.ts] | [DOC: docs/concepts/GLOSSARY.md#policy] |
| OT-P1-002 | Shortcut Profile Management | [CODE: api/shortcut_profiles.go], [CODE: api/shortcut_profiles_pg.go], [CODE: ui/src/components/settings/ShortcutProfilesSection.tsx] | [DOC: docs/internal/STORAGE_AUDIT.md] |
| OT-P1-003 | AI Provider Policy Controls | [CODE: api/ai_provider_config.go], [CODE: api/ai_provider_config_pg.go], [CODE: ui/src/components/IntegrationsPanel.tsx] | [DOC: docs/internal/STORAGE_AUDIT.md] |
| OT-P1-004 | Operational Observability Coverage | [CODE: api/metrics.go], [CODE: api/events.go] | [DOC: docs/internal/SEAMS.md#6-cross-cutting] |

### P2 – Future

| Target | Description | Implementation | Docs |
|--------|-------------|----------------|------|
| OT-P2-001 | Collaborative Session Modes | Not yet implemented — requires multi-subscriber session model and auth | [DOC: docs/internal/PROBLEMS.md] |
| OT-P2-002 | Persisted Workspace Presets | Not yet implemented — requires workspace state serialization | [DOC: docs/internal/PROBLEMS.md] |
