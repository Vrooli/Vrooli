# `domains/audio` — Frontend Audio Adoption Boundary

This directory is the planned frontend home for code that will be adopted from
the future `scenarios/audio-tools` scenario. It is the UI mirror of the
backend `internal/audioports` package: a narrow surface that conversation /
terminal / settings components depend on, so the underlying provider
implementation can swap from in-process local logic to an audio-tools client
without touching the consumer surface.

## What lives here

The `index.ts` re-exports the audio capability surface that the rest of the
UI is allowed to import. Today those re-exports point at the existing
`hooks/voice/**` and `hooks/tts/**` modules — no behaviour change. The
boundary exists primarily to:

1. Make the seam visible to readers and code review.
2. Let the audio-tools adoption ship as a single redirect at the re-export
   layer rather than as a sprawling import-path rename.
3. Capture which hooks/components are "audio capability" (movable to
   audio-tools) vs "terminal/conversation glue" (stays in web-console).

## Classification

### Reusable audio capability (future audio-tools owners)

- `hooks/voice/VoiceStreamProvider.ts` — WS streaming transcription
- `hooks/voice/WhisperProvider.ts` — HTTP batch transcription
- `hooks/voice/WebSpeechProvider.ts` — browser fallback
- `hooks/voice/vad.ts` — Voice Activity Detection (pure functions)
- `hooks/voice/audioUtils.ts` — filter chain (pure function)
- `hooks/voice/sharedAudioContext.ts` — singleton AudioContext
- `hooks/voice/micReadiness.ts` — mic permission/readiness state
- `hooks/voice/wakeword/**` — wake-word matching (pure)
- `hooks/tts/KokoroProvider.ts` — Kokoro TTS playback
- `hooks/tts/BrowserTTSProvider.ts` — browser TTS fallback
- `hooks/tts/types.ts` — TTSProvider interface

### Web-console specific (stays after extraction)

- `hooks/voice/commands.ts`, `commandParser.ts` — voice-command vocabulary
  targeted at the terminal
- `hooks/voice/audioCues.ts`, `activity.ts` — recording UX feedback
- `hooks/useVoiceInput.ts` — orchestrator wiring providers into the active
  terminal pane (will keep web-console terminal-input integration)
- `hooks/useTextToSpeech.ts` — TTS orchestration tied to conversation
  cursor / listened state
- `domains/tts-playback/**` — conversation playback state machine
- `components/VoiceMicButton.tsx`, `VoiceCommandSuggestion.tsx`,
  `VoiceRejectionBanner.tsx` — UI affordances coupled to terminal panes
- `components/AudioPlayerBar.tsx`, `EnableAudioBanner.tsx` — playback UI
  tied to conversation events

## Migration plan

When `scenarios/audio-tools` lands:

1. The reusable-capability files above move into `audio-tools/ui/`.
2. `domains/audio/index.ts` re-points to `@audio-tools/ui` (or whichever
   client surface audio-tools publishes) instead of the in-tree hooks.
3. Web-console-specific consumers (Workspace, TerminalPane, terminal input
   gate, settings sections) keep their existing imports — the rename
   happens entirely at the boundary.

See also: `docs/internal/SEAMS.md#frontend-audio-adoption-boundary-seam-ui`
and `api/internal/audioports/` (the backend counterpart).
