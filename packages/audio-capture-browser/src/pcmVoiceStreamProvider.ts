import { concatInt16, encodeWavFromPcm16, frameToCanonicalPcm16, TARGET_SAMPLE_RATE } from "./pcm";
import { createCanonicalPcmCapture, type PcmCapture } from "./pcmCapture";
import { digestAudio, encodeAudioFrame, newSessionIdentity } from "./protocol";
import { IndexedDBTurnJournalStore, MemoryTurnJournalStore, TurnJournal } from "./turnJournal";
import { forgetUnfinishedSession, loadUnfinishedSession, rememberUnfinishedSession } from "./sessionIdentity";
import { dispatchStreamMessage } from "./streamMessages";
import { requireVoiceTransport, type VoiceTransport, type VoiceTransportStatus } from "./transport";
import { publishStreamDiagnostic, StreamDiagnosticRecorder, type StreamTurnDiagnostic } from "./streamDiagnostic";
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
  /** Optional qualification-only wire batch size; production defaults to 100 ms. */
  readonly wireBatchSamples?: number;
  /** Optional test seam; production retains the fixed bounded recovery budget. */
  readonly maxRetainedTurnAudioBytes?: number;
}

export const MAX_CONSECUTIVE_RECONNECTS = 5;
export const RECONNECT_BACKOFF_MS = [1_000, 2_000, 4_000, 8_000, 8_000] as const;
export const PCM_WIRE_BATCH_SAMPLES = 1_600; // approximately 100 ms at 16 kHz
export const PCM_WIRE_BATCH_FLUSH_MS = 100;
export const FALLBACK_MIN_DELAY_MS = 10_000;
/**
 * Whole-turn audio is retained only for the bounded unary recovery/retry path.
 * The streaming journal has the same 16 MiB ceiling; keeping a second,
 * unbounded PCM copy would make a long native-streaming turn grow with its
 * duration even when the server is healthy. Once the whole-turn copy reaches
 * its ceiling, recovery uses the journal's unacknowledged tail instead.
 */
export const MAX_RETAINED_TURN_AUDIO_BYTES = 16 * 1024 * 1024;

/** Give a stopped turn time to drain durable writes before HTTP recovery. */
export function fallbackDelayMs(capturedSamples: bigint, pendingWrites: number): number {
  const audioDurationMs = Number((capturedSamples * 1_000n) / BigInt(TARGET_SAMPLE_RATE));
  const backlogMs = Math.max(0, pendingWrites) * PCM_WIRE_BATCH_FLUSH_MS;
  return Math.max(FALLBACK_MIN_DELAY_MS, audioDurationMs + backlogMs);
}

export class PcmStreamWriteError extends Error {
  readonly code = "capture_write_failed";

  constructor(readonly cause: unknown) {
    super("Audio capture could not be persisted safely.");
    this.name = "PcmStreamWriteError";
  }
}

/** Shared durable PCM transport used by scenario adapters. */
export class PcmVoiceStreamProvider {
  private readonly options: SharedPcmVoiceStreamProviderOptions;
  private readonly wireBatchSamples: number;
  private readonly maxRetainedTurnAudioBytes: number;
  private transport: VoiceTransport;
  language = "en";
  private ws: WebSocket | null = null;
  private capture: PcmCapture | null = null;
  private stream: MediaStream | null = null;
  private lease: MicLease | null = null;
  private journal: TurnJournal | null = null;
  private writes: Promise<void> = Promise.resolve();
  private journalAcks: Promise<void> = Promise.resolve();
  private pendingWriteCount = 0;
  private pending: ArrayBuffer[] = [];
  private batchSamples: Int16Array[] = [];
  private batchSampleCount = 0;
  private batchStartSample = 0n;
  private batchTimer: ReturnType<typeof setTimeout> | null = null;
  private allPcm: Int16Array[] = [];
  private allPcmBytes = 0;
  private retainedAudioOverflow = false;
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
    this.wireBatchSamples = Number.isSafeInteger(options.wireBatchSamples) && (options.wireBatchSamples ?? 0) > 0
      ? options.wireBatchSamples!
      : PCM_WIRE_BATCH_SAMPLES;
    this.maxRetainedTurnAudioBytes = Number.isSafeInteger(options.maxRetainedTurnAudioBytes) && (options.maxRetainedTurnAudioBytes ?? 0) > 0
      ? options.maxRetainedTurnAudioBytes!
      : MAX_RETAINED_TURN_AUDIO_BYTES;
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
  private publishDiagnostic(): void {
    const diagnostic = this.diagnostic.read();
    publishStreamDiagnostic(diagnostic);
    this.onDiagnostic?.(diagnostic);
  }
  private status(code: string, message: string): void { this.diagnostic.status(code); this.publishDiagnostic(); this.onStatus?.({ code, message }); }
  private cleanServerError(message: string): string {
    return /dial tcp|connect(?:ion)? refused|wss?:\/\//i.test(message)
      ? "The speech backend is unavailable; retry the turn."
      : message;
  }

  private handleWriteFailure(error: unknown): void {
    if (this.terminalFailure) return;
    const typed = error instanceof PcmStreamWriteError ? error : new PcmStreamWriteError(error);
    this.terminalFailure = true;
    this.diagnostic.error(typed.code);
    // Keep the public diagnostic metadata-only, but retain the browser error
    // class so qualification can distinguish quota, transaction, and
    // sequencing failures without exposing an audio payload or exception
    // message. This is especially important for accelerated journal soak.
    const cause = typed.cause as { name?: unknown } | null | undefined;
    if (typeof cause?.name === "string" && /^[A-Za-z][A-Za-z0-9_]*$/.test(cause.name)) {
      this.diagnostic.error(`${typed.code}:${cause.name}`);
    }
    this.diagnostic.terminal("failed", typed.code);
    this.publishDiagnostic();
    this.status(typed.code, "Audio recovery storage failed; the captured turn was retained for recovery.");
    this.onError?.("Audio capture could not be saved safely; the turn was retained for recovery.");
  }

  private hasRetainedTurnAudio(): boolean {
    if (!this.retainedAudioOverflow && this.allPcm.length > 0) return true;
    return (this.journal?.read().retainedBytes ?? 0) > 0;
  }

  /**
   * Choose the safest bounded recovery source. Short turns keep a complete
   * retry copy for speaker-policy/empty-result UX. Once that deliberately
   * bounded copy overflows during a long healthy session, the durable journal
   * still contains every frame the server has not acknowledged; recovering
   * that tail is safer than declaring the turn unrecoverable.
   */
  private recoveryPcm(): Int16Array | null {
    if (!this.retainedAudioOverflow && this.allPcm.length > 0) {
      return concatInt16(this.allPcm);
    }
    const chunks = this.journal?.read().chunks ?? [];
    if (chunks.length === 0) return null;
    return concatInt16(chunks.map((chunk) => new Int16Array(chunk.audio)));
  }

  /** Remove durable turn data only after a terminal transcript is committed. */
  private discardJournalAfterTerminal(): void {
    const journal = this.journal;
    this.journal = null;
    if (!journal) return;
    void Promise.all([this.writes, this.journalAcks])
      .then(() => journal.discard())
      .catch(() => {
        // The transcript is already committed, so this cannot change the
        // result. Keep the failure visible as a durability/storage signal so
        // a browser that cannot clean up does not silently accumulate turns.
        this.status("durability_cleanup_failed", "Completed speech was delivered, but local recovery storage could not be cleaned up.");
      });
  }

  private failWithoutRetainedAudio(code: string, message: string): void {
    if (this.finalReceived || this.terminalFailure) return;
    this.terminalFailure = true;
    this.diagnostic.error(code);
    this.diagnostic.terminal("failed", code);
    this.publishDiagnostic();
    this.onError?.(message);
  }

  private retainPcm(samples: Int16Array): void {
    if (this.retainedAudioOverflow) return;
    if (this.allPcmBytes + samples.byteLength > this.maxRetainedTurnAudioBytes) {
      // Once the budget is exceeded, discard the partial recovery copy too.
      // A partial WAV is not a safe retry artifact, and retaining it would
      // still make memory proportional to the turn up to the limit.
      this.allPcm = [];
      this.allPcmBytes = 0;
      this.retainedAudioOverflow = true;
      return;
    }
    this.allPcm.push(samples);
    this.allPcmBytes += samples.byteLength;
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
    const startSample = this.sample;
    this.sample += BigInt(samples.length);
		this.retainPcm(samples);
		for (let i = 0; i < samples.length; i += 1) {
			if (Math.abs(samples[i]) > 256) {
				this.diagnostic.signalObserved();
				break;
			}
		}
		this.diagnostic.capturedSamples(this.sample);
		this.diagnostic.captureObserved();
		if (this.batchSampleCount === 0) this.batchStartSample = startSample;
    let offset = 0;
    while (offset < samples.length) {
      const room = this.wireBatchSamples - this.batchSampleCount;
      const take = Math.min(room, samples.length - offset);
      this.batchSamples.push(samples.slice(offset, offset + take));
      this.batchSampleCount += take;
      offset += take;
      if (this.batchSampleCount === this.wireBatchSamples) this.flushBatch();
    }
    if (this.batchSampleCount > 0 && this.batchTimer === null) {
      this.batchTimer = setTimeout(() => {
        this.batchTimer = null;
        this.flushBatch();
			}, PCM_WIRE_BATCH_FLUSH_MS);
		}
		// Report the last wire interval touched by this capture callback. A
		// large virtual-corpus callback may flush many intervals, so publishing
		// before batching would incorrectly leave capturedSequence at zero while
		// the transport sends a much larger sequence range.
		const lastCaptured = this.batchSampleCount === 0 && this.sequence > 0n ? this.sequence - 1n : this.sequence;
		this.diagnostic.captured(lastCaptured);
		this.publishDiagnostic();
	}

  private flushBatch(): void {
    if (this.batchSampleCount === 0) return;
    if (this.batchTimer) {
      clearTimeout(this.batchTimer);
      this.batchTimer = null;
    }
    const audio = concatInt16(this.batchSamples);
    const sequence = this.sequence++;
    const startSample = this.batchStartSample;
    const endSample = startSample + BigInt(this.batchSampleCount);
    this.batchSamples = [];
    this.batchSampleCount = 0;
    this.batchStartSample = endSample;
    const bytes = new Uint8Array(audio.byteLength);
    bytes.set(new Uint8Array(audio.buffer, audio.byteOffset, audio.byteLength));
    this.pendingWriteCount += 1;
    this.writes = this.writes.then(async () => {
      try {
        const sha256 = await digestAudio(bytes.buffer);
        const frame = encodeAudioFrame({ sequence, startSample, endSample, audio, sha256: new Uint8Array(sha256) });
        await this.journal?.append({ sequence, startSample, endSample, audio: bytes.buffer, sha256 });
        this.diagnostic.retained(this.journal?.read().retainedBytes ?? 0);
        if (this.ws?.readyState === WebSocket.OPEN && this.pending.length === 0) {
          this.ws.send(frame);
          this.diagnostic.sent(sequence);
          this.publishDiagnostic();
        } else this.pending.push(frame);
      } catch (error: unknown) {
        this.handleWriteFailure(error);
      } finally {
        this.pendingWriteCount = Math.max(0, this.pendingWriteCount - 1);
      }
    });
  }

  private install(ws: WebSocket, replay: boolean): void {
    ws.onopen = () => {
      this.ws = ws; this.reconnects = 0;
      this.flushBatch();
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
      onPartial: (text) => {
        this.diagnostic.partial();
        this.publishDiagnostic();
        this.onPartial?.(text);
      },
      onSegmentFinal: (text, index) => {
        this.diagnostic.committed();
        this.publishDiagnostic();
        this.onSegmentFinal?.(text, index);
      },
      onSegmentAccepted: (index, score, threshold) => this.onSegmentAccepted?.(index, score, threshold),
      onSegmentRejected: (index, score, threshold) => this.onSegmentRejected?.(index, score, threshold),
      onSpeakerStatus: (enabled, profileConfigured) => this.onSpeakerStatus?.(enabled, profileConfigured),
      onVadState: (snapshot) => this.onVadState?.(snapshot),
      onStatus: (code, text, processed, providerIdentity) => {
        this.diagnostic.providerIdentity(providerIdentity?.providerId, providerIdentity?.modelId);
        this.serverAcknowledged = true;
        if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
        if (processed !== undefined) {
          this.diagnostic.processed(processed);
          this.diagnostic.retained(this.journal?.read().retainedBytes ?? 0);
          this.publishDiagnostic();
          // Acknowledgements are ordered independently from the capture write
          // queue. The send that produced this status only occurs after its
          // journal append completed; chaining onto the entire future capture
          // queue would let an hour of pending appends hit the quota before
          // the first acknowledgement could compact anything.
          this.journalAcks = this.journalAcks
            .then(async () => {
              await this.journal?.acknowledgeProcessed(processed);
              this.diagnostic.retained(this.journal?.read().retainedBytes ?? 0);
              this.publishDiagnostic();
            })
            .catch(() => {
              this.status("durability_reduced", "Audio recovery acknowledgement could not be persisted.");
            });
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
        // The server intentionally sends an empty terminal `final` after one
        // or more durable `segment-final` messages: those segments already
        // contain the committed transcript and the terminal frame only
        // transitions the turn. Treating that envelope as an empty
        // transcription starts retained-audio fallback after successful
        // segmented speech, which can leave the UI waiting forever for the
        // processed marker. Only an empty final with no committed segment is
        // eligible for recovery.
        if (!text.trim() && this.hasRetainedTurnAudio() && this.deliveredSegments.size === 0 && !this.terminalFailure) {
          void this.fallback();
          return;
        }
        if (!text.trim() && this.deliveredSegments.size === 0 && this.retainedAudioOverflow) {
          this.failWithoutRetainedAudio("retained_audio_budget_exceeded", "Streaming transcription ended without a complete bounded recovery copy; retry the turn.");
          return;
        }
        this.finalReceived = true;
        this.unbindTracks();
        if (this.finalPendingTimer) clearTimeout(this.finalPendingTimer);
        if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
        this.onResult?.(text);
        this.diagnostic.terminal("completed", "final");
        this.publishDiagnostic();
        this.discardJournalAfterTerminal();
        forgetUnfinishedSession();
      },
      onError: (code, text) => {
        // A deterministic/backend restart is a recoverable transport event.
        // Keep the turn in progress so onclose can replay the durable journal;
        // surfacing it through onError would make hosts mark the recording
        // failed before the bounded reconnect path gets a chance to recover.
        if (code === "backend_restart") {
          this.diagnostic.status(code);
          this.publishDiagnostic();
          this.status(code, this.cleanServerError(text));
          return;
        }
        this.diagnostic.error(code);
        this.diagnostic.terminal("failed", code);
        this.publishDiagnostic();
        if (code === "incomplete_coverage") this.terminalFailure = true;
        this.onError?.(this.cleanServerError(text));
      },
    }, this.deliveredSegments);
	ws.onclose = () => {
		this.flushBatch();
		if (this.stopped || this.finalReceived || this.terminalFailure || !this.capture) {
			// The backend may close the socket just before the user presses
			// stop. In that ordering there is no live socket on which to send
			// the terminal done frame, and the normal reconnect branch is
			// intentionally disabled by `stopped`. Recover the retained turn
			// through the same bounded HTTP path instead of leaving the
			// diagnostic in `preparing` forever.
			if (this.stopped && !this.finalReceived && !this.terminalFailure && this.hasRetainedTurnAudio()) void this.fallback();
			else if (this.stopped && !this.finalReceived && this.retainedAudioOverflow) this.failWithoutRetainedAudio("retained_audio_budget_exceeded", "Streaming transcription ended without a complete bounded recovery copy; retry the turn.");
			return;
		}
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
		if (this.finalReceived || this.terminalFailure) return;
		if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
		this.fallbackTimer = null;
	this.stopped = true;
    this.capture?.stop();
    this.capture = null;
    this.unbindTracks();
		this.releaseOwnStream("provider-error");
		if (!this.hasRetainedTurnAudio()) {
			this.failWithoutRetainedAudio(
				this.retainedAudioOverflow ? "retained_audio_budget_exceeded" : "no_recoverable_audio",
				this.retainedAudioOverflow
					? "Streaming transcription ended without a complete bounded recovery copy; retry the turn."
					: "Streaming transcription ended without recoverable audio.",
			);
			return;
		}
    try {
      this.status("buffered_recovery", "Streaming degraded; recovering retained audio through bounded batch transcription.");
      const pcm = this.recoveryPcm();
      if (!pcm || pcm.length === 0) {
        this.failWithoutRetainedAudio(
          this.retainedAudioOverflow ? "retained_audio_budget_exceeded" : "no_recoverable_audio",
          this.retainedAudioOverflow
            ? "Streaming transcription ended without a complete bounded recovery copy; retry the turn."
            : "Streaming transcription ended without recoverable audio.",
        );
        return;
      }
      this.finalReceived = true;
      const retained = encodeWavFromPcm16(pcm, TARGET_SAMPLE_RATE);
      const recovered = this.transport.transcribeRetainedWithIdentity
        ? await this.transport.transcribeRetainedWithIdentity(retained, this.language)
        : { text: await this.transport.transcribeRetained(retained, this.language) };
      this.diagnostic.providerIdentity(recovered.providerIdentity?.providerId, recovered.providerIdentity?.modelId);
      const transcript = recovered.text;
      const provider = recovered.providerIdentity?.providerId;
      const model = recovered.providerIdentity?.modelId;
      const identity = provider || model ? ` with ${provider ?? "unknown provider"}${model ? ` (${model})` : ""}` : "";
      this.status("buffered_recovery_completed", `Buffered recovery completed${identity}.`);
      this.onResult?.(transcript.trim());
      this.diagnostic.terminal("completed", "http_fallback");
      this.publishDiagnostic();
      this.discardJournalAfterTerminal();
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
	this.stopped = false; this.finalReceived = false; this.terminalFailure = false; this.doneSent = false; this.reconnects = 0; this.micRecoveryAttempted = false; this.pending = []; this.batchSamples = []; this.batchSampleCount = 0; this.batchStartSample = 0n; this.journalAcks = Promise.resolve(); this.pendingWriteCount = 0; this.allPcm = []; this.allPcmBytes = 0; this.retainedAudioOverflow = false; this.sequence = 0n; this.sample = 0n;
    const recovered = loadUnfinishedSession();
    if (recovered) { this.sessionId = recovered.sessionId; this.resumeToken = recovered.resumeToken; }
    else { this.sessionId = newSessionIdentity(); this.resumeToken = newSessionIdentity(); rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken }); }
    this.wsUrl = this.transport.buildStreamUrl(this.language, this.sessionId, this.resumeToken);
    // Publish the session before touching browser microphone APIs. A denied,
    // busy, or disappeared device must leave a machine-readable terminal
    // diagnostic; otherwise hosts can remain in "preparing" with no evidence
    // explaining why recording never reached the wire.
    this.diagnostic.reset(this.sessionId, 0, "reduced");
    this.diagnostic.status("capture_starting");
    this.publishDiagnostic();
    try {
      this.lease = prewarmed
        ? registerMicStream("voice-stream", prewarmed)
        : await this.acquireStream();
    } catch (error) {
      this.diagnostic.error("capture_start_failed");
      this.diagnostic.terminal("failed", "capture_start_failed");
      this.publishDiagnostic();
      throw error;
    }
    this.stream = this.lease.stream; this.bindTracks(this.stream);
    this.tailDropArmed = false;
    this.allStartedAt = Date.now();
    this.diagnostic.captureStarted();
    this.publishDiagnostic();
    const persistent = this.options.journalFactory?.() ?? new TurnJournal(new IndexedDBTurnJournalStore(), this.sessionId, 0n, this.maxRetainedTurnAudioBytes, "persistent");
    try { const snapshot = await persistent.restore(); this.journal = persistent; this.sequence = snapshot.nextSequence; this.sample = snapshot.nextSample; this.batchStartSample = snapshot.nextSample; }
    catch { this.journal = new TurnJournal(new MemoryTurnJournalStore(), this.sessionId, 0n, 16 * 1024 * 1024, "reduced"); this.status("durability_reduced", "Persistent audio recovery is unavailable in this browser."); }
    // Open the transport before starting capture production. Real microphone
    // factories are event-driven, while accelerated qualification sources may
    // emit a large virtual turn immediately; connecting first lets the server
    // acknowledge and compact durable journal frames while capture continues.
    this.connect(this.sequence > 0n);
    this.capture = await this.makeCapture(this.stream);
  }

  stop(): void {
    this.stopped = true;
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
	this.capture?.stop();
	this.capture = null;
	this.flushBatch();
	const stoppedAt = Date.now();
	const fullTurnRecovery = !this.retainedAudioOverflow && this.allPcm.length > 0
		? concatInt16(this.allPcm)
		: null;
	this.lastTurn = this.tailDropArmed || !fullTurnRecovery
		? null
	  : { blob: encodeWavFromPcm16(fullTurnRecovery, TARGET_SAMPLE_RATE), mimeType: "audio/wav", durationMs: stoppedAt - this.allStartedAt, capturedAt: stoppedAt };
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
	if (this.hasRetainedTurnAudio() && !this.finalReceived && !this.terminalFailure) {
      this.finalPendingTimer = setTimeout(() => this.status("final_pending", "Speech audio was sent; waiting for the backend to finish transcription."), 3_000);
      this.fallbackTimer = setTimeout(() => { void this.fallback(); }, fallbackDelayMs(this.sample, this.pendingWriteCount));
    }
    this.releaseOwnStream("manual-stop");
  }
  dispose(): void {
    this.stop();
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
    if (this.finalPendingTimer) clearTimeout(this.finalPendingTimer);
    if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
    this.unbindTracks();
    if (this.batchTimer) clearTimeout(this.batchTimer);
    this.batchTimer = null;
	this.journal = null; this.pending = []; this.allPcm = []; this.allPcmBytes = 0; this.retainedAudioOverflow = false; this.lastTurn = null;
  }
}
