// audio-integration — canonical copy-paste reference for adopters of
// audio-tools.
//
// This folder exists verbatim in audio-tools/ui (canonical) and is copied
// byte-for-byte into each consumer scenario (e.g. web-console). There is
// no cross-scenario import path; each scenario owns its own copy. See
// README.md.

export { MicReadinessIndicator } from "./MicReadinessIndicator";
export type { MicReadinessIndicatorProps } from "./MicReadinessIndicator";

// Connect client + React context for consumers calling audio-tools RPCs.
export {
  createAudioToolsClient,
  AudioToolsProvider,
  useAudioToolsClient,
  useAudioToolsUnavailableReason,
  getActiveAudioToolsClient,
  setActiveAudioToolsClientForTesting,
} from "./client";
export type {
  AudioToolsClient,
  CreateAudioToolsClientOptions,
  AudioToolsProviderProps,
} from "./client";

// =============================================================================
// Voice (STT) capability surface.
// =============================================================================

export * from "./hooks/voice/types";
export * from "./hooks/voice/audioUtils";
export * from "./hooks/voice/audioCues";
export * from "./hooks/voice/activity";
export * from "./hooks/voice/vad";
export * from "./hooks/voice/sharedAudioContext";
export * from "./hooks/voice/micReadiness";
export { VoiceStreamProvider } from "./hooks/voice/VoiceStreamProvider";
export { WhisperProvider } from "./hooks/voice/WhisperProvider";
export { WebSpeechProvider } from "./hooks/voice/WebSpeechProvider";
export * from "./hooks/voice/wakeword";

// =============================================================================
// TTS capability surface.
// =============================================================================

export * from "./hooks/tts/types";
export { KokoroProvider } from "./hooks/tts/KokoroProvider";
export { BrowserTTSProvider } from "./hooks/tts/BrowserTTSProvider";

// =============================================================================
// API surfaces (audio-only operations bound to audio-tools).
// =============================================================================

export {
  createTtsApi,
  useTtsApi,
  synthesizeTTS,
  fetchCachedTTS,
  getTTSVoices,
  getTTSConfig,
  updateTTSConfig,
  getTTSSummarizeConfig,
  updateTTSSummarizeConfig,
  reportTTSEvent,
} from "./api/tts";
export type { TTSConfig, TTSSummarizeConfig, TTSVoiceInfo, TTSPlaybackEvent } from "./api/tts";

export {
  createVoiceApi,
  useVoiceApi,
  buildVoiceStreamWsUrl,
  transcribeAudio,
  transcribeAudioBypassFilter,
  transcribeAudioWithRetry,
  getVoiceStreamConfig,
  updateVoiceStreamConfig,
  getWakeWordConfig,
  updateWakeWordConfig,
  deleteWakeWordConfig,
  getSpeakerVerificationConfig,
  updateSpeakerVerificationConfig,
  getSpeakerVerificationStatus,
  listSpeakerVerificationProfiles,
  enrollSpeakerVerificationProfile,
  clearSpeakerVerificationProfile,
  removeSpeakerVerificationProfile,
  deleteSpeakerVerificationProfile,
} from "./api/voice";
export type {
  VoiceStreamConfig,
  WakeWordConfig,
  SpeakerVerificationConfig,
  SpeakerVerificationProfile,
  SpeakerVerificationInfo,
  SpeakerVerificationStatusResponse,
  SpeakerVerificationEnrollmentResponse,
  SpeakerVerificationEnrollResult,
} from "./api/voice";
