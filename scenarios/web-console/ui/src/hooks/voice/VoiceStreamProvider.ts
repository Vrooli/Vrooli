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

import { transcribeAudioWithRetry, buildVoiceStreamWsUrl } from "../../lib/api";
import type { TranscriptionProvider } from "./types";
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
  /** All audio chunks collected for HTTP fallback if streaming fails entirely. */
  private allChunks: Blob[] = [];
  /** Running count of audio chunks sent via WebSocket. */
  private chunkCount = 0;
  /** Running total of bytes sent via WebSocket. */
  private totalBytesSent = 0;
  private static readonly MAX_RECONNECTS = 2;
  private static readonly RECONNECT_DELAYS = [1_000, 3_000];
  private firstPartialLogged = false;
  language = "en";
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;
  /** Fired when a segment-final transcription arrives from the backend. */
  onSegmentFinal: ((text: string, segmentIndex: number) => void) | null = null;

  getStream(): MediaStream | null {
    return this.stream;
  }

  /** Send a segment-boundary signal to the backend, triggering a high-quality
   *  retranscription of the current speech segment. */
  sendSegmentBoundary(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "segment-boundary" }));
    }
  }

  private setupWsHandlers(ws: WebSocket): void {
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string) as { type: string; text?: string; segmentIndex?: number };
        if (msg.type === "segment-final" && msg.text !== undefined) {
          this.onSegmentFinal?.(msg.text, msg.segmentIndex ?? 0);
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
          if (text) {
            this.onResult?.(text);
          }
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

  async start(): Promise<void> {
    // Clean up stale state from previous recording session
    if (this.ws) {
      this.ws.onclose = null; // prevent reconnect/fallback logic from firing
      this.ws.close();
      this.ws = null;
    }
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
    }
    this.mediaRecorder = null;

    const micStart = Date.now();
    try {
      this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      this.onError?.("Microphone access denied");
      return;
    }
    console.info("[voice] getUserMedia took %dms", Date.now() - micStart);

    // Reset session state
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

    // Start MediaRecorder IMMEDIATELY after mic acquisition.
    // Chunks are buffered in pendingChunks until the WebSocket connects.
    // This eliminates the audio gap between getUserMedia and WS open.
    this.mediaRecorder = new MediaRecorder(this.stream, {
      mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
        ? "audio/webm;codecs=opus"
        : "audio/webm",
      audioBitsPerSecond: AUDIO_BITRATE,
    });
    this.mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) {
        this.chunkCount++;
        this.totalBytesSent += e.data.size;
        // Keep a copy for HTTP fallback in case streaming fails entirely.
        this.allChunks.push(e.data);
        e.data.arrayBuffer().then((buf) => {
          if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(buf);
          } else {
            // Buffer until WebSocket connects (or during reconnection)
            this.pendingChunks.push(buf);
          }
        });
      }
    };
    this.mediaRecorder.start(STREAM_CHUNK_INTERVAL_MS);

    // Open WebSocket connection (audio is already being captured above)
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

  stop(): void {
    this.intentionallyStopped = true;
    this.stopTime = Date.now();
    const recordingDuration = this.stopTime - this.recordingStartTime;
    console.info("[voice] Stop: recordingDuration=%dms, chunks=%d, bytes=%d",
      recordingDuration, this.chunkCount, this.totalBytesSent);

    if (this.mediaRecorder?.state === "recording") {
      // Defer mic release and "done" signal until after final data is flushed.
      this.mediaRecorder.onstop = () => {
        if (this.ws?.readyState === WebSocket.OPEN) {
          this.ws.send(JSON.stringify({ type: "done" }));
        }
        this.stream?.getTracks().forEach((t) => t.stop());
        this.stream = null;
      };
      this.mediaRecorder.stop();
    } else {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "done" }));
      }
      this.stream?.getTracks().forEach((t) => t.stop());
      this.stream = null;
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
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
    }
    this.ws?.close();
    this.ws = null;
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
    this.pendingChunks = [];
  }
}
