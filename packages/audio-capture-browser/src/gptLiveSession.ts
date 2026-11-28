import { DuplexAudio, type DuplexAudioCallbacks, type DuplexAudioChunk } from "./duplexAudio";

const SOCKET_OPEN = 1;
const DEFAULT_MAX_INBOUND_BYTES = 2 * 1024 * 1024;
const DEFAULT_MAX_BUFFERED_AUDIO_BYTES = 256 * 1024;
const DEFAULT_SOCKET_OPEN_TIMEOUT_MS = 15_000;
const DEFAULT_READY_TIMEOUT_MS = 15_000;
const MAX_PLAYBACK_FRAME_BASE64 = 65_536;
const MAX_CLIENT_TEXT_BYTES = 16 * 1024;
const MAX_OPEN_MESSAGE_BYTES = 16 * 1024;
const MAX_CLIENT_MESSAGE_BYTES = 2 * 1024 * 1024;

export interface GPTLiveOpenRequest {
  readonly voice: string;
  readonly language: string;
  readonly model: string;
  readonly provider_id: string;
  readonly remote_allowed: boolean;
  readonly consent: boolean;
  readonly consent_receipt: string;
  readonly budget_micros: number;
}

/** JSON shape emitted by Go's encoding/json for session.SessionEvent. */
export interface GPTLiveSessionEvent {
  readonly EventID?: string;
  readonly SessionID?: string;
  readonly Type?: string;
  readonly EmittedAt?: string;
  readonly TranscriptDelta?: { readonly Text?: string };
  readonly TranscriptFinal?: { readonly Text?: string };
  readonly AssistantDelta?: { readonly Text?: string };
  readonly AssistantFinal?: { readonly Text?: string; readonly HadAudio?: boolean };
  readonly VAD?: { readonly State?: string };
  readonly Tool?: { readonly Name?: string; readonly PayloadJSON?: string };
  readonly AudioOutput?: {
    readonly Bytes?: string;
    readonly Format?: string;
    readonly SampleRate?: number;
    readonly Sequence?: number;
    readonly IsFinal?: boolean;
  };
  readonly ProviderError?: { readonly Message?: string };
  readonly BargeInCancel?: { readonly Reason?: string; readonly CanceledEventID?: string };
  readonly Closed?: { readonly Reason?: string; readonly ResumeFromEventID?: string };
}

export interface GPTLiveMedia {
  start(mode: "websocket"): Promise<string | undefined>;
  ready(): void;
  acceptAudio(chunk: DuplexAudioChunk): void;
  interrupt(): void;
  mute(muted: boolean): void;
  stop(): void;
}

export interface GPTLiveSocket {
  readonly readyState: number;
  readonly bufferedAmount: number;
  onopen: ((event: Event) => void) | null;
  onmessage: ((event: MessageEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
  onclose: ((event: CloseEvent) => void) | null;
  send(data: string): void;
  close(code?: number, reason?: string): void;
}

export type GPTLiveSocketFactory = (url: string) => GPTLiveSocket;

export interface GPTLiveSessionOptions {
  readonly url: string;
  readonly socketFactory?: GPTLiveSocketFactory;
  readonly mediaFactory?: (callbacks: DuplexAudioCallbacks) => GPTLiveMedia;
  readonly maxInboundBytes?: number;
  readonly maxBufferedAudioBytes?: number;
  readonly socketOpenTimeoutMs?: number;
  readonly readyTimeoutMs?: number;
  readonly onEvent?: (event: GPTLiveSessionEvent) => void;
  readonly onLevel?: (level: number) => void;
  readonly onConnected?: () => void;
  readonly onError?: (error: Error) => void;
}

/** Browser client for Audio Tools' authenticated JSON WebSocket contract. */
export class GPTLiveSessionClient {
  private readonly media: GPTLiveMedia;
  private socket: GPTLiveSocket | null = null;
  private state: "idle" | "starting" | "ready" | "closed" = "idle";
  private startupReject: ((error: Error) => void) | null = null;
  private socketOpenReject: ((error: Error) => void) | null = null;
  private socketOpenTimer: ReturnType<typeof setTimeout> | null = null;
  private startupTimer: ReturnType<typeof setTimeout> | null = null;
  private sequence = 0;
  private openSent = false;

  constructor(private readonly options: GPTLiveSessionOptions) {
    const callbacks: DuplexAudioCallbacks = {
      onAudio: (chunk) => this.sendAudio(chunk),
      onLevel: (level) => options.onLevel?.(level),
      onConnected: () => options.onConnected?.(),
      onError: (error) => this.fail(error),
    };
    this.media = options.mediaFactory?.(callbacks) ?? new DuplexAudio(callbacks);
  }

  /** Call directly from the microphone gesture so browser audio can unlock. */
  async start(request: GPTLiveOpenRequest): Promise<void> {
    if (this.state !== "idle") throw new Error("GPT-Live session can only be started once");
    if (!Number.isSafeInteger(request.budget_micros) || request.budget_micros <= 0) {
      throw new Error("GPT-Live session budget must be a positive safe integer");
    }
    this.state = "starting";
    let socket: GPTLiveSocket;
    try {
      socket = (this.options.socketFactory ?? ((url) => new WebSocket(url)))(this.options.url);
    } catch (error) {
      const failure = asError(error, "GPT-Live WebSocket could not be created");
      this.fail(failure);
      throw failure;
    }
    this.socket = socket;
    let resolveSocketOpen!: () => void;
    const socketOpened = new Promise<void>((resolve, reject) => {
      resolveSocketOpen = resolve;
      this.socketOpenReject = reject;
    });
    this.socketOpenTimer = setTimeout(
      () => this.fail(new Error("GPT-Live WebSocket open timed out")),
      this.options.socketOpenTimeoutMs ?? DEFAULT_SOCKET_OPEN_TIMEOUT_MS,
    );
    socket.onopen = () => {
      this.clearSocketOpenWait();
      resolveSocketOpen();
    };
    socket.onmessage = (event) => this.onSocketMessage(event.data);
    socket.onerror = () => this.fail(new Error("GPT-Live WebSocket failed"));
    socket.onclose = (event) => {
      if (this.state !== "closed") this.fail(new Error(event.reason || "GPT-Live WebSocket closed"));
    };

    const ready = new Promise<void>((resolve, reject) => {
      this.startupReject = reject;
      this.readyResolver = resolve;
    });
    // A provider/server error can arrive between sending `open` and reaching
    // the later await below; attach a rejection handler immediately.
    void ready.catch(() => {});
    try {
      // Capture may start while the socket connects, but DuplexAudio gates its
      // onAudio callback until ready() is called after the server ACK.
      const mediaStarted = this.media.start("websocket");
      const mediaFailure = mediaStarted.catch((error: unknown) => {
        this.fail(asError(error, "GPT-Live microphone could not start"));
        throw error;
      });
      await Promise.all([socketOpened, mediaFailure]);
      this.socketOpenReject = null;
      if (!this.isStarting()) throw new Error("GPT-Live session ended before admission");
      this.sendJSON({ type: "open", ...request });
      this.openSent = true;
      const timeout = this.options.readyTimeoutMs ?? DEFAULT_READY_TIMEOUT_MS;
      this.startupTimer = setTimeout(() => this.fail(new Error("GPT-Live admission timed out")), timeout);
      await ready;
    } catch (error) {
      if (!this.isClosed()) this.fail(asError(error, "GPT-Live session could not start"));
      throw error;
    }
  }

  private readyResolver: (() => void) | null = null;

  private onSocketMessage(data: unknown): void {
    if (this.state === "closed") return;
    try {
      if (typeof data !== "string") throw new Error("GPT-Live expected a JSON text frame");
      if (byteLength(data) > (this.options.maxInboundBytes ?? DEFAULT_MAX_INBOUND_BYTES)) {
        throw new Error("GPT-Live inbound frame exceeded its size limit");
      }
      const value: unknown = JSON.parse(data);
      if (!isRecord(value)) throw new Error("GPT-Live server frame must be an object");
      if (value.type === "ready") {
        if (this.state !== "starting" || !this.openSent) throw new Error("Unexpected GPT-Live admission acknowledgment");
        this.state = "ready";
        this.clearStartupWait();
        this.media.ready();
        this.readyResolver?.();
        this.readyResolver = null;
        this.startupReject = null;
        return;
      }
      if (value.type === "error") {
        throw new Error(typeof value.message === "string" ? value.message : "GPT-Live session was rejected");
      }
      if (this.state !== "ready") throw new Error("GPT-Live session event arrived before admission");
      const event = value as GPTLiveSessionEvent;
      const eventType = event.Type ?? (typeof value.type === "string" ? value.type : "");
      if (eventType === "vad" && event.VAD?.State === "speech_start") this.media.interrupt();
      if (eventType === "barge_in_cancel") this.media.interrupt();
      if (eventType === "audio_output") this.playAudio(event.AudioOutput);
      this.options.onEvent?.(event);
      if (eventType === "provider_error") {
        throw new Error(event.ProviderError?.Message || "GPT-Live provider failed");
      }
      if (eventType === "closed") this.finish();
    } catch (error) {
      this.fail(asError(error, "Invalid GPT-Live server frame"));
    }
  }

  private playAudio(audio: GPTLiveSessionEvent["AudioOutput"]): void {
    if (!audio || audio.Format !== "pcm_s16le" || audio.SampleRate !== 24_000 ||
      typeof audio.Bytes !== "string" || audio.Bytes.length === 0 ||
      audio.Bytes.length > MAX_PLAYBACK_FRAME_BASE64) {
      throw new Error("GPT-Live returned an unsupported or oversized audio frame");
    }
    this.media.acceptAudio({ data: audio.Bytes, sampleRate: audio.SampleRate, numChannels: 1, samplesPerChannel: null, itemId: null });
  }

  private sendAudio(chunk: DuplexAudioChunk): void {
    if (this.state !== "ready") return;
    if (chunk.sampleRate !== 24_000 || chunk.numChannels !== 1 || !chunk.samplesPerChannel || chunk.data.length === 0) {
      this.fail(new Error("GPT-Live capture returned an invalid PCM frame"));
      return;
    }
    try {
      this.sendJSON({ type: "audio", audio_base64: chunk.data, format: "pcm_s16le", sample_rate: 24_000, sequence: this.sequence++ }, true);
    } catch (error) {
      this.fail(asError(error, "GPT-Live audio send failed"));
    }
  }

  sendText(text: string): void {
    if (!text.trim()) throw new Error("GPT-Live text input must not be empty");
    if (byteLength(text) > MAX_CLIENT_TEXT_BYTES) throw new Error("GPT-Live text input exceeded its size limit");
    this.sendJSON({ type: "text", text });
  }

  bargeIn(reason: "explicit" | "vad" = "explicit"): void {
    this.media.interrupt();
    this.sendJSON({ type: "barge_in", reason });
  }

  mute(muted: boolean): void { this.media.mute(muted); }

  close(): void {
    if (this.state === "closed") return;
    if (this.state === "ready") {
      try { this.sendJSON({ type: "close" }); } catch { /* socket cleanup below is authoritative */ }
    }
    this.cleanup();
  }

  private sendJSON(message: Record<string, unknown>, audio = false): void {
    const socket = this.socket;
    if (!socket || socket.readyState !== SOCKET_OPEN || this.state === "closed") {
      throw new Error("GPT-Live WebSocket is not open");
    }
    const encoded = JSON.stringify(message);
    const payloadBytes = byteLength(encoded);
    const messageLimit = message.type === "open" ? MAX_OPEN_MESSAGE_BYTES : MAX_CLIENT_MESSAGE_BYTES;
    if (payloadBytes > messageLimit) throw new Error("GPT-Live client message exceeded its size limit");
    if (audio && socket.bufferedAmount + payloadBytes > (this.options.maxBufferedAudioBytes ?? DEFAULT_MAX_BUFFERED_AUDIO_BYTES)) {
      throw new Error("GPT-Live audio backpressure limit exceeded");
    }
    socket.send(encoded);
  }

  private fail(error: Error): void {
    if (this.state === "closed") return;
    this.state = "closed";
    this.clearSocketOpenWait();
    this.clearStartupWait();
    const reject = this.startupReject;
    this.startupReject = null;
    const rejectOpen = this.socketOpenReject;
    this.socketOpenReject = null;
    this.readyResolver = null;
    this.releaseResources();
    reject?.(error);
    rejectOpen?.(error);
    this.options.onError?.(error);
  }

  private finish(): void { this.cleanup(); }

  private cleanup(): void {
    if (this.state === "closed") return;
    this.state = "closed";
    this.clearSocketOpenWait();
    this.clearStartupWait();
    this.startupReject?.(new Error("GPT-Live session closed before admission"));
    this.startupReject = null;
    this.socketOpenReject?.(new Error("GPT-Live session closed before the WebSocket opened"));
    this.socketOpenReject = null;
    this.readyResolver = null;
    this.releaseResources();
  }

  private clearStartupWait(): void {
    if (this.startupTimer !== null) clearTimeout(this.startupTimer);
    this.startupTimer = null;
  }

  private clearSocketOpenWait(): void {
    if (this.socketOpenTimer !== null) clearTimeout(this.socketOpenTimer);
    this.socketOpenTimer = null;
  }

  private isStarting(): boolean { return this.state === "starting"; }
  private isClosed(): boolean { return this.state === "closed"; }

  private releaseResources(): void {
    const socket = this.socket;
    this.socket = null;
    if (socket) {
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      try { socket.close(1000, "session ended"); } catch { /* cleanup remains idempotent */ }
    }
    this.media.stop();
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function byteLength(value: string): number {
  return new TextEncoder().encode(value).byteLength;
}

function asError(value: unknown, fallback: string): Error {
  return value instanceof Error ? value : new Error(fallback);
}
