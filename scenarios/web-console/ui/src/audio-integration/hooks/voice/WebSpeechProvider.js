// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// WebSpeechProvider — Browser-native Web Speech API fallback.
// Provides continuous recognition with interim results. Quality and availability
// vary by browser. Final fallback when Whisper is entirely unavailable.
// HOST DIFFERENCE: this keeps web-console's scenario-local Web Speech wiring;
// the provider contract and state types come from the shared package.
import { acquireMicStream, releaseMicLease } from "./micOwnership";
import { classifyMicError } from "./types";
export class WebSpeechProvider {
    constructor() {
        this.recognition = null;
        this.micStream = null;
        /** Registry lease for the provider-acquired mic stream (level metering). */
        this.lease = null;
        this.stopped = false;
        /** Tracks how many results have already been dispatched via onResult. */
        this.processedResultCount = 0;
        this.lang = "en-US";
        this.onResult = null;
        this.onError = null;
        this.onPartial = null;
    }
    getStream() {
        return this.micStream;
    }
    /**
     * Web Speech API does not expose the underlying audio bytes — the browser
     * consumes them internally. There is no blob to retain, so this is always
     * null. The UI sees `getLastTurnAudio() === null` and shows a rejection
     * banner without the "Transcribe anyway" button.
     */
    getLastTurnAudio() {
        return null;
    }
    /** No-op: Web Speech API does not retain audio. */
    disposeLastTurn() {
        // intentionally empty — nothing to dispose
    }
    // Web Speech streams transcripts, not audio; nothing for the host to
    // tail-drop. No-op to satisfy the TranscriptionProvider interface.
    dropTail() { }
    async start() {
        const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
        if (!Ctor) {
            console.info("[voice] WebSpeech: API not available");
            this.onError?.("Web Speech API not available");
            return;
        }
        // Acquire a fresh mic stream for audio level monitoring only — WebSpeech
        // handles its own audio capture internally.
        try {
            this.lease = await acquireMicStream("web-speech", { audio: true });
            this.micStream = this.lease.stream;
        }
        catch (err) {
            this.onError?.(classifyMicError(err));
            return;
        }
        this.stopped = false;
        this.processedResultCount = 0;
        this.recognition = new Ctor();
        this.recognition.continuous = true;
        this.recognition.interimResults = true;
        this.recognition.lang = this.lang;
        this.recognition.onresult = (event) => {
            // event.results is cumulative -- it contains ALL results from the start
            // of the session. Only process results we haven't dispatched yet.
            let newFinalText = "";
            let interimText = "";
            for (let i = this.processedResultCount; i < event.results.length; i++) {
                const result = event.results[i];
                if (result?.isFinal) {
                    newFinalText += result[0]?.transcript ?? "";
                    // Mark all results up to and including this one as processed.
                    // We can't skip indices because the API guarantees results
                    // finalize in order.
                    this.processedResultCount = i + 1;
                }
                else {
                    interimText += result?.[0]?.transcript ?? "";
                }
            }
            if (interimText)
                this.onPartial?.(interimText);
            if (newFinalText.trim()) {
                console.info("[voice] WebSpeech: result, %d chars", newFinalText.trim().length);
                this.onResult?.(newFinalText.trim());
            }
        };
        this.recognition.onerror = (event) => {
            if (event.error !== "aborted") {
                console.info("[voice] WebSpeech: error=%s", event.error);
                this.onError?.(`Speech recognition error: ${event.error}`);
            }
        };
        this.recognition.onend = () => {
            // Browser may end continuous recognition spontaneously; restart unless
            // intentionally stopped. There is a brief gap (~100-500ms) during which
            // no audio is captured -- this is an inherent browser limitation.
            // processedResultCount persists across restarts (it's an instance field,
            // not tied to the recognition instance), so previously finalized results
            // are correctly skipped after restart.
            if (!this.stopped && this.recognition) {
                console.info("[voice] WebSpeech: auto-restart");
                try {
                    this.recognition.start();
                }
                catch { /* already started or disposed */ }
            }
        };
        console.info("[voice] WebSpeech: started, lang=%s", this.lang);
        this.recognition.start();
    }
    stop() {
        console.info("[voice] WebSpeech: stopped");
        this.stopped = true;
        this.recognition?.stop();
        this.recognition = null;
        // Release mic so the browser indicator turns off.
        if (this.lease) {
            releaseMicLease(this.lease, "manual-stop");
            this.lease = null;
        }
        this.micStream = null;
    }
    dispose() {
        this.stop();
    }
}
