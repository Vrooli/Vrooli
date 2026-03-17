// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// WebSpeechProvider — Browser-native Web Speech API fallback.
// Provides continuous recognition with interim results. Quality and availability
// vary by browser. Final fallback when Whisper is entirely unavailable.

import type { TranscriptionProvider } from "./types";

// Web Speech API type declarations (not included in all TS libs)
interface SpeechRecognitionResultItem {
  transcript: string;
  confidence: number;
}

interface SpeechRecognitionResult {
  readonly length: number;
  item(index: number): SpeechRecognitionResultItem;
  [index: number]: SpeechRecognitionResultItem;
  isFinal: boolean;
}

interface SpeechRecognitionResultList {
  readonly length: number;
  item(index: number): SpeechRecognitionResult;
  [index: number]: SpeechRecognitionResult;
}

interface SpeechRecognitionEvent extends Event {
  readonly results: SpeechRecognitionResultList;
}

interface SpeechRecognitionErrorEvent extends Event {
  readonly error: string;
  readonly message: string;
}

interface SpeechRecognitionInstance extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onresult: ((event: SpeechRecognitionEvent) => void) | null;
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null;
  onend: (() => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
}

interface SpeechRecognitionConstructor {
  new (): SpeechRecognitionInstance;
}

declare global {
  interface Window {
    SpeechRecognition?: SpeechRecognitionConstructor;
    webkitSpeechRecognition?: SpeechRecognitionConstructor;
  }
}

export class WebSpeechProvider implements TranscriptionProvider {
  private recognition: SpeechRecognitionInstance | null = null;
  private micStream: MediaStream | null = null;
  private stopped = false;
  /** Tracks how many results have already been dispatched via onResult. */
  private processedResultCount = 0;
  lang = "en-US";
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;

  getStream(): MediaStream | null {
    return this.micStream;
  }

  async start(): Promise<void> {
    const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
    if (!Ctor) {
      this.onError?.("Web Speech API not available");
      return;
    }
    // Acquire mic stream for audio level monitoring
    try {
      this.micStream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      this.onError?.("Microphone access denied");
      return;
    }
    this.stopped = false;
    this.processedResultCount = 0;
    this.recognition = new Ctor();
    this.recognition.continuous = true;
    this.recognition.interimResults = true;
    this.recognition.lang = this.lang;
    this.recognition.onresult = (event: SpeechRecognitionEvent) => {
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
        } else {
          interimText += result?.[0]?.transcript ?? "";
        }
      }
      if (interimText) this.onPartial?.(interimText);
      if (newFinalText.trim()) this.onResult?.(newFinalText.trim());
    };
    this.recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      if (event.error !== "aborted") {
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
        try { this.recognition.start(); } catch { /* already started or disposed */ }
      }
    };
    this.recognition.start();
  }

  stop(): void {
    this.stopped = true;
    this.recognition?.stop();
    this.recognition = null;
    // Release mic so the browser indicator turns off
    this.micStream?.getTracks().forEach((t) => t.stop());
    this.micStream = null;
  }

  dispose(): void {
    this.stop();
  }
}
