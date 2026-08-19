# Web Console — Known Problems and Risks

## 1. Interactive CLI Fidelity Edge Cases

PTY handling for complex interactive CLIs (Claude Code, Codex) may have edge cases around resize during active output, cursor reporting conflicts, and reconnect during mid-escape-sequence. Requires dedicated e2e validation.

## 2. Mobile Browser Variability

Floating keyboard toolbar behavior varies across mobile browsers (Safari iOS, Chrome Android, Firefox Mobile). Virtual keyboard interaction with xterm.js focus management needs testing on real devices.

## 3. SQLite Concurrency Under Load

While single-user bounds concurrency, rapid terminal output from multiple panes could create write contention on the SQLite transcript table. WAL mode mitigates but may need monitoring.

## 4. AI Provider Timeout and Summarize Model Selection

**PARTIALLY RESOLVED** (2026-05-17): The TTS summarize path now separates
audio-tools unreachable, deadline exceeded, and selected Ollama model missing.
Model selection is exposed in Voice Output settings through a catalog returned
by audio-tools. The catalog marks recommended, installed, missing,
default-eligible, and reasoning models, and reasoning models are excluded from
default selection.

Current default: `llama3.2:3b`, because it is installed locally and was
validated as a fast non-reasoning fallback. Research-backed candidates
(`gemma3:4b`, `gemma3n:e2b`, `phi4-mini:3.8b`) remain uninstalled on the
target machine as of 2026-05-17, so they are visible as recommended missing
models with `ollama pull` commands instead of being silently selected.

Remaining gap: run a local benchmark after explicitly installing newer
candidates. The default should change only if a candidate beats
`llama3.2:3b` on latency and preserves summary quality without visible
reasoning/prompt leakage.

## 5. Standards: Setup Steps Configuration (MEDIUM)

**RESOLVED** (2026-02-19): Simplified setup steps in service.json (removed conditional wrappers). Auditor now reports 0 standards violations.

## 6. Go Lint: Remaining errcheck Warnings

**RESOLVED** (2026-02-19): Fixed the 2 remaining errcheck warnings in `session_test.go` by wrapping deferred Delete calls in `func() { _ = sm.Delete(id) }()`.

## 7. Lighthouse SEO: 82% (Below 90% Threshold)

**RESOLVED** (2026-02-19): Added meta description to index.html and robots.txt. Lighthouse SEO now 100%.

## 8. Offline Output Buffer Growth

Sessions with default never-expire policy could accumulate unbounded transcript data in SQLite. May need eventual transcript rotation or archival strategy (deferred — not MVP).

## 8a. Web Mic Lifecycle Wedge and iOS/WebKit Residual (2026-07-02)

**RESOLVED for web-console-owned JS lifecycle drift:** server-final, user stop,
auto-stop, error, and page-lifecycle stop paths now converge on the same capture
teardown path, and production `getUserMedia` calls are forced through the mic
ownership registry. Persisted `lowLatencyVoice` / `wakeWordEnabled` flags no
longer acquire the mic on mount or tab-visible; prewarm/passive arming requires
explicit mic-control intent. Tests cover the old server-final-before-client-stop
wedge, stale-lease self-heal, no eager acquisition, and the structural
`getUserMedia` boundary.

**Residual platform limitation we cannot fix from web JS:** if iOS/WebKit fully
freezes or kills the page's JavaScript before `visibilitychange: hidden`,
`pagehide`, or `freeze` is delivered, no application code can run
`MediaStreamTrack.stop()`. In that narrow case the OS mic indicator may remain
active until the OS reclaims the track. The mitigation is to release on the
earliest delivered lifecycle signal and cancel in-flight starts; a native shell
would only matter for this hard-freeze class, not for the now-fixed web-console
state-machine wedge.

## 8b. Idle audio-session activation (background-audio interruption + wedge prevention) (2026-07-07)

**Symptom:** on iOS, opening web-console (HTTPS PWA) stopped other apps' audio
(Spotify/YouTube) — but only after the first touch/scroll, with low-latency/keep-
alive off. Separately, the mic could get wedged until a device reboot (survives
PWA force-quit — the platform residual in §8a).

**Root cause (the preventable part):** we activated the iOS `AVAudioSession` when
we did not need to, and never released it.
- `ensureAudioContextOnGesture()` installed a document-wide capture-phase
  `pointerdown`/`keydown` listener on mount and, on the **first interaction of any
  kind**, created+`resume()`d the shared AudioContext — purely to shave ~20-50ms
  off a *possible future* mic press. Resuming a WebAudio context activates the iOS
  audio session and interrupts other apps' non-mixable audio, even when the user
  never used voice.
- The context was app-lifetime and **never suspended**, so it held the session
  indefinitely.
- Low-latency voice mode (`lowLatencyVoice` + `micReadiness` pre-warm) similarly
  held a `getUserMedia` stream open while idle.

**Fix (prevention, not recovery):**
- Removed eager `ensureAudioContextOnGesture`; the AudioContext is resumed lazily
  inside the real voice/cue gesture and **suspended when idle** — on background
  (`visibilitychange:hidden`) and ~1.5s after a capture turn ends
  (`armIdleSuspend`/`keepAudioContextAwake`/`suspendSharedAudioContext` in
  `sharedAudioContext.ts`).
- Removed low-latency voice mode entirely (setting, `micReadiness` module,
  `retainStream`, injected pre-warmed streams). Providers always acquire and own a
  fresh mic stream per turn and release it fully on stop.
- Result: the audio session is active only while actually capturing or cueing, so
  background audio is no longer interrupted, and the unnecessary session churn
  that plausibly fed the §8a wedge is gone.

**Second offender (found after the first fix shipped): TTS "unlock on any gesture".**
`useTextToSpeechCore` installed persistent global `pointerdown`/`keydown`/
`touchstart` listeners that called `KokoroProvider.unlock()` — a **silent
`HTMLAudioElement.play()`** — whenever the provider reported not-yet-unlocked. On
iOS, playing an audio element (even a muted silent one) activates AVAudioSession
and ducks other apps' audio. Because the provider is rebuilt whenever the TTS
backend re-resolves (`unlocked` resets to `false`), the persistent listener
re-ran the silent play on essentially every interaction — the "after a while,
web-console pauses my music on any tap, and only a force-quit fixes it" report
(force-quit destroys the primed audio element). **Fix:** removed the preemptive
unlock from the gesture handler (it still flips the no-audio `browserAudioReady`
flag). The media element is now unlocked lazily — a real speak in a user gesture
unlocks it naturally, and a programmatic speak blocked by autoplay surfaces
`needsUnlock` + the Enable-Audio affordance (a deliberate tap → `unlock(true)`).

Together: the app now activates the iOS audio session **only while actually
capturing or playing speech**, never on incidental interaction.

This cannot *recover* an already-wedged mic (still a platform limitation, §8a),
but it removes the app behaviours that put the session into that state. See
[VOICE-LATENCY.md §3 (AudioContext lifecycle) + §5 (low-latency removed)].

## 8c. Mic auto-stops mid-sentence under kyutai — client RMS VAD is sole authority (2026-07-08, under triage)

**Symptom:** during *continuous* dictation the microphone abruptly stops ~10 s in
(before any real timeout) and the rest of the utterance is lost. This surfaced
right after the streaming backpressure wedge was fixed (audio-tools `PROBLEMS.md`
2026-07-08): the fix synced mic-state to actual transcription, so a premature
turn-end is now *visible* instead of masked by a frozen-but-still-green mic.

**Root cause (diagnosed):** the live STT engine is **kyutai/passthrough**, which
emits **no `vad-state`** events (unlike whisper-VADSegment). So on the client the
auto-stop SSOT `decideAutoStop` (`hooks/voice/autoStopDecision.ts`) has no server
tick (`serverVad.receivedAt === 0`) and falls to **§2: the client RMS VAD is the
SOLE turn-ending authority**. Under whisper the server VAD was authoritative and
vetoed the client VAD's known "stops while I'm still talking" false positives
(both signals had to agree); under kyutai that veto is gone. A client-VAD silence
verdict (`vadTick` → `stop` after `silenceTimeoutMs` of RMS below `speechThreshold`)
then ends the *whole turn* whenever the analyser reads silence — which happens when
the audio source stops delivering samples even though the user is still speaking:
  1. **Muted mic track** — the OS/browser mutes the track on sleep/wake, a
     default-input-device change, or another app seizing the mic. The `ended`
     event does NOT fire for muting (`streamHealth.isTrackUsable`), so no samples
     flow yet the track stays "live" and the analyser reads pure silence.
  2. **Suspended AudioContext** — if the shared context suspends mid-capture
     (`ctx.state !== "running"`), the analyser returns flat/silent data.
  3. **Threshold miscalibration** — the adaptive `speechThreshold` (= noise floor
     × 3) rising above the user's real speech RMS in a noisy room.

Relay evidence: sessions close `graceful=true tailFinalDelivered=true` in a
regular ~10–12 s cadence — i.e. **client-initiated** stops, not backend kills. The
frequent kyutai `reaping wedged prior streaming session` warnings are a *separate*
benign-but-noisy lock-release-latency issue (the relay dials kyutai and takes
`MODEL.lock` the instant the browser WS connects — including the mount-time
pre-connect — so overlapping/rapid sessions reap already-delivered holders; NOT
the cause of the mic stop; tracked as a follow-up).

**First changes shipped (2026-07-08):**
- **Instrumentation to make the next repro conclusive** (`useVoiceCore.ts`): every
  VAD/auto-stop log now carries `rms / trackAlive / trackMuted / ctx.state`; an
  `audio-source WENT SILENT / RECOVERED` line fires the instant the source stops
  delivering; `provider.onError` logs its triggering reason; `endCapture` logs its
  `reason`.
- **Honest mic-source handling** (`VoiceStreamProvider.attachCaptureLifecycleHandlers`
  + `MediaRecorder.onerror`): a mid-capture `mute` surfaces a `mic_muted` status
  (transient, turn kept open); a terminal `ended` recovers the retained audio via
  HTTP fallback instead of drifting into a silent VAD stop; an encoder error is
  surfaced. Covered by `VoiceStreamProvider.trackLifecycle.test.ts`.
- **Guard**: a client-VAD (`client-fallback`) auto-stop is **suppressed when the
  audio source is not delivering samples** (muted / ctx not running) — a silence
  verdict on a non-delivering source is an artifact, not a real pause; a suspended
  context is resumed. Terminal loss is still owned by the track handlers +
  no-audio watchdog.

**To close (needs one real-device repro with DevTools console open):** dictate
continuously ≥30 s until the mic stops, then read the console. The decisive line
is the `VAD client-stop` / `auto-stop` at the stop instant:
  - `trackMuted=true` or `ctx=suspended` → the source went silent (mute/suspend);
    the guard now keeps the turn alive — confirm dictation resumes on unmute.
  - `trackAlive=true trackMuted=false ctx=running` with `rms` *below* `speechThresh`
    → threshold miscalibration; the durable fix is to give kyutai a server-VAD (so
    the client isn't the sole authority) or make the sole-authority client VAD more
    conservative (longer sustained-silence requirement / segment-boundary-continue
    instead of hard turn-stop). Do NOT blind-tune thresholds without this line.

See audio-tools `docs/internal/PROBLEMS.md` (2026-07-08 wedge entry + follow-up).

## 8d. Mic won't start — "could not determine audio codec" (2026-07-08, FIXED + live-proven)

**Symptom:** the mic fails to start with `provider.onError → ending turn:
audio-tools/audioformat: could not determine audio codec (declare input_format)`.
Intermittent at first, then persistent (every session `segments=0` in the relay
log).

**Root cause:** the streaming path relied on the backend **sniffing** the codec
from the first audio frame (`buildVoiceStreamWsUrl` declared no `format`). The
backend's `audioformat.Detect` needs the first chunk's leading bytes to be the
WebM/EBML header (`0x1A45DFA3`) — the header MediaRecorder emits in its FIRST
`ondataavailable` chunk. But `VoiceStreamProvider.ondataavailable` sent each
chunk via `e.data.arrayBuffer().then(send)` — **independent async promises that
can resolve out of order**. The header chunk is the largest, so its
`arrayBuffer()` can consistently resolve *after* the next chunk once timing
shifts (GC/load) — putting a headerless frame on the wire first → sniff fails.
Because it's timing-driven it presents as "worked, then persistently broke."

**Fix (two parts, both shipped + live-proven against the running backend):**
1. **Ordered delivery** (`VoiceStreamProvider`): send the Blob directly and
   synchronously in `ondataavailable` order (`enqueueOutbound`/`flushPending`,
   `pendingChunks: Blob[]`) — no async `arrayBuffer()` that can reorder. The
   header chunk is now always the first frame on the wire. Regression test in
   `VoiceStreamProvider.trackLifecycle.test.ts` ("sends audio chunks in recording
   order as raw Blobs").
2. **Declare-first** (`buildVoiceStreamWsUrl`): declare `format=webm` (the
   web-console always records WebM/Opus) so the backend skips the fragile sniff
   entirely — defense-in-depth if any frame is ever reordered/dropped.

**Live proof (synthetic client → audio-tools :19630):** no-format + headerless
first chunk → `could not determine audio codec` (reproduces the bug); `format=webm`
+ headerless first chunk → no codec error (declare-first bypasses sniff);
`format=webm` + header-first → clean `status`, stream processes.

## 8e. Mic stays live but transcription stalls mid-session (2026-07-08, instrumented, under triage)

**Symptom:** the original wedge shape — after a while (tens of seconds) the mic
stays active and audio keeps streaming, but new transcription stops appearing.
Confirmed from a real console log: a ~110s session delivered ~17 segments then
`Final received: 0 chars`; the WS closed only at the *end* (after the final), so
it is **not** a kyutai reap (a reap closes the connection mid-session → client
reconnect, which did not happen).

**What it is NOT:** the send-side backpressure wedge (fixed 2026-07-08 — kyutai's
decode enqueues to a non-blocking send worker; verified) and not the codec bug
(§8d, fixed). Kyutai's reap warnings are a *separate* lock-release-latency issue
that hits short overlapping sessions (0-segment closes), not this stall.

**Leading hypothesis:** kyutai's streaming LM context saturating over a long
continuous session. `server.py` force-commits at `MAX_SEGMENT_FRAMES`, so
continuous speech normally keeps committing; a stall where partials AND segments
both stop, correlated with session *duration* (not audio content), points to the
streaming transformer degrading to padding-only output once its context window
fills. To be confirmed by the new instrumentation before any server-side change.

**Instrumentation shipped to make the next repro conclusive:**
- `VoiceStreamProvider` transcription-stall watchdog: logs `⚠ transcription STALL:
  no partial/segment for <gap>ms while still recording (chunks/bytes/segments)`
  whenever the backend stops emitting for >4s while audio keeps streaming.
- `segment-final #N at +Tms (gap=…): "…"` logged on every committed segment (so
  the exact time segments stop is visible), plus per-partial event tracking.
- **Also fixed:** all voice console logs used printf `%.4f` — NOT a browser
  console specifier, so every such line rendered with its args shifted by one
  (e.g. `trackAlive=running, trackMuted=true` was really ctx/trackAlive). Now
  template literals. Prior logs from before this fix must be read de-shifted.

**To close:** dictate continuously until it stalls; the console will show the last
`segment-final` time and the `⚠ transcription STALL` gap. If segments stop at a
consistent *elapsed time* regardless of speech, it confirms context saturation →
fix in kyutai `server.py` (reset/rotate the LM streaming state on a bounded
window). If it correlates with audio, look at the VAD/commit path instead.

## 8f. Mid-recording empty final masked backend loss (2026-07-09, fixed in working tree)

**Symptom:** the user could keep recording while the backend stream died, then the
client accepted a `final` with empty text as successful completion. `onclose`
then skipped reconnect/fallback because `finalReceived=true`, silently dropping
retained audio.

**Fix:** `VoiceStreamProvider` now treats any `final` received while
`MediaRecorder.state === "recording"` and the stop was not intentional as backend
loss: it does not set `finalReceived`, does not call `onResult`, closes the WS,
and lets the existing reconnect-then-HTTP-fallback path recover retained audio.
The kyutai relay also preserves typed busy as `stt_busy`, which the client
surfaces as a visible busy status.

## 9. E2E Issues

**PARTIALLY RESOLVED** (2026-02-19): Added BAS workflows for terminal command execution, route-level session persistence, reconnect replay, and multi-pane independence:
- `bas/cases/01-foundation/01-terminal/launch-custom-command-executes.json`
- `bas/cases/01-foundation/01-terminal/session-metadata-persists-across-route-navigation.json`
- `bas/cases/01-foundation/01-terminal/reconnect-offline-buffer-replay.json`
- `bas/cases/01-foundation/01-terminal/multi-pane-independent-io.json`
- `bas/cases/01-foundation/01-terminal/interactive-stdin-roundtrip.json`
- `bas/cases/01-foundation/01-terminal/session-persists-across-full-reload.json`

**Remaining gaps:**
- No BAS mobile viewport workflow yet for floating toolbar key/chord behavior.
- Playbooks phase currently blocks on BAS startup when browser-automation-studio dependencies are unavailable (e2e workflows present but cannot execute).

**Added recovery coverage (2026-07-12):**
`bas/cases/02-messages/02-voice/deterministic-incomplete-coverage.json`
executed successfully as `06696480-3902-47e3-8db6-8e7971ef42e0`. It grants the
WAV-backed fake microphone to the real Web Console UI, crosses its same-origin
voice WebSocket proxy, injects a server close only after the first PCM chunk,
and asserts the visible `incomplete_coverage` error plus metadata-only
diagnostic export. The Audio Tools fault gate was enabled only for this run and
restored to false afterward. This is deterministic desktop-browser evidence,
not a substitute for the remaining iOS/Android device matrix.

## 9a. File Preview — Intentionally Deferred (2026-06-30)

The file-preview subsystem (`api/internal/filepreview`, `FilePreviewService`, blob/range route, UI renderer registry) ships these deliberate deferrals — all are working-as-intended, not bugs:

- **PDF.js deferred.** PDFs render via the native browser viewer (`<iframe>` at the blob href) with a download/open fallback. PDF.js is only worth adopting if native rendering proves inadequate in validation across target browsers.
- **No media transcoding.** Audio/video play through native browser codecs only. Unsupported codecs (`.mov`, `.flac`, …) show a clear "your browser may not support this format" hint plus download — web-console does not transcode.
- **CLI is metadata/text only.** `web-console file-preview resolve` and `file-preview text` cover programmatic metadata + bounded text. Blob streaming/download has no CLI command (no clear operator workflow yet); it stays UI/browser-only because the blob route is consumed by native media elements.
- **In-memory preview-id store.** Preview ids live in a process-local store with a 30m TTL; they do not survive an API restart. That is acceptable for the single-operator, reopen-on-demand UX. A persistent store would only matter for long-lived shareable preview links, which are out of scope.

## 10. Audio Extraction Prep — Deferred Sub-Phases (2026-05-16)

Active initiative: `swarm-manager/initiatives/continuous-audio-platform`. The
plan at `/home/matthalloran8/.vrooli/plans/web-console-audio-extraction-prep-for-audio-tools.md`
defines 8 phases. The 2026-05-16 implementation pass completed:

- **Phase 0** (baseline) — focused voice/TTS tests green.
- **Phase 2 (partial)** — `NormalizeTextForSpeech`, `SplitIntoSpeechParagraphs`,
  `TTSMaxChunkLength` and their tests now live in `api/internal/tts`. The
  `inttts.*` qualifier is the new call shape for `package main` consumers
  (`conversation_router.go`, `conversation_adapter.go`, `conversation_store.go`,
  `tts_summarization_service.go`).
- **Phase 3** — voice domain dependency direction inverted. Handler types,
  Service interface, Backend interface, and the production Adapter all live in
  `api/internal/voice` (`types.go`, `handler_adapter.go`). `handlers/voice/module.go`
  is now a thin re-export shim; `handlers/voice/adapter.go` deleted. Exit
  criterion verified: `rg 'web-console/handlers/voice|voiceH\\.' internal/voice
  --type go` → zero hits.
- **Phase 5 (partial)** — `TestGreenfield_TTSReusableCoreNotInPackageMain` and
  `TestGreenfield_InternalAudioDomainsDoNotImportHandlers` added to
  `api/greenfield_assertions_test.go`. They lock the boundary against
  regression.

The following sub-phases are deferred and **must be completed before the
`scenarios/audio-tools` greenfield item can land cleanly**:

- **Phase 2 (remaining) — DONE 2026-05-16.** All reusable TTS core types and
  functions have moved into `api/internal/tts`:
  - `Cache`, `CacheKey`, `CacheEntry`, `CacheStats`, `NewCache`,
    `cacheKeyHash` live in `internal/tts/cache.go`.
  - `Summarizer`, `SummarizerResponse`, `NewSummarizer`, `StripThinkTags`,
    `summarizeSystemPrompts`, `summarizeTokenBudget` live in
    `internal/tts/summarizer.go`.
  - `SummarizationService`, `SummarizeRequest`, `SummarizeResult`,
    `NewSummarizationService`, sentinel errors `ErrSummarizeCoolingDown`,
    `ErrSummarizeBudgetInThink`, `ErrSummarizeTruncated`,
    `ErrSummarizeEmptyAfterStrip`, `ErrSummarizeTrulyEmpty`,
    `SummarizeErrorMessage`, `classifyEmptySummary` live in
    `internal/tts/summarization_service.go`. The previous
    `audioports.SpeechTextProcessor` indirection inside the service is gone;
    the service calls `NormalizeTextForSpeech` / `SplitIntoSpeechParagraphs`
    directly (same package — no port needed). The port abstraction at the
    consumer (conversation/store) layer is unchanged.
  - `DefaultConfig`, `ConfigPatch.Apply`, `LoadConfig`, `SaveConfig` live in
    `internal/tts/config.go` (the `Config` and `ConfigPatch` types
    pre-existed in `types.go`).
  - `DefaultSummarizeConfig`, `SummarizeConfigPatch.Apply`,
    `LoadSummarizeConfig`, `SaveSummarizeConfig`,
    `MinSummarizeTimeoutSeconds`, `DefaultSummarizeTimeoutSeconds`,
    `MaxSummarizeTimeoutSeconds` live in `internal/tts/summarize_config.go`.
  - Pure-logic tests moved alongside: `cache_test.go`, `config_test.go`,
    `summarize_config_test.go`, `summarizer_test.go`,
    `summarization_service_test.go` all under `internal/tts`.
  - Server-tied glue stays in `package main`: `tts_cache.go` keeps
    `invalidateTTSCacheForEvent`, `preSynthesizeTTS`,
    `synthesizeParagraphs`; `tts_config.go` keeps `getTTSConfig`/
    `setTTSConfig`/`getTTSSummarizeConfig`/`setTTSSummarizeConfig`/
    `resolveTTSSummarizeConfigPath`. The `Server` fields now type-hold
    `inttts.Config`, `inttts.SummarizeConfig`, `inttts.Cache`,
    `inttts.Summarizer`, `inttts.SummarizationService` directly — no
    aliases.
  - `greenfield_assertions_test.go::TestGreenfield_TTSReusableCoreNotInPackageMain`
    locks the boundary against regression for the moved files and the
    type/function definitions.
- **Phase 4 (extension) — DONE 2026-05-16.** `LocalSpeechToText` and
  `LocalTextToSpeech` adapters now live under `internal/audioports/`
  (`local_stt.go`, `local_tts.go`) and are wired by `NewServer` into
  `Server.sttPort` / `Server.ttsPort`. `SetSpeechToText` / `SetTextToSpeech`
  setters let tests substitute fakes. Orchestration callsites in
  `tts_adapter.go` (Synthesize / ListVoices / GetCache) and
  `tts_cache.go::synthesizeParagraphs` / `preSynthesizeTTS` route through
  `s.ttsPort`; cache writes in the on-demand path still go via
  `s.ttsCache.Put` because the conversation-event-keyed cache is web-console
  glue (per the Phase 8 dossier). The greenfield assertion
  `TestGreenfield_OrchestrationGoesThroughAudioPorts` in
  `greenfield_assertions_test.go` locks the boundary: package-main files
  (other than `main.go` which constructs the local adapters) must not
  reference `s.voiceService.Transcribe`, `s.ttsSynthesizer.Synthesize`,
  `s.ttsVoiceLister.ListVoices`, `internal/voice.Service.Transcribe`,
  `internal/tts.KokoroSynthesizer`, or `internal/tts.KokoroVoiceLister`. The
  summarize pipeline (`SummarizationService`, `SummarizeRequest`,
  `SummarizeResult`, `SummarizeErrorMessage`, `ErrSummarizeCoolingDown`) is
  explicitly NOT routed through audioports in this pass — that port shape
  remains an open audio-tools-shared-contract question (see Phase 8
  dossier). Also resolved the orchestration-level double-normalize wart:
  `conversation_router.go::asyncSummarizeAndNotify` and
  `conversation_adapter.go::SummarizeEvent` now pass `event.Text` directly
  to `ttsSummarization.Summarize`, since normalization is a property of the
  summarize pipeline (handled inside `SummarizationService.run`).
- **Phase 6 (extension) — DONE 2026-05-16.** All UI consumers now import
  audio capability surface through `ui/src/domains/audio` rather than
  reaching directly into `hooks/voice/**` / `hooks/tts/**`. 16 consumer
  files migrated (Workspace, TerminalPane, WorkspacePaneShell, MessagesPane,
  MobileToolbar, VoiceRejectionBanner, VoiceCommandSuggestion, AudioPlayerBar,
  tts/AudioSettingsContent, settings/VoiceInputSection, api/voice.ts,
  domains/tts-playback/types.ts, and five `__tests__/*` consumer tests).
  Added re-exports to `ui/src/domains/audio/index.ts`: `CommandSuggestion`,
  `VoiceRejection` (voice types), `TTSPlaybackState`, `TTSPlaybackCapabilities`
  (tts types), and the reusable wake-word surface (`createWakeWordEngine`,
  `MIN_ENROLLMENT_SAMPLES`, `MAX_ENROLLMENT_SAMPLES`, `AudioFeatures`,
  `WakeWordTemplate`, `useWakeWordTest`). Web-console-specific paths
  (`hooks/voice/commands`, `commandParser`, `audioCues`, `activity`,
  `useVoiceInput`, `useTextToSpeech`) deliberately stay direct per README
  classification. Boundary locked by
  `ui/src/__tests__/audio-boundary.test.ts`, which walks `src/**/*.{ts,tsx}`,
  skips the adoption surface / underlying hooks / test files, and flags any
  `from "...hooks/voice/...|hooks/tts/..."` import outside the documented
  web-console-specific allowlist. Verification: `pnpm type-check`,
  `pnpm test -- --run` (116/116 batches pass, including the new boundary
  test), and `pnpm build` all clean.
- **Phase 7 — DONE 2026-05-16.** `ARCHITECTURE.md` now has an "Audio
  Ownership Map" section with backend/frontend ownership tables, the
  Connected Scenarios Registry contract, and the capability-port adoption
  seam description. DOMAINS.md and SEAMS.md already covered earlier in the
  pass.
- **Connected scenarios registry — DONE 2026-05-16.** `audio-tools` is now
  declared in `capabilities.Known` with `DependencyKind: DependencyScenario`;
  `ScenarioChecker` probes via `vrooli scenario status <slug>`; the
  Integrations settings tab renders a "Connected Scenarios" subsection
  separate from "Local Resources". When `scenarios/audio-tools` ships, no
  registry change is required — the existing checker will start returning
  `available` automatically.
- **Phase 8 — DOSSIER BELOW 2026-05-16.** Source dossier captured inline (next
  subsection) since in-repo edits to
  `swarm-manager/execute/audio-tools-greenfield-scenario/plan.md` were not
  authorized in this pass. Promote into that plan when the audio-tools
  greenfield item is picked up.

### Phase 8 Dossier — input for `audio-tools-greenfield-scenario`

**Source inventory — files to port into audio-tools (reusable capability core):**

Backend:
- `api/internal/voice/transcribe.go` — Whisper HTTP proxy, language passthrough, hallucination filter (deduplicate identical repeated outputs).
- `api/internal/voice/stream_ws.go` — streaming transcription, partials, segment finals, final retranscription, cancellation, VAD boundaries.
- `api/internal/voice/speaker.go` + `speaker_client.go` + `speaker_config.go` — speaker verification accept/reject/advisory/fallback decisions.
- `api/internal/voice/wakeword.go`, `config.go` — wake-word config and voice config schema.
- `api/internal/audio/transcode.go` — ffmpeg-backed transcode primitives.
- `api/internal/tts/` — entire package: `kokoro_synthesize.go`, `kokoro_voices.go`, `normalizer.go`, `chunker.go`, `service.go`, `types.go`.
- `api/tts_cache.go` — `TTSCache`, `TTSCacheKey`, `cacheKeyHash`, LRU + eviction (event-keyed `Evict()` is a thin convenience over key prefix — provider-level cache is the reusable part).
- `api/tts_summarizer.go` — Ollama-backed `TTSSummarizer`, `summarizeSystemPrompts`, `summarizeTokenBudget`, `stripThinkTags`.
- `api/tts_summarization_service.go` — `TTSSummarizationService` with cooldown, inflight dedupe, empty-summary classification, timeout/error sentinels.
- `api/tts_config.go`, `api/tts_summarize_config.go` — config schemas + load/save (atomic JSON write).

Frontend:
- `ui/src/hooks/voice/**` — capture readiness, VAD utilities, provider mechanics (Whisper/VoiceStream/WebSpeech).
- `ui/src/hooks/tts/**` — Kokoro provider, browser TTS provider, cache playback controls, unlock primitive.
- `ui/src/hooks/useVoiceInput.ts`, `useTextToSpeech.ts` — split: lifecycle/readiness portion is reusable; terminal-input-gate wiring stays.
- See [`ui/src/domains/audio/README.md`](../../ui/src/domains/audio/README.md) for the authoritative per-file classification.

**Files that stay in web-console (consumer integration glue — do NOT port):**

- `api/conversation_router.go`, `api/conversation_store.go`, `api/conversation_adapter.go` — auto-TTS trigger policy, listened-cursor, fan-out.
- `api/tts_hook_handler.go` — Claude Stop hook attribution.
- `api/codex_tailer.go` — Codex rollout tailer.
- `api/tts_playback.go` — playback ack / status snapshots tied to conversation cursor.
- `api/terminal_ws_input.go` — terminal-input-gate routing.
- `api/internal/audioports/` — adoption boundary lives in the consumer; audio-tools provides an implementation.
- `ui/src/domains/tts-playback/**` — conversation-cursor state machine.
- `ui/src/components/VoiceMicButton.tsx`, `VoiceCommandSuggestion.tsx`, `VoiceRejectionBanner.tsx` — terminal-target UX (the underlying capture primitive is reusable; the targeting policy is not).

**P0 behavior currently proven by web-console tests (must be replicated in audio-tools):**

- Transcribe: language passthrough, hallucination filter, speaker verification bypass via flag, gating by speaker verification result.
- Stream WS: partials emitted while listening, segment-final on VAD boundary, final retranscription on stop, cancellation, speaker advisory/reject flows.
- Transcode: passthrough when no resample needed, ffmpeg invocation arg shape.
- TTS config & summarize config: load returns defaults when file missing, atomic save, patch-apply semantics, timeout clamp on load.
- TTS service: synthesize via Kokoro, voice list, Kokoro availability gating, invalid-args rejection, cache hit/miss, cache eviction by `eventID`.
- Normalization/chunking: `NormalizeTextForSpeech`, `SplitIntoSpeechParagraphs` for code blocks, tables, diagrams, lists.
- Summarizer + service: cooldown after timeout, inflight dedupe by `EventID`, `<think>` stripping, four-way empty-summary classification (budget-in-think / truncated / empty-after-strip / truly-empty), timeout clamp.

**Contract decisions already settled by `audio-provider-routing-contract`:**

- Provider precedence is BYOK → Vrooli/LPBS → Local. Web-console does not implement this — audio-tools owns it.
- `ErrInsufficientCredits` short-circuits without falling through providers (BYOK/LPBS fail-fast).
- Canonical voice mapping is owned by audio-tools (provider-specific voice IDs map to canonical names).
- LPBS usage reporting and credit metering are not web-console concerns.

**Tests that should be ported or rewritten in audio-tools:**

- `api/voice_transcribe_test.go` → audio-tools STT service tests (drop the speaker-verification bypass cases that exercise web-console env-flag wiring).
- `api/internal/voice/stream_ws_test.go` (if present) → audio-tools streaming-STT tests.
- `api/internal/audio/transcode_test.go` → audio-tools transcoder tests.
- `api/internal/tts/service_test.go`, `chunker_test.go`, `normalizer_test.go` → audio-tools TTS tests.
- `api/tts_cache_test.go` → split: provider-level cache tests to audio-tools; conversation-eviction tests stay in web-console.
- `api/tts_summarizer_test.go`, `tts_summarization_service_test.go` → audio-tools summarization tests.
- `api/tts_config_test.go`, `api/tts_summarize_config_test.go` → audio-tools config tests.

Tests that **stay** in web-console: `tts_hook_handler_test.go`, `tts_router_test.go` (conversation routing), `conversation_*_test.go` (auto-TTS trigger, listened cursor).

**Open questions for `audio-tools-shared-scenario-contract`:**

- Should `EventID` survive across the audio-tools boundary, or does audio-tools take an opaque cache key from the consumer? (Web-console invalidates by `EventID` today on conversation summarization.)
- Is the `SpeechTextProcessor` port worth exposing remotely, or do consumers just call `Synthesize` and let audio-tools normalize/split internally?
- Streaming STT transport: WebSocket-only, or Connect-streaming as the canonical surface with WebSocket as a browser-friendly mapping?
- Speaker verification: separate Connect service or a `Transcribe` request flag?
- Does audio-tools own the `qwen3:4b` summarization model choice, or does the consumer pass model preference?
- Whose responsibility is the unlock-on-user-gesture primitive for browser TTS — audio-tools-provided component, or web-console-owned?

**Adoption checklist for `web-console-adopt-audio-tools` (depends on this dossier):**

1. Replace `audioports.LocalSpeechTextProcessor` with `audioports.AudioToolsProcessor` (HTTP/Connect client).
2. Replace `Server.ttsSynthesizer` wiring (currently `internal/tts.NewKokoroSynthesizer`) with an audio-tools-backed `TextToSpeech` port.
3. Replace `Server.voiceService` wiring with an audio-tools-backed `SpeechToText` port.
4. Delete `api/internal/voice/`, `api/internal/audio/`, `api/internal/tts/` (or shrink to thin pass-through adapters if any web-console-specific behavior remains).
5. Update `capabilities.Known["audio-tools"]` checker — already wired; ScenarioChecker starts returning `available` automatically.
6. UI: change `ui/src/domains/audio/index.ts` re-exports from `hooks/voice/**` / `hooks/tts/**` to the audio-tools client; delete `ui/src/hooks/voice/**` and `ui/src/hooks/tts/**`.
7. Migrate remaining UI consumers (Workspace, TerminalPane, settings) to import from `domains/audio` (deferred ratchet — see Phase 6 extension below).

Preexisting failing tests not caused by this work:
- `TestDocsManifestResolves` and `TestDocsNoStaleOldPaths` reference removed
  doc paths (`DESIGN.md`, `concepts/FLOWS.md`, `internal/PROGRESS.md`, etc.).
Stale `docs/manifest.json` and `docs/START-HERE.md`; unrelated to audio
extraction prep.

## Work ladder

- Rung: W3 (implementation)
- Evidence: W0 comparison remains aligned: goals `hosted-cloud-tier-foundation` and `portal-front-door` do not contradict the archive plan, while `OT-P0-003` and `OT-P0-008` require durable session continuity and drawer controls. W1 passes with `business-health validate scenario web-console`; W2 passes with `vrooli scenario requirements validate web-console`, both with zero findings after repairing legacy statuses, stale validation refs, orphan targets, and the missing remote-terminal target. The archive implementation itself does not yet exist.
- Blocker: none; proceed through the scenario maturity ladder while implementing the archive plan.
- Measured: 2026-08-18.
