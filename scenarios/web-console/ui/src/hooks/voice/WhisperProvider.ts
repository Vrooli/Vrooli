// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// WhisperProvider — HTTP batch transcription provider.
// Collects all audio locally via MediaRecorder, sends a single POST on stop.
// Used when the backend supports Whisper but not the streaming WebSocket endpoint.
//
// Retention: after each turn stops, the combined audio Blob is moved from the
// transient `chunks` buffer into `lastTurn` so the hook can offer a
// "Transcribe anyway" retry on speaker-verification rejection. The blob is
// released on `disposeLastTurn()` or on the next `start()`.

import { transcribeAudioWithRetry } from "../../lib/api";
import type { LastTurnAudio, TranscriptionProvider } from "./types";
import { AUDIO_BITRATE, WHISPER_FAILED_SENTINEL } from "./types";

export class WhisperProvider implements TranscriptionProvider {
  private mediaRecorder: MediaRecorder | null = null;
  private chunks: Blob[] = [];
  private stream: MediaStream | null = null;
  /** Retained audio from the most recent completed turn, or null. */
  private lastTurn: LastTurnAudio | null = null;
  language = "en";
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;

  getStream(): MediaStream | null {
    return this.stream;
  }

  getLastTurnAudio(): LastTurnAudio | null {
    return this.lastTurn;
  }

  disposeLastTurn(): void {
    this.lastTurn = null;
  }

  private micAcquireTime = 0;
  private recordingStartTime = 0;

  // DOC: docs/internal/VOICE-LATENCY.md#stream-injection-vs-stream-acquisition
  async start(preWarmedStream?: MediaStream): Promise<void> {
    // A new turn starts — the previous turn's retained audio is no longer
    // relevant and would grow unbounded if we didn't drop it here.
    this.lastTurn = null;

    // Accept a pre-warmed stream (low-latency mode) or acquire a fresh one.
    if (preWarmedStream && preWarmedStream.getTracks().every((t) => t.readyState === "live")) {
      this.stream = preWarmedStream;
      this.micAcquireTime = 0;
      console.info("[voice] WhisperHTTP: using pre-warmed stream");
    } else {
      const micStart = Date.now();
      try {
        this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      } catch {
        this.onError?.("Microphone access denied");
        return;
      }
      this.micAcquireTime = Date.now() - micStart;
      console.info("[voice] WhisperHTTP: getUserMedia took %dms", this.micAcquireTime);
    }
    this.chunks = [];
    this.recordingStartTime = Date.now();
    const mimeType = MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
      ? "audio/webm;codecs=opus"
      : "audio/webm";
    this.mediaRecorder = new MediaRecorder(this.stream, {
      mimeType,
      audioBitsPerSecond: AUDIO_BITRATE,
    });
    this.mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) this.chunks.push(e.data);
    };
    this.mediaRecorder.onstop = async () => {
      const stopTime = Date.now();
      const recordingDuration = stopTime - this.recordingStartTime;
      this.stream?.getTracks().forEach((t) => t.stop());
      this.stream = null;

      const blob = new Blob(this.chunks, { type: "audio/webm" });
      this.chunks = [];
      if (blob.size === 0) {
        console.info("[voice] WhisperHTTP: empty recording (%dms), skipping transcription", recordingDuration);
        return;
      }
      // Retain the full-turn audio before we ship it upstream. The hook may
      // need it later for a bypass-filter retry if the server rejects the
      // transcript. Retention is released on disposeLastTurn() or next start().
      this.lastTurn = {
        blob,
        mimeType,
        durationMs: recordingDuration,
        capturedAt: stopTime,
      };
      console.info("[voice] WhisperHTTP: recording=%dms, audioSize=%d bytes, sending POST", recordingDuration, blob.size);
      const transcribeStart = Date.now();
      try {
        const text = await transcribeAudioWithRetry(blob, 2, this.language);
        console.info("[voice] WhisperHTTP: transcription took %dms, %d chars", Date.now() - transcribeStart, text.trim().length);
        if (text.trim()) this.onResult?.(text.trim());
      } catch {
        console.warn("[voice] WhisperHTTP: transcription failed after %dms", Date.now() - transcribeStart);
        this.onError?.(WHISPER_FAILED_SENTINEL);
      }
    };
    this.mediaRecorder.start();
  }

  stop(): void {
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
      // Mic release happens in onstop after final data is flushed.
    } else {
      this.stream?.getTracks().forEach((t) => t.stop());
      this.stream = null;
    }
  }

  dispose(): void {
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
    }
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
    // Dispose is a full cleanup event; drop retained audio too.
    this.lastTurn = null;
  }
}
