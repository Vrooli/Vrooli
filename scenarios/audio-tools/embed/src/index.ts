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
// Status: skeleton surface during Phase F rollout. Concrete components will
// be ported from web-console hooks/components in follow-up sessions, then
// generalized (terminal-input gates removed, callback props introduced).

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
