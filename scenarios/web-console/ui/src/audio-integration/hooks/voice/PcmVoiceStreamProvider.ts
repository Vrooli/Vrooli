import { transcribeAudioWithRetry, buildVoiceStreamWsUrl } from "../../api/voice";
import {
  concatInt16,
  createCanonicalPcmCapture,
  digestAudio,
  encodeAudioFrame,
  encodeWavFromPcm16,
  frameToCanonicalPcm16,
  forgetUnfinishedSession,
  IndexedDBTurnJournalStore,
  loadUnfinishedSession,
  MemoryTurnJournalStore,
  newSessionIdentity,
  rememberUnfinishedSession,
  TARGET_SAMPLE_RATE,
  TurnJournal,
  type JournalSnapshot,
  type PcmCapture,
} from "@vrooli/audio-capture-browser";
import { acquireMicStream, releaseMicLease, type MicLease } from "./micOwnership";
import { getSharedAudioContext } from "./sharedAudioContext";
import type { LastTurnAudio, TranscriptionProvider } from "./types";
import { classifyMicError, computeFinalTimeout, WHISPER_FAILED_SENTINEL } from "./types";

/**
 * Canonical Web Console streaming provider. It deliberately has no terminal
 * vocabulary or proxy policy: it captures canonical PCM, journals it before a
 * v2 send, and relies on the same-origin URL adapter for transport routing.
 */
export class PcmVoiceStreamProvider implements TranscriptionProvider {
  private ws: WebSocket | null = null;
  private capture: PcmCapture | null = null;
  private lease: MicLease | null = null;
  private stream: MediaStream | null = null;
  private sessionId = "";
  private resumeToken = "";
  private wsUrl = "";
  private journal: TurnJournal | null = null;
  private writes: Promise<void> = Promise.resolve();
  private pending: ArrayBuffer[] = [];
  private allPCM: Int16Array[] = [];
  private retainedPcmBytes = 0;
  private retentionExhausted = false;
  private nextSequence = 0n;
  private nextSample = 0n;
  private reconnects = 0;
  private doneSent = false;
  private intentionallyStopped = false;
  private tailDropArmed = false;
  private finalReceived = false;
  private finalTimeout: ReturnType<typeof setTimeout> | null = null;
  private lastTurn: LastTurnAudio | null = null;
  private preconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private preconnected = false;
  private recordingStartedAt = 0;
  private stoppingAt = 0;
  private static readonly MAX_RETAINED_PCM_BYTES = 16 * 1024 * 1024;

  language = "en";
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onStatus: ((status: { code: string; message: string }) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;
  onSegmentFinal: ((text: string, segmentIndex: number) => void) | null = null;
  onSegmentAccepted: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSegmentRejected: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSpeakerStatus: ((enabled: boolean, profileConfigured: boolean) => void) | null = null;
  onVadState: ((snapshot: { voiced: boolean; silenceElapsedMs: number; silenceTimeoutMs: number; tickSeq: number; silenceTimedOut: boolean }) => void) | null = null;

  getStream(): MediaStream | null { return this.stream; }
  getLastTurnAudio(): LastTurnAudio | null { return this.lastTurn; }
  disposeLastTurn(): void { this.lastTurn = null; }

  private newIdentity(): boolean {
    try {
      this.sessionId = newSessionIdentity();
      this.resumeToken = newSessionIdentity();
      rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken });
      return true;
    } catch {
      this.onError?.("Secure browser identity generation is unavailable; dictation recovery cannot start.");
      return false;
    }
  }

  private releaseMic(reason: "manual-stop" | "owner-replaced" | "unmount" | "provider-error"): void {
    releaseMicLease(this.lease, reason);
    this.lease = null;
    this.stream = null;
  }

  private async initializeJournal(): Promise<JournalSnapshot> {
    const persistent = new TurnJournal(new IndexedDBTurnJournalStore(), this.sessionId, 0n, 16 * 1024 * 1024, "persistent");
    try {
      const snapshot = await persistent.restore();
      this.journal = persistent;
      return snapshot;
    } catch {
      const reduced = new TurnJournal(new MemoryTurnJournalStore(), this.sessionId, 0n, 16 * 1024 * 1024, "reduced");
      this.journal = reduced;
      this.onStatus?.({ code: "durability_reduced", message: "Persistent audio recovery is unavailable in this browser." });
      return reduced.read();
    }
  }

  private queueFrame(samples: Int16Array): void {
    if (this.tailDropArmed || this.retentionExhausted) return;
    if (this.retainedPcmBytes + samples.byteLength > PcmVoiceStreamProvider.MAX_RETAINED_PCM_BYTES) {
      this.retentionExhausted = true;
      this.onStatus?.({ code: "recovery_quota_exhausted", message: "Audio recovery storage reached its limit; this turn was stopped before further audio could be lost." });
      this.onError?.("Audio recovery storage reached its limit. Start a new turn to continue.");
      return;
    }
    const sequence = this.nextSequence++;
    const startSample = this.nextSample;
    const endSample = startSample + BigInt(samples.length);
    this.nextSample = endSample;
    this.retainedPcmBytes += samples.byteLength;
    const pcm = new Uint8Array(samples.byteLength);
    pcm.set(new Uint8Array(samples.buffer, samples.byteOffset, samples.byteLength));
    this.allPCM.push(samples);
    this.writes = this.writes.then(async () => {
      const digest = await digestAudio(pcm.buffer);
      const frame = encodeAudioFrame({ sequence, startSample, endSample, audio: samples, sha256: new Uint8Array(digest) });
      await this.journal?.append({ sequence, startSample, endSample, audio: pcm.buffer, sha256: digest });
      if (this.ws?.readyState === WebSocket.OPEN && this.pending.length === 0) this.ws.send(frame);
      else this.pending.push(frame);
    }).catch(() => this.onError?.("Audio recovery storage failed; stop and retry this turn."));
  }

  private flush(ws: WebSocket, replay: boolean): void {
    void this.writes.then(() => {
      if (ws.readyState !== WebSocket.OPEN) return;
      if (replay) {
        this.pending = [];
        for (const chunk of this.journal?.replayAfter(-1n) ?? []) {
          ws.send(encodeAudioFrame({ sequence: chunk.sequence, startSample: chunk.startSample, endSample: chunk.endSample, audio: new Uint8Array(chunk.audio), sha256: new Uint8Array(chunk.sha256) }));
        }
      } else {
        for (const frame of this.pending) ws.send(frame);
        this.pending = [];
      }
      this.sendDone(ws);
    });
  }

  private sendDone(ws: WebSocket): void {
    if (this.intentionallyStopped && !this.doneSent && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "done" }));
      this.doneSent = true;
    }
  }

  private setupSocket(ws: WebSocket, replay: boolean): void {
    ws.onopen = () => {
      this.ws = ws;
      this.flush(ws, replay);
      this.onStatus?.({ code: replay ? "replaying" : "stream_connected", message: replay ? "Replaying retained audio after reconnect." : "Streaming transcription connected." });
    };
    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data as string) as { type: string; text?: string; code?: string; processedSequence?: number; segmentIndex?: number; score?: number; threshold?: number; enabled?: boolean; profileConfigured?: boolean; voiced?: boolean; silenceElapsedMs?: number; silenceTimeoutMs?: number; tickSeq?: number; silenceTimedOut?: boolean };
        if (message.type === "status") {
          if (message.code === "processed_acknowledgement" && message.processedSequence !== undefined) {
            const processedSequence = message.processedSequence;
            this.writes = this.writes.then(async () => this.journal?.acknowledgeProcessed(BigInt(processedSequence)) ?? undefined);
          }
          this.onStatus?.({ code: message.code ?? "stream_status", message: message.text ?? "Streaming transcription status updated." });
        } else if (message.type === "partial" && message.text) this.onPartial?.(message.text);
        else if (message.type === "segment-final" && message.text !== undefined) this.onSegmentFinal?.(message.text, message.segmentIndex ?? 0);
        else if (message.type === "segment-accepted") this.onSegmentAccepted?.(message.segmentIndex ?? 0, message.score ?? 0, message.threshold ?? 0);
        else if (message.type === "segment-rejected") this.onSegmentRejected?.(message.segmentIndex ?? 0, message.score ?? 0, message.threshold ?? 0);
        else if (message.type === "speaker-status") this.onSpeakerStatus?.(Boolean(message.enabled), Boolean(message.profileConfigured));
        else if (message.type === "vad-state") this.onVadState?.({ voiced: Boolean(message.voiced), silenceElapsedMs: message.silenceElapsedMs ?? 0, silenceTimeoutMs: message.silenceTimeoutMs ?? 0, tickSeq: message.tickSeq ?? 0, silenceTimedOut: Boolean(message.silenceTimedOut) });
        else if (message.type === "final") {
          this.finalReceived = true;
          if (this.finalTimeout) clearTimeout(this.finalTimeout);
          this.onResult?.(message.text?.trim() ?? "");
          this.writes = this.writes.then(async () => this.journal?.discard() ?? undefined);
          forgetUnfinishedSession();
        } else if (message.type === "error") this.onError?.(message.text ?? "Streaming transcription failed");
      } catch { /* malformed status cannot change capture state */ }
    };
    ws.onclose = () => {
      if (this.finalReceived || this.intentionallyStopped || this.capture === null) return;
      if (this.reconnects >= 2) {
        this.onError?.(WHISPER_FAILED_SENTINEL);
        return;
      }
      const delay = this.reconnects++ === 0 ? 1_000 : 3_000;
      this.onStatus?.({ code: "reconnecting", message: "Connection interrupted; replaying retained audio." });
      setTimeout(() => this.connect(true), delay);
    };
  }

  private connect(replay: boolean): void {
    const ws = new WebSocket(this.wsUrl);
    this.setupSocket(ws, replay);
  }

  preConnect(language: string): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) return;
    this.language = language;
    if (!this.newIdentity()) return;
    this.wsUrl = buildVoiceStreamWsUrl(this.language, this.sessionId, this.resumeToken);
    this.preconnected = true;
    this.connect(false);
    this.preconnectTimer = setTimeout(() => {
      if (!this.capture) {
        this.ws?.close();
        this.ws = null;
        this.preconnected = false;
        forgetUnfinishedSession();
      }
    }, 30_000);
  }

  async start(): Promise<void> {
    if (this.preconnectTimer) clearTimeout(this.preconnectTimer);
    const reusePreconnect = this.preconnected && this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING);
    this.preconnected = false;
    if (!reusePreconnect) {
      this.ws?.close();
      const recovered = loadUnfinishedSession();
      if (recovered) {
        this.sessionId = recovered.sessionId;
        this.resumeToken = recovered.resumeToken;
        this.onStatus?.({ code: "recovery_resuming", message: "Resuming retained audio from an interrupted turn." });
      } else if (!this.newIdentity()) return;
      this.wsUrl = buildVoiceStreamWsUrl(this.language, this.sessionId, this.resumeToken);
    }
    try {
      this.lease = await acquireMicStream("voice-stream", { audio: true });
      this.stream = this.lease.stream;
    } catch (error) {
      this.onError?.(classifyMicError(error));
      return;
    }
    this.finalReceived = false;
    this.intentionallyStopped = false;
    this.tailDropArmed = false;
    this.doneSent = false;
    this.reconnects = 0;
    this.pending = [];
    this.allPCM = [];
    this.retainedPcmBytes = 0;
    this.retentionExhausted = false;
    this.nextSequence = 0n;
    this.nextSample = 0n;
    this.writes = Promise.resolve();
    this.recordingStartedAt = Date.now();
    const snapshot = await this.initializeJournal();
    this.nextSequence = snapshot.nextSequence;
    this.nextSample = snapshot.nextSample;
    this.capture = await createCanonicalPcmCapture(
      getSharedAudioContext(),
      this.stream,
      (samples, rate) => this.queueFrame(frameToCanonicalPcm16(samples, rate)),
    );
    if (reusePreconnect && this.ws) this.flush(this.ws, snapshot.chunks.length > 0);
    else this.connect(snapshot.chunks.length > 0);
  }

  sendSegmentBoundary(): void { if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ type: "segment-boundary" })); }
  sendVadState(speaking: boolean): void { if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ type: speaking ? "vad-speech-start" : "vad-speech-end" })); }
  dropTail(): void { this.tailDropArmed = true; }

  stop(): void {
    this.intentionallyStopped = true;
    this.stoppingAt = Date.now();
    this.capture?.stop();
    this.capture = null;
    this.lastTurn = this.tailDropArmed || this.allPCM.length === 0 ? null : { blob: encodeWavFromPcm16(concatInt16(this.allPCM), TARGET_SAMPLE_RATE), mimeType: "audio/wav", durationMs: this.stoppingAt - this.recordingStartedAt, capturedAt: this.stoppingAt };
    this.releaseMic("manual-stop");
    this.writes = this.writes.then(() => { if (this.ws) this.sendDone(this.ws); });
    const timeout = computeFinalTimeout(this.stoppingAt - this.recordingStartedAt);
    this.finalTimeout = setTimeout(() => { if (!this.finalReceived) this.attemptFallback(); }, timeout);
  }

  private attemptFallback(): void {
    if (this.allPCM.length === 0) { this.onError?.(WHISPER_FAILED_SENTINEL); return; }
    transcribeAudioWithRetry(encodeWavFromPcm16(concatInt16(this.allPCM), TARGET_SAMPLE_RATE), 2, this.language)
      .then((text) => { this.finalReceived = true; this.onResult?.(text.trim()); })
      .catch(() => this.onError?.(WHISPER_FAILED_SENTINEL));
  }

  dispose(): void {
    this.intentionallyStopped = true;
    if (this.finalTimeout) clearTimeout(this.finalTimeout);
    if (this.preconnectTimer) clearTimeout(this.preconnectTimer);
    this.capture?.stop();
    this.capture = null;
    this.ws?.close();
    this.ws = null;
    this.releaseMic("unmount");
    this.pending = [];
    this.lastTurn = null;
  }
}
