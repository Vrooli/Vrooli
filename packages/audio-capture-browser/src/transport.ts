import type { StreamMessage } from "./streamMessages";

export interface VoiceProviderIdentity {
  readonly providerId?: string;
  readonly modelId?: string;
  readonly providerTier?: "local" | "byok" | "vrooli";
}

/** Exact route selected by the stream. Retained recovery must not pick a new tier. */
export interface RetainedTranscriptionRoute extends VoiceProviderIdentity {
  readonly providerTier: "local";
  readonly modelId: string;
}

export interface RetainedTranscription {
  readonly text: string;
  readonly providerIdentity?: VoiceProviderIdentity;
}

export interface VoiceStreamStart {
  readonly language: string;
  readonly sessionId: string;
  readonly resumeToken: string;
  readonly protocolVersion: 2;
  readonly generation: bigint;
  readonly inputFormat: "pcm_s16le";
  readonly sampleRateHz: number;
  readonly skipSpeakerVerification?: boolean;
  readonly initialPrompt?: string;
}

export interface VoiceAudioChunk {
  readonly sequence: bigint;
  readonly startSample: bigint;
  readonly endSample: bigint;
  readonly audio: Uint8Array;
  readonly sha256: Uint8Array;
}

export interface VoiceStreamConnection {
  /** True only after the server accepts this session and the stream can accept requests. */
  isOpen(): boolean;
  sendAudio(chunk: VoiceAudioChunk): void;
  /** Sends the protocol's terminal StreamEnd request and closes request input. */
  end(): void;
  /** Cancels this stream. The server session remains resumable by identity. */
  close(): void;
}

export interface VoiceStreamHandlers {
  onReady(): void;
  onEvent(event: StreamMessage): void;
  onClose(error?: unknown): void;
}

export interface VoiceTransport {
  openStream(start: VoiceStreamStart, handlers: VoiceStreamHandlers): VoiceStreamConnection;
  /** Only transports that honor this exact-model contract may attempt recovery. */
  readonly retainedRoutePolicy?: "exact-model";
  transcribeRetained(blob: Blob, language: string, route: RetainedTranscriptionRoute): Promise<string>;
  /** Optional metadata-preserving form used by diagnostics and recovery UI. */
  transcribeRetainedWithIdentity?(blob: Blob, language: string, route: RetainedTranscriptionRoute): Promise<RetainedTranscription>;
  getStreamConfig?(): Promise<unknown>;
}

export interface VoiceTransportStatus {
  readonly code: string;
  readonly message: string;
}

let registeredVoiceTransport: VoiceTransport | null = null;

/** Register the consuming scenario's transport adapter before the shared hook runs. */
export function registerVoiceTransport(transport: VoiceTransport): void {
  registeredVoiceTransport = transport;
}

export function getDefaultVoiceTransport(): VoiceTransport | null {
  return registeredVoiceTransport;
}

export function requireVoiceTransport(): VoiceTransport {
  if (!registeredVoiceTransport) {
    throw new Error("Voice transport is not registered. Call the scenario's registerVoiceTransport() during application bootstrap.");
  }
  return registeredVoiceTransport;
}
