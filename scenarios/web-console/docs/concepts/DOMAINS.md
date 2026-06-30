# Web Console API Domains

This document buckets every existing web-console API route into a **domain**.
A domain is the migration unit for the Connect-RPC rewrite: each domain
becomes one proto package, one `api/internal/<domain>/` business-logic
package, one `api/handlers/<domain>/` transport package, and one entry in
`api/internal/modules/registry.go`.

The buckets here are the authoritative migration plan. The order of the
**Migration order** section is the order in which PRs should land.

## Conventions

- **Proto package**: `vrooli.web_console.v1.<domain>` →
  `packages/proto/schemas/web-console/v1/<domain>/<domain>.proto`.
- **Handler package**: `scenarios/web-console/api/handlers/<domain>/`
  exporting `Endpoints []module.EndpointDescriptor` plus the Connect
  service implementation.
- **Business logic**: `scenarios/web-console/api/internal/<domain>/`,
  imported by the handler.
- **REST exception** (the only mechanically-allowed escape hatch): tag
  the descriptor with `RESTException{Reason: …}` where Reason is one of
  `multipart_upload`, `webhook_receiver`, `third_party_shape`,
  `ops_probe`. Anything else must be a Connect procedure.
- WebSockets and long-lived streams are **not** Connect today; they will
  migrate to Connect server-streaming (or stay tagged `third_party_shape`
  if the browser/CDP/electron-updater wire shape forces it). Streams
  migrate last so the typed-RPC pattern is well-trodden before we touch
  them.

## Domains

### `health` ✅ migrated (Phase 0)

| Path | Method | Notes |
|---|---|---|
| `/health` | GET | `RESTException: ops_probe` — lifecycle systems / probes |
| `/api/v1/health` | GET | same handler, client-facing mount |

Both paths share one handler; the `/api/v1/health` mount is intentional
(client callers) but only one descriptor lives in the registry — the
`/health` ops-probe form.

---

### `sessions` — core session lifecycle

| Path | Method | Connect RPC (proposed) |
|---|---|---|
| `/api/v1/sessions` | POST | `Sessions.Create` |
| `/api/v1/sessions` | GET | `Sessions.List` |
| `/api/v1/sessions/recoverable` | GET | `Sessions.ListRecoverable` |
| `/api/v1/sessions/recoverable/{id}` | DELETE | `Sessions.DismissRecoverable` |
| `/api/v1/sessions/{id}/recover` | POST | `Sessions.Recover` |
| `/api/v1/sessions/{id}` | GET | `Sessions.Get` |
| `/api/v1/sessions/{id}` | DELETE | `Sessions.Delete` |
| `/api/v1/sessions/{id}/policy` | GET | `Sessions.GetPolicy` |
| `/api/v1/sessions/{id}/policy` | PUT | `Sessions.UpdatePolicy` |

Policy could live in its own `session_policy` domain, but it shares
session ownership rules and the policy struct is small; keep it inside
`sessions` until either the message or the access-control story grows.

---

### `terminal` — WebSocket terminal I/O and image upload

| Path | Method | Notes |
|---|---|---|
| `/api/v1/sessions/{id}/ws` | GET | WebSocket. Migrate to Connect server-stream **last**, after the static RPC domains are landed. `RESTException: third_party_shape` until then. |
| `/api/v1/sessions/{id}/upload` | POST | `RESTException: multipart_upload` — image upload for path injection. Stays multipart even post-Connect. |

---

### `workspace` — cross-device layout, panes, tab groups

| Path | Method | Connect RPC (proposed) |
|---|---|---|
| `/api/v1/workspace/layout` | GET | `Workspace.GetLayout` |
| `/api/v1/workspace/layout` | PUT | `Workspace.SaveLayout` |
| `/api/v1/workspace/panes/{session_id}` | PUT | `Workspace.UpdatePane` |
| `/api/v1/workspace/panes/{session_id}` | DELETE | `Workspace.DeletePane` |
| `/api/v1/workspace/groups` | POST | `Workspace.CreateGroup` |
| `/api/v1/workspace/groups/{id}` | PUT | `Workspace.UpdateGroup` |
| `/api/v1/workspace/groups/{id}` | DELETE | `Workspace.DeleteGroup` |

---

### `conversation` — conversation history per session

| Path | Method | Connect RPC (proposed) |
|---|---|---|
| `/api/v1/sessions/{id}/conversation` | GET | `Conversation.Get` |
| `/api/v1/sessions/{id}/conversation/cursor` | PUT | `Conversation.UpdateCursor` |
| `/api/v1/sessions/{id}/conversation/{eventId}/summarize` | POST | `Conversation.SummarizeEvent` |

File-reference resolution/preview moved out of this domain into its own
`file_preview` domain (see below) so the conversation domain stays
semantic conversation state.

---

### `file_preview` — resolve local file refs + serve preview bytes

| Path | Method | Transport |
|---|---|---|
| `FilePreviewService/Resolve` | POST | Connect-RPC — path → preview metadata + opaque `preview_id` + `blob_url` |
| `FilePreviewService/GetTextContent` | POST | Connect-RPC — bounded UTF-8 for text kinds (≤1 MiB) |
| `/api/v1/sessions/{id}/file-previews/{previewId}/blob` | GET/HEAD | REST exception (`ops_probe`) — byte-range stream for native `<img>/<video>/<audio>/<iframe>` |

A reusable subsystem (not conversation state): message links/chips are
today's only caller, but agent records, BAS runs, and plan drill-downs can
call the same resolve → preview-id → blob/text path. Binary/media bytes
never travel through Connect. See `docs/concepts/ARCHITECTURE.md#file-preview`.

---

### `settings` — user preferences

| Path | Method | Connect RPC (proposed) |
|---|---|---|
| `/api/v1/settings/session-defaults` | GET | `Settings.GetSessionDefaults` |
| `/api/v1/settings/session-defaults` | PUT | `Settings.UpdateSessionDefaults` |

Small but standalone; first migration after the substrate because it has
no streams, no auth nuances, and no shared message shapes.

---

### `shortcuts` — keyboard shortcut profiles

| Path | Method | Connect RPC (proposed) |
|---|---|---|
| `/api/v1/shortcuts` | GET | `Shortcuts.GetEffective` |
| `/api/v1/shortcuts/profiles` | GET | `Shortcuts.ListProfiles` |
| `/api/v1/shortcuts/profiles` | PUT | `Shortcuts.UpsertProfile` |
| `/api/v1/shortcuts/profiles/{id}` | DELETE | `Shortcuts.DeleteProfile` |

---

### `ai` — AI command generation and provider config

| Path | Method | Connect RPC (proposed) |
|---|---|---|
| `/api/v1/ai/generate` | POST | `AI.Generate` |
| `/api/v1/ai/suggest` | POST | `AI.Suggest` |
| `/api/v1/ai/config` | GET | `AI.GetConfig` |
| `/api/v1/ai/config` | PUT | `AI.UpdateConfig` |
| `/api/v1/ai/health` | GET | `AI.GetHealth` |

All five share the AI provider abstraction; one domain. `ai/health` is a
plain Connect unary, not a service-level probe — it returns provider
reachability, distinct from the lifecycle `/health`.

---

### `capabilities` — voice-input feature detection

| Path | Method | Connect RPC (proposed) |
|---|---|---|
| `/api/v1/capabilities` | GET | `Capabilities.Get` |
| `/api/v1/capabilities/liveness` | GET | `Capabilities.Liveness` |

Smallest non-health domain. Good second migration after `settings` to
exercise the substrate on a read-only endpoint pair.

---

### `voice` — voice input pipeline

| Path | Method | Notes |
|---|---|---|
| `/api/v1/voice/transcribe` | POST | `Voice.Transcribe` (Connect unary) |
| `/api/v1/voice/stream` | GET | WebSocket. `RESTException: third_party_shape` until streams migrate. |
| `/api/v1/voice/config` | GET/PUT | `Voice.GetConfig` / `Voice.UpdateConfig` |
| `/api/v1/voice/wakeword` | GET/PUT/DELETE | `Voice.GetWakeWord` / `UpdateWakeWord` / `DeleteWakeWord` |
| `/api/v1/voice/speaker/config` | GET/PUT | `Voice.GetSpeakerConfig` / `UpdateSpeakerConfig` |
| `/api/v1/voice/speaker/status` | GET | `Voice.GetSpeakerStatus` |
| `/api/v1/voice/speaker/profiles` | GET | `Voice.ListSpeakerProfiles` |
| `/api/v1/voice/speaker/enroll` | POST | `Voice.EnrollSpeaker` |
| `/api/v1/voice/speaker/profile` | DELETE | `Voice.ClearSpeakerBinding` |
| `/api/v1/voice/speaker/profile/remove` | POST | `Voice.RemoveSpeakerProfile` |
| `/api/v1/voice/speaker/profile/delete` | POST | `Voice.DeleteSpeakerProfile` |

Largest single domain. Plan to split internal/voice/ by sub-feature
(transcribe, wakeword, speaker) for code organization, but they share
one proto service since the UI treats them as one capability.

The two POST-for-delete endpoints (`profile/remove`, `profile/delete`)
are historical and will collapse into proper RPCs on migration —
`RemoveSpeaker` and `DeleteSpeaker` with body args. No REST exception
needed; the legacy URL shapes go away.

**Extraction destination (2026-05-16):** STT, streaming transcription, VAD
events, wake-word, and speaker verification primitives are planned for
extraction into the future `scenarios/audio-tools` scenario per the
`continuous-audio-platform` initiative. As of 2026-05-16 the domain owns:
`api/internal/voice/types.go` (HandlerService, Backend, transport shapes),
`api/internal/voice/handler_adapter.go` (validation + orchestration), and
`api/internal/voice/service.go` (state + persistence + Whisper client +
SpeakerClient). `handlers/voice` is now transport-only (re-exports +
Connect-RPC). Web-console will keep voice-command resolution, transcript
insertion gates, and active-pane routing after extraction; everything else
moves to audio-tools.

---

### `tts` — text-to-speech (Kokoro backend)

| Path | Method | Connect RPC (proposed) |
|---|---|---|
| `/api/v1/tts/config` | GET/PUT | `TTS.GetConfig` / `TTS.UpdateConfig` |
| `/api/v1/tts/status` | GET | `TTS.GetStatus` |
| `/api/v1/tts/events` | POST | `TTS.PostEvent` |
| `/api/v1/tts/summarize/config` | GET/PUT | `TTS.GetSummarizeConfig` / `TTS.UpdateSummarizeConfig` |
| `/api/v1/tts/synthesize` | POST | `TTS.Synthesize` |
| `/api/v1/tts/cache/{eventId}` | GET | `TTS.GetCache` — returns audio bytes |
| `/api/v1/tts/voices` | GET | `TTS.ListVoices` |

`tts/synthesize` and `tts/cache/{eventId}` return audio bytes. Connect
can carry `bytes` payloads, so these stay Connect — no REST exception.
If we later expose direct browser `<audio src="…">` consumption, that
becomes a separate REST asset endpoint with `third_party_shape`.

**Extraction destination (2026-05-16):** TTS synthesis, voice catalog,
text normalization, paragraph splitting, summarization, and caching are
planned for extraction into `scenarios/audio-tools`. As of 2026-05-16,
`api/internal/tts/` owns: `types.go` (HandlerService, Config, Status,
Synth/Cache/Voice DTOs), `service.go` (HandlerService impl backed by
function-pointer `Deps`), `kokoro_synthesize.go`, `kokoro_voices.go`,
`normalizer.go` (`NormalizeTextForSpeech`), `chunker.go`
(`SplitIntoSpeechParagraphs`, `TTSMaxChunkLength`). Web-console retains
hook routing, terminal/conversation attribution, playback ack storage,
auto-TTS trigger policy. The cache/summarizer/config persistence
files in `package main` (`tts_cache.go`, `tts_summarizer.go`,
`tts_summarization_service.go`, `tts_config.go`, `tts_summarize_config.go`)
are scheduled for the next extraction-prep pass — see PROBLEMS.md §10.

---

### `hooks` — Claude Code lifecycle hooks (inbound)

| Path | Method | Notes |
|---|---|---|
| `/api/v1/hooks/stop` | POST | `RESTException: webhook_receiver` — called by Claude Code CLI |
| `/api/v1/hooks/prompt-submit` | POST | `RESTException: webhook_receiver` — same |

These are inbound webhooks from a CLI we do not control. They stay REST
with the `webhook_receiver` tag and never become Connect.

---

### `events` — SSE event stream

| Path | Method | Notes |
|---|---|---|
| `/api/v1/events` | GET | SSE. `RESTException: third_party_shape` — EventSource is a browser primitive; migrating to Connect server-stream is possible but lower priority than typed RPCs. |

---

### `metrics` — Prometheus scrape target

| Path | Method | Notes |
|---|---|---|
| `/api/v1/metrics` | GET | `RESTException: third_party_shape` — Prometheus wire format. Never Connect. |

---

## Migration order

Each step is one PR. Each PR adds a proto package, an
`internal/<domain>/` package, a `handlers/<domain>/` package, registers
the handler in `internal/modules/registry.go`, removes the old
`HandleFunc` lines from `main.go`, regenerates `.vrooli/endpoints.json`,
and updates UI/CLI callers to use the generated Connect client.

Ordering principles: smallest blast radius first; finish all
static-RPC domains before touching streams; one risk axis per PR.

1. **`settings`** — 2 routes, no streams, no fan-out. Proves the
   end-to-end pattern (proto → gen → handler → UI client → CLI mapping).
2. **`capabilities`** — 2 routes, read-only. Confirms the pattern on a
   second domain with no DB writes.
3. **`shortcuts`** — 4 routes, simple CRUD over profiles.
4. **`workspace`** — 7 routes; first real domain-shape stress test
   (nested IDs, multiple resources sharing one service).
5. **`ai`** — 5 routes; introduces the provider-config message types
   used elsewhere.
6. **`conversation`** — 5 routes; first domain with rich nested
   messages (events, file refs) — exercises proto reuse.
7. **`sessions`** — 9 routes; the core domain. By this point the
   pattern is well-trodden, so we land the load-bearing migration with
   the lowest pattern-uncertainty.
8. **`tts`** — 9 routes incl. bytes payloads. First domain with binary
   responses through Connect.
9. **`voice`** — 17 routes excluding the WebSocket stream. Largest
   single PR; consider splitting by sub-feature if review burden grows.
10. **`hooks`** — 2 routes, REST-tagged only (`webhook_receiver`). No
    proto, just registry entries. Fast.
11. **`metrics`** — 1 route, REST-tagged (`third_party_shape`).
    Registry entry only.
12. **`events`** — 1 SSE route. REST-tagged until streams migrate.
13. **`terminal/upload`** — multipart upload tagged
    `multipart_upload`; the upload doesn't need a stream rewrite.
14. **Streams** (terminal WS, voice stream, events SSE) — final phase,
    migrate to Connect server-streaming together so the streaming
    pattern is decided once. May stay REST-tagged if Connect-stream
    doesn't fit the browser-side constraint (xterm.js, EventSource).

## When to add a new domain

Add a domain if the messages or persistence boundary is independent of
existing domains. Do **not** add a domain just to fit a URL prefix; the
URL is incidental and disappears once everything is Connect. Likewise,
do not split a domain into multiple proto services just because the
handler file is getting long — split the internal/ package instead.
