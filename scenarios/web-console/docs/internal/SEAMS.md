# Web Console — Seams & Responsibility Boundaries

Last updated: 2026-08-26

> **Audio adoption (2026-05-16):** `internal/voice/`, `internal/tts/`,
> `handlers/voice/`, `handlers/tts/`, and the `web-console/v1/voice` +
> `web-console/v1/tts` protos have all been deleted. Every audio capability
> (STT, TTS synthesis, voice listing, summarization, speaker verification,
> wake word) lives in the **audio-tools** scenario and is consumed via:
>
> - Backend: `internal/audioports.Remote*` adapters (RemoteSpeechToText,
>   RemoteTextToSpeech, RemoteSpeechTextProcessor, RemoteSummarizer)
>   wrapping the audio-tools Connect clients.
> - Frontend: `@vrooli/audio-capture-browser` (pnpm file: link from
>   `packages/audio-capture-browser/`), with Web Console retaining only
>   terminal-targeting and recovery UX.
>
> The web-console-internal `tts_hook_status.go` REST endpoints
> (`/api/v1/tts-hook/{status,config,ack,playback}`) cover only Claude
> project-settings hook routing diagnostics + the auto/backend/startMuted
> preference triple. They are an enumerated REST exception
> (RESTReasonHostHookGlue) and never cross scenario boundaries.
>
> The terminal workspace must boot when audio-tools is absent. Lifecycle may
> try to start audio-tools, and Settings may show lifecycle-owned start/restart
> commands for the scenario, but web-console must not start Whisper, Kokoro, or
> other audio provider resources directly.

## Voice turn diagnostics (2026-07-11)

`PcmVoiceStreamProvider` is the shared same-origin transport adapter;
the shared `StreamDiagnosticRecorder` from `@vrooli/audio-capture-browser`
is the privacy boundary. It records only opaque session identity, protocol
state, retained durability level, coverage cursors, status/error codes, and
terminal reason—never PCM or transcript text. `useVoiceCore` exposes its
sanitized JSON export to every mic surface, and `VoiceMicButton` makes it
available from the failed-turn recovery tooltip. This keeps browser capture
semantics shared while leaving Web Console responsible for its own recovery UX.

## Audio Summarize Model Catalog (added 2026-05-17)

**Owner boundary:** audio-tools owns summarize model policy, catalog metadata,
Ollama `/api/tags` inspection, and persisted summarize config. Web Console
owns only the same-origin admin surface and settings UI.

- Audio-tools API: `SummarizeService.ListSummarizeModels` returns known
  recommended candidates merged with locally installed Ollama models.
- Web-console API: `AudioAdminService.ListSummarizeModels` mirrors the shape
  with web-console-owned proto messages. UI code imports only
  `@vrooli/proto-types/web-console/*`.
- Backend adapter: `api/internal/audioports.RemoteSummarizeConfigAdmin`
  forwards config and catalog calls. `connect.CodeFailedPrecondition` maps to
  `audiotools.ErrFailedPrecondition` so a missing selected model does not look
  like global audio-tools downtime.
- UI hook: `ui/src/components/settings/useSummarizeSettings.ts` owns load,
  save, model-list, and error state for the settings surface.

Key invariant: model install/pull operations are operator actions outside this
seam. The UI may show `ollama pull <model>` but must not run it.

## Input delivery (refactored 2026-04-24)

Terminal input now flows through a kind-discriminated path. See
[TERMINAL-INPUT-PROTOCOL.md](TERMINAL-INPUT-PROTOCOL.md) for the full
contract.

- `PTY.WriteInput(data, kind)` ([CODE: api/pty.go]) replaced the
  legacy `PTY.Write(p) (int, error)`. `realPTY` ignores `kind`;
  `tmuxPTY` routes `keystroke` via `tmux send-keys -l --` and `paste`
  via `tmux load-buffer` + `paste-buffer -d`, both after cancelling
  any active tmux client mode. Closes the "Ctrl+C unblocks lost
  input" Bug A.
- `stdin_ack.reason` carries typed failure codes
  (`tmux_write_failed`, `pty_closed`, `not_ready`, `invalid_input`).
  The UI surfaces the reason in the paste context menu.
- `TerminalContextMenu` waits for settlement via
  `subscribeInputSettled` before closing, showing
  `Pasting… → Pasted` or `Paste failed: <reason>`. Closes Bug B.
- Terminal replay uses a server-side emulator and a self-contained ANSI
  snapshot (`api/terminal/`). Each live stdout frame also carries an
  `output_cursor`; the session retains an 8 MiB frame-boundary ring. A
  reconnect sends `hello{want_resume,rendered_through}` and receives only the
  covered delta, or an explicit `resync` followed by the authoritative
  snapshot when the cursor has expired. `history_end.output_cursor` records
  the renderer's new checkpoint.

## The handoff seam (added 2026-08-27)

One generic verb moves a message from one session to another inside a group.
Design record: [ROLES-AND-HANDOFFS-UX.md](ROLES-AND-HANDOFFS-UX.md).

**The seam is `submitToActiveTerminal(data, intent, targetId)`**
([CODE: ui/src/hooks/useSessionManager.ts]). Its third parameter already
addressed a specific pane, so the missing part was never the transport — it was
the UI and the queueing.

- `ui/src/lib/handoff.ts` — `renderHandoffPrompt`, the send path's only text
  transformation. It imports nothing at all.
- `ui/src/hooks/useHandoff.ts` — `sendHandoff`, which returns a per-target
  `sent` / `queued` / `failed` result.
- `ui/src/lib/captureRules.ts` — the matcher, which produces suggestions.

**The send path and the matcher share no code.** Deleting every capture rule
must not change one line of the send path's behaviour, and the way that is
guaranteed is that neither side can reach the other. Three greps hold the line,
and all three must return nothing:

```bash
# 1. Nothing in the shipped surface is named for one workflow.
grep -rn --exclude=web-console-api --exclude=cli \
  "plan_path\|planPath\|plan_file\|implementer_id\|planner_id\|critic_id" \
  scenarios/web-console/api scenarios/web-console/ui/src \
  scenarios/web-console/cli packages/proto/schemas/web-console

# 2. The send path does not reach the matcher.
grep -rn "captureRules" scenarios/web-console/ui/src/lib/handoff.ts \
  scenarios/web-console/ui/src/hooks/useHandoff.ts

# 3. Seeded content is data, not a privileged code path.
grep -rn "is_builtin\|isBuiltin\|builtin" \
  scenarios/web-console/api/internal/grouptemplates \
  scenarios/web-console/api/internal/handoffrules
```

## The snippet seam (added 2026-08-28)

The mechanism record is
[SNIPPETS-AND-MESSAGE-ACTIONS-UX.md](SNIPPETS-AND-MESSAGE-ACTIONS-UX.md).
`ui/src/lib/snippetVars.ts` is the only text transformation for a snippet. It
recognizes only lowercase named tokens and supplied string values, and it has
no imports. Promotion is a one-way command call; neither the snippet row nor
any role, template, or skill stores a link back to the other surface.

These four greps must return no match:

```bash
# 1. No workflow- or skill-link field entered the snippet surface.
grep -rn --exclude=web-console-api --exclude=cli \
  "plan_path\|planPath\|plan_file\|snippet_skill_id\|skill_id" \
  scenarios/web-console/api scenarios/web-console/ui/src \
  scenarios/web-console/cli packages/proto/schemas/web-console

# 2. Neither text transformation nor the send path can reach the matcher.
grep -rn "captureRules" scenarios/web-console/ui/src/lib/handoff.ts \
  scenarios/web-console/ui/src/lib/snippetVars.ts \
  scenarios/web-console/ui/src/hooks/useHandoff.ts

# 3. Seeded snippets are ordinary rows, never privileged built-ins.
grep -rn "is_builtin\|isBuiltin\|SeedSnippetID" \
  scenarios/web-console/api scenarios/web-console/ui/src \
  | grep -v "api/internal/snippets/seed.go"

# 4. Substitution stays dependency-free.
grep -n "^import" scenarios/web-console/ui/src/lib/snippetVars.ts
```

### Delivery, and why a handoff is never dropped

`submitToActiveTerminal` returns `{status: "rejected", reason: "disposed"}` when
no terminal handle exists for the target. A session created one millisecond ago
has no mounted terminal, so a naive send is dropped **silently** — the single
most likely defect in this feature.

Two existing mechanisms remove the need to invent anything:

1. **The pending-input queue** (`useStdinStream`). Queued text is visible to the
   operator, discardable, and manually flushable. A handoff enqueues; it does
   not race.
2. **The pending-map pattern.** `Workspace.tsx` already carried
   `pendingGroupBySessionRef`, drained by the session-reconcile effect when the
   pane appears. `pendingHandoffBySessionRef` has the identical shape and is
   drained in the same effect — a second draining mechanism would be a second
   place for a message to go missing.

`queued` is a first-class result, never collapsed into `sent`. Reporting success
for text still sitting in a queue is the failure this seam exists to prevent.

### Role and pane

A role is the durable identity inside a group; a pane is the runtime projection
of a live session. They are joined by `session_id`, and roles are optional —
every pre-roles grouping behaviour works with `workspace_roles` empty. Full
table contract: [data-model.md](../reference/data-model.md#workspace_roles).

`ReassignPane` re-keys a pane during session recovery; `ReassignRoleSession`
performs the matching move for roles, so a recovered session keeps its role
rather than leaving it aimed at a session id that no longer exists.

## Session decomposition (refactored 2026-04-24)

`api/session.go` was split by concern — all methods still on
`*Session`, but each file now names a single responsibility:

- [CODE: api/session.go] — Session struct, lifecycle (Create, Delete,
  Exit, Resize, WriteInput, ProbeReady), policy, readLoop.
- [CODE: api/broadcast.go] — Output fan-out, per-client coalesce,
  pending-buffer trim, SIGWINCH recovery gating.
  (`ClientInfo`, `broadcast`, `deliver`, `FlushPending`,
  `maybeSIGWINCHRecovery`, `notifyIfThreshold`)
- [CODE: api/terminal/] — Decoded terminal emulator (parser, screen
  grid, bounded scrollback ring, alt-buffer flag) and the ANSI
  snapshot serializer. The session holds one `*terminal.Emulator`;
  every PTY read is fed into it, and every `Subscribe()` returns a
  self-contained snapshot.
- [CODE: api/terminal_ws.go] — WS upgrade + handler glue.
- [CODE: api/terminal_ws_input.go] — Per-message input dispatch
  (kind-aware stdin, resize, ping/pong, conversation_event_ack).

Greenfield assertion tests in [CODE: api/greenfield_assertions_test.go]
enforce:

- No `ptmx.Write(` outside `pty.go`/`pty_tmux.go`.
- No legacy `PTY.Write(p []byte) (int, error)` method declaration.
- No references to deleted rework/phase-2 plan filenames.
- `SIGWINCH` via `SetSize` only inside `maybeSIGWINCHRecovery` /
  `Resize` (checked across both `session.go` and `broadcast.go`).

## Terminal reliability seams (2026-08-26)

The terminal path has three explicit boundaries that must remain transport-
neutral:

- **Platform availability:** OS-specific PTY and echo files are selected by
  build tags. The capability registry and `.vrooli/service.json` publish the
  resulting status; callers use typed unsupported errors instead of a
  runtime host-support boolean.
- **Persistent input/control:** one bounded per-session writer queue orders
  reliable stdin, best-effort control bytes, and ANSI responder replies. The
  persistent backend keeps a long-lived tmux control client for command-style
  operations; the PTY interface hides that channel from session and HTTP code.
- **Protocol state:** `ui/src/lib/terminalProtocol.ts` is a DOM-free reducer
  for replay, live, resync, echo, size, and presence state. WebSocket effects
  remain in the hook, while this reducer is tested independently of xterm.

Session synchronization follows the same ownership split: `emuMu` protects
the emulator and snapshot cache, `clientsMu` protects viewers/leases/presence,
and `ptyMu` protects only replacement of the PTY pointer. The locks are
acquired in `clientsMu` → `emuMu` order and no lock is held across backend
I/O. Backend echo state is sampled into a session cache with a 250 ms floor;
the WebSocket path emits only changed combined state and retains a five-second
maximum refresh while a client is connected.

## Responsibility Zones

### Remote terminal federation

`api/remote_terminal.go` is the server-side federation adapter for
`vrooli-bridge`. It is an intentional REST/WS exception because the browser
terminal wire protocol is JSON-over-WebSocket while Bridge sessions are binary
protobuf-over-WebSocket. The browser receives only target readiness facts and
a short-lived web-console session ID; Bridge owner and re-authentication tokens
remain server-side. An enrolled local operator session is preferred, and the
shared `nodeclient` owns per-request Bridge discovery, authentication, and
stream setup. The adapter translates stdin sequence numbers, stdout, resize,
acknowledgements, launch commands, and close events. A target is unavailable
when its shared readiness facts or server-side operator authorization fail.

### 1. Entry / Presentation
**Owner**: `ui/src/components/`
- [CODE: ui/src/components/Workspace.tsx] — **Stable core**: pane grid layout, header, empty-state UI. Delegates all session logic to `useSessionManager` hook.
- [CODE: ui/src/components/ErrorBanner.tsx] — **Volatile edge**: reusable error display with category/recovery/retry. Single place to change error UX.
- [CODE: ui/src/components/TerminalPane.tsx] — xterm.js rendering plus pane-local conversation ingestion, received/seen acknowledgements, and provider control plumbing. It does not own auto-TTS policy or listened-cursor commits.
- [CODE: ui/src/components/MessagesPane.tsx] — semantic messages rendering with per-message TTS controls (read-from-here, read-one, stop); delegates TTS execution to TerminalPane via Workspace callbacks; never owns TTS provider directly
- [CODE: ui/src/components/TerminalLauncher.tsx] — Modal UI for session creation and shortcut selection (reads shortcuts from [CODE: ui/src/consts/shortcuts.ts])
- [CODE: ui/src/components/SessionDrawer.tsx] — Sidebar with session list and delete controls
- [CODE: ui/src/components/MobileToolbar.tsx] — Floating key toolbar for mobile input injection
- [CODE: ui/src/components/AiSuggestBar.tsx] — Inline AI suggestion bar for mobile; sits above MobileToolbar, shows debounced command suggestions from textarea input
- [CODE: ui/src/components/KeyComboPicker.tsx] — Bottom sheet picker for sending key combos with a single tap
- [CODE: ui/src/components/VoiceCommandSuggestion.tsx] — Command confirmation bar displayed above MobileToolbar when a voice command is detected in persistent mode

### 1b. Session Orchestration
**Owner**: [CODE: ui/src/hooks/useSessionManager.ts]
- `useSessionManager` — **Stable core**: pane state, session CRUD callbacks, terminal ref management, pending command bookkeeping. Separates lifecycle logic from layout.

### 1c. Conversation Orchestration
**Owner**: [CODE: ui/src/hooks/useConversationSession.ts], [CODE: ui/src/stores/useConversationStore.ts]
- `useConversationSession` — hydrates one session's conversation history and persists cursor updates to the API.
- `useConversationStore` — client-side semantic source of truth for conversation events, unread counts, listened state, and per-session terminal/messages view mode.
- Key invariant: conversation features consume `ConversationEvent`s, never raw PTY output history.

### 2. Transport / Protocol
**Owner**: [CODE: ui/src/hooks/terminal/useTerminalSession.ts] (client), [CODE: api/terminal_ws.go] (server)
- `useTerminalSession` — Composes three focused hooks and exposes a single surface:
  - [CODE: ui/src/hooks/terminal/useTerminalTransport.ts] owns the WebSocket lifecycle (connect, reconnect backoff, visibility-aware defer) and a monotonic `wsGen` counter.
  - [CODE: ui/src/hooks/terminal/useStdinStream.ts] owns cumulative UTF-8 offsets, the offline pending-input queue, reconnect reconciliation, and replay.
  - The session hook wires session_ready gating, history_end replay, and `pty_state` → local-echo enable/disable into a shared `TerminalInputGate`.
- `terminal_ws.go` — Server-side WebSocket upgrade, message framing, PTY I/O bridging, ping/pong. Emits `session_ready{gen}`, `history_end{total_bytes, resumed}`, and `pty_state{altBuffer}` in addition to stdout/sync_warning/stdin_ack.
- Key invariant: every stdin path (xterm.onData, mobile toolbar, paste, voice, upload) flows through the same gate (§2c), so a single state-aware decision point governs whether a byte goes to the PTY or is held.

### 2c. Single-path Input Gate
**Owner**: [CODE: ui/src/components/terminal/inputGate.ts]
- `TerminalInputGate.submit(data, source: InputSource)` returns a typed `GateResult`:
  - `{status: "sent", seq}` — handed to the WebSocket stack.
  - `{status: "queued", reason}` — pending; flushes on next session_ready. Reasons: `"not-ready"`, `"ws-closed"`, `"paused"`.
  - `{status: "rejected", reason}` — refused. Reasons: `"empty"`, `"disposed"`.
- Paste source uses the same reliable stdin lane as every other operator payload. Mouse-report bytes are the only xterm input routed to the best-effort control lane; paste is never held in a second mouse-mode queue.
- Every consumer imports the gate result type; there is no `boolean`-returning shortcut. See `greenfield-assertions.test.ts` for the enforcing tests.

### 2b. Conversation Ingestion
**Owner**: [CODE: api/conversation_router.go], [CODE: api/tts_hook_handler.go], [CODE: api/codex_tailer.go], [CODE: api/grok_tailer.go], [CODE: api/opencode_watcher.go]
- Claude hook adapter parses Stop-hook payloads and appends assistant conversation events (`source=claude_hook`).
- Codex tailer parses rollout output and appends user/assistant conversation events (`source=codex_tailer`).
- Grok tailer parses `updates.jsonl` ACP turns under a per-session `GROK_HOME` and appends user/assistant text at each `turn_completed` boundary (`source=grok_tailer`).
- OpenCode watcher subscribes to a managed `opencode serve` SSE stream and reconciles via `GET /session/{id}/message`, appending user/assistant text (`source=opencode_api`).
- `AppendAssistant` / `AppendUser` (the `ConversationDispatcher` seam) is the only semantic ingestion path; every adapter routes through it. No adapter writes the conversation store directly or scrapes PTY output.
- Source names are a stable, documented set: `claude_hook`, `codex_tailer`, `grok_tailer`, `opencode_api`.
- Key invariant: source adapters produce normalized conversation events first; TTS is downstream of those events.
- Checkpoint invariant: each adapter persists a replay-safe, source-scoped cursor in `agent_transcript_checkpoints`. Codex and Grok use byte offsets (Grok advances only at turn boundaries); OpenCode uses a per-session JSON high-water mark. `ConversationStore` short-window dedup is the second line of defense, never the primary guard.

### 3. Domain / Session Lifecycle
**Owner**: [CODE: api/session.go], [CODE: api/pty.go]
- `PTY` interface ([CODE: api/pty.go#PTY]) — Abstracts PTY process behind `Read`/`Write`/`SetSize`/`Close`/`Kill`. Default `realPTY` wraps creack/pty; tests substitute `fakePTY` (pipe-based).
- `PTYFactory` ([CODE: api/pty.go#PTYFactory]) — Function type `func(shell, cols, rows) (PTY, error)`. Injected into SessionManager via `NewSessionManagerWithFactory()`.
- `Session` — PTY process wrapper: delegates I/O to `PTY` interface, manages subscribe/unsubscribe/broadcast, offline buffer, exit signaling via `exitCh` channel. Includes UTF-8 boundary buffering in `readLoop`, frame coalescing in `broadcast`/`deliver` (with ANSI-boundary-aware capping via `snapToCleanBoundary`), and chunked history replay via `historyChunkSize` constant.
- `FlushPending(ch)` — Testability seam for the coalescing mechanism. The WS output forwarder calls this after each successful write. Chunks coalesced data at `historyChunkSize` (64 KB) to prevent browser UI freezes from large blobs. Tests can call it directly to control the drain cycle and verify coalesced data integrity.
- `splitCompleteUTF8(data)` — Pure function seam for UTF-8 boundary detection. Splits byte slices at complete codepoint boundaries, enabling isolated unit testing of the buffering logic.
- `SessionManager` — Session CRUD, resize (delegates to `PTY.SetSize()`), auto-cleanup on exit (listens on `Session.Done()`)
- Key invariant: Session signals its own exit; SessionManager owns the cleanup decision

### 3b. Domain / Conversation Lifecycle
**Owner**: [CODE: api/conversation_store.go]
- `ConversationStore` — In-memory ordered event log keyed by session ID.
- Owns:
  - append ordering via per-session `sequence`
  - cursor state (`lastSeenSequence`, `lastListenedSequence`)
  - per-event delivery/TTS/consumption state
  - list/get operations for conversation history APIs
- Key invariant: semantic response state is stored here, not reconstructed from PTY output or browser-rendered terminal state.

### 4. HTTP Transport (REST)
**Owner**: [CODE: api/session_handlers.go], [CODE: api/conversation_handlers.go]
- Request parsing, response formatting, HTTP status codes
- [CODE: api/session_handlers.go#sessionToResponse] — domain-to-transport conversion
- Policy sub-resource handlers (`handleGetPolicy`, `handleUpdatePolicy`) — operate on `/sessions/{id}/policy`, co-located with other session endpoints
- Delegates all business logic to `SessionManager` and domain modules
- Conversation handlers expose:
  - `GET /api/v1/sessions/{id}/conversation`
  - `PUT /api/v1/sessions/{id}/conversation/cursor`

### 4a. Sanctioned REST Exceptions (UI → API)

Every UI → API call goes through Connect-RPC except for the endpoints
below. All are template-sanctioned `RESTReason` values (see
`api/internal/module/module.go::RESTReason*` in the react-vite
template). Adding another REST surface requires either picking another
enumerated `RESTReason` or extending the template — not a one-off
exemption.

| File | Endpoint | Reason | Why REST |
|---|---|---|---|
| [CODE: ui/src/api/health.ts] | `GET /health` | `RESTReasonOpsProbe` | The API liveness probe must answer before Connect-RPC routing is wired up. Load balancers, lifecycle checks, and `curl` need the simplest possible shape. The proto in `packages/proto/schemas/web-console/v1/health/health.proto` carries the JSON wire shape so the response decodes through `fromJson(ResponseSchema, ...)` for type safety — there is no hand-rolled `HealthResponse` type. |
| [CODE: ui/src/api/uploads.ts] | `POST /sessions/{id}/upload` | `RESTReasonMultipartUpload` | Multipart binary upload. Connect-RPC binary payloads are non-trivial; the template explicitly enumerates multipart as an allowed shape. Metadata around uploads (if any) stays proto-typed; only the raw bytes ride the REST edge. |
| [CODE: ui/src/api/filePreview.ts] (consumed by native elements, not `fetch`) | `GET\|HEAD /sessions/{id}/file-previews/{previewId}/blob` | `RESTReasonOpsProbe` | Byte-range blob stream consumed directly by native `<img>/<video>/<audio>/<iframe>` `src`/`href` — browser-native transport Connect cannot express (the same category as `terminal_ws`). The opaque, session-bound `preview_id` (never a raw path) is issued by `FilePreviewService.Resolve`; preview metadata + bounded text stay proto-typed over Connect. Bytes never travel through Connect. |

**Regression guard**: [CODE: ui/src/api/__tests__/no-rest-exceptions.test.ts]
greps `ui/src/api/*.ts` for the literal token `fetch(` and fails if it
appears outside `health.ts` and `uploads.ts`. `filePreview.ts` does not
trip it: the blob bytes are loaded by native element `src`/`href`
attributes, not a `fetch(` call, so the file-preview REST surface adds no
`fetch(` to the API layer.

### 5. Integration / Infrastructure
**Owner**: [CODE: api/main.go], [CODE: ui/src/lib/api.ts]
- `main.go` — Database connection, router setup, health checks, server lifecycle
- `api.ts` — HTTP/WS client functions, URL construction via `@vrooli/api-base`
- Key invariant: API contracts expose semantic conversation data directly; clients should not infer conversation state from terminal frames.

### 6. Cross-Cutting
- **Logging**: `log.Printf()` in Go API (simple, adequate for single-user)
- **Error handling**: Structured JSON errors via [CODE: api/session_handlers.go#errorCatalog], TanStack Query in UI
- **Formatting**: [CODE: ui/src/lib/format.ts] — reusable shell name, time, ID truncation utilities
- **Selectors**: [CODE: ui/src/consts/selectors.ts] — centralized data-testid registry for automation
- **Shortcuts**: [CODE: ui/src/consts/shortcuts.ts] — **Volatile edge**: shortcut definitions, decoupled from launcher component

## TTS Playback Lifecycle Seam (UI, added 2026-07-07)

Streaming-tail-durability made spoken output survive the three teardown
races that used to truncate it. The ownership boundary is:
`useTtsPlaybackController` (domain policy: which event plays, queueing,
version) → `useTextToSpeechCore` (provider lifecycle) → `KokoroProvider`
(single `HTMLAudioElement`). The durable guarantees:

- **Resilient per-paragraph sequence.** `KokoroProvider.speakSequence`
  synthesizes paragraphs pipelined (concurrency 2) and plays each as its
  own track. A single paragraph's synth-reject or MP3 decode error is
  isolated — retry once → per-paragraph browser-voice fallback →
  skip-with-notice — and the sequence **continues**. Only a real
  stop/dispose abort halts the tail. Non-fatal degradation surfaces via
  `onParagraphOutcome` (observability), never gating playback. This is the
  TTS twin of the audio-tools event-durability contract (each paragraph is a
  durable ordered unit): see
  [`scenarios/audio-tools/docs/domains/stt/streaming-pipeline.md#event-durability-contract`](../../../audio-tools/docs/domains/stt/streaming-pipeline.md#event-durability-contract).
  [CODE: ui/src/audio-integration/hooks/tts/KokoroProvider.ts]
- **Playback survives pane unmount / warm-set eviction.** The workspace
  keeps only `WARM_SET_SIZE` panes mounted; an evicted pane used to
  dispose its provider mid-tail. `useTextToSpeechCore` opts into a
  process-wide `ttsPlaybackRegistry` keyed by session id
  (`playbackOwnerKey` + `persistPlaybackAcrossUnmount`): on unmount while
  speaking it **hands the provider off** instead of disposing; a remount
  **re-adopts the same instance** (single owner, no leak). The registry
  disposes an orphaned provider once its tail settles (`onSettled`). A
  new `speak` calls `stopOrphansExcept` so two sessions never speak at
  once; genuine session-end (`removePane`) calls `registry.stop`.
  [CODE: ui/src/audio-integration/hooks/tts/playbackRegistry.ts]
- **Mid-playback messages queue, not drop.** An assistant message
  arriving while TTS is busy is enqueued in a bounded FIFO
  (`MAX_AUTOPLAY_QUEUE = 8`, oldest dropped on overflow) and spoken when
  the current one ends (`drainPendingAutoplay`), honoring the current
  playback intent — replacing the old `!isSpeaking` guard that silently
  dropped it. [CODE: ui/src/domains/tts-playback/useTtsPlaybackController.ts]

- **Replays serve from the byte cache (per paragraph).** The synth path
  threads a real `event_id` plus a per-paragraph `chunk_index` (proto,
  additive) so each paragraph populates the audio-tools byte cache under a
  distinct key — no whole-message collision. `useTextToSpeechCore`'s
  replay path fetches chunk `0..N-1`; a full hit plays them via
  `KokoroProvider.speakFromBlobs` with **no synthesis**, and any miss falls
  through to synth (which repopulates every chunk). The cache is populated
  as a side-effect of the live per-paragraph synth (zero extra synth); the
  chain-path `Synthesize` handler in audio-tools now writes the cache it
  reads (previously a dead seam — replays always missed).
  [CODE: ui/src/audio-integration/api/tts.ts],
  [CODE: scenarios/audio-tools/api/handlers/tts/connect_handler.go]

## STT Ingress Durability Seam (UI, added 2026-07-08)

The dictation (STT) direction obeys the same audio-tools event-durability
contract as TTS playback:
[`scenarios/audio-tools/docs/domains/stt/streaming-pipeline.md#event-durability-contract`](../../../audio-tools/docs/domains/stt/streaming-pipeline.md#event-durability-contract).
The client is the last hop; its durable guarantees:

- **Lossless tail recovery via a committed-length cursor.** A turn that ends on
  an uncommitted partial (a teardown race dropped the flush) promotes exactly
  the remainder of the latest partial that lies BEYOND the durable segment-finals
  already committed — recovering the full uncommitted tail without ever
  double-appending committed words. Replaces the single overwritten
  trailing-partial slot. [CODE: ui/src/audio-integration/hooks/voice/trailingPartial.ts]
  (`uncommittedRemainder`), wired in [CODE: ui/src/audio-integration/hooks/useVoiceCore.ts].
- **Coalesced partial render.** Interim `partial` text is throttled to one paint
  per animation frame (durable segment-finals still render immediately), so a
  high partial rate cannot jank the main thread and re-introduce client-side
  backpressure. Cancelled on every turn-terminal path. [CODE: ui/src/audio-integration/hooks/useVoiceCore.ts]
- **Processed-coverage retention.** `PcmVoiceStreamProvider` writes each
  canonical PCM frame to the bounded origin-local turn journal before it is
  released to the same-origin WebSocket. It compacts only on the server's
  `processed_acknowledgement`; persisted next-sequence and sample cursors stay
  intact even when all replay bytes have compacted. [CODE:
  ui/src/audio-integration/hooks/voice/PcmVoiceStreamProvider.ts]

## Testability Seams

### Terminal Screen Read Seam (API)
**File**: `api/terminal/view.go`
**Purpose**: Expose the decoded grid (cells, cursor, alt-buffer flag, scrollback count) as plain Go values so programmatic consumers — Connect-RPC, CLI, agents — can inspect the screen without parsing ANSI.

| Component | Surface | Notes |
|-----------|---------|-------|
| `Emulator.Cursor`, `Cells`, `View` | Deep-copy reads under the owner's mutex | Outputs are owned by the caller |
| `Emulator.PlainText(includeScrollback)` | Plain UTF-8, trailing blanks stripped | Replaces the historical `stripANSI` helper for screen-text use cases |
| `handlers/terminal.TerminalService.GetScreen` | Connect-RPC wrapper | Adds `plain_text` convenience field |

### Session Input Seam (API)
**File**: `api/session/input.go`
**Purpose**: Single typed envelope (`SessionInput`) plus a single `applyInput` PTY-write call site. WS input handler, recovery adapter, Connect TerminalService.SendInput, and the legacy ANSI responder all funnel through here.

| Component | Variants |
|-----------|----------|
| `InputText`, `InputKeys`, `InputRaw` | Constructors; the value type is opaque to callers |
| `KeyMap` (interface) | The terminal Connect handler's `DefaultKeyMap` resolves Enter/Tab/arrows/F1-F12/Ctrl+&lt;letter&gt; for programmatic callers; the WebSocket path encodes keys in the browser |
| `Session.SendInput` | Resolves bytes via `KeyMap` and calls `applyInput` (the single PTY-write seam) |

### Terminal Control Event Seam (API)
**File**: `api/terminal/events.go`
**Purpose**: Stream parsed control events (alt-buffer enter/exit, CSI queries DA1/DA3/XTVERSION/DECRQM 2026) for observers that need them. The ANSI auto-responder (Phase 3, 2026-05-13) consumes this stream and writes server-side replies through `Session.SendInput`; no inline byte-scan exists in `readLoop` any more.

| Component | Surface |
|-----------|---------|
| `Emulator.ControlEvents()` | Read end of a bounded (256) channel; lazily allocated |
| `Session.startAnsiResponder()` | Spawns the observer goroutine for non-persistent backends |
| `ansiReplyFor(ControlEvent) []byte` | Pure mapping from event → reply bytes (unit-testable) |

Backpressure: drop-oldest; the read loop must never block. The responder skips persistent (tmux) backends — tmux answers queries for its own panes.

### ANSI Strip Seam (API)
**File**: `api/terminal/strip.go`
**Purpose**: Stateless helper for callers that have a byte stream they want to render as plain text without spinning up an emulator (e.g. conversation-log normalization, dedup-key computation). For grid-level reads — visible cells, cursor, scrollback — use `Emulator.View()` / `Emulator.PlainText()` instead.

| Component | Surface |
|-----------|---------|
| `terminal.StripEscapes([]byte) []byte` | Removes CSI / OSC / two-byte ESC sequences; preserves UTF-8 |

Replaces the pre-Phase-3 `stripANSI` helper that lived in `package main`.

### Conversation Dispatcher Seam (API)
**File**: `api/conversation_router.go`
**Purpose**: Narrow interface (`ConversationDispatcher`) for publishing trusted assistant and user conversation events to a terminal session. Lets non-Server callers — hook handlers, the codex/grok tailers, the OpenCode watcher, future adapters — depend on a small surface instead of `*Server`, and lets tests substitute a fake dispatcher. The OpenCode watcher additionally depends on the fakeable `opencode.Client` seam (session list / message history / SSE events) and an injectable `startServer`, so its backfill, reconcile, reconnect, and attribution logic are unit-tested against an in-memory fake with no real `opencode serve`.

| Component | Surface |
|-----------|---------|
| `ConversationDispatcher.AppendAssistant(text, sessionID, source)` | Publish an assistant response, run TTS routing |
| `ConversationDispatcher.AppendUser(text, sessionID, source)` | Publish a user prompt (no TTS) |
| `*Server` implements both implicitly | Compile-time check via `var _ ConversationDispatcher = (*Server)(nil)` |

### Conversation Hub Seam (API)
**Files**: `api/conversation_hub.go`, `api/events_stream.go`
**Purpose**: Process-wide conversation event channel. Every conversation event (assistant/user append, async summarize update) is published once to `ConversationHub` and fanned out to all Server-Sent Events subscribers over `GET /api/v1/events/stream`, decoupled from any single session's terminal WebSocket. The browser opens ONE stream for ALL sessions, so unread badges and conversation deltas no longer depend on a per-session terminal WS being open. This replaced the per-session conversation fan-out entirely — there is exactly one conversation-event channel.

| Component | Surface |
|-----------|---------|
| `ConversationHub.Publish(env) int64` | Assigns the next monotonic global id, retains in the ring buffer, fans out to subscribers (never blocks; drops + resync-signals a full subscriber) |
| `ConversationHub.Subscribe(lastEventID) (*hubSubscriber, []HubEnvelope, gap)` | Registers a client; replays retained envelopes newer than the cursor; `gap=true` when the cursor predates the retained window |
| `ConversationHub.Unsubscribe(sub)` | Removes a client from the fan-out (idempotent) |
| `Server.publishConversationEvent(event)` | Single publish path; maps `event.IsUpdate` → kind (`conversation_event_update` / `conversation_event`) |
| `Server.handleEventStream(w, r)` | SSE handler; honors `Last-Event-ID` header / `?last_event_id=` query (header wins); emits `conversation_out_of_sync` on gap so the client backfills via `GET /conversation?since_sequence=N` |

**Knobs (package vars/consts for tests)**: `hubRingSize` (replay buffer depth, default 1024), `hubSubscriberBuffer` (per-subscriber channel, default 256), `hubHeartbeatInterval` (SSE keepalive comment cadence, default 15s).

### File Preview Resolver + Preview-ID Store Seam (API)
**File**: `api/internal/filepreview/` (resolver, classification, store), `api/file_preview_handlers.go` (adapter + blob route)
**Purpose**: Keep path resolution, MIME/kind classification, and the opaque preview-id store transport-neutral and independently unit-testable, and keep the blob route from ever becoming an arbitrary local file server.

| Component | Surface |
|-----------|---------|
| `filepreview.Resolver` | `Resolve(sessionCwd, cwdErr, rawPath) (*Target, error)` — pure: probes absolute → session_cwd → project_root, classifies by extension + content sniff, downgrades oversize text. `ReadText(*Target)` re-validates UTF-8 + size cap. No session/Connect/mux deps. |
| `filepreview.Store` | `Issue(sessionID, *Target) (id, expiry)` / `Lookup(sessionID, id)` — in-memory, session-bound, expiring (`DefaultTTL` 30m). All miss modes collapse to `ErrPreviewNotFound`. `now`/`rand` are injectable for tests. |
| Blob handler | `Server.handleFilePreviewBlob` — accepts only a preview id (never a raw path), re-stats the file (409 on size/mtime drift, 404 on delete), sets `Content-Type`/`no-store`/`nosniff`/`Content-Disposition`, then `http.ServeContent` for Range/HEAD/206/416. |

**Invariants**: directory traversal is impossible because resolution happens before id issuance; the blob route binds id↔session; a swapped file is never served under a stale id.

### File Preview Renderer Registry Seam (UI)
**File**: `ui/src/components/file-preview/` (controller, renderers, registry)
**Purpose**: Keep `MessagesFileViewer` a thin shell — a normalized `PreviewModel` state machine feeding a kind-keyed renderer table — so new preview kinds are additive, not new special-cases.

| Component | Surface |
|-----------|---------|
| `useFilePreviewController(sessionId)` | Owns the `idle → resolving → loadingText → ready \| unsupported \| error` state machine; request-id guarded; the only seam surfaces (MessagesPane, future) call to open a path. |
| `renderers` registry (`renderers/index.ts`) | `Record<PreviewKind, PreviewRenderer>` — one component per kind; `rendererForKind` falls back to the unsupported renderer. |
| `previewBlobHref` / `format.ts` | Pure helpers (blob URL join, `formatBytes`, `parseDelimited` CSV/TSV parser) split out for fast-refresh + unit tests. |

### API↔CLI Parity Seam (CLI)
**File**: `cli/parity_test.go`
**Purpose**: Lock in the contract that every Connect-RPC method has a matching CLI command. Drift in either direction fails the test with a punch list so an agent can't ship a new RPC without a CLI command — and can't quietly drop a CLI command that the proto seed still references.

| Component | Surface |
|-----------|---------|
| `gen-endpoints` manifest coverage gate | Loads `cli/manifest.json` and registered endpoint descriptors, then asserts every Connect procedure is either bound in the manifest or explicitly omitted there. |
| `parityCLISkipIDs` | Explicit opt-out map for endpoints that genuinely cannot have a CLI form (server streams, long-lived subscriptions). Adding here requires justification |
| Note marker | An endpoint's `rest_exception.note` can contain `cli:skip` to opt out without touching test code |
| `builtinCLICommands` | Allowlist for commands provided by cli-core's `NewStandardScenarioApp` (e.g. `status`) that domain registration doesn't surface |

**Invariants**: command lookups strip the `web-console ` binary prefix and drop trailing `--flag` segments so `capabilities --liveness` resolves to the same key as `capabilities` (the flag is parsed inside the handler, not by the dispatcher).

### Backend Plug-Point Seam (API)
**Files**: `api/internal/backend/plug.go`, `api/internal/backend/backend.go`, `api/backends/claude/`, `api/backends/codex/`
**Purpose**: Optional, code-only extension hooks on `backend.Descriptor` so per-backend behavior (key encoding, prompt detection, idle gating) lives next to the backend instead of branching in the session pipeline. All fields are `json:"-"` so the descriptor's wire shape is unchanged.

| Component | Surface |
|-----------|---------|
| `backend.KeyMap` | `Encode(name) ([]byte, bool)` — symbolic key → bytes (e.g. `Ctrl+C` → `\x03`). Nil means session default. |
| `backend.PromptDetector` | `IsAwaitingInput(view ScreenView) bool` — backend-aware "agent is at input prompt" signal. Nil means fall back to idle heuristics. |
| `backend.IdleHeuristic` | `QuietWindowExceeded(sinceLastMillis int64) bool` — backend-aware quiet-window decision. Nil means session default. |
| `backend.ScreenView` | Narrow read surface for detectors: `Cols/Rows/CursorRow/CursorCol/PlainText`. Session adapts its richer screen type to this interface. |
| `backends/claude/` | `FilterEnv`, `DefaultPromptDetector` — claude-specific env stripping + ❯-glyph-on-cursor-row detector. |
| `backends/codex/` | `SharedHome`, `SessionHome`, `SessionsDir`, `PrepareSessionHome`, `ExtractAssistantText`, `ExtractUserText`, `RolloutLine`, `DefaultPromptDetector` — codex-specific home layout + rollout parsing + heuristic detector. |

**Invariants**: Nil-safe (verified by `internal/backend/plug_test.go`); no import cycle (interfaces live in `internal/backend`, not in `session/`); descriptor JSON unchanged so the UI's backend picker keeps working byte-for-byte.

### PTY Factory Seam (API)
**File**: `api/pty.go`
**Purpose**: Decouple session management from real PTY/process spawning for fast, deterministic tests.

| Component | Production | Test |
|-----------|-----------|------|
| `PTYFactory` | `defaultPTYFactory` → `realPTY` (creack/pty + exec.Command) | `fakePTYFactory` → `fakePTYWithOutput` (io.Pipe-based) |
| `SessionManager` | `NewSessionManager()` uses `defaultPTYFactory` | `NewSessionManagerWithFactory(factory)` accepts any `PTYFactory` |
| `Session.pty` | `realPTY` (real shell process) | `fakePTYWithOutput` (pipe-based, instant I/O) |

**Benefits**: Tests run without spawning shell processes (faster, no OS dependencies for core logic), resize delegates to the `PTY` interface (testable without ioctl), kill/close behavior is verifiable via the fake's state.

### WebSocket Factory Seam (UI)
**File**: `ui/src/hooks/terminal/useTerminalSession.ts`
**Purpose**: Decouple WebSocket transport from terminal protocol handling for testable hook behavior.

| Component | Production | Test |
|-----------|-----------|------|
| `createSocket` param | `defaultSocketFactory` → `new WebSocket(url)` | Custom factory returning mock/fake WebSocket |
| `ANSI` constants | Used internally for terminal messages | Exported for test assertions |
| `SocketFactory` type | `(url: string) => WebSocket` | Same signature, mock implementation |

**Benefits**: Hook can be tested with a mock WebSocket (no real connections needed), message handling logic (stdout/exit/error/ping) can be exercised in isolation.

### API-Base Mock Seam (UI)
**File**: `ui/src/test-utils/mocks.ts`
**Purpose**: Centralize `@vrooli/api-base` mock so all test files that depend on API URL resolution use a single, consistent factory.

| Component | Production | Test |
|-----------|-----------|------|
| `@vrooli/api-base` module | `resolveApiBase()` reads env/window config | `apiBaseMock()` returns deterministic localhost URLs |
| `buildApiUrl` / `buildWsUrl` | Constructs URLs from runtime base | Pass-through concatenation for predictable assertions |

**Benefits**: Eliminates 5-file mock duplication (previously each test file copied 7 lines of mock config with inconsistent port numbers). Single change point when the api-base interface evolves.

### Shared Test Doubles Seam (UI)
**File**: `ui/src/test-utils/mocks.ts`
**Purpose**: Provide reusable test doubles for WebSocket, terminal, and session data so tests focus on behavior, not boilerplate setup.

| Double | What It Replaces | Used By |
|--------|-----------------|---------|
| `FakeWebSocket` | Real `WebSocket` via `SocketFactory` seam | `terminal-session hook tests` |
| `createMockTerminal()` | xterm.js `Terminal` instance | WebSocket hook tests |
| `findWriteCall()` | Inline assertion search across terminal writes | WebSocket hook tests |
| `makeSessions()` | Inline session data construction | Component tests (SessionDrawer, etc.) |
| `createMockSession()` | Inline `SessionInfo` object literals | Any test needing session data |
| `mockFetchSuccess()` / `mockFetchError()` | Repeated `globalThis.fetch = vi.fn(...)` pattern | API client tests |

**Benefits**: New tests can set up realistic test data in one line. Mock behavior is consistent across test files. Changes to data shapes (e.g., adding a field to `SessionInfo`) require updating one factory, not many test files.

### Policy Selection Parse Seam (UI)
**File**: `ui/src/consts/policy-options.ts`
**Purpose**: Centralize policy select value parsing to avoid duplicated string-splitting logic and undefined behavior across session UIs.

| Component | Before | After |
|-----------|--------|-------|
| `SessionDrawer` | Inline parse (`if val === "never" else split(":")`) | Uses `parsePolicySelection(value)` helper |
| `SessionsPage` | Inline parse with separate branch/split logic | Uses `parsePolicySelection(value)` helper |
| Invalid values | Implicitly assumed valid | Explicit `null` return; caller no-ops safely |

**Benefits**: Single source of truth for UI policy parsing decisions, tighter edge-case tests at seam boundaries, and reduced drift risk between pages.

### Integrations Panel Seam (UI)
**Files**: `ui/src/components/IntegrationsPanel.tsx`, `ui/src/hooks/useCapabilities.ts`
**Purpose**: Display all resource/scenario dependency statuses via the CapabilityRegistry. Uses react-query with 30s polling, gated by `open` prop to avoid fetching when the settings modal is closed.

| Component | Production | Test |
|-----------|------------|------|
| `useCapabilities` hook | Calls `fetchCapabilities` with 30s refetch | Mock `fetchCapabilities` via `globalThis.fetch` |
| `IntegrationsPanel` | Renders capability cards with status icons, feature pills, diagnostic messages | `renderWithProviders` + mocked fetch |

**Benefits**: Unified view of all dependency health (Whisper, Kokoro, Ollama, OpenRouter) in one panel. No API injection seam needed — react-query's QueryClient injection in test-utils handles test isolation.

### Voice Command Parser Seam (UI)
**File**: `ui/src/hooks/voice/commandParser.ts`
**Purpose**: Decouple command detection from transcription and UI so parsing logic is testable in isolation.

| Component | Production | Test |
|-----------|-----------|------|
| `parseCommandDirect()` | Matches transcribed text against command vocabulary (no prefix required) | Direct function calls with controlled input strings |
| `levenshtein()` | Fuzzy matching for Whisper misrecognitions | Direct function calls with known edit distances |

**Benefits**: Command detection is a pure function with no UI or provider dependencies. All matching logic (fuzzy matching, number extraction) can be tested with simple string inputs. Wake word detection handles activation separately (see Wake Word Engine seam below).

### Wake Word Engine Seam (UI)
**Files**: `ui/src/audio-integration/hooks/voice/wakeword/types.ts`, `ui/src/audio-integration/hooks/voice/wakeword/engine.ts`, `ui/src/audio-integration/hooks/voice/wakeword/dtw.ts`, `ui/src/audio-integration/hooks/voice/wakeword/trim.ts`, `ui/src/audio-integration/hooks/voice/wakeword/passiveListener.ts`
**Purpose**: Isolate audio feature extraction and comparison behind a strategy interface so the MFCC+DTW implementation can be swapped for a neural embedding model later.

| Component | Production | Test |
|-----------|-----------|------|
| `WakeWordEngine` interface | Strategy abstraction: `extractFeatures` / `compare` / `compareBest(…, calibration?)` / `calibrate` | Allows mock engines in integration tests |
| `MfccDtwEngine` | Extracts 13-coeff MFCCs, normalizes (CMVN), drops c0 (energy) from the distance, compares via symmetric-step DTW, scores relative to enrollment calibration | `engine.test.ts` — CMVN, c0/loudness invariance, trim, calibrate, synthetic separation |
| `createWakeWordEngine()` | Factory — single point of change for swapping implementations | Tests call factory to verify wiring |
| `PassiveListener` | VAD + ring buffer + MFCC/DTW loop running in RAF tick; passes `template.calibration` into `compareBest` | Unit-testable via mocked engine and VAD refs |
| `extractMfcc()` | Pure-JS MFCC extraction (FFT, mel filterbank, DCT) | Tested with known-frequency sine waves |
| `trimSilence()` | Endpoint silence trim applied first inside `extractFeatures` (uniform across all consumers) | `engine.test.ts` — padded clip self-matches unpadded |
| `dtwDistance()` | Symmetric-step DTW (diagonal ×2, `/(n+m)` normalization), Sakoe-Chiba band, c0 excluded via `startCoeff` | `dtw.test.ts` — identity, time-warp invariance, c0 exclusion, corner reachability |
| `calibrate()` / `calibratedScore()` | Derives (μ,σ) of intra-enrollment-set DTW distances; maps a live distance to a 0–1 score relative to the user's own consistency | `dtw.test.ts` / `engine.test.ts` — anchors at μ and μ+kσ, monotonicity |

**Scoring contract**: A match score answers "how consistent is this utterance with the user's enrollment set," not "1/(1+raw distance)." `EngineCalibration` (μ,σ) and the MFCC features are BOTH derived on load from the persisted RAW audio — **never serialized** (no proto field). `WakeWordEngine.compareBest` takes an optional `calibration`; absent it falls back to an uncalibrated logistic so the engine is usable pre-calibration. Any future engine (e.g. `embedding-v1`) must implement `calibrate` too. The shared `WAKE_WORD_AUDIO_CONSTRAINTS` pins identical `getUserMedia` settings across enrollment / settings-test / passive paths so the acoustic channel matches at detection time.

**Benefits**: All wake word detection runs client-side (no audio leaves the browser during passive mode). The `WakeWordEngine` interface is the replacement seam — swapping to neural embeddings requires only a new class implementing the same interface and updating `createWakeWordEngine()`.

### Voice Segment Boundary Seam (API)
**File**: `api/internal/voice/stream_ws.go`
**Purpose**: Segment-final transcription runs in a goroutine separate from the partial ticker, allowing high-quality retranscription without blocking streaming partials.

| Component | Production | Test |
|-----------|-----------|------|
| Segment boundary channel | Receives from WebSocket input loop | Can be directly sent to in tests |
| Segment-final goroutine | Calls Whisper with transcoded audio | Mockable via injected `HTTPDoer` + transcode function |

**Benefits**: Segment finals are decoupled from the partial transcription loop, so each can be tested independently.

### Voice Command Vocabulary Seam (UI)
**File**: `ui/src/hooks/voice/commands.ts`
**Purpose**: Command definitions and execution are separated from UI components via the CommandContext interface.

| Component | Production | Test |
|-----------|-----------|------|
| `VOICE_COMMANDS` | Fixed command list with execute functions | Directly iterable for testing |
| `CommandContext` | Real session/terminal handles from Workspace | Mock implementations with assertion tracking |

**Benefits**: Commands can be tested by calling execute() with mock CommandContext, verifying the right terminal sequences are sent.

### Voice Input Provider Seam (UI)
**Files**: `ui/src/hooks/useVoiceInput.ts` (orchestrator), `ui/src/hooks/voice/` (modules)
**Purpose**: Decouple transcription backend selection from recording lifecycle, enabling testable voice input with swappable providers.

**Module structure** (each file has one responsibility):
- `voice/types.ts` — shared types, constants, `TranscriptionProvider` interface, `VoiceState` enum
- `voice/PcmVoiceStreamProvider.ts` — replay-safe PCM-v2 WebSocket provider
  (preferred); the older MediaRecorder provider is not selected by the voice
  core
- `voice/WhisperProvider.ts` — HTTP batch transcription provider
- `voice/WebSpeechProvider.ts` — Browser-native fallback + SpeechRecognition types
- `voice/vad.ts` — Voice Activity Detection pure functions
- `voice/autoStopDecision.ts` — Pure auto-stop verdict and matching mic-button ring projection
- `voice/audioUtils.ts` — `createAudioFilterChain` pure function
- `voice/index.ts` — barrel re-exports

**State machine**: `idle` -> `preparing` -> `recording` -> `transcribing` -> `idle`
(Replaces the old `isRecording`/`isTranscribing` boolean combo which allowed impossible states.)

| Component | Production | Test |
|-----------|-----------|------|
| `WhisperProvider` | Records via MediaRecorder, POSTs audio to `/api/v1/voice/transcribe` | Mock `navigator.mediaDevices` + mock fetch |
| `PcmVoiceStreamProvider` | Captures canonical PCM, journals before send, then streams v2 frames to `/api/v1/voice/stream` through the same-origin proxy; reconnect replay is deduplicated by the server session ledger | Mock WebSocket + mic + capture/journal seams |
| `WebSpeechProvider` | Uses browser SpeechRecognition API with `continuous: true`, `interimResults: true` | Mock `window.SpeechRecognition` |
| `TranscriptionProvider` interface | `start()`, `stop()`, `onResult`, `onError`, `onPartial` callbacks | Same interface, deterministic behavior |
| Mic ownership registry | Sole production path to `navigator.mediaDevices.getUserMedia()`; providers call `acquireMicStream(owner, constraints)` | Mock `getUserMedia`, assert lease owner/release |
| AudioContext singleton | Reused across recording sessions; resumed if suspended | Mock constructor, assert single creation |
| Language parameter | `voiceLanguage` from store -> `lang` (WebSpeech) / `language` (Whisper/Stream); `"auto"` omits language param for Whisper auto-detection | Set store value, assert provider property |
| Audio buffering (PcmVoiceStreamProvider) | v2 frames are written to the turn journal before they enter the pending queue; journal data is authoritative for reconnect replay | Mock WS in CONNECTING state, verify journal-before-send ordering |
| WS reconnection | 2 attempts with exponential backoff (1s, 3s) + at-least-once journal replay; server deduplicates by sequence/range/digest | FakeWebSocket close simulation |
| Stale WS cleanup | `start()` closes previous WS and resets MediaRecorder before creating new ones | Call `start()` twice, verify first WS is closed |
| Final timeout | `computeFinalTimeout(elapsed)`: max(10s, 2x recording duration), capped at 60s | Pure function, table-driven unit tests |
| Audio bitrate | `AUDIO_BITRATE = 48_000` for MediaRecorder `audioBitsPerSecond` | Constant, ~6KB/s on localhost |
| Stream chunk interval | `STREAM_CHUNK_INTERVAL_MS = 250` | Constant assertion |
| `createAudioFilterChain` | Builds highpass (80Hz) + lowpass (8kHz) Butterworth filter chain -> `MediaStreamAudioDestinationNode` + `AnalyserNode` | Mock AudioContext with fake node factories |
| `computeSlidingNoiseFloor` | 25th-percentile sliding window (30 samples ~= 2s at 15Hz) with asymmetric hysteresis (immediate rise, gradual decay at 0.5x/s) | Pure function, table-driven unit tests |
| `vadTick(vad, rms, now, silenceTimeoutMs)` | Exported pure function. Drives VAD state machine; accepts `silenceTimeoutMs` parameter (default 2000ms) | Direct unit testing with synthetic VadRefs and timestamps |
| `decideAutoStop()` / `decideAutoStopRing()` | Shared server/client authority for stopping and the countdown ring; stale-but-latched server timeout remains terminal | Pure tests + `VoiceMicButton` render tests |
| `processedResultCount` | WebSpeechProvider instance field tracking dispatched result indices to prevent cumulative duplication; persists across spontaneous browser restarts | Controllable SpeechRecognition stub fires cumulative `onresult` events |
| `startRecording` error guard | `try/finally` ensures `startingRef` is always cleared, preventing permanent lockout | Throw during capability check, assert subsequent recording succeeds |
| Capability liveness check | Pre-recording debounced check uses `fetchCapabilitiesLiveness` (GET-only, no test transcription) for fast response; full check only on mount | Mock both endpoints, verify liveness is used pre-recording |
| `capCheckResolvedRef` | Gates provider creation until mount-time capability check resolves, preventing wrong provider type | Click mic before mount check, verify streaming provider is used |

**Benefits**: Voice input can be tested without real microphone access or Whisper server. Fallback chain (Whisper -> Web Speech -> disabled) is testable by controlling capability fetch responses. AudioContext reuse prevents browser context limit exhaustion. Each provider is independently testable in its own module. State machine prevents impossible state combinations.

### Mic Ownership Seam (UI)
**File**: `ui/src/audio-integration/hooks/voice/micOwnership.ts`
**Purpose**: Single owner + lease registry for EVERY browser `getUserMedia` audio
stream opened by web-console UI. Gives every live mic stream exactly one
observable owner, lets page-lifecycle emergency cleanup release all of them
without per-owner handlers, and makes `MediaStreamTrack.stop()` the single
release path. Closes the iOS-PWA "mic indicator on while the UI looks idle"
failure class (passive wake-word / prewarm / settings captures leaking live
tracks). [DOC: docs/internal/VOICE-LATENCY.md#page-lifecycle-mic-cleanup-always-on-for-all-mic-owners]

| Component | Production | Test |
|-----------|-----------|------|
| `acquireMicStream(owner, constraints, opts?)` | `getUserMedia` + register a lease under a named `MicOwner` | Mock `getUserMedia`, assert lease + owner |
| `registerMicStream(owner, stream, opts?)` | Register an externally-acquired stream | Pass a fake stream, assert snapshot |
| `releaseMicLease(lease, reason)` / `lease.release` | Stop all tracks once, run `onRelease`, idempotent | Double-release → one `stop()`, one `onRelease` |
| `releaseAllMicLeases(reason, predicate?)` | Release every (filtered) lease | Predicate selects owners |
| `getActiveMicLeases()` | Metadata-only snapshots (never the raw stream) | Assert owner/trackCount, no `stream` field |
| `subscribeMicLeases(listener)` | Notify on every acquire/release with a metadata-only snapshot, so the UI can derive live-mic honesty without polling | Subscribe, acquire/release, assert snapshot + no `stream` field |
| `installMicLifecycleCleanup(resolveScope?)` | Ref-counted `visibilitychange`/`pagehide`/`freeze` backstop; `resolveScope(event)` (injected by useVoiceCore from `decideMicLifecycle`) selects `all` vs `non-active` per event/platform | Inject a resolver returning `all`, assert active recording released on hidden |
| lease `onRelease(reason)` | Owner resets its own state when released by anyone | Fire OS `ended`, assert owner reset |

**Key invariant**: lease release is idempotent and stops tracks exactly once;
`onRelease` lets the owner (micReadiness, PassiveListener, settings flows) reset
its own state when the registry or the OS releases the lease.

### Voice Capture Lifecycle Controller Seam (UI)
**Files**: `ui/src/audio-integration/hooks/voice/voiceCaptureController.ts`, `ui/src/audio-integration/hooks/voice/micLifecyclePolicy.ts`
**Purpose**: Single authority for transitioning provider/capture ownership in
`useVoiceCore`. Before this seam, provider replacement/disposal/error cleanup was
scattered across several hook branches — a provider could be replaced without
disposing the old one first, leaking a live mic track (the iOS-PWA "stuck
indicator" class). The controller wraps the existing `providerRef` (reads stay
`providerRef.current`; only sanctioned mutations go through it) and owns the
start-cancellation generation token + stale-lease recovery. The pure
`micLifecyclePolicy` helpers make the platform privacy decision and the
registry-vs-UI honesty check reviewable and unit-testable.
[DOC: docs/internal/VOICE-LATENCY.md#page-lifecycle-mic-cleanup-always-on-for-all-mic-owners]

| Component | Production | Test |
|-----------|-----------|------|
| `controller.replace(next, reason)` | Dispose previous provider (release lease) BEFORE installing next — atomic, no-op if already current | Replace, assert old disposed once + ref is next |
| `controller.shutdown(reason)` | Cancel in-flight start + dispose provider + run hook capture teardown; idempotent | Double shutdown, assert one dispose + teardown ran |
| `controller.beginStart()` / `isCurrentStart()` / `cancelStarts()` | Generation token so a late-resolving `start()` releases its lease instead of entering recording | `cancelStarts` then assert token stale |
| `controller.recoverStaleLeases({voiceState,…})` | Release orphaned leases (`selectStaleLeases`) + dispose dangling provider + log invariant violation | Register an idle `voice-stream` lease, assert released + logged |
| `decideMicLifecycle({event, standalonePwa})` | Pure: which leases release and whether active capture stops per event/platform. `visible` never arms the mic by itself | Matrix test (hidden PWA → all; desktop → non-active; visible → no release/no stop) |
| `selectStaleLeases({leases, voiceState, …})` | Pure: which live leases the workflow should not hold | Active-owner-while-idle / prewarm-off / passive-no-listener |
| `isStandaloneDisplayMode()` | `navigator.standalone` or `display-mode: standalone` | Impure detector; the decision it feeds is the pure unit |

**Key invariant**: provider cleanup is idempotent and replay-safe; an error,
fallback, cancel, unmount, hidden, pagehide, freeze, or stale-start path always
releases the mic track and never leaves the UI idle while a provider owns a live
stream. `voiceState` is workflow state; the registry is hardware truth. The old
`voiceState === "passive"` branch is retired; passive wake-word listening is
represented by `passiveListeningActive` and a `passive-wake-word` lease.

### Voice Latency — Stream Ownership Seam (UI)
**Files**: `ui/src/audio-integration/hooks/voice/micReadiness.ts`, `ui/src/audio-integration/hooks/voice/sharedAudioContext.ts`
**Purpose**: Decouple mic stream lifecycle from provider lifecycle, enabling pre-warmed streams for near-instant activation while maintaining testability.
[DOC: docs/internal/VOICE-LATENCY.md]

| Component | Production | Test |
|-----------|-----------|------|
| `micReadiness.acquireStream()` | Acquires a `low-latency-prewarm` lease via the mic ownership registry, caches it after explicit mic-control intent | Mock `getUserMedia`, verify mount/visibility do not call it and intent does |
| `micReadiness.releaseStream()` | Releases the lease (stops all tracks, resets state) | Verify `track.stop()` called |
| `sharedAudioContext.getSharedAudioContext()` | Returns singleton AudioContext | Mock constructor, assert single creation |
| `sharedAudioContext.ensureAudioContextOnGesture()` | One-shot pointerdown/keydown listener | Simulate event, verify context created |
| `VoiceStreamProvider.preConnect()` | Opens WebSocket early, 30s timeout | Mock WebSocket, verify reuse in `start()` |
| `VoiceStreamProvider.start(preWarmedStream?)` | Uses injected stream or acquires its own lease | Pass mock stream, verify no `getUserMedia` |
| `VoiceStreamProvider` lease vs injected stream | Provider releases only a stream it acquired (lease set); injected stream's lease stays with micReadiness | Inject a stream, stop, verify tracks not stopped |
| `vad.createVadRefsFromCache(cached)` | Seeds thresholds from localStorage cache | Pure function, assert `waitingForSpeech` state |
| `vad.extractCacheableFloor(vad)` | Extracts current thresholds for persistence | Pure function round-trip test |
| `api.getCapabilitiesLivenessSnapshot()` | Synchronous read of cached capabilities | Verify no network call in `startRecording()` |

**Key invariant**: Ownership is tracked by a lease, not the `retainStream` flag.
A provider holds a lease only for a stream it acquired itself; an injected
pre-warmed stream's lease stays with micReadiness. `stop()`/`dispose()` release
only the provider's own lease — never another owner's tracks.

**Key invariant**: Page-lifecycle cleanup releases passive/prewarm/settings mic
owners on hidden and ALL owners on pagehide; an active recording is stopped on
hidden (privacy). Re-arm happens only after explicit mic-control intent, never
from visibility alone.

### Audio Transcoding Seam (API)
**File**: `api/internal/audio/transcode.go`
**Purpose**: Decouple audio format conversion from transcription handlers for testable preprocessing.

| Component | Production | Test |
|-----------|-----------|------|
| `audio.Transcode` | ffmpeg stdin/stdout pipe (16kHz mono WAV) | `voice.Service.SetTranscode(...)` with no-op passthrough or tracking function |
| ffmpeg lookup | `sync.Once` + `exec.LookPath` caches ffmpeg availability | Implicitly controlled by the injected transcode function |

**Benefits**: Audio preprocessing can be tested without ffmpeg installed. Both batch (`Voice.Transcribe`) and streaming (`HandleStreamWS`) paths share the same seam. Graceful fallback to raw audio when ffmpeg is unavailable or transcoding fails.

### Voice Stream WebSocket Seam (API)
**File**: `api/internal/voice/stream_ws.go`
**Purpose**: Decouple streaming transcription from the Whisper service for testable WebSocket behavior.

| Component | Production | Test |
|-----------|-----------|------|
| `voice.HTTPDoer` | `*http.Client` passed from `api/main.go` into `voice.NewService` | Fake/httptest client through `Service.SetHTTPClient(...)` |
| `whisperURL` service field | Resolved from `WHISPER_URL` at startup | Swapped to `httptest.NewServer` URL with `Service.SetWhisperURL(...)` |
| `TranscribeBytes(ctx, url, httpClient, transcode, audio, language, doTranscode, initialPrompt)` | Optionally transcodes via the injected function, then calls Whisper through `HTTPDoer`; appends `initial_prompt` to URL when non-empty | Uses mock Whisper handler to verify payload size, language, and URL params |
| `transcode` parameter | `true` for final transcription (ffmpeg WAV), `false` for partials (raw WebM) | Track injected transcode call count per WS lifecycle — 0 for partials, 1 for final |
| `transcode` service field | `audio.Transcode` → ffmpeg 16kHz mono WAV | No-op passthrough via `Service.SetTranscode(...)` |
| `voice.Config` on `voice.Service` | Runtime-configurable struct with `FlushIntervalMs`, `MinDeltaBytes`, `OverlapBytes`. Read once per session (snapshot pattern). Backed by the scenario `api-core/storage` state path. | Tests set config on the service before dialing WS. Each test gets its own `srv` — no cleanup needed. |
| `VoiceStreamConfig` GET/PUT API | `GET /api/v1/voice/config` returns current config. `PUT /api/v1/voice/config` partial-updates, validates, persists to disk. | `TestHandleGetVoiceConfig`, `TestHandleUpdateVoiceConfig_*` use `httptest.NewRecorder` with `voiceConfigTestServer` helper. |
| `firstTick` eager bypass | On the first ticker tick, bypasses `MinDeltaBytes` gate to reduce perceived latency for the initial partial | `TestVoiceStreamWS_EagerFirstPartial` verifies sub-MinDeltaBytes audio triggers on first tick |
| Delta offset tracking | `lastPartialOffset` advances per tick; only new bytes (`buf[offset:]`) sent to Whisper | `trackingWhisperHandler` records payload sizes — each partial ≈ delta size, not full buffer |
| `initial_prompt` context | `lastNWords(previousTranscript, 10)` appended to Whisper URL for context continuity | Tracking handler captures URLs — first partial has no prompt, subsequent partials have prompt with prior words |
| `lastNWords(s, n)` | Returns last N whitespace-delimited words of string | Pure function, table-driven unit tests |
| Language passthrough | Read from WS upgrade query `?language=`; empty = Whisper auto-detect | Verify mock Whisper URL has/lacks `language=` param |
| `partialCtx` / `partialCancel` | Derived context for ticker goroutine's `transcribeBytes` calls; cancelled on recording stop to abort in-flight Whisper HTTP requests, freeing the (often single-threaded) Whisper server for the final retranscribe | `TestVoiceStreamWS_PartialCancelledOnDone` — slow mock partial is cancelled, final arrives within 3s |
| `finalCtx` (30s timeout) | Independent context for final retranscribe, derived from `context.Background()` | Decouples final transcription timeout from session recording duration; existing final transcription tests exercise this path |
| `tickerDone` WaitGroup | Ensures ticker goroutine has fully exited before final retranscribe starts — prevents race where partial and final compete for Whisper | Implicit in `TestVoiceStreamWS_PartialCancelledOnDone` sequencing |

**Benefits**: Full WebSocket streaming lifecycle (connect, binary chunks, partials, done, final) is testable without a real Whisper server. Delta-based partial transcription provides real-time UI feedback during recording. Transcode skip for partials reduces latency ~100-200ms per tick. Audio overlap improves Whisper accuracy at chunk boundaries. Full retranscription always produces the final result for maximum accuracy. Partial cancellation on recording stop prevents Whisper contention and reduces finalization latency. All pipeline parameters are runtime-tunable via `VoiceStreamConfig` (GET/PUT `/api/v1/voice/config`) and exposed in the Settings modal as "Advanced Streaming" controls.

### Speaker Verification Resource Seam (API)
**Files**: `api/speaker_verification_client.go`, `api/speaker_verification_config.go`, `api/speaker_verification_handlers.go`
**Purpose**: Decouple speaker identity verification from the `speaker-verification` resource for testable enrollment, verification gating, and config persistence.

| Component | Production | Test |
|-----------|-----------|------|
| `SpeakerVerificationResourceClient` | HTTP client calling the `speaker-verification` resource (`/v1/verify`, `/v1/extract`, `/v1/profiles`, `/ready`, `/v1/info`) | `httptest.NewServer` with canned responses — match/mismatch/error/extraction |
| `evaluateSpeakerVerification(ctx, audio)` | Calls resource `Verify()`, applies mode (`filter`/`advisory`/`off`), returns `speakerVerificationGateDecision` | Tested by injecting fake resource server into `Server.speakerVerification`; covers accept, reject, advisory pass-through, fallback on error |
| `SpeakerVerificationConfig` | Persistent JSON config: enabled, profileId, threshold, mode, rejectBehavior, fallbackWithoutVerification | Round-trip persistence, validation (threshold bounds, mode enum, required profileId), CRUD handler tests |
| `SpeakerVerificationConfigPatch.Apply()` | Partial update semantics for PUT `/api/v1/voice/speaker/config` | Handler tests verify patch merging and validation |
| Speaker-gated WebSocket flow | On segment-boundary and final: `evaluateSpeakerVerification` decides accept/reject before transcript emission | `TestVoiceStreamWS_SpeakerVerification_*` — 8 tests covering status message, accepted/rejected finals, segment accept/reject, advisory mode, disabled state, fallback policy |
| Capability detection | `speaker-verification` registered in `CapabilityRegistry` with features `voice-speaker-verification`, `voice-enrollment` | `fakeChecker` status injection in status handler test |

**Endpoints**:
- `GET /api/v1/voice/speaker/config` — current config
- `PUT /api/v1/voice/speaker/config` — partial update with validation
- `GET /api/v1/voice/speaker/status` — capability + resource health + profile state
- `GET /api/v1/voice/speaker/profiles` — list enrolled profiles
- `POST /api/v1/voice/speaker/enroll` — multipart enrollment audio → resource → config update
- `DELETE /api/v1/voice/speaker/profile` — clear active profile binding

**Benefits**: Speaker verification can be tested end-to-end through the WebSocket and HTTP layers without a running `speaker-verification` resource. The `evaluateSpeakerVerification` decision function is the single point where verification policy is applied, making it easy to test all mode/fallback combinations. Config persistence uses the same snapshot-per-session pattern as voice config.

### Speaker Verification Acceptance Decision Seam (API)
**File**: `api/speaker_verification_handlers.go`
**Purpose**: Narrow decision layer between segment audio and transcript commit.

| Component | Production | Test |
|-----------|-----------|------|
| `speakerVerificationGateDecision` struct | Captures: Enabled, Applied, Allowed, Matched, Score, Threshold, ProfileID, Mode, ErrorMessage, Extracted | Assertions on individual fields in WebSocket and HTTP transcribe tests |
| Mode routing | `filter` → reject on mismatch, `advisory` → allow + log, `off` → skip | Covered by `TestVoiceStreamWS_SpeakerVerification_RejectedFinal` (filter) and `_AdvisoryModeAllowsThrough` (advisory) |
| Fallback policy | `FallbackWithoutVerification: false` → reject on resource error; `true` → allow through | `TestVoiceStreamWS_SpeakerVerification_FallbackPolicy` and `_FallbackAllowed` |

**Benefits**: The gate decision struct decouples "what happened" (score, match, error) from "what to do" (allow/reject), keeping the WebSocket handler simple and the decision logic independently testable.

### Target Speaker Extraction Seam (API)
**File**: `api/speaker_verification_handlers.go`
**Purpose**: Isolate the enrolled speaker's voice from audio mixtures before transcription. Encapsulates the TSE decision as a single function with a clean fallback chain.

| Component | Production | Test |
|-----------|-----------|------|
| `extractTargetSpeaker(ctx, audio)` | Calls resource `/v1/extract` to separate + identify target speaker, returns cleaned audio + `speakerVerificationGateDecision` with `Extracted=true` | `TestExtractTargetSpeaker_Enabled` — mock extract endpoint returns WAV, verify extracted flag set |
| TSE disabled fallback | When `ExtractionEnabled=false`, delegates to `evaluateSpeakerVerification()` and returns original audio | `TestExtractTargetSpeaker_Disabled` — verify original audio returned, `Extracted=false` |
| TSE error fallback | When extract endpoint errors (5xx, timeout), falls back to verify-only on original audio | `TestExtractTargetSpeaker_ClientError` — mock 500 response, verify fallback to verify-only |
| `SpeakerExtractionResult` | Parsed from binary WAV response body + `X-Speaker-Score`, `X-Speaker-Matched`, `X-Duration-Ms`, `X-Audio-Seconds` headers | `TestExtract_Success` — mock server returns WAV + headers, verify parsing |
| `ExtractionEnabled` config | Independent toggle: both `Enabled` and `ExtractionEnabled` must be true for TSE | Round-trip config persistence test, patch apply test |
| WebSocket integration | Segment-final and session-final paths call `extractTargetSpeaker()` when enabled, sending cleaned audio to Whisper | `segment-accepted` message includes `extracted: true` |

**Fallback chain**:
```
TSE enabled + resource available → extract + verify extracted audio
TSE enabled + resource error    → fall back to verify-only on original audio
TSE disabled                    → verify-only on original audio (current behavior)
Verification disabled           → no gating at all (current behavior)
```

**Benefits**: TSE is independently toggleable via `extractionEnabled` config. The single `extractTargetSpeaker()` function is the only integration point, making it trivial to test the entire fallback chain. The resource `/v1/extract` endpoint returns binary WAV (not JSON-wrapped), avoiding serialization overhead when forwarding to Whisper.

### LocalEcho Clock Seam (UI)
**File**: `ui/src/lib/localEcho.ts`
**Purpose**: Decouple time-dependent prediction aging from system clock for deterministic tests.

| Component | Production | Test |
|-----------|-----------|------|
| `clock` constructor param | `Date.now` (default) | Fake clock function with `advance()` / `set()` helpers |
| Prediction aging | Predictions older than 2s auto-reset | Fake clock advances past threshold to trigger reset |
| Pending cap | Max 32 pending predictions; echoing disabled at cap | Test sends 33+ predictions to verify overflow behavior |

**Benefits**: Time-dependent local echo behavior (stale prediction reset, overflow cap) can be tested deterministically without real delays. The clock injection is a single constructor parameter with a sensible default.

### Terminal Emulator Seam (API)
**File**: `api/terminal/emulator.go`
**Purpose**: Owns the durable decoded state of a PTY's output (screen + alt-buffer + bounded scrollback). The session never inspects raw PTY bytes for replay.

| Component | Production | Test |
|-----------|-----------|------|
| `terminal.New(Options)` | Constructs an emulator sized to the session | `New(Options{Cols, Rows, ScrollbackLines})` with deterministic options |
| `Emulator.Feed([]byte)` | Called from `Session.broadcast` for every PTY read | Driven directly with crafted byte streams to assert state |
| `Emulator.Snapshot()` | Returns a self-contained ANSI replay payload | Round-trip: `New().Feed(Snapshot()) ≡ original` (`emulator_test.go`) |
| `Emulator.Resize(cols, rows)` | Called from `Session.Resize` | Asserts scrollback line count is preserved |

**Benefits**: Replay is replay-safe across alt-buffer/charset/resize transitions. The snapshot remains the recovery source of truth while the bounded cursor ring avoids retransmitting already-rendered output during short reconnects.

**Boundary**: `api/terminal/` ↔ `api/session.go` (`Subscribe()` returns the snapshot and cursor-bearing frame stream) ↔ `api/terminal_ws.go` (resume handshake, cursor frames, and snapshot fallback) ↔ `ui/src/hooks/terminal/useTerminalSession.ts` (renderer checkpoint and snapshot-mode flag).

### Combo Sequence Delay Seam (UI)
**File**: `ui/src/lib/comboSequence.ts`
**Purpose**: Decouple multi-step key combo timing from real `setTimeout` for deterministic tests.

| Component | Production | Test |
|-----------|-----------|------|
| `DelayFn` parameter | Default `setTimeout`-based delay | Synchronous `vi.fn(() => Promise.resolve())` |
| `sendComboSequence(steps, onInput, delay?)` | Sends steps with real timing gaps | Sends steps instantly with tracked delay calls |

**Benefits**: Multi-step combos (e.g., Ctrl+C ×2 with 80ms gap) can be tested without real timers. Delay call assertions verify correct timing values.

### Capability Registry Seam (API)
**File**: `api/capabilities.go`, `api/capabilities_checkers.go`
**Purpose**: Decouple capability detection from specific service health checks, enabling testable capability discovery.

| Component | Production | Test |
|-----------|-----------|------|
| `StatusChecker` interface | `ResourceChecker` pings Whisper at localhost:8090 | httptest server returning configurable status |
| `CapabilityRegistry` | Caches results with 30s TTL | Inject mock checkers, verify caching behavior |

**Benefits**: New capabilities can be added by implementing `StatusChecker` and registering in the registry. Tests verify caching and fallback without network calls.

### Backend Registry Seam (API)
**File**: `api/backend_registry.go`
**Purpose**: Decouple session creation from specific PTY backends. The registry maps BackendID to PTYFactory, enabling tests to register mock backends without tmux.

| Component | Production | Test |
|-----------|-----------|------|
| `BackendRegistry` | `InitDefaultRegistry()` registers standard + persistent | `NewBackendRegistry()` with custom descriptors |
| `checkTmuxAvailable` | Function var probing real tmux binary | Override to return controlled availability |
| `BackendDescriptor` | Contains real availability + reason | Controllable available/reason fields |

### Session Metadata Store Seam (API)
**File**: `api/session_store.go`
**Purpose**: Decouple session persistence from SQLite. Tests use `InMemorySessionStore` for fast, isolated metadata CRUD without database setup.

| Component | Production | Test |
|-----------|-----------|------|
| `SessionMetadataStore` | `SQLSessionStore` (SQLite) | `InMemorySessionStore` (map-based) |
| `SessionManager.store` | Set via `SetStore()` | Set to `InMemorySessionStore` or nil |

### tmux Discovery Seam (API)
**File**: `api/pty_tmux.go`
**Purpose**: `DiscoverTmuxSessions()` and `tmuxAttach()` are standalone functions that can be tested with controlled tmux state or mocked for recovery tests.

## Text-to-Speech Provider Seam

**Location (frontend):** `ui/src/hooks/tts/types.ts` (`TTSProvider` interface)
**Location (backend):** `api/tts_synthesize.go` (`TTSSynthesizer` interface), `api/tts_voices.go` (`TTSVoiceLister` interface)

### Frontend Provider Pattern

The `TTSProvider` interface enables swapping between synthesis backends:

| Implementation | Module | When used |
|---------------|--------|-----------|
| `KokoroProvider` | `hooks/tts/KokoroProvider.ts` | Runtime backend decision resolves to Kokoro |
| `BrowserTTSProvider` | `hooks/tts/BrowserTTSProvider.ts` | Runtime backend decision resolves to browser speech synthesis or auto-mode fallback |

**Provider selection** is handled by `useTextToSpeech` hook:
1. Read server-backed preference from `/api/v1/tts/config`
2. Check `kokoro-tts` capability via `/api/v1/capabilities/liveness`
3. Resolve the runtime backend:
   - `auto`: prefer Kokoro, else browser
   - `kokoro`: strict Kokoro only
   - `browser`: strict browser only
4. Expose both the active backend and a human-readable `backendReason`

**Fallback**: If `backend=auto` and `KokoroProvider` fails at runtime, both `speak()` and `speakParagraphs()` attempt a best-effort fallback to `BrowserTTSProvider`.

**Testing**: Replace provider instance in tests with mock implementing `TTSProvider`. No global mocking needed.

### Backend Seams

**Internal TTS service boundary** (`api/internal/tts/types.go`, `api/internal/tts/service.go`, wired by `api/tts_adapter.go`):
- Production: `internal/tts.Service` implements `internal/tts.HandlerService`; `handlers/tts` aliases that contract for Connect-RPC, and `newTTSAdapter(*Server)` adapts current web-console config stores, status callbacks, capability lookup, cache, synthesizer, and voice lister into `internal/tts.Deps`.
- Purpose: TTS config validation, summarize config validation, status assembly, synthesis normalization, cache lookup, and voice listing behavior live outside `package main`.
- Test: `api/internal/tts/service_test.go` covers core service behavior directly; handler tests still drive the Connect handler contract.

**`TTSSynthesizer` interface** (`api/internal/tts/kokoro_synthesize.go`, aliased by `api/tts_synthesize.go`):
- Production: `internal/tts.KokoroSynthesizer` — proxies to Kokoro-FastAPI `/v1/audio/speech`
- Test: Mock returning `io.ReadCloser` with test audio bytes
- Injected through `internal/tts.Deps.SynthesizeAudio`

**`TTSVoiceLister` interface** (`api/internal/tts/kokoro_voices.go`, aliased by `api/tts_voices.go`):
- Production: `internal/tts.KokoroVoiceLister` — proxies to Kokoro-FastAPI `/v1/audio/voices`
- Test: Mock returning `[]TTSVoice` slice
- Injected through `internal/tts.Deps.ListVoiceCatalog`

### Capability Gating

**`KokoroChecker`** (`capabilities_checkers.go`):
- Liveness: GET `/v1/audio/voices` → 200
- Full check: POST `/v1/audio/speech` with test text, verify non-empty audio response
- Cached by `CapabilityRegistry` with 30s TTL

### Hook Delivery Chain

**Path**: `web-console hooks register` → `claude-code` resource hook reconciliation → Claude Code Stop hook in repo-root `.claude/settings.json` / transcript sources → conversation append/fan-out → WebSocket `conversation_event` → UI `useTerminalSession` → `TerminalPane` append + received/seen ack → `useTtsPlaybackController.handleIncomingEvent` → `TerminalPaneHandle.speakText` → `useTextToSpeech.speakParagraphs` → conversation playback ack/cursor update

**Seam points**:
1. `web-console hooks register` ↔ `claude-code` resource: scenario declares desired hook; resource owns settings-path resolution, JSON merge, and idempotent healing
2. Claude Stop hook ↔ API: HTTP POST with `X-Hook-Token` auth header
3. `routeTTSCandidate` ↔ source adapters: backend routing only accepts explicit terminal ownership; it does not infer from PTY output
4. Conversation store/fan-out ↔ WebSocket: normalized events broadcast to session subscribers
5. `useTerminalSession` ↔ `TerminalPane`: client receives `conversation_event` and emits delivery/playback acknowledgements
6. `TerminalPane` ↔ `useTtsPlaybackController`: TerminalPane owns provider plumbing; controller owns auto-play intent, queue, selected target, and listened commits
7. `useTextToSpeech` ↔ `TTSProvider`: injectable Kokoro/Browser implementations

**Testing**: `tts_hook_handler_test.go` covers Claude session mapping. `tts_router_test.go` covers candidate routing/dedup. `codex_tailer_test.go` includes an E2E test from rollout file → owning terminal candidate. `mid_session_conversation_test.go` locks in that attribution holds for sessions started as plain shells (no shortcut) and that unattributed payloads cannot bleed into other sessions. User-facing contract: [guides/CONVERSATION_TRACKING.md](../guides/CONVERSATION_TRACKING.md).

### Two Independent TTS Trigger Paths

1. **Claude Code Hook** (`web-console hooks dispatch` → `handleHookStop`): Active push. Claude Code fires a Stop hook after each response. Web-console registers a portable Go command hook, so the terminal environment can inject `WC_WEB_CONSOLE_SESSION_ID` directly into the payload. Claude keeps its native shared `~/.claude` session storage unchanged, so sign-in and onboarding state are preserved.
2. **CodexTailer** (`codex_tailer.go`, `internal/tailer`): Passive poll. Watches each terminal session's dedicated `CODEX_HOME/sessions/` tree and extracts assistant text. Each terminal gets a prepared `CODEX_HOME` overlay only when Codex is launched: shared auth/config and regenerable runtime state resolve to `~/.codex`, while rollout/session data remains terminal-owned. Rollout ownership is therefore explicit from the filesystem path, not inferred from text.

Both paths converge as normalized conversation events. Frontend auto-play is gated by `autoTtsEnabled`, active pane ownership, assistant role, and persisted playback intent. The client reports delivery and playback outcomes over conversation-event acknowledgements.

**Dedup cache** (`ttsDedup` in `tts_router.go`): routing uses a time-bounded event-identity cache keyed from `source + session + cleaned text`. Entries expire after `ttsDedupTTL` (30s). The `ttl` field is injectable for testing.

**ANSI stripping**: `routeTTSCandidate()` strips ANSI before publishing the candidate so browser correlation and playback always operate on speakable text.

**`staleTimeout` injectable field** (`codex_tailer.go`): The `CodexTailer.staleTimeout` field overrides the default `codexStaleTimeout` (1 hour) for testing. When non-zero, `tailFile()` uses this value for the stale timer. Tests use short values (100ms) to verify watcher cleanup without waiting an hour.

## Boundary Violations Fixed

### Phase 2 (2026-02-19) — Responsibility Boundaries
| Violation | Before | After |
|-----------|--------|-------|
| WebSocket protocol in TerminalPane | TerminalPane mixed xterm.js rendering with WS protocol | Extracted to `useTerminalSession` hook |
| Data formatting in SessionDrawer JSX | Inline `split("/").pop()`, `toLocaleTimeString()` | Extracted to `lib/format.ts` utilities |
| setTimeout shortcut injection | `setTimeout(500)` timing assumption in Workspace | Event-driven `onReady` callback from TerminalPane |
| ANSI escape codes scattered | Hardcoded `\x1b[90m` in TerminalPane | Centralized `ANSI` constants in useTerminalSession |
| Implicit onExit callback | `readLoop(onExit func(string))` mutated SessionManager | `exitCh` channel; SessionManager listens on `Done()` |
| Silent JSON decode errors | `_ = json.Decode()` in handler | Logged with `log.Printf` for debugging |

### Phase 3 (2026-02-19) — Seam Discovery & Enforcement
| Violation | Before | After |
|-----------|--------|-------|
| PTY creation hardcoded in SessionManager | `exec.Command` + `pty.StartWithSize` inline in `Create()` | `PTYFactory` function type; `defaultPTYFactory` in production |
| Session held raw `*os.File` + `*exec.Cmd` | `s.ptmx.Write()`, `s.cmd.Process.Kill()` | `s.pty.Write()`, `s.pty.Kill()` via PTY interface |
| Resize bypassed Session encapsulation | `pty.Setsize(sess.ptmx, ...)` in SessionManager | `sess.pty.SetSize(cols, rows)` via PTY interface |
| WebSocket hardcoded in hook | `new WebSocket(url)` in useEffect | `createSocket(url)` with injectable factory |
| ANSI constants private | `const ANSI` not exported | `export const ANSI` for test verification |

### Phase 8 (2026-02-19) — Change Axis & Evolution Resilience
| Change | Before | After |
|--------|--------|-------|
| Shortcut data in component | Hardcoded `DEFAULT_SHORTCUTS` array in `TerminalLauncher.tsx` | Extracted to `consts/shortcuts.ts`; component imports from data module |
| Error banner duplicated | Two inline error banner implementations in Workspace.tsx (empty + active state) | Single `ErrorBanner.tsx` component used in both states |
| Session logic in layout | Workspace.tsx mixed pane layout with session CRUD, ref management, error state | Extracted `useSessionManager` hook; Workspace is pure layout |
| No variation tests | Tests validated specific values only | Added structural invariant tests (`TestErrorCatalog_StructuralInvariants`, `TestSessionLimit_VariousLimits`, `shortcuts.test.ts`) |

### Workspace Store Seam (API + UI)
**Files**: `api/workspace_store.go` (interface), `api/workspace_store_pg.go` (production), `api/workspace_store_mem.go` (test), `ui/src/hooks/useWorkspaceSync.ts` (client)
**Purpose**: Decouple workspace layout persistence from HTTP handlers and UI components for cross-device sync with testable storage.

| Component | Production | Test |
|-----------|-----------|------|
| `WorkspaceStore` interface | `SQLWorkspaceStore` backed by SQLite | `MemWorkspaceStore` backed by in-memory maps |
| `workspace_panes` table | `ON DELETE CASCADE` from sessions; `ON DELETE SET NULL` from tab_groups | Same semantics in `MemWorkspaceStore` |
| `useWorkspaceSync` (UI) | Fire-and-forget API calls with debounced reorder saves | Mock API functions in tests |
| Tab groups | `tab_groups` table with sort_order and collapse state | In-memory group map with UUID generation |

**Benefits**: Workspace layout (pane order, tab groups, active pane) syncs across devices via SQLite. UI remains snappy via optimistic Zustand updates. In-memory store enables handler tests without database. Tab groups support is built into the data model from the start.

## Remaining Ownership Issues

1. ~~**Shortcut defaults hardcoded** in `TerminalLauncher.tsx`~~ — **Resolved Phase 8**: Extracted to `consts/shortcuts.ts`
2. ~~**No reconnect logic**~~ — **Resolved**: `useTerminalSession` now auto-reconnects with exponential backoff (max 5 attempts) and defers reconnection when the tab is backgrounded via `visibilitychange` listener
3. ~~**No session persistence**~~ — **Resolved**: Workspace pane metadata persisted in SQLite `workspace_panes` table with cross-device sync via `WorkspaceStore` interface
4. **No structured logging** — Simple `log.Printf` across API; should use structured logger at integration boundaries
5. ~~**API client hardcoded in Workspace**~~ — **Resolved Phase 8**: Session lifecycle extracted to `useSessionManager` hook
6. **Drop detection as transport concern** — Per-client frame drop counting with configurable threshold (`WC_DROP_NOTIFY_THRESHOLD`) sends `sync_warning` messages to affected clients

## Change Axes

Primary axes of change identified in Phase 8, with current cost assessment and structural adjustments.

### Axis 1: Shortcut Profiles (P0-006, P1-010)
**What changes**: Adding/removing/modifying launch shortcuts, config-driven profiles
**Cost before Phase 8**: Medium — shortcuts hardcoded in `TerminalLauncher.tsx` component mixed with UI rendering
**Cost after Phase 8**: Low — all shortcut data lives in `consts/shortcuts.ts`, component only renders from props
**Files to touch**: `consts/shortcuts.ts` (data), optionally `TerminalLauncher.tsx` (if UI changes needed)
**Test coverage**: `shortcuts.test.ts` validates structural invariants (uniqueness, non-empty, PRD compliance)

### Axis 2: Toolbar Keys & Key Combos (P0-007)
**What changes**: Adding/removing mobile toolbar keys, escape sequences, or key combos
**Cost**: Already low — `TOOLBAR_KEYS` in `consts/toolbar-keys.ts` and `KEY_COMBOS` in `consts/key-combos.ts` are declarative arrays
**Files to touch**: `consts/toolbar-keys.ts` (single keys), `consts/key-combos.ts` (multi-step combos)
**Test coverage**: `toolbar-keys.test.ts` validates escape sequences; `key-combos.test.ts` validates combo definitions and search filter

### Axis 3: Error Codes & Recovery (API + UI)
**What changes**: Adding new error types, adjusting recovery hints, new categories
**Cost**: Low — API: add entry to `errorCatalog` map in `session_handlers.go`; UI: `ErrorBanner.tsx` renders any `ErrorInfo` shape
**Files to touch**: `session_handlers.go` (catalog entry), optionally `useTerminalSession.ts` (WS recovery)
**Test coverage**: `TestErrorCatalog_StructuralInvariants` validates all entries have valid category, message, recovery, status. `TestWriteJSONError_UnknownCode_Fallback` verifies graceful degradation for new codes.
**Invariant**: Unknown codes fall back to `internal` category with generic recovery hint.

### Axis 4: Session Policies (P1-001)
**What changes**: Adding config knobs (env vars), new policy limits, expiration behavior
**Cost**: Low — `config.go` centralizes all tunables with env var mapping and validation/clamping
**Files to touch**: `config.go` (add field + env var), `session.go` (apply policy)
**Test coverage**: `config_test.go` covers defaults, env overrides, clamping, invalid fallback. `TestSessionLimit_VariousLimits` validates limit behavior across multiple values.

### Axis 5: WebSocket Protocol (P0-002b)
**What changes**: Adding message types, changing framing, adjusting handshake
**Cost**: High (inherently coupled) — requires coordinated changes in `terminal_ws.go` and `useTerminalSession.ts`
**Files to touch**: `terminal_ws.go` (server), `useTerminalSession.ts` (client), both message type definitions
**Mitigation**: Message types are string constants on both sides; `TerminalMessage` interface/struct serves as the protocol contract. Adding new types is additive and backward-compatible.

### Axis 6: Terminal Appearance
**What changes**: Theme colors, font stack, font size
**Cost**: Low — all values centralized in `consts/config.ts`, consumed by `TerminalPane.tsx` only
**Files to touch**: `consts/config.ts`
**Test coverage**: `config.test.ts` validates exports

### Axis 7: Pane Layout
**What changes**: Grid behavior, min dimensions, column logic
**Cost**: Low — grid constants in `consts/config.ts`, grid CSS logic isolated in `Workspace.tsx`
**Files to touch**: `consts/config.ts` (constants), `Workspace.tsx` (grid style block)

### Stable Core vs Volatile Edges Summary

| Module | Stability | Notes |
|--------|-----------|-------|
| `session.go` / `SessionManager` | **Stable core** | PTY lifecycle, subscribe/broadcast — unlikely to change shape |
| `pty.go` / PTY interface | **Stable core** | Abstraction boundary — changes only if PTY API changes |
| `terminal_ws.go` / WS protocol | **Stable core** | Message framing — additive changes only |
| `main.go` / server wiring | **Stable core** | Router + middleware — rarely touched |
| `useTerminalSession.ts` | **Stable core** | WS lifecycle hook — additive message types only |
| `useSessionManager.ts` | **Stable core** | Session orchestration — change when API contract changes |
| `TerminalPane.tsx` | **Stable core** | xterm.js rendering — change only for terminal feature additions |
| `consts/shortcuts.ts` | **Volatile edge** | Shortcut definitions — expected to change with profiles (P1-010) |
| `consts/config.ts` | **Volatile edge** | UI tunables — expected to grow with new features |
| `config.go` | **Volatile edge** | API tunables — expected to grow with new policies (P1-001) |
| `errorCatalog` (session_handlers.go) | **Volatile edge** | Error definitions — grows with new error paths |
| `TOOLBAR_KEYS` (consts/toolbar-keys.ts) | **Volatile edge** | Single-key definitions — grows with new mobile keys |
| `KEY_COMBOS` (consts/key-combos.ts) | **Volatile edge** | Multi-step combo definitions — grows with new terminal key combos |
| `ErrorBanner.tsx` | **Volatile edge** | Error display — changes with new recovery UX |
| `Workspace.tsx` | **Mixed** | Layout is stable; wiring to hooks is stable; grid behavior may evolve |

## Decision Points

Phase 14 extracted the following decision points into named, testable helpers. Each entry identifies **where the decision is made** and **what it decides**.

### Extracted Decision Helpers (API)

| Helper | File | Decision | Inputs |
|--------|------|----------|--------|
| `classifyCreateError(err)` | `session_handlers.go` | Maps session creation errors (limit, PTY, unknown) to HTTP error codes and categories | `error` sentinel type |
| `applySessionDefaults(shell, cols, rows)` | `session.go` | Substitutes configured defaults for zero/empty caller values | Zero-value convention |
| `isSessionLimitReached()` | `session.go` | Whether a new session should be rejected (MaxSessions cap; 0 = unlimited) | Session count, config |
| `buildPolicyResponse(sess, policy)` | `session_policy.go` | Constructs policy response with TTL/expiry fields (0 = never-expire) | Session creation time, policy mode |
| `resolveShell()` | `config.go` | Full shell fallback chain: `WC_DEFAULT_SHELL` → `$SHELL` → `/bin/sh` | Environment variables |
| `checkProviderResponse(resp, name)` | `ai_generate.go` | Whether an AI provider HTTP response indicates success (200) or failure | HTTP status code |
| `extractCommand(raw)` via `knownCodeFences` | `ai_generate.go` | Which markdown fences to strip from AI output (bash, sh, generic only) | Raw AI text |
| `customDurationMin` / `customDurationMax` | `session_policy.go` | Allowed range for custom policy durations (1m–7d) | Named constants |

### Extracted Decision Helpers (UI)

| Helper | File | Decision | Inputs |
|--------|------|----------|--------|
| `isCleanWsClose(code)` | `useTerminalSession.ts` | Whether a WebSocket close is intentional (1000/1001) vs. unexpected | Close code |

### Decision Groupings by Domain

| Domain | Where Decisions Live | Key Functions |
|--------|---------------------|---------------|
| **Session lifecycle** | `session.go` | `applySessionDefaults`, `isSessionLimitReached`, `Create`, `Delete` |
| **Session policy** | `session_policy.go` | `ValidatePolicy`, `ResolveTTL`, `IsExpired`, `buildPolicyResponse` |
| **Error classification** | `session_handlers.go` | `classifyCreateError`, `writeJSONError`, `errorCatalog` |
| **AI command extraction** | `ai_generate.go` | `extractCommand`, `knownCodeFences`, `checkProviderResponse` |
| **Configuration** | `config.go` | `resolveShell`, `envInt`, `LoadConfig` |
| **WebSocket transport** | `terminal_ws.go` (server), `useTerminalSession.ts` (client) | `isCleanWsClose`, WS message dispatch |

### Well-Extracted vs. Still-Scattered

**Well-extracted** (each has a single named "home"):
- Error classification → `classifyCreateError` + `errorCatalog`
- Policy response construction → `buildPolicyResponse`
- Shell resolution → `resolveShell`
- AI output cleaning → `extractCommand` with explicit `knownCodeFences`
- WS close classification → `isCleanWsClose`
- Session defaults → `applySessionDefaults`
- Session limit check → `isSessionLimitReached`

**Still scattered** (documented, not yet refactored):
- `parseDurationMs` (UI) vs `time.ParseDuration` (Go): two separate duration parsers with different capabilities; Go accepts `1h30m`, UI only handles single-unit `1h`/`30m`/`45s`
- WebSocket origin check: `CheckOrigin: func(r *http.Request) bool { return true }` — security decision expressed as always-true with comment justification only

## Architecture Clarity Notes

### Phase 15 — Cognitive Load Reduction (2026-02-19)

**Major simplifications:**
- **Error writing consolidation**: Renamed `writeJSONError(w, status, code, message)` → `writeCatalogError(w, code, message)`. The old function accepted a `status` parameter that always matched the catalog entry, creating a confusing dual-source pattern. The new function derives status entirely from the catalog, reducing the reader's mental burden from "which source wins?" to "catalog owns the contract".
- **Duration/countdown co-location**: Moved `parseDurationMs()` and `formatCountdown()` from `SessionDrawer.tsx` (where they were scattered among UI components) into `lib/format.ts` (where all formatting utilities live). Reading SessionDrawer no longer requires mentally separating formatting math from layout logic.
- **AiInput Enter key clarity**: Renamed `handleKeyDown` → `handleEnterKey` with explicit comment documenting the two-phase behavior (generate vs execute). Added a named `hasGeneratedCommand` variable to make the state-machine decision self-documenting.
- **WS teardown comment**: Added inline explanation of the `select/default` pattern in the output forwarder's `writerDone` close, making the "close-only-once" invariant immediately visible.

**Complexity hotspots remaining:**
- `SessionDrawer.tsx` still mixes the `useCountdown` hook (timing logic) with session list rendering — could be extracted to a standalone hook but the coupling is currently acceptable.
- AI input state machine has a 3-handler mutation surface (`onChange`, `handleEnterKey`, `handleGenerate`), but the states are now explicitly named.

## Architecture Alignment

### Phase 17 — Screaming Architecture Audit (2026-02-19)

**Mental Model**: Web Console is a browser terminal with 6 domain concepts:
1. Session Management (PTY-backed terminal sessions)
2. Terminal I/O (WebSocket bridge)
3. AI Command Generation (NL → shell command)
4. Shortcut Profiles (configurable launch shortcuts)
5. Observability (events, metrics)
6. Configuration (tunables)

**Logical → Physical Alignment**:

The API uses a hybrid organization:
- **Cross-cutting infrastructure** (`errors.go`) owns the error catalog, types, and HTTP error helpers — all handler files depend on this single source.
- **Core domain (sessions)** is layer-split: `session.go` (domain) + `session_handlers.go` (HTTP). Justified by being the largest feature.
- **Feature modules** (AI, shortcuts, metrics) are feature-sliced: each file owns domain + HTTP together.
- **AI generation** (`ai_generate.go`) owns the full pipeline including config-aware orchestration (`generateWithConfig`). Config storage/health lives in `ai_provider_config.go`.
- **Policy** is a special case: domain logic in `session_policy.go`, but HTTP handlers in `session_handlers.go` (because they're session sub-resource endpoints on `/sessions/{id}/policy`).

**Gaps fixed in iteration 1:**
| Gap | Before | After |
|-----|--------|-------|
| Policy handlers in wrong file | `session_policy.go` mixed domain logic with HTTP handlers for `/sessions/{id}/policy` | Handlers moved to `session_handlers.go`; `session_policy.go` is pure domain |
| Wrong error code for shortcut deletion | `handleDeleteShortcutProfile` returned `session_not_found` error code | New `profile_not_found` error code in catalog with correct recovery hint |
| Misplaced docs | `docs/PROBLEMS.md` and `docs/PROGRESS.md` duplicated at root and `docs/internal/` | Removed root copies; `docs/internal/` is canonical |
| RESEARCH.md not in standard location | `docs/RESEARCH.md` outside `internal/` | Moved to `docs/internal/RESEARCH.md`; manifest updated |
| File Map incomplete | Architecture doc listed only 19 files | Expanded to 35 files covering all API modules, UI pages, hooks, and utilities |
| No organizational pattern documented | Readers couldn't tell if feature-slicing was intentional | Added "Code Organization Pattern" section to ARCHITECTURE.md |

**Gaps fixed in iteration 2:**
| Gap | Before | After |
|-----|--------|-------|
| Error infrastructure in session_handlers.go | `session_handlers.go` mixed error catalog + types + helpers with session HTTP handlers (god file, 405 lines, 5+ concerns) | Extracted to `errors.go`; session_handlers.go is now pure session HTTP handlers |
| `generateWithConfig` misplaced in config file | `ai_provider_config.go` owned generation orchestration logic that only `ai_generate.go` called | Moved to `ai_generate.go`; config file owns only storage + health tracking |
| Duplicate POLICY_OPTIONS (UI) | Identical `POLICY_OPTIONS` array defined in both `SessionDrawer.tsx` and `SessionsPage.tsx` | Extracted to `consts/policy-options.ts`; both components import from single source |
| Duplicate countdown logic (UI) | `useCountdown` hook in `SessionDrawer.tsx` + `PolicyCountdown` component in `SessionsPage.tsx` duplicated same timer logic | Extracted shared `hooks/useCountdown.ts`; both consumers use the single hook |
| Inline ID truncation (UI) | `SessionsPage.tsx` used `session.id.slice(0, 8)` instead of existing `truncateId()` utility | Now uses `truncateId()` from `lib/format.ts` consistently |
| `policyKey` helper duplicated | Inline in `SessionDrawer.tsx` + repeated pattern in `SessionsPage.tsx` | Exported from `consts/policy-options.ts`; both components share it |

**Remaining drift** (documented, not addressed):
- Feature-sliced files (`shortcut_profiles.go`, `ai_provider_config.go`) could benefit from handler extraction if they grow larger, but current size doesn't warrant the split
- `ai_generate.go` contains both provider implementations (Ollama, OpenRouter) and the HTTP handler; if more providers are added, extracting providers into separate files would improve clarity

## Observability Surface

### Phase 20 (2026-02-19) — Signal & Feedback Surface Design

#### Key Observable States

| State | Where Surfaced | Signal |
|-------|---------------|--------|
| Server healthy / degraded | `GET /health`, `GET /api/v1/metrics` | Health endpoint returns DB status; metrics show uptime and counters |
| Session created / active | `GET /api/v1/sessions`, `GET /api/v1/events`, UI session list | Session list, `session.created` event, `ActiveSessions` metric |
| Session connected (WS) | `GET /api/v1/events`, `GET /api/v1/metrics` | `session.connected` event, `ActiveConnections` metric |
| Session exited | WS `exit` message (with real exit code), `[EVENT]` log, UI terminal text | Exit code forwarded to client; red text for non-zero, gray for clean exit |
| Session expired (policy) | `[EVENT]` log, `GET /api/v1/events` | `session.terminated` event with reason/policy/duration details |
| Policy updated | `[EVENT]` log, `GET /api/v1/events` | `session.policy_updated` event (named constant, not inline string) |
| AI generation attempted | `[EVENT]` log, `GET /api/v1/events`, `GET /api/v1/ai/health` | `ai.generate` event with provider name; per-provider health tracking |
| AI provider failure | API log, `GET /api/v1/ai/health`, UI error in AiInput | Per-provider error count/rate; structured error with recovery hint |
| Offline buffer overflow | API log (once per session) | `session %s: offline buffer full` — de-duplicated, one-shot |
| Policy update failure | UI inline error banner in SessionDrawer | Red banner with recovery hint, auto-dismiss 5s |
| Integrations panel failure | UI inline error in IntegrationsPanel | Red text with error message |
| TTS backend decision | Settings -> Voice Output (TTS) | Active backend + `backendReason` |
| Claude hook registration | Settings -> Voice Output (TTS), `GET /api/v1/tts/status` | `hookRegistered`, `hookReason`, settings file path |
| Last auto-TTS delivery | Settings -> Voice Output (TTS), API logs | `lastDelivery` object + `tts-delivery:` log line |
| Browser audio lockout | Settings -> Voice Output (TTS), terminal banner | `Browser audio: Blocked until you interact with the page` |

#### Signal Inventory

**HTTP Endpoints (observability-focused):**
| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Service + DB readiness |
| `GET /api/v1/metrics` | Atomic counters: sessions, connections, messages, AI, uptime |
| `GET /api/v1/events?limit=N` | Recent structured events from in-memory ring buffer (default 50, max 1000) |
| `GET /api/v1/ai/health` | Per-provider availability, latency, error rate |
| `GET /api/v1/tts/status` | Hook registration, last delivery result, Kokoro liveness, persisted TTS config |

**Structured Events** (all emitted via `EventLogger`, logged as `[EVENT] {json}`):
| Constant | Value | Details |
|----------|-------|---------|
| `EventSessionCreated` | `session.created` | shell, cols, rows |
| `EventSessionConnected` | `session.connected` | — |
| `EventSessionDisconnected` | `session.disconnected` | — |
| `EventSessionTerminated` | `session.terminated` | reason, policy, duration |
| `EventSessionDeleted` | `session.deleted` | — |
| `EventPaneResized` | `pane.resized` | cols, rows |
| `EventAIGenerate` | `ai.generate` | provider, prompt |
| `EventSessionPolicyUpdate` | `session.policy_updated` | mode, duration |
| `EventWorkspaceLayoutUpdated` | `workspace.layout_updated` | active_pane |
| `EventPaneUpdated` | `pane.updated` | name |
| `EventTabGroupCreated` | `group.created` | group_id, name |
| `EventTabGroupUpdated` | `group.updated` | group_id, name |
| `EventTabGroupDeleted` | `group.deleted` | group_id |

**WebSocket Protocol Signals:**
| Message | Direction | Signal |
|---------|-----------|--------|
| `exit` | Server→Client | Process exited; `code` field carries real exit code (0=clean, non-zero=failure) |
| `error` | Server→Client | Runtime error with known recovery hints for common cases |
| `pong` | Server→Client | Keepalive response confirming connection liveness |
| `resize_info` | Server→Client | Informational: reports effective PTY size after resize (may differ from requested if other clients are larger) |
| `size_info` | Server→Client | Authoritative shared grid plus leader/lease state for all viewers |
| `take_lease` | Client→Server | Explicit transfer of terminal size authority to this connection |
| `sync_warning` | Server→Client | Coalescing notification: `coalesced_frames` count indicates output frames merged due to slow consumption (data is preserved, not lost) |

**UI Feedback Surfaces:**

**Size Lease seam:** `api/session/sizelease.go` owns declared sizes and the
single authoritative PTY resize path; `ui/src/hooks/terminal/useTerminalSession.ts`
owns application of `size_info` and the Take over command. Device labels are
recognition-only client assertions and are never authorization inputs.
| Component | Signal Type | Behavior |
|-----------|------------|----------|
| App startup | Loading spinner | "Connecting to API..." with 3 retries, then error page with retry button |
| Session creation | ErrorBanner | Structured error with category, recovery hint, retry button; auto-dismiss 8s |
| Terminal exit | ANSI text | Gray "[Session ended]" for code 0; red "[Session ended with exit code N]" for non-zero |
| WS disconnect | ANSI text | Gray "[Disconnected]" for clean close; red "[Connection lost]" with recovery guidance |
| Policy update failure | Inline banner in drawer | Red alert with recovery hint, auto-dismiss 5s, dismissible |
| Provider panel failure | Inline warning | Amber warning showing error message |
| AI generation | Loading spinner + error | Spinning icon during generation; inline error with message on failure |

#### Remaining Signal Debt

1. **No event stream endpoint** — Events are polled via `GET /api/v1/events`. An SSE or WebSocket-based real-time event stream would enable live dashboards without polling. Low priority for single-user.
2. **No structured logging** — API uses `log.Printf` (text). A structured logger (slog) would enable machine-parseable log aggregation. Documented in PROBLEMS.md, deferred.
3. **No Prometheus/OpenTelemetry** — Metrics are JSON-only poll. External observability integration is a future concern.
4. ~~**WebSocket reconnect**~~ — **Resolved**: Auto-reconnect with exponential backoff + visibility-aware deferral in `useTerminalSession`.
5. **Session delete from UI** — No confirmation feedback beyond the session disappearing from the list. Low priority.

## Audio Extraction Prep — Domain Boundary Seams (2026-05-16)

These rows capture the audio-tools extraction-prep state. Each row identifies a
seam currently in `web-console/api/internal/{voice,tts,audio}`. Future
`scenarios/audio-tools` will own implementations behind the same interfaces;
the future web-console adopter will swap the local implementations for
`audio-tools` clients without touching orchestration code.

### TTS HandlerService Seam (API)
- **File**: [CODE: scenarios/web-console/api/internal/tts/types.go]
- **Interface**: `inttts.HandlerService` — Connect-RPC TTS handler depends on
  this. Production impl is `inttts.Service` constructed via `newTTSAdapter` in
  `api/tts_adapter.go`.
- **Test substitution**: pass a fake `HandlerService` to
  `handlers/tts.NewConnectHandler` — `handlers/tts/connect_handler_test.go`
  patterns apply.
- **Maturity**: Stable. Phase 4 will add a `TextToSpeech` *capability* port on
  top to decouple orchestration from this transport-shaped contract.

### TTS Synthesizer / VoiceLister Seam (API)
- **Files**: [CODE: scenarios/web-console/api/internal/tts/service.go],
  [CODE: scenarios/web-console/api/internal/tts/kokoro_synthesize.go]
- **Interface**: `Deps.SynthesizeAudio`, `Deps.ListVoiceCatalog` —
  function-pointer seams the Service uses for synthesis and voice enumeration.
- **Production impl**: `KokoroSynthesizer.Synthesize`,
  `KokoroVoiceLister.ListVoices`, wired through `tts_adapter.go`.
- **Test substitution**: in-package tests construct `inttts.Deps` with
  closures.
- **Maturity**: Stable. Future audio-tools will provide the
  synthesize/list-voices HTTP/Connect clients behind the same Deps shape.

### TTS Text Pipeline Seam (API)
- **Files**: [CODE: scenarios/web-console/api/internal/tts/normalizer.go],
  [CODE: scenarios/web-console/api/internal/tts/chunker.go]
- **Functions**: `NormalizeTextForSpeech`, `SplitIntoSpeechParagraphs`,
  `TTSMaxChunkLength`. Pure functions, no Server state.
- **Callers**: `conversation_router.go`, `conversation_adapter.go`,
  `conversation_store.go`, `tts_summarization_service.go` — all via the
  `inttts.` package qualifier.
- **Boundary enforced by**: `TestGreenfield_TTSReusableCoreNotInPackageMain`.
- **Maturity**: Stable; ready for verbatim extraction into audio-tools.

### Voice HandlerService Seam (API)
- **File**: [CODE: scenarios/web-console/api/internal/voice/types.go]
- **Interface**: `intvoice.HandlerService` — Connect-RPC Voice handler depends
  on this. Production impl is `intvoice.Adapter`, wrapping a `Backend`
  (`intvoice.Service` in production).
- **Test substitution**: pass a fake `HandlerService` to
  `handlers/voice.NewConnectHandler`; or pass a fake `Backend` to
  `intvoice.Adapter{}` for higher-fidelity orchestration tests.
- **Maturity**: Stable as of 2026-05-16 Phase 3 inversion. Previously these
  types lived in `handlers/voice` and `internal/voice` imported the handler
  package — that direction has been reversed.

### Voice Backend (Storage/State) Seam (API)
- **Files**: [CODE: scenarios/web-console/api/internal/voice/types.go],
  [CODE: scenarios/web-console/api/internal/voice/service.go]
- **Interface**: `intvoice.Backend` — the Adapter's storage/state seam
  covering Whisper capability checks, speaker evaluation, stream/speaker/
  wakeword config persistence, and the speaker resource client.
- **Production impl**: `intvoice.Service` — owns config paths, in-memory
  state, the `SpeakerClient`, and the Whisper `HTTPDoer`.
- **Test substitution**: fakes implement the 22 Backend methods; see existing
  speaker_extraction tests and `voice_test_helpers_test.go` patterns.

### Audio Transcoder Seam (API)
- **File**: [CODE: scenarios/web-console/api/internal/audio/transcode.go]
- **Function**: `Transcode(ctx, audio) ([]byte, error)` — invokes ffmpeg.
  Passed into `intvoice.NewService` as the `transcode` callback so the WS
  pipeline can swap it for a passthrough in tests.
- **Maturity**: Stable. Audio-tools will eventually own this; current shape is
  already extraction-ready.

### Boundary Enforcement Tests (API)
- **File**: [CODE: scenarios/web-console/api/greenfield_assertions_test.go]
- **Tests**:
  - `TestGreenfield_TTSReusableCoreNotInPackageMain` — reusable TTS text
    primitives must live in internal/tts.
  - `TestGreenfield_InternalAudioDomainsDoNotImportHandlers` —
    internal/{voice,tts,audio} cannot import handlers/*.
- These are intentional ratchets: passing them is the precondition for
  audio-tools scenario generation.

### Audio Capability Port Seam (API)
- **Files**: [CODE: scenarios/web-console/api/internal/audioports/ports.go],
  [CODE: scenarios/web-console/api/internal/audioports/local_processor.go]
- **Interfaces**: `SpeechToText`, `TextToSpeech`, `SpeechTextProcessor`. These
  are web-console-owned capability ports — the abstraction conversation /
  terminal / TTS orchestration depends on. Local production implementation
  (`LocalSpeechTextProcessor`) is backed by `internal/tts`; the future
  audio-tools client will implement the same interfaces and slot in without
  touching orchestration code.
- **Wired into**: `Server.speechProcessor` (main.go), `ConversationStore.processor`,
  `TTSSummarizationService.processor`. Each has a `SetSpeechProcessor` setter
  and a default-bound `speechProcessor()` accessor so tests don't have to
  inject explicitly.
- **Boundary enforced by**:
  `TestGreenfield_OrchestrationRoutesThroughAudioPorts` — package-main `.go`
  files cannot call `inttts.NormalizeTextForSpeech(` /
  `inttts.SplitIntoSpeechParagraphs(` directly. Reads must go through the
  port.
- **Maturity**: Stable for the text pipeline (`SpeechTextProcessor`).
  `SpeechToText` and `TextToSpeech` are declared but the orchestration sites
  for Transcribe/Synthesize/ListVoices still call internal services directly
  through `tts_adapter.go`. Routing those through the ports is a follow-up
  pass scoped in PROBLEMS.md §10.

### Frontend Audio Adoption Boundary Seam (UI)
- **Files**: [CODE: scenarios/web-console/ui/src/domains/audio/index.ts],
  [CODE: scenarios/web-console/ui/src/domains/audio/README.md]
- **Purpose**: The UI mirror of `internal/audioports`. The re-export surface
  is the only path consumer modules should use when reaching for reusable
  audio capability code (`VoiceStreamProvider`, `WhisperProvider`,
  `WebSpeechProvider`, VAD, audio filter chain, shared AudioContext,
  `KokoroProvider`, `BrowserTTSProvider`, `TranscriptionProvider`/`TTSProvider`
  type contracts).
- **Migration**: When audio-tools ships its UI surface, redirecting these
  re-exports is the only edit needed; consumer imports across `Workspace`,
  `TerminalPane`, terminal input gate, and the settings sections stay stable.
- **Classification of web-console-specific (non-extractable) UI** is recorded
  in the README: voice-command parser, audio cues / recording activity,
  `useVoiceInput` orchestrator, `useTextToSpeech` orchestrator,
  `tts-playback` controller, mic button / rejection banner /
  command-suggestion components, and the audio player bar tied to the
  conversation cursor.
- **Maturity**: Stable as a documented boundary; no behaviour change
  shipped. Migrating consumer imports to use `domains/audio` rather than
  reaching into `hooks/voice/**`/`hooks/tts/**` directly is a follow-up
  ratchet (recorded in PROBLEMS.md §10).

### Connected Scenarios Registry Seam (API + UI)
- **API file**: [CODE: scenarios/web-console/api/internal/capabilities/registry.go]
- **Checker file**: [CODE: scenarios/web-console/api/internal/capabilities/checkers.go]
  (`ScenarioChecker`)
- **Action file**: [CODE: scenarios/web-console/api/internal/capabilities/actions.go]
  (`LifecycleActionService`)
- **Handler file**: [CODE: scenarios/web-console/api/handlers/capabilities/connect_handler.go]
  (`RunAction`)
- **UI file**: [CODE: scenarios/web-console/ui/src/components/IntegrationsPanel.tsx]
- **Purpose**: Single source of truth for which other Vrooli scenarios this
  web console adopts. Each `capabilities.Def` with `DependencyKind ==
  DependencyScenario` is a declared integration; the static catalogue lives in
  `capabilities.Known` so a single edit there registers a scenario across the
  capabilities Connect-RPC response, the Integrations settings panel, and any
  future feature gate.
- **Runtime probe**: `ScenarioChecker.CheckResult` shells out to `vrooli
  scenario status <slug> --json` and decodes typed fields (`status`,
  `health_status`, `health_error`, and `start_operation`). The `Run` field is
  the command-runner seam; tests substitute a closure. Status classification
  must not depend on substring searches over arbitrary CLI text.
- **Recovery boundary**: Status results may carry `reason_code`,
  `action_kind`, `action_label`, and `operator_command` for the UI. Scenario
  recovery delegates through `LifecycleActionService`, which accepts only
  `scenario_start`/`scenario_restart` for declared `DependencyScenario`
  entries, invokes `vrooli scenario start|restart <slug> --json --timeout N`,
  then blocks once with `vrooli scenario wait <slug> --json --timeout N`.
  Commands are executed as argv, not shell strings. Resource/provider recovery
  remains with the owning scenario/provider lifecycle; audio internals belong
  to audio-tools.
- **UI behaviour**: `IntegrationsPanel` groups entries by `dependencyKind`.
  Scenario integrations render in the "Connected Scenarios" subsection with
  typed reason/action guidance when unavailable. Backend-supported scenario
  actions show explicit Start/Restart buttons with progress/result/error state
  and refresh the capabilities query after completion. Operator-only states
  show the command but no button. Resource capabilities (Ollama, OpenRouter,
  and future local services) render in a separate "Local Resources" subsection.
  Whisper, Kokoro, speaker verification, and other audio providers do not
  appear as web-console-owned resources.
- **Audio-tools entry**: registered today with `slug: "audio-tools"`. Its
  features list is the contract audio-tools fills:
  `voice-input`, `voice-streaming`, `voice-speaker-verification`,
  `voice-enrollment`, `voice-output`, `tts-summarization`, `tts-cache`,
  `tts-paragraph-split`, `audio-provider-routing`.
- **Test files**:
  [CODE: scenarios/web-console/api/internal/capabilities/scenario_checker_test.go]
  (typed status, legacy list shape, lifecycle operation, malformed JSON, and
  unavailable cases);
  [CODE: scenarios/web-console/api/internal/capabilities/actions_test.go]
  (delegated lifecycle command construction, failures, declared-slug
  restriction, and argv safety);
  [CODE: scenarios/web-console/api/handlers/capabilities/connect_handler_test.go]
  (RunAction transport mapping);
  [CODE: scenarios/web-console/ui/src/__tests__/integrations-panel.test.tsx]
  (grouping, typed reason/action metadata, delegated action success/failure,
  and operator-only guidance).
- **Maturity**: Stable. Future enrichment (e.g. audio-tools self-reporting
  *which* provider — Whisper vs cloud API — it's currently using) lands by
  extending `Def`/`State` or by a follow-up RPC; the current shape leaves
  room without re-plumbing the panel.

### Audio admin / runtime ports (API)

- **Where**: [CODE: api/internal/audioports/ports.go] (interfaces),
  [CODE: api/internal/audioports/contracts.go] (proto ↔ domain mappers
  + typed enums), `remote_*.go` (Remote* adapters: speaker / wake-word
  / stream-config / tts-config / summarize-config / playback-event).
- **Seam**: every handler in `handlers/audio_admin` and
  `handlers/audio_runtime` accepts these interfaces as `Deps`; tests
  pass small fakes. See
  [CODE: api/handlers/audio_admin/connect_handler_test.go] and
  [CODE: api/handlers/audio_runtime/connect_handler_test.go].
- **Boundary rule**: handlers consume only audioports domain types.
  audio-tools proto types never cross the handler boundary;
  `contracts.go` is the single conversion point.

### Voice WS reverse proxy (API)

- **Where**: [CODE: api/voice_stream_proxy.go] — registered at
  `/api/v1/voice/stream`.
- **What it does**: opens a same-origin WebSocket to the browser,
  dials audio-tools' upstream WS, and bridges frames both directions.
  The browser never sees audio-tools' host; web-console owns the
  scenario-URL resolution + re-resolution.
- **Seam**: takes an `audiotoolsint.URLResolver` so tests can inject
  a fake-upstream URL.

## Overlay keyboard avoidance

**The seam.** `useAppViewport` (`ui/src/hooks/useAppViewport.ts`) measures
`window.visualViewport` and writes `--rcl-viewport-height`,
`--rcl-keyboard-inset` and `--rcl-safe-top` onto the document element. That is
the host half of the React Component Library's viewport contract, documented on
`BaseStyles`: *"a host that knows better assigns these six properties on the
document element and every library surface follows it."* Each overlay
primitive's CSS consumes them behind `[data-avoid-keyboard]`.

**The trap.** The primitive's `avoidKeyboard` prop defaults to `false`. An
overlay that never mentions it compiles, renders, and passes review — and then
on a phone the virtual keyboard slides up over the field being typed into.
Twelve of fifteen overlays were in that state. The correct code and the broken
code differ by an absent line, which is exactly what code review does not see.

**The guard.** `ui/src/__tests__/overlay-keyboard-contract.test.ts` requires
every file rendering `ResponsiveDialog`, `FullPageDrawer`, `BottomSheet` or
`DrawerShell` to mention `avoidKeyboard` — either opting in, or opting out with
a stated reason in a comment directly above. It also asserts it found at least
ten overlays, so a refactor cannot empty its subject set and pass vacuously.

To find every overlay and its choice:

```
grep -rlE "<(ResponsiveDialog|FullPageDrawer|BottomSheet|DrawerShell)\b" ui/src/components --include=*.tsx
```
