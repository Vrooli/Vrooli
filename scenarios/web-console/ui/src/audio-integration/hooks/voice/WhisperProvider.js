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
// HOST DIFFERENCE: the HTTP endpoint is web-console's local RPC adapter;
// capture ownership, error classification, and the provider contract are
// shared with swarm-manager through @vrooli/audio-capture-browser.
import { transcribeAudioWithRetry } from "../../api/voice";
import { acquireMicStream, releaseMicLease } from "./micOwnership";
import { AUDIO_BITRATE, WHISPER_FAILED_SENTINEL, classifyMicError } from "./types";
export class WhisperProvider {
    constructor() {
        this.mediaRecorder = null;
        this.chunks = [];
        this.stream = null;
        /** Registry lease for the provider-acquired mic stream. */
        this.lease = null;
        /** Retained audio from the most recent completed turn, or null. */
        this.lastTurn = null;
        this.language = "en";
        this.onResult = null;
        this.onError = null;
        this.micAcquireTime = 0;
        this.recordingStartTime = 0;
    }
    getStream() {
        return this.stream;
    }
    /** Release the provider-owned mic stream (stops its tracks). Idempotent. */
    releaseOwnStream(reason) {
        if (this.lease) {
            releaseMicLease(this.lease, reason);
            this.lease = null;
        }
        this.stream = null;
    }
    getLastTurnAudio() {
        return this.lastTurn;
    }
    disposeLastTurn() {
        this.lastTurn = null;
    }
    // Batch mode has no tail to drop — the full turn ships in a single POST
    // on stop(), so auto-stop arming is a no-op here. Keeps the provider
    // interface uniform with PcmVoiceStreamProvider.
    dropTail() { }
    async start() {
        // A new turn starts — the previous turn's retained audio is no longer
        // relevant and would grow unbounded if we didn't drop it here.
        this.lastTurn = null;
        // Always acquire (and own) a fresh mic stream.
        const micStart = Date.now();
        try {
            this.lease = await acquireMicStream("whisper", { audio: true });
            this.stream = this.lease.stream;
        }
        catch (err) {
            this.onError?.(classifyMicError(err));
            return;
        }
        this.micAcquireTime = Date.now() - micStart;
        console.info("[voice] WhisperHTTP: getUserMedia took %dms", this.micAcquireTime);
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
            if (e.data.size > 0)
                this.chunks.push(e.data);
        };
        this.mediaRecorder.onstop = async () => {
            const stopTime = Date.now();
            const recordingDuration = stopTime - this.recordingStartTime;
            this.releaseOwnStream("manual-stop");
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
                const trimmed = text.trim();
                console.info("[voice] WhisperHTTP: transcription took %dms, %d chars", Date.now() - transcribeStart, trimmed.length);
                // Resolve the turn exactly once, even when empty, so the host can
                // surface a recoverable notice instead of wedging on "transcribing".
                // See TranscriptionProvider.onResult contract.
                this.onResult?.(trimmed);
            }
            catch {
                console.warn("[voice] WhisperHTTP: transcription failed after %dms", Date.now() - transcribeStart);
                this.onError?.(WHISPER_FAILED_SENTINEL);
            }
        };
        this.mediaRecorder.start();
    }
    stop() {
        if (this.mediaRecorder?.state === "recording") {
            this.mediaRecorder.stop();
            // Mic release happens in onstop after final data is flushed.
        }
        else {
            this.releaseOwnStream("manual-stop");
        }
    }
    dispose() {
        if (this.mediaRecorder?.state === "recording") {
            this.mediaRecorder.stop();
        }
        this.releaseOwnStream("unmount");
        // Dispose is a full cleanup event; drop retained audio too.
        this.lastTurn = null;
    }
}
