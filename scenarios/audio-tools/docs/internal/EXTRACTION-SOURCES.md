# Extraction Sources

> **Source scenario**: `scenarios/web-console`
> **Extraction date**: 2026-05-16
> **Plan**: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`
> **Provenance**: web-console committed extraction-prep on branch `agi` (HEAD ~ 2026-05-16) before this scenario was generated. The prep pass classified every reusable-vs-glue file under `domains/audio/README.md` and migrated callsites through `audioports.SpeechToText / TextToSpeech / SpeechTextProcessor`.

## Why this scenario exists

Reusable audio capability lived inside web-console under `internal/{tts,voice,audio,audioports}` and `ui/src/{hooks/voice,hooks/tts,components}`. It could not be reused by swarm-manager, agent-manager, phone-agent, or future scenarios without copying. The extraction-prep pass established Ports + a re-export boundary; this scenario is the destination that flips the Local* adapters into a remote service.

## Boundary classification

### Backend (Go) — ported wholesale

| Web-console path | Audio-tools destination | Notes |
|---|---|---|
| `api/internal/tts/cache.go` | `api/internal/tts/cache.go` | Content-addressable cache; web-console keeps event→content-key wrapper. |
| `api/internal/tts/chunker.go` | `api/internal/tts/chunker.go` | Pure. |
| `api/internal/tts/config.go` | `api/internal/tts/config.go` | Defaults moved to audio-tools state. |
| `api/internal/tts/normalizer.go` | `api/internal/tts/normalizer.go` | Pure; exposed via TTS proto NormalizeForSpeech RPC. |
| `api/internal/tts/summarizer.go` | `api/internal/tts/summarizer.go` | Will be wrapped by `internal/summarize.OllamaClient` for SummarizeService.Summarize. |
| `api/internal/tts/summarization_service.go` | `api/internal/tts/summarization_service.go` | Inflight dedupe + cooldown; used by Local summarize provider. |
| `api/internal/tts/summarize_config.go` | `api/internal/tts/summarize_config.go` | Settings storage moves to audio-tools state. |
| `api/internal/tts/service.go` | `api/internal/tts/service.go` | Handler-facing TTS service struct. |
| `api/internal/tts/kokoro_synthesize.go` | `api/internal/tts/kokoro_synthesize.go` | Local backend; will be wrapped by `internal/ai/ttschain.LocalProvider`. |
| `api/internal/tts/kokoro_voices.go` | `api/internal/tts/kokoro_voices.go` | Adapter-side voice catalog; canonical mapping added in `internal/tts/voice_catalog.go`. |
| `api/internal/tts/types.go` | `api/internal/tts/types.go` | Domain types. |
| `api/internal/voice/*.go` (all) | `api/internal/voice/` | Whisper transcribe, stream WS, speaker verification, wake-word. |
| `api/internal/audio/transcode.go` | `api/internal/audio/transcode.go` | ffmpeg wrapper. |
| `api/internal/audioports/{ports,local_stt,local_tts,local_processor}.go` | `api/internal/audioports/` | Local-tier ports stay here as the in-process binding used by chain LocalProvider. |
| `api/internal/capabilities/*` | `api/internal/capabilities/` | Resource availability registry used by voice/STT readiness probes. |

### Backend (Go) — to extract or rewrite

| Web-console path | Audio-tools destination | Notes |
|---|---|---|
| (in `internal/tts/summarizer.go`) Ollama client | `api/internal/summarize/ollama_client.go` | Lift the Ollama call out so SummarizeService has a clean Local backend independent of TTS chunking. |
| (new) | `api/internal/ai/{sttchain,ttschain,summarizechain}/*` | Mirror BAS pattern; chain owns precedence + availability + factory + per-request client. |
| (new) | `api/internal/byok/*` | Per-provider adapter registry: openai-whisper, deepgram, openai-tts, elevenlabs, openrouter. |
| (new) | `api/internal/lpbs/*` | LPBS clients per capability; flag-off by default. |
| (new) | `api/internal/session/{session,events,bargein,resample}.go` | Voice-session abstraction with pub/sub event bus. |
| (new) | `api/internal/transports/browser/ws_handler.go` | Browser-voice WS over the session abstraction. |
| (new) | `api/handlers/{stt,tts,summarize,audio,session,settings,usage}/` | Connect-RPC handler packages. |
| (delete) `api/internal/notes/*` + `api/handlers/notes/*` | (none) | Template example; not part of audio-tools surface. |

### Backend — stays in web-console (conversation/terminal glue)

| Web-console path | Reason |
|---|---|
| `api/tts_cache.go` (orchestration) | Conversation-event-keyed cache invalidation; sits above audio-tools cache. |
| `api/tts_hook_handler.go` | Hook routing wired into the terminal pane. |
| `api/tts_router_test.go` | Tests the hook router. |
| `api/internal/sessioncontext/*` (orchestration) | Web-console-specific session linking. |

### Frontend — ported wholesale (reusable hooks/components)

| Web-console path | Audio-tools destination | Notes |
|---|---|---|
| `ui/src/hooks/voice/VoiceStreamProvider.ts` | `ui/src/hooks/voice/VoiceStreamProvider.ts` | |
| `ui/src/hooks/voice/WhisperProvider.ts` | `ui/src/hooks/voice/WhisperProvider.ts` | URL resolution points at audio-tools API instance. |
| `ui/src/hooks/voice/WebSpeechProvider.ts` | `ui/src/hooks/voice/WebSpeechProvider.ts` | |
| `ui/src/hooks/voice/vad.ts` | `ui/src/hooks/voice/vad.ts` | |
| `ui/src/hooks/voice/audioUtils.ts` | `ui/src/hooks/voice/audioUtils.ts` | |
| `ui/src/hooks/voice/sharedAudioContext.ts` | `ui/src/hooks/voice/sharedAudioContext.ts` | |
| `ui/src/hooks/voice/micReadiness.ts` | `ui/src/hooks/voice/micReadiness.ts` | |
| `ui/src/hooks/voice/wakeword/*` | `ui/src/hooks/voice/wakeword/` | Pure MFCC+DTW engine. |
| `ui/src/hooks/voice/types.ts` | `ui/src/hooks/voice/types.ts` | |
| `ui/src/hooks/tts/KokoroProvider.ts` | `ui/src/hooks/tts/KokoroProvider.ts` | URL resolution at audio-tools API. |
| `ui/src/hooks/tts/BrowserTTSProvider.ts` | `ui/src/hooks/tts/BrowserTTSProvider.ts` | |
| `ui/src/hooks/tts/types.ts` | `ui/src/hooks/tts/types.ts` | |
| `ui/src/components/VoiceMicButton.tsx` | `ui/src/embed/VoiceInputButton.tsx` | Renamed + generalized (terminal-input gate removed; accepts `commandHandler` callback). |
| `ui/src/components/AudioPlayerBar.tsx` | `ui/src/embed/AudioPlayerBar.tsx` | Generalized; accepts `audioUrl | audioBytes`. |
| `ui/src/components/VoiceCommandSuggestion.tsx` | `ui/src/embed/VoiceCommandSuggestion.tsx` | |
| `ui/src/components/VoiceRejectionBanner.tsx` | `ui/src/embed/VoiceRejectionBanner.tsx` | |
| `ui/src/components/EnableAudioBanner.tsx` | `ui/src/embed/EnableAudioBanner.tsx` | |
| `ui/src/components/tts/*` (reusable subset) | `ui/src/embed/` | Mic-readiness indicator + similar. |
| `ui/src/components/settings/VoiceInputSection.tsx` | `ui/src/embed/VoiceSettingsPanel.tsx` | Rendered as `<VoiceSettingsPanel>` for consumer settings pages. |
| `ui/src/components/settings/TtsSettingsSection.tsx` | `ui/src/embed/TtsSettingsPanel.tsx` | Same. |

### Frontend — stays in web-console (conversation/terminal orchestration)

| Web-console path | Reason |
|---|---|
| `ui/src/hooks/useVoiceInput.ts` | Terminal-input gating + conversation cursor wiring. |
| `ui/src/hooks/useTextToSpeech.ts` | Conversation playback ack tracking. |
| `ui/src/hooks/voice/commands.ts` + `commandParser.ts` | Terminal command vocabulary. |
| `ui/src/domains/audio/index.ts` | Re-export boundary; the file stays — only its underlying imports flip to `@audio-tools/embed`. |
| Web-console-only component subtrees referencing terminal panes, conversation cursors, etc. | Out of scope. |

## Provenance commits

- web-console extraction-prep: branch `agi` HEAD ~ 2026-05-16 (commit hash captured at time of port).
- audio-tools scaffold: `vrooli scenario generate react-vite --id audio-tools --display-name "Audio Tools" --design vrooli-default` (2026-05-16).
- 7 audio-tools proto schemas authored (stt, tts, summarize, audio, session, settings, usage) and verified via `make lint`, `make generate`, `make check`.
- 5 backend domain packages copied + imports rewritten (`web-console/internal/*` → `audio-tools/internal/*`), `gorilla/websocket` dep added, `go test ./internal/{audio,tts,voice,capabilities,audioports}/...` green.

## Future agent gate

Before adding new backend code under `internal/`, future agents MUST read this file and confirm whether the new code belongs to a ported domain (extend in place), a new audio-tools-only construct (add to `internal/ai/`, `internal/session/`, `internal/byok/`, `internal/lpbs/`), or web-console-specific glue (does not belong here at all — push back to web-console).
