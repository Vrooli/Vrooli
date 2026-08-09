import {
  bytesToFeatures,
  createWakeWordEngine,
  PassiveListener,
  type VoiceCoreServices,
} from "@vrooli/audio-capture-browser";
import { getVoiceStreamConfig, getWakeWordConfig, transcribeAudio, transcribeAudioBypassFilter } from "./api/voice";
import { createAudioFilterChain } from "./hooks/voice/audioUtils";
import { playRecordingStartCue, playRecordingStopCue } from "./hooks/voice/audioCues";
import { WhisperProvider } from "./hooks/voice/WhisperProvider";
import { WebSpeechProvider } from "./hooks/voice/WebSpeechProvider";
import { PcmVoiceStreamProvider } from "./hooks/voice/PcmVoiceStreamProvider";

// HOST DIFFERENCE: narrow swarm adapter for its API/proto clients and browser
// fallback providers; lifecycle/orchestration lives in the shared package.
// The package owns orchestration and lifecycle policy. This object is the
// deliberately narrow scenario boundary for API calls and browser-provider
// implementations that cannot be shared across hosts.
export const voiceCoreServices: VoiceCoreServices = {
  PcmVoiceStreamProvider,
  getVoiceStreamConfig,
  getWakeWordConfig,
  transcribeAudio,
  transcribeAudioBypassFilter,
  createAudioFilterChain,
  playRecordingStartCue,
  playRecordingStopCue,
  WhisperProvider,
  WebSpeechProvider,
  bytesToFeatures,
  createWakeWordEngine,
  PassiveListener,
};
