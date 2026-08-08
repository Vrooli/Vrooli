import type { TranscriptionProvider } from "./types";
import { PcmVoiceStreamProvider } from "../pcmVoiceStreamProvider";
import type { AudioFeatures, PassiveListenerOpts, WakeWordEngine } from "./wakeword";

export interface VoiceCoreWakeWordConfig {
  configured: boolean;
  template: {
    samples: Array<{ audio: Uint8Array }>;
    label: string;
    updatedAt: string;
  } | null;
}

export interface VoiceCoreServices {
  PcmVoiceStreamProvider: new () => PcmVoiceStreamProvider;
  getVoiceStreamConfig: () => Promise<unknown>;
  getWakeWordConfig: () => Promise<VoiceCoreWakeWordConfig>;
  transcribeAudio: (audio: Blob, language?: string) => Promise<string>;
  transcribeAudioBypassFilter: (audio: Blob, language?: string) => Promise<string>;
  createAudioFilterChain: (
    context: AudioContext,
    source: MediaStreamAudioSourceNode,
  ) => { analyser: AnalyserNode; filteredStream: MediaStream; nodes: AudioNode[] };
  playRecordingStartCue: () => void;
  playRecordingStopCue: () => void;
  WhisperProvider: new () => TranscriptionProvider;
  WebSpeechProvider: new () => TranscriptionProvider & { lang: string };
  bytesToFeatures: (audio: Uint8Array, engine: WakeWordEngine) => Promise<AudioFeatures>;
  createWakeWordEngine: () => WakeWordEngine;
  PassiveListener: new (options: PassiveListenerOpts) => {
    start: () => Promise<void>;
    dispose: (reason?: Parameters<NonNullable<PassiveListenerOpts["onMicReleased"]>>[0]) => void;
  };
}
