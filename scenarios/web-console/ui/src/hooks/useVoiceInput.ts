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

interface SpeechRecognitionEventInit extends EventInit {
  results: SpeechRecognitionResultList;
}

interface SpeechRecognitionEvent extends Event {
  readonly results: SpeechRecognitionResultList;
}

interface SpeechRecognitionErrorEventInit extends EventInit {
  error: string;
  message?: string;
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

import { useState, useEffect, useRef, useCallback } from "react";
import { fetchCapabilities, transcribeAudio } from "../lib/api";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

export type VoiceBackend = "whisper" | "web-speech" | "none";

export interface VoiceInputState {
  supported: boolean;
  backend: VoiceBackend;
  isRecording: boolean;
  isTranscribing: boolean;
  error: string | null;
}

interface TranscriptionProvider {
  start(): void;
  stop(): void;
  onResult: ((text: string) => void) | null;
  onError: ((error: string) => void) | null;
}

class WhisperProvider implements TranscriptionProvider {
  private mediaRecorder: MediaRecorder | null = null;
  private chunks: Blob[] = [];
  private stream: MediaStream | null = null;
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;

  async init(): Promise<void> {
    this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  }

  start(): void {
    if (!this.stream) return;
    this.chunks = [];
    this.mediaRecorder = new MediaRecorder(this.stream, {
      mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
        ? "audio/webm;codecs=opus"
        : "audio/webm",
    });
    this.mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) this.chunks.push(e.data);
    };
    this.mediaRecorder.onstop = async () => {
      const blob = new Blob(this.chunks, { type: "audio/webm" });
      this.chunks = [];
      if (blob.size === 0) return;
      try {
        const text = await transcribeAudio(blob);
        if (text.trim()) this.onResult?.(text.trim());
      } catch (err) {
        this.onError?.(err instanceof Error ? err.message : "Transcription failed");
      }
    };
    this.mediaRecorder.start();
  }

  stop(): void {
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
    }
  }

  dispose(): void {
    this.stop();
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
  }
}

class WebSpeechProvider implements TranscriptionProvider {
  private recognition: SpeechRecognitionInstance | null = null;
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;

  start(): void {
    const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
    if (!Ctor) {
      this.onError?.("Web Speech API not available");
      return;
    }
    this.recognition = new Ctor();
    this.recognition.continuous = false;
    this.recognition.interimResults = false;
    this.recognition.lang = "en-US";
    this.recognition.onresult = (event: SpeechRecognitionEvent) => {
      const text = event.results[0]?.[0]?.transcript;
      if (text?.trim()) this.onResult?.(text.trim());
    };
    this.recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      if (event.error !== "aborted") {
        this.onError?.(`Speech recognition error: ${event.error}`);
      }
    };
    this.recognition.start();
  }

  stop(): void {
    this.recognition?.stop();
    this.recognition = null;
  }
}

export function useVoiceInput(onTranscript: (text: string) => void) {
  const voiceEnabled = useWorkspaceStore((s) => s.voiceEnabled);
  const [state, setState] = useState<VoiceInputState>({
    supported: false,
    backend: "none",
    isRecording: false,
    isTranscribing: false,
    error: null,
  });

  const providerRef = useRef<TranscriptionProvider | null>(null);
  const onTranscriptRef = useRef(onTranscript);
  onTranscriptRef.current = onTranscript;

  // Detect available backend on mount
  useEffect(() => {
    if (!voiceEnabled) {
      setState((s) => ({ ...s, supported: false, backend: "none" }));
      return;
    }

    let cancelled = false;
    (async () => {
      try {
        const caps = await fetchCapabilities();
        if (cancelled) return;
        const whisper = caps.capabilities.find((c) => c.id === "whisper-stt");
        if (whisper?.status === "available") {
          const provider = new WhisperProvider();
          try {
            await provider.init();
            if (cancelled) {
              provider.dispose();
              return;
            }
            providerRef.current = provider;
            setState((s) => ({ ...s, supported: true, backend: "whisper" }));
            return;
          } catch {
            // Mic permission denied, fall through to web speech
          }
        }
      } catch {
        // Capabilities fetch failed, fall through
      }

      if (cancelled) return;

      // Try Web Speech API
      const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
      if (Ctor) {
        providerRef.current = new WebSpeechProvider();
        setState((s) => ({ ...s, supported: true, backend: "web-speech" }));
        return;
      }

      setState((s) => ({ ...s, supported: false, backend: "none" }));
    })();

    return () => {
      cancelled = true;
      if (providerRef.current instanceof WhisperProvider) {
        providerRef.current.dispose();
      }
      providerRef.current = null;
    };
  }, [voiceEnabled]);

  const startRecording = useCallback(() => {
    const provider = providerRef.current;
    if (!provider || state.isRecording) return;

    provider.onResult = (text) => {
      setState((s) => ({ ...s, isRecording: false, isTranscribing: false, error: null }));
      onTranscriptRef.current(text);
    };
    provider.onError = (error) => {
      setState((s) => ({ ...s, isRecording: false, isTranscribing: false, error }));
    };

    setState((s) => ({ ...s, isRecording: true, error: null }));
    provider.start();
  }, [state.isRecording]);

  const stopRecording = useCallback(() => {
    const provider = providerRef.current;
    if (!provider || !state.isRecording) return;

    setState((s) => ({
      ...s,
      isRecording: false,
      isTranscribing: state.backend === "whisper",
    }));
    provider.stop();
  }, [state.isRecording, state.backend]);

  return {
    ...state,
    startRecording,
    stopRecording,
  };
}
