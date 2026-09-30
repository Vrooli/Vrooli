// audio-integration — audio-tools' scenario-local transport adapter surface.
// HOST DIFFERENCE: this adapter binds the shared browser package to the
// audio-tools-owned API and is not a shared implementation to copy.

import { registerVoiceTransport as registerBrowserVoiceTransport } from "@vrooli/audio-capture-browser";
import { buildVoiceStreamWsUrl, transcribeAudioWithRetry, transcribeAudioWithRetryDetailed } from "./api/voice";

export function registerVoiceTransport(): void {
  registerBrowserVoiceTransport({
    buildStreamUrl: (language, sessionId, resumeToken) => buildVoiceStreamWsUrl(language, sessionId, resumeToken),
    transcribeRetained: (blob, language) => transcribeAudioWithRetry(blob, 2, language),
    transcribeRetainedWithIdentity: (blob, language) => transcribeAudioWithRetryDetailed(blob, 2, language),
  });
}

export { MicReadinessIndicator } from "./MicReadinessIndicator";
export type { MicReadinessIndicatorProps } from "./MicReadinessIndicator";
export { useVoiceConfigStore } from "./hooks/useVoiceConfigStore";
export type { VoiceConfigState } from "./hooks/useVoiceConfigStore";
export { useHydrateVoiceConfig } from "./hooks/useHydrateVoiceConfig";
export { useStreamDegradation, BUFFERED_MODE_NOTICE } from "./hooks/useStreamDegradation";
export { StreamingDegradationNotice } from "../components/streaming/StreamingDegradationNotice";


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
//
// The protocol, PCM provider, voice core, VAD, lifecycle, and wake-word
// substrate are exported from the shared browser package. Only the
// microphone readiness seam remains host-local where it exists.
export * from "@vrooli/audio-capture-browser";
export { WhisperProvider } from "./hooks/voice/WhisperProvider";

// TTS capability surface.
// =============================================================================

export type { TTSBackend, TTSPlaybackCapabilities, TTSPlaybackState, TTSProvider, TTSSpeakOptions, TTSVoiceInfo } from "@vrooli/audio-capture-browser";
export { KokoroProvider, BrowserTTSProvider } from "@vrooli/audio-capture-browser";

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
export type { TTSConfig, TTSSummarizeConfig, TTSPlaybackEvent } from "./api/tts";

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

// =============================================================================
// Generic, scenario-agnostic core React hooks.
// =============================================================================

export { useScenarioVoiceCore as useVoiceCore } from "./hooks/useVoiceCore";
export type { UseVoiceCoreOptions, VoiceCapabilityProbe } from "./hooks/useVoiceCore";

export { useTextToSpeechCore } from "@vrooli/audio-capture-browser";
export type {
  UseTextToSpeechCoreOptions,
  TTSCoreSpeakSettings,
  TTSCoreState,
  TTSCorePlaybackEvent,
} from "@vrooli/audio-capture-browser";

// Canonical capability + feature slugs for audio-tools (drift-safe via proto enum).
export { AUDIO_TOOLS_CAPABILITY_SLUG, featureSlug, allFeatureSlugs, AudioToolsFeature } from "./features";
