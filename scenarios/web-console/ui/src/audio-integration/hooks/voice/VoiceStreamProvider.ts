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
import { isStreamUsable } from "./streamHealth";
import { acquireMicStream, releaseMicLease, type MicLease, type MicReleaseReason } from "./micOwnership";
import type { LastTurnAudio, TranscriptionProvider } from "./types";
import { AUDIO_BITRATE, STREAM_CHUNK_INTERVAL_MS, WHISPER_FAILED_SENTINEL, classifyMicError, computeFinalTimeout } from "./types";

export class VoiceStreamProvider implements TranscriptionProvider {
  private ws: WebSocket | null = null;
  private mediaRecorder: MediaRecorder | null = null;
  private stream: MediaStream | null = null;
  /**
   * Registry lease for the mic stream this provider acquired. The provider
   * always owns its own stream, so this is non-null while capturing and its
   * release stops the tracks.
   * DOC: docs/internal/SEAMS.md#mic-ownership-seam
   */
  private lease: MicLease | null = null;
  private finalReceived = false;
  private finalTimeout: ReturnType<typeof setTimeout> | null = null;
  private wsUrl = "";
  private reconnectAttempt = 0;
  private intentionallyStopped = false;
  private recordingStartTime = 0;
  /** Timestamp when stop() was called -- used to measure stop-to-final latency. */
  private stopTime = 0;
  /**
   * Audio chunks awaiting send (WS not yet open, reconnecting, or ordered
   * behind an earlier chunk). Stored as Blobs and sent verbatim so recording
   * ORDER is preserved: the first MediaRecorder chunk carries the WebM/EBML
   * header, and the backend's codec sniff (and ffmpeg) require it to arrive
   * first. The previous `e.data.arrayBuffer().then(send)` path used independent
   * async promises that could resolve out of order (the header chunk is the
   * largest, so its arrayBuffer() could resolve AFTER the next chunk), putting a
   * headerless chunk on the wire first → `could not determine audio codec`.
   */
  private pendingChunks: Blob[] = [];
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
   * Running byte total of `allChunks`, used to bound retention. WebM/opus is a
   * container whose first chunk carries the header and whose clusters must stay
   * contiguous to decode, so retention is bounded as a decodable PREFIX: once
   * MAX_RETAINED_AUDIO_BYTES is reached we stop retaining further chunks (they
   * still stream live over the WS) rather than dropping the header/middle. The
   * HTTP fallback + last-turn retry therefore cover up to this bound from the
   * start of the turn — generous enough to never trip a normal session, but it
   * caps memory on a pathologically long or stuck one. Reset each `start()`.
   */
  private retainedBytes = 0;
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
  /**
   * When true, the `ondataavailable` handler drops incoming chunks instead
   * of forwarding to the WS, and `stop()` skips `snapshotLastTurn()` and
   * sends `{ type: "done" }` synchronously. Armed by `dropTail()` from the
   * auto-stop path; reset on each `start()`.
   */
  private tailDropArmed = false;
  private droppedChunkCount = 0;
  /**
   * Upper bound on retained turn audio (~66 min of 32 kbps opus). This bounds
   * memory on a stuck/very-long session while keeping a valid decodable prefix
   * for the HTTP fallback / retry. See `retainedBytes`.
   */
  private static readonly MAX_RETAINED_AUDIO_BYTES = 16 * 1024 * 1024;
  private static readonly MAX_RECONNECTS = 2;
  private static readonly RECONNECT_DELAYS = [1_000, 3_000];
  /** Timeout for pre-connected WebSocket — closed if start() isn't called. */
  private static readonly PRE_CONNECT_TIMEOUT_MS = 30_000;
  private firstPartialLogged = false;
  /** Count of unparseable WS frames this turn — surfaced in logs, never thrown. */
  private malformedMessageCount = 0;
  private preConnectTimer: ReturnType<typeof setTimeout> | null = null;
  private serverAckTimer: ReturnType<typeof setTimeout> | null = null;
  private finalProgressTimer: ReturnType<typeof setTimeout> | null = null;
  private serverAckReceived = false;
  private serverAckPendingNotified = false;
  private finalProgressPendingNotified = false;
  /** Wall-clock of the last partial/segment/final received — drives the
   *  transcription-stall watchdog so a mid-session stall (backend stops emitting
   *  while audio keeps streaming and the mic stays live) is VISIBLE in the log
   *  instead of a silent "it stopped transcribing after a while". */
  private lastTranscriptEventTime = 0;
  private transcriptStallTimer: ReturnType<typeof setTimeout> | null = null;
  private segmentCount = 0;
  /** How long without any transcription event (while still recording) before we
   *  flag a stall. Longer than a normal partial gap, shorter than a human pause
   *  that would auto-stop the turn. */
  private static readonly TRANSCRIPT_STALL_MS = 4_000;
  /** True when the current WS was opened via preConnect() and hasn't been
   *  consumed by start() yet. Prevents start() from closing a pre-connected WS. */
  private isPreConnectedWs = false;
  language = "en";
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

  /**
   * Release the mic stream this provider acquired (stops its tracks via the
   * registry lease). Idempotent — the lease is nulled after release.
   */
  private releaseOwnStream(reason: MicReleaseReason): void {
    if (this.lease) {
      releaseMicLease(this.lease, reason);
      this.lease = null;
    }
    this.stream = null;
  }

  /**
   * Wire mic-track + MediaRecorder lifecycle handlers so a mid-capture failure
   * of the audio source is OBSERVED and handled honestly instead of silently
   * starving the analyser (which drifts into a confusing VAD silence auto-stop —
   * "the mic just stopped" with no cause in the logs).
   *
   * Three distinct source-failure modes, each previously invisible:
   *  - `mute`  — the browser/OS mutes the track (sleep/wake, default-input-device
   *    change, or another app seizing the mic). The `ended` event does NOT fire
   *    for muting (see streamHealth.isTrackUsable), so no samples flow yet the
   *    track stays "live": the analyser reads pure silence and the client RMS VAD
   *    auto-stops mid-utterance. We log it loudly so a following auto-stop is
   *    attributable, and surface a soft notice. Mutes are often transient, so we
   *    do NOT end the turn here.
   *  - `ended` — the track is terminally gone. Recover the turn's retained audio
   *    via the HTTP fallback rather than losing it to a silent VAD stop.
   *  - MediaRecorder `error` — the encoder itself failed; surface it as an error.
   */
  private attachCaptureLifecycleHandlers(): void {
    const tracks = this.stream?.getAudioTracks?.() ?? [];
    for (const track of tracks) {
      if (typeof track.addEventListener !== "function") continue;
      track.addEventListener("mute", () => {
        console.warn(
          "[voice] mic track MUTED mid-capture at +%dms (sleep/wake, default-input-device change, or another app seized the mic). "
            + "The analyser now reads SILENCE — any VAD auto-stop after this is the CAUSE, not a real pause. chunks=%d",
          Date.now() - this.recordingStartTime, this.chunkCount);
        this.onStatus?.({
          code: "mic_muted",
          message: "Microphone muted by the system. If dictation stops, tap the mic again.",
        });
      });
      track.addEventListener("unmute", () => {
        console.info("[voice] mic track UNMUTED at +%dms — audio resumed", Date.now() - this.recordingStartTime);
      });
      track.addEventListener("ended", () => {
        const dur = Date.now() - this.recordingStartTime;
        console.warn(
          "[voice] mic track ENDED mid-capture at +%dms (OS/hardware/another-app revoked the mic). "
            + "intentionallyStopped=%s finalReceived=%s chunks=%d — recovering retained audio via HTTP.",
          dur, this.intentionallyStopped, this.finalReceived, this.chunkCount);
        if (this.intentionallyStopped || this.finalReceived) return;
        // The source is terminally gone; don't drift into a silent VAD stop that
        // discards the tail. Stop reconnect attempts and recover what we captured.
        this.intentionallyStopped = true;
        this.stopTime = Date.now();
        this.clearServerAckTimer();
        if (this.finalTimeout) { clearTimeout(this.finalTimeout); this.finalTimeout = null; }
        this.onStatus?.({
          code: "mic_track_ended",
          message: "The microphone stopped unexpectedly — recovering your transcript.",
        });
        this.ws?.close();
        this.attemptHttpFallback();
      });
    }
  }

  getLastTurnAudio(): LastTurnAudio | null {
    return this.lastTurn;
  }

  disposeLastTurn(): void {
    this.lastTurn = null;
  }

  /**
   * Arm in-flight audio drop. Subsequent `ondataavailable` blobs are
   * discarded instead of being sent over the WS, and `stop()` will commit
   * the server-side segment immediately without retaining a tail blob.
   * Called by the auto-stop path so post-verdict audio (the encoder's
   * 250 ms chunk in progress + anything spoken during teardown) does not
   * leak into the transcription.
   */
  dropTail(): void {
    this.tailDropArmed = true;
    console.info("[voice] tail-drop armed");
  }

  /**
   * Send one audio chunk in strict recording order. Sends immediately only when
   * the socket is open AND nothing is already queued ahead of it; otherwise it
   * queues so it can never jump ahead of the header-bearing first chunk. Sending
   * the Blob directly (no async arrayBuffer()) keeps ordering deterministic.
   */
  private enqueueOutbound(blob: Blob): void {
    const ws = this.ws;
    if (ws && ws.readyState === WebSocket.OPEN && this.pendingChunks.length === 0) {
      ws.send(blob);
    } else {
      this.pendingChunks.push(blob);
    }
  }

  /** Flush queued chunks to the socket in order once it is open. */
  private flushPending(): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    for (const chunk of this.pendingChunks) this.ws.send(chunk);
    this.pendingChunks = [];
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
    const shouldClearPendingNotice = this.serverAckPendingNotified;
    this.serverAckReceived = true;
    this.serverAckPendingNotified = false;
    this.clearServerAckTimer();
    if (shouldClearPendingNotice) {
      this.onStatus?.({
        code: "stream_connected",
        message: "Speech backend acknowledged the stream.",
      });
    }
  }

  private armServerAckWatchdog(): void {
    this.clearServerAckTimer();
    this.serverAckTimer = setTimeout(() => {
      if (!this.serverAckReceived && !this.finalReceived) {
        this.serverAckPendingNotified = true;
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
        this.finalProgressPendingNotified = true;
        this.onStatus?.({
          code: "final_pending",
          message: "Speech audio was sent; waiting for the backend to finish transcription.",
        });
      }
    }, 3000);
  }

  private clearTranscriptStallTimer(): void {
    if (this.transcriptStallTimer) {
      clearTimeout(this.transcriptStallTimer);
      this.transcriptStallTimer = null;
    }
  }

  /** Record a transcription event and (re)arm the stall watchdog. */
  private markTranscriptEvent(): void {
    this.lastTranscriptEventTime = Date.now();
    this.armTranscriptStallWatchdog();
  }

  /** Warn (and keep warning) if no transcription event arrives for
   *  TRANSCRIPT_STALL_MS while audio is still streaming — the "mic stays live
   *  but transcription stopped" wedge symptom, made observable. */
  private armTranscriptStallWatchdog(): void {
    this.clearTranscriptStallTimer();
    this.transcriptStallTimer = setTimeout(() => {
      if (this.finalReceived || this.intentionallyStopped) return;
      if (this.mediaRecorder?.state !== "recording") return;
      const gap = Date.now() - this.lastTranscriptEventTime;
      console.warn(
        `[voice] ⚠ transcription STALL: no partial/segment for ${gap}ms while still recording `
        + `(chunks=${this.chunkCount}, bytes=${this.totalBytesSent}, segments=${this.segmentCount}) `
        + `— backend stopped emitting; audio is still streaming`);
      // Re-arm so the stall is reported until it recovers or the turn ends.
      this.armTranscriptStallWatchdog();
    }, VoiceStreamProvider.TRANSCRIPT_STALL_MS);
  }

  private markFinalProgressComplete(): void {
    const shouldClearPendingNotice = this.finalProgressPendingNotified;
    this.finalProgressPendingNotified = false;
    this.clearFinalProgressTimer();
    if (shouldClearPendingNotice) {
      this.onStatus?.({
        code: "transcription_complete",
        message: "Speech transcription completed.",
      });
    }
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
          this.segmentCount++;
          const preview = msg.text.length > 60 ? msg.text.slice(0, 60) + "…" : msg.text;
          console.info("[voice] segment-final #%d at +%dms (gap=%dms): \"%s\"",
            msg.segmentIndex ?? 0, Date.now() - this.recordingStartTime,
            this.lastTranscriptEventTime > 0 ? Date.now() - this.lastTranscriptEventTime : 0, preview);
          this.markTranscriptEvent();
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
              latency, msg.text.length > 60 ? msg.text.slice(0, 60) + "\u2026" : msg.text);
            this.firstPartialLogged = true;
          }
          this.markTranscriptEvent();
          this.onPartial?.(msg.text);
        } else if (msg.type === "final") {
          if (this.mediaRecorder?.state === "recording" && !this.intentionallyStopped) {
            console.warn(
              "[voice] Final arrived while MediaRecorder is still recording at +%dms (chars=%d). "
                + "Treating as backend loss and routing through reconnect/fallback.",
              Date.now() - this.recordingStartTime,
              msg.text?.trim().length ?? 0);
            this.ws?.close();
            return;
          }
          this.finalReceived = true;
          this.markFinalProgressComplete();
          this.clearTranscriptStallTimer();
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
          if (msg.code === "stt_busy") {
            this.onStatus?.({
              code: "stt_busy",
              message: "Speech model is busy with another recording. Try again in a moment.",
            });
          }
          // The server now sends a clean, user-actionable message for a
          // backend-down failure (no raw `dial tcp …` transport string, plan
          // L2). A "backend_starting" code means on-demand recovery is bringing
          // the speech backend up — surface the server's "is starting…" message
          // verbatim so the user knows to retry shortly.
          this.onError?.(msg.text ?? "Streaming transcription failed");
        }
      } catch {
        // Malformed/non-JSON frame. Don't crash the handler, but don't let it
        // vanish silently either — a burst here can explain a turn that never
        // resolved. Log the first one (and every 10th after) so it's greppable.
        this.malformedMessageCount++;
        if (this.malformedMessageCount === 1 || this.malformedMessageCount % 10 === 0) {
          console.warn("[voice] Ignored malformed WS message (#%d)", this.malformedMessageCount);
        }
      }
    };

    ws.onerror = () => {
      console.warn("[voice] WebSocket error \u2014 will attempt HTTP fallback on close");
      // Don't emit error here; onclose fires after onerror and will handle fallback.
    };

    ws.onclose = () => {
      console.info("[voice] WebSocket closed, finalReceived:", this.finalReceived);
      this.clearServerAckTimer();
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
            this.armServerAckWatchdog();
            // Flush chunks buffered during reconnection (in order)
            this.flushPending();
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
    this.retainedBytes = 0;
    const httpFallbackStart = Date.now();
    transcribeAudioWithRetry(blob, 2, this.language)
      .then((text) => {
        const trimmed = text.trim();
        console.info("[voice] HTTP fallback transcription took %dms, %d chars", Date.now() - httpFallbackStart, trimmed.length);
        // Resolve the turn exactly once, even when empty. An empty result is
        // not "nothing happened" — it's a recoverable silent loss the host
        // must surface (retry banner) rather than leave the UI wedged on
        // "transcribing". See TranscriptionProvider.onResult contract.
        this.finalReceived = true;
        this.markFinalProgressComplete();
        this.onResult?.(trimmed);
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

  async start(): Promise<void> {
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
    this.releaseOwnStream("owner-replaced");

    // ── Stream acquisition ──
    // The provider always acquires (and owns) a fresh mic stream. We no longer
    // accept an injected pre-warmed stream — holding a mic idle to save latency
    // was the low-latency ducking/wedge anti-pattern that was removed.
    const micStart = Date.now();
    try {
      this.lease = await acquireMicStream("voice-stream", { audio: true });
      this.stream = this.lease.stream;
    } catch (err) {
      this.onError?.(classifyMicError(err));
      return;
    }
    console.info("[voice] getUserMedia took %dms", Date.now() - micStart);

    // A freshly-acquired track can briefly start muted and unmute within a few
    // hundred ms, so we do NOT hard-fail here (that would false-reject valid
    // streams). If it stays silent, the no-audio watchdog in useVoiceCore
    // surfaces an actionable error and stops the session.
    if (!isStreamUsable(this.stream)) {
      console.warn("[voice] Acquired mic stream starts muted/not-usable; relying on no-audio watchdog if it persists");
    }

    // Reset session state. The new turn starts now; the previous turn's
    // retained audio is replaced (single-slot retention).
    this.wsUrl = buildVoiceStreamWsUrl(this.language);
    this.finalReceived = false;
    this.firstPartialLogged = false;
    this.serverAckReceived = false;
    this.serverAckPendingNotified = false;
    this.finalProgressPendingNotified = false;
    this.clearServerAckTimer();
    this.clearFinalProgressTimer();
    this.reconnectAttempt = 0;
    this.intentionallyStopped = false;
    this.recordingStartTime = Date.now();
    this.stopTime = 0;
    this.pendingChunks = [];
    this.allChunks = [];
    this.retainedBytes = 0;
    this.chunkCount = 0;
    this.totalBytesSent = 0;
    this.lastTurn = null;
    this.tailDropArmed = false;
    this.droppedChunkCount = 0;
    this.malformedMessageCount = 0;
    this.segmentCount = 0;
    this.clearTranscriptStallTimer();

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
        if (this.tailDropArmed) {
          this.droppedChunkCount++;
          return;
        }
        this.chunkCount++;
        this.totalBytesSent += e.data.size;
        // Keep a copy for HTTP fallback in case streaming fails entirely, up to
        // the retention bound (decodable prefix — see retainedBytes).
        if (this.retainedBytes < VoiceStreamProvider.MAX_RETAINED_AUDIO_BYTES) {
          this.allChunks.push(e.data);
          this.retainedBytes += e.data.size;
        }
        // Send in strict order (see pendingChunks) — never via an async
        // arrayBuffer() microtask that could reorder the header chunk.
        this.enqueueOutbound(e.data);
      }
    };
    this.mediaRecorder.onerror = (ev) => {
      const err = (ev as unknown as { error?: DOMException }).error;
      console.error("[voice] MediaRecorder error at +%dms: %s",
        Date.now() - this.recordingStartTime, err?.name ?? err?.message ?? String(err ?? "unknown"));
      if (this.intentionallyStopped || this.finalReceived) return;
      this.onError?.(classifyMicError(err));
    };
    this.mediaRecorder.start(STREAM_CHUNK_INTERVAL_MS);

    // Observe mid-capture source failures (mute/ended/encoder-error) so a dead
    // audio source is handled honestly rather than surfacing as a mystery VAD
    // stop with no cause in the logs.
    this.attachCaptureLifecycleHandlers();

    // Arm the transcription-stall watchdog: if no partial/segment arrives within
    // TRANSCRIPT_STALL_MS while audio keeps streaming, log it — the "mic stays
    // live but transcription stopped" wedge symptom, made observable.
    this.markTranscriptEvent();

    // ── WebSocket connection ──
    // Reuse a pre-connected WebSocket if available (from preConnect()).
    // Otherwise open a new one. Either way, install recording-session handlers.
    if (hasPreConnectedWs && this.ws) {
      console.info("[voice] Reusing pre-connected WebSocket");
      if (this.ws.readyState === WebSocket.OPEN) this.armServerAckWatchdog();
      // Flush any buffered chunks from the pre-connect phase (in order)
      this.flushPending();
      // Replace the lightweight pre-connect handlers with full recording handlers
      this.setupWsHandlers(this.ws);
    } else {
      const wsConnStart = Date.now();
      this.ws = new WebSocket(this.wsUrl);
      this.ws.onopen = () => {
        console.info("[voice] WebSocket connected in %dms, flushing %d buffered chunks", Date.now() - wsConnStart, this.pendingChunks.length);
        this.armServerAckWatchdog();
        // Flush chunks that were buffered before the WebSocket connected (in order)
        this.flushPending();
      };
      this.setupWsHandlers(this.ws);
    }
  }

  stop(): void {
    this.intentionallyStopped = true;
    this.stopTime = Date.now();
    this.clearTranscriptStallTimer();
    const recordingDuration = this.stopTime - this.recordingStartTime;
    console.info("[voice] Stop: recordingDuration=%dms, chunks=%d, bytes=%d",
      recordingDuration, this.chunkCount, this.totalBytesSent);

    // Tail-drop path: server already declared end-of-utterance, commit the
    // already-buffered segment NOW and skip tail retention. The encoder is
    // still allowed to finish (its final ondataavailable is dropped by the
    // ondataavailable gate above).
    if (this.tailDropArmed) {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "done" }));
      }
      if (this.droppedChunkCount > 0) {
        console.info("[voice] tail-drop dropped %d chunks", this.droppedChunkCount);
      }
      if (this.mediaRecorder?.state === "recording") {
        this.mediaRecorder.onstop = () => {
          this.releaseOwnStream("manual-stop");
        };
        this.mediaRecorder.stop();
      } else {
        this.releaseOwnStream("manual-stop");
      }
    } else if (this.mediaRecorder?.state === "recording") {
      // Defer "done" signal until after final data is flushed.
      this.mediaRecorder.onstop = () => {
        if (this.ws?.readyState === WebSocket.OPEN) {
          this.ws.send(JSON.stringify({ type: "done" }));
        }
        // Snapshot retained audio AFTER the final ondataavailable fires so the
        // retained blob includes the last tail of the turn.
        this.snapshotLastTurn();
        // Stop the mic — the turn is over. Releasing the lease stops the tracks
        // so the OS mic indicator clears and the audio session is freed.
        this.releaseOwnStream("manual-stop");
      };
      this.mediaRecorder.stop();
    } else {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "done" }));
      }
      this.snapshotLastTurn();
      this.releaseOwnStream("manual-stop");
    }

    if (!this.finalReceived) {
      const elapsed = Date.now() - this.recordingStartTime;
      const timeout = computeFinalTimeout(elapsed);
      const sentBytes = this.totalBytesSent;
      this.armFinalProgressWatchdog(sentBytes);
      this.finalTimeout = setTimeout(() => {
        if (!this.finalReceived) {
          this.clearFinalProgressTimer();
          // The streaming `final` never arrived (slow segment, or the session
          // dropped mid-turn and the reconnected server session lost context).
          // Don't just error and discard the audio — fall back to a one-shot
          // HTTP transcription of the FULL retained turn, which recovers the
          // message the streaming path lost. attemptHttpFallback resolves the
          // turn (text, empty, or error) so the UI never wedges.
          console.warn("[voice] Streaming final timed out after %dms — falling back to HTTP transcription of retained audio", timeout);
          this.ws?.close();
          this.attemptHttpFallback();
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
    this.clearTranscriptStallTimer();
    if (this.preConnectTimer) {
      clearTimeout(this.preConnectTimer);
      this.preConnectTimer = null;
    }
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
    }
    this.ws?.close();
    this.ws = null;
    // Release the provider-owned mic stream. DOC: docs/internal/SEAMS.md#mic-ownership-seam
    this.releaseOwnStream("unmount");
    this.pendingChunks = [];
    // Dispose is a full cleanup; drop retained audio too. The hook calls
    // this when reclaiming the provider, not when ending a turn.
    this.lastTurn = null;
  }
}
