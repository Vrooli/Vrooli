# Web Console — Seams & Responsibility Boundaries

Last updated: 2026-03-17

## Responsibility Zones

### 1. Entry / Presentation
**Owner**: `ui/src/components/`
- [CODE: ui/src/components/Workspace.tsx] — **Stable core**: pane grid layout, header, empty-state UI. Delegates all session logic to `useSessionManager` hook.
- [CODE: ui/src/components/ErrorBanner.tsx] — **Volatile edge**: reusable error display with category/recovery/retry. Single place to change error UX.
- [CODE: ui/src/components/TerminalPane.tsx] — xterm.js rendering plus pane-local conversation consumption (active-pane auto-TTS, seen/listened cursor advancement). Exposes `speakText`/`speakSequence` via TerminalPaneHandle for MessagesPane TTS delegation
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
**Owner**: [CODE: ui/src/hooks/useTerminalSocket.ts] (client), [CODE: api/terminal_ws.go] (server)
- `useTerminalSocket` — Manages WebSocket connection, bidirectional I/O (stdin/stdout), conversation event delivery, conversation event acknowledgments, resize messages, keepalive, and lifecycle events (exit, error, disconnect). Signals readiness via `onReady` callback. Accepts optional `createSocket` factory for test injection.
- `terminal_ws.go` — Server-side WebSocket upgrade, message framing, PTY I/O bridging, ping/pong
- Key invariant: terminal transport carries both raw PTY frames and semantic `conversation_event` side-channel messages, but only the conversation side-channel drives unread/messages/TTS logic.

### 2b. Conversation Ingestion
**Owner**: [CODE: api/conversation_router.go], [CODE: api/tts_hook_handler.go], [CODE: api/codex_tailer.go]
- Claude hook adapter parses Stop-hook payloads and appends assistant conversation events.
- Codex tailer parses rollout output and appends assistant conversation events.
- `appendConversationEvent(...)` is the only semantic ingestion path.
- Key invariant: source adapters produce normalized conversation events first; TTS is downstream of those events.

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

## Testability Seams

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
**File**: `ui/src/hooks/useTerminalSocket.ts`
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
| `FakeWebSocket` | Real `WebSocket` via `SocketFactory` seam | `useTerminalSocket.hook.test.ts` |
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
**Files**: `ui/src/hooks/voice/wakeword/types.ts`, `ui/src/hooks/voice/wakeword/engine.ts`, `ui/src/hooks/voice/wakeword/passiveListener.ts`
**Purpose**: Isolate audio feature extraction and comparison behind a strategy interface so the MFCC+DTW implementation can be swapped for a neural embedding model later.

| Component | Production | Test |
|-----------|-----------|------|
| `WakeWordEngine` interface | Strategy abstraction for feature extraction + comparison | Allows mock engines in integration tests |
| `MfccDtwEngine` | Extracts 13-coefficient MFCCs, compares via DTW with Sakoe-Chiba band | Direct unit tests with synthetic audio signals |
| `createWakeWordEngine()` | Factory — single point of change for swapping implementations | Tests call factory to verify wiring |
| `PassiveListener` | VAD + ring buffer + MFCC/DTW loop running in RAF tick | Unit-testable via mocked engine and VAD refs |
| `extractMfcc()` | Pure-JS MFCC extraction (FFT, mel filterbank, DCT) | Tested with known-frequency sine waves |
| `dtwDistance()` | DTW with Sakoe-Chiba band constraint | Tested with identical, shifted, and unrelated sequences |

**Benefits**: All wake word detection runs client-side (no audio leaves the browser during passive mode). The `WakeWordEngine` interface is the replacement seam — swapping to neural embeddings requires only a new class implementing the same interface and updating `createWakeWordEngine()`.

### Voice Segment Boundary Seam (API)
**File**: `api/voice_stream_ws.go`
**Purpose**: Segment-final transcription runs in a goroutine separate from the partial ticker, allowing high-quality retranscription without blocking streaming partials.

| Component | Production | Test |
|-----------|-----------|------|
| Segment boundary channel | Receives from WebSocket input loop | Can be directly sent to in tests |
| Segment-final goroutine | Calls Whisper with transcoded audio | Mockable via `transcribeBytes` |

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
- `voice/VoiceStreamProvider.ts` — WebSocket streaming provider (preferred)
- `voice/WhisperProvider.ts` — HTTP batch transcription provider
- `voice/WebSpeechProvider.ts` — Browser-native fallback + SpeechRecognition types
- `voice/vad.ts` — Voice Activity Detection pure functions
- `voice/audioUtils.ts` — `createAudioFilterChain` pure function
- `voice/index.ts` — barrel re-exports

**State machine**: `idle` -> `preparing` -> `recording` -> `transcribing` -> `idle`
(Replaces the old `isRecording`/`isTranscribing` boolean combo which allowed impossible states.)

| Component | Production | Test |
|-----------|-----------|------|
| `WhisperProvider` | Records via MediaRecorder, POSTs audio to `/api/v1/voice/transcribe` | Mock `navigator.mediaDevices` + mock fetch |
| `VoiceStreamProvider` | Starts MediaRecorder immediately on mic acquisition, buffers chunks until WebSocket connects, then streams to `/api/v1/voice/stream` | Mock WebSocket + mic + MediaRecorder |
| `WebSpeechProvider` | Uses browser SpeechRecognition API with `continuous: true`, `interimResults: true` | Mock `window.SpeechRecognition` |
| `TranscriptionProvider` interface | `start()`, `stop()`, `onResult`, `onError`, `onPartial` callbacks | Same interface, deterministic behavior |
| `MediaDevicesAdapter` | `navigator.mediaDevices.getUserMedia()` | Mock that resolves/rejects for permission tests |
| AudioContext singleton | Reused across recording sessions; resumed if suspended | Mock constructor, assert single creation |
| Language parameter | `voiceLanguage` from store -> `lang` (WebSpeech) / `language` (Whisper/Stream); `"auto"` omits language param for Whisper auto-detection | Set store value, assert provider property |
| Audio buffering (VoiceStreamProvider) | Chunks buffer in `pendingChunks` before WS connects; flushed on `ws.onopen` | Mock WS in CONNECTING state, verify chunks buffered then flushed |
| WS reconnection | 2 attempts with exponential backoff (1s, 3s) + chunk buffering during reconnection | FakeWebSocket close simulation |
| Stale WS cleanup | `start()` closes previous WS and resets MediaRecorder before creating new ones | Call `start()` twice, verify first WS is closed |
| Final timeout | `computeFinalTimeout(elapsed)`: max(10s, 2x recording duration), capped at 60s | Pure function, table-driven unit tests |
| Audio bitrate | `AUDIO_BITRATE = 48_000` for MediaRecorder `audioBitsPerSecond` | Constant, ~6KB/s on localhost |
| Stream chunk interval | `STREAM_CHUNK_INTERVAL_MS = 250` | Constant assertion |
| `createAudioFilterChain` | Builds highpass (80Hz) + lowpass (8kHz) Butterworth filter chain -> `MediaStreamAudioDestinationNode` + `AnalyserNode` | Mock AudioContext with fake node factories |
| `computeSlidingNoiseFloor` | 25th-percentile sliding window (30 samples ~= 2s at 15Hz) with asymmetric hysteresis (immediate rise, gradual decay at 0.5x/s) | Pure function, table-driven unit tests |
| `vadTick(vad, rms, now, silenceTimeoutMs)` | Exported pure function. Drives VAD state machine; accepts `silenceTimeoutMs` parameter (default 2000ms) | Direct unit testing with synthetic VadRefs and timestamps |
| `processedResultCount` | WebSpeechProvider instance field tracking dispatched result indices to prevent cumulative duplication; persists across spontaneous browser restarts | Controllable SpeechRecognition stub fires cumulative `onresult` events |
| `startRecording` error guard | `try/finally` ensures `startingRef` is always cleared, preventing permanent lockout | Throw during capability check, assert subsequent recording succeeds |
| Capability liveness check | Pre-recording debounced check uses `fetchCapabilitiesLiveness` (GET-only, no test transcription) for fast response; full check only on mount | Mock both endpoints, verify liveness is used pre-recording |
| `capCheckResolvedRef` | Gates provider creation until mount-time capability check resolves, preventing wrong provider type | Click mic before mount check, verify streaming provider is used |

**Benefits**: Voice input can be tested without real microphone access or Whisper server. Fallback chain (Whisper -> Web Speech -> disabled) is testable by controlling capability fetch responses. AudioContext reuse prevents browser context limit exhaustion. Each provider is independently testable in its own module. State machine prevents impossible state combinations.

### Audio Transcoding Seam (API)
**File**: `api/audio_transcode.go`
**Purpose**: Decouple audio format conversion from transcription handlers for testable preprocessing.

| Component | Production | Test |
|-----------|-----------|------|
| `transcodeAudio` package var | `defaultTranscodeAudio` → ffmpeg stdin/stdout pipe (16kHz mono WAV) | No-op passthrough or tracking function via `t.Cleanup` |
| `checkFfmpeg` | `sync.Once` + `exec.LookPath` caches ffmpeg availability | Implicitly controlled by `transcodeAudio` override |

**Benefits**: Audio preprocessing can be tested without ffmpeg installed. Both batch (`handleVoiceTranscribe`) and streaming (`transcribeBytes`) paths share the same seam. Graceful fallback to raw audio when ffmpeg is unavailable or transcoding fails.

### Voice Stream WebSocket Seam (API)
**File**: `api/voice_stream_ws.go`
**Purpose**: Decouple streaming transcription from the Whisper service for testable WebSocket behavior.

| Component | Production | Test |
|-----------|-----------|------|
| `whisperURL` package var | Points to `localhost:8090/asr?output=json` | Swapped to `httptest.NewServer` URL via `t.Cleanup` defer |
| `transcribeBytes(ctx, audio, language, transcode, initialPrompt)` | Optionally transcodes via `transcodeAudio`, then calls Whisper; appends `initial_prompt` to URL when non-empty | Uses mock Whisper handler to verify payload size, language, and URL params |
| `transcode` parameter | `true` for final transcription (ffmpeg WAV), `false` for partials (raw WebM) | Track `transcodeAudio` call count per WS lifecycle — 0 for partials, 1 for final |
| `transcodeAudio` package var | `defaultTranscodeAudio` → ffmpeg 16kHz mono WAV | No-op passthrough via `t.Cleanup` in `setupVoiceWSServer` |
| `VoiceStreamConfig` on `Server` | Runtime-configurable struct with `FlushIntervalMs`, `MinDeltaBytes`, `OverlapBytes`. Read once per session (snapshot pattern). Backed by `store/voice-config.json`. | Tests set config via `srv.setVoiceConfig(VoiceStreamConfig{...})` before dialing WS. Each test gets its own `srv` — no cleanup needed. |
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

### Terminal Cache Storage Seam (UI)
**File**: `ui/src/lib/terminalCache.ts`
**Purpose**: Abstracts terminal state persistence for cache-based history resume.

| Component | Production | Test |
|-----------|-----------|------|
| `saveTerminalCache()` | Serializes terminal state to `sessionStorage` | Direct `sessionStorage.clear()` in `beforeEach`; tested via public API without mocking internal storage |
| `loadTerminalCache()` | Reads and deserializes cached terminal state from `sessionStorage` | Same — tested via public API |
| `clearTerminalCache()` | Removes cached state from `sessionStorage` | Same — tested via public API |

**Benefits**: Enables instant visual restore on page refresh. Server sends only delta output. Tests verify cache lifecycle without browser dependencies.

**Boundary**: `terminalCache.ts` ↔ `TerminalPane.tsx` (serialize/restore) ↔ `useTerminalSocket.ts` (offset negotiation)

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

**`TTSSynthesizer` interface** (`tts_synthesize.go`):
- Production: `KokoroSynthesizer` — proxies to Kokoro-FastAPI `/v1/audio/speech`
- Test: Mock returning `io.ReadCloser` with test audio bytes
- Injected via `Server.ttsSynthesizer` field

**`TTSVoiceLister` interface** (`tts_voices.go`):
- Production: `KokoroVoiceLister` — proxies to Kokoro-FastAPI `/v1/audio/voices`
- Test: Mock returning `[]TTSVoice` slice
- Injected via `Server.ttsVoiceLister` field

### Capability Gating

**`KokoroChecker`** (`capabilities_checkers.go`):
- Liveness: GET `/v1/audio/voices` → 200
- Full check: POST `/v1/audio/speech` with test text, verify non-empty audio response
- Cached by `CapabilityRegistry` with 30s TTL

### Hook Delivery Chain

**Path**: `tts-hooks.sh` → `claude-code` resource hook reconciliation → Claude Code Stop hook in repo-root `.claude/settings.json` → `handleHookStop` / `CodexTailer` → `routeTTSCandidate` → `SendTTS` → WebSocket `tts_candidate` side-channel → UI `useTerminalSocket` `onTTSCandidate` → terminal-visible correlation → `useTextToSpeech.speakParagraphs` → WebSocket `tts_ack`

**Seam points**:
1. `tts-hooks.sh` ↔ `claude-code` resource: scenario declares desired hook; resource owns settings-path resolution, JSON merge, and idempotent healing
2. Claude Stop hook ↔ API: HTTP POST with `X-Hook-Token` auth header
3. `routeTTSCandidate` ↔ source adapters: backend routing only accepts explicit terminal ownership; it does not infer from PTY output
4. `SendTTS` ↔ WebSocket: buffered candidate fan-out (non-blocking, drops on full)
5. `useTerminalSocket` ↔ `TerminalPane`: client receives `tts_candidate` and emits `tts_ack`
6. `TerminalPane` ↔ xterm.js buffer: rendered terminal text is the source of truth for correlation
7. `useTextToSpeech` ↔ `TTSProvider`: injectable Kokoro/Browser implementations

**Testing**: `tts_hook_handler_test.go` covers Claude session mapping. `tts_router_test.go` covers candidate routing/dedup. `codex_tailer_test.go` includes an E2E test from rollout file → owning terminal candidate.

### Two Independent TTS Trigger Paths

1. **Claude Code Hook** (`tts-hooks.sh` → `claude-code` reconcile → `claude-stop-hook.sh` → `handleHookStop`): Active push. Claude Code fires a Stop hook after each response. Web-console now uses a command hook instead of a raw HTTP hook so the terminal environment can inject `WC_WEB_CONSOLE_SESSION_ID` directly into the payload. Claude keeps its native shared `~/.claude` session storage unchanged, so sign-in and onboarding state are preserved.
2. **CodexTailer** (`codex_tailer.go`): Passive poll. Watches each terminal session's dedicated `CODEX_HOME/sessions/` tree and extracts assistant text. Each terminal gets a prepared `CODEX_HOME` overlay: shared auth/config is symlinked from `~/.codex`, while rollout/session data remains terminal-owned. Rollout ownership is therefore explicit from the filesystem path, not inferred from text.

Both paths converge at `routeTTSCandidate()` which gates on: `autoEnabled`, explicit target session ownership, and **dedup check**. Browser-side correlation happens later against the rendered xterm buffer, and the client reports outcomes back via `tts_ack`. `/api/v1/tts/status` exposes both backend routing and client acknowledgment state.

**Dedup cache** (`ttsDedup` in `tts_router.go`): routing uses a time-bounded event-identity cache keyed from `source + session + cleaned text`. Entries expire after `ttsDedupTTL` (30s). The `ttl` field is injectable for testing.

**ANSI stripping**: `routeTTSCandidate()` strips ANSI before publishing the candidate so browser correlation and playback always operate on speakable text.

**`staleTimeout` injectable field** (`codex_tailer.go`): The `CodexTailer.staleTimeout` field overrides the default `codexStaleTimeout` (1 hour) for testing. When non-zero, `tailFile()` uses this value for the stale timer. Tests use short values (100ms) to verify watcher cleanup without waiting an hour.

## Boundary Violations Fixed

### Phase 2 (2026-02-19) — Responsibility Boundaries
| Violation | Before | After |
|-----------|--------|-------|
| WebSocket protocol in TerminalPane | TerminalPane mixed xterm.js rendering with WS protocol | Extracted to `useTerminalSocket` hook |
| Data formatting in SessionDrawer JSX | Inline `split("/").pop()`, `toLocaleTimeString()` | Extracted to `lib/format.ts` utilities |
| setTimeout shortcut injection | `setTimeout(500)` timing assumption in Workspace | Event-driven `onReady` callback from TerminalPane |
| ANSI escape codes scattered | Hardcoded `\x1b[90m` in TerminalPane | Centralized `ANSI` constants in useTerminalSocket |
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
2. ~~**No reconnect logic**~~ — **Resolved**: `useTerminalSocket` now auto-reconnects with exponential backoff (max 5 attempts) and defers reconnection when the tab is backgrounded via `visibilitychange` listener
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
**Files to touch**: `session_handlers.go` (catalog entry), optionally `useTerminalSocket.ts` (WS recovery)
**Test coverage**: `TestErrorCatalog_StructuralInvariants` validates all entries have valid category, message, recovery, status. `TestWriteJSONError_UnknownCode_Fallback` verifies graceful degradation for new codes.
**Invariant**: Unknown codes fall back to `internal` category with generic recovery hint.

### Axis 4: Session Policies (P1-001)
**What changes**: Adding config knobs (env vars), new policy limits, expiration behavior
**Cost**: Low — `config.go` centralizes all tunables with env var mapping and validation/clamping
**Files to touch**: `config.go` (add field + env var), `session.go` (apply policy)
**Test coverage**: `config_test.go` covers defaults, env overrides, clamping, invalid fallback. `TestSessionLimit_VariousLimits` validates limit behavior across multiple values.

### Axis 5: WebSocket Protocol (P0-002b)
**What changes**: Adding message types, changing framing, adjusting handshake
**Cost**: High (inherently coupled) — requires coordinated changes in `terminal_ws.go` and `useTerminalSocket.ts`
**Files to touch**: `terminal_ws.go` (server), `useTerminalSocket.ts` (client), both message type definitions
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
| `useTerminalSocket.ts` | **Stable core** | WS lifecycle hook — additive message types only |
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
| `isCleanWsClose(code)` | `useTerminalSocket.ts` | Whether a WebSocket close is intentional (1000/1001) vs. unexpected | Close code |

### Decision Groupings by Domain

| Domain | Where Decisions Live | Key Functions |
|--------|---------------------|---------------|
| **Session lifecycle** | `session.go` | `applySessionDefaults`, `isSessionLimitReached`, `Create`, `Delete` |
| **Session policy** | `session_policy.go` | `ValidatePolicy`, `ResolveTTL`, `IsExpired`, `buildPolicyResponse` |
| **Error classification** | `session_handlers.go` | `classifyCreateError`, `writeJSONError`, `errorCatalog` |
| **AI command extraction** | `ai_generate.go` | `extractCommand`, `knownCodeFences`, `checkProviderResponse` |
| **Configuration** | `config.go` | `resolveShell`, `envInt`, `LoadConfig` |
| **WebSocket transport** | `terminal_ws.go` (server), `useTerminalSocket.ts` (client) | `isCleanWsClose`, WS message dispatch |

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
| `sync_warning` | Server→Client | Coalescing notification: `coalesced_frames` count indicates output frames merged due to slow consumption (data is preserved, not lost) |

**UI Feedback Surfaces:**
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
4. ~~**WebSocket reconnect**~~ — **Resolved**: Auto-reconnect with exponential backoff + visibility-aware deferral in `useTerminalSocket`.
5. **Session delete from UI** — No confirmation feedback beyond the session disappearing from the list. Low priority.
