// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Voice Input Hook — Orchestrator
// ================================
//
// Thin orchestrator that wires together the three transcription providers,
// audio level monitoring, VAD, and React state management. Provider
// implementations, VAD logic, and audio utilities live in ./voice/.
//
// State machine:
//   One-shot:   idle -> preparing -> recording -> transcribing -> idle
//   Persistent: idle -> preparing -> listening -> idle
//
// The "preparing" state is visible to the UI while the mic is being acquired
// and the provider is initializing.

import { useState, useEffect, useRef, useCallback } from "react";
import { fetchCapabilities, getCapabilitiesLivenessSnapshot, refreshCapabilitiesLiveness, getVoiceStreamConfig } from "../lib/api";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { createAudioFilterChain } from "./voice/audioUtils";
import { playRecordingStartCue, playRecordingStopCue } from "./voice/audioCues";
import { createVadRefs, createVadRefsFromCache, extractCacheableFloor, loadNoiseFloorCache, saveNoiseFloorCache, vadTick, VAD_FLOOR_CACHE_MAX_AGE_MS } from "./voice/vad";
import { getSharedAudioContext, ensureAudioContextOnGesture } from "./voice/sharedAudioContext";
import { acquireStream as acquireMicStream, releaseStream as releaseMicStream, getStream as getMicStream, isStreamAlive as isMicStreamAlive, installVisibilityHandler } from "./voice/micReadiness";
import { VoiceStreamProvider } from "./voice/VoiceStreamProvider";
import { WhisperProvider } from "./voice/WhisperProvider";
import { WebSpeechProvider } from "./voice/WebSpeechProvider";
import { parseCommandDirect } from "./voice/commandParser";
import { createWakeWordEngine, PassiveListener } from "./voice/wakeword";
import { getWakeWordConfig } from "../lib/api";
import type { WakeWordEngine, WakeWordTemplate } from "./voice/wakeword";
import {
  CAP_CHECK_FAIL_THRESHOLD,
  WHISPER_FAILED_SENTINEL,
} from "./voice/types";
import type {
  TranscriptionProvider,
  VoiceBackend,
  VoiceInputState,
  VoiceMode,
  VoiceSegment,
  CommandSuggestion,
  StartRecordingOpts,
} from "./voice/types";

// Re-export public types and utilities for consumers and tests
export type { TranscriptionProvider, VoiceBackend, VoiceState, VoiceMode, VoiceInputState, VoiceSegment, CommandSuggestion, StartRecordingOpts } from "./voice/types";
export { WHISPER_FAILED_SENTINEL, CAP_CHECK_FAIL_THRESHOLD, AUDIO_BITRATE, STREAM_CHUNK_INTERVAL_MS, computeFinalTimeout } from "./voice/types";
export { createAudioFilterChain } from "./voice/audioUtils";
export type { VadState, VadRefs, VadAction, CachedNoiseFloor } from "./voice/vad";
export { VAD_DEFAULT_SILENCE_TIMEOUT_MS, VAD_DEFAULT_SEGMENT_SILENCE_MS, VAD_FLOOR_CACHE_MAX_AGE_MS, createVadRefs, createVadRefsFromCache, extractCacheableFloor, loadNoiseFloorCache, saveNoiseFloorCache, computeSlidingNoiseFloor, vadTick } from "./voice/vad";
export { getSharedAudioContext, ensureAudioContextOnGesture, closeSharedAudioContext } from "./voice/sharedAudioContext";

const INITIAL_STATE: VoiceInputState = {
  supported: false,
  backend: "none",
  voiceState: "idle",
  error: null,
  audioLevel: 0,
  fallbackNotice: null,
  partialTranscript: "",
  voiceMode: "one-shot",
  segments: [],
  commandSuggestion: null,
  speakerNotice: null,
  speakerVerificationEnabled: false,
  speakerProfileConfigured: false,
  wakeWordConfigured: false,
};

export interface UseVoiceInputCallbacks {
  /** Called when a completed transcript is available (both one-shot and persistent). */
  onTranscript: (text: string) => void;
  /** Called when a voice command is confirmed by the user. */
  onCommandExecute?: (suggestion: CommandSuggestion) => void;
}

export function useVoiceInput(onTranscript: (text: string) => void) {
  const voiceEnabled = useWorkspaceStore((s) => s.voiceEnabled);
  const voiceLanguage = useWorkspaceStore((s) => s.voiceLanguage);
  const vadSilenceTimeoutMs = useWorkspaceStore((s) => s.vadSilenceTimeoutMs);
  const persistentMode = useWorkspaceStore((s) => s.persistentMode);
  const wakeWordEnabled = useWorkspaceStore((s) => s.wakeWordEnabled);
  const segmentSilenceMs = useWorkspaceStore((s) => s.segmentSilenceMs);
  const lowLatencyVoice = useWorkspaceStore((s) => s.lowLatencyVoice);
  const [state, setState] = useState<VoiceInputState>(INITIAL_STATE);

  // Derived booleans for backward compatibility with UI components
  const isRecording = state.voiceState === "recording";
  const isListening = state.voiceState === "listening";
  const isTranscribing = state.voiceState === "transcribing";
  const isPreparing = state.voiceState === "preparing";
  const isPassive = state.voiceState === "passive";
  /** True when mic is active in either mode (excludes passive). */
  const isActive = isRecording || isListening;

  const providerRef = useRef<TranscriptionProvider | null>(null);
  const onTranscriptRef = useRef(onTranscript);
  onTranscriptRef.current = onTranscript;
  const backendRef = useRef<VoiceBackend>(state.backend);
  backendRef.current = state.backend;
  const streamingAvailableRef = useRef(false);

  // Keep refs in sync with reactive store values so non-React code (RAF tick, callbacks) sees latest
  const persistentModeRef = useRef(persistentMode);
  persistentModeRef.current = persistentMode;
  const wakeWordEnabledRef = useRef(wakeWordEnabled);
  wakeWordEnabledRef.current = wakeWordEnabled;
  const segmentSilenceMsRef = useRef(segmentSilenceMs);
  segmentSilenceMsRef.current = segmentSilenceMs;
  const lowLatencyVoiceRef = useRef(lowLatencyVoice);
  lowLatencyVoiceRef.current = lowLatencyVoice;

  // Wake word engine and template refs
  const wakeWordEngineRef = useRef<WakeWordEngine | null>(null);
  const wakeWordTemplateRef = useRef<WakeWordTemplate | null>(null);
  const passiveListenerRef = useRef<PassiveListener | null>(null);

  // Segment tracking for persistent mode
  const segmentsRef = useRef<VoiceSegment[]>([]);

  // Audio level monitoring refs -- AudioContext is reused across recording
  // sessions to avoid hitting the browser's 6-8 context limit.
  const audioCtxRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  /** Audio nodes created by startLevelMonitor — must be disconnected on stop
   *  to prevent zombie node accumulation in the shared AudioContext. */
  const audioNodesRef = useRef<AudioNode[]>([]);
  const rafRef = useRef<number>(0);
  const lastTickRef = useRef(0);
  const audioLevelRef = useRef(0);
  const levelSyncRef = useRef<ReturnType<typeof setInterval> | null>(null);
  /** Guard against zombie RAF ticks. When stopLevelMonitor() is called from
   *  inside the tick callback (e.g. VAD-triggered stop), the tick function
   *  must not reschedule itself. Without this, requestAnimationFrame(tick)
   *  at the end of tick() creates a zombie loop that competes with future
   *  sessions for the shared lastTickRef throttle, starving real ticks and
   *  feeding rms=0 into VAD. */
  const levelMonitorActiveRef = useRef(false);

  // VAD refs
  const vadRef = useRef(createVadRefs());
  const vadActiveRef = useRef(false);
  const stopRecordingRef = useRef<(() => void) | null>(null);
  const vadSilenceTimeoutRef = useRef(vadSilenceTimeoutMs);
  vadSilenceTimeoutRef.current = vadSilenceTimeoutMs;

  const noAudioTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  /** Guards against concurrent startRecording calls during async startup. */
  const startingRef = useRef(false);
  /** Cleanup function for the Page Visibility handler (low-latency mode). */
  const visibilityCleanupRef = useRef<(() => void) | null>(null);
  /** Ref for isActive so non-React callbacks can read it. */
  const isActiveRef = useRef(false);
  isActiveRef.current = isRecording || isListening;
  /** When true, stopRecording was called during startup -- recording should abort after start completes. */
  const stopRequestedRef = useRef(false);

  // ── Audio Cue Session Guard ──
  //
  // Tracks whether we're in a cue-eligible recording session. This decouples
  // audio cues from the mic hardware lifecycle: cues play ONLY when the user
  // is actively recording/listening, never during mic pre-warm, visibility
  // release, cleanup/dispose, or error recovery.
  //
  // The guard ensures cues are always paired: a start cue is always followed
  // by exactly one stop cue for the same session, regardless of which code
  // path ends the recording (user stop, VAD auto-stop, abort during startup,
  // or unmount).
  //
  // DOC: docs/internal/VOICE-LATENCY.md#audio-cue-contract
  const cueSessionActiveRef = useRef(false);

  // Track whether the mount-time capability check has resolved,
  // so startRecording knows if streamingAvailableRef is trustworthy.
  const capCheckResolvedRef = useRef(false);

  // Hydrate workspace store from backend config on mount. This ensures the store
  // has the correct values even if Settings hasn't been opened yet.
  const hydratedRef = useRef(false);
  useEffect(() => {
    if (!voiceEnabled || hydratedRef.current) return;
    hydratedRef.current = true;

    // Hydrate voice stream config
    getVoiceStreamConfig()
      .then((cfg) => {
        const store = useWorkspaceStore.getState();
        if (cfg.persistentMode !== store.persistentMode) store.setPersistentMode(cfg.persistentMode);
        if (cfg.wakeWordEnabled !== store.wakeWordEnabled) store.setWakeWordEnabled(cfg.wakeWordEnabled);
        if (cfg.segmentSilenceMs && cfg.segmentSilenceMs !== store.segmentSilenceMs) store.setSegmentSilenceMs(cfg.segmentSilenceMs);
      })
      .catch(() => { /* Use store defaults */ });

    // Load wake word template and initialize engine
    getWakeWordConfig()
      .then((cfg) => {
        if (cfg.configured && cfg.template) {
          wakeWordTemplateRef.current = cfg.template;
          if (!wakeWordEngineRef.current) {
            wakeWordEngineRef.current = createWakeWordEngine();
          }
          setState((s) => s.wakeWordConfigured ? s : { ...s, wakeWordConfigured: true });
        }
      })
      .catch(() => { /* No wake word configured */ });
  }, [voiceEnabled]);

  // Sync voice mode state from reactive store value
  useEffect(() => {
    setState((s) => {
      const target: VoiceMode = persistentMode ? "persistent" : "one-shot";
      return s.voiceMode === target ? s : { ...s, voiceMode: target };
    });
  }, [persistentMode]);

  // ── Low-latency voice lifecycle ──
  // DOC: docs/internal/VOICE-LATENCY.md#visibility-based-mic-lifecycle
  //
  // When low-latency voice is enabled, install a Page Visibility handler that
  // releases the mic on tab hidden and re-acquires on visible. Also pre-warm
  // the mic stream immediately.
  useEffect(() => {
    if (!voiceEnabled) return;

    if (lowLatencyVoice) {
      // Pre-warm the mic stream
      acquireMicStream().catch((err) => {
        console.warn("[voice] Low-latency: initial mic pre-warm failed:", err);
      });

      // Install visibility handler
      const cleanup = installVisibilityHandler({
        isRecordingActive: () => isActiveRef.current,
        isLowLatencyEnabled: () => lowLatencyVoiceRef.current,
      });
      visibilityCleanupRef.current = cleanup;

      return () => {
        cleanup();
        visibilityCleanupRef.current = null;
        // Release the pre-warmed stream when low-latency is turned off
        releaseMicStream();
      };
    } else {
      // Low-latency was turned off — clean up any leftover state
      if (visibilityCleanupRef.current) {
        visibilityCleanupRef.current();
        visibilityCleanupRef.current = null;
      }
      releaseMicStream();
    }
  }, [voiceEnabled, lowLatencyVoice]);

  /** Handle a segment-final transcript in persistent mode. */
  const handleSegmentFinal = useCallback((text: string, segmentIndex: number) => {
    if (!text.trim()) return;

    // Check for command match (text-based, no prefix needed — wake word detected at audio level)
    const parsed = parseCommandDirect(text);
    if (parsed) {
      console.info("[voice] Command detected: %s (confidence=%.2f)", parsed.command.id, parsed.confidence);
      const suggestion: CommandSuggestion = {
        id: `cmd-${Date.now()}-${segmentIndex}`,
        commandId: parsed.command.id,
        description: parsed.command.description,
        confidence: parsed.confidence,
        rawText: parsed.rawText,
        timestamp: Date.now(),
        args: parsed.args,
      };
      setState((s) => ({ ...s, commandSuggestion: suggestion, partialTranscript: "" }));
      return;
    }

    // Not a command — append as dictation text
    const finalText = text.trim();
    segmentsRef.current = [
      ...segmentsRef.current.slice(0, segmentIndex),
      { text: finalText, isFinal: true },
      ...segmentsRef.current.slice(segmentIndex + 1),
    ];
    setState((s) => ({
      ...s,
      segments: [...segmentsRef.current],
      partialTranscript: "",
    }));
    // Deliver the segment text to the transcript callback
    onTranscriptRef.current(finalText);
  }, []);

  /** Session counter for diagnostic logging — helps correlate log lines
   *  across multiple recording sessions within the same component mount. */
  const sessionCountRef = useRef(0);

  const startLevelMonitor = useCallback(async (stream: MediaStream) => {
    try {
      // Use the shared AudioContext singleton. It was pre-created on the first
      // user gesture by ensureAudioContextOnGesture(), so it should already be
      // in "running" state. We still check for "suspended" as a safety net.
      // DOC: docs/internal/VOICE-LATENCY.md#pre-create-audiocontext-on-first-gesture
      const ctx = getSharedAudioContext();
      audioCtxRef.current = ctx;
      if (ctx.state !== "running") {
        // This resume is a safety net — the primary resume happens in
        // startRecording (within the user gesture context). If we get here
        // with a non-running context, the gesture-context resume failed or
        // the browser suspended the context during provider.start().
        console.warn("[voice] S%d AudioContext still %s in startLevelMonitor (gesture resume may have failed)",
          sessionCountRef.current, ctx.state);
        await ctx.resume().catch(() => {});
      }

      const sessionId = sessionCountRef.current;
      const trackStates = stream.getTracks().map((t) => `${t.kind}:${t.readyState}`);
      console.info("[voice] S%d startLevelMonitor: ctx.state=%s, stream.active=%s, tracks=[%s]",
        sessionId, ctx.state, stream.active, trackStates.join(","));

      // Disconnect any lingering nodes from a previous session to prevent
      // zombie node accumulation in the AudioContext audio graph.
      for (const node of audioNodesRef.current) {
        try { node.disconnect(); } catch { /* already disconnected */ }
      }
      audioNodesRef.current = [];

      const source = ctx.createMediaStreamSource(stream);
      const { analyser, nodes } = createAudioFilterChain(ctx, source);
      analyserRef.current = analyser;
      audioNodesRef.current = [source, ...nodes];

      const data = new Uint8Array(analyser.frequencyBinCount);
      lastTickRef.current = 0;
      levelMonitorActiveRef.current = true;
      /** Counts non-throttled ticks in this session for early diagnostic logging. */
      let tickCount = 0;

      // Sync audioLevel ref -> React state at 10 Hz (100ms).
      levelSyncRef.current = setInterval(() => {
        const l = audioLevelRef.current;
        setState((s) => (Math.abs(s.audioLevel - l) < 0.01 ? s : { ...s, audioLevel: l }));
      }, 100);

      const tick = () => {
        // Zombie guard: if stopLevelMonitor was called (e.g. VAD-triggered
        // stop from within this tick), do NOT reschedule. Without this,
        // zombie ticks accumulate and starve real level monitors.
        if (!levelMonitorActiveRef.current) return;

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

        // Log first 5 non-throttled ticks + every 150th tick (~10s) for diagnostics
        tickCount++;
        if (tickCount <= 5 || tickCount % 150 === 0) {
          const trackAlive = stream.getTracks().every((t) => t.readyState === "live");
          console.info("[voice] S%d tick#%d: rms=%.4f, ctx.state=%s, trackAlive=%s, vadState=%s",
            sessionId, tickCount, rms, ctx.state, trackAlive,
            vadActiveRef.current ? vadRef.current.state : "inactive");
        }

        // VAD check
        if (vadActiveRef.current) {
          const prevState = vadRef.current.state;
          const result = vadTick(vadRef.current, rms, Date.now(), vadSilenceTimeoutRef.current);
          if (vadRef.current.state !== prevState) {
            console.debug("[voice] VAD:", prevState, "\u2192", vadRef.current.state,
              "rms=" + rms.toFixed(3), "speechThresh=" + vadRef.current.speechThreshold.toFixed(3));
            // Notify backend of speech state changes so it can skip
            // partial transcription during silence (prevents Whisper hallucinations).
            const provider = providerRef.current;
            if (provider && "sendVadState" in provider) {
              const sp = provider as VoiceStreamProvider;
              if (vadRef.current.state === "speechDetected") {
                sp.sendVadState(true);
              } else if (vadRef.current.state === "watchingSilence" || vadRef.current.state === "waitingForSpeech") {
                sp.sendVadState(false);
              }
            }
          }
          if (result === "segment-boundary") {
            // In persistent mode: trigger segment-final transcription
            const provider = providerRef.current;
            if (provider && "sendSegmentBoundary" in provider) {
              (provider as VoiceStreamProvider).sendSegmentBoundary();
              const silenceDuration = Date.now() - vadRef.current.silenceStart;
              console.info("[voice] Segment boundary sent to backend, silenceDuration=%dms, segmentSilenceMs=%d",
                silenceDuration, vadRef.current.segmentSilenceMs);
            }
          } else if (result === "stop") {
            console.info("[voice] S%d VAD stop: silenceElapsed=%dms, timeout=%dms, rms=%.4f, speechThresh=%.4f, silenceThresh=%.4f",
              sessionCountRef.current, Date.now() - vadRef.current.silenceStart,
              vadSilenceTimeoutRef.current, rms,
              vadRef.current.speechThreshold, vadRef.current.silenceThreshold);
            // In one-shot mode, stop recording as usual.
            if (!persistentModeRef.current) {
              stopRecordingRef.current?.();
            }
            // In persistent mode, this fires after segmentSilenceMs + remaining silence.
            // We treat it as one final segment boundary, then reset to waitingForSpeech
            // so "stop" doesn't fire again on every subsequent tick.
            if (persistentModeRef.current) {
              const provider = providerRef.current;
              if (provider && "sendSegmentBoundary" in provider) {
                (provider as VoiceStreamProvider).sendSegmentBoundary();
              }
              // Reset VAD to wait for new speech — prevents repeated "stop" cascade
              vadRef.current.state = "waitingForSpeech";
              vadRef.current.recordingStart = Date.now();
              vadRef.current.segmentBoundaryEmitted = false;
            }
          } else if (result === "no-speech") {
            console.info("[voice] S%d VAD no-speech after %dms, rms=%.4f",
              sessionCountRef.current, Date.now() - vadRef.current.recordingStart, rms);
            vadActiveRef.current = false;
            vadRef.current.state = "idle";
            stopRecordingRef.current?.();
            setState((s) => ({ ...s, error: "No speech detected" }));
          }
        }

        rafRef.current = requestAnimationFrame(tick);
      };
      rafRef.current = requestAnimationFrame(tick);
    } catch (err) {
      console.error("[voice] S%d startLevelMonitor FAILED:", sessionCountRef.current, err);
    }
  }, []);

  const stopLevelMonitor = useCallback(() => {
    levelMonitorActiveRef.current = false;
    cancelAnimationFrame(rafRef.current);
    rafRef.current = 0;
    lastTickRef.current = 0;
    if (levelSyncRef.current) {
      clearInterval(levelSyncRef.current);
      levelSyncRef.current = null;
    }
    audioLevelRef.current = 0;
    analyserRef.current = null;
    // Disconnect all audio nodes to prevent zombie node accumulation
    for (const node of audioNodesRef.current) {
      try { node.disconnect(); } catch { /* already disconnected */ }
    }
    audioNodesRef.current = [];
    setState((s) => (s.audioLevel === 0 ? s : { ...s, audioLevel: 0 }));
  }, []);

  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const capCheckFailCountRef = useRef(0);
  const speakerNoticeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Optimistic mount: show the mic button immediately and check Whisper in
  // the background. The user can start speaking before the check resolves.
  //
  // DOC: docs/internal/VOICE-LATENCY.md#background-capability-check
  // DOC: docs/internal/VOICE-LATENCY.md#pre-create-audiocontext-on-first-gesture
  useEffect(() => {
    if (!voiceEnabled) {
      setState((s) => ({ ...s, supported: false, backend: "none" }));
      return;
    }

    // Show button immediately -- optimistic default assumes Whisper.
    setState((s) => ({ ...s, supported: true, backend: "whisper" }));

    // Pre-create AudioContext on the first user gesture anywhere in the app.
    // This eliminates ~20-50ms from the recording start path by ensuring the
    // context is already in "running" state when the mic button is pressed.
    ensureAudioContextOnGesture();

    let cancelled = false;
    let bgRefreshInterval: ReturnType<typeof setInterval> | null = null;

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

          // Seed the synchronous snapshot so startRecording() never blocks.
          // The mount check used fetchCapabilities (full check), but we also
          // need the liveness cache populated for the snapshot getter.
          refreshCapabilitiesLiveness().catch(() => {});

          // Pre-connect the WebSocket so it's ready when the user presses
          // the mic button, eliminating 10-100ms of connection latency.
          // DOC: docs/internal/VOICE-LATENCY.md#websocket-pre-connection
          if (streamingAvailableRef.current) {
            if (!providerRef.current) {
              providerRef.current = new VoiceStreamProvider();
            }
            if (providerRef.current instanceof VoiceStreamProvider) {
              const lang = voiceLanguage === "auto" ? "" : (voiceLanguage.split("-")[0] ?? "en");
              providerRef.current.preConnect(lang);
            }
          }

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

    // Background capability refresh — keeps the synchronous snapshot warm
    // so startRecording() never needs to await a network call. Runs every 25s
    // (inside the server-side 30s cache TTL) to ensure freshness.
    bgRefreshInterval = setInterval(() => {
      refreshCapabilitiesLiveness().catch(() => {});
    }, 25_000);

    return () => {
      cancelled = true;
      if (bgRefreshInterval) clearInterval(bgRefreshInterval);
      // Clear the cue session guard WITHOUT playing the stop cue. Unmount is
      // not a user-initiated recording stop — it's a lifecycle event. Playing
      // a cue here would be confusing (e.g., closing a tab shouldn't chime).
      cueSessionActiveRef.current = false;
      providerRef.current?.dispose();
      providerRef.current = null;
      // Do NOT close the shared AudioContext here — it is app-lifetime and
      // managed by sharedAudioContext.ts. Individual audio nodes are disconnected
      // by stopLevelMonitor() which is called before unmount.
      if (noAudioTimerRef.current) {
        clearTimeout(noAudioTimerRef.current);
        noAudioTimerRef.current = null;
      }
      if (speakerNoticeTimerRef.current) {
        clearTimeout(speakerNoticeTimerRef.current);
        speakerNoticeTimerRef.current = null;
      }
      startingRef.current = false;
    };
  }, [voiceEnabled]);

  const startRecording = useCallback(async (opts?: StartRecordingOpts) => {
    if (state.voiceState !== "idle" || startingRef.current) return;
    startingRef.current = true;
    stopRequestedRef.current = false;
    sessionCountRef.current++;

    // Show "preparing" state immediately for visual feedback
    const prepareStart = Date.now();
    console.info("[voice] S%d startRecording: backend=%s, streaming=%s, vadEnabled=%s, persistent=%s",
      sessionCountRef.current, backendRef.current, streamingAvailableRef.current,
      opts?.vadEnabled, persistentModeRef.current);
    setState((s) => ({ ...s, voiceState: "preparing", error: null }));

    // ── Resume AudioContext in user gesture context ──
    // Mobile browsers (Chrome Android, Safari iOS) suspend the AudioContext for
    // power saving between user gestures. ctx.resume() MUST be called synchronously
    // within the user gesture call stack — after an `await`, the gesture context is
    // lost and the browser silently refuses to resume. We call resume() here (before
    // any async operations) rather than in startLevelMonitor (which runs after
    // `await provider.start()`).
    //
    // Without this, the AnalyserNode returns stale silence data, causing:
    //   - Volume indicator stuck at 0
    //   - VAD sees rms=0 → premature stop or no-speech timeout
    //
    // This is safe to call on desktop too — it's a no-op when ctx.state is "running".
    try {
      const ctx = getSharedAudioContext();
      if (ctx.state !== "running") {
        console.info("[voice] S%d Resuming AudioContext (state=%s) in gesture context", sessionCountRef.current, ctx.state);
        ctx.resume().catch(() => {});
      }
    } catch { /* AudioContext unavailable */ }

    // Determine the mode for this session
    const isPersistent = persistentModeRef.current;

    // Persistent mode requires Whisper streaming — if not available, disable persistent mode
    if (isPersistent && (backendRef.current !== "whisper" || !streamingAvailableRef.current)) {
      console.warn("[voice] Persistent mode requires Whisper streaming, falling back to one-shot");
      persistentModeRef.current = false;
    }

    try {
      // ── Synchronous capability check ──
      // DOC: docs/internal/VOICE-LATENCY.md#background-capability-check
      //
      // Instead of blocking on a network request, read the synchronous snapshot
      // populated by the background refresh interval (25s cycle). This eliminates
      // 50-500ms from the recording start path.
      //
      // If the snapshot is null (very first activation before mount check resolved),
      // we fall through to the optimistic Whisper assumption (already set in the
      // mount effect). The background refresh will populate the snapshot shortly.
      const capSnapshot = getCapabilitiesLivenessSnapshot();
      if (capSnapshot) {
        const whisper = capSnapshot.capabilities.find((c) => c.id === "whisper-stt");
        if (whisper?.status !== "available") {
          capCheckFailCountRef.current++;
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
          capCheckFailCountRef.current = 0;
          streamingAvailableRef.current = whisper.features?.includes("voice-streaming") ?? false;
          if (state.backend === "web-speech") {
            providerRef.current?.dispose();
            providerRef.current = null;
            setState((s) => ({ ...s, backend: "whisper", fallbackNotice: null }));
          }
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

      // Wire up segment-final handler for persistent mode
      if (provider instanceof VoiceStreamProvider) {
        provider.onSegmentFinal = handleSegmentFinal;
        provider.onSegmentAccepted = (_segmentIndex, score, threshold) => {
          setState((s) => ({
            ...s,
            speakerVerificationEnabled: true,
            speakerProfileConfigured: true,
            speakerNotice: score < threshold ? "Speaker verification advisory: transcript kept." : s.speakerNotice,
          }));
        };
        provider.onSegmentRejected = (_segmentIndex, score, threshold) => {
          if (speakerNoticeTimerRef.current) clearTimeout(speakerNoticeTimerRef.current);
          setState((s) => ({
            ...s,
            speakerVerificationEnabled: true,
            speakerProfileConfigured: true,
            speakerNotice: score > 0
              ? `Ignored speech that did not match your voice (${score.toFixed(2)} < ${threshold.toFixed(2)})`
              : "Ignored speech that did not match your voice",
            partialTranscript: "",
          }));
          speakerNoticeTimerRef.current = setTimeout(() => {
            setState((s) => (s.speakerNotice ? { ...s, speakerNotice: null } : s));
          }, 3000);
        };
        provider.onSpeakerStatus = (enabled, profileConfigured) => {
          setState((s) => ({
            ...s,
            speakerVerificationEnabled: enabled,
            speakerProfileConfigured: profileConfigured,
          }));
        };
      }

      provider.onResult = (text) => {
        console.info("[voice] S%d onResult: %d chars, vadActive=%s, vadState=%s",
          sessionCountRef.current, text.length, vadActiveRef.current, vadRef.current.state);
        // Clear cue session — the stop cue already played in stopRecording().
        // If onResult fires without stopRecording() (e.g. server-side stop),
        // we still clear the guard to prevent a stale cue on next session.
        cueSessionActiveRef.current = false;
        if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }

        // Noise floor is now saved in stopRecording() (before VAD state reset).
        // Previously this guard was always false because stopRecording clears
        // vadActiveRef before onResult fires.
        vadActiveRef.current = false;
        vadRef.current.state = "idle";
        stopLevelMonitor();
        setState((s) => ({
          ...s,
          voiceState: "idle",
          error: null,
          audioLevel: 0,
          partialTranscript: "",
          segments: [],
        }));
        // In persistent mode, segment-finals deliver text incrementally.
        // The final message contains only the un-segmented tail (speech
        // after the last segment boundary). Deliver it if non-empty.
        if (text) {
          onTranscriptRef.current(text);
        }

        // ── Low-latency: release-then-reacquire cycle ──
        // DOC: docs/internal/VOICE-LATENCY.md#audio-ducking-deep-dive
        //
        // Release the mic immediately to stop audio ducking on mobile. Then
        // re-acquire after 500ms so the stream is warm for the next recording.
        if (lowLatencyVoiceRef.current) {
          releaseMicStream();
          setTimeout(() => {
            if (lowLatencyVoiceRef.current) {
              acquireMicStream().catch(() => {});
            }
          }, 500);
        }
      };
      provider.onError = (error) => {
        // Clear cue session without playing stop cue — errors are not normal
        // recording stops. Playing a pleasant "done" chime after an error
        // would be misleading.
        cueSessionActiveRef.current = false;
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

      // ── Stream injection (low-latency mode) ──
      // DOC: docs/internal/VOICE-LATENCY.md#stream-injection-vs-stream-acquisition
      //
      // If low-latency voice is enabled and a pre-warmed stream exists, inject
      // it into the provider to skip the getUserMedia call (~50-300ms).
      // Also tell VoiceStreamProvider to retain the stream after recording
      // so it can be re-used for subsequent sessions.
      let preWarmedStream: MediaStream | undefined;
      if (lowLatencyVoiceRef.current && isMicStreamAlive()) {
        preWarmedStream = getMicStream() ?? undefined;
        if (provider instanceof VoiceStreamProvider) {
          provider.retainStream = true;
        }
      } else if (provider instanceof VoiceStreamProvider) {
        provider.retainStream = false;
      }

      const providerStartTime = Date.now();
      await provider.start(preWarmedStream);
      console.info("[voice] Provider.start() took %dms (includes getUserMedia)", Date.now() - providerStartTime);

      // If start() failed (e.g. permission denied), onError already set state.
      // Check if the mic stream was acquired before entering recording state.
      const stream = provider.getStream();
      if (stream) {
        // Arm VAD
        // DOC: docs/internal/VOICE-LATENCY.md#persistent-noise-floor-cache
        if (opts?.vadEnabled || isPersistent) {
          vadActiveRef.current = true;

          // Try to seed from cached noise floor to skip the 500ms calibration.
          // The sliding window adaptation still runs and will self-correct if
          // the environment has changed. A drift guard in vadTick detects gross
          // mismatches (>3x divergence) and resets from live data immediately.
          const cached = loadNoiseFloorCache();
          const cacheAge = cached ? Date.now() - cached.timestamp : Infinity;
          if (cached && cacheAge < VAD_FLOOR_CACHE_MAX_AGE_MS) {
            vadRef.current = createVadRefsFromCache(cached);
            console.info("[voice] Noise floor cache: loaded (age=%ds, floor=%.4f)",
              Math.round(cacheAge / 1000), cached.silenceThreshold / 1.5);
          } else {
            vadRef.current = createVadRefs();
            vadRef.current.state = "calibrating";
            if (cached) {
              console.info("[voice] Noise floor cache: expired (age=%ds), will recalibrate",
                Math.round(cacheAge / 1000));
            }
          }

          vadRef.current.recordingStart = Date.now();
          // Enable segment boundary detection in persistent mode
          if (isPersistent) {
            vadRef.current.segmentSilenceMs = segmentSilenceMsRef.current;
          }
        }

        // Reset segment tracking for persistent mode
        segmentsRef.current = [];

        const targetState = isPersistent ? "listening" : "recording";
        console.info("[voice] %s started (preparing took %dms)", targetState, Date.now() - prepareStart);
        setState((s) => ({
          ...s,
          voiceState: targetState,
          voiceMode: isPersistent ? "persistent" : "one-shot",
          segments: [],
          commandSuggestion: null,
        }));
        cueSessionActiveRef.current = true;
        playRecordingStartCue();
        startLevelMonitor(stream);

        // Warn if no audio detected after 2s (catches dead/muted mics)
        if (noAudioTimerRef.current) clearTimeout(noAudioTimerRef.current);
        noAudioTimerRef.current = setTimeout(() => {
          if (audioLevelRef.current === 0) {
            setState((s) => (s.voiceState === "recording" || s.voiceState === "listening")
              ? { ...s, error: "No audio detected \u2014 check your microphone" }
              : s);
            console.warn("[voice] No audio detected after 2s");
          }
        }, 2000);

        // If stop was requested during async start, abort immediately.
        // The start cue already played above, so play the matching stop cue
        // to keep the pair balanced.
        if (stopRequestedRef.current) {
          stopRequestedRef.current = false;
          if (cueSessionActiveRef.current) {
            cueSessionActiveRef.current = false;
            playRecordingStopCue();
          }
          if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
          vadActiveRef.current = false;
          vadRef.current.state = "idle";
          stopLevelMonitor();
          setState((s) => ({
            ...s,
            voiceState: "idle",
            audioLevel: 0,
            partialTranscript: "",
            segments: [],
            speakerNotice: null,
          }));
          provider.dispose();
          providerRef.current = null;
          return;
        }
      } else {
        setState((s) => s.voiceState === "preparing" ? { ...s, voiceState: "idle" } : s);
      }
    } finally {
      startingRef.current = false;
    }
  }, [state.voiceState, state.backend, voiceLanguage, startLevelMonitor, stopLevelMonitor, handleSegmentFinal]);

  const stopRecording = useCallback(() => {
    // If start is in progress, signal it to abort after completing
    if (startingRef.current) {
      stopRequestedRef.current = true;
      console.info("[voice] S%d stopRecording: deferred (start in progress)", sessionCountRef.current);
      return;
    }

    const provider = providerRef.current;
    if (!provider || !isActive) {
      console.warn("[voice] S%d stopRecording: no-op (provider=%s, isActive=%s)",
        sessionCountRef.current, !!provider, isActive);
      return;
    }

    console.info("[voice] S%d %s stopped", sessionCountRef.current, isListening ? "Listening" : "Recording");
    // Only play the stop cue if a cue session is active (start cue was played).
    // This prevents the stop sound from firing during cleanup, error recovery,
    // or any other path that disposes the provider without a preceding start cue.
    if (cueSessionActiveRef.current) {
      cueSessionActiveRef.current = false;
      playRecordingStopCue();
    }
    if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }

    // Persist the noise floor BEFORE resetting VAD state. Previously this
    // lived in onResult, but stopRecording always clears vadActiveRef before
    // onResult fires, so the save guard was always false — the cache was
    // never written. Moving it here ensures the thresholds are captured
    // while the VAD state is still valid.
    // DOC: docs/internal/VOICE-LATENCY.md#persistent-noise-floor-cache
    if (vadActiveRef.current && vadRef.current.state !== "idle") {
      const floor = extractCacheableFloor(vadRef.current);
      saveNoiseFloorCache(floor);
      console.info("[voice] S%d Noise floor saved (speech=%.4f, silence=%.4f)",
        sessionCountRef.current, floor.speechThreshold, floor.silenceThreshold);
    }

    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    stopLevelMonitor();

    if (isListening) {
      // Persistent mode: stop cleanly, the final segment-final will be
      // the last retranscription from the backend's "done" handler.
      setState((s) => ({
        ...s,
        voiceState: state.backend === "whisper" ? "transcribing" : "idle",
        audioLevel: 0,
        partialTranscript: "",
      }));
    } else {
      setState((s) => ({
        ...s,
        voiceState: state.backend === "whisper" ? "transcribing" : "idle",
        audioLevel: 0,
        partialTranscript: "",
      }));
    }
    setTimeout(() => provider.stop(), 120);
  }, [isActive, isListening, state.backend, stopLevelMonitor]);

  const cancelTranscription = useCallback(() => {
    const provider = providerRef.current;
    if (!provider || !isTranscribing) return;

    console.info("[voice] Transcription cancelled");
    // Clear cue session without playing stop cue — cancellation is not a
    // normal recording stop. The stop cue already played when stopRecording()
    // transitioned the state to "transcribing".
    cueSessionActiveRef.current = false;
    provider.onResult = null;
    provider.onError = null;
    if (provider.onPartial !== undefined) provider.onPartial = null;
    if (provider instanceof VoiceStreamProvider) {
      provider.onSegmentFinal = null;
      provider.onSegmentAccepted = null;
      provider.onSegmentRejected = null;
      provider.onSpeakerStatus = null;
    }
    provider.dispose();
    providerRef.current = null;

    if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
    if (speakerNoticeTimerRef.current) { clearTimeout(speakerNoticeTimerRef.current); speakerNoticeTimerRef.current = null; }
    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    stopLevelMonitor();

    setState((s) => ({
      ...s,
      voiceState: "idle",
      error: null,
      audioLevel: 0,
      partialTranscript: "",
      segments: [],
      commandSuggestion: null,
      speakerNotice: null,
    }));
  }, [isTranscribing, stopLevelMonitor]);

  /** Dismiss a command suggestion (either confirmed or rejected). */
  const dismissCommandSuggestion = useCallback(() => {
    setState((s) => s.commandSuggestion ? { ...s, commandSuggestion: null } : s);
  }, []);

  // Keep the ref in sync so the tick loop can call stopRecording
  stopRecordingRef.current = stopRecording;

  // ── Passive wake word listening ──

  /** Enter passive listening mode (VAD + MFCC/DTW, no backend streaming). */
  const enterPassiveMode = useCallback(async () => {
    if (!wakeWordEngineRef.current || !wakeWordTemplateRef.current) {
      console.warn("[voice] Cannot enter passive mode: no wake word configured");
      return;
    }
    if (passiveListenerRef.current) {
      passiveListenerRef.current.dispose();
      passiveListenerRef.current = null;
    }

    const listener = new PassiveListener({
      engine: wakeWordEngineRef.current,
      template: wakeWordTemplateRef.current,
      audioContext: audioCtxRef.current ?? undefined,
      onWakeWordDetected: (stream: MediaStream) => {
        console.info("[voice] Wake word detected — activating mic");
        // Transition from passive to active recording.
        // The PassiveListener has stopped its loop but kept the stream alive.
        setState((s) => ({ ...s, voiceState: "preparing" }));
        passiveListenerRef.current = null;

        // Save the AudioContext from the listener for reuse
        if (listener.getAudioContext()) {
          audioCtxRef.current = listener.getAudioContext();
        }

        // Start recording using the existing mic stream
        startRecording({ vadEnabled: true });
      },
      onError: (error: string) => {
        console.error("[voice] Passive listener error:", error);
        setState((s) => ({ ...s, voiceState: "idle", error }));
        passiveListenerRef.current = null;
      },
    });

    passiveListenerRef.current = listener;
    setState((s) => ({ ...s, voiceState: "passive", error: null }));
    await listener.start();
  }, [startRecording]);

  /** Exit passive listening mode and return to idle. */
  const exitPassiveMode = useCallback(() => {
    if (passiveListenerRef.current) {
      passiveListenerRef.current.dispose();
      passiveListenerRef.current = null;
    }
    setState((s) => s.voiceState === "passive" ? { ...s, voiceState: "idle" } : s);
  }, []);

  return {
    ...state,
    // Derived booleans for UI components
    isRecording,
    isListening,
    isTranscribing,
    isPreparing,
    isPassive,
    isActive,
    startRecording,
    stopRecording,
    cancelTranscription,
    dismissCommandSuggestion,
    enterPassiveMode,
    exitPassiveMode,
  };
}
