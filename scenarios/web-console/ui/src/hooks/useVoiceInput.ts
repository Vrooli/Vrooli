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

interface _SpeechRecognitionEventInit extends EventInit {
  results: SpeechRecognitionResultList;
}

interface SpeechRecognitionEvent extends Event {
  readonly results: SpeechRecognitionResultList;
}

interface _SpeechRecognitionErrorEventInit extends EventInit {
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
  /** 0–1 audio level from the microphone while recording */
  audioLevel: number;
}

interface TranscriptionProvider {
  start(): void | Promise<void>;
  stop(): void;
  getStream(): MediaStream | null;
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
    // Request mic permission upfront. Release immediately — start() re-acquires.
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    stream.getTracks().forEach((t) => t.stop());
  }

  getStream(): MediaStream | null {
    return this.stream;
  }

  async start(): Promise<void> {
    // Re-acquire stream each time so the mic indicator only shows during recording.
    // Permission was already granted during init(), so this won't re-prompt.
    try {
      this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      this.onError?.("Microphone access denied");
      return;
    }
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
    // Release mic so the browser indicator turns off
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
  }

  dispose(): void {
    this.stop();
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
  }
}

class WebSpeechProvider implements TranscriptionProvider {
  private recognition: SpeechRecognitionInstance | null = null;
  private micStream: MediaStream | null = null;
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;

  /** Request mic permission upfront so the browser prompts the user. */
  async init(): Promise<void> {
    // Acquire and immediately release — just to trigger the permission prompt.
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    stream.getTracks().forEach((t) => t.stop());
  }

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
    // Release mic so the browser indicator turns off
    this.micStream?.getTracks().forEach((t) => t.stop());
    this.micStream = null;
  }

  dispose(): void {
    this.stop();
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
    audioLevel: 0,
  });

  const providerRef = useRef<TranscriptionProvider | null>(null);
  const onTranscriptRef = useRef(onTranscript);
  onTranscriptRef.current = onTranscript;

  // Audio level monitoring refs
  const audioCtxRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const rafRef = useRef<number>(0);

  const startLevelMonitor = useCallback((stream: MediaStream) => {
    try {
      const ctx = new AudioContext();
      const source = ctx.createMediaStreamSource(stream);
      const analyser = ctx.createAnalyser();
      analyser.fftSize = 256;
      source.connect(analyser);
      audioCtxRef.current = ctx;
      analyserRef.current = analyser;

      const data = new Uint8Array(analyser.frequencyBinCount);
      const tick = () => {
        analyser.getByteTimeDomainData(data);
        // Compute RMS level normalized to 0–1
        let sum = 0;
        for (let i = 0; i < data.length; i++) {
          const v = ((data[i] ?? 128) - 128) / 128;
          sum += v * v;
        }
        const rms = Math.sqrt(sum / data.length);
        // Scale up for visibility (RMS of speech is typically 0.05–0.3)
        const level = Math.min(1, rms * 4);
        setState((s) => {
          if (Math.abs(s.audioLevel - level) < 0.01) return s;
          return { ...s, audioLevel: level };
        });
        rafRef.current = requestAnimationFrame(tick);
      };
      rafRef.current = requestAnimationFrame(tick);
    } catch {
      // AudioContext not available — no level monitoring
    }
  }, []);

  const stopLevelMonitor = useCallback(() => {
    cancelAnimationFrame(rafRef.current);
    rafRef.current = 0;
    audioCtxRef.current?.close().catch(() => {});
    audioCtxRef.current = null;
    analyserRef.current = null;
    setState((s) => (s.audioLevel === 0 ? s : { ...s, audioLevel: 0 }));
  }, []);

  // Detect available backend on mount — no mic permission request.
  // The provider is created lazily on first startRecording().
  useEffect(() => {
    if (!voiceEnabled) {
      setState((s) => ({ ...s, supported: false, backend: "none" }));
      return;
    }

    let cancelled = false;
    (async () => {
      // Check if Whisper is available via capabilities API
      try {
        const caps = await fetchCapabilities();
        if (cancelled) return;
        const whisper = caps.capabilities.find((c) => c.id === "whisper-stt");
        if (whisper?.status === "available") {
          setState((s) => ({ ...s, supported: true, backend: "whisper" }));
          return;
        }
      } catch {
        // Capabilities fetch failed, fall through
      }

      if (cancelled) return;

      // Check if Web Speech API is available (no mic request needed)
      const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
      if (Ctor) {
        setState((s) => ({ ...s, supported: true, backend: "web-speech" }));
        return;
      }

      setState((s) => ({ ...s, supported: false, backend: "none" }));
    })();

    return () => {
      cancelled = true;
      const provider = providerRef.current;
      if (provider instanceof WhisperProvider || provider instanceof WebSpeechProvider) {
        provider.dispose();
      }
      providerRef.current = null;
    };
  }, [voiceEnabled]);

  const startRecording = useCallback(async () => {
    if (state.isRecording) return;

    // Lazily create provider on first use
    if (!providerRef.current) {
      if (state.backend === "whisper") {
        providerRef.current = new WhisperProvider();
      } else if (state.backend === "web-speech") {
        providerRef.current = new WebSpeechProvider();
      } else {
        return;
      }
    }

    const provider = providerRef.current;
    provider.onResult = (text) => {
      stopLevelMonitor();
      setState((s) => ({ ...s, isRecording: false, isTranscribing: false, error: null, audioLevel: 0 }));
      onTranscriptRef.current(text);
    };
    provider.onError = (error) => {
      stopLevelMonitor();
      setState((s) => ({ ...s, isRecording: false, isTranscribing: false, error, audioLevel: 0 }));
    };

    // Clear previous error but don't set isRecording yet — provider.start()
    // may trigger a permission prompt. We only show recording state after
    // the mic is actually acquired.
    setState((s) => ({ ...s, error: null }));
    await provider.start();

    // If start() failed (e.g. permission denied), onError already set state.
    // Check if the mic stream was acquired before entering recording state.
    const stream = provider.getStream();
    if (stream) {
      setState((s) => ({ ...s, isRecording: true }));
      startLevelMonitor(stream);
    }
  }, [state.isRecording, state.backend, startLevelMonitor, stopLevelMonitor]);

  const stopRecording = useCallback(() => {
    const provider = providerRef.current;
    if (!provider || !state.isRecording) return;

    stopLevelMonitor();
    setState((s) => ({
      ...s,
      isRecording: false,
      isTranscribing: state.backend === "whisper",
      audioLevel: 0,
    }));
    provider.stop();
  }, [state.isRecording, state.backend, stopLevelMonitor]);

  return {
    ...state,
    startRecording,
    stopRecording,
  };
}
