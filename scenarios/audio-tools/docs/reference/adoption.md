# Adoption — embedding audio-tools in a consumer scenario

> Plan: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`

This document is the copy-paste-ready recipe a consumer scenario follows to adopt audio-tools as its STT / TTS / summarize / audio-processing backend, plus the UI components from `@audio-tools/embed`.

## 1. Declare the dependency

In `<consumer>/.vrooli/service.json`:

```json
{
  "dependencies": {
    "scenarios": {
      "audio-tools": {
        "required": false,
        "startup_policy": "try_start",
        "description": "Shared audio capabilities (STT, TTS, summarization, audio processing).",
        "degraded_behavior": "Audio features degrade with a visible 'audio degraded' UI state when audio-tools is unreachable."
      }
    }
  }
}
```

Flip `required: false → true` after a soak period (Phase I9 flag-day in web-console's case).

## 2. Build an integration adapter

```
<consumer>/api/integrations/audiotools/
  client.go      // generated Connect clients + discovery + refresh-on-failure
  discovery.go   // EnvResolver / CachedResolver wrapping api-core/discovery
  contracts.go   // NormalizeError + ErrUnavailable / ErrInsufficientCredits / ErrInvalidArgument
```

See [web-console's adapter](../../../web-console/api/integrations/audiotools/) for the worked example.

## 3. Wire Remote* port adapters

If your scenario uses an internal ports interface (audioports.SpeechToText / TextToSpeech / SpeechTextProcessor), implement Remote* variants that delegate to the integration adapter:

```go
srv.SetSpeechToText(&audioports.RemoteSpeechToText{Client: client})
srv.SetTextToSpeech(&audioports.RemoteTextToSpeech{Client: client})
srv.conversationStore.SetSpeechProcessor(&audioports.RemoteSpeechTextProcessor{Client: client})
```

## 4. Embed the UI

```tsx
// In <consumer>/ui/src/domains/audio/index.ts
export {
  VoiceInputButton,
  AudioPlayerBar,
  EnableAudioBanner,
  MicReadinessIndicator,
  VoiceRejectionBanner,
  VoiceCommandSuggestion,
  VoiceSettingsPanel,
  TtsSettingsPanel,
} from "@audio-tools/embed";
```

Existing import sites under `ui/src/...` keep importing from `domains/audio`; the boundary file is the only place that needs to change when the underlying source flips.

## 5. Calling the API from a non-Go consumer

```bash
# Transcribe via Connect-RPC JSON over HTTP/1.1
curl -X POST http://localhost:${AUDIO_TOOLS_PORT}/vrooli.audio_tools.v1.stt.STTService/Transcribe \
  -H 'Content-Type: application/json' \
  -H 'X-Audio-BYOK-Provider: openai-whisper' \
  -H 'X-Audio-BYOK-Key: sk-...' \
  -d '{"audio":"<base64-bytes>","format":"wav","language":"en"}'

# Synthesize
curl -X POST http://localhost:${AUDIO_TOOLS_PORT}/vrooli.audio_tools.v1.tts.TTSService/Synthesize \
  -H 'Content-Type: application/json' \
  -d '{"text":"hello","voice":"voice.feminine.warm","response_format":"mp3"}' \
  --output out.mp3
```

## 6. Provider routing checklist

- BYOK → Vrooli → Local is **fixed**. Do not pass a custom precedence.
- `ErrInsufficientCredits` (Connect code `ResourceExhausted`) **short-circuits**. Don't retry.
- `ErrUnknownBYOKProvider` / `ErrMissingBYOKProvider` (Connect code `InvalidArgument`) terminates without fallback. Validate `X-Audio-BYOK-Provider` before calling.
- Canonical voice IDs are exactly: `voice.feminine.warm`, `voice.feminine.neutral`, `voice.masculine.warm`, `voice.masculine.neutral`, `voice.neutral.default`. Provider-specific names go through `voice_overrides["tier:provider-id"]`.

## 7. Restart resilience

The integration adapter re-resolves the audio-tools URL on any transport failure. Consumers do NOT need to restart on an audio-tools deploy. If `required: true`, consumers fail-fast on startup if audio-tools cannot be reached — surface an actionable operator error in that case.

## 8. Test fakes

For unit tests, stand up a fake audio-tools Connect server in-process. The provider-chain handlers in `scenarios/audio-tools/handlers/*/` are the canonical mock targets; embed each handler's Connect server with stubbed methods returning fixed responses.
