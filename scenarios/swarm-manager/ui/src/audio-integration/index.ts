// audio-integration — swarm-manager's audio surface.
//
// Calls are made via the same-origin Connect transport to swarm-manager's
// own AudioAdminService and AudioRuntimeService. The browser never sees
// audio-tools' host; swarm-manager's API owns the inter-scenario hop.

import { registerVoiceTransport as registerBrowserVoiceTransport } from "@vrooli/audio-capture-browser";
import { buildVoiceStreamWsUrl, transcribeAudioWithRetry } from "./api/voice";

export function registerVoiceTransport(): void {
  registerBrowserVoiceTransport({
    buildStreamUrl: (language, sessionId, resumeToken) => buildVoiceStreamWsUrl(language, sessionId, resumeToken),
    transcribeRetained: (blob, language) => transcribeAudioWithRetry(blob, 2, language),
  });
}

export { MicReadinessIndicator } from "./MicReadinessIndicator";
export type { MicReadinessIndicatorProps } from "./MicReadinessIndicator";
export { useVoiceConfigStore } from "./hooks/useVoiceConfigStore";
export type { VoiceConfigState } from "./hooks/useVoiceConfigStore";
export { useHydrateVoiceConfig } from "./hooks/useHydrateVoiceConfig";



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
export { WebSpeechProvider } from "./hooks/voice/WebSpeechProvider";

// TTS capability surface.
// =============================================================================

export type { TTSBackend, TTSPlaybackCapabilities, TTSPlaybackState, TTSProvider, TTSSpeakOptions, TTSVoiceInfo } from "@vrooli/audio-capture-browser";
export { KokoroProvider, BrowserTTSProvider } from "@vrooli/audio-capture-browser";

// =============================================================================
// API surfaces (audio operations against swarm-manager's own API).
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
export type { TTSConfig, TTSSummarizeConfig, TTSSummarizeModel, TTSPlaybackEvent } from "./api/tts";

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
