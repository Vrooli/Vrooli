// audio-integration — web-console's audio surface.
//
// Calls are made via the same-origin Connect transport to web-console's
// own AudioAdminService and AudioRuntimeService. The browser never sees
// audio-tools' host; web-console's API owns the inter-scenario hop.

export { MicReadinessIndicator } from "./MicReadinessIndicator";
export type { MicReadinessIndicatorProps } from "./MicReadinessIndicator";

export {
  useServerVadStateStore,
  setServerVadState,
  resetServerVadState,
  _resetServerVadStateForTesting,
  SERVER_VAD_STALE_MS,
} from "./hooks/useServerVadStateStore";
export type { ServerVadStateSnapshot } from "./hooks/useServerVadStateStore";

// =============================================================================
// Voice (STT) capability surface.
// =============================================================================

export * from "./hooks/voice/types";
export * from "./hooks/voice/audioUtils";
export * from "./hooks/voice/audioCues";
export * from "./hooks/voice/activity";
export * from "./hooks/voice/vad";
export * from "./hooks/voice/autoStopDecision";
export * from "./hooks/voice/sharedAudioContext";
export * from "./hooks/voice/micOwnership";
export * from "./hooks/voice/micLifecyclePolicy";
export * from "./hooks/voice/voiceCaptureController";
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
// API surfaces (audio operations against web-console's own API).
// =============================================================================

export {
  synthesizeTTS,
  fetchCachedTTS,
  getTTSVoices,
  getTTSConfig,
  updateTTSConfig,
  getTTSSummarizeConfig,
  listTTSSummarizeModels,
  updateTTSSummarizeConfig,
  reportTTSEvent,
} from "./api/tts";
export type { TTSConfig, TTSSummarizeConfig, TTSSummarizeModel, TTSVoiceInfo, TTSPlaybackEvent } from "./api/tts";

export {
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
  WakeWordSampleData,
  WakeWordTemplateData,
  WakeWordSampleInput,
  WakeWordTemplateInput,
  SpeakerVerificationConfig,
  SpeakerVerificationProfile,
  SpeakerVerificationInfo,
  SpeakerVerificationStatusResponse,
  SpeakerVerificationEnrollmentResponse,
  SpeakerVerificationEnrollResult,
} from "./api/voice";

// =============================================================================
// Generic, scenario-agnostic core React hooks.
// =============================================================================

export { useVoiceCore } from "./hooks/useVoiceCore";
export type { UseVoiceCoreOptions, VoiceCapabilityProbe } from "./hooks/useVoiceCore";

export { useTextToSpeechCore } from "./hooks/useTextToSpeechCore";
export type {
  UseTextToSpeechCoreOptions,
  TTSCoreSpeakSettings,
  TTSCoreState,
  TTSCorePlaybackEvent,
} from "./hooks/useTextToSpeechCore";

// Canonical capability + feature slugs for audio-tools (drift-safe via proto enum).
export { AUDIO_TOOLS_CAPABILITY_SLUG, featureSlug, allFeatureSlugs, AudioToolsFeature } from "./features";
