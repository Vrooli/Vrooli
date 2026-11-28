import { concatInt16, encodeWavFromPcm16, StreamingPcm16Resampler, TARGET_SAMPLE_RATE } from "./pcm";
import { createCanonicalPcmCapture, type PcmCapture } from "./pcmCapture";
import { digestAudio, newSessionIdentity } from "./streamIdentity";
import { IndexedDBTurnJournalStore, MemoryTurnJournalStore, TurnJournal } from "./turnJournal";
import { forgetUnfinishedSession, loadUnfinishedSession, rememberUnfinishedSession } from "./sessionIdentity";
import { dispatchStreamEvent } from "./streamMessages";
import { requireVoiceTransport, type VoiceAudioChunk, type VoiceStreamConnection, type VoiceStreamStart, type VoiceTransport, type VoiceTransportStatus } from "./transport";
import { publishStreamDiagnostic, StreamDiagnosticRecorder, type StreamTurnDiagnostic } from "./streamDiagnostic";
import { RetainedPcmBuffer } from "./retainedPcmBuffer";
import { classifyPcmWriteFailure } from "./pcmWriteFailurePolicy";
import { ReconnectBackoff } from "./reconnectBackoff";
export { MAX_CONSECUTIVE_RECONNECTS, RECONNECT_BACKOFF_MS } from "./reconnectBackoff";
import { registerMicStream, releaseMicLease, type MicLease, type MicReleaseReason } from "./voice/micOwnership";
import { getSharedAudioContext } from "./voice/sharedAudioContext";
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
  readonly onSegmentFinal?: (text: string, index: number, speakerPolicy?: string, endSample?: number) => void;
  readonly captureFactory?: SharedPcmCaptureFactory;
  readonly journalFactory?: () => TurnJournal;
  readonly retainStream?: boolean;
  /** Optional qualification-only wire batch size; production defaults to 100 ms. */
  readonly wireBatchSamples?: number;
  /** Send a small first frame to reduce time-to-first-token on native streams. */
  readonly fastFirstWireBatch?: boolean;
  /** Optional test seam; production retains the fixed bounded recovery budget. */
  readonly maxRetainedTurnAudioBytes?: number;
}

export const PCM_WIRE_BATCH_SAMPLES = 1_600; // approximately 100 ms at 16 kHz
// Send the first voiced audio promptly so the native decoder can begin while
// the normal 100 ms batching policy continues to control steady-state wire
// overhead. This is intentionally a transport optimization, not a quality
// shortcut: the server still receives every sample with exact cursors.
export const PCM_FIRST_WIRE_BATCH_SAMPLES = 320; // approximately 20 ms at 16 kHz
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
  private readonly fastFirstWireBatch: boolean;
  private readonly maxRetainedTurnAudioBytes: number;
  private readonly retainedPcm: RetainedPcmBuffer;
  private transport: VoiceTransport;
  language = "en";
  // Assigned as soon as the transport opens so capture frames stay queued
  // while the voice stream is handshaking.
  private connection: VoiceStreamConnection | null = null;
  private capture: PcmCapture | null = null;
  private stream: MediaStream | null = null;
  private lease: MicLease | null = null;
  private journal: TurnJournal | null = null;
  // Admission may overlap durable recovery, but replay and PCM writes must not
  // begin until the journal has established the authoritative cursors.
  private journalReady = false;
  private writes: Promise<void> = Promise.resolve();
  private captureGeneration = 0;
  private captureSourceGeneration = 0;
  private queuedBytes = 0;
  private journalAcks: Promise<void> = Promise.resolve();
  private pendingWriteCount = 0;
  private pending: VoiceAudioChunk[] = [];
  private replaying = false;
  private batchSamples: Int16Array[] = [];
  private batchSampleCount = 0;
  private batchStartSample = 0n;
  private batchTimer: ReturnType<typeof setTimeout> | null = null;
  private sessionId = "";
  private resumeToken = "";
  private activationGeneration = -1;
  private connectionReady = false;
  private preconnectWarmed = false;
  private preconnectResumedSession = false;
  private sequence = 0n;
  private lastRecoveryProgressSequence = -1n;
  private sample = 0n;
  private readonly reconnectBackoff = new ReconnectBackoff();
  private stopped = false;
  private finalReceived = false;
  private startRequestedAtMs: number | null = null;
  private captureReadyReported = false;
  private reacquiringGeneration: number | null = null;
  private readonly deliveredSegments = new Set<string>();
  private allStartedAt = 0;
  private tailDropArmed = false;
  private lastTurn: LastTurnAudio | null = null;
  private diagnostic = new StreamDiagnosticRecorder();
  private readonly pcmResampler = new StreamingPcm16Resampler();
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
	onSegmentFinal: ((text: string, index: number, speakerPolicy?: string, endSample?: number) => void) | null = null;
  onSegmentAccepted: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSegmentRejected: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSpeakerStatus: ((enabled: boolean, profileConfigured: boolean) => void) | null = null;
  onVadState: ((snapshot: { voiced: boolean; silenceElapsedMs: number; silenceTimeoutMs: number; tickSeq: number; silenceTimedOut: boolean }) => void) | null = null;

  constructor(options: SharedPcmVoiceStreamProviderOptions = {}) {
    this.options = options;
    this.wireBatchSamples = Number.isSafeInteger(options.wireBatchSamples) && (options.wireBatchSamples ?? 0) > 0
      ? options.wireBatchSamples!
      : PCM_WIRE_BATCH_SAMPLES;
    this.fastFirstWireBatch = options.fastFirstWireBatch ?? false;
    this.maxRetainedTurnAudioBytes = Number.isSafeInteger(options.maxRetainedTurnAudioBytes) && (options.maxRetainedTurnAudioBytes ?? 0) > 0
      ? options.maxRetainedTurnAudioBytes!
      : MAX_RETAINED_TURN_AUDIO_BYTES;
    this.retainedPcm = new RetainedPcmBuffer(this.maxRetainedTurnAudioBytes);
    this.transport = options.transport ?? requireVoiceTransport();
    this.language = options.language ?? "en";
    this.captureFactory = options.captureFactory ?? ((stream, onFrame) => {
      if (typeof AudioContext === "undefined") return { stop: () => {} };
      return createCanonicalPcmCapture(this.options.getAudioContext?.() ?? getSharedAudioContext(), stream, onFrame);
    });
    this.retainStream = options.retainStream ?? false;
    this.onStatus = options.onStatus ?? null;
    this.onResult = options.onResult ?? null;
    this.onError = options.onError ?? null;
    this.onPartial = options.onPartial ?? null;
    this.onSegmentFinal = options.onSegmentFinal ?? null;
  }

  getStream(): MediaStream | null { return this.stream; }

  /** True when the permission-warmed voice stream can accept capture frames. */
  isStreamConnected(): boolean {
    return this.connectionReady && (this.connection?.isOpen() ?? false);
  }

  getDiagnostic(): StreamTurnDiagnostic { return this.diagnostic.read(); }
  exportDiagnostic(): string { return this.diagnostic.exportJSON(); }
  /** Preserve the consumer's user-action timestamp across asynchronous startup. */
  markStartRequested(atMs?: number): void {
    const now = typeof globalThis.performance?.now === "function" ? globalThis.performance.now() : Date.now();
    this.startRequestedAtMs = Number.isFinite(atMs) && atMs! >= 0 ? atMs! : now;
  }
  /** Records one transcript revision after the consuming UI committed it. */
  recordVisibleTranscriptRevision(): void {
    this.diagnostic.partial();
    this.diagnostic.visibleRevisionRendered();
    this.publishDiagnostic();
  }
  getLastTurnAudio(): LastTurnAudio | null { return this.lastTurn; }
  disposeLastTurn(): void { this.lastTurn = null; }
  private publishDiagnostic(): void {
    const diagnostic = this.diagnostic.read();
    publishStreamDiagnostic(diagnostic);
    this.onDiagnostic?.(diagnostic);
  }

  private markCaptureReady(): void {
    if (this.captureReadyReported || !this.connectionReady || !this.capture || this.stopped) return;
    this.captureReadyReported = true;
    this.diagnostic.captureReady();
    this.publishDiagnostic();
    this.startRequestedAtMs = null;
  }

  private downgradePersistentJournal(): TurnJournal | null {
    const current = this.journal;
    const snapshot = current?.read();
    if (!snapshot || snapshot.durability !== "persistent") return null;
    const reduced = new TurnJournal(
      new MemoryTurnJournalStore(),
      snapshot.sessionId,
      snapshot.generation,
      16 * 1024 * 1024,
      "reduced",
    );
    reduced.seed(snapshot);
    this.journal = reduced;
    this.diagnostic.durability("reduced");
    this.status("durability_reduced", "Persistent recovery storage became unavailable; the active turn remains recoverable in bounded memory.");
    return reduced;
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
    this.capture?.stop();
    this.capture = null;
    this.releaseOwnStream("manual-stop");
    this.diagnostic.error(typed.code);
    // Keep the public diagnostic metadata-only, but retain the browser error
    // class so qualification can distinguish quota, transaction, and
    // sequencing failures without exposing an audio payload or exception
    // message. This is especially important for accelerated journal soak.
    const cause = typed.cause as { name?: unknown } | null | undefined;
    if (typeof cause?.name === "string" && /^[A-Za-z][A-Za-z0-9_]*$/.test(cause.name)) {
      this.diagnostic.error(`${typed.code}:${cause.name}`);
    }
    const message = typed.cause instanceof Error ? typed.cause.message : String(typed.cause ?? "");
    const classification = classifyPcmWriteFailure(typeof cause?.name === "string" ? cause.name : "", message);
    this.diagnostic.error(`${typed.code}:${classification.reason}`);
    if (classification.sequenceDetail) {
      const [expectedSequence, expectedSample, actualSequence, actualSample] = classification.sequenceDetail;
      this.diagnostic.error(`${typed.code}:cursor_${expectedSequence}_got_${actualSequence}`);
      this.diagnostic.error(`${typed.code}:sample_${expectedSample}_got_${actualSample}`);
    }
    this.diagnostic.terminal("failed", typed.code);
    this.publishDiagnostic();
    this.status(typed.code, "Audio recovery storage failed; the captured turn was retained for recovery.");
    this.onError?.("Audio capture could not be saved safely; the turn was retained for recovery.");
  }

  private hasRetainedTurnAudio(): boolean {
    if (this.retainedPcm.hasCompleteAudio) return true;
    return (this.journal?.retainedBytes ?? 0) > 0;
  }

  /**
   * Choose the safest bounded recovery source. Short turns keep a complete
   * retry copy for speaker-policy/empty-result UX. Once that deliberately
   * bounded copy overflows during a long healthy session, the durable journal
   * still contains every frame the server has not acknowledged; recovering
   * that tail is safer than declaring the turn unrecoverable.
   */
  private recoveryPcm(): Int16Array | null {
    const completeCopy = this.retainedPcm.snapshot();
    if (completeCopy) return completeCopy;
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
    this.retainedPcm.append(samples);
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
    if (this.stopped || this.reacquiringGeneration === this.captureGeneration || !this.capture) return;
    const recoveryGeneration = this.captureGeneration;
    this.reacquiringGeneration = recoveryGeneration;
    this.status("mic_reacquiring", "The microphone ended; reacquiring the input while preserving the turn journal.");
    this.captureSourceGeneration += 1;
    this.capture.stop(); this.capture = null;
    this.releaseOwnStream("ended");
    try {
      const lease = await this.acquireStream();
      if (this.stopped || recoveryGeneration !== this.captureGeneration) {
        releaseMicLease(lease, "ended");
        return;
      }
      this.lease = lease;
      this.stream = lease.stream; this.bindTracks(lease.stream);
      const capture = await this.makeCapture(lease.stream);
      if (this.stopped || recoveryGeneration !== this.captureGeneration) {
        capture.stop();
        if (this.lease === lease) this.releaseOwnStream("ended");
        else releaseMicLease(lease, "ended");
        return;
      }
      this.capture = capture;
      this.micRecoveryAttempted = false;
      this.status("mic_reacquired", "Microphone reacquired; the active stream and recovery journal were preserved.");
    } catch {
      if (this.stopped || recoveryGeneration !== this.captureGeneration) return;
      // The lease is installed before makeCapture(), which may reject while
      // constructing the worklet/ScriptProcessor graph. A failed recovery is
      // terminal, so release any newly acquired stream before reporting it.
      this.releaseOwnStream("provider-error");
      this.terminalFailure = true;
      this.diagnostic.error("mic_source_lost");
      this.diagnostic.terminal("failed", "mic_source_lost");
      this.publishDiagnostic();
      this.status("mic_source_lost", "Microphone access was lost and could not be reacquired; the recovery journal was retained.");
      this.onError?.("Microphone access was lost and could not be reacquired.");
    } finally {
      if (this.reacquiringGeneration === recoveryGeneration) this.reacquiringGeneration = null;
    }
  }

  private async makeCapture(stream: MediaStream): Promise<PcmCapture> {
    const generation = this.captureGeneration;
    const sourceGeneration = ++this.captureSourceGeneration;
    return this.captureFactory(stream, (samples, rate) => {
      const callbackStartedAtMs = typeof globalThis.performance?.now === "function" ? globalThis.performance.now() : Date.now();
      if (generation !== this.captureGeneration
        || sourceGeneration !== this.captureSourceGeneration
        || this.stopped
        || this.terminalFailure) return;
      this.diagnostic.captureRate(rate);
      this.queue(this.pcmResampler.process(samples, rate));
      const callbackEndedAtMs = typeof globalThis.performance?.now === "function" ? globalThis.performance.now() : Date.now();
      this.diagnostic.captureCallbackCost(callbackEndedAtMs - callbackStartedAtMs, samples.length * 1_000 / rate);
    });
  }

  private queue(samples: Int16Array): void {
    if (this.tailDropArmed || this.terminalFailure || this.stopped) return;
    if (this.queuedBytes + this.batchSampleCount * 2 + samples.byteLength + (this.journal?.retainedBytes ?? 0) > MAX_RETAINED_TURN_AUDIO_BYTES) {
      this.handleWriteFailure(new Error("journal quota exhausted before capture; audio was not discarded"));
      return;
    }
    const startSample = this.sample;
    this.sample += BigInt(samples.length);
		this.retainPcm(samples);
		for (let i = 0; i < samples.length; i += 1) {
			if (Math.abs(samples[i] ?? 0) > 256) {
				const samplesAfterSignal = samples.length - i - 1;
				this.diagnostic.signalObserved(samplesAfterSignal * 1_000 / TARGET_SAMPLE_RATE, startSample + BigInt(i));
				break;
			}
		}
		this.diagnostic.capturedSamples(this.sample);
		this.diagnostic.captureObserved();
		if (this.batchSampleCount === 0) this.batchStartSample = startSample;
    let offset = 0;
    while (offset < samples.length) {
      const room = this.wireBatchLimit() - this.batchSampleCount;
      const take = Math.min(room, samples.length - offset);
      this.batchSamples.push(samples.slice(offset, offset + take));
      this.batchSampleCount += take;
      offset += take;
      if (this.batchSampleCount === this.wireBatchLimit()) this.flushBatch();
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
    const generation = this.captureGeneration;
    const journal = this.journal;
    this.queuedBytes += bytes.byteLength;
    this.pendingWriteCount += 1;
    this.writes = this.writes.then(async () => {
      try {
        if (generation !== this.captureGeneration || this.terminalFailure || !journal) return;
    const sha256 = await digestAudio(bytes.buffer);
        if (generation !== this.captureGeneration || this.terminalFailure) return;
        const journalChunk = { sequence, startSample, endSample, audio: bytes.buffer, sha256 };
        const chunk: VoiceAudioChunk = { ...journalChunk, audio: new Uint8Array(bytes.buffer), sha256: new Uint8Array(sha256) };
        try {
          await (this.journal ?? journal).append(journalChunk);
        } catch (error) {
          if (generation !== this.captureGeneration || this.terminalFailure) return;
          const reduced = this.downgradePersistentJournal();
          if (!reduced) throw error;
          await reduced.append(journalChunk);
        }
        if (generation !== this.captureGeneration || this.terminalFailure) return;
        this.diagnostic.retained(this.journal?.retainedBytes ?? 0);
        if (this.connection?.isOpen() && !this.replaying && this.pending.length === 0) {
          this.connection.sendAudio(chunk);
          this.diagnostic.sent(sequence);
          this.publishDiagnostic();
        } else {
          this.pending.push(chunk);
        }
      } catch (error) {
        if (generation === this.captureGeneration) this.handleWriteFailure(error);
      } finally {
        if (generation === this.captureGeneration) {
          this.queuedBytes -= bytes.byteLength;
          this.pendingWriteCount = Math.max(0, this.pendingWriteCount - 1);
        }
      }
    });
  }

  private wireBatchLimit(): number {
    return this.fastFirstWireBatch && this.sequence === 0n
      ? Math.min(this.wireBatchSamples, PCM_FIRST_WIRE_BATCH_SAMPLES)
      : this.wireBatchSamples;
  }

  private streamStart(): VoiceStreamStart {
    return {
      language: this.language,
      sessionId: this.sessionId,
      resumeToken: this.resumeToken,
      protocolVersion: 2,
      generation: 0n,
      inputFormat: "pcm_s16le",
      sampleRateHz: TARGET_SAMPLE_RATE,
    };
  }

  /** Bind one voice stream to this provider; callbacks are fenced by identity. */
  private install(replay: boolean): void {
    const slot: { connection: VoiceStreamConnection | null } = { connection: null };
    const connectionIsCurrent = () => slot.connection !== null && this.connection === slot.connection;
    const handlers = {
      onReady: () => {
        if (!connectionIsCurrent() || this.terminalFailure) {
          slot.connection?.close();
          return;
        }
        this.connectionReady = true;
        this.markCaptureReady();
        if (this.journalReady) this.activateConnection(slot.connection!, replay || this.sequence > 0n);
      },
      onEvent: (event: import("./streamMessages").StreamMessage) => {
        if (!connectionIsCurrent()) return;
        const generation = this.captureGeneration;
        dispatchStreamEvent(event, {
          onPartial: (text) => {
            this.onPartial?.(text);
          },
          onSegmentFinal: (text, index, speakerPolicy, endSample) => {
            this.diagnostic.committed(endSample);
            this.publishDiagnostic();
            this.onSegmentFinal?.(text, index, speakerPolicy, endSample);
          },
          onSegmentAccepted: (index, score, threshold) => this.onSegmentAccepted?.(index, score, threshold),
          onSegmentRejected: (index, score, threshold) => this.onSegmentRejected?.(index, score, threshold),
          onProviderIdentity: (identity) => {
            this.diagnostic.providerIdentity(identity.providerId, identity.modelId, identity.providerTier);
            this.publishDiagnostic();
          },
          onSpeakerStatus: (enabled, profileConfigured) => this.onSpeakerStatus?.(enabled, profileConfigured),
          onVadState: (snapshot) => this.onVadState?.(snapshot),
          onStatus: (code, text, processed, providerIdentity) => {
            this.diagnostic.providerIdentity(providerIdentity?.providerId, providerIdentity?.modelId, providerIdentity?.providerTier);
            this.serverAcknowledged = true;
            if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
            if (processed !== undefined) {
              this.diagnostic.processed(processed);
              this.diagnostic.retained(this.journal?.retainedBytes ?? 0);
              this.publishDiagnostic();
              const journal = this.journal;
              this.journalAcks = this.journalAcks
                .then(async () => {
                  if (generation !== this.captureGeneration || this.terminalFailure || !journal) return;
                  await journal.acknowledgeProcessed(processed);
                  if (generation !== this.captureGeneration || this.terminalFailure) return;
                  // A ready handshake alone is not evidence that a reconnect
                  // recovered any audio. Reset the retry budget only after a
                  // newly processed cursor has been durably compacted; repeated
                  // ready-then-close sockets with no progress must exhaust.
                  if (processed > this.lastRecoveryProgressSequence) {
                    this.lastRecoveryProgressSequence = processed;
                    this.reconnectBackoff.reset();
                  }
                  this.diagnostic.retained(this.journal?.retainedBytes ?? 0);
                  this.publishDiagnostic();
                })
                .catch(() => {
                  if (generation !== this.captureGeneration) return;
                  this.status("durability_reduced", "Audio recovery acknowledgement could not be persisted.");
                });
            }
            this.status(code, text);
          },
          onFinal: (text) => {
            if (this.terminalFailure) return;
            if (!this.stopped) {
              this.status("backend_degraded", "The speech backend ended the stream while recording; recovering retained audio.");
              slot.connection?.close();
              return;
            }
            void this.writes.then(() => {
              if (generation !== this.captureGeneration || this.terminalFailure || this.finalReceived) return;
              if (!text.trim() && this.hasRetainedTurnAudio() && this.deliveredSegments.size === 0 && !this.terminalFailure) {
                void this.fallback();
                return;
              }
              if (!text.trim() && this.deliveredSegments.size === 0 && this.retainedPcm.isOverflowed) {
                this.failWithoutRetainedAudio("retained_audio_budget_exceeded", "Streaming transcription ended without a complete bounded recovery copy; retry the turn.");
                return;
              }
              this.finalReceived = true;
              this.diagnostic.streamingFinalReceived();
              if (text.trim()) {
                this.diagnostic.committed(Number(this.sample));
                this.publishDiagnostic();
              }
              this.unbindTracks();
              if (this.finalPendingTimer) clearTimeout(this.finalPendingTimer);
              if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
              this.onResult?.(text);
              this.diagnostic.terminal("completed", "final");
              this.publishDiagnostic();
              this.discardJournalAfterTerminal();
              forgetUnfinishedSession();
            });
          },
          onError: (code, text) => {
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
      },
      onClose: (error?: unknown) => {
        if (!connectionIsCurrent()) return;
        const failedBeforeAdmission = error != null && !this.connectionReady;
        this.connectionReady = false;
        this.activationGeneration = -1;
        if (this.connection === slot.connection) this.connection = null;
        this.flushBatch();
        if (failedBeforeAdmission) {
          this.status("stream_admission_failed", "The speech server did not accept the streaming session; retained audio is protected while automatic recovery retries.");
        }
        this.handleConnectionClosed();
      },
    };
    const connection = this.transport.openStream(this.streamStart(), handlers);
    slot.connection = connection;
    this.connection = connection;
  }

  private activateConnection(connection: VoiceStreamConnection, replay: boolean): void {
    const generation = this.captureGeneration;
    if (!this.journalReady || this.activationGeneration === generation || this.connection !== connection || this.terminalFailure) return;
    this.activationGeneration = generation;
    this.replaying = replay;
    if (!replay) this.flushBatch();
    this.serverAcknowledged = false;
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
    this.serverAckTimer = setTimeout(() => {
      if (!this.serverAcknowledged && !this.finalReceived) this.status("server_ack_pending", "Waiting for the speech backend to acknowledge the stream.");
    }, 1_000);

    const sendEndWhenStopped = () => {
      if (generation !== this.captureGeneration || this.terminalFailure || this.connection !== connection || !connection.isOpen() || this.doneSent) return;
      connection.end();
      this.doneSent = true;
      this.diagnostic.done();
      this.publishDiagnostic();
    };
    void this.writes.then(async () => {
      if (generation !== this.captureGeneration || this.terminalFailure || this.connection !== connection || !connection.isOpen()) return;
      if (replay) {
        const replayFloor = this.journal?.nextSequence ?? this.sequence;
        this.pending = this.pending.filter((item) => item.sequence >= replayFloor);
        for (const chunk of this.journal?.replayAfter(-1n) ?? []) {
          connection.sendAudio({ sequence: chunk.sequence, startSample: chunk.startSample, endSample: chunk.endSample, audio: new Uint8Array(chunk.audio), sha256: new Uint8Array(chunk.sha256) });
          this.diagnostic.sent(chunk.sequence);
        }
      }
      this.replaying = false;
      for (const item of this.pending.sort((a, b) => Number(a.sequence - b.sequence))) {
        if (!connection.isOpen()) break;
        connection.sendAudio(item);
        this.diagnostic.sent(item.sequence);
      }
      if (this.pending.length > 0) this.diagnostic.sent(this.sequence - 1n);
      this.pending = [];
      this.publishDiagnostic();
      if (this.stopped) sendEndWhenStopped();
      this.status(replay ? "replaying" : "stream_connected", replay ? "Replaying retained audio after reconnect." : "Streaming transcription connected.");
    });
  }

  private handleConnectionClosed(): void {
    const reconnectAfterStop = this.stopped && !this.finalReceived && !this.terminalFailure && this.hasRetainedTurnAudio();
    if (this.finalReceived || this.terminalFailure || (!this.capture && !reconnectAfterStop)) {
      if (this.stopped && !this.finalReceived && !this.terminalFailure && this.retainedPcm.isOverflowed) {
        this.failWithoutRetainedAudio("retained_audio_budget_exceeded", "Streaming transcription ended without a complete bounded recovery copy; retry the turn.");
      }
      return;
    }
    // A stopped turn may still be waiting for its final. If that socket closes,
    // preserve the bounded reconnect budget and replay every unacknowledged
    // interval before considering unary recovery. Some selected native models
    // are streaming-only and cannot serve the retained-audio batch path.
    if (reconnectAfterStop && this.fallbackTimer) {
      clearTimeout(this.fallbackTimer);
      this.fallbackTimer = null;
    }
    const delay = this.reconnectBackoff.nextDelay();
    if (delay === null) {
      this.status("reconnect_exhausted", "Streaming recovery attempts were exhausted; using retained-audio recovery.");
      void this.fallback();
      return;
    }
    this.status("reconnecting", "Connection interrupted; replaying retained audio.");
    const generation = this.captureGeneration;
    setTimeout(() => {
      const stoppedTurnRetained = this.stopped && !this.finalReceived && !this.terminalFailure && this.hasRetainedTurnAudio();
      if (generation === this.captureGeneration && !this.terminalFailure && (!this.stopped || stoppedTurnRetained)) this.connect(true);
    }, delay);
  }

  private connect(replay: boolean): void {
    this.connectionReady = false;
    this.replaying = replay;
    this.install(replay);
  }

  preConnect(language: string): void {
    if (this.connection) return;
    this.language = language;
    const recovered = loadUnfinishedSession();
    this.preconnectResumedSession = recovered !== null;
    this.sessionId = recovered?.sessionId ?? newSessionIdentity();
    this.resumeToken = recovered?.resumeToken ?? newSessionIdentity();
    this.preconnectWarmed = true;
    this.connect(false);
    this.preconnectTimer = setTimeout(() => {
      if (!this.capture) {
        this.connection?.close();
        this.connection = null;
        this.connectionReady = false;
        this.preconnectWarmed = false;
        this.preconnectResumedSession = false;
      }
    }, 30_000);
  }

  dropTail(): void { this.tailDropArmed = true; }

  markStopRequested(): void {
    this.diagnostic.stopRequested();
    this.publishDiagnostic();
  }

	private async fallback(): Promise<void> {
		if (this.finalReceived || this.terminalFailure) return;
    const generation = this.captureGeneration;
		if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
		this.fallbackTimer = null;
	this.stopped = true;
    this.capture?.stop();
    this.capture = null;
    this.unbindTracks();
		this.releaseOwnStream("provider-error");
		// Make bounded recovery an explicit lifecycle state immediately. The
		// native stream can have acknowledged every captured interval while its
		// empty terminal is being recovered through the configured batch engine;
		// leaving the diagnostic in recording hides processed coverage and makes
		// the voice control appear stuck during a legitimate recovery.
		this.diagnostic.state("recovering");
		this.publishDiagnostic();
		if (!this.hasRetainedTurnAudio()) {
			this.failWithoutRetainedAudio(
				this.retainedPcm.isOverflowed ? "retained_audio_budget_exceeded" : "no_recoverable_audio",
				this.retainedPcm.isOverflowed
					? "Streaming transcription ended without a complete bounded recovery copy; retry the turn."
					: "Streaming transcription ended without recoverable audio.",
			);
			return;
		}
    try {
      this.flushBatch();
      await this.writes;
      if (generation !== this.captureGeneration || this.terminalFailure) return;
      this.status("buffered_recovery", "Streaming degraded; recovering retained audio through bounded batch transcription.");
      const pcm = this.recoveryPcm();
      if (!pcm || pcm.length === 0) {
        this.failWithoutRetainedAudio(
          this.retainedPcm.isOverflowed ? "retained_audio_budget_exceeded" : "no_recoverable_audio",
          this.retainedPcm.isOverflowed
            ? "Streaming transcription ended without a complete bounded recovery copy; retry the turn."
            : "Streaming transcription ended without recoverable audio.",
        );
        return;
      }
      this.finalReceived = true;
      const retained = encodeWavFromPcm16(pcm, TARGET_SAMPLE_RATE);
      const primary = this.diagnostic.read();
      if (this.transport.retainedRoutePolicy !== "exact-model"
        || primary.providerTier !== "local"
        || !primary.modelId) {
        throw new Error("retained_route_unavailable");
      }
      const route = {
        providerTier: "local" as const,
        providerId: primary.providerId,
        modelId: primary.modelId,
      };
      const recovered = this.transport.transcribeRetainedWithIdentity
        ? await this.transport.transcribeRetainedWithIdentity(retained, this.language, route)
        : { text: await this.transport.transcribeRetained(retained, this.language, route) };
      if (generation !== this.captureGeneration || this.terminalFailure) return;
      // The exact selected model remains the authority for this turn's route.
      // Keep the streaming identity as the provider-cell diagnostic even if
      // the unary adapter reports a different internal implementation name.
      const primaryIdentity = this.diagnostic.read();
      if (!primaryIdentity.providerId && !primaryIdentity.modelId) {
        this.diagnostic.providerIdentity(recovered.providerIdentity?.providerId, recovered.providerIdentity?.modelId);
      }
      const transcript = recovered.text;
      const provider = recovered.providerIdentity?.providerId;
      const model = recovered.providerIdentity?.modelId;
      const identity = provider || model ? ` with ${provider ?? "unknown provider"}${model ? ` (${model})` : ""}` : "";
      this.status("buffered_recovery_completed", `Buffered recovery completed${identity}.`);
      // Bounded HTTP recovery is itself the terminal text commit for routes
      // that never emit a streaming final. Record the same stop-to-result lag
      // as the native terminal path before publishing the terminal state.
      if (this.diagnostic.read().committedTextLagMs === null) {
        this.diagnostic.committed(Number(this.sample));
        this.publishDiagnostic();
      }
      this.onResult?.(transcript.trim());
      this.diagnostic.terminal("completed", "http_fallback");
      this.publishDiagnostic();
      this.discardJournalAfterTerminal();
      forgetUnfinishedSession();
    } catch (err) {
      if (generation !== this.captureGeneration) return;
      const code = err instanceof Error && err.message === "retained_route_unavailable"
        ? "retained_route_unavailable"
        : "http_fallback_failed";
      this.diagnostic.error(code);
      this.diagnostic.terminal("failed", code);
      this.publishDiagnostic();
      this.onError?.(code === "retained_route_unavailable"
        ? "Streaming stopped; retained audio could not be sent because the exact local recovery route is unavailable."
        : "Streaming transcription and retained-audio recovery failed.");
    }
  }

  async start(prewarmed?: MediaStream): Promise<void> {
    this.captureGeneration += 1;
    const startGeneration = this.captureGeneration;
    this.journalReady = false;
    if (this.preconnectTimer) clearTimeout(this.preconnectTimer);
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
    if (this.finalPendingTimer) clearTimeout(this.finalPendingTimer);
    if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
	let warmedConnection = this.preconnectWarmed ? this.connection : null;
	this.preconnectWarmed = false;
	const preconnectResumedSession = this.preconnectResumedSession;
	this.preconnectResumedSession = false;
	let preconnected = warmedConnection !== null
      && this.sessionId !== ""
      && this.resumeToken !== "";
	if (!preconnected && this.connection) {
		// Only a stream opened explicitly by preConnect belongs to this turn.
		// Fence a prior turn's queued callbacks before replacing its connection.
		const previous = this.connection;
		this.connection = null;
		this.connectionReady = false;
		previous.close();
	}
	this.stopped = false; this.finalReceived = false; this.terminalFailure = false; this.doneSent = false; this.reconnectBackoff.reset(); this.lastRecoveryProgressSequence = -1n; this.micRecoveryAttempted = false; this.pending = []; this.replaying = false; this.batchSamples = []; this.batchSampleCount = 0; this.batchStartSample = 0n; this.journalAcks = Promise.resolve(); this.writes = Promise.resolve(); this.queuedBytes = 0; this.pendingWriteCount = 0; this.retainedPcm.clear();
	this.pcmResampler.reset();
	if (!preconnected) {
		this.sessionId = "";
		this.resumeToken = "";
		this.sequence = 0n;
		this.sample = 0n;
	}
    const recovered = loadUnfinishedSession();
    const requiresRecoveredJournal = preconnected ? preconnectResumedSession : recovered !== null;
    if (!preconnected) {
		if (recovered) { this.sessionId = recovered.sessionId; this.resumeToken = recovered.resumeToken; }
		else { this.sessionId = newSessionIdentity(); this.resumeToken = newSessionIdentity(); rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken }); }
	} else if (!preconnectResumedSession) {
		// A permission-warmed fresh identity is provisional until recording is
		// requested. Persist it now, but never overwrite an unfinished session
		// during preConnect before its journal has been restored.
		rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken });
	}
    // Publish the session before touching browser microphone APIs. A denied,
    // busy, or disappeared device must leave a machine-readable terminal
    // diagnostic; otherwise hosts can remain in "preparing" with no evidence
    // explaining why recording never reached the wire.
    this.diagnostic.reset(this.sessionId, 0, "reduced");
    this.captureReadyReported = false;
    if (this.startRequestedAtMs == null) {
      this.startRequestedAtMs = typeof globalThis.performance?.now === "function" ? globalThis.performance.now() : Date.now();
    }
    this.diagnostic.startRequested(this.startRequestedAtMs);
    this.diagnostic.status("capture_starting");
    this.publishDiagnostic();
    const persistent = this.options.journalFactory?.() ?? new TurnJournal(new IndexedDBTurnJournalStore(), this.sessionId, 0n, this.maxRetainedTurnAudioBytes, "persistent");
    // Restore can involve several IndexedDB transactions and replay records.
    // Start it alongside microphone acquisition so an already-authorized mic
    // does not pay journal latency serially. The later readiness gate still
    // prevents any replay or new PCM from reaching the server early.
    const restoreResult = persistent.restore().then(
      (snapshot) => ({ snapshot, error: null as unknown }),
      (error: unknown) => ({ snapshot: undefined, error }),
    );
    let lease: MicLease;
    try {
      lease = prewarmed
        ? registerMicStream("voice-stream", prewarmed)
        : await this.acquireStream();
    } catch (error) {
      if (this.stopped || startGeneration !== this.captureGeneration) return;
      this.diagnostic.error("capture_start_failed");
      this.diagnostic.terminal("failed", "capture_start_failed");
      this.publishDiagnostic();
      throw error;
    }
    if (this.stopped || startGeneration !== this.captureGeneration) {
      releaseMicLease(lease, "manual-stop");
      return;
    }
    this.lease = lease;
    this.stream = this.lease.stream; this.bindTracks(this.stream);
    this.tailDropArmed = false;
    this.allStartedAt = Date.now();
    // Keep the exported turn diagnostic aligned with the recorder lifecycle.
    // Hosts use this state to distinguish an acknowledged, fully drained turn
    // from a stream that is still preparing; omitting it leaves processed
    // readiness hidden even when every wire batch has been acknowledged.
    this.diagnostic.state("recording");
    this.diagnostic.captureStarted();
    this.publishDiagnostic();
    // Open admission while storage recovery is in flight. `journalReady`
    // fences onReady/activation; no replay or new audio can be sent until the
    // recovered sequence and sample cursors have been installed below.
    if (!(preconnected && warmedConnection)) this.connect(this.sequence > 0n);
    const restored = await restoreResult;
    if (this.stopped || startGeneration !== this.captureGeneration) return;
    const missingRecoveredState = !restored.snapshot || (
      !persistent.hasRestoredDurableState() && !persistent.wasExpiredOnRestore()
    );
    if (requiresRecoveredJournal && (restored.error || missingRecoveredState)) {
		// The server may already have accepted intervals under this identity.
		// Without its durable client cursor, sequence zero is not a safe resume
		// point. Keep the opaque identity for a later retry and refuse to start a
		// reduced-memory turn that could conflict with the server ledger.
		const message = "The previous voice session's recovery journal is unavailable. No new audio was captured; check browser storage and retry.";
		this.stopped = true;
		this.terminalFailure = true;
		this.journal = null;
		this.capture?.stop();
		this.capture = null;
		this.releaseOwnStream("provider-error");
		const connection = this.connection;
		this.connection = null;
		this.connectionReady = false;
		connection?.close();
		if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
		this.diagnostic.error("journal_restore_failed");
		this.diagnostic.terminal("failed", "journal_restore_failed");
		this.publishDiagnostic();
		this.status("journal_restore_failed", message);
		throw new Error(message);
	}
    if (!restored.error && restored.snapshot) {
      const snapshot = restored.snapshot;
      if (persistent.wasExpiredOnRestore()) {
        const staleConnection = this.connection;
        this.connection = null;
        this.connectionReady = false;
        staleConnection?.close();
        warmedConnection = null;
        preconnected = false;
        this.preconnectWarmed = false;
        forgetUnfinishedSession();
        this.sessionId = newSessionIdentity();
        this.resumeToken = newSessionIdentity();
        rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken });
        persistent.rebindAfterExpiry(this.sessionId, 0n);
        this.diagnostic.reset(this.sessionId, 0, "reduced");
        this.diagnostic.startRequested(this.startRequestedAtMs ?? undefined);
        this.diagnostic.status("capture_starting");
        this.publishDiagnostic();
        this.status("retention_expired", "Previous recovery audio expired and was deleted; starting a fresh voice session.");
      }
      this.journal = persistent;
      this.sequence = snapshot.nextSequence;
      this.sample = snapshot.nextSample;
      this.batchStartSample = snapshot.nextSample;
    } else {
      this.journal = new TurnJournal(new MemoryTurnJournalStore(), this.sessionId, 0n, 16 * 1024 * 1024, "reduced");
      this.status("durability_reduced", "Persistent audio recovery is unavailable in this browser.");
    }
    if (this.stopped || startGeneration !== this.captureGeneration) return;
    this.journalReady = true;
    // Open the transport before starting capture production. Real microphone
    // factories are event-driven, while accelerated qualification sources may
    // emit a large virtual turn immediately; connecting first lets the server
    // acknowledge and compact durable journal frames while capture continues.
    // Reuse a stream opened by preConnect(). Opening another stream here adds
    // a new stream setup to the user's first turn and can also leave
    // the preconnected stream orphaned. A preconnected stream already uses
    // the restored session identity, so only connect when there is no usable
    // transport (or when the preconnect was closed before capture began).
    if (preconnected && warmedConnection) {
      // The warmed stream retains its callbacks, which read the current
      // capture generation and fence by active connection identity.
      this.activationGeneration = -1;
      if (this.connectionReady) this.activateConnection(warmedConnection, this.sequence > 0n);
    } else if (!this.connection) {
      this.connect(this.sequence > 0n);
    } else if (this.connectionReady) {
      this.activateConnection(this.connection, this.sequence > 0n);
    }
    let capture: PcmCapture;
    try {
      capture = await this.makeCapture(this.stream);
    } catch (error) {
      if (!this.stopped && startGeneration === this.captureGeneration) {
        // Permission can succeed while AudioWorklet and its compatibility
        // graph both fail. Treat that as a failed startup and release every
        // resource acquired by this turn; otherwise the rejected start leaves
        // the microphone leased and its transport open with no capture graph.
        this.stopped = true;
        this.terminalFailure = true;
        this.unbindTracks();
        this.releaseOwnStream("provider-error");
        const connection = this.connection;
        this.connection = null;
        this.connectionReady = false;
        connection?.close();
        this.diagnostic.error("capture_start_failed");
        this.diagnostic.terminal("failed", "capture_start_failed");
        this.publishDiagnostic();
        this.onError?.("Audio capture could not be started.");
      }
      throw error;
    }
    // Capture factories may wait for AudioWorklet module loading. A stop (or a
    // newer start) can happen during that wait, before `this.capture` exists;
    // do not install a graph after its owning turn has already ended.
    if (this.stopped || startGeneration !== this.captureGeneration) {
      capture.stop();
      return;
    }
    this.capture = capture;
    // A transport can fail before an asynchronous capture graph is installed.
    // Its close handler then has no capture to reconnect; retry here once the
    // capture owner exists so the turn cannot remain silently disconnected.
    if (!this.connection && !this.terminalFailure) this.connect(this.sequence > 0n);
    this.markCaptureReady();
  }

  stop(): void {
    this.stopped = true;
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
	this.capture?.stop();
	this.capture = null;
	this.flushBatch();
	const stoppedAt = Date.now();
	const fullTurnRecovery = this.retainedPcm.snapshot();
	this.lastTurn = this.tailDropArmed || !fullTurnRecovery
		? null
	  : { blob: encodeWavFromPcm16(fullTurnRecovery, TARGET_SAMPLE_RATE), mimeType: "audio/wav", durationMs: stoppedAt - this.allStartedAt, capturedAt: stoppedAt };
    // PCM frames are appended through `writes`; sending `done` synchronously
    // here can overtake the final frame when the encoder callback and stop
    // button land in the same turn. Drain the write chain first so the server
    // can account for every captured chunk before it evaluates coverage.
    const generation = this.captureGeneration;
    if (this.connection?.isOpen()) {
      const connection = this.connection;
      void this.writes.then(() => {
        if (generation !== this.captureGeneration || this.terminalFailure || this.connection !== connection || !connection.isOpen() || this.doneSent) return;
        connection.end();
        this.doneSent = true;
        this.diagnostic.done();
        this.publishDiagnostic();
      if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
      this.fallbackTimer = setTimeout(() => { void this.fallback(); }, fallbackDelayMs(this.sample, this.pendingWriteCount));
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
    this.captureGeneration += 1;
    if (this.serverAckTimer) clearTimeout(this.serverAckTimer);
    if (this.finalPendingTimer) clearTimeout(this.finalPendingTimer);
    if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
    this.unbindTracks();
    this.connection?.close();
    this.connection = null;
    this.connectionReady = false;
    if (this.batchTimer) clearTimeout(this.batchTimer);
    this.batchTimer = null;
	this.journal = null; this.pending = []; this.retainedPcm.clear(); this.lastTurn = null;
  }
}
