# INVARIANTS

Non-negotiable contracts. Changing any of these is an RFC-level decision, not a refactor.

## Provider precedence (sttchain / ttschain / summarizechain)

1. The order is **declared per capability**, not chosen per request:
   - **STT and TTS**: Local → BYOK → Vrooli (`LocalFirst: true`). Speech
     prefers the on-device route so a local session does not leave the host.
   - **Summarize**: BYOK → Vrooli → Local (the historical order). The
     summarizer's local path is a resource-ollama role, not a speech engine.
2. `ErrInsufficientCredits` from the Vrooli tier **short-circuits** the chain.
   It does NOT fall through to Local. Credit exhaustion is a billing failure,
   not an availability failure.
3. `ErrUnknownBYOKProvider` and `ErrMissingBYOKProvider` terminate the chain
   without fallback. Silent provider selection is forbidden — the caller must
   name the BYOK provider explicitly.
4. Per-tier availability is cached: BYOK and Vrooli use the configured TTLs;
   Local is probed by the actual call (or the engine health probe) and is never
   pre-probed into a false negative.
5. A provider that is not declared in the provider catalog cannot be selected.
   A catalog entry that has no executor is a `planned`/`watch` declaration, not
   a runnable provider.
6. **Encoded in**: `api/internal/ai/chains/tiered/` and the three
   capability-specific `NewChain` adapters (`sttchain`, `ttschain`,
   `summarizechain`).

## Canonical voice catalog (TTS)

1. Canonical IDs are exactly: `voice.feminine.warm`, `voice.feminine.neutral`, `voice.masculine.warm`, `voice.masculine.neutral`, `voice.neutral.default`. Adding a canonical ID requires updating every registered TTS adapter.
2. Every registered TTS adapter MUST declare a mapping for every canonical ID. Missing entries fail startup with a structured error.
3. Override resolution order: per-request `voice_overrides["tier:provider-id"]` → adapter's declared mapping → adapter default + warning log.
4. **Encoded in**: `api/internal/tts/voice_catalog.go`. `ttschain.Request.VoiceOverrides` carries per-call overrides.

## TTS cache

1. The audio-tools TTS cache is **content-addressable**. Key = SHA256(`voice|speed|format|text-content`).
2. Event-keyed invalidation lives in **consumer glue** (e.g., web-console conversation orchestration), not in audio-tools.
3. `GetCache` accepts either `content_hash` (direct hit) OR `event_id + voice + speed + version` (consumer-mapped lookup).
4. **Encoded in**: `tts.proto.GetCacheRequest`, `api/internal/tts/cache.go`.

## Voice-session contract

1. A session has at most one in-flight assistant action at a time. `MarkInflight` records the eventID; `ClearInflight` is called by the transport when the action completes naturally.
2. Barge-in (`BargeIn(reason)`) MUST: clear inflight, invoke the registered `CancelHook` (so the transport can stop streaming TTS), AND emit a `BargeInCancel` SessionEvent to every observer. If no inflight action is recorded, BargeIn is a no-op.
3. Multiple observers per session are first-class. Slow observers drop events rather than blocking the publisher.
4. Close emits a `SessionClosed` event then closes every observer channel. Subsequent `Subscribe` returns `ErrSessionClosed`.
5. **Encoded in**: `api/internal/session/session.go` + `session_test.go`.

## Credential handling

1. BYOK keys travel in per-request metadata headers (`X-Audio-BYOK-Key`, `X-Audio-BYOK-Provider`). They are never stored in audio-tools state with the bare key — only the redacted fingerprint (`sk-***abcd`) goes into the `BYOKCredentialSummary` returned by `SettingsService`.
2. The provider chain logs MUST NOT include the raw key in any form. The credential-redacting logger mirrors BAS's pattern.
3. CLI BYOK entry is via interactive secret prompt or environment variable, never as a command-line argument.
4. **Encoded in**: `api/internal/byok/`, `api/internal/byokstore/`, and the
   settings/BYOK Connect services. The CLI prompts interactively or reads an
   environment variable; it never accepts the secret as an argv value.

## Provider and engine identity

1. A **provider id** is the stable identity used by health, lifecycle,
   settings, usage, and operator commands (`whisper-stt`, `kyutai-stt`,
   `sherpa-stt`, `kokoro-tts`, `ollama`, `openai-whisper`).
2. An **engine id** is the STT stream-selection identity
   (`whisper-local`, `kyutai`, `sherpa-streaming`). Engine ids may appear in
   stream evidence and diagnostics; they must not silently stand in for a
   provider id on a health or lifecycle surface.
3. The provider catalog owns the mapping in both directions. Adding an engine
   without its provider (or the reverse) is a conformance failure.
4. Public results (`TranscribeResponse`, stream `done`, TTS/summarize results)
   report the canonical provider id. Internal evidence (soak diagnostics,
   experiment/eval cells) may report the engine id because it names what
   actually ran.
5. Test-only engines (`virtual-replay`) are never production providers and are
   never selectable from the production manifest.

## One declaration, generated projections

1. Every active provider/operation has exactly one declaration in the catalog
   seed. Consumers filter; they do not keep a second list.
2. Generated surfaces carry a source/codegen/drift header and are changed only
   by running the generator. A hand-edited generated file fails the drift gate.
3. Adding a provider of an existing kind is a data change plus at most one
   adapter; adding an operation in an existing engine/family is a declaration
   plus its executor.
4. A declaration without an executor is `planned`/`watch`, not `active`. An
   executor without a declaration fails the build.

## Model registry is a runtime surface

1. `ModelRegistryService.ListModels` and `ResolveModel` are projections of the
   validated seed. A surface that needs model facts asks the registry; it does
   not keep a second hand-written model list.
2. `ResolveModel(commercial = true)` fails closed: a record whose commercial use
   is not explicitly allowed is never returned, even when it ranks highest.
3. The registry mirrors the runtime authority it does not own. Ollama role
   policy lives in `resources/ollama/model-policy.json`; a drift test fails when
   the registry stops naming the policy's primary model and fallbacks.
4. A registry record never implies live availability. `readiness` describes
   whether an adapter exists, not whether a resource is running; health belongs
   to `health_status` and `provider_lifecycle`.

## Model default eligibility

1. A model may be selected as a commercial default only when its license permits
   commercial use and its readiness is `available` with a dated qualification
   receipt.
2. `candidate`, `experimental`, and `watch` models are visible but never become
   defaults by omission.
3. Non-commercial weights (for example CC BY-NC) require an explicit,
   operator-visible non-commercial mode; they are never a silent fallback.
4. A batch model is not a streaming model. Streaming eligibility comes from the
   engine/runtime facts, not from benchmark speed.

## External process invocation

1. ffmpeg/ffprobe run as argv-only `exec.CommandContext` calls. No shell, no
   string interpolation into a shell.
2. Pipe-driven invocations pass `-nostdin` and `-hide_banner` so headless and
   Windows hosts behave the same as an interactive terminal.
3. Every long-lived child process is bound to the session context and is closed
   on cancel/teardown; no process survives its session.

## Transport classification

1. The four `RESTReason*` constants in `templates/scenarios/react-vite/api/internal/module/module.go` are the only escape hatches from Connect-RPC for proto-owned operations. For audio-tools:
   - `multipart_upload`: none in P0.
   - `ops_probe`: `GET /health`.
   - `webhook_receiver`: none in P0.
   - `third_party_shape`: none in P0.
2. Browser voice streaming uses the proto-owned `STTService.TranscribeStream` Connect bidi endpoint. The retired `/api/v1/voice/stream` WebSocket route is not mounted.
3. No new endpoints in web-console for audio. Consumers reach audio-tools directly via the integrations adapter.

## Streaming v2 replay and segment delivery

1. A v2 audio chunk is retained until the server emits a processed-coverage
   acknowledgement or an explicit terminal failure. A stream close or empty
   final is not evidence of coverage.
2. The Connect bidi handler commits each durable transcript segment
   into the session ledger before emitting it. The identity is provider-supplied
   when available; otherwise it is deterministically derived from session,
   generation, and canonical sample range.
3. A Connect reconnect re-emits committed segments in canonical timeline
   order with `segmentId`. The browser provider retains IDs for the active turn and
   delivers each ID to host UI exactly once. This lets replay repair a missed
   server-to-browser delivery without duplicate composer or dictation text.
4. **Encoded in**: `handlers/stt/transcribe_stream.go`,
   `internal/stt/session/ledger.go`, and both PCM v2 browser providers.

## Greenfield discipline

1. No compatibility shims inside audio-tools. No legacy aliases. No `// removed` comments. No `_unused` named vars carried for symmetry.
2. Migration adapters in **consumers** (web-console) are allowed only if they have a named removal trigger.
3. Pure-function utilities that cross the scenario boundary are still owned by audio-tools and called via Connect-RPC (`tts.NormalizeForSpeech`, `tts.SplitParagraphs`). No shared `packages/audio-text-utils` package — wrap-not-use.
