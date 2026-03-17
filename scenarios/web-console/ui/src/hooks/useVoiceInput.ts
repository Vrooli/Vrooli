// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Voice Input Hook — Orchestrator
// ================================
//
// Thin orchestrator that wires together the three transcription providers,
// audio level monitoring, VAD, and React state management. Provider
// implementations, VAD logic, and audio utilities live in ./voice/.
//
// State machine: idle -> preparing -> recording -> transcribing -> idle
// The "preparing" state is visible to the UI while the mic is being acquired
// and the provider is initializing.

import { useState, useEffect, useRef, useCallback } from "react";
import { fetchCapabilities, fetchCapabilitiesLiveness } from "../lib/api";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { createAudioFilterChain } from "./voice/audioUtils";
import { createVadRefs, vadTick } from "./voice/vad";
import { VoiceStreamProvider } from "./voice/VoiceStreamProvider";
import { WhisperProvider } from "./voice/WhisperProvider";
import { WebSpeechProvider } from "./voice/WebSpeechProvider";
import {
  CAP_CHECK_FAIL_THRESHOLD,
  WHISPER_FAILED_SENTINEL,
} from "./voice/types";
import type {
  TranscriptionProvider,
  VoiceBackend,
  VoiceInputState,
  StartRecordingOpts,
  VoiceState,
} from "./voice/types";

// Re-export public types and utilities for consumers and tests
export type { TranscriptionProvider, VoiceBackend, VoiceState, VoiceInputState, StartRecordingOpts } from "./voice/types";
export { WHISPER_FAILED_SENTINEL, CAP_CHECK_FAIL_THRESHOLD, AUDIO_BITRATE, STREAM_CHUNK_INTERVAL_MS, computeFinalTimeout } from "./voice/types";
export { createAudioFilterChain } from "./voice/audioUtils";
export type { VadState, VadRefs } from "./voice/vad";
export { VAD_DEFAULT_SILENCE_TIMEOUT_MS, createVadRefs, computeSlidingNoiseFloor, vadTick } from "./voice/vad";

const INITIAL_STATE: VoiceInputState = {
  supported: false,
  backend: "none",
  voiceState: "idle",
  error: null,
  audioLevel: 0,
  fallbackNotice: null,
  partialTranscript: "",
};

export function useVoiceInput(onTranscript: (text: string) => void) {
  const voiceEnabled = useWorkspaceStore((s) => s.voiceEnabled);
  const voiceLanguage = useWorkspaceStore((s) => s.voiceLanguage);
  const vadSilenceTimeoutMs = useWorkspaceStore((s) => s.vadSilenceTimeoutMs);
  const [state, setState] = useState<VoiceInputState>(INITIAL_STATE);

  // Derived booleans for backward compatibility with UI components
  const isRecording = state.voiceState === "recording";
  const isTranscribing = state.voiceState === "transcribing";
  const isPreparing = state.voiceState === "preparing";

  const providerRef = useRef<TranscriptionProvider | null>(null);
  const onTranscriptRef = useRef(onTranscript);
  onTranscriptRef.current = onTranscript;
  const backendRef = useRef<VoiceBackend>(state.backend);
  backendRef.current = state.backend;
  const streamingAvailableRef = useRef(false);

  // Audio level monitoring refs -- AudioContext is reused across recording
  // sessions to avoid hitting the browser's 6-8 context limit.
  const audioCtxRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const rafRef = useRef<number>(0);
  const lastTickRef = useRef(0);
  const audioLevelRef = useRef(0);
  const levelSyncRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // VAD refs
  const vadRef = useRef(createVadRefs());
  const vadActiveRef = useRef(false);
  const stopRecordingRef = useRef<(() => void) | null>(null);
  const vadSilenceTimeoutRef = useRef(vadSilenceTimeoutMs);
  vadSilenceTimeoutRef.current = vadSilenceTimeoutMs;

  const noAudioTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  /** Guards against concurrent startRecording calls during async startup. */
  const startingRef = useRef(false);
  /** When true, stopRecording was called during startup -- recording should abort after start completes. */
  const stopRequestedRef = useRef(false);

  // Track whether the mount-time capability check has resolved,
  // so startRecording knows if streamingAvailableRef is trustworthy.
  const capCheckResolvedRef = useRef(false);

  const startLevelMonitor = useCallback((stream: MediaStream) => {
    try {
      let ctx = audioCtxRef.current;
      if (!ctx) {
        ctx = new AudioContext();
        audioCtxRef.current = ctx;
      }
      if (ctx.state === "suspended") {
        ctx.resume().catch(() => {});
      }

      const source = ctx.createMediaStreamSource(stream);
      const { analyser } = createAudioFilterChain(ctx, source);
      analyserRef.current = analyser;

      const data = new Uint8Array(analyser.frequencyBinCount);
      lastTickRef.current = 0;

      // Sync audioLevel ref -> React state at 10 Hz (100ms).
      levelSyncRef.current = setInterval(() => {
        const l = audioLevelRef.current;
        setState((s) => (Math.abs(s.audioLevel - l) < 0.01 ? s : { ...s, audioLevel: l }));
      }, 100);

      const tick = () => {
        // Throttle to ~15 Hz -- audio analysis doesn't need 60 fps.
        const now = performance.now();
        if (now - lastTickRef.current < 66) {
          rafRef.current = requestAnimationFrame(tick);
          return;
        }
        lastTickRef.current = now;

        analyser.getByteTimeDomainData(data);
        let sum = 0;
        for (let i = 0; i < data.length; i++) {
          const v = ((data[i] ?? 128) - 128) / 128;
          sum += v * v;
        }
        const rms = Math.sqrt(sum / data.length);
        audioLevelRef.current = Math.min(1, rms * 4);

        // VAD check
        if (vadActiveRef.current) {
          const prevState = vadRef.current.state;
          const result = vadTick(vadRef.current, rms, Date.now(), vadSilenceTimeoutRef.current);
          if (vadRef.current.state !== prevState) {
            console.debug("[voice] VAD:", prevState, "\u2192", vadRef.current.state,
              "rms=" + rms.toFixed(3), "speechThresh=" + vadRef.current.speechThreshold.toFixed(3));
          }
          if (result === "stop") {
            stopRecordingRef.current?.();
          } else if (result === "no-speech") {
            vadActiveRef.current = false;
            vadRef.current.state = "idle";
            stopRecordingRef.current?.();
            setState((s) => ({ ...s, error: "No speech detected" }));
          }
        }

        rafRef.current = requestAnimationFrame(tick);
      };
      rafRef.current = requestAnimationFrame(tick);
    } catch {
      // AudioContext not available -- no level monitoring
    }
  }, []);

  const stopLevelMonitor = useCallback(() => {
    cancelAnimationFrame(rafRef.current);
    rafRef.current = 0;
    lastTickRef.current = 0;
    if (levelSyncRef.current) {
      clearInterval(levelSyncRef.current);
      levelSyncRef.current = null;
    }
    audioLevelRef.current = 0;
    analyserRef.current = null;
    setState((s) => (s.audioLevel === 0 ? s : { ...s, audioLevel: 0 }));
  }, []);

  const lastCapCheckRef = useRef(0);
  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const capCheckFailCountRef = useRef(0);

  // Optimistic mount: show the mic button immediately and check Whisper in
  // the background. The user can start speaking before the check resolves.
  useEffect(() => {
    if (!voiceEnabled) {
      setState((s) => ({ ...s, supported: false, backend: "none" }));
      return;
    }

    // Show button immediately -- optimistic default assumes Whisper.
    setState((s) => ({ ...s, supported: true, backend: "whisper" }));
    lastCapCheckRef.current = Date.now();

    let cancelled = false;
    (async () => {
      try {
        const mountCapStart = Date.now();
        const caps = await fetchCapabilities();
        console.info("[voice] Mount capability check took %dms", Date.now() - mountCapStart);
        if (cancelled) return;
        const whisper = caps.capabilities.find((c) => c.id === "whisper-stt");
        if (whisper?.status === "available") {
          streamingAvailableRef.current = whisper.features?.includes("voice-streaming") ?? false;
          capCheckResolvedRef.current = true;
          console.info("[voice] Backend confirmed: whisper, streaming=%s", streamingAvailableRef.current);
          return;
        }
      } catch (err) {
        console.warn("[voice] Capabilities fetch failed on mount:", err);
      }

      if (cancelled) return;
      capCheckResolvedRef.current = true;

      // Whisper unavailable -- downgrade (but don't disrupt an in-progress recording)
      const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
      if (Ctor) {
        console.info("[voice] Backend: web-speech (Whisper unavailable)");
        setState((s) => s.voiceState !== "idle" ? s : { ...s, backend: "web-speech" });
        return;
      }

      console.info("[voice] Backend: none (no voice backend available)");
      setState((s) => s.voiceState !== "idle"
        ? { ...s, error: "Voice not available" }
        : { ...s, supported: false, backend: "none" });
    })();

    return () => {
      cancelled = true;
      providerRef.current?.dispose();
      providerRef.current = null;
      audioCtxRef.current?.close().catch(() => {});
      audioCtxRef.current = null;
      if (noAudioTimerRef.current) {
        clearTimeout(noAudioTimerRef.current);
        noAudioTimerRef.current = null;
      }
      startingRef.current = false;
    };
  }, [voiceEnabled]);

  const startRecording = useCallback(async (opts?: StartRecordingOpts) => {
    if (state.voiceState !== "idle" || startingRef.current) return;
    startingRef.current = true;
    stopRequestedRef.current = false;

    // Show "preparing" state immediately for visual feedback
    const prepareStart = Date.now();
    setState((s) => ({ ...s, voiceState: "preparing", error: null }));

    try {
      // Pre-recording capability check (debounced to every 10s)
      const isWhisperOrFallback = state.backend === "whisper" || state.backend === "web-speech";
      if (isWhisperOrFallback && Date.now() - lastCapCheckRef.current > 10_000) {
        lastCapCheckRef.current = Date.now();
        try {
          // Use liveness check (GET-only) for pre-recording checks,
          // not the full capability check that includes a test transcription.
          const capCheckStart = Date.now();
          const caps = await fetchCapabilitiesLiveness();
          console.info("[voice] Pre-record liveness check took %dms", Date.now() - capCheckStart);
          const whisper = caps.capabilities.find((c) => c.id === "whisper-stt");
          if (whisper?.status !== "available") {
            capCheckFailCountRef.current++;
            console.warn(`[voice] Whisper unavailable (attempt ${capCheckFailCountRef.current}/${CAP_CHECK_FAIL_THRESHOLD})`);
            if (capCheckFailCountRef.current >= CAP_CHECK_FAIL_THRESHOLD && state.backend === "whisper") {
              const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
              if (Ctor) {
                providerRef.current?.dispose();
                providerRef.current = null;
                setState((s) => ({
                  ...s,
                  backend: "web-speech",
                  fallbackNotice: "Whisper unavailable \u2014 using browser speech recognition",
                }));
                if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
                fallbackTimerRef.current = setTimeout(() => {
                  setState((s) => (s.fallbackNotice ? { ...s, fallbackNotice: null } : s));
                }, 5000);
              }
            }
          } else {
            console.info("[voice] Capability check: Whisper available");
            capCheckFailCountRef.current = 0;
            if (state.backend === "web-speech") {
              providerRef.current?.dispose();
              providerRef.current = null;
              streamingAvailableRef.current = whisper.features?.includes("voice-streaming") ?? false;
              setState((s) => ({
                ...s,
                backend: "whisper",
                fallbackNotice: null,
              }));
            }
          }
        } catch (err) {
          console.warn("[voice] Capabilities check failed:", err);
          // Network error -- don't count as Whisper being unavailable
        }
      }

      // If the mount-time check hasn't resolved yet, wait for a quick
      // capability check before creating the provider so we get the right one.
      if (!capCheckResolvedRef.current && backendRef.current === "whisper") {
        try {
          const mountFallbackStart = Date.now();
          const caps = await fetchCapabilitiesLiveness();
          console.info("[voice] Mount-fallback liveness check took %dms", Date.now() - mountFallbackStart);
          const whisper = caps.capabilities.find((c) => c.id === "whisper-stt");
          if (whisper?.status === "available") {
            streamingAvailableRef.current = whisper.features?.includes("voice-streaming") ?? false;
          }
          capCheckResolvedRef.current = true;
        } catch {
          // Proceed with defaults
          capCheckResolvedRef.current = true;
        }
      }

      // Lazily create provider on first use (backendRef reflects any fallback changes)
      if (!providerRef.current) {
        if (backendRef.current === "whisper") {
          providerRef.current = streamingAvailableRef.current
            ? new VoiceStreamProvider()
            : new WhisperProvider();
          console.info("[voice] Provider:", streamingAvailableRef.current ? "VoiceStream" : "WhisperHTTP");
        } else if (backendRef.current === "web-speech") {
          providerRef.current = new WebSpeechProvider();
          console.info("[voice] Provider: WebSpeech");
        } else {
          setState((s) => ({ ...s, voiceState: "idle" }));
          return;
        }
      }

      // Set language from store
      const provider = providerRef.current;
      const langCode = voiceLanguage === "auto" ? "" : (voiceLanguage.split("-")[0] ?? "en");
      if ("language" in provider) provider.language = langCode;
      if ("lang" in provider) {
        (provider as WebSpeechProvider).lang = voiceLanguage === "auto"
          ? (navigator.language || "en-US")
          : voiceLanguage;
      }
      provider.onResult = (text) => {
        console.info("[voice] Transcript:", text.length, "chars");
        if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
        vadActiveRef.current = false;
        vadRef.current.state = "idle";
        stopLevelMonitor();
        setState((s) => ({ ...s, voiceState: "idle", error: null, audioLevel: 0, partialTranscript: "" }));
        onTranscriptRef.current(text);
      };
      provider.onError = (error) => {
        if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
        vadActiveRef.current = false;
        vadRef.current.state = "idle";
        stopLevelMonitor();

        // Whisper failed after retry -- try falling back to Web Speech
        if (error === WHISPER_FAILED_SENTINEL) {
          const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
          if (Ctor) {
            providerRef.current = new WebSpeechProvider();
            setState((s) => ({
              ...s,
              voiceState: "idle",
              error: null,
              audioLevel: 0,
              backend: "web-speech",
              fallbackNotice: "Whisper unavailable \u2014 using browser speech recognition",
            }));
            if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
            fallbackTimerRef.current = setTimeout(() => {
              setState((s) => (s.fallbackNotice ? { ...s, fallbackNotice: null } : s));
            }, 5000);
            return;
          }
          setState((s) => ({
            ...s,
            voiceState: "idle",
            error: "Transcription failed",
            audioLevel: 0,
          }));
          return;
        }

        setState((s) => ({ ...s, voiceState: "idle", error, audioLevel: 0 }));
      };
      if (provider.onPartial !== undefined) {
        provider.onPartial = (text) => {
          setState((s) => ({ ...s, partialTranscript: text }));
        };
      }

      const providerStartTime = Date.now();
      await provider.start();
      console.info("[voice] Provider.start() took %dms (includes getUserMedia)", Date.now() - providerStartTime);

      // If start() failed (e.g. permission denied), onError already set state.
      // Check if the mic stream was acquired before entering recording state.
      const stream = provider.getStream();
      if (stream) {
        // Arm VAD if requested
        if (opts?.vadEnabled) {
          vadActiveRef.current = true;
          vadRef.current = createVadRefs();
          vadRef.current.state = "calibrating";
          vadRef.current.recordingStart = Date.now();
        }

        console.info("[voice] Recording started (preparing took %dms)", Date.now() - prepareStart);
        setState((s) => ({ ...s, voiceState: "recording" }));
        startLevelMonitor(stream);

        // Warn if no audio detected after 2s (catches dead/muted mics)
        if (noAudioTimerRef.current) clearTimeout(noAudioTimerRef.current);
        noAudioTimerRef.current = setTimeout(() => {
          if (audioLevelRef.current === 0) {
            setState((s) => s.voiceState === "recording" ? { ...s, error: "No audio detected \u2014 check your microphone" } : s);
            console.warn("[voice] No audio detected after 2s");
          }
        }, 2000);

        // If stop was requested during async start, abort immediately.
        if (stopRequestedRef.current) {
          stopRequestedRef.current = false;
          if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
          vadActiveRef.current = false;
          vadRef.current.state = "idle";
          stopLevelMonitor();
          setState((s) => ({
            ...s,
            voiceState: "idle",
            audioLevel: 0,
            partialTranscript: "",
          }));
          provider.dispose();
          providerRef.current = null;
          return;
        }
      } else {
        // provider.start() failed -- onError should have fired, but ensure
        // we don't get stuck in "preparing" state.
        setState((s) => s.voiceState === "preparing" ? { ...s, voiceState: "idle" } : s);
      }
    } finally {
      // Always clear the starting guard, even on unexpected errors.
      startingRef.current = false;
    }
  }, [state.voiceState, state.backend, voiceLanguage, startLevelMonitor, stopLevelMonitor]);

  const stopRecording = useCallback(() => {
    // If start is in progress, signal it to abort after completing
    if (startingRef.current) {
      stopRequestedRef.current = true;
      return;
    }

    const provider = providerRef.current;
    if (!provider || !isRecording) return;

    console.info("[voice] Recording stopped");
    if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    stopLevelMonitor();
    setState((s) => ({
      ...s,
      voiceState: state.backend === "whisper" ? "transcribing" : "idle",
      audioLevel: 0,
      partialTranscript: "",
    }));
    provider.stop();
  }, [isRecording, state.backend, stopLevelMonitor]);

  const cancelTranscription = useCallback(() => {
    const provider = providerRef.current;
    if (!provider || !isTranscribing) return;

    console.info("[voice] Transcription cancelled");
    provider.onResult = null;
    provider.onError = null;
    if (provider.onPartial !== undefined) provider.onPartial = null;
    provider.dispose();
    providerRef.current = null;

    if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    stopLevelMonitor();

    setState((s) => ({
      ...s,
      voiceState: "idle",
      error: null,
      audioLevel: 0,
      partialTranscript: "",
    }));
  }, [isTranscribing, stopLevelMonitor]);

  // Keep the ref in sync so the tick loop can call stopRecording
  stopRecordingRef.current = stopRecording;

  return {
    ...state,
    // Derived booleans for backward compat with UI components
    isRecording,
    isTranscribing,
    isPreparing,
    startRecording,
    stopRecording,
    cancelTranscription,
  };
}
