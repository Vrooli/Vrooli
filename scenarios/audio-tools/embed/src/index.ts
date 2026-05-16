// @audio-tools/embed — adoptable React component surface.
//
// This is the public re-export boundary for consumer scenarios (web-console,
// future swarm-manager/agent-manager/phone-agent). Consumers import:
//
//   import { VoiceInputButton, AudioPlayerBar } from "@audio-tools/embed";
//
// Components here are web-console-agnostic: they accept generic callback
// props (`onTranscript`, `commandHandler`, `audioUrl | audioBytes`) and never
// reference terminal panes, conversation cursors, or session IDs.
//
// The package ships the full adoptable component set plus the audio-tools
// API client; consumers compose them against their own data flow. Each
// component takes generic callback props (`onTranscript`, `commandHandler`,
// `audioUrl | audioBytes`) and never references terminal panes or session ids.

export { VoiceInputButton } from "./VoiceInputButton";
export { AudioPlayerBar } from "./AudioPlayerBar";
export { EnableAudioBanner } from "./EnableAudioBanner";
export { MicReadinessIndicator } from "./MicReadinessIndicator";
export { VoiceRejectionBanner } from "./VoiceRejectionBanner";
export { VoiceCommandSuggestion } from "./VoiceCommandSuggestion";
export { VoiceSettingsPanel } from "./VoiceSettingsPanel";
export { TtsSettingsPanel } from "./TtsSettingsPanel";

export type { VoiceInputButtonProps } from "./VoiceInputButton";
export type { AudioPlayerBarProps } from "./AudioPlayerBar";
export type { EnableAudioBannerProps } from "./EnableAudioBanner";
export type { MicReadinessIndicatorProps } from "./MicReadinessIndicator";
export type { VoiceRejectionBannerProps } from "./VoiceRejectionBanner";
export type { VoiceCommandSuggestionProps } from "./VoiceCommandSuggestion";
export type { VoiceSettingsPanelProps } from "./VoiceSettingsPanel";
export type { TtsSettingsPanelProps } from "./TtsSettingsPanel";

// Connect client + React context for consumers that want to call audio-tools
// RPCs (Transcribe, Synthesize, Summarize) from custom UI alongside the
// adoptable components.
export {
  createAudioToolsClient,
  AudioToolsProvider,
  useAudioToolsClient,
} from "./client";
export type {
  AudioToolsClient,
  CreateAudioToolsClientOptions,
  AudioToolsProviderProps,
} from "./client";

// =============================================================================
// Voice (STT) capability surface — full re-export from the ported modules.
// `export *` keeps the surface comprehensive so consumer scenarios can pull
// any provider/utility without poking into deep paths.
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
// TTS capability surface
// =============================================================================

export * from "./hooks/tts/types";
export { KokoroProvider } from "./hooks/tts/KokoroProvider";
export { BrowserTTSProvider } from "./hooks/tts/BrowserTTSProvider";

// =============================================================================
// API surfaces (audio-only operations bound to audio-tools)
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
