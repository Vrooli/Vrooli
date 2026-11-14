import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  STTService,
  type TranscribeResponse,
} from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import { STTAdminService } from "@vrooli/proto-types/audio-tools/v1/stt/stt_admin_pb";
import { AudioFormat, ResponseFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { TTSService } from "@vrooli/proto-types/audio-tools/v1/tts/tts_pb";
import {
  PcmVoiceStreamProvider,
  PassiveListener,
  bytesToFeatures,
  createAudioFilterChain,
  createWakeWordEngine,
  playRecordingFaultCue,
  playRecordingStartCue,
  playRecordingStopCue,
  registerVoiceTransport,
  type AudioFeatures,
  type TranscriptionProvider,
  type VoiceCoreServices,
  type VoiceCapabilityProbe,
  type WakeWordEngine,
} from "@vrooli/audio-capture-browser";

import { API_BASE } from "../../api/client";

const configuredAudioToolsURL = (
  (import.meta.env.VITE_AUDIO_TOOLS_API_URL as string | undefined)
  ?? (import.meta.env.AUDIO_TOOLS_API_URL as string | undefined)
  ?? ""
).trim();

function audioFormat(blob: Blob): AudioFormat {
  const type = blob.type.toLowerCase();
  if (type.includes("wav")) return AudioFormat.WAV;
  if (type.includes("ogg")) return AudioFormat.OGG;
  if (type.includes("mp3") || type.includes("mpeg")) return AudioFormat.MP3;
  if (type.includes("flac")) return AudioFormat.FLAC;
  return AudioFormat.WEBM;
}

async function transcribe(client: ReturnType<typeof createAudioToolsClients>["stt"], blob: Blob): Promise<TranscribeResponse> {
  return client.transcribe({
    audio: new Uint8Array(await blob.arrayBuffer()),
    format: audioFormat(blob),
    language: "",
    skipSpeakerVerification: false,
    initialPrompt: "",
  });
}

function createAudioToolsClients(baseURL: string) {
  const transport = createConnectTransport({ baseUrl: baseURL });
  return {
    stt: createClient(STTService, transport),
    sttAdmin: createClient(STTAdminService, transport),
  };
}

export interface PortalSpeechRuntime {
  enabled: boolean;
  synthesize(text: string, signal?: AbortSignal): Promise<Blob>;
  capabilityCheck: () => Promise<boolean>;
}

/**
 * Optional server-side speech output. The runtime keeps the endpoint
 * credential-free and lets the caller abort synthesis before an audio effect
 * is started. When Audio Tools is absent, callers can choose browser speech
 * synthesis without changing the text composer.
 */
export function createPortalSpeechRuntime(baseURL = configuredAudioToolsURL): PortalSpeechRuntime {
  const enabled = baseURL.length > 0;
  const client = createClient(TTSService, createConnectTransport({ baseUrl: baseURL || API_BASE }));
  return {
    enabled,
    synthesize: async (text, signal) => {
      if (!enabled) throw new Error("Audio Tools is not configured.");
      const response = await client.synthesize({ text, voice: "", voiceOverrides: [], speed: 0, responseFormat: ResponseFormat.MP3, eventId: "", version: "", chunkIndex: 0 }, { signal });
      return new Blob([response.audio as Uint8Array<ArrayBuffer>], { type: response.contentType || "audio/mpeg" });
    },
    capabilityCheck: async () => {
      if (!enabled) return false;
      try {
        const response = await fetch(`${baseURL.replace(/\/$/, "")}/health`, { cache: "no-store" });
        return response.ok;
      } catch {
        return false;
      }
    },
  };
}

export const portalSpeech = createPortalSpeechRuntime();

function wsBase(baseURL: string): string {
  if (baseURL.startsWith("https://")) return `wss://${baseURL.slice("https://".length)}`;
  if (baseURL.startsWith("http://")) return `ws://${baseURL.slice("http://".length)}`;
  return baseURL;
}

function streamURL(baseURL: string, language: string, sessionID: string, resumeToken: string): string {
  const params = new URLSearchParams({ format: "pcm_s16le", protocol_version: "2" });
  if (language) params.set("language", language);
  if (sessionID) params.set("session_id", sessionID);
  if (resumeToken) params.set("resume_token", resumeToken);
  return `${wsBase(baseURL).replace(/\/$/, "")}/api/v1/voice/stream?${params.toString()}`;
}

class AudioToolsBatchProvider implements TranscriptionProvider {
  private recorder: MediaRecorder | null = null;
  private stream: MediaStream | null = null;
  private chunks: Blob[] = [];
  private lastTurn: { blob: Blob; mimeType: string; durationMs: number; capturedAt: number } | null = null;
  private startedAt = 0;
  language = "en";
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;
  onStatus: ((status: { code: string; message: string }) => void) | null = null;

  constructor(private readonly client: ReturnType<typeof createAudioToolsClients>["stt"]) {}

  getStream() { return this.stream; }
  getLastTurnAudio() { return this.lastTurn; }
  disposeLastTurn() { this.lastTurn = null; }
  dropTail() {}

  async start(preWarmedStream?: MediaStream) {
    this.lastTurn = null;
    this.stream = preWarmedStream?.getTracks().every((track) => track.readyState === "live")
      ? preWarmedStream
      : await navigator.mediaDevices.getUserMedia({ audio: true });
    this.chunks = [];
    this.startedAt = Date.now();
    const mimeType = MediaRecorder.isTypeSupported("audio/webm;codecs=opus") ? "audio/webm;codecs=opus" : "audio/webm";
    this.recorder = new MediaRecorder(this.stream, { mimeType });
    this.recorder.ondataavailable = (event) => { if (event.data.size > 0) this.chunks.push(event.data); };
    this.recorder.onstop = () => {
      const blob = new Blob(this.chunks, { type: mimeType });
      const durationMs = Date.now() - this.startedAt;
      this.stream?.getTracks().forEach((track) => track.stop());
      this.stream = null;
      this.chunks = [];
      if (blob.size === 0) {
        this.onResult?.("");
        return;
      }
      this.lastTurn = { blob, mimeType, durationMs, capturedAt: Date.now() };
      void transcribe(this.client, blob)
        .then((response) => this.onResult?.(response.text.trim()))
        .catch(() => this.onError?.("Voice transcription is unavailable; your typed message is still available."));
    };
    this.recorder.start();
  }

  stop() {
    if (this.recorder?.state === "recording") this.recorder.stop();
    else this.stream?.getTracks().forEach((track) => track.stop());
  }

  dispose() {
    if (this.recorder?.state === "recording") this.recorder.stop();
    this.stream?.getTracks().forEach((track) => track.stop());
    this.stream = null;
    this.lastTurn = null;
  }
}

export interface PortalVoiceServices {
  services: VoiceCoreServices;
  enabled: boolean;
  capabilityCheck: () => Promise<VoiceCapabilityProbe>;
}

export function createPortalVoiceServices(baseURL = configuredAudioToolsURL): PortalVoiceServices {
  const enabled = baseURL.length > 0;
  const clients = createAudioToolsClients(baseURL || API_BASE);
  registerVoiceTransport({
    buildStreamUrl: (language, sessionID, resumeToken) => streamURL(baseURL || API_BASE, language, sessionID, resumeToken),
    transcribeRetained: async (blob, language) => {
      const response = await clients.stt.transcribe({
        audio: new Uint8Array(await blob.arrayBuffer()),
        format: audioFormat(blob),
        language,
        skipSpeakerVerification: false,
        initialPrompt: "",
      });
      return response.text;
    },
  });
  const services: VoiceCoreServices = {
    PcmVoiceStreamProvider,
    getVoiceStreamConfig: () => clients.sttAdmin.getStreamConfig({}),
    getWakeWordConfig: () => Promise.resolve({ configured: false, template: null }),
    transcribeAudio: async (blob) => (await transcribe(clients.stt, blob)).text,
    transcribeAudioBypassFilter: async (blob) => (await transcribe(clients.stt, blob)).text,
    createAudioFilterChain,
    playRecordingStartCue,
    playRecordingStopCue,
    playRecordingFaultCue,
    WhisperProvider: class extends AudioToolsBatchProvider {
      constructor() { super(clients.stt); }
    },
    bytesToFeatures: (bytes: Uint8Array, engine: WakeWordEngine): Promise<AudioFeatures> => bytesToFeatures(bytes, engine),
    createWakeWordEngine,
    PassiveListener,
  };
  const capabilityCheck = async (): Promise<VoiceCapabilityProbe> => {
    if (!enabled) return { whisperHealthy: false, streamingAvailable: false, capabilityReason: "Audio Tools is not configured." };
    try {
      const response = await fetch(`${baseURL.replace(/\/$/, "")}/health`, { cache: "no-store" });
      if (!response.ok) return { whisperHealthy: false, streamingAvailable: false, capabilityReason: `Audio Tools health returned HTTP ${response.status}.` };
      return { whisperHealthy: true, streamingAvailable: true };
    } catch {
      return { whisperHealthy: false, streamingAvailable: false, capabilityReason: "Audio Tools is unavailable." };
    }
  };
  return { services, enabled, capabilityCheck };
}

export const portalVoice = createPortalVoiceServices();
