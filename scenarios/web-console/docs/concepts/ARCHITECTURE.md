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
   - **Output forwarder**: PTY → `Session.broadcast()` → WebSocket client (also forwards `sync_warning` on coalesce thresholds and `pty_state` on alt-buffer transitions)
   - **Input loop**: WebSocket → `Session.Write()` → PTY stdin
4. Client-side session hook [CODE: ui/src/hooks/terminal/useTerminalSession.ts#useTerminalSession] composes three focused hooks: `useTerminalTransport` (WebSocket lifecycle + `wsGen` counter), `useStdinAck` (seq/ack protocol + pending queue with wsGen write barrier), and a shared `TerminalInputGate` ([CODE: ui/src/components/terminal/inputGate.ts]) that is the single path every input source (xterm.onData, MobileToolbar, paste, voice, upload) flows through
5. `readLoop` splits PTY output at UTF-8 codepoint boundaries so that partial multi-byte sequences are buffered across reads, preventing JSON encoding corruption
6. When a client's output channel is full, frames are **coalesced** (merged into a per-client pending buffer) rather than dropped. The pending buffer is capped at a fixed `pendingBufferMax`; on overflow the oldest bytes are truncated and the next snapshot replay restores correct state. The forwarder calls `FlushPending` after each successful WebSocket write to drain coalesced data in 64 KB chunks to prevent browser UI freezes
7. After a trimmed buffer is fully flushed, the session considers a SIGWINCH-based screen recovery. Recovery is **gated** by [CODE: api/session.go#maybeSIGWINCHRecovery]: it is suppressed while the PTY is in the alternate screen buffer (tracked by [CODE: api/pty_state.go#PTYStateTracker] parsing `\x1b[?1049h`/`1047`/`47` sequences) and rate-limited to at most one SIGWINCH per `SIGWINCHCooldownMs` (default 1000ms). This prevents the tmux status-bar interleaving that occurred under earlier unconditional recovery
8. Alt-buffer state is broadcast to clients as `pty_state` messages. The session hook disables the `LocalEchoController` while in alt-buffer so predictive echo does not flicker under TUI redraws
9. Goroutine lifecycle uses `context.WithCancel`: the input loop's exit cancels the context, which the output forwarder selects on — no goroutine leaks on WebSocket disconnect

### Resize Strategy

One shared pseudo-terminal has one physical winsize, so Web Console uses a
**size lease**, not last-writer-wins. Each connection may declare its preferred
grid, but only the lease holder can apply a declared size to the PTY. The first
viewer receives the lease; it moves on explicit Take over, on follower input,
or to the oldest remaining viewer when the leader disconnects. `size_info`
broadcasts the authoritative grid and leader state to every viewer.

Followers render that one grid faithfully rather than pretending to reflow it:
`fitGrid` computes its fitted viewport, `archetypeForGrid` derives a geometric
frame from grid aspect ratio, and `DeviceFrame` provides the labelled Take over
affordance. A follower never has an independent terminal grid.

### Terminal Snapshot Replay

**Source of truth**: a server-side terminal emulator (`api/terminal/`) decodes the PTY byte stream into a screen grid + alt-buffer flag + bounded scrollback ring. The emulator is the durable representation; raw PTY bytes are not retained.

**Wire flow on every WS open** (fresh OR reconnect):

```
Server: Subscribe()                          ; locks emulator, captures Snapshot()
Server → stdout {data: <snapshot bytes>}…   ; chunked at 64 KB
Server → history_end {}                      ; pure delimiter, no fields
Server → live stdout / stdin_ack / etc.
```

The snapshot is a self-contained ANSI byte stream that recreates the exact `(screen, alt-buffer, scrollback)` triple in any conforming xterm-compatible client:

1. `\x1bc` — full reset.
2. Each scrollback line, oldest first, terminated by `\r\n` so the receiver naturally scrolls them into its own scrollback.
3. Visible normal-buffer rows; the last row has no trailing `\r\n` so the cursor doesn't trigger a final scroll.
4. If the server is in alt-buffer: `\x1b[?1049h` followed by alt-buffer rows.
5. CUP to current cursor + SGR matching the current pen.

**Client**: xterm.js is a pure renderer. On every WS open the hook calls `terminal.reset()`, writes every snapshot stdout frame verbatim, then on `history_end` flips to live mode and writes subsequent stdout frames as live PTY output. There is no client-side cache, no byte-offset accounting, and no duplication-detection logic — every reconnect rebuilds state from the snapshot.

**Why the rewrite happened**: the previous raw-PTY-byte history ring was not replay-safe across alt-buffer transitions. A captured stream containing an unmatched `\x1b[?1049h` would leave the reconnecting xterm stuck in alt-buffer (where scrollback is disabled by VT spec), making history appear to vanish.

**Invariants** (enforced by tests in `api/terminal/emulator_test.go`):

- `Feed` is total: never errors, consumes every byte; malformed escapes are dropped.
- `Snapshot()` is idempotent under no input.
- `Snapshot()` is complete: feeding the snapshot bytes into a fresh `Emulator` reproduces the same triple.
- Alt-buffer is opaque to scrollback — bytes written while `InAltBuffer()` returns true never enter the scrollback ring.
- `Resize` preserves scrollback line count.

**Key files**: `api/terminal/` (emulator + snapshot serializer), `api/session.go` (`Subscribe()` returns the snapshot), `api/terminal_ws.go` (chunks snapshot frames before `history_end`), `ui/src/hooks/terminal/useTerminalSession.ts` (snapshot-mode flag), `ui/src/lib/terminalConfig.ts` (`TERMINAL_SCROLLBACK_LINES` shared constant).

### Voice Input

1. User presses Alt+Space (desktop) or taps the mic button.
2. Web-console checks the `audio-tools` capability from [CODE: api/internal/capabilities/registry.go].
3. When `audio-tools` is available, the UI and API use the audio-tools adoption boundary (`audioports.Remote*` and `@audio-tools/embed`) for STT, streaming, speaker verification, TTS, summarization, and provider routing.
4. When `audio-tools` is unavailable, the terminal workspace still boots and audio features degrade in place. Capability refreshes re-enable audio paths after audio-tools returns; web-console does not run raw Whisper/Kokoro probes or start provider resources directly.
5. Transcribed text is injected into the terminal through the existing input gate.

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

### Message Export (client-local)

Operators can turn an explicit subset of conversation events into paste-ready coding-agent context. The flow is entirely client-local — no export endpoint, no dependency, and conversation text never leaves the browser for formatting:

1. The Message Navigator ([CODE: ui/src/components/MessageJumpList.tsx]) exposes an always-visible Export header action in normal jump mode. Activating it enters an explicit selection mode where result rows toggle checkboxes instead of jumping; jump and playback-select semantics are unchanged.
2. Selected event IDs live in [CODE: ui/src/components/MessagesPane.tsx] — the single session-scoped source of truth shared by the navigator and the drawer. Selection survives filter/sort/query changes (hidden selections stay counted) and is normalized against available events whenever the conversation refreshes.
3. Continue opens [CODE: ui/src/components/MessageExportDrawer.tsx] (built on the shared [CODE: ui/src/components/DrawerShell.tsx]), which is a formatter/preview/copy surface only — never a second selector.
4. Deterministic ordering, escaping, rendering (Agent XML, Markdown transcript, quote blocks, plain text), and token estimation live in the pure, React-free [CODE: ui/src/lib/messageExport.ts]. Exports always render in ascending `ConversationEvent.sequence` with original roles, exactly the chosen messages, and no invented purpose wrapper.
5. Copy uses the browser Clipboard API with transient success feedback and non-destructive error feedback; closing the drawer returns to the navigator with the selection intact.

Ownership boundary: MessagesPane owns selection state; MessageJumpList owns selection affordances; messageExport.ts owns all formatting; MessageExportDrawer owns format choice, preview, and clipboard interaction.

### File Preview

File preview is a third supporting data plane, separate from terminal I/O and conversation events. It is **not** semantic conversation state — it is a reusable subsystem any web-console surface (today: conversation message links + inline path chips) can call to open a resolved local artifact.

Two transports, by intent:

1. **Metadata + bounded text — Connect-RPC** `FilePreviewService` ([CODE: api/handlers/file_preview]). `Resolve(path)` returns a `PreviewModel`: canonical path, basename, `:line`, classification (`PreviewKind`), MIME type, size, capability flags (`can_preview`/`can_download`/`supports_range`/`text_content_available`), an opaque short-lived `preview_id`, a same-origin `blob_url`, and warnings. `GetTextContent(preview_id)` returns ≤1 MiB UTF-8 for text kinds (markdown/code/text/csv/diff).
2. **Bytes — REST blob/range** `GET|HEAD /api/v1/sessions/{id}/file-previews/{previewId}/blob` ([CODE: api/file_preview_handlers.go]). A sanctioned REST exception (reason `ops_probe`) because byte-range streaming consumed directly by native `<img>/<video>/<audio>/<iframe>` is browser-native and cannot be a Connect call. Binary/media bytes never travel through Connect.

Resolution + classification + the preview-id store live in the transport-neutral [CODE: api/internal/filepreview] package (independently unit-tested, reusable). The UI side is registry-based: a viewer state machine ([CODE: ui/src/components/file-preview/useFilePreviewController.ts]) feeds a normalized `PreviewModel` to [CODE: ui/src/components/MessagesFileViewer.tsx], which routes on `kind` through the renderer registry ([CODE: ui/src/components/file-preview/renderers/index.ts]) — one dedicated renderer per kind (markdown, code/text, image/svg, pdf, audio, video, csv, diff, unsupported).

Flow: message link/chip → `openPreview(path)` → `Resolve` issues a session-bound `preview_id` → text kinds fetch `GetTextContent`; blob kinds load lazily via `blob_href` in the renderer. The blob handler re-stats the file on every request and returns 409 if size/mtime changed since resolve, 404 for unknown/expired/session-mismatched ids.

### Error Handling

All API errors return structured JSON with `code`, `category`, `recovery`, and `retry` fields. See [Error Semantics](../internal/ERROR_SEMANTICS.md) for the full contract.

Client-side [CODE: ui/src/lib/api.ts#APIError] parses these into typed errors. The [CODE: ui/src/components/ErrorBanner.tsx] component renders recovery hints and retry buttons based on error metadata.

## Audio Ownership Map

Web-console owns terminal/conversation glue and the same-origin adoption surface. `audio-tools` owns reusable audio capability behavior: STT, TTS, normalization, summarization, transcoding, provider mechanics, provider lifecycle, and provider recovery.

This section is the current ownership contract. The service manifest declares `audio-tools` as optional with `startup_policy: "try_start"`: lifecycle should try to bring it up, but web-console must not fail terminal boot when it is absent.

### Backend Ownership

| Layer | Today's location | Future ownership | Notes |
|---|---|---|---|
| Reusable STT (transcribe, stream WS, VAD, speaker verification) | `audio-tools` | `audio-tools` | Web-console reaches this through RemoteSpeechToText / embed surfaces only |
| Reusable TTS core (synthesize, voices, normalize, paragraph split, chunk) | `audio-tools` | `audio-tools` | Provider routing and provider lifecycle belong to audio-tools |
| Audio processing (transcode) | `audio-tools` | `audio-tools` | ffmpeg/resampling/codec policy stay with the provider owner |
| Capability ports (consumer-facing interfaces) | `api/internal/audioports/` | **web-console** | Stays as the adoption boundary; future audio-tools client implements these |
| Conversation auto-TTS trigger policy | `api/conversation_router.go`, `api/conversation_store.go` | **web-console** | Decides *when* to ask for audio for which session/event |
| TTS cache (pre-synthesis, event invalidation) | `api/tts_cache.go` | Split — provider-level cache to audio-tools; conversation-event eviction stays in web-console | `eventID`-keyed invalidation is conversation glue; raw audio caching is capability behavior |
| TTS summarization service (cooldown, inflight dedupe) | `audio-tools` via `api/internal/audioports/` | `audio-tools` | Pure summarizer + service; orchestration trigger stays in web-console |
| Hook attribution (Claude Stop hook, Codex tailer → ConversationEvent) | `api/tts_hook_handler.go`, `api/codex_tailer.go` | **web-console** | Per-session attribution is not an audio concern |
| Playback ack / status snapshots | `api/tts_playback.go` | **web-console** | Tied to conversation cursor semantics |
| Connect-RPC transport for audio features | `audio-tools` clients behind `api/internal/audioports/` | Split | Web-console-owned transports are only same-origin proxy/adoption seams, not raw audio provider APIs |

### Frontend Ownership

| Layer | Today's location | Future ownership | Notes |
|---|---|---|---|
| Audio adoption boundary (re-export surface) | `ui/src/domains/audio/` | **web-console** | Single import path orchestration code uses; pointed at in-tree hooks today, swapped to audio-tools client at adoption time |
| Mic capture, VAD, audio context, provider mechanics | `@audio-tools/embed` / copied adoption boundary | `audio-tools` | Reusable across any audio-consuming scenario; web-console keeps only terminal targeting glue |
| `useVoiceInput`, `useTextToSpeech` hook orchestration | `ui/src/hooks/use*.ts` | Split — generic readiness/lifecycle to audio-tools; terminal-input-gate wiring stays | See [`ui/src/domains/audio/README.md`](../../ui/src/domains/audio/README.md) for the per-file classification |
| `tts-playback` controller (listened-cursor, auto-TTS policy) | `ui/src/domains/tts-playback/` | **web-console** | Conversation-cursor state machine is web-console concern |
| Terminal voice command targeting + transcript injection | `ui/src/components/terminal/**`, `VoiceMicButton.tsx` | **web-console** | Routes to active pane through `TerminalInputGate` |
| Settings panels (`VoiceInputSection`, `TtsSettingsSection`) | `ui/src/components/settings/` | Split — generic capability controls to audio-tools; terminal-input integration toggles stay | Today bundled; future "show what audio-tools provides" panel lives in [`IntegrationsPanel`](../../ui/src/components/IntegrationsPanel.tsx) |

### Connected Scenarios Registry

The single source of truth for "which other scenarios does web-console integrate with" is `api/internal/capabilities/registry.go`. Each entry declares:

- `DependencyKind` — `DependencyResource` (local resource) or `DependencyScenario` (another scenario).
- The features it unlocks (used by [CODE: api/internal/capabilities/checkers.go] to gate behavior).
- The probe used to detect availability (`vrooli scenario status <slug>` for scenarios).

Scenario status is decoded from typed `vrooli scenario status <slug> --json` fields. Unavailable states carry reason/action metadata for Settings. When the action is a scenario start/restart, the `RunAction` capability RPC delegates to the Vrooli lifecycle contract (`vrooli scenario start|restart <slug> --json`, then one `vrooli scenario wait <slug> --json`) and only accepts declared `DependencyScenario` entries. Operator-only states render the command without a button. Provider/resource repair remains owned by the scenario that owns the provider, notably audio-tools for Whisper/Kokoro/audio internals.

`audio-tools` is registered today as a `DependencyScenario` with the 9 feature keys it will unlock when shipped. The probe shells out via the Vrooli CLI (per the [wrap-not-use principle](../../../../docs/concepts/principles/wrap-not-use.md)) and returns "not yet available" until the scenario exists.

The Integrations settings tab renders this registry as two grouped subsections: **Connected Scenarios** (other Vrooli scenarios this one depends on) and **Local Resources** (Ollama/OpenRouter and other directly owned local resources). When audio-tools ships, no registry change is required: the existing checker starts returning `available`, and the dependent feature gates flip on automatically.

### Capability Ports — Adoption Seam

[CODE: api/internal/audioports/ports.go] declares `SpeechToText`, `TextToSpeech`, and `SpeechTextProcessor`. Conversation, terminal, and hook orchestration code talks **only** to these ports — never directly to `internal/voice`, `internal/tts`, or Kokoro/Whisper/Ollama clients. The local implementation (in package main wiring) is backed by today's `internal/*` packages; the future audio-tools implementation will be backed by an HTTP/Connect/WebSocket client. Adoption is then a single wiring change.

Greenfield assertion tests in [CODE: api/greenfield_assertions_test.go] lock the rule against regression.

### UI ↔ own-API only (audio_admin + audio_runtime)

The browser only ever talks to web-console's own API. The audio surface
the UI calls is web-console-owned:

- `AudioAdminService` — speaker config + status + profiles, wake-word
  config + template lifecycle, stream-config, TTS config, summarize
  config. Proto schema in
  [`packages/proto/schemas/web-console/v1/audio_admin/`](../../../../packages/proto/schemas/web-console/v1/audio_admin/);
  handler in [CODE: api/handlers/audio_admin/].
- `AudioRuntimeService` — per-utterance Transcribe, Synthesize,
  ListVoices, GetTTSCache, Summarize, RecordPlaybackEvent. Proto
  schema in
  [`packages/proto/schemas/web-console/v1/audio_runtime/`](../../../../packages/proto/schemas/web-console/v1/audio_runtime/);
  handler in [CODE: api/handlers/audio_runtime/].

Both handlers delegate to `internal/audioports` Remote* adapters
(`RemoteSpeakerAdmin`, `RemoteWakeWordAdmin`, `RemoteStreamConfigAdmin`,
`RemoteTTSConfigAdmin`, `RemoteSummarizeConfigAdmin`,
`RemotePlaybackEventRecorder`) that call audio-tools server-side via
Connect-RPC, re-resolving the audio-tools URL on transport failure.

Voice streaming WebSocket: `/api/v1/voice/stream` is a transparent WS
reverse proxy ([CODE: api/voice_stream_proxy.go]) — same browser URL,
web-console-mediated upstream dial to audio-tools. CORS is never
required on audio-tools because the browser never originates a
cross-origin request against it.

The boundary test
[`scenarios/web-console/ui/src/__tests__/audio-boundary.test.ts`](../../ui/src/__tests__/audio-boundary.test.ts)
asserts no UI file imports `@vrooli/proto-types/audio-tools/*` or the
retired `@audio-tools/embed` package.

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
