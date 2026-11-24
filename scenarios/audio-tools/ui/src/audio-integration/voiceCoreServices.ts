// HOST DIFFERENCE: supplies audio-tools API and provider adapters to the
// shared browser voice orchestration.
import {
  bytesToFeatures,
  createWakeWordEngine,
  PassiveListener,
  type VoiceCoreServices,
} from "@vrooli/audio-capture-browser";
import { getVoiceStreamConfig, getWakeWordConfig, transcribeAudio, transcribeAudioBypassFilter } from "./api/voice";
import { createAudioFilterChain } from "./hooks/voice/audioUtils";
import { playRecordingStartCue, playRecordingStopCue, playRecordingFaultCue } from "./hooks/voice/audioCues";
import { WhisperProvider } from "./hooks/voice/WhisperProvider";
import { PcmVoiceStreamProvider } from "./hooks/voice/PcmVoiceStreamProvider";
import { useVoiceConfigStore } from "./hooks/useVoiceConfigStore";

// Kyutai needs a complete 50 ms opening frame to preserve its first words.
// Use the shared package's smaller speculative first frame only when the
// hydrated server selection is Sherpa; keep Kyutai and unknown selections at
// the 50 ms opening cadence.
class AudioToolsPcmVoiceStreamProvider extends PcmVoiceStreamProvider {
  constructor() {
    const engineId = useVoiceConfigStore.getState().engineId;
    super({ wireBatchSamples: 800, fastFirstWireBatch: engineId === "sherpa-streaming" });
  }
}

// The package owns orchestration and lifecycle policy. This object is the
// deliberately narrow scenario boundary for API calls and browser-provider
// implementations that cannot be shared across hosts.
export const voiceCoreServices: VoiceCoreServices = {
  PcmVoiceStreamProvider: AudioToolsPcmVoiceStreamProvider,
  getVoiceStreamConfig,
  getWakeWordConfig: async () => {
    const config = await getWakeWordConfig();
    return {
      configured: config.configured,
      template: config.template
        ? {
            label: config.template.label,
            updatedAt: config.template.updatedAt,
            samples: config.template.samples as unknown as Array<{ audio: Uint8Array }>,
          }
        : null,
    };
  },
  transcribeAudio,
  transcribeAudioBypassFilter,
  createAudioFilterChain,
  playRecordingStartCue,
  playRecordingStopCue,
  playRecordingFaultCue,
  WhisperProvider,
  bytesToFeatures,
  createWakeWordEngine,
  PassiveListener,
};
