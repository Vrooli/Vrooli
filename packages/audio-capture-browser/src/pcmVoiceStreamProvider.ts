import { concatInt16, encodeWavFromPcm16, frameToCanonicalPcm16, TARGET_SAMPLE_RATE } from "./pcm";
import { createCanonicalPcmCapture, type PcmCapture } from "./pcmCapture";
import { digestAudio, encodeAudioFrame, newSessionIdentity } from "./protocol";
import { IndexedDBTurnJournalStore, MemoryTurnJournalStore, TurnJournal } from "./turnJournal";
import { forgetUnfinishedSession, loadUnfinishedSession, rememberUnfinishedSession } from "./sessionIdentity";
import { dispatchStreamMessage } from "./streamMessages";
import { requireVoiceTransport, type VoiceTransport, type VoiceTransportStatus } from "./transport";
import { StreamDiagnosticRecorder, type StreamTurnDiagnostic } from "./streamDiagnostic";
import { registerMicStream, releaseMicLease, type MicLease, type MicReleaseReason } from "./voice/micOwnership";
import type { LastTurnAudio } from "./voice/types";

export type SharedPcmCaptureFactory = (
  stream: MediaStream,
  onFrame: (samples: Float32Array, sampleRate: number) => void,
) => PcmCapture | Promise<PcmCapture>;

export interface SharedPcmVoiceStreamProviderOptions {
  readonly transport?: VoiceTransport;
  readonly language?: string;
  readonly getUserMedia?: () => Promise<MediaStream>;
  readonly getAudioContext?: () => AudioContext;
  readonly onStatus?: (status: VoiceTransportStatus) => void;
  readonly onResult?: (text: string) => void;
  readonly onError?: (message: string) => void;
  readonly onPartial?: (text: string) => void;
  readonly onSegmentFinal?: (text: string, index: number) => void;
  readonly captureFactory?: SharedPcmCaptureFactory;
  readonly journalFactory?: () => TurnJournal;
  readonly retainStream?: boolean;
}

export const MAX_CONSECUTIVE_RECONNECTS = 5;
export const RECONNECT_BACKOFF_MS = [1_000, 2_000, 4_000, 8_000, 8_000] as const;

/** Shared durable PCM transport used by scenario adapters. */
export class PcmVoiceStreamProvider {
  private readonly options: SharedPcmVoiceStreamProviderOptions;
  private transport: VoiceTransport;
  language = "en";
  private ws: WebSocket | null = null;
  private capture: PcmCapture | null = null;
  private stream: MediaStream | null = null;
  private lease: MicLease | null = null;
  private journal: TurnJournal | null = null;
  private writes: Promise<void> = Promise.resolve();
  private pending: ArrayBuffer[] = [];
  private allPcm: Int16Array[] = [];
  private sessionId = "";
  private resumeToken = "";
  private wsUrl = "";
  private sequence = 0n;
  private sample = 0n;
  private reconnects = 0;
  private stopped = false;
  private finalReceived = false;
  private reacquiring = false;
  private readonly deliveredSegments = new Set<string>();
  private allStartedAt = 0;
  private tailDropArmed = false;
  private lastTurn: LastTurnAudio | null = null;
  private diagnostic = new StreamDiagnosticRecorder();
  private preconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private serverAckTimer: ReturnType<typeof setTimeout> | null = null;
  private finalPendingTimer: ReturnType<typeof setTimeout> | null = null;
  private fallbackTimer: ReturnType<typeof setTimeout> | null = null;
  private serverAcknowledged = false;
  private terminalFailure = false;
  private doneSent = false;
  private micRecoveryAttempted = false;
  private trackHandlers: Array<{
    track: MediaStreamTrack;
    onended: () => void;
    onmute: () => void;
    onunmute: () => void;
  }> = [];
  captureFactory: SharedPcmCaptureFactory;
  retainStream = false;

  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onStatus: ((status: VoiceTransportStatus) => void) | null = null;
  onDiagnostic: ((diagnostic: StreamTurnDiagnostic) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;
  onSegmentFinal: ((text: string, index: number) => void) | null = null;
  onSegmentAccepted: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSegmentRejected: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSpeakerStatus: ((enabled: boolean, profileConfigured: boolean) => void) | null = null;
  onVadState: ((snapshot: { voiced: boolean; silenceElapsedMs: number; silenceTimeoutMs: number; tickSeq: number; silenceTimedOut: boolean }) => void) | null = null;

  constructor(options: SharedPcmVoiceStreamProviderOptions = {}) {
    this.options = options;
    this.transport = options.transport ?? requireVoiceTransport();
    this.language = options.language ?? "en";
    this.captureFactory = options.captureFactory ?? ((stream, onFrame) => {
      if (typeof AudioContext === "undefined") return { stop: () => {} };
      return createCanonicalPcmCapture(this.options.getAudioContext?.() ?? new AudioContext(), stream, onFrame);
    });
    this.retainStream = options.retainStream ?? false;
    this.onStatus = options.onStatus ?? null;
    this.onResult = options.onResult ?? null;
    this.onError = options.onError ?? null;
    this.onPartial = options.onPartial ?? null;
    this.onSegmentFinal = options.onSegmentFinal ?? null;
  }

  getStream(): MediaStream | null { return this.stream; }

  getDiagnostic(): StreamTurnDiagnostic { return this.diagnostic.read(); }
  exportDiagnostic(): string { return this.diagnostic.exportJSON(); }
  getLastTurnAudio(): LastTurnAudio | null { return this.lastTurn; }
  disposeLastTurn(): void { this.lastTurn = null; }
  private publishDiagnostic(): void { this.onDiagnostic?.(this.diagnostic.read()); }
  private status(code: string, message: string): void { this.diagnostic.status(code); this.publishDiagnostic(); this.onStatus?.({ code, message }); }
  private cleanServerError(message: string): string {
    return /dial tcp|connect(?:ion)? refused|wss?:\/\//i.test(message)
      ? "The speech backend is unavailable; retry the turn."
      : message;
  }

  private async acquireStream(): Promise<MicLease> {
    const stream = this.options.getUserMedia
      ? await this.options.getUserMedia()
      : await navigator.mediaDevices.getUserMedia({ audio: true });
    return registerMicStream("voice-stream", stream);
  }

  private releaseOwnStream(reason: MicReleaseReason): void {
    this.unbindTracks();
    releaseMicLease(this.lease, reason);
    this.lease = null;
    this.stream = null;
  }

  private bindTracks(stream: MediaStream): void {
    this.unbindTracks();
    for (const track of stream.getTracks()) {
      const onended = () => {
        if (this.stopped || this.finalReceived || this.micRecoveryAttempted) return;
        this.micRecoveryAttempted = true;
        void this.reacquire();
      };
      const onmute = () => {
        if (this.stopped || this.finalReceived) return;
        this.status("mic_muted", "The microphone is muted; the stream remains open.");
      };
      const onunmute = () => {
        if (this.stopped || this.finalReceived) return;
        this.status("mic_unmuted", "The microphone is active again.");
      };
      track.onended = onended;
      track.onmute = onmute;
      track.onunmute = onunmute;
      this.trackHandlers.push({ track, onended, onmute, onunmute });
    }
  }

  private unbindTracks(): void {
    for (const { track, onended, onmute, onunmute } of this.trackHandlers) {
      if (track.onended === onended) track.onended = null;
      if (track.onmute === onmute) track.onmute = null;
      if (track.onunmute === onunmute) track.onunmute = null;
    }
    this.trackHandlers = [];
  }

  private async reacquire(): Promise<void> {
    if (this.stopped || this.reacquiring || !this.capture) return;
    this.reacquiring = true;
    this.status("mic_reacquiring", "The microphone ended; reacquiring the input while preserving the turn journal.");
    this.capture.stop(); this.capture = null;
    this.releaseOwnStream("ended");
    try {
      const lease = await this.acquireStream();
      if (this.stopped) { releaseMicLease(lease, "ended"); return; }
      this.lease = lease;
      this.stream = lease.stream; this.bindTracks(lease.stream);
      this.capture = await this.makeCapture(lease.stream);
      this.micRecoveryAttempted = false;
      this.status("mic_reacquired", "Microphone reacquired; the active stream and recovery journal were preserved.");
    } catch {
      this.terminalFailure = true;
      this.diagnostic.error("mic_source_lost");
      this.diagnostic.terminal("failed", "mic_source_lost");
      this.publishDiagnostic();
      this.status("mic_source_lost", "Microphone access was lost and could not be reacquired; the recovery journal was retained.");
      this.onError?.("Microphone access was lost and could not be reacquired.");
    } finally { this.reacquiring = false; }
  }

  private async makeCapture(stream: MediaStream): Promise<PcmCapture> {
    return this.captureFactory(stream, (samples, rate) => this.queue(frameToCanonicalPcm16(samples, rate)));
  }

  private queue(samples: Int16Array): void {
    if (this.tailDropArmed) return;
    const sequence = this.sequence++;
    const startSample = this.sample;
    const endSample = startSample + BigInt(samples.length);
    this.sample = endSample;
    const bytes = new Uint8Array(samples.byteLength);
    bytes.set(new Uint8Array(samples.buffer, samples.byteOffset, samples.byteLength));
    this.allPcm.push(samples);
    this.diagnostic.captured(sequence);
    this.publishDiagnostic();
    this.writes = this.writes.then(async () => {
      const sha256 = await digestAudio(bytes.buffer);
      const frame = encodeAudioFrame({ sequence, startSample, endSample, audio: samples, sha256: new Uint8Array(sha256) });
      await this.journal?.append({ sequence, startSample, endSample, audio: bytes.buffer, sha256 });
      if (this.ws?.readyState === WebSocket.OPEN && this.pending.length === 0) {
        this.ws.send(frame);
        this.diagnostic.sent(sequence);
        this.publishDiagnostic();
      } else this.pending.push(frame);
    });
  }

  private install(ws: WebSocket, replay: boolean): void {
    ws.onopen = () => {
      this.ws = ws; this.reconnects = 0;
      this.serverAcknowledged = false;
      if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
      this.serverAckTimer = setTimeout(() => {
        if (!this.serverAcknowledged && !this.finalReceived) this.status("server_ack_pending", "Waiting for the speech backend to acknowledge the stream.");
      }, 1_000);
      void this.writes.then(() => {
        if (ws.readyState !== WebSocket.OPEN) return;
        if (replay) {
          for (const chunk of this.journal?.replayAfter(-1n) ?? []) {
            ws.send(encodeAudioFrame({ sequence: chunk.sequence, startSample: chunk.startSample, endSample: chunk.endSample, audio: new Uint8Array(chunk.audio), sha256: new Uint8Array(chunk.sha256) }));
            this.diagnostic.sent(chunk.sequence);
          }
        } else {
          for (const frame of this.pending) ws.send(frame);
          if (this.pending.length > 0) {
            this.diagnostic.sent(this.sequence - 1n);
          }
        }
        this.publishDiagnostic();
        this.pending = [];
        if (this.stopped && !this.doneSent) {
          ws.send(JSON.stringify({ type: "done" }));
          this.doneSent = true;
          this.diagnostic.done();
          this.publishDiagnostic();
        }
      });
      this.status(replay ? "replaying" : "stream_connected", replay ? "Replaying retained audio after reconnect." : "Streaming transcription connected.");
    };
    ws.onmessage = (event) => dispatchStreamMessage(event.data, {
      onPartial: (text) => this.onPartial?.(text),
      onSegmentFinal: (text, index) => this.onSegmentFinal?.(text, index),
      onSegmentAccepted: (index, score, threshold) => this.onSegmentAccepted?.(index, score, threshold),
      onSegmentRejected: (index, score, threshold) => this.onSegmentRejected?.(index, score, threshold),
      onSpeakerStatus: (enabled, profileConfigured) => this.onSpeakerStatus?.(enabled, profileConfigured),
      onVadState: (snapshot) => this.onVadState?.(snapshot),
      onStatus: (code, text, processed) => {
        this.serverAcknowledged = true;
        if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
        if (processed !== undefined) {
          this.diagnostic.processed(processed);
          this.publishDiagnostic();
          this.writes = this.writes.then(() => this.journal?.acknowledgeProcessed(processed) ?? undefined);
        }
        this.status(code, text);
      },
      onFinal: (text) => {
        if (this.terminalFailure) return;
        // A terminal final while capture is still active is not a successful
        // turn. It means the backend ended its stream before the user stopped
        // recording (including an empty final), so accepting it would discard
        // the retained PCM and bypass reconnect/HTTP recovery.
        if (!this.stopped) {
          this.status("backend_degraded", "The speech backend ended the stream while recording; recovering retained audio.");
          ws.close();
          return;
        }
        // A stopped turn with captured PCM but an empty terminal frame is not
        // a successful transcription. The transport may have closed during
        // recovery and emitted its empty drain frame; preserve the captured
        // turn and use the same HTTP recovery path as reconnect exhaustion.
        // Keep the no-audio case below as an intentional empty result.
        if (!text.trim() && this.allPcm.length > 0 && !this.terminalFailure) {
          void this.fallback();
          return;
        }
        this.finalReceived = true;
        this.unbindTracks();
        if (this.finalPendingTimer) clearTimeout(this.finalPendingTimer);
        if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
        this.onResult?.(text);
        this.diagnostic.terminal("completed", "final");
        this.publishDiagnostic();
        this.journal = null;
        forgetUnfinishedSession();
      },
      onError: (code, text) => {
        this.diagnostic.error(code);
        this.diagnostic.terminal("failed", code);
        this.publishDiagnostic();
        if (code === "incomplete_coverage") this.terminalFailure = true;
        this.onError?.(this.cleanServerError(text));
      },
    }, this.deliveredSegments);
    ws.onclose = () => {
      if (this.stopped || this.finalReceived || this.terminalFailure || !this.capture) return;
      if (this.reconnects >= MAX_CONSECUTIVE_RECONNECTS) {
        this.status("reconnect_exhausted", "Streaming recovery attempts were exhausted; using retained-audio recovery.");
        void this.fallback();
        return;
      }
      const delay = RECONNECT_BACKOFF_MS[this.reconnects] ?? RECONNECT_BACKOFF_MS.at(-1)!;
      this.reconnects += 1;
      this.status("reconnecting", "Connection interrupted; replaying retained audio.");
      setTimeout(() => this.connect(true), delay);
    };
  }

  private connect(replay: boolean): void { const ws = new WebSocket(this.wsUrl); this.install(ws, replay); }

  preConnect(language: string): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) return;
    this.language = language;
    this.sessionId = newSessionIdentity();
    this.resumeToken = newSessionIdentity();
    rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken });
    this.wsUrl = this.transport.buildStreamUrl(this.language, this.sessionId, this.resumeToken);
    this.connect(false);
    this.preconnectTimer = setTimeout(() => {
      if (!this.capture) {
        this.ws?.close();
        this.ws = null;
        forgetUnfinishedSession();
      }
    }, 30_000);
  }

  sendSegmentBoundary(): void {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ type: "segment-boundary" }));
  }

  sendVadState(speaking: boolean): void {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ type: speaking ? "vad-speech-start" : "vad-speech-end" }));
  }

  dropTail(): void { this.tailDropArmed = true; }

  private async fallback(): Promise<void> {
    this.stopped = true;
    this.capture?.stop();
    this.capture = null;
    this.unbindTracks();
    this.releaseOwnStream("provider-error");
    if (this.allPcm.length === 0) {
      this.terminalFailure = true;
      this.diagnostic.terminal("failed", "no_recoverable_audio");
      this.publishDiagnostic();
      this.onError?.("Streaming transcription ended without recoverable audio.");
      return;
    }
    try {
      this.finalReceived = true;
      const transcript = await this.transport.transcribeRetained(encodeWavFromPcm16(concatInt16(this.allPcm), TARGET_SAMPLE_RATE), this.language);
      this.onResult?.(transcript.trim());
      this.diagnostic.terminal("completed", "http_fallback");
      this.publishDiagnostic();
      this.journal = null;
      forgetUnfinishedSession();
    } catch {
      this.diagnostic.error("http_fallback_failed");
      this.diagnostic.terminal("failed", "http_fallback_failed");
      this.publishDiagnostic();
      this.onError?.("Streaming transcription and retained-audio recovery failed.");
    }
  }

  async start(prewarmed?: MediaStream): Promise<void> {
    if (this.preconnectTimer) clearTimeout(this.preconnectTimer);
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
    if (this.finalPendingTimer) clearTimeout(this.finalPendingTimer);
    if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
    this.stopped = false; this.finalReceived = false; this.terminalFailure = false; this.doneSent = false; this.reconnects = 0; this.micRecoveryAttempted = false; this.pending = []; this.allPcm = []; this.sequence = 0n; this.sample = 0n;
    const recovered = loadUnfinishedSession();
    if (recovered) { this.sessionId = recovered.sessionId; this.resumeToken = recovered.resumeToken; }
    else { this.sessionId = newSessionIdentity(); this.resumeToken = newSessionIdentity(); rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken }); }
    this.wsUrl = this.transport.buildStreamUrl(this.language, this.sessionId, this.resumeToken);
    this.lease = prewarmed
      ? registerMicStream("voice-stream", prewarmed)
      : await this.acquireStream();
    this.stream = this.lease.stream; this.bindTracks(this.stream);
    this.tailDropArmed = false;
    this.allStartedAt = Date.now();
    this.diagnostic.reset(this.sessionId, 0, "reduced");
    const persistent = this.options.journalFactory?.() ?? new TurnJournal(new IndexedDBTurnJournalStore(), this.sessionId, 0n, 16 * 1024 * 1024, "persistent");
    try { const snapshot = await persistent.restore(); this.journal = persistent; this.sequence = snapshot.nextSequence; this.sample = snapshot.nextSample; }
    catch { this.journal = new TurnJournal(new MemoryTurnJournalStore(), this.sessionId, 0n, 16 * 1024 * 1024, "reduced"); this.status("durability_reduced", "Persistent audio recovery is unavailable in this browser."); }
    this.capture = await this.makeCapture(this.stream); this.connect(this.sequence > 0n);
  }

  stop(): void {
    this.stopped = true;
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
    this.capture?.stop();
    this.capture = null;
    const stoppedAt = Date.now();
    this.lastTurn = this.tailDropArmed || this.allPcm.length === 0
      ? null
      : { blob: encodeWavFromPcm16(concatInt16(this.allPcm), TARGET_SAMPLE_RATE), mimeType: "audio/wav", durationMs: stoppedAt - this.allStartedAt, capturedAt: stoppedAt };
    // PCM frames are appended through `writes`; sending `done` synchronously
    // here can overtake the final frame when the encoder callback and stop
    // button land in the same turn. Drain the write chain first so the server
    // can account for every captured chunk before it evaluates coverage.
    if (this.ws?.readyState === WebSocket.OPEN) {
      void this.writes.then(() => {
        if (this.ws?.readyState !== WebSocket.OPEN || this.doneSent) return;
        this.ws.send(JSON.stringify({ type: "done" }));
        this.doneSent = true;
        this.diagnostic.done();
        this.publishDiagnostic();
      });
    }
    if (this.allPcm.length > 0 && !this.finalReceived && !this.terminalFailure) {
      this.finalPendingTimer = setTimeout(() => this.status("final_pending", "Speech audio was sent; waiting for the backend to finish transcription."), 3_000);
      this.fallbackTimer = setTimeout(() => { void this.fallback(); }, 10_000);
    }
    this.releaseOwnStream("manual-stop");
  }
  dispose(): void {
    this.stop();
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
    if (this.finalPendingTimer) clearTimeout(this.finalPendingTimer);
    if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
    this.unbindTracks();
    this.journal = null; this.pending = []; this.allPcm = []; this.lastTurn = null;
  }
}
