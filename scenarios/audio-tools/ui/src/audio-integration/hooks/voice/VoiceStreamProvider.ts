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
import { createScriptProcessorPcmCapture, type PcmCapture, type PcmCaptureFactory } from "./pcmCapture";
import { concatInt16, encodeWavFromPcm16, frameToCanonicalPcm16, TARGET_SAMPLE_RATE } from "./pcm";

export class VoiceStreamProvider implements TranscriptionProvider {
  private ws: WebSocket | null = null;
  /** Live PCM capture, or null when not recording. */
  private capture: PcmCapture | null = null;
  /**
   * PCM capture factory seam. Production wires the ScriptProcessor-based
   * capture; tests inject a fake that lets them push synthetic frames
   * without a real AudioContext.
   */
  captureFactory: PcmCaptureFactory = createScriptProcessorPcmCapture;
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
  private pendingChunks: ArrayBufferView[] = [];
  /**
   * All canonical PCM captured this turn (every frame, accepted or rejected).
   * Used for two purposes:
   *   1. HTTP fallback (wrapped as WAV) if WebSocket streaming fails entirely.
   *   2. Full-turn retention for the "Transcribe anyway" retry flow when
   *      speaker verification rejects a segment — see `lastTurn` below.
   * Cleared on each new `start()`.
   */
  private allPcm: Int16Array[] = [];
  /**
   * Retained audio from the most recent completed turn, snapshotted in
   * `stop()` or `dispose()` so the hook can offer a bypass-filter retry
   * after rejection. Released by `disposeLastTurn()` or the next `start()`.
   */
  private lastTurn: LastTurnAudio | null = null;
  /** Container type of retained audio. PCM capture is wrapped as WAV. */
  private static readonly RETAINED_MIME_TYPE = "audio/wav";
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
    // The Int16Array is a fresh full-buffer allocation, so sending the view
    // sends exactly its s16le bytes as one binary WS frame.
    const ws = this.ws;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(pcm16);
    } else {
      // Buffer until WebSocket connects (or during reconnection).
      this.pendingChunks.push(pcm16);
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
      if (ws.readyState === WebSocket.OPEN) ws.send(chunk);
    }
    this.pendingChunks = [];
  }

  private sendDoneIfStopped(ws: WebSocket): void {
    if (!this.intentionallyStopped || this.doneSent || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({ type: "done" }));
    this.doneSent = true;
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
        this.onStatus?.({
          code: "server_ack_pending",
          message: "Waiting for the speech backend to acknowledge the stream.",
        });
      }
    }, 1500);
  }

  private armFinalProgressWatchdog(sentBytes: number): void {
    this.clearFinalProgressTimer();
    this.finalProgressTimer = setTimeout(() => {
      if (!this.finalReceived && this.intentionallyStopped && sentBytes > 0) {
        this.onStatus?.({
          code: "final_pending",
          message: "Speech audio was sent; waiting for the backend to finish transcription.",
        });
      }
    }, 3000);
  }

  private setupWsHandlers(ws: WebSocket): void {
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string) as {
          type: string;
          text?: string;
          // Error class on type==="error" (plan L2): "backend_starting" (the
          // speech backend is being auto-started — transient, retry shortly),
          // "backend_unavailable" (operator action needed), or "provider_failure".
          code?: string;
          segmentIndex?: number;
          score?: number;
          threshold?: number;
          enabled?: boolean;
          profileConfigured?: boolean;
          voiced?: boolean;
          silenceElapsedMs?: number;
          silenceTimeoutMs?: number;
          tickSeq?: number;
          silenceTimedOut?: boolean;
        };
        this.markServerProgress();
        if (msg.type === "status") {
          this.onStatus?.({
            code: msg.code ?? "stream_status",
            message: msg.text ?? "Streaming transcription status updated.",
          });
        } else if (msg.type === "segment-final" && msg.text !== undefined) {
          this.onSegmentFinal?.(msg.text, msg.segmentIndex ?? 0);
        } else if (msg.type === "segment-accepted") {
          this.onSegmentAccepted?.(msg.segmentIndex ?? 0, msg.score ?? 0, msg.threshold ?? 0);
        } else if (msg.type === "segment-rejected") {
          this.onSegmentRejected?.(msg.segmentIndex ?? 0, msg.score ?? 0, msg.threshold ?? 0);
        } else if (msg.type === "vad-state") {
          this.onVadState?.({
            voiced: Boolean(msg.voiced),
            silenceElapsedMs: msg.silenceElapsedMs ?? 0,
            silenceTimeoutMs: msg.silenceTimeoutMs ?? 0,
            tickSeq: msg.tickSeq ?? 0,
            silenceTimedOut: Boolean(msg.silenceTimedOut),
          });
        } else if (msg.type === "speaker-status") {
          this.onSpeakerStatus?.(Boolean(msg.enabled), Boolean(msg.profileConfigured));
        } else if (msg.type === "partial" && msg.text) {
          if (!this.firstPartialLogged) {
            const latency = Date.now() - this.recordingStartTime;
            console.info("[voice] First partial received, latency=%dms, text=%s",
              latency, msg.text.length > 60 ? msg.text.slice(0, 60) + "…" : msg.text);
            this.firstPartialLogged = true;
          }
          this.onPartial?.(msg.text);
        } else if (msg.type === "final") {
          this.finalReceived = true;
          this.clearFinalProgressTimer();
          if (this.finalTimeout) {
            clearTimeout(this.finalTimeout);
            this.finalTimeout = null;
          }
          const text = msg.text?.trim() ?? "";
          const stopToFinal = this.stopTime > 0 ? Date.now() - this.stopTime : 0;
          const totalDuration = Date.now() - this.recordingStartTime;
          console.info("[voice] Final received: %d chars, stopToFinal=%dms, total=%dms, chunks=%d, bytes=%d",
            text.length, stopToFinal, totalDuration, this.chunkCount, this.totalBytesSent);
          // Always call onResult — even with empty text — so the UI resets
          // from "transcribing" back to "idle". Empty finals happen when
          // speaker verification rejects the audio.
          this.onResult?.(text);
        } else if (msg.type === "error") {
          // The server now sends a clean, user-actionable message for a
          // backend-down failure (no raw `dial tcp …` transport string, plan
          // L2). A "backend_starting" code means on-demand recovery is bringing
          // the speech backend up — surface the server's "is starting…" message
          // verbatim so the user knows to retry shortly.
          this.onError?.(msg.text ?? "Streaming transcription failed");
        }
      } catch {
        // Ignore malformed messages
      }
    };

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
            this.flushPendingChunks(newWs);
            this.sendDoneIfStopped(newWs);
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
      this.onError?.(WHISPER_FAILED_SENTINEL);
      return;
    }
    console.warn("[voice] Streaming failed — falling back to HTTP transcription");
    const blob = encodeWavFromPcm16(concatInt16(this.allPcm), TARGET_SAMPLE_RATE);
    this.allPcm = [];
    const httpFallbackStart = Date.now();
    transcribeAudioWithRetry(blob, 2, this.language)
      .then((text) => {
        const transcript = text.trim();
        console.info("[voice] HTTP fallback transcription took %dms, %d chars", Date.now() - httpFallbackStart, transcript.length);
        this.finalReceived = true;
        this.onResult?.(transcript);
      })
      .catch(() => {
        console.warn("[voice] HTTP fallback failed after %dms", Date.now() - httpFallbackStart);
        this.onError?.(WHISPER_FAILED_SENTINEL);
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
    this.wsUrl = buildVoiceStreamWsUrl(this.language);

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
    this.wsUrl = buildVoiceStreamWsUrl(this.language);
    this.finalReceived = false;
    this.firstPartialLogged = false;
    this.serverAckReceived = false;
    this.clearServerAckTimer();
    this.clearFinalProgressTimer();
    this.reconnectAttempt = 0;
    this.intentionallyStopped = false;
    this.doneSent = false;
    this.recordingStartTime = Date.now();
    this.stopTime = 0;
    this.pendingChunks = [];
    this.allPcm = [];
    this.chunkCount = 0;
    this.totalBytesSent = 0;
    this.lastTurn = null;
    this.tailDropArmed = false;
    this.droppedChunkCount = 0;

    // Start PCM capture IMMEDIATELY after mic acquisition. Frames are
    // buffered in pendingChunks until the WebSocket connects. This
    // eliminates the audio gap between getUserMedia and WS open.
    this.capture = this.captureFactory(this.stream, (samples, rate) => this.handleFrame(samples, rate));

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
          this.flushPendingChunks(this.ws);
          this.sendDoneIfStopped(this.ws);
        }
      };
      if (this.ws.readyState === WebSocket.OPEN) {
        this.armServerAckWatchdog();
        this.flushPendingChunks(this.ws);
        this.sendDoneIfStopped(this.ws);
      }
    } else {
      const wsConnStart = Date.now();
      this.ws = new WebSocket(this.wsUrl);
      this.ws.onopen = () => {
        console.info("[voice] WebSocket connected in %dms, flushing %d buffered chunks", Date.now() - wsConnStart, this.pendingChunks.length);
        this.armServerAckWatchdog();
        // Flush chunks that were buffered before the WebSocket connected
        if (this.ws) {
          this.flushPendingChunks(this.ws);
          this.sendDoneIfStopped(this.ws);
        }
      };
      this.setupWsHandlers(this.ws);
    }
  }

  stop(): void {
    this.intentionallyStopped = true;
    this.stopTime = Date.now();
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

    if (this.ws?.readyState === WebSocket.OPEN) this.sendDoneIfStopped(this.ws);

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
            this.onError?.(
              "Streaming transcription timed out — no audio reached the server (check the microphone and the /api/v1/voice/stream connection).",
            );
          }
        }
      }, timeout);
    }
  }

  dispose(): void {
    this.intentionallyStopped = true;
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
