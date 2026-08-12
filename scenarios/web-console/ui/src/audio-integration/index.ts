// audio-integration — web-console's audio surface.
//
// Calls are made via the same-origin Connect transport to web-console's
// own AudioAdminService and AudioRuntimeService. The browser never sees
// audio-tools' host; web-console's API owns the inter-scenario hop.
// HOST DIFFERENCE: web-console exposes its own health probes and omits swarm's
// config/client context; capture orchestration remains package-owned.

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
export { ttsPlaybackRegistry } from "@vrooli/audio-capture-browser";

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
export { useScenarioVoiceInput, probeWhisperHealth } from "./hooks/useVoiceCore";

export { useTextToSpeechCore } from "@vrooli/audio-capture-browser";
export type {
  UseTextToSpeechCoreOptions,
  TTSCoreSpeakSettings,
  TTSCoreState,
  TTSCorePlaybackEvent,
} from "@vrooli/audio-capture-browser";

// Canonical capability + feature slugs for audio-tools (drift-safe via proto enum).
export { AUDIO_TOOLS_CAPABILITY_SLUG, featureSlug, allFeatureSlugs, AudioToolsFeature } from "./features";
