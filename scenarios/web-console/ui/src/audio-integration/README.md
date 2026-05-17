# audio-integration

Web-console's audio surface. Talks **only** to web-console's own API
(same-origin Connect transport) — never to audio-tools directly.

## Architecture

```
                  same-origin (Connect-Web)
audio-integration ───────────────────────────▶ web-console API
                                                    │
                                                    │ Connect-RPC (server↔server)
                                                    ▼
                                               audio-tools API
```

The browser never sees audio-tools' host. Web-console's API owns the
inter-scenario hop through `internal/audioports.Remote*` adapters.

## What lives here

- `api/voice.ts`, `api/tts.ts` — Connect clients for web-console's
  `AudioAdminService` + `AudioRuntimeService`, plus the type-shaped
  wrappers consumers import.
- `api/protomap.ts` — boundary translators between the legacy
  string-typed embed surface and the web-console-owned proto enums
  (`AudioFormat`, `SpeakerMode`, `RejectBehavior`, etc.).
- `hooks/voice/*`, `hooks/tts/*` — the embed-style voice + TTS hooks
  (VAD, wakeword, providers) that consumers use directly.
- `index.ts` — public re-exports for the rest of the web-console UI.

## Hard rule

The boundary test `src/__tests__/audio-boundary.test.ts` enforces that
**no file under `src/`** imports `@vrooli/proto-types/audio-tools/*` or
the retired `@audio-tools/embed` package. New audio surfaces should be
added to web-console's `audio_admin` / `audio_runtime` proto schemas
(in `packages/proto/schemas/web-console/v1/`) and exposed by the
matching handler in `scenarios/web-console/api/handlers/`.
