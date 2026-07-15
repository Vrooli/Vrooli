// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// VoiceStreamProvider — WebSocket streaming transcription provider (preferred).
// Captures microphone audio as raw 16 kHz mono signed-16-bit PCM and streams
// it to the Go backend over WebSocket for real-time partial transcription via
// Whisper. Declaring `format=pcm_s16le` takes the server's ffmpeg-free
// fast-path (the audioformat substrate's identity decoder), so browser
// sessions never spin up a server-side transcoder. Falls back to HTTP batch
// transcription if the WebSocket fails after all reconnect attempts.
//
// Audio capture starts immediately on mic acquisition (not on WS open) to
// eliminate the gap between getUserMedia and WebSocket connection. PCM frames
// captured before the socket opens are buffered in pendingChunks and flushed
// on open.

import { transcribeAudioWithRetry, buildVoiceStreamWsUrl } from "../../api/voice";
import type { LastTurnAudio, TranscriptionProvider } from "./types";
import { WHISPER_FAILED_SENTINEL, computeFinalTimeout } from "./types";
import { createCanonicalPcmCapture, type AsyncPcmCaptureFactory, type PcmCapture } from "./pcmCapture";
import { concatInt16, encodeWavFromPcm16, frameToCanonicalPcm16, TARGET_SAMPLE_RATE } from "./pcm";
import { digestAudio, dispatchStreamMessage, encodeAudioFrame, forgetUnfinishedSession, IndexedDBTurnJournalStore, loadUnfinishedSession, MemoryTurnJournalStore, newSessionIdentity, rememberUnfinishedSession, StreamDiagnosticRecorder, TurnJournal, type JournalSnapshot, type StreamTurnDiagnostic } from "@vrooli/audio-capture-browser";

export class VoiceStreamProvider implements TranscriptionProvider {
  private ws: WebSocket | null = null;
  /** Live PCM capture, or null when not recording. */
  private capture: PcmCapture | null = null;
  /**
   * PCM capture factory seam. Production wires the ScriptProcessor-based
   * capture; tests inject a fake that lets them push synthetic frames
   * without a real AudioContext.
   */
  captureFactory: AsyncPcmCaptureFactory = createCanonicalPcmCapture;
  private stream: MediaStream | null = null;
  private finalReceived = false;
  private finalTimeout: ReturnType<typeof setTimeout> | null = null;
  private wsUrl = "";
  private reconnectAttempt = 0;
  private intentionallyStopped = false;
  private doneSent = false;
  private recordingStartTime = 0;
  /** Timestamp when stop() was called -- used to measure stop-to-final latency. */
  private stopTime = 0;
  private pendingChunks: Array<{ frame: ArrayBuffer; sequence: bigint }> = [];
  private sessionId = "";
  private resumeToken = "";
	// Durable server segment identities survive WebSocket reconnects. Keep this
	// turn-local set so replay fills an interrupted delivery without appending
	// text the host already received before the close.
  private deliveredSegmentIDs = new Set<string>();
  private nextSequence = 0n;
  private nextSample = 0n;
  private journal: TurnJournal | null = null;
  private journalWrites: Promise<void> = Promise.resolve();
  private resumingRecoveredTurn = false;
  /** Metadata-only support record; never contains transcript text or PCM. */
  private diagnostic = new StreamDiagnosticRecorder();
  /**
   * All canonical PCM captured this turn (every frame, accepted or rejected).
   * Used for two purposes:
   *   1. HTTP fallback (wrapped as WAV) if WebSocket streaming fails entirely.
   *   2. Full-turn retention for the "Transcribe anyway" retry flow when
   *      speaker verification rejects a segment — see `lastTurn` below.
   * Cleared on each new `start()`.
   */
  private allPcm: Int16Array[] = [];
  private retainedPcmBytes = 0;
  private retentionExhausted = false;
  /**
   * Retained audio from the most recent completed turn, snapshotted in
   * `stop()` or `dispose()` so the hook can offer a bypass-filter retry
   * after rejection. Released by `disposeLastTurn()` or the next `start()`.
   */
  private lastTurn: LastTurnAudio | null = null;
  /** Container type of retained audio. PCM capture is wrapped as WAV. */
  private static readonly RETAINED_MIME_TYPE = "audio/wav";
  /** The fallback/retry buffer has the same hard bound as the durable journal. */
  private static readonly MAX_RETAINED_PCM_BYTES = 16 * 1024 * 1024;
  /** Running count of PCM frames sent via WebSocket. */
  private chunkCount = 0;
  /** Running total of bytes sent via WebSocket. */
  private totalBytesSent = 0;
  /**
   * When true, the frame handler drops incoming PCM instead of forwarding
   * to the WS, and `stop()` sends `{ type: "done" }` synchronously and skips
   * tail retention. Armed by `dropTail()` from the auto-stop path; reset on
   * each `start()`.
   */
  private tailDropArmed = false;
  private droppedChunkCount = 0;
  private static readonly MAX_RECONNECTS = 2;
  private static readonly RECONNECT_DELAYS = [1_000, 3_000];
  /** Timeout for pre-connected WebSocket — closed if start() isn't called. */
  private static readonly PRE_CONNECT_TIMEOUT_MS = 30_000;
  private firstPartialLogged = false;
  private preConnectTimer: ReturnType<typeof setTimeout> | null = null;
  private serverAckTimer: ReturnType<typeof setTimeout> | null = null;
  private finalProgressTimer: ReturnType<typeof setTimeout> | null = null;
  private serverAckReceived = false;
	// A terminal error and a final frame can arrive in the same durable drain.
	// A final only means the recognizer stopped; it must not compact the browser
	// journal unless the server has explicitly confirmed complete coverage.
  private terminalFailure = false;
  /** True when the current WS was opened via preConnect() and hasn't been
   *  consumed by start() yet. Prevents start() from closing a pre-connected WS. */
  private isPreConnectedWs = false;
  language = "en";
  /** When true, stop() does not call track.stop() — the stream is retained
   *  for re-use by the mic readiness module (low-latency voice mode).
   *  DOC: docs/internal/VOICE-LATENCY.md#audio-ducking-deep-dive */
  retainStream = false;
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onStatus: ((status: { code: string; message: string }) => void) | null = null;
  onDiagnostic: ((diagnostic: StreamTurnDiagnostic) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;
  /** Fired when a segment-final transcription arrives from the backend. */
  onSegmentFinal: ((text: string, segmentIndex: number) => void) | null = null;
  onSegmentAccepted: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSegmentRejected: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSpeakerStatus: ((enabled: boolean, profileConfigured: boolean) => void) | null = null;
  /**
   * Fired on every server-emitted `vad-state` message. The host wires this
   * to `useServerVadStateStore.set()` so the mic-button ring can render
   * server-driven silence progress. See plan
   * /home/matthalloran8/.vrooli/plans/server-driven-mic-ring-streamvadstate-event.md.
   */
  onVadState: ((snapshot: {
    voiced: boolean;
    silenceElapsedMs: number;
    silenceTimeoutMs: number;
    tickSeq: number;
    silenceTimedOut: boolean;
  }) => void) | null = null;

  getStream(): MediaStream | null {
    return this.stream;
  }

  getDiagnostic(): StreamTurnDiagnostic {
    return this.diagnostic.read();
  }

  exportDiagnostic(): string {
    return this.diagnostic.exportJSON();
  }

  private publishDiagnostic(): void {
    this.onDiagnostic?.(this.diagnostic.read());
  }

  private emitStatus(code: string, message: string): void {
    this.diagnostic.status(code);
    this.publishDiagnostic();
    this.onStatus?.({ code, message });
  }

  private emitError(code: string, message: string): void {
    this.diagnostic.error(code);
    this.publishDiagnostic();
    this.onError?.(message);
  }

  getLastTurnAudio(): LastTurnAudio | null {
    return this.lastTurn;
  }

  disposeLastTurn(): void {
    this.lastTurn = null;
  }

  /**
   * Arm in-flight audio drop. Subsequent captured PCM frames are discarded
   * instead of being sent over the WS, and `stop()` will commit the
   * server-side segment immediately without retaining a tail blob. Called by
   * the auto-stop path so post-verdict audio (anything spoken during
   * teardown) does not leak into the transcription.
   */
  dropTail(): void {
    this.tailDropArmed = true;
    console.info("[voice] tail-drop armed");
  }

  /**
   * Handle one captured PCM frame: downsample to the canonical 16 kHz rate,
   * convert to signed-16-bit PCM, and forward it to the WebSocket (or buffer
   * it until the socket opens). Frames are dropped while tail-drop is armed.
   * Retains the canonical PCM for the HTTP fallback / retry path.
   */
  private handleFrame(samples: Float32Array, sampleRate: number): void {
    if (this.tailDropArmed) {
      this.droppedChunkCount++;
      return;
    }
    const pcm16 = frameToCanonicalPcm16(samples, sampleRate);
    if (pcm16.length === 0) return;
    this.chunkCount++;
    this.totalBytesSent += pcm16.byteLength;
    // Keep a copy for HTTP fallback / retry in case streaming fails entirely.
    this.allPcm.push(pcm16);
    this.queueCapturedFrame(pcm16);
  }

  private queueCapturedFrame(pcm16: Int16Array): void {
    if (this.retentionExhausted || this.tailDropArmed) return;
    if (this.retainedPcmBytes + pcm16.byteLength > VoiceStreamProvider.MAX_RETAINED_PCM_BYTES) {
      this.retentionExhausted = true;
      this.emitStatus("recovery_quota_exhausted", "Audio recovery storage reached its limit; this turn was stopped before further audio could be lost.");
      this.emitError("recovery_quota_exhausted", "Audio recovery storage reached its limit. Start a new turn to continue.");
      return;
    }
    const startSample = this.nextSample;
    const endSample = startSample + BigInt(pcm16.length);
    const sequence = this.nextSequence;
    this.nextSequence += 1n;
    this.nextSample = endSample;
    this.retainedPcmBytes += pcm16.byteLength;
    this.diagnostic.captured(sequence);
    this.publishDiagnostic();
    // Copy into a plain ArrayBuffer for IndexedDB. Typed-array backing stores
    // are ArrayBufferLike in TypeScript and can be SharedArrayBuffers.
    const pcmBytes = new Uint8Array(pcm16.byteLength);
    pcmBytes.set(new Uint8Array(pcm16.buffer, pcm16.byteOffset, pcm16.byteLength));
    const pcm = pcmBytes.buffer;
    this.journalWrites = this.journalWrites
      .then(async () => {
        const digest = await digestAudio(pcm);
        const frame = encodeAudioFrame({ sequence, startSample, endSample, audio: pcm16, sha256: new Uint8Array(digest) });
        await this.journal?.append({ sequence, startSample, endSample, audio: pcm, sha256: digest });
        return frame;
      })
      .then((frame) => {
        if (this.ws?.readyState === WebSocket.OPEN && this.pendingChunks.length === 0) {
          this.ws.send(frame);
          this.diagnostic.sent(sequence);
          this.publishDiagnostic();
        } else this.pendingChunks.push({ frame, sequence });
      })
      .catch(() => {
        // Do not release a chunk with no stable identity. The full turn remains
        // in allPcm for explicit unary recovery, and the user receives a clear
        // durability failure instead of a silent at-most-once stream.
        this.emitError("journal_write_failed", "Secure audio recovery could not record this chunk; stop and retry the turn.");
      });
  }

  /**
   * Serialize a server processed cursor after every prior append. This is
   * important when an acknowledgement races the final IndexedDB write: no
   * captured chunk may be compacted before it has first become durable.
   */
  private acknowledgeProcessed(sequence: bigint): void {
    this.diagnostic.processed(sequence);
    this.publishDiagnostic();
    this.journalWrites = this.journalWrites
      .then(async () => this.journal?.acknowledgeProcessed(sequence) ?? undefined)
      .catch(() => {
        this.emitStatus("durability_reduced", "Unable to compact local audio recovery state.");
      });
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
      this.emitStatus("durability_reduced", "Persistent audio recovery is unavailable in this browser.");
      return reduced.read();
    }
  }

  private createNewSessionIdentity(): boolean {
    try {
      this.sessionId = newSessionIdentity();
      this.resumeToken = newSessionIdentity();
		this.deliveredSegmentIDs.clear();
      rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken });
      return true;
    } catch {
      this.onError?.("Secure browser identity generation is unavailable; dictation recovery cannot start.");
      return false;
    }
  }

  /** Send a segment-boundary signal to the backend, triggering a high-quality
   *  retranscription of the current speech segment. */
  sendSegmentBoundary(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "segment-boundary" }));
    }
  }

  /** Notify the backend of VAD speech state changes so it can skip
   *  partial transcription during silence (prevents Whisper hallucinations). */
  sendVadState(speaking: boolean): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      const msgType = speaking ? "vad-speech-start" : "vad-speech-end";
      this.ws.send(JSON.stringify({ type: msgType }));
      const elapsed = Date.now() - this.recordingStartTime;
      console.debug("[voice] VAD %s sent at +%dms, chunks=%d, bytes=%d",
        msgType, elapsed, this.chunkCount, this.totalBytesSent);
    }
  }

  /**
   * Build a retained `LastTurnAudio` (WAV) from the captured PCM. Called at
   * turn end (stop or dispose) so the hook can offer a retry without
   * requiring the user to re-record.
   */
  private snapshotLastTurn(): void {
    if (this.allPcm.length === 0) {
      this.lastTurn = null;
      return;
    }
    const blob = encodeWavFromPcm16(concatInt16(this.allPcm), TARGET_SAMPLE_RATE);
    const durationMs = Math.max(0, Date.now() - this.recordingStartTime);
    this.lastTurn = {
      blob,
      mimeType: VoiceStreamProvider.RETAINED_MIME_TYPE,
      durationMs,
      capturedAt: Date.now(),
    };
  }

  private flushPendingChunks(ws: WebSocket): void {
    for (const chunk of this.pendingChunks) {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(chunk.frame);
        this.diagnostic.sent(chunk.sequence);
      }
    }
    this.pendingChunks = [];
    this.publishDiagnostic();
  }

  /**
   * A socket may open while IndexedDB is still committing the early frames.
   * Wait for the serialized journal queue before releasing that backlog, then
   * emit `done` only after every queued frame has been released.
   */
  private flushJournaledChunksAndDone(ws: WebSocket): void {
    void this.journalWrites.then(() => {
      if (ws.readyState !== WebSocket.OPEN) return;
      this.flushPendingChunks(ws);
      this.sendDoneIfStopped(ws);
    });
  }

  /**
   * A reconnect is an at-least-once replay boundary. Recreate each retained
   * v2 frame from the journal rather than trusting a prior socket write; the
   * server ledger deduplicates identical sequence/range/digest triples. Clear
   * the transient queue because its entries are already represented by the
   * authoritative journal snapshot.
   */
  private replayJournaledChunksAndDone(ws: WebSocket): void {
    void this.journalWrites.then(() => {
      if (ws.readyState !== WebSocket.OPEN) return;
      const replay = this.journal?.replayAfter(-1n) ?? [];
      if (replay.length === 0) {
        this.flushPendingChunks(ws);
        this.sendDoneIfStopped(ws);
        return;
      }
      this.pendingChunks = [];
      for (const chunk of replay) {
        ws.send(encodeAudioFrame({
          sequence: chunk.sequence,
          startSample: chunk.startSample,
          endSample: chunk.endSample,
          audio: new Uint8Array(chunk.audio),
          sha256: new Uint8Array(chunk.sha256),
        }));
        this.diagnostic.sent(chunk.sequence);
      }
      this.publishDiagnostic();
      this.sendDoneIfStopped(ws);
    });
  }

  private sendDoneIfStopped(ws: WebSocket): void {
    if (!this.intentionallyStopped || this.doneSent || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({ type: "done" }));
    this.doneSent = true;
    this.diagnostic.done();
    this.publishDiagnostic();
  }

  private clearServerAckTimer(): void {
    if (this.serverAckTimer) {
      clearTimeout(this.serverAckTimer);
      this.serverAckTimer = null;
    }
  }

  private clearFinalProgressTimer(): void {
    if (this.finalProgressTimer) {
      clearTimeout(this.finalProgressTimer);
      this.finalProgressTimer = null;
    }
  }

  private markServerProgress(): void {
    this.serverAckReceived = true;
    this.clearServerAckTimer();
  }

  private armServerAckWatchdog(): void {
    this.clearServerAckTimer();
    this.serverAckTimer = setTimeout(() => {
      if (!this.serverAckReceived && !this.finalReceived) {
        this.emitStatus("server_ack_pending", "Waiting for the speech backend to acknowledge the stream.");
      }
    }, 1500);
  }

  private armFinalProgressWatchdog(sentBytes: number): void {
    this.clearFinalProgressTimer();
    this.finalProgressTimer = setTimeout(() => {
      if (!this.finalReceived && this.intentionallyStopped && sentBytes > 0) {
        this.emitStatus("final_pending", "Speech audio was sent; waiting for the backend to finish transcription.");
      }
    }, 3000);
  }

  private setupWsHandlers(ws: WebSocket): void {
    ws.onmessage = (event) => dispatchStreamMessage(event.data, {
      onStatus: (code, text, processedSequence) => {
        this.markServerProgress();
        const coverage = code === "processed_acknowledgement" ? ` Processed through chunk ${processedSequence ?? -1n}.` : "";
        if (code === "processed_acknowledgement" && processedSequence !== undefined) this.acknowledgeProcessed(processedSequence);
        this.emitStatus(code, `${text}${coverage}`);
      },
      onPartial: (text) => {
        this.markServerProgress();
          if (!this.firstPartialLogged) {
            const latency = Date.now() - this.recordingStartTime;
            console.info("[voice] First partial received, latency=%dms, text=%s",
              latency, text.length > 60 ? text.slice(0, 60) + "…" : text);
            this.firstPartialLogged = true;
          }
        this.onPartial?.(text);
      },
      onSegmentFinal: (text, index) => { this.markServerProgress(); this.onSegmentFinal?.(text, index); },
      onSegmentAccepted: (index, score, threshold) => { this.markServerProgress(); this.onSegmentAccepted?.(index, score, threshold); },
      onSegmentRejected: (index, score, threshold) => { this.markServerProgress(); this.onSegmentRejected?.(index, score, threshold); },
      onVadState: (state) => { this.markServerProgress(); this.onVadState?.(state); },
      onSpeakerStatus: (enabled, profileConfigured) => { this.markServerProgress(); this.onSpeakerStatus?.(enabled, profileConfigured); },
      onFinal: (text) => {
		  if (this.terminalFailure) {
			  this.finalReceived = true;
			  this.clearFinalProgressTimer();
			  if (this.nextSequence > 0n) {
				  this.diagnostic.terminal("failed", "incomplete_coverage");
				  this.emitStatus("recovery_retained", "The backend did not confirm all captured audio. Retained audio is available for recovery.");
			  } else {
				  this.journalWrites = this.journalWrites.then(async () => this.journal?.discard() ?? undefined);
				  forgetUnfinishedSession();
			  }
			  this.publishDiagnostic();
			  return;
		  }
          this.markServerProgress();
          this.finalReceived = true;
          this.clearFinalProgressTimer();
          if (this.finalTimeout) {
            clearTimeout(this.finalTimeout);
            this.finalTimeout = null;
          }
          const stopToFinal = this.stopTime > 0 ? Date.now() - this.stopTime : 0;
          const totalDuration = Date.now() - this.recordingStartTime;
          console.info("[voice] Final received: %d chars, stopToFinal=%dms, total=%dms, chunks=%d, bytes=%d",
            text.length, stopToFinal, totalDuration, this.chunkCount, this.totalBytesSent);
          // Always call onResult — even with empty text — so the UI resets
          // from "transcribing" back to "idle". Empty finals happen when
          // speaker verification rejects the audio.
          this.onResult?.(text);
          this.diagnostic.terminal("completed", "final");
          this.publishDiagnostic();
          // The server emitted processed coverage before terminal final. Delete
          // local recovery material only after its serialized compaction work.
          this.journalWrites = this.journalWrites.then(async () => this.journal?.discard() ?? undefined);
          forgetUnfinishedSession();
      },
	  onError: (code, text) => {
		  this.terminalFailure = true;
		  this.markServerProgress();
		  this.emitError(code, text);
	  },
    }, this.deliveredSegmentIDs);

    ws.onerror = () => {
      console.warn("[voice] WebSocket error — will attempt HTTP fallback on close");
      // Don't emit error here; onclose fires after onerror and will handle fallback.
    };

    ws.onclose = () => {
      console.info("[voice] WebSocket closed, finalReceived:", this.finalReceived);
      this.clearServerAckTimer();
      if (this.finalReceived || this.intentionallyStopped) return;

      // Attempt reconnect with exponential backoff if capture is still active.
      if (this.reconnectAttempt < VoiceStreamProvider.MAX_RECONNECTS && this.capture) {
        const delay = VoiceStreamProvider.RECONNECT_DELAYS[this.reconnectAttempt] ?? 3_000;
        this.reconnectAttempt++;
        this.diagnostic.state("reconnecting", "socket_closed");
        this.publishDiagnostic();
        console.warn("[voice] WebSocket reconnect attempt %d/%d, delay=%dms, pendingChunks=%d",
          this.reconnectAttempt, VoiceStreamProvider.MAX_RECONNECTS, delay, this.pendingChunks.length);

        setTimeout(() => {
          const newWs = new WebSocket(this.wsUrl);
          const connTimeout = setTimeout(() => {
            newWs.close();
            // onclose will fire again -- next attempt or final failure
          }, delay + 2_000);

          newWs.onopen = () => {
            clearTimeout(connTimeout);
            this.ws = newWs;
            this.setupWsHandlers(newWs);
            this.armServerAckWatchdog();
            // Flush chunks buffered during reconnection
            this.replayJournaledChunksAndDone(newWs);
          };
          newWs.onerror = () => {
            clearTimeout(connTimeout);
          };
        }, delay);
        return;
      }

      // All reconnects exhausted -- fall back to HTTP transcription
      this.attemptHttpFallback();
    };
  }

  /**
   * Fall back to HTTP transcription using all captured PCM, wrapped as WAV.
   * Called when WebSocket streaming fails and all reconnects are exhausted.
   */
  private attemptHttpFallback(): void {
    if (this.allPcm.length === 0) {
      this.diagnostic.terminal("failed", "no_recoverable_audio");
      this.publishDiagnostic();
      this.emitError("no_recoverable_audio", WHISPER_FAILED_SENTINEL);
      return;
    }
    console.warn("[voice] Streaming failed — falling back to HTTP transcription");
    const blob = encodeWavFromPcm16(concatInt16(this.allPcm), TARGET_SAMPLE_RATE);
    this.allPcm = [];
    this.retainedPcmBytes = 0;
    this.retentionExhausted = false;
    const httpFallbackStart = Date.now();
    transcribeAudioWithRetry(blob, 2, this.language)
      .then((text) => {
        const transcript = text.trim();
        console.info("[voice] HTTP fallback transcription took %dms, %d chars", Date.now() - httpFallbackStart, transcript.length);
        this.finalReceived = true;
        this.onResult?.(transcript);
        this.diagnostic.terminal("completed", "http_recovery");
        this.publishDiagnostic();
      })
      .catch(() => {
        console.warn("[voice] HTTP fallback failed after %dms", Date.now() - httpFallbackStart);
        this.diagnostic.terminal("failed", "http_recovery_failed");
        this.publishDiagnostic();
        this.emitError("http_recovery_failed", WHISPER_FAILED_SENTINEL);
      });
  }

  /**
   * Pre-connect the WebSocket so it's already open when start() is called,
   * eliminating 10-100ms of connection latency from the recording start path.
   *
   * The pre-connected WebSocket has a timeout — if start() is not called within
   * PRE_CONNECT_TIMEOUT_MS, the connection is closed to free server resources.
   *
   * No-op if a WebSocket is already open or connecting.
   *
   * DOC: docs/internal/VOICE-LATENCY.md#websocket-pre-connection
   */
  preConnect(language: string): void {
    // Don't pre-connect if we already have an open/connecting WebSocket
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    this.language = language;
    if (!this.createNewSessionIdentity()) return;
    this.wsUrl = buildVoiceStreamWsUrl(this.language, this.sessionId, this.resumeToken);

    const wsConnStart = Date.now();
    this.ws = new WebSocket(this.wsUrl);
    this.isPreConnectedWs = true;
    this.ws.onopen = () => {
      console.info("[voice] WebSocket pre-connected in %dms", Date.now() - wsConnStart);
    };
    this.ws.onerror = () => {
      console.warn("[voice] WebSocket pre-connect failed, will connect on demand");
      this.ws = null;
    };
    this.ws.onclose = () => {
      // Only clear if this is still the pre-connected WS (not replaced by start())
      if (this.ws?.readyState === WebSocket.CLOSED) {
        this.ws = null;
      }
    };

    // Set a timeout to close the pre-connected WS if start() isn't called.
    // This prevents holding an idle connection on the server indefinitely.
    if (this.preConnectTimer) clearTimeout(this.preConnectTimer);
    this.preConnectTimer = setTimeout(() => {
      if (this.ws && !this.capture) {
        // No recording started — close the idle pre-connection
        this.ws.close();
        this.ws = null;
        forgetUnfinishedSession();
      }
      this.preConnectTimer = null;
    }, VoiceStreamProvider.PRE_CONNECT_TIMEOUT_MS);
  }

  // DOC: docs/internal/VOICE-LATENCY.md#stream-injection-vs-stream-acquisition
  async start(preWarmedStream?: MediaStream): Promise<void> {
    // Cancel pre-connect timeout since we're starting for real now
    if (this.preConnectTimer) {
      clearTimeout(this.preConnectTimer);
      this.preConnectTimer = null;
    }

    // Clean up stale state from previous recording session, but preserve
    // a pre-connected WebSocket (from preConnect()) for reuse.
    const hasPreConnectedWs = this.isPreConnectedWs && this.ws &&
      (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING);
    this.isPreConnectedWs = false; // Consumed — next start() won't reuse
    if (this.ws && !hasPreConnectedWs) {
      this.ws.onclose = null; // prevent reconnect/fallback logic from firing
      this.ws.close();
      this.ws = null;
    }
    if (this.capture) {
      this.capture.stop();
      this.capture = null;
    }

    // ── Stream acquisition ──
    // DOC: docs/internal/VOICE-LATENCY.md#stream-injection-vs-stream-acquisition
    //
    // Accept an optional pre-warmed stream (from micReadiness.ts when low-latency
    // voice mode is enabled). Only call getUserMedia if no pre-warmed stream is
    // available or if its tracks have ended (browser revoked access).
    if (preWarmedStream && preWarmedStream.getTracks().every((t) => t.readyState === "live")) {
      this.stream = preWarmedStream;
      console.info("[voice] Low-latency: injecting pre-warmed stream into VoiceStreamProvider");
    } else {
      if (preWarmedStream) {
        console.warn("[voice] Low-latency: pre-warmed stream tracks ended, provider will acquire fresh");
      }
      const micStart = Date.now();
      try {
        this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      } catch {
        this.onError?.("Microphone access denied");
        return;
      }
      console.info("[voice] getUserMedia took %dms", Date.now() - micStart);
    }

    // Reset session state. The new turn starts now; the previous turn's
    // retained audio is replaced (single-slot retention).
    if (!hasPreConnectedWs) {
      const recovered = loadUnfinishedSession();
      if (recovered) {
        this.sessionId = recovered.sessionId;
        this.resumeToken = recovered.resumeToken;
        this.diagnostic.state("recovering", "recovery_resuming");
        this.emitStatus("recovery_resuming", "Resuming retained audio from an interrupted turn.");
      } else {
        if (!this.createNewSessionIdentity()) {
          this.stream?.getTracks().forEach((track) => track.stop());
          this.stream = null;
          return;
        }
      }
    }
    this.wsUrl = buildVoiceStreamWsUrl(this.language, this.sessionId, this.resumeToken);
    this.finalReceived = false;
    this.firstPartialLogged = false;
    this.serverAckReceived = false;
	this.terminalFailure = false;
    this.clearServerAckTimer();
    this.clearFinalProgressTimer();
    this.reconnectAttempt = 0;
    this.intentionallyStopped = false;
    this.doneSent = false;
    this.recordingStartTime = Date.now();
    this.stopTime = 0;
    this.pendingChunks = [];
    this.nextSequence = 0n;
    this.nextSample = 0n;
    this.journalWrites = Promise.resolve();
    const journalSnapshot = await this.initializeJournal();
    this.nextSequence = journalSnapshot.nextSequence;
    this.nextSample = journalSnapshot.nextSample;
    this.resumingRecoveredTurn = journalSnapshot.chunks.length > 0;
    this.diagnostic.reset(this.sessionId, 0, journalSnapshot.durability);
    this.diagnostic.state(this.resumingRecoveredTurn ? "recovering" : "recording");
    this.publishDiagnostic();
    this.allPcm = [];
    this.chunkCount = 0;
    this.totalBytesSent = 0;
    this.lastTurn = null;
    this.tailDropArmed = false;
    this.droppedChunkCount = 0;

    // Start PCM capture IMMEDIATELY after mic acquisition. Frames are
    // buffered in pendingChunks until the WebSocket connects. This
    // eliminates the audio gap between getUserMedia and WS open.
    this.capture = await this.captureFactory(this.stream, (samples, rate) => this.handleFrame(samples, rate));

    // ── WebSocket connection ──
    // Reuse a pre-connected WebSocket if available (from preConnect()).
    // Otherwise open a new one. Either way, install recording-session handlers.
    if (hasPreConnectedWs && this.ws) {
      console.info("[voice] Reusing pre-connected WebSocket");
      // Replace the lightweight pre-connect handlers with full recording handlers
      this.setupWsHandlers(this.ws);
      this.ws.onopen = () => {
        console.info("[voice] Pre-connected WebSocket opened during recording, flushing %d buffered chunks", this.pendingChunks.length);
        this.armServerAckWatchdog();
        if (this.ws) {
          this.flushJournaledChunksAndDone(this.ws);
        }
      };
      if (this.ws.readyState === WebSocket.OPEN) {
        this.armServerAckWatchdog();
        if (this.resumingRecoveredTurn) this.replayJournaledChunksAndDone(this.ws);
        else this.flushJournaledChunksAndDone(this.ws);
      }
    } else {
      const wsConnStart = Date.now();
      this.ws = new WebSocket(this.wsUrl);
      this.ws.onopen = () => {
        console.info("[voice] WebSocket connected in %dms, flushing %d buffered chunks", Date.now() - wsConnStart, this.pendingChunks.length);
        this.armServerAckWatchdog();
        // Flush chunks that were buffered before the WebSocket connected,
        // after their journal commits complete.
        if (this.ws) {
          if (this.resumingRecoveredTurn) this.replayJournaledChunksAndDone(this.ws);
          else this.flushJournaledChunksAndDone(this.ws);
        }
      };
      this.setupWsHandlers(this.ws);
    }
  }

  stop(): void {
    this.intentionallyStopped = true;
    this.stopTime = Date.now();
    this.diagnostic.state("recording", "stop_requested");
    this.publishDiagnostic();
    const recordingDuration = this.stopTime - this.recordingStartTime;
    console.info("[voice] Stop: recordingDuration=%dms, chunks=%d, bytes=%d",
      recordingDuration, this.chunkCount, this.totalBytesSent);

    // Stop PCM capture. Unlike MediaRecorder there is no buffered encoder
    // tail to flush — ScriptProcessor delivers frames synchronously up to
    // disconnect — so end-of-utterance can be signalled immediately.
    if (this.capture) {
      this.capture.stop();
      this.capture = null;
    }
    if (!this.retainStream) {
      this.stream?.getTracks().forEach((t) => t.stop());
      this.stream = null;
    }

    // The terminal control frame is deliberately queued after journal writes:
    // the server must never receive `done` before the audio it covers has been
    // durably recorded and released to the socket.
    this.journalWrites = this.journalWrites.then(() => {
      if (this.ws?.readyState === WebSocket.OPEN) this.sendDoneIfStopped(this.ws);
    });

    // Tail-drop path: the server already declared end-of-utterance, so skip
    // tail retention. Otherwise retain the turn's audio for a possible
    // bypass-filter retry.
    if (this.tailDropArmed) {
      if (this.droppedChunkCount > 0) {
        console.info("[voice] tail-drop dropped %d frames", this.droppedChunkCount);
      }
    } else {
      this.snapshotLastTurn();
    }

    if (!this.finalReceived) {
      const elapsed = Date.now() - this.recordingStartTime;
      const timeout = computeFinalTimeout(elapsed);
      const sentBytes = this.totalBytesSent;
      this.armFinalProgressWatchdog(sentBytes);
      this.finalTimeout = setTimeout(() => {
        if (!this.finalReceived) {
          this.clearFinalProgressTimer();
          this.ws?.close();
          // Classify the timeout honestly instead of a single ambiguous
          // string. By this point the mic was granted and recording ran, so
          // this is a backend no-final, not a permission problem (permission
          // denial surfaces "Microphone access denied" earlier). Distinguish
          // "audio streamed but no transcript came back" (likely the local
          // provider declining streaming) from "no audio reached the server".
          if (sentBytes > 0) {
            console.warn("[voice] Streaming backend produced no final after %dms; falling back to HTTP transcription", timeout);
            this.attemptHttpFallback();
          } else {
            this.diagnostic.terminal("failed", "no_audio_reached_server");
            this.publishDiagnostic();
            this.emitError("no_audio_reached_server",
              "Streaming transcription timed out — no audio reached the server (check the microphone and the /api/v1/voice/stream connection).",
            );
          }
        }
      }, timeout);
    }
  }

  dispose(): void {
    this.intentionallyStopped = true;
    if (!this.finalReceived) {
      this.diagnostic.terminal("cancelled", "client_disposed");
      this.publishDiagnostic();
    }
    if (this.finalTimeout) {
      clearTimeout(this.finalTimeout);
      this.finalTimeout = null;
    }
    this.clearServerAckTimer();
    this.clearFinalProgressTimer();
    if (this.preConnectTimer) {
      clearTimeout(this.preConnectTimer);
      this.preConnectTimer = null;
    }
    if (this.capture) {
      this.capture.stop();
      this.capture = null;
    }
    this.ws?.close();
    this.ws = null;
    // Always stop tracks on dispose, regardless of retainStream —
    // dispose is a full cleanup, not a recording-end event.
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
    this.pendingChunks = [];
    // Dispose is a full cleanup; drop retained audio too. The hook calls
    // this when reclaiming the provider, not when ending a turn.
    this.lastTurn = null;
  }
}
