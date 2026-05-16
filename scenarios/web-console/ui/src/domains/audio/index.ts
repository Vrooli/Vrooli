// Frontend audio adoption boundary.
//
// This file is the single re-export surface for code that will move into the
// future `scenarios/audio-tools` scenario. Conversation, terminal, settings,
// and other consumer modules should import from `domains/audio` rather than
// reaching directly into `hooks/voice/**` or `hooks/tts/**`. When audio-tools
// ships, redirecting these re-exports is the only change needed in this
// boundary file; consumer imports stay stable.
//
// Today these re-exports point at the existing in-tree hooks — extraction
// preparation, not extraction. See `domains/audio/README.md` for the
// classification of reusable vs web-console-specific code.

// -----------------------------------------------------------------------------
// Voice (STT) capability surface
// -----------------------------------------------------------------------------

export type {
  CommandSuggestion,
  TranscriptionProvider,
  VoiceRejection,
  VoiceState,
} from "../../hooks/voice/types";

export { VoiceStreamProvider } from "../../hooks/voice/VoiceStreamProvider";
export { WhisperProvider } from "../../hooks/voice/WhisperProvider";
export { WebSpeechProvider } from "../../hooks/voice/WebSpeechProvider";

// Pure-function utilities. After extraction these come from audio-tools.
export { createAudioFilterChain } from "../../hooks/voice/audioUtils";

// AudioContext / mic-readiness substrate.
export { getSharedAudioContext } from "../../hooks/voice/sharedAudioContext";

// Wake-word matching (pure). Reusable per README classification.
export {
  createWakeWordEngine,
  MIN_ENROLLMENT_SAMPLES,
  MAX_ENROLLMENT_SAMPLES,
} from "../../hooks/voice/wakeword";
export type {
  AudioFeatures,
  WakeWordTemplate,
} from "../../hooks/voice/wakeword";
export { useWakeWordTest } from "../../hooks/voice/wakeword/useWakeWordTest";

// -----------------------------------------------------------------------------
// TTS capability surface
// -----------------------------------------------------------------------------

export type {
  TTSPlaybackCapabilities,
  TTSPlaybackState,
  TTSProvider,
} from "../../hooks/tts/types";
export { KokoroProvider } from "../../hooks/tts/KokoroProvider";
export { BrowserTTSProvider } from "../../hooks/tts/BrowserTTSProvider";
