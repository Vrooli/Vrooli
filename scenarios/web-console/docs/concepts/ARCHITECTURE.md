# Architecture

Web Console is a browser-based terminal that connects to PTY processes on the host via WebSocket. It is designed for single-operator use on a personal Vrooli server.

It now has two distinct data planes:

1. Terminal I/O for raw PTY fidelity.
2. Conversation events for semantic assistant-response features such as auto-TTS, unread counts, and messages view.

Terminal input itself has two lanes. The reliable stdin lane carries operator
payloads and receives cumulative UTF-8 byte-offset acknowledgements; the best-effort
control lane carries synthetic terminal bytes such as recovery gestures and is
never replayed after reconnect. The stdin lane preserves four source intents:
`typing`, `bulk_text`, `named_key`, and `control`. `control` is represented on
the separate control lane, while the other three intents select the appropriate
PTY or tmux delivery primitive.

## System Layers

### Browser launch and viewport ownership

The visual concept for launch is continuity: Aquila's existing icon and name
sit on the same slate canvas from the installed launch image through the first
interactive workspace. `ui/src/launch-screen.html` and `launch-screen.css` own
this small composition. Vite inlines them into the initial HTML; `LaunchScreen`
uses the same trusted markup during the health query, lazy workspace import,
and session hydration. There is no minimum display time or simulated progress.
The progress indicator stops animating for reduced-motion users. The OS launch
images under `ui/public/public/launch/` render the same composition and existing
icon in the dimensions declared by `ui/src/launch-profiles.json`. Their links
follow Apple's [startup-image mechanism](https://developer.apple.com/library/archive/documentation/AppleApplications/Reference/SafariWebContent/ConfiguringWebApplications/ConfiguringWebApplications.html).

`App` owns `useAppViewport` before any asynchronous loading boundary. It sizes
html, body, React root, and workspace to one measured frame and publishes
keyboard state to the workspace store. The shared library hook owns keyboard
detection, including touch keyboards that contract both layout and visual
viewports; Web Console does not infer a second keyboard state from focus.
Keyboard-open toolbar layout reduces outer gutters, keeps key sizes intact,
and removes the home-indicator padding because that edge meets the keyboard.

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
│  │  [CODE: api/session/session_manager.go]  │   │
│  │  PTY interface [CODE: api/pty.go]         │   │
│  └──────────────────────────────────────────┘   │
│       │                                         │
│  ┌──────────┐                                   │
│  │  Config   │  [CODE: api/internal/config/config.go] │
│  └──────────┘                                   │
└─────────────────────────────────────────────────┘
```

## Data Flow

### Session Creation

1. User clicks "New Terminal" in [CODE: ui/src/components/Workspace.tsx]
2. Launcher presents options in [CODE: ui/src/components/TerminalLauncher.tsx]
3. `useSessionManager` calls `POST /api/v1/sessions` via [CODE: ui/src/api/sessions.ts#createSession]
4. Connect handler in [CODE: api/handlers/sessions/connect_handler.go#Create] delegates to `SessionManager.Create()`
5. `SessionManager` uses `PTYFactory` to spawn a shell process via [CODE: api/pty.go#defaultPTYFactory]
6. Session starts `readLoop()` goroutine for PTY output fan-out

### Terminal I/O

1. UI opens WebSocket to `/api/v1/sessions/{id}/ws`
2. Server upgrades connection in [CODE: api/terminal_ws.go#handleTerminalWS]
3. Two concurrent loops bridge browser ↔ PTY:
   - **Output forwarder**: PTY → `Session.broadcast()` → WebSocket client (also forwards `sync_warning` on coalesce thresholds and `pty_state` on alt-buffer transitions)
   - **Input loop**: WebSocket → `Session.Write()` → PTY stdin
4. Client-side session hook [CODE: ui/src/hooks/terminal/useTerminalSession.ts#useTerminalSession] composes `useTerminalTransport` (WebSocket lifecycle), the cumulative-offset stdin stream, and a shared `TerminalInputGate` ([CODE: ui/src/components/terminal/inputGate.ts]) that is the single path every input source (xterm.onData, MobileToolbar, paste, voice, upload) flows through
5. `readLoop` splits PTY output at UTF-8 codepoint boundaries so that partial multi-byte sequences are buffered across reads, preventing JSON encoding corruption
6. When a client's output channel is full, frames are **coalesced** (merged into a per-client pending buffer) rather than dropped. The pending buffer is capped at a fixed `pendingBufferMax`; on overflow the oldest bytes are truncated and the next snapshot replay restores correct state. The forwarder calls `FlushPending` after each successful WebSocket write to drain coalesced data in 64 KB chunks to prevent browser UI freezes
7. After a trimmed buffer is fully flushed, the session restores the authoritative screen from its emulator snapshot. The snapshot and plain-text projections are produced together by [CODE: api/session/session.go#ScreenWithText], so a reconnect cannot combine bytes from different moments. Alternate-buffer state is tracked by the terminal emulator and carried in the session's `pty_state` messages rather than by a parallel recovery handler.
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

**Client**: xterm.js is a pure renderer. On every WS open the hook calls `terminal.reset()`, writes every snapshot stdout frame verbatim, then on `history_end` flips to live mode and writes subsequent stdout frames as live PTY output. The input stream tracks cumulative UTF-8 byte offsets and reconciles only the unaccepted suffix after a reconnect; it never retries a timed-out payload by timer. Prediction is a retractable overlay and never mutates xterm's server-owned buffer.

**Why the rewrite happened**: the previous raw-PTY-byte history ring was not replay-safe across alt-buffer transitions. A captured stream containing an unmatched `\x1b[?1049h` would leave the reconnecting xterm stuck in alt-buffer (where scrollback is disabled by VT spec), making history appear to vanish.

**Invariants** (enforced by tests in `api/terminal/emulator_test.go`):

- `Feed` is total: never errors, consumes every byte; malformed escapes are dropped.
- `Snapshot()` is idempotent under no input.
- `Snapshot()` is complete: feeding the snapshot bytes into a fresh `Emulator` reproduces the same triple.
- Alt-buffer is opaque to scrollback — bytes written while `InAltBuffer()` returns true never enter the scrollback ring.
- `Resize` preserves scrollback line count.

**Key files**: `api/terminal/` (emulator + snapshot serializer), `api/session/session.go` (`Subscribe()` returns the snapshot), `api/terminal_ws.go` (chunks snapshot frames before `history_end`), `ui/src/hooks/terminal/useTerminalSession.ts` (snapshot-mode flag), `ui/src/lib/terminalConfig.ts` (`TERMINAL_SCROLLBACK_LINES` shared constant).

### Voice Input

1. User presses Alt+Space (desktop) or taps the mic button.
2. Web-console checks the `audio-tools` capability from [CODE: api/internal/capabilities/registry.go].
3. When `audio-tools` is available, the UI and API use the audio-tools adoption boundary (`audioports.Remote*` and `@vrooli/audio-capture-browser`) for STT, streaming, speaker verification, TTS, summarization, and provider routing.
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
3. Continue opens [CODE: ui/src/components/MessageExportDrawer.tsx] (built on the shared [CODE: ui/src/components/WorkspacePaneShell.tsx]), which is a formatter/preview/copy surface only — never a second selector.
4. Deterministic ordering, escaping, rendering (Agent XML, Markdown transcript, quote blocks, plain text), and token estimation live in the pure, React-free [CODE: ui/src/lib/messageExport.ts]. Exports always render in ascending `ConversationEvent.sequence` with original roles, exactly the chosen messages, and no invented purpose wrapper.
5. Copy uses the browser Clipboard API with transient success feedback and non-destructive error feedback; closing the drawer returns to the navigator with the selection intact.

Ownership boundary: MessagesPane owns selection state; MessageJumpList owns selection affordances; messageExport.ts owns all formatting; MessageExportDrawer owns format choice, preview, and clipboard interaction.

### File Preview

File preview is a third supporting data plane, separate from terminal I/O and conversation events. It is **not** semantic conversation state — it is a reusable subsystem any web-console surface (today: conversation message links + inline path chips) can call to open a resolved local artifact.

Two transports, by intent:

1. **Metadata + bounded text — Connect-RPC** `FilePreviewService` ([CODE: api/handlers/file_preview]). `Resolve(path)` returns a `PreviewModel`: canonical path, basename, `:line`, classification (`PreviewKind`), MIME type, size, capability flags (`can_preview`/`can_download`/`supports_range`/`text_content_available`), an opaque short-lived `preview_id`, a same-origin `blob_url`, and warnings. `GetTextContent(preview_id)` returns ≤1 MiB UTF-8 for text kinds (markdown/code/text/csv/diff).
2. **Bytes — REST blob/range** `GET|HEAD /api/v1/sessions/{id}/file-previews/{previewId}/blob` ([CODE: api/file_preview_handlers.go]). A sanctioned REST exception (reason `stream_upgrade`) because byte-range streaming consumed directly by native `<img>/<video>/<audio>/<iframe>` is browser-native and cannot be a Connect call. Binary/media bytes never travel through Connect.

Resolution + classification + the preview-id store live in the transport-neutral [CODE: api/internal/filepreview] package (independently unit-tested, reusable). The UI side is registry-based: a viewer state machine ([CODE: ui/src/components/file-preview/useFilePreviewController.ts]) feeds a normalized `PreviewModel` to [CODE: ui/src/components/MessagesFileViewer.tsx], which routes on `kind` through the renderer registry ([CODE: ui/src/components/file-preview/renderers/index.ts]) — one dedicated renderer per kind (markdown, code/text, image/svg, pdf, audio, video, csv, spreadsheet, document, presentation, ebook, archive, diff, directory, unsupported).

Flow: message link/chip → `openPreview(path)` → `Resolve` issues a session-bound `preview_id` → text kinds fetch `GetTextContent`; blob kinds load lazily via `blob_href` in the renderer. The blob handler re-stats the file on every request and returns 409 if size/mtime changed since resolve, 404 for unknown/expired/session-mismatched ids.

Resolution preserves literal Unicode filenames. After literal candidates are absent, a path beginning with a full-width slash and containing no ASCII slashes gets one absolute-path fallback with full-width separators replaced; successful recovery returns a warning. This also covers percent-encoded Markdown hrefs without applying general Unicode normalization. HTML runtime audio/video admission uses the same extension registry as standalone previews, including `.m4a`, rather than a separate subset.

HTML previews load relative presentation assets through `GET|HEAD /api/v1/sessions/{id}/file-previews/{previewId}/runtime/{resource}`. The route confines resources to the HTML document's directory, rejects escaping symlinks, and preserves the script-only iframe sandbox. Allowed audio/video resources stream directly from disk without a total-file-size cap; other presentation assets retain a 16 MiB bound. Both byte routes support native HEAD, suffix ranges, 64-bit offsets, and 206/416 responses. Preview writes renew the server's 150-second write deadline per chunk, allowing active transfers to outlast the ordinary response budget while bounding stalled writes. This does not change the preview ID's 30-minute lifetime; reopening an expired preview obtains a fresh ID.

HTML viewing controls follow the drawer header surface: one shared `IconButton`
swaps Preview/Source using the existing icon morph animation. A single compact
row also provides Responsive, Phone (390px), Tablet (768px), and Desktop (1440px)
viewport widths, zoom steps, and Fit. Preset selection fits immediately; manual
zoom preserves the chosen CSS viewport and permits horizontal panning. Frame
height tracks available space at the current scale. Resize observation keeps Fit
current through drawer resizing and phone rotation. Viewport/zoom choices survive
source switches but reset for a different preview. Source line links and truncated
files still open as source. This is a bounded W3 interaction improvement to the
single-task artifact drawer, using existing tokens and primitives. The desktop
and mobile composition is recorded in [the toolbar mockup](../mockups/html-preview-toolbar.svg). Target validation: focused renderer regressions, scoped
Test Genie unit phase, and desktop/mobile browser inspection.

HTML previews support independent 1–5× inspection magnification. The host applies
one clipped compositor transform to the iframe per animation frame. The authored
body is never scaled or measured during a gesture. An isolated bootstrap forwards
screen-coordinate gestures through a message channel bound to that exact iframe;
coordinates stay stable when the iframe itself scales. Pan offsets clamp to the
viewport so the canvas cannot expose white gutters. Panning beyond the viewport
continues through the document via bounded scroll messages. Pinch keeps document
scroll offsets, viewport width/media queries, toolbar zoom and chrome stable.
Normal single-finger scrolling remains native at 1×. Pinching out restores 1×;
resize and toolbar changes reset inspection. Trackpad pinch is also supported.
The bootstrap is embedded as a raw inline module for production/profile parity;
its messages grant only inspection and scrolling, with no host operations.

#### Audio preview player

Audio previews use a focused listening card within the existing artifact drawer.
The shared content model is track identity/artwork, duration/format/size, seekable
waveform, transport, repeat/speed/volume, inspection tools, and file actions.
This is an immersive single-task sub-surface. On phones the compact identity,
waveform and large transport targets come first; inspection and secondary file
actions expand on demand. Wider drawers retain that hierarchy with more breathing
room. The [audio composition mockup](../mockups/audio-preview-player.svg) extends
the operator-approved concept. Existing semantic tokens and library primitives
own appearance; this does not change the app theme or audio-provider boundary.

The native HTML audio element owns streaming and playback. Controls reflect media
events, handle rejected play requests and buffering, and stop/release the element
when navigating or closing. Seeking is keyboard/touch accessible. Waveform zoom
exposes a movable time window; A/B markers define a reversible playback-only loop,
without editing the file. Whole-track repeat and segment repeat are exclusive.
System playback routing remains available through native controls on demand.

Optional analysis fetches at most 24 MiB after finite metadata reports at most
10 minutes, with cancellation and a timeout. An offline 8 kHz decode supplies real
waveform peaks; it never connects playback to a Web Audio graph. Embedded track
tags/artwork use the lazy-loaded governed music-metadata parser. Missing artwork,
unsupported decoding, excessive size/duration, or failed analysis degrade to a
music glyph and ordinary seek bar; playback and download remain independent.
Artwork uses bounded raster blobs whose URLs are revoked on cleanup. Validation
covers transport, resource cleanup, fallback/error cases, loops and zoom, locale
parity, scoped Test Genie unit checks, and mobile/desktop browser inspection.

#### Image, PDF, video and table previews

These four sub-surfaces share the audio player's **focused artifact inspection**
concept: compact controls, a content-first stage, quiet metadata and file actions,
semantic colors and shared library controls. The approved scope is the operator's
request to implement all four preview recommendations. Composition is recorded in
[the desktop/mobile family mockup](../mockups/artifact-preview-family.svg).

Content/action model, independent of viewport:

- Image/SVG: original dimensions/format/size, Fit/100%/zoom, focal pinch/wheel
  magnification, bounded panning, rotation and transparency/light/dark backgrounds.
  SVG remains an inert image resource. Resizing refits automatically in Fit mode.
- PDF: page/count/navigation, Fit width/Fit page/zoom, searchable selectable text,
  loading/password/error states and independent download/open actions. PDF.js and
  its worker, CMaps, fonts and WASM are bundled locally. PDF scripting and form
  editing are disabled. Visible-page rendering and bounded canvases control memory;
  closing the drawer destroys the document, render tasks and worker transport.
- Video: native inline streaming with event-driven playback, seek/skip, speed,
  whole-video repeat, volume/mute, fullscreen and on-demand system controls. It
  shares the tested media transport with audio and stops/releases on navigation.
- CSV/TSV: original values, explicit loaded-row/column counts, global filtering,
  stable numeric-aware sorting, adjustable columns, cell copy, pagination and
  source access. Existing text/row bounds remain explicit; sorting/filtering apply
  only to loaded rows. Ragged rows retain extra fields. Narrow drawers compose
  a record list/detail view from the same data model rather than shrinking a wide
  table. Users can switch between record and table views at any width.

Images, PDFs and video are immersive single-task sub-surfaces: controls stay
reachable above/below the stage, independent of drawer width. The table archetype
uses conditional composition measured from the drawer, not the browser viewport.
Unsupported codecs, PDF failures and oversized table shapes preserve download
and source access. Source changes reset inspection state. Validation covers
interaction math, real media/document behavior, parsing/data integrity, teardown,
keyboard/touch access, locale parity, and narrow/wide browser inspection.

#### Markdown and configuration readers

Extend focused artifact inspection to Markdown and JSON/YAML. The operator
approved starting with these two recommendations. These are read-only immersive
sub-surfaces, using existing ArtifactChrome, semantic tokens and shared controls.
Markdown supplies rendered/source modes, literal in-document search with match
navigation, text-size controls and an outline derived from rendered headings.
A wide drawer can retain an outline beside the reading column; a narrow drawer
opens the outline on demand and returns to the selected section. Preserve the
existing Markdown renderer, safe links, diagrams and local-file navigation.
Reader controls never mutate source or intercept external navigation.

JSON/YAML supply Structure/Source modes, key/value/path filtering, collapsed
collections and an inspector with full original values and copy value/path/source
actions. On wide drawers the selected value sits beside the tree; narrow drawers
use a dedicated detail view with Back. Search includes collapsed descendants.
Retain original scalar spellings (including large numbers), explicit document
selection for YAML streams, empty/null distinctions, and source fallback for
invalid, truncated, overly deep/large or unsupported shapes. Parsing runs in a
terminable local worker with character/node/depth limits. Aliases remain references,
custom tags/complex keys use explicit source fallback; no YAML object construction,
implicit code execution or alias expansion. Structured viewing does not add an
editor or change file resolution, API kinds, authorization or source downloads.
See [reader composition](../mockups/artifact-reader-previews.svg).

### Error Handling

All API errors return structured JSON with `code`, `category`, `recovery`, and `retry` fields. See [Error Semantics](../internal/ERROR_SEMANTICS.md) for the full contract.

Client-side [CODE: ui/src/lib/errors.ts#APIError] parses these into typed errors. The [CODE: ui/src/components/ErrorBanner.tsx] component renders recovery hints and retry buttons based on error metadata.

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
| Audio adoption boundary (re-export surface) | `ui/src/audio-integration/` | **web-console** | Single import path orchestration code uses; pointed at in-tree hooks today, swapped to audio-tools client at adoption time |
| Mic capture, VAD, audio context, provider mechanics | `@vrooli/audio-capture-browser` | `audio-tools` | Reusable across any audio-consuming scenario; web-console keeps only terminal targeting glue |
| `useVoiceInput`, `useTextToSpeech` hook orchestration | `ui/src/hooks/use*.ts` | Split — generic readiness/lifecycle to audio-tools; terminal-input-gate wiring stays | See [`ui/src/audio-integration/README.md`](../../ui/src/audio-integration/README.md) for the per-file classification |
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
retains a private voice-stream provider instead of using
`@vrooli/audio-capture-browser`.

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
- **AI generation** (`api/handlers/ai/adapter.go` and `api/internal/ai/service.go`) owns the generation pipeline — providers, prompt building, extraction, and config-aware orchestration. The internal config stores own provider configuration and health tracking.
- **Policy handlers** are co-located with the session Connect handlers because they operate on session sub-resource endpoints (`/sessions/{id}/policy`). Policy domain logic and the expiration sweeper live in `api/session/session_policy.go`.
- **Message snippets** are a self-contained feature module under `api/internal/snippets/` and `api/handlers/snippets/`. The repository interface hides the in-memory and SQLite stores; the UI consumes only the generated Connect contract through `ui/src/api/snippets.ts`.

The UI uses **component-per-file** with hooks extracted into `hooks/`, constants into `consts/`, and utilities into `lib/`. The app now ships as a single workspace surface with feature-local section modules under `components/settings/` for the unified settings experience. Shared domain constants (policy options, shortcuts, toolbar keys) live in `consts/` and are imported by multiple components to avoid duplication.

## File Map

| File | Responsibility |
|------|---------------|
| `api/main.go` | Server wiring, routes, middleware, health checks |
| `api/conversation_store.go` | In-memory conversation event store, cursor tracking, playback state |
| `api/conversation_router.go` | Conversation-event append/orchestration entry point for AI response sources |
| `api/conversation_handlers.go` | Conversation REST handlers (`GET /conversation`, `PUT /conversation/cursor`) |
| `api/session/session.go` | Session domain logic, UTF-8 boundary buffering, frame coalescing |
| `api/session/session_manager.go` | SessionManager lifecycle, backend selection and remote launch |
| `api/pty.go` | PTY interface and factory (testability seam) |
| `api/errors.go` | Error catalog, structured error types, HTTP error-writing helpers |
| `api/handlers/sessions/` | Session Connect handlers and transport adapter |
| `api/session/session_policy.go` | Expiration policy sweeper |
| `api/terminal_ws.go` | WebSocket upgrade + bidirectional I/O bridge (context-based goroutine lifecycle) |
| `api/internal/config/config.go` | Environment-based configuration with validation |
| `api/handlers/ai/adapter.go` | AI command generation and transport projection |
| `api/internal/ai/service.go` | Provider chain orchestration, prompt context and health |
| `api/repository.go` | Storage interfaces: `ShortcutStore`, `AIConfigStore` |
| `api/internal/ai/config_store_mem.go` | In-memory AI provider config store + health tracking |
| `api/internal/ai/config_store_sql.go` | SQLite AI provider config store (hybrid: config in SQLite, health in-memory) |
| `api/shortcut_profiles.go` | In-memory shortcut profile store + domain types |
| `api/shortcut_profiles_sql.go` | SQLite shortcut profile store |
| `api/internal/events/events.go` | Structured event logging (session lifecycle, AI) |
| `api/internal/metrics/metrics.go` | Operational metrics collection |
| `api/internal/snippets/` | Snippet types, validation, seed data, and in-memory/SQLite repository implementations |
| `api/handlers/snippets/` | Connect-RPC snippet CRUD, touch, and one-way skill-promotion adapter |
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
| `ui/src/hooks/terminal/useTerminalTransport.ts` | WebSocket protocol handling |
| `ui/src/hooks/useMediaQuery.ts` | Responsive settings shell behavior |
| `ui/src/hooks/useCountdown.ts` | Policy countdown timer (shared by SessionDrawer + SessionsPage) |
| `ui/src/components/TerminalPane.tsx` | xterm.js rendering |
| `ui/src/components/MessagesPane.tsx` | Semantic messages-mode rendering for a single session |
| `ui/src/components/snippets/` | Shared picker, variable completion, body editor, and save sheet |
| `ui/src/components/messages/messageActions.ts` | Declared, capability-gated message action registry |
| `ui/src/lib/snippetVars.ts` | Import-free named-variable substitution |
| `ui/src/components/TerminalLauncher.tsx` | New-terminal modal with shortcuts |
| `ui/src/components/MobileToolbar.tsx` | Floating keyboard toolbar |
| `ui/src/components/AiInput.tsx` | AI command input with generate/execute flow |
| `ui/src/components/IntegrationsPanel.tsx` | Dependency health status display (all resources/scenarios) |
| `ui/src/components/ErrorBanner.tsx` | Structured error display |
| `ui/src/components/ErrorBoundary.tsx` | React error boundary with region labels |
| `ui/src/api/` | HTTP, Connect and WebSocket client functions |
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
| OT-P0-002 | Production-Grade Web Terminal Fidelity | [CODE: api/pty.go], [CODE: api/terminal_ws.go], [CODE: ui/src/hooks/terminal/useTerminalTransport.ts] | [DOC: docs/internal/TEMPORAL-FLOWS.md] |
| OT-P0-003 | Durable Session Continuity | [CODE: api/session/session.go] (offline buffer, reconnect), [CODE: api/handlers/sessions/] | [DOC: docs/internal/INVARIANTS.md] |
| OT-P0-004 | Proxy-Correct Networking via api-base | [CODE: ui/src/api/client.ts], `@vrooli/api-base` integration | [DOC: docs/reference/configuration.md] |
| OT-P0-005 | AI Input with Provider Fallback | [CODE: api/internal/ai/service.go] (provider chain, fallback), [CODE: ui/src/components/AiInput.tsx] | [DOC: docs/internal/ASSUMPTIONS.md#behavioral-assumptions] |
| OT-P0-006 | New Terminal Launcher with Configurable Shortcuts | [CODE: ui/src/components/TerminalLauncher.tsx], [CODE: ui/src/consts/shortcuts.ts], [CODE: api/shortcut_profiles.go] | [DOC: docs/concepts/GLOSSARY.md#shortcut] |
| OT-P0-007 | Mobile Terminal Usability Toolbar | [CODE: ui/src/components/MobileToolbar.tsx], [CODE: ui/src/consts/toolbar-keys.ts] | [DOC: docs/internal/PROBLEMS.md#experience-follow-ups] |
| OT-P0-008 | Sidebar/Drawer Controls Surface | [CODE: ui/src/components/SettingsModal.tsx], [CODE: ui/src/components/settings/SessionManagementSection.tsx] | [DOC: docs/internal/SEAMS.md#1-entry--presentation] |

### P1 – Should Have

| Target | Description | Implementation | Docs |
|--------|-------------|----------------|------|
| OT-P1-001 | Session Policy Controls | [CODE: api/session/session_policy.go], [CODE: ui/src/consts/policy-options.ts], [CODE: ui/src/hooks/useCountdown.ts] | [DOC: docs/concepts/GLOSSARY.md#policy] |
| OT-P1-002 | Shortcut Profile Management | [CODE: api/shortcut_profiles.go], [CODE: api/shortcut_profiles_sql.go], [CODE: ui/src/components/settings/ShortcutProfilesSection.tsx] | [DOC: docs/internal/SEAMS.md#storage-ownership-and-persistence-boundaries] |
| OT-P1-003 | AI Provider Policy Controls | [CODE: api/internal/ai/config_store_mem.go], [CODE: api/internal/ai/config_store_sql.go], [CODE: ui/src/components/IntegrationsPanel.tsx] | [DOC: docs/internal/SEAMS.md#storage-ownership-and-persistence-boundaries] |
| OT-P1-004 | Operational Observability Coverage | [CODE: api/internal/metrics/metrics.go], [CODE: api/internal/events/events.go] | [DOC: docs/internal/SEAMS.md#6-cross-cutting] |

## Remote desktop presentation

Web Console owns the operator-facing desktop pane, not desktop authority.
It requests a typed session from Bridge, consumes the Device Control/WebRTC
offer, renders one selected display, sends raw input and semantic actions only
when the controller lease is held, and presents typed readiness/refusal
states. It must not call a node directly, store credentials, or implement
capture/input semantics.

### P2 – Future

| Target | Description | Implementation | Docs |
|--------|-------------|----------------|------|
| OT-P2-001 | Collaborative Session Modes | Not yet implemented — requires multi-subscriber session model and auth | [DOC: docs/internal/PROBLEMS.md] |
| OT-P2-002 | Persisted Workspace Presets | Not yet implemented — requires workspace state serialization | [DOC: docs/internal/PROBLEMS.md] |

## Browser desktop pane

The Web Console desktop pane is a presentation adapter over the generated
Device Control contract. It distinguishes offline, no-session, permission,
ready, connecting, disconnected, viewer, controller, revoked, and failed states. It attaches
the authorized media stream to a browser video element and disables pointer,
keyboard, wheel, and clipboard controls unless the current controller lease and
epoch are present. Controller input is translated into bounded typed pointer,
wheel, and key actions against the selected display; clipboard sends enforce the
64 KiB UTF-8 bound before transport. It does not construct node URLs or retain
desktop credentials.

## Source, change, and workbook inspection

The approved artifact-inspection concept extends to code/logs, patches, and Excel.
Reuse semantic artifact chrome and RCL controls; do not introduce a separate palette.
The surfaces are read-only, with original download/copy preserved.

- Source: literal in-file search with next/previous matches; go to an original line;
  click/Shift-click line selection and exact selected-line copying; wrap/text size.
  Logs expose optional severity filtering while retaining original line numbers.
  Bound syntax highlighting and rendered rows; disclose partial text and search scope.
- Changes: parse unified patches into files/hunks and old/new line numbers. Show
  additions/deletions, file/hunk navigation, wrap and unified/split modes. Wide drawers
  compare old/new columns; phones default to unified changes. Unsupported patch
  syntax retains the original source and download rather than inventing a diff.
- Workbooks: classify .xlsx/.xls/.xlsm/.xlsb as a blob-backed spreadsheet kind.
  Parse locally in a terminating worker, with bounded fetch, time, sheet, row,
  column and cell budgets. Never execute macros, formulas, HTML or hyperlinks.
  Surface saved values and formula text separately; cached values may be absent
  or stale. Display sheet visibility, merged ranges and explicit bounded-preview
  notices. Preserve original coordinates, blank/false/zero values, numeric/date
  formatting, sheet order and source download. No recalculation or editing.
  Desktop uses a sticky-header grid and cell inspector; narrow drawers show row
  summaries leading to row/cell details with focus restoration. Provide search of
  loaded cells, sheet selection, paging, address navigation and cell copying.
  Corrupt/encrypted/unsupported workbooks retain explanatory recovery/download.

Dependency choice: SheetJS CE 0.20.3 from its pinned official distribution
(https://docs.sheetjs.com/docs/getting-started/installation/nodejs/), governed by
Scenario Dependency Analyzer. The stale npm xlsx release is excluded. ExcelJS
was considered but lacks legacy XLS reading. Code loads only in the workbook
worker, which is destroyed on close/source change/timeout.

Mockup: docs/mockups/source-diff-workbook-previews.svg. Established preview concept
and the operator's explicit “all 3” authorization govern this scoped continuation.

Visual refinement after operator review: source, changes and workbooks use a
composed artifact header with an icon tile, descriptive metadata and compact
tools. Options are contained disclosures, search opens on demand, and reading
canvases have a restrained border/radius with intentional spacing. Change review
uses explicit comparison headers and colored counts. Workbooks add desktop sheet
tabs and a value-first cell inspector with icon copy actions. Phone toolbar
controls retain sheet context without a second permanent header.

Workbook preview budgets are 15 MiB input, 64 MiB per ZIP entry/128 MiB declared
ZIP expansion, ten seconds, 30 sheets, 2,000 rows, 100 columns, 50,000 positions,
and four million serialized value/formula characters. Omitted values are explicit.
Source and patch views page at 400 rows; patch parsing is capped at 12,000 lines.
The FullPageDrawer 1.4.6 owner fix pairs physical centering/translation so RTL does
not create horizontal scrolling. Scoped qualification and broader limitations
are retained in docs/internal/PROBLEMS.md.

Toolbar refinement (2026-10-18): source, changes and workbook tools keep a stable
identity row. Workbook sheet navigation gets its own full-width row on phones.
Reading controls use the existing ResponsiveDialog with explicit close affordance,
labeled settings sections, segmented view choices and aligned jump fields; opening
settings cannot stretch or clip the preview toolbar. Escape/backdrop returns focus
to its trigger; a successful jump closes settings and focuses its destination.
Inputs use 16px type to avoid automatic zoom in mobile Safari.

Diff content regression repair (2026-10-18): changed rows derive colors from
existing --wc-error-text/--wc-error-detail and --wc-success semantic tokens.
Unified/split reading uses compact 24px minimum rows, shared number widths,
aligned paired rows and canvas-level horizontal scrolling. Context stays neutral;
missing split partners have a distinct blank surface. Controls retain the accepted
responsive options design. Browser checks must verify computed colors and row
geometry in addition to parser/component tests.

Diff surface refinement (2026-10-18): the scrolling diff canvas is edge-to-edge
beneath file navigation, with no inset card margin, outer border or rounded
corners. Internal line gutters and comparison dividers retain their meaning.


#### Word documents and folder browsing

Extend the existing focused artifact inspection concept. Word is an immersive
reading sub-surface; folders are a list/browser sub-surface. Reuse semantic tokens,
InspectionHeader, responsive settings and library buttons. Keep content edge to
edge rather than adding decorative cards inside the drawer.

Word (.docx): render semantic headings, paragraphs, lists, tables, embedded raster
images and footnotes as a reflowing document. A comfortable reading measure and
clear type hierarchy replace authored page geometry. Desktop offers an optional
outline alongside the document; phone outline replaces the reading pane and
returns to the selected heading. Search highlights and navigates literal matches;
text sizing lives in the established responsive settings dialog. Copy text and
download remain available. Explain once in settings that pagination and complex
Word layout may differ. Do not imply exact printed-page fidelity or support for
legacy .doc. Parse only in a bounded, disposable worker; validate ZIP bounds,
disable external resources and embedded styles, sanitize all converter HTML,
and prevent scripts/forms/SVG or unsolicited resource requests. Invalid,
encrypted and oversized inputs have a useful retry/download state.

Folders: compact folder identity/actions, a full-width labeled search field,
on-demand sort/hidden options, quiet count and a single list scroller. Desktop
adds aligned size/date metadata; phones place file type/size/date beneath the
name. Use restrained file icons, neutral readable names, deliberate touch targets,
and explicit empty/no-match/loading/limited-list feedback. Filtering says that it
covers loaded items when more pages exist, and paging stays reachable. Parent
navigation is distinct from history Back. Back restores a cached location and
keeps the drawer open, including after file previews and failed navigation.
Breadcrumbs stay usable without competing with dismiss controls; keyboard focus
returns to a stable navigation target. Preserve server sort, paging, symlink and
permission semantics. See ../mockups/word-folder-previews.svg.


Word conversion uses governed Mammoth 1.13.0 in its own worker; DOMPurify 3.4.17
sanitizes the resulting semantic HTML before insertion. This follows the
[Mammoth security guidance](https://github.com/mwilliamson/mammoth.js#security)
and [DOMPurify allowlist contract](https://github.com/cure53/DOMPurify).
The reader admits at most 15 MiB input, 64 MiB declared ZIP expansion, 20,000
HTML elements, 2 MiB non-image HTML and 8 MiB encoded raster images (3 MiB each).
A ten-second worker deadline and termination on close/source change bound work.
Shared Office helpers own streamed byte bounds and ZIP preflight; Excel retains
its prior bounds. Embedded style maps and external file access stay disabled.
Only embedded PNG/JPEG/GIF/WebP images survive the rendering allowlist; links
stay inert until activation, external links open with noopener/noreferrer, and
local notes scroll inside the reader. Empty/error states retain download;
fetch retry re-resolves the target to renew stale handles. Word reader controls
and folder settings use the existing responsive options owner.

Folder navigation is rendered in a dedicated band, separately from dismissal.
Back retains its hit target across folder/file transitions, disables at the
starting folder and restores focus to the navigation region. Current breadcrumbs
reveal themselves when the portal-mounted content appears. Sorting retains the
cached folder and options during refresh; settings keep focus and use guarded
aria-disabled state while listing changes are pending. File metadata uses bidi
isolation, and row rules use the already-alpha border token directly.


### Read-only archive previews

ZIP, TAR, and gzip-compressed TAR (`.tar.gz`, `.tgz`) use Go's standard archive
readers in `api/internal/filepreview`, on every supported host OS. No shell
commands or extraction endpoint participate. Other archive formats show a clear
unsupported message and retain the original download. Listing is metadata only;
opening a regular member explicitly creates a bounded, session-bound virtual
preview. Nested archives require another explicit open and share their root's
work budget. Member identifiers are physical ordinals, preserving duplicates.

Limits: 128 MiB input; 128 MiB actual expanded stream work per root capability;
16 MiB per opened member; 5,000 visible members; 4 MiB retained metadata; 1 MiB
per TAR extended header (the standard reader's enforced bound); 4,096-byte names;
32 path components; three archive levels including the root; two concurrent
archive jobs; a cooperative ten-second deadline per job; 64 MiB / 256 virtual
previews retained globally. Active HTTP responses remain charged to that quota
until released, including after logical expiry. Admission covers parsing through
atomic capability publication. Transient parser buffers are bounded by job count.
ZIP central directories are bounded and counted before standard reader allocation.
ZIP64 directory records and multipart archives degrade to download. TAR scans include skipped file
bodies, metadata, padding, and gzip trailer validation in actual byte accounting.
Sparse files, links, devices, encrypted entries and unsupported ZIP methods are
never opened. Cancellation and byte limits apply to reads, not header claims.

Virtual names use portable slash semantics and never reach a host path resolver.
Absolute paths, traversal, drive/UNC paths, backslashes, control characters and
ambiguous Windows components are blocked. Unsafe names remain visible at the
archive root as inert metadata. HTML and Markdown members use source inspection;
member runtime resources and host execution/handoff are unavailable. Blob responses
apply a restrictive CSP and never render active content in the application origin.
Every virtual handle expires with its root, and checks root file freshness.

The viewer uses the established inspection toolbar, with a concise archive summary,
filter, folder breadcrumbs, sort/hidden options, and a full-width file list without
an inner card. Desktop rows align size and status; phone rows put secondary facts
under the name. Back restores the exact archive folder/filter/options, and never
closes the drawer. Failed inspection retains the original download and recovery.
Desktop/phone composition: `docs/mockups/archive-previews.svg`.


### PowerPoint inspection surface

PowerPoint is an immersive, read-only inspection surface using the established
InspectionHeader/InspectionOptions chrome. The slide itself is the focal plane,
with no surrounding card or decorative container. Desktop can show an optional
thumbnail rail beside the fitted slide; a phone uses an on-demand slide grid that
replaces the canvas and returns to it after selection. Notes and extracted slide
text are available on demand, including empty notes. Previous/next, a slide
counter, fit/zoom and scoped keyboard navigation remain available. Preserve slide
order, hidden-slide identity and the original download. See
`docs/mockups/powerpoint-previews.svg` for the paired compositions.

Modern PPTX/PPSX/POTX packages render locally through the pinned browser OOXML
renderer; no external Office viewer, host converter, installation or upload is
required. Legacy PPT and encrypted, invalid or over-budget packages have explicit
fallback states. Animations, macros, live media and external assets are not run.
Typography may vary with installed fonts; unsupported visual features are
disclosed in the options and partial-render failures are surfaced, never hidden.

Untrusted bytes first pass a terminable worker: 20 MiB input, 4,000 ZIP records,
16 MiB per entry, 64 MiB actual total expansion, 2 MiB per XML part, bounded XML
depth/element count and 300 slides. Raster headers bound individual images to 24 million pixels
and total package image area to 64 million pixels; XML is bounded to 64 levels
and 200,000 total elements. Streamed inflation stops at actual budgets,
independent of declared ZIP sizes. Ambiguous/duplicate paths, traversal, encrypted
entries, unsupported compression, malformed XML and entity declarations are
rejected. Parsing and rendering use an opaque-origin sandboxed iframe with a
restrictive CSP, external media disabled, no network access, no same-origin,
forms, popups or host navigation privileges. The owned renderer is built as inert
source in a separate lazy asset in both development and production. The host's
bounded, abortable same-origin fetch retains access cookies across an authenticated proxy;
the source then executes only in the opaque frame under a fresh nonce-only
script policy. The frame makes no anonymous renderer/module requests. Startup
failure and its deadline report viewer unavailability; only a parser that
actually started can report a processing-budget timeout. The parent accepts messages only
from the active iframe and validates bounded plain metadata. All workers, slide
handles, media URLs, listeners and pending requests are released on change/close.

### EPUB reading surface

EPUB is a semantic reflowable reader using the Word reader's typography and
shared inspection chrome. Show book title/creator, chapter position and estimated progress (weighted by chapter byte length);
previous/next chapters and an on-demand contents list make a whole publication
navigable. Desktop can retain contents beside the text; phones use a contents
screen and return to the chapter on selection. Text sizing, comfortable reading
width, within-chapter search and original download are available. Preserve
spine order, nested navigation labels, fragments and within-book links.

EPUB shares the publication worker's bounded ZIP transport, actual decompression
budgets, CRC/path checks and termination. Package/container/OPF and EPUB 3 nav or
EPUB 2 NCX metadata are parsed with DOMParser after bounded worker validation.
Limit the spine to 300 chapters and rendered HTML to 2 MiB per chapter. Reject
encrypted/DRM books and invalid/unsupported packages with explicit recovery.
Publishers' scripts, forms, stylesheets, remote resources and embedded active
content are omitted. Sanitize semantic XHTML through the existing document
allowlist, with bounded raster images rewritten to owned blob URLs. Internal
links resolve against package parts and cannot escape to host paths. External
links require an explicit click and use safe protocols/no opener. No content
is uploaded and no host converter is required. Fixed-layout books disclose
their reflow limitation rather than claiming page fidelity. See
`docs/mockups/epub-previews.svg`.

### Plain-text inspection surface

Source inspection uses a full-width, unframed reading surface below its controls.
Plain text defaults to wrapping and compact 28px rows at the default font size;
code retains its existing spacing and wrapping default. The idle line count appears
in the header once. The lower source status row appears only for selection/copy
feedback or pagination. Logical line numbers and copied bytes remain unchanged.
Archive members and local files share this presentation.

### 3D model inspection

The existing focused artifact-inspection concept extends to a read-only 3D
viewer, an immersive single-task drawer. The operator authorized implementation
of the proposed 3D formats. Content model: model identity/format, static geometry
and authored materials, orbit/pan/zoom, fit and canonical views, wireframe, grid,
lighting, background, counts, loading/recovery and original download. Desktop
puts inspection controls alongside the stage; narrow drawers keep the stage
primary, place fit/zoom in a thumb bar, and reveal secondary settings through
the existing responsive options dialog. No extra card surrounds the canvas.
Composition: docs/mockups/model-preview.svg. This extends the already approved
artifact family; no token or primitive redesign is included.

Support boundary: GLB and self-contained glTF 2.0, OBJ geometry, ASCII/binary STL.
External buffers, textures and material libraries are never fetched. A bounded
local worker validates input, graph/geometry/allocation bounds and embedded
resources before rendering. Unsupported compression/extensions and sidecar
assets retain download with an explicit explanation. Browser WebGL2 capability
is required; failure/context loss preserves recovery/download. No OS tool or
remote conversion service participates. Rendering is lazy, demand-driven,
resolution-bounded and fully disposed on close or source change.

3D budgets: 20 MiB streamed input, 600,000 vertices / 200,000 triangles,
256 draw primitives across model instances, 2,048 nodes / depth 64, eight
scenes, 32 MiB decoded accessor storage, 32 textures/images, 4,096px texture
edges, eight megapixels per texture and 16 megapixels total. JSON syntax is
bounded before object materialization; cycles/shared parents, sparse accessors,
non-finite coordinates and unsupported feature extensions are rejected before
Three.js parsing. A 45-second download deadline, ten-second worker processing
deadline and twenty-second renderer deadline have distinct fallbacks.
The renderer is a lazy local chunk; no network models/decoders/environments
are loaded. Studio reflections are generated locally at 128px. The canvas
caps device-pixel ratio at 1.75 and total pixels at two million. Static views
render on demand; explicit rotation pauses when the page is hidden.
All workers, frames, observers, controls, geometries, materials, textures,
image bitmaps, environment targets and the graphics context have owned cleanup.

Authored thin glass/plastic materials using `KHR_materials_transmission` are
supported by the existing GLTFLoader. Preflight validates their factor (0–1)
and optional embedded texture reference before rendering; they retain the same
image and texture budgets. The renderer's transmission target inherits the
two-million-pixel canvas limit and is disposed with the renderer. This does not
enable compression decoders, external textures, or arbitrary extensions.

The same bounded material admission supports `KHR_materials_clearcoat` and
`KHR_materials_specular`, using the installed loader's authored shading. Check
their scalar factors against 0–1, specular RGB for nonnegative finite values
(HDR colors above one are valid), normal-map scale for finite values, and every
optional texture index/coordinate. These physical
material extensions cannot be combined with unlit on one material. All their
images still pass the common embedded-image and texture allocation checks;
no additional asset acquisition or decoder is enabled.

### Font specimens

The focused artifact-inspection family extends to local TTF, OTF, WOFF and
WOFF2 files. Concept: a typographic specimen, with the font itself as the focal
content and no extra card around it. Content model: authored family/style,
editable sample, size, spacing and line height, actual mapped Unicode characters,
variable axes where authored, ligatures/kerning, background and original download.
Desktop exposes specimen/character tabs and a size slider together; narrow drawers
use a compact mode selector and thumb-sized size controls. Secondary tools and
metadata use the existing responsive InspectionOptions dialog. Characters are
paginated and searchable by character or U+ code point. The browser renders the
file's authored face without synthetic bold/italic. Default specimens select a
supported language sample or mapped characters when a font lacks the preferred
script; the hero uses actual mappings for icon/script-specific fonts. Changing text does not modify
the original. See ../mockups/font-preview.svg. This is an extension of the
operator-approved artifact family, not a token/primitive redesign.

Support is browser-native via FontFace constructed from bounded bytes, with a
unique family per preview and explicit document.fonts deletion on replacement,
close or failure. No OS installation, remote font service, external URL or shell
converter is used. Metadata/coverage parsing uses fontkit in a terminable worker. Before library
parsing, an unregistered worker FontFace requires native font sanitization;
fontkit's Brotli decoder can otherwise expand beyond its expected output.
Input is capped at 10 MiB and declared expanded sfnt storage at 32 MiB; table,
character and axis counts are bounded. Download, parser and native-loader
deadlines retain meaningful retry/download fallbacks. Font collections are not
in the initial support boundary. Native browser sanitization is authoritative
for rendering; invalid or browser-unsupported fonts preserve the original.


### SQLite database inspection

Extend the existing focused artifact-inspection family with a read-only SQLite
browser. Content model: database identity, snapshot time, ordinary tables,
column types/keys, indexes and foreign keys, paginated records, per-column
filter/sort, full bounded cell details, copy and CSV export of the current page.
No SQL editor, mutations, remote database credentials, extension loading, or
persistent browser storage. Identify SQLite by its header as well as common
.sqlite/.sqlite3/.db extensions; a .db extension alone does not prove SQLite.

Desktop composition: searchable table selector, Data/Schema modes, native table
with sortable headers and cell inspection in the shared options dialog. Narrow
drawers: table selector then compact record list, drill into a record and its
cells with local Back (never drawer dismissal). All columns remain reachable.
Use shared ArtifactChrome and semantic tokens, without a card around content.
Filters explicitly target a selected column. Binary cells show length; JSON is
formatted on demand. NULL and empty strings remain distinguishable. Export is
bounded to the displayed page and neutralizes spreadsheet formula prefixes.

The existing session-bound blob capability gains an explicit SQLite snapshot
representation. Source is opened read-only, trusted schema disabled, and a
consistent read transaction/SQLite backup yields standalone bytes including
committed WAL changes. No raw filesystem copy of an active database; no writes,
checkpoint, or journal-mode changes to source. Bounded snapshot size 32 MiB,
2 concurrent snapshots, 5-second work deadline, opaque errors, no-store headers.
Virtual archive members can be inspected as standalone bytes but WAL-dependent
members must fail clearly. A browser worker owns the official SQLite WASM runtime
and an engine-read-only in-memory connection. Generated internal queries only,
quoted identifiers, bound filters, no executable schema objects or views.
Bound schema/table counts, column counts, cell bytes, response bytes, page size,
and query duration; terminate the worker on close/replacement or deadline.
Encrypted/corrupt/non-SQLite/oversized sources get recovery and download options.
Snapshots are point-in-time; Refresh obtains a new preview handle and snapshot.
