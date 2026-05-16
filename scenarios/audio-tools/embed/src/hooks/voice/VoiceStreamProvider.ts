// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// VoiceStreamProvider — WebSocket streaming transcription provider (preferred).
// Streams audio chunks to the Go backend over WebSocket for real-time partial
// transcription via Whisper. Falls back to HTTP batch transcription if the
// WebSocket connection fails after all reconnect attempts are exhausted.
//
// Audio capture starts immediately on mic acquisition (not on WS open) to
// eliminate the audio gap between getUserMedia and WebSocket connection.
// Chunks are buffered in pendingChunks until the WebSocket is ready.

import { transcribeAudioWithRetry, buildVoiceStreamWsUrl } from "../../api/voice";
import type { LastTurnAudio, TranscriptionProvider } from "./types";
import { AUDIO_BITRATE, STREAM_CHUNK_INTERVAL_MS, WHISPER_FAILED_SENTINEL, computeFinalTimeout } from "./types";

export class VoiceStreamProvider implements TranscriptionProvider {
  private ws: WebSocket | null = null;
  private mediaRecorder: MediaRecorder | null = null;
  private stream: MediaStream | null = null;
  private finalReceived = false;
  private finalTimeout: ReturnType<typeof setTimeout> | null = null;
  private wsUrl = "";
  private reconnectAttempt = 0;
  private intentionallyStopped = false;
  private recordingStartTime = 0;
  /** Timestamp when stop() was called -- used to measure stop-to-final latency. */
  private stopTime = 0;
  private pendingChunks: ArrayBuffer[] = [];
  /**
   * All audio chunks collected for the current turn (every segment, accepted
   * or rejected). Used for two purposes:
   *   1. HTTP fallback if WebSocket streaming fails entirely.
   *   2. Full-turn retention for the "Transcribe anyway" retry flow when
   *      speaker verification rejects a segment — see `lastTurn` below.
   * Cleared on each new `start()`.
   */
  private allChunks: Blob[] = [];
  /**
   * Retained audio from the most recent completed turn. Snapshotted in
   * `stop()` or `dispose()` so the hook can offer a bypass-filter retry
   * after rejection. Released by `disposeLastTurn()` or the next `start()`.
   */
  private lastTurn: LastTurnAudio | null = null;
  /** Mime type of retained audio (webm/opus or webm, set at MediaRecorder init). */
  private lastTurnMimeType = "audio/webm";
  /** Running count of audio chunks sent via WebSocket. */
  private chunkCount = 0;
  /** Running total of bytes sent via WebSocket. */
  private totalBytesSent = 0;
  private static readonly MAX_RECONNECTS = 2;
  private static readonly RECONNECT_DELAYS = [1_000, 3_000];
  /** Timeout for pre-connected WebSocket — closed if start() isn't called. */
  private static readonly PRE_CONNECT_TIMEOUT_MS = 30_000;
  private firstPartialLogged = false;
  private preConnectTimer: ReturnType<typeof setTimeout> | null = null;
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
  onPartial: ((text: string) => void) | null = null;
  /** Fired when a segment-final transcription arrives from the backend. */
  onSegmentFinal: ((text: string, segmentIndex: number) => void) | null = null;
  onSegmentAccepted: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSegmentRejected: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSpeakerStatus: ((enabled: boolean, profileConfigured: boolean) => void) | null = null;

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
   * Build a retained `LastTurnAudio` from the currently-collected chunks.
   * Called at turn end (stop or dispose) so the hook can offer a retry
   * without requiring the user to re-record.
   */
  private snapshotLastTurn(): void {
    if (this.allChunks.length === 0) {
      this.lastTurn = null;
      return;
    }
    const blob = new Blob(this.allChunks, { type: this.lastTurnMimeType });
    const durationMs = Math.max(0, Date.now() - this.recordingStartTime);
    this.lastTurn = {
      blob,
      mimeType: this.lastTurnMimeType,
      durationMs,
      capturedAt: Date.now(),
    };
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

  private setupWsHandlers(ws: WebSocket): void {
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string) as {
          type: string;
          text?: string;
          segmentIndex?: number;
          score?: number;
          threshold?: number;
          enabled?: boolean;
          profileConfigured?: boolean;
        };
        if (msg.type === "segment-final" && msg.text !== undefined) {
          this.onSegmentFinal?.(msg.text, msg.segmentIndex ?? 0);
        } else if (msg.type === "segment-accepted") {
          this.onSegmentAccepted?.(msg.segmentIndex ?? 0, msg.score ?? 0, msg.threshold ?? 0);
        } else if (msg.type === "segment-rejected") {
          this.onSegmentRejected?.(msg.segmentIndex ?? 0, msg.score ?? 0, msg.threshold ?? 0);
        } else if (msg.type === "speaker-status") {
          this.onSpeakerStatus?.(Boolean(msg.enabled), Boolean(msg.profileConfigured));
        } else if (msg.type === "partial" && msg.text) {
          if (!this.firstPartialLogged) {
            const latency = Date.now() - this.recordingStartTime;
            console.info("[voice] First partial received, latency=%dms, text=%s",
              latency, msg.text.length > 60 ? msg.text.slice(0, 60) + "\u2026" : msg.text);
            this.firstPartialLogged = true;
          }
          this.onPartial?.(msg.text);
        } else if (msg.type === "final") {
          this.finalReceived = true;
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
          this.onError?.(msg.text ?? "Streaming transcription failed");
        }
      } catch {
        // Ignore malformed messages
      }
    };

    ws.onerror = () => {
      console.warn("[voice] WebSocket error \u2014 will attempt HTTP fallback on close");
      // Don't emit error here; onclose fires after onerror and will handle fallback.
    };

    ws.onclose = () => {
      console.info("[voice] WebSocket closed, finalReceived:", this.finalReceived);
      if (this.finalReceived || this.intentionallyStopped) return;

      // Attempt reconnect with exponential backoff if recording is still active.
      if (this.reconnectAttempt < VoiceStreamProvider.MAX_RECONNECTS
          && this.mediaRecorder?.state === "recording") {
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
            // Flush chunks buffered during reconnection
            for (const chunk of this.pendingChunks) {
              if (newWs.readyState === WebSocket.OPEN) newWs.send(chunk);
            }
            this.pendingChunks = [];
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
   * Fall back to HTTP transcription using all collected audio chunks.
   * Called when WebSocket streaming fails and all reconnects are exhausted.
   */
  private attemptHttpFallback(): void {
    if (this.allChunks.length === 0) {
      this.onError?.(WHISPER_FAILED_SENTINEL);
      return;
    }
    console.warn("[voice] Streaming failed \u2014 falling back to HTTP transcription");
    const blob = new Blob(this.allChunks, { type: "audio/webm" });
    this.allChunks = [];
    const httpFallbackStart = Date.now();
    transcribeAudioWithRetry(blob, 2, this.language)
      .then((text) => {
        console.info("[voice] HTTP fallback transcription took %dms, %d chars", Date.now() - httpFallbackStart, text.trim().length);
        if (text.trim()) {
          this.finalReceived = true;
          this.onResult?.(text.trim());
        }
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
      if (this.ws && !this.mediaRecorder) {
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
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
    }
    this.mediaRecorder = null;

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
    this.reconnectAttempt = 0;
    this.intentionallyStopped = false;
    this.recordingStartTime = Date.now();
    this.stopTime = 0;
    this.pendingChunks = [];
    this.allChunks = [];
    this.chunkCount = 0;
    this.totalBytesSent = 0;
    this.lastTurn = null;

    // Start MediaRecorder IMMEDIATELY after mic acquisition.
    // Chunks are buffered in pendingChunks until the WebSocket connects.
    // This eliminates the audio gap between getUserMedia and WS open.
    this.lastTurnMimeType = MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
      ? "audio/webm;codecs=opus"
      : "audio/webm";
    this.mediaRecorder = new MediaRecorder(this.stream, {
      mimeType: this.lastTurnMimeType,
      audioBitsPerSecond: AUDIO_BITRATE,
    });
    this.mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) {
        this.chunkCount++;
        this.totalBytesSent += e.data.size;
        // Keep a copy for HTTP fallback in case streaming fails entirely.
        this.allChunks.push(e.data);
        e.data.arrayBuffer().then((buf) => {
          // Capture a local reference: the turn may have ended (and the WS
          // reassigned) between when this microtask was queued and when it
          // runs. Reading this.ws twice across the check + send would race.
          const ws = this.ws;
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(buf);
          } else {
            // Buffer until WebSocket connects (or during reconnection)
            this.pendingChunks.push(buf);
          }
        });
      }
    };
    this.mediaRecorder.start(STREAM_CHUNK_INTERVAL_MS);

    // ── WebSocket connection ──
    // Reuse a pre-connected WebSocket if available (from preConnect()).
    // Otherwise open a new one. Either way, install recording-session handlers.
    if (hasPreConnectedWs && this.ws) {
      console.info("[voice] Reusing pre-connected WebSocket");
      // Flush any buffered chunks from the pre-connect phase
      for (const chunk of this.pendingChunks) {
        if (this.ws.readyState === WebSocket.OPEN) this.ws.send(chunk);
      }
      this.pendingChunks = [];
      // Replace the lightweight pre-connect handlers with full recording handlers
      this.setupWsHandlers(this.ws);
    } else {
      const wsConnStart = Date.now();
      this.ws = new WebSocket(this.wsUrl);
      this.ws.onopen = () => {
        console.info("[voice] WebSocket connected in %dms, flushing %d buffered chunks", Date.now() - wsConnStart, this.pendingChunks.length);
        // Flush chunks that were buffered before the WebSocket connected
        for (const chunk of this.pendingChunks) {
          if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(chunk);
        }
        this.pendingChunks = [];
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

    if (this.mediaRecorder?.state === "recording") {
      // Defer "done" signal until after final data is flushed.
      this.mediaRecorder.onstop = () => {
        if (this.ws?.readyState === WebSocket.OPEN) {
          this.ws.send(JSON.stringify({ type: "done" }));
        }
        // Snapshot retained audio AFTER the final ondataavailable fires so the
        // retained blob includes the last tail of the turn.
        this.snapshotLastTurn();
        // When retainStream is true (low-latency mode), keep the stream alive
        // for re-use in subsequent recordings. The mic readiness module manages
        // the stream lifecycle instead.
        // DOC: docs/internal/VOICE-LATENCY.md#audio-ducking-deep-dive
        if (!this.retainStream) {
          this.stream?.getTracks().forEach((t) => t.stop());
          this.stream = null;
        }
      };
      this.mediaRecorder.stop();
    } else {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "done" }));
      }
      this.snapshotLastTurn();
      if (!this.retainStream) {
        this.stream?.getTracks().forEach((t) => t.stop());
        this.stream = null;
      }
    }

    if (!this.finalReceived) {
      const elapsed = Date.now() - this.recordingStartTime;
      const timeout = computeFinalTimeout(elapsed);
      this.finalTimeout = setTimeout(() => {
        if (!this.finalReceived) {
          this.ws?.close();
          this.onError?.("Streaming transcription timed out");
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
    if (this.preConnectTimer) {
      clearTimeout(this.preConnectTimer);
      this.preConnectTimer = null;
    }
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
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
