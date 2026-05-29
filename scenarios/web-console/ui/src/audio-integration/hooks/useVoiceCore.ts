// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// audio-integration is the canonical copy-paste reference for adopters.
// It includes defensive runtime guards (browser feature detection, race-
// safety checks) that TypeScript's strict-type checking views as
// unnecessary based on declared types but which catch real-world drift.
// Same rationale as useTextToSpeechCore — see that file's note.
/* eslint-disable react-hooks/exhaustive-deps */
//
// Voice Input Core — Generic, Scenario-Agnostic Orchestrator
// ===========================================================
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
//
// This is the generic core: configuration is injected via `opts` rather than
// read from a workspace store, and capability checks / command parsing are
// pluggable hooks. Adopters (e.g. web-console) wrap this with a thin adapter
// that wires their own store and APIs in.

import { useState, useEffect, useRef, useCallback } from "react";
import { getVoiceStreamConfig, transcribeAudioBypassFilter, getWakeWordConfig } from "../index";
import { createAudioFilterChain } from "../index";
import { playRecordingStartCue, playRecordingStopCue } from "../index";
import { buildVoiceActivitySnapshot, IDLE_VOICE_ACTIVITY, voiceActivitySnapshotsEqual } from "../index";
import { createVadRefs, createVadRefsFromCache, extractCacheableFloor, loadNoiseFloorCache, saveNoiseFloorCache, vadTick, VAD_FLOOR_CACHE_MAX_AGE_MS } from "../index";
import { getSharedAudioContext, ensureAudioContextOnGesture } from "../index";
import { acquireStream as acquireMicStream, releaseStream as releaseMicStream, getStream as getMicStream, isStreamAlive as isMicStreamAlive, installVisibilityHandler } from "../index";
import { VoiceStreamProvider } from "../index";
import { setServerVadState, resetServerVadState, useServerVadStateStore, SERVER_VAD_STALE_MS } from "./useServerVadStateStore";
import { decideAutoStop } from "./voice/autoStopDecision";
import { decidePassiveArm } from "./voice/passiveArmDecision";
import { WhisperProvider } from "../index";
import { WebSpeechProvider } from "../index";
import { bytesToFeatures, createWakeWordEngine, PassiveListener } from "../index";
import type { AudioFeatures, WakeWordEngine, WakeWordTemplate } from "../index";
import {
  CAP_CHECK_FAIL_THRESHOLD,
  WHISPER_FAILED_SENTINEL,
} from "../index";
import type {
  TranscriptionProvider,
  VoiceBackend,
  VoiceInputState,
  VoiceMode,
  VoiceSegment,
  VoiceRejection,
  CommandSuggestion,
  StartRecordingOpts,
} from "../index";

const INITIAL_STATE: VoiceInputState = {
  supported: false,
  backend: "none",
  voiceState: "idle",
  error: null,
  audioLevel: 0,
  voiceActivity: IDLE_VOICE_ACTIVITY,
  fallbackNotice: null,
  partialTranscript: "",
  voiceMode: "one-shot",
  segments: [],
  commandSuggestion: null,
  rejectedAudio: null,
  speakerVerificationEnabled: false,
  speakerProfileConfigured: false,
  wakeWordConfigured: false,
};

/**
 * Retention TTL for a rejection's audio blob. After this timeout fires the
 * rejection is auto-dismissed and the retained audio is released. Chosen
 * long enough for a distracted user to come back, short enough that a
 * forgotten banner does not pin memory for the whole session.
 */
const REJECTION_RETENTION_TTL_MS = 5 * 60 * 1000;

/** Generate a stable id for a new rejection. Opaque to consumers. */
function generateRejectionId(): string {
  return `rej-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

/**
 * Result returned by `capabilityCheck`. The core only cares whether Whisper
 * is healthy; adopters can probe their own capability surface to compute it.
 */
export interface VoiceCapabilityProbe {
  whisperHealthy: boolean;
  /** Whether the Whisper backend supports voice-streaming (WS). Defaults to true when omitted. */
  streamingAvailable?: boolean;
}

export interface UseVoiceCoreOptions {
  // What today comes from useWorkspaceStore in web-console:
  voiceEnabled: boolean;
  voiceLanguage: string;
  vadSilenceTimeoutMs: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
  /** Live wake-word match sensitivity (DTW threshold). Single source of truth
   *  for both the settings test and the passive listener — see the threshold
   *  sync effect below. */
  wakeWordThreshold: number;
  segmentSilenceMs: number;
  lowLatencyVoice: boolean;
  // What today comes from web-console's capabilities API:
  /** Optional. Defaults to `{ whisperHealthy: true, streamingAvailable: true }` when omitted. */
  capabilityCheck?: () => Promise<VoiceCapabilityProbe>;
  // What today comes from web-console's commandParser:
  /** Optional. Defaults to `() => null` (no command detection). */
  parseCommand?: (text: string) => CommandSuggestion | null;
  // Callbacks:
  onTranscript: (text: string) => void;
  onCommandSuggest?: (suggestion: CommandSuggestion) => void;
}

const DEFAULT_CAPABILITY_CHECK = async (): Promise<VoiceCapabilityProbe> => ({
  whisperHealthy: true,
  streamingAvailable: true,
});

export function useVoiceCore(opts: UseVoiceCoreOptions) {
  const {
    voiceEnabled,
    voiceLanguage,
    vadSilenceTimeoutMs,
    persistentMode,
    wakeWordEnabled,
    wakeWordThreshold,
    segmentSilenceMs,
    lowLatencyVoice,
    onTranscript,
  } = opts;
  // Stable refs for callbacks/options so async paths see the latest without
  // re-creating effects on every render.
  const capabilityCheckRef = useRef(opts.capabilityCheck ?? DEFAULT_CAPABILITY_CHECK);
  capabilityCheckRef.current = opts.capabilityCheck ?? DEFAULT_CAPABILITY_CHECK;
  const parseCommandRef = useRef<(text: string) => CommandSuggestion | null>(
    opts.parseCommand ?? (() => null),
  );
  parseCommandRef.current = opts.parseCommand ?? (() => null);
  const onCommandSuggestRef = useRef(opts.onCommandSuggest);
  onCommandSuggestRef.current = opts.onCommandSuggest;

  // Ref mirror so effects can read the latest language without re-running on every change.
  const voiceLanguageRef = useRef(voiceLanguage);
  voiceLanguageRef.current = voiceLanguage;
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

  // Keep refs in sync with reactive option values so non-React code (RAF tick, callbacks) sees latest
  const persistentModeRef = useRef(persistentMode);
  persistentModeRef.current = persistentMode;
  const wakeWordEnabledRef = useRef(wakeWordEnabled);
  wakeWordEnabledRef.current = wakeWordEnabled;
  const wakeWordThresholdRef = useRef(wakeWordThreshold);
  wakeWordThresholdRef.current = wakeWordThreshold;
  const segmentSilenceMsRef = useRef(segmentSilenceMs);
  segmentSilenceMsRef.current = segmentSilenceMs;
  const lowLatencyVoiceRef = useRef(lowLatencyVoice);
  lowLatencyVoiceRef.current = lowLatencyVoice;

  // Wake word engine and template refs
  const wakeWordEngineRef = useRef<WakeWordEngine | null>(null);
  const wakeWordTemplateRef = useRef<WakeWordTemplate | null>(null);
  const passiveListenerRef = useRef<PassiveListener | null>(null);
  /** Latches true after a passive-listener start fails (e.g. mic permission
   *  denied) so the auto-arm effect does not retry-storm getUserMedia every
   *  idle render. Cleared when the wake-word toggle (or voiceEnabled) flips. */
  const passiveStartBlockedRef = useRef(false);

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
  const stopRecordingRef = useRef<((opts?: { reason?: "auto" | "user" }) => void) | null>(null);
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

  // Load wake word template and initialize engine on mount.
  // The legacy workspace-store hydration (persistentMode / wakeWordEnabled /
  // segmentSilenceMs sync) is a host concern and lives in the adapter, not
  // here — the core hook accepts these values via `opts`.
  const hydratedRef = useRef(false);
  useEffect(() => {
    if (!voiceEnabled || hydratedRef.current) return;
    hydratedRef.current = true;

    // Touch getVoiceStreamConfig so any host-side prefetch / mocked surface
    // still resolves; the response itself is opaque to the core (host owns
    // the store).
    getVoiceStreamConfig().catch(() => { /* host owns the store, ignore */ });

    // Load wake word template and initialize engine. The template persists RAW
    // audio; the passive listener matches on MFCC features, so re-derive them
    // here via the shared helper (features are never persisted — an engine
    // upgrade re-extracts from the stored audio with no re-enrollment).
    getWakeWordConfig()
      .then(async (cfg) => {
        if (!cfg.configured || !cfg.template) return;
        const engine = wakeWordEngineRef.current ?? createWakeWordEngine();
        wakeWordEngineRef.current = engine;
        const samples = (
          await Promise.all(
            cfg.template.samples.map((s) => bytesToFeatures(s.audio, engine).catch(() => null)),
          )
        ).filter((f): f is AudioFeatures => f !== null);
        if (samples.length === 0) return;
        // Derive score calibration from the enrollment set (how consistent the
        // user's own takes are with each other). Like the MFCC features, this is
        // re-derived on every load and never persisted — see EngineCalibration.
        const calibration = engine.calibrate(samples);
        // Match sensitivity is driven by the LIVE wakeWordThreshold (the slider
        // the user adjusts and the settings test uses), NOT the value baked into
        // the template at save time. Persisted template.threshold is kept on the
        // wire but is no longer authoritative — this is the single source of
        // truth that keeps the test and the passive listener in agreement. The
        // threshold-sync effect below keeps a running listener current.
        wakeWordTemplateRef.current = {
          samples,
          label: cfg.template.label,
          threshold: wakeWordThresholdRef.current,
          updatedAt: cfg.template.updatedAt,
          calibration,
        };
        setState((s) => s.wakeWordConfigured ? s : { ...s, wakeWordConfigured: true });
      })
      .catch(() => { /* No wake word configured */ });
  }, [voiceEnabled]);

  // Sync voice mode state from reactive option value
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
    const parsed = parseCommandRef.current(text);
    if (parsed) {
      console.info("[voice] Command detected via host parser");
      setState((s) => ({ ...s, commandSuggestion: parsed, partialTranscript: "" }));
      onCommandSuggestRef.current?.(parsed);
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
    // Deliver the segment text to the transcript callback. Consecutive
    // committed segments in a turn must be space-separated, otherwise the
    // sinks (the terminal writes raw PTY input; the mobile toolbar appends)
    // run them together ("...sentence.Now here..."). Segments always commit on
    // a speech pause — whole words, never mid-word — so a plain leading space
    // is correct. Skip it for the first segment of the turn and when the
    // segment opens with closing punctuation. This is engine-agnostic (the
    // same path serves Whisper VAD segments) and needs no STT contract change:
    // segment ordering is the only context required, and it lives here.
    const delivered =
      segmentIndex > 0 && !/^[\s,.!?;:]/.test(finalText) ? ` ${finalText}` : finalText;
    onTranscriptRef.current(delivered);
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
        const audioLevel = Math.min(1, rms * 4);
        audioLevelRef.current = audioLevel;

        // Log first 5 non-throttled ticks + every 150th tick (~10s) for diagnostics
        tickCount++;
        if (tickCount <= 5 || tickCount % 150 === 0) {
          const trackAlive = stream.getTracks().every((t) => t.readyState === "live");
          console.info("[voice] S%d tick#%d: rms=%.4f, ctx.state=%s, trackAlive=%s, vadState=%s",
            sessionId, tickCount, rms, ctx.state, trackAlive,
            vadActiveRef.current ? vadRef.current.state : "inactive");
        }

        // VAD check
        const vadNow = Date.now();
        if (vadActiveRef.current) {
          const prevState = vadRef.current.state;
          const result = vadTick(vadRef.current, rms, vadNow, vadSilenceTimeoutRef.current);
          if (vadRef.current.state !== prevState) {
            console.debug("[voice] VAD:", prevState, "→", vadRef.current.state,
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
            console.info("[voice] S%d VAD client-stop: silenceElapsed=%dms, timeout=%dms, rms=%.4f, speechThresh=%.4f, silenceThresh=%.4f",
              sessionCountRef.current, vadNow - vadRef.current.silenceStart,
              vadSilenceTimeoutRef.current, rms,
              vadRef.current.speechThreshold, vadRef.current.silenceThreshold);
            // Persistent mode: treat as one final segment boundary then reset.
            // One-shot mode is handled below via decideAutoStop — keeps the
            // server-VAD SSOT precedence centralised.
            if (persistentModeRef.current) {
              const provider = providerRef.current;
              if (provider && "sendSegmentBoundary" in provider) {
                (provider as VoiceStreamProvider).sendSegmentBoundary();
              }
              vadRef.current.state = "waitingForSpeech";
              vadRef.current.recordingStart = vadNow;
              vadRef.current.segmentBoundaryEmitted = false;
            }
          } else if (result === "no-speech") {
            console.info("[voice] S%d VAD no-speech after %dms, rms=%.4f",
              sessionCountRef.current, vadNow - vadRef.current.recordingStart, rms);
            vadActiveRef.current = false;
            vadRef.current.state = "idle";
            stopRecordingRef.current?.({ reason: "auto" });
            setState((s) => ({ ...s, error: "No speech detected" }));
          }

          // One-shot auto-stop SSOT: server-VAD-led with client-VAD fallback.
          // Pure helper keeps the precedence reviewable + duplicated across
          // the three audio-integration copies; see voice/autoStopDecision.ts
          // and plan audio-tools-stt-accuracy-auto-stop-ssot.md §7 Phase 2.
          if (!persistentModeRef.current && vadActiveRef.current) {
            const serverSnap = useServerVadStateStore.getState();
            const nowPerfMs = typeof performance !== "undefined" && typeof performance.now === "function"
              ? performance.now()
              : Date.now();
            const verdict = decideAutoStop({
              serverVad: serverSnap,
              clientVadResult: result,
              nowPerf: nowPerfMs,
              staleTickMs: SERVER_VAD_STALE_MS,
            });
            if (verdict.kind === "stop") {
              const serverAge = serverSnap.receivedAt > 0
                ? Math.round(nowPerfMs - serverSnap.receivedAt)
                : -1;
              console.info("[voice] S%d auto-stop source=%s serverAge=%dms serverSilence=%d/%dms clientResult=%s",
                sessionCountRef.current, verdict.source, serverAge,
                serverSnap.silenceElapsedMs, serverSnap.silenceTimeoutMs,
                result ?? "null");
              stopRecordingRef.current?.({ reason: "auto" });
            }
          }
        }

        if (!levelMonitorActiveRef.current) return;

        const voiceActivity = buildVoiceActivitySnapshot({
          vadActive: vadActiveRef.current,
          vad: vadRef.current,
          rms,
          audioLevel,
          nowMs: vadNow,
          silenceTimeoutMs: vadSilenceTimeoutRef.current,
          voiceMode: persistentModeRef.current ? "persistent" : "one-shot",
        });
        setState((s) => {
          if (Math.abs(s.audioLevel - audioLevel) < 0.01 && voiceActivitySnapshotsEqual(s.voiceActivity, voiceActivity)) {
            return s;
          }
          return { ...s, audioLevel, voiceActivity };
        });

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
    audioLevelRef.current = 0;
    analyserRef.current = null;
    // Disconnect all audio nodes to prevent zombie node accumulation
    for (const node of audioNodesRef.current) {
      try { node.disconnect(); } catch { /* already disconnected */ }
    }
    audioNodesRef.current = [];
    setState((s) => (s.audioLevel === 0 && voiceActivitySnapshotsEqual(s.voiceActivity, IDLE_VOICE_ACTIVITY)
      ? s
      : { ...s, audioLevel: 0, voiceActivity: IDLE_VOICE_ACTIVITY }));
  }, []);

  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const capCheckFailCountRef = useRef(0);
  /**
   * TTL timer for the currently-displayed rejection. Replaced on every new
   * rejection so only the freshest rejection's 5-minute clock is active.
   */
  const rejectionTtlTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  /**
   * Latest rejection metadata captured from `onSegmentRejected` callbacks
   * during an active turn. The banner is not shown mid-turn — streaming
   * providers only snapshot their retained audio after `stop()`, so we hold
   * the score/threshold here and surface the banner when the turn ends
   * (inside the provider's `onResult` / error paths below).
   */
  const pendingRejectionRef = useRef<{ score: number; threshold: number } | null>(null);
  /** Convenience: the hook's own reference to the current rejection so
   *  callbacks outside React state don't need to re-read `state`. */
  const rejectedAudioRef = useRef<VoiceRejection | null>(null);

  /**
   * Move the currently-retained turn audio from the provider into visible
   * rejection state, consuming the pending rejection metadata. Called when a
   * turn has ended and we know speaker verification rejected at least one
   * segment.
   *
   * Ordering matters: we capture the blob reference via
   * `provider.getLastTurnAudio()` first (provider keeps its own reference),
   * then atomically replace state. The previous rejection's state drops its
   * reference naturally when React discards the old value; the provider's
   * reference is released later by `disposeRejection()` (manual dismiss,
   * successful retry, or TTL).
   *
   */
  const surfacePendingRejection = useCallback(() => {
    const pending = pendingRejectionRef.current;
    if (!pending) return;
    pendingRejectionRef.current = null;

    const provider = providerRef.current;
    const audio = provider?.getLastTurnAudio() ?? null;
    const id = generateRejectionId();
    const createdAt = Date.now();

    const rejection: VoiceRejection = audio
      ? {
          kind: "retryable",
          id,
          blob: audio.blob,
          mimeType: audio.mimeType,
          durationMs: audio.durationMs,
          score: pending.score,
          threshold: pending.threshold,
          createdAt,
          status: "idle",
        }
      : {
          kind: "explanatory",
          id,
          reason: "This provider does not retain audio; please record again to retry.",
          score: pending.score,
          threshold: pending.threshold,
          createdAt,
        };

    rejectedAudioRef.current = rejection;
    setState((s) => ({ ...s, rejectedAudio: rejection }));

    // (Re)arm the retention TTL. A new rejection always replaces any
    // previous TTL timer — only the freshest rejection's clock ticks.
    if (rejectionTtlTimerRef.current) clearTimeout(rejectionTtlTimerRef.current);
    rejectionTtlTimerRef.current = setTimeout(() => {
      // Only fire if this same rejection is still displayed. The ref check
      // prevents a stale timer from clobbering a newer rejection that
      // replaced us after TTL started.
      if (rejectedAudioRef.current?.id === id) {
        const p = providerRef.current;
        p?.disposeLastTurn();
        rejectedAudioRef.current = null;
        rejectionTtlTimerRef.current = null;
        setState((s) => (s.rejectedAudio?.id === id ? { ...s, rejectedAudio: null } : s));
      }
    }, REJECTION_RETENTION_TTL_MS);
  }, []);

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
        const probe = await capabilityCheckRef.current();
        console.info("[voice] Mount capability check took %dms", Date.now() - mountCapStart);
        if (cancelled) return;
        if (probe.whisperHealthy) {
          streamingAvailableRef.current = probe.streamingAvailable ?? true;
          capCheckResolvedRef.current = true;
          console.info("[voice] Backend confirmed: whisper, streaming=%s", streamingAvailableRef.current);

          // Pre-connect the WebSocket so it's ready when the user presses
          // the mic button, eliminating 10-100ms of connection latency.
          // DOC: docs/internal/VOICE-LATENCY.md#websocket-pre-connection
          if (streamingAvailableRef.current) {
            if (!providerRef.current) {
              providerRef.current = new VoiceStreamProvider();
            }
            if (providerRef.current instanceof VoiceStreamProvider) {
              const currentLanguage = voiceLanguageRef.current;
              const lang = currentLanguage === "auto" ? "" : (currentLanguage.split("-")[0] ?? "en");
              providerRef.current.preConnect(lang);
            }
          }

          return;
        }
      } catch (err) {
        console.warn("[voice] Capabilities probe failed on mount:", err);
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

    // Background capability refresh — keeps the snapshot warm so
    // startRecording() never needs to await a network call.
    bgRefreshInterval = setInterval(() => {
      capabilityCheckRef.current().catch(() => {});
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
      if (rejectionTtlTimerRef.current) {
        clearTimeout(rejectionTtlTimerRef.current);
        rejectionTtlTimerRef.current = null;
      }
      rejectedAudioRef.current = null;
      pendingRejectionRef.current = null;
      startingRef.current = false;
    };
  }, [voiceEnabled]);

  const startRecording = useCallback(async (startOpts?: StartRecordingOpts) => {
    if (state.voiceState !== "idle" || startingRef.current) return;
    startingRef.current = true;
    stopRequestedRef.current = false;
    sessionCountRef.current++;
    // Tear down background wake-word listening (if armed) so the recorder owns
    // the mic cleanly. Wake-word detection also funnels through here and has
    // already disposed its listener, so this is a no-op in that path. The
    // auto-arm effect re-arms passive listening once this turn returns to idle.
    if (passiveListenerRef.current) {
      passiveListenerRef.current.dispose();
      passiveListenerRef.current = null;
    }
    // Clear any server-VAD snapshot from a prior session BEFORE the first
    // tick of this one. The sticky silenceTimedOut latch (and the prior
    // session's receivedAt) would otherwise leak across sessions and stop
    // the new recording instantly. See useServerVadStateStore.resetServerVadState.
    resetServerVadState();

    // Show "preparing" state immediately for visual feedback
    const prepareStart = Date.now();
    console.info("[voice] S%d startRecording: backend=%s, streaming=%s, vadEnabled=%s, persistent=%s",
      sessionCountRef.current, backendRef.current, streamingAvailableRef.current,
      startOpts?.vadEnabled, persistentModeRef.current);
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
      // ── Capability re-probe (best-effort, non-blocking) ──
      // DOC: docs/internal/VOICE-LATENCY.md#background-capability-check
      //
      // Kick off a fresh probe and let it settle into refs/state. We do NOT
      // await it on the critical start path; the background interval has
      // typically populated the snapshot already. The probe is here mainly so
      // a stale "unhealthy" state can flip back to "healthy" on next start.
      capabilityCheckRef.current().then((probe) => {
        if (!probe.whisperHealthy) {
          capCheckFailCountRef.current++;
          if (capCheckFailCountRef.current >= CAP_CHECK_FAIL_THRESHOLD && backendRef.current === "whisper") {
            const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
            if (Ctor) {
              providerRef.current?.dispose();
              providerRef.current = null;
              setState((s) => ({
                ...s,
                backend: "web-speech",
                fallbackNotice: "Whisper unavailable — using browser speech recognition",
              }));
              if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
              fallbackTimerRef.current = setTimeout(() => {
                setState((s) => (s.fallbackNotice ? { ...s, fallbackNotice: null } : s));
              }, 5000);
            }
          }
        } else {
          capCheckFailCountRef.current = 0;
          streamingAvailableRef.current = probe.streamingAvailable ?? true;
          if (backendRef.current === "web-speech") {
            providerRef.current?.dispose();
            providerRef.current = null;
            setState((s) => ({ ...s, backend: "whisper", fallbackNotice: null }));
          }
        }
      }).catch(() => {});

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

      // Set language from opts
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
        // A segment-accepted event proves verification is wired up and the
        // profile is configured. We no longer surface a soft banner when a
        // near-miss is accepted — the user only sees a notice when action
        // is available (rejection → retry).
        provider.onSegmentAccepted = (_segmentIndex, _score, _threshold) => {
          setState((s) => ({
            ...s,
            speakerVerificationEnabled: true,
            speakerProfileConfigured: true,
          }));
        };
        // A rejection during a live turn only records the metadata. The
        // banner is deferred until the turn ends (provider's `onResult` /
        // error handler below), because the streaming provider does not
        // snapshot retained audio until `stop()` completes. Multiple
        // rejections in one turn collapse into the last one's score —
        // single-slot retention, one blob per turn.
        provider.onSegmentRejected = (_segmentIndex, score, threshold) => {
          pendingRejectionRef.current = { score, threshold };
          setState((s) => ({
            ...s,
            speakerVerificationEnabled: true,
            speakerProfileConfigured: true,
            partialTranscript: "",
          }));
        };
        provider.onVadState = (snapshot) => {
          setServerVadState(snapshot);
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
          voiceActivity: IDLE_VOICE_ACTIVITY,
          partialTranscript: "",
          segments: [],
        }));
        // Turn ended — surface any pending rejection as a persistent banner.
        // At this point the streaming provider has already snapshotted its
        // retained audio (see VoiceStreamProvider.stop), so
        // getLastTurnAudio() returns the full turn for the retry action.
        surfacePendingRejection();
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
        // Error ends the turn: if speaker verification rejected segments
        // during this turn, surface the banner so the user can still retry
        // with whatever audio was retained.
        surfacePendingRejection();

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
              voiceActivity: IDLE_VOICE_ACTIVITY,
              backend: "web-speech",
              fallbackNotice: "Whisper unavailable — using browser speech recognition",
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
            voiceActivity: IDLE_VOICE_ACTIVITY,
          }));
          return;
        }

        setState((s) => ({ ...s, voiceState: "idle", error, audioLevel: 0, voiceActivity: IDLE_VOICE_ACTIVITY }));
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
        if (startOpts?.vadEnabled || isPersistent) {
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
              ? { ...s, error: "No audio detected — check your microphone" }
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
          // Don't touch rejectedAudio here — the rejection banner (if any) is
          // from a prior completed turn and belongs to the user to dismiss.
          // Disposing the provider below releases its own retained blob;
          // state-level rejection keeps its own copy of the reference.
          setState((s) => ({
            ...s,
            voiceState: "idle",
            audioLevel: 0,
            voiceActivity: IDLE_VOICE_ACTIVITY,
            partialTranscript: "",
            segments: [],
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
  }, [state.voiceState, state.backend, voiceLanguage, startLevelMonitor, stopLevelMonitor, handleSegmentFinal, surfacePendingRejection]);

  const stopRecording = useCallback((opts?: { reason?: "auto" | "user" }) => {
    const reason = opts?.reason ?? "user";
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
        voiceActivity: IDLE_VOICE_ACTIVITY,
        partialTranscript: "",
      }));
    } else {
      setState((s) => ({
        ...s,
        voiceState: state.backend === "whisper" ? "transcribing" : "idle",
        audioLevel: 0,
        voiceActivity: IDLE_VOICE_ACTIVITY,
        partialTranscript: "",
      }));
    }
    // Auto-stop: server-VAD verdict has already fired; capturing more audio
    // would leak post-verdict words into the transcript. Arm tail-drop so
    // the encoder's in-flight chunk is discarded and `{type:"done"}` is sent
    // synchronously to commit the segment. User-tap: preserve the 120 ms
    // settle delay so the encoder's final ondataavailable still ships.
    if (reason === "auto") {
      provider.dropTail();
      provider.stop();
    } else {
      setTimeout(() => provider.stop(), 120);
    }
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
    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    stopLevelMonitor();

    // Cancelling a transcription is a user action on the current turn only;
    // a prior-turn rejection banner is the user's to dismiss explicitly.
    // We clear in-flight pending rejection metadata (there's no banner yet),
    // but leave visible `rejectedAudio` alone.
    pendingRejectionRef.current = null;
    setState((s) => ({
      ...s,
      voiceState: "idle",
      error: null,
      audioLevel: 0,
      voiceActivity: IDLE_VOICE_ACTIVITY,
      partialTranscript: "",
      segments: [],
      commandSuggestion: null,
    }));
  }, [isTranscribing, stopLevelMonitor]);

  /** Dismiss a command suggestion (either confirmed or rejected). */
  const dismissCommandSuggestion = useCallback(() => {
    setState((s) => s.commandSuggestion ? { ...s, commandSuggestion: null } : s);
  }, []);

  /**
   * Dismiss the current rejection banner. Releases the retained audio on
   * the provider and clears the TTL timer. Safe to call when no banner is
   * showing — becomes a no-op.
   */
  const dismissRejection = useCallback(() => {
    if (rejectionTtlTimerRef.current) {
      clearTimeout(rejectionTtlTimerRef.current);
      rejectionTtlTimerRef.current = null;
    }
    providerRef.current?.disposeLastTurn();
    rejectedAudioRef.current = null;
    setState((s) => (s.rejectedAudio ? { ...s, rejectedAudio: null } : s));
  }, []);

  /**
   * Retry transcription of the retained audio while bypassing the server's
   * speaker-verification filter for this single request. No-op if the
   * current rejection has no retained audio (explanatory kind) or if a
   * retry is already in flight.
   *
   * On success the transcript is delivered via the normal `onTranscript`
   * callback and the rejection banner is dismissed. On failure the banner
   * flips to `status: "failed"` with an error message; the user can retry
   * again or dismiss.
   */
  const retryWithoutFilter = useCallback(async () => {
    const current = rejectedAudioRef.current;
    if (!current || current.kind !== "retryable" || current.status === "retrying") {
      return;
    }

    const retryingRejection: VoiceRejection = {
      ...current,
      status: "retrying",
      errorMessage: undefined,
    };
    rejectedAudioRef.current = retryingRejection;
    setState((s) => (s.rejectedAudio?.id === current.id
      ? { ...s, rejectedAudio: retryingRejection }
      : s));

    const langSetting = voiceLanguageRef.current;
    const lang = langSetting === "auto" ? "" : (langSetting.split("-")[0] ?? "en");

    try {
      const text = await transcribeAudioBypassFilter(current.blob, lang);
      const trimmed = text.trim();
      // The user may have dismissed between the await and now — only act
      // if the rejection we're finishing is still the displayed one.
      if (rejectedAudioRef.current?.id !== current.id) return;

      if (trimmed) {
        onTranscriptRef.current(trimmed);
        // Success: dismiss banner and release retained audio.
        if (rejectionTtlTimerRef.current) {
          clearTimeout(rejectionTtlTimerRef.current);
          rejectionTtlTimerRef.current = null;
        }
        providerRef.current?.disposeLastTurn();
        rejectedAudioRef.current = null;
        setState((s) => (s.rejectedAudio?.id === current.id
          ? { ...s, rejectedAudio: null }
          : s));
      } else {
        const failed: VoiceRejection = {
          ...current,
          status: "failed",
          errorMessage: "No speech detected in audio",
        };
        rejectedAudioRef.current = failed;
        setState((s) => (s.rejectedAudio?.id === current.id
          ? { ...s, rejectedAudio: failed }
          : s));
      }
    } catch (err) {
      if (rejectedAudioRef.current?.id !== current.id) return;
      // Cap the error string so a verbose server body doesn't break the
      // banner layout. 200 chars is enough context for the user.
      const raw = err instanceof Error ? err.message : "Network error";
      const msg = raw.length > 200 ? raw.slice(0, 197) + "…" : raw;
      const failed: VoiceRejection = {
        ...current,
        status: "failed",
        errorMessage: msg,
      };
      rejectedAudioRef.current = failed;
      setState((s) => (s.rejectedAudio?.id === current.id
        ? { ...s, rejectedAudio: failed }
        : s));
    }
  }, []);

  // Keep the ref in sync so the tick loop can call stopRecording
  stopRecordingRef.current = stopRecording;

  // ── Passive wake word listening ──
  //
  // Passive listening runs entirely in the BACKGROUND: it does NOT change
  // voiceState (which stays "idle"), so the mic button keeps its normal
  // appearance and stays pressable. The wake word is a *secondary* trigger —
  // the user can still tap-to-talk / use persistent mode exactly as if wake
  // word were off. A press routes through startRecording, which tears the
  // passive listener down so the recorder owns the mic.

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

    // Pick up the latest sensitivity before arming (the user may have moved the
    // slider since the template loaded). The sync effect keeps it current after.
    wakeWordTemplateRef.current.threshold = wakeWordThresholdRef.current;

    // Reuse the app-lifetime shared AudioContext. It is resumed on the first
    // user gesture (ensureAudioContextOnGesture); a context the listener
    // created itself would start suspended on a fresh page load and the
    // analyser would read silence, so passive VAD would never fire.
    const sharedCtx = audioCtxRef.current ?? getSharedAudioContext();
    audioCtxRef.current = sharedCtx;

    const listener = new PassiveListener({
      engine: wakeWordEngineRef.current,
      template: wakeWordTemplateRef.current,
      audioContext: sharedCtx,
      onWakeWordDetected: (_stream: MediaStream) => {
        console.info("[voice] Wake word detected — activating mic");
        // The wake word fires the SAME path as a manual button press. The
        // provider re-acquires its own mic stream in startRecording, so dispose
        // the listener fully here — otherwise its mic stream stays live (mic
        // indicator stuck on, and a second stream contends for the device).
        // dispose() leaves the shared AudioContext open (ownAudioCtx === false).
        listener.dispose();
        passiveListenerRef.current = null;
        startRecording({ vadEnabled: true });
      },
      onError: (error: string) => {
        console.error("[voice] Passive listener error:", error);
        // Latch so the auto-arm effect stops retrying until the toggle flips.
        // Background listening must not surface as a user-facing error or flip
        // voiceState — it stays invisible; only log.
        passiveStartBlockedRef.current = true;
        passiveListenerRef.current = null;
        void error;
      },
    });

    passiveListenerRef.current = listener;
    await listener.start();
  }, [startRecording]);

  /** Stop background passive listening (does not touch voiceState). */
  const exitPassiveMode = useCallback(() => {
    if (passiveListenerRef.current) {
      passiveListenerRef.current.dispose();
      passiveListenerRef.current = null;
    }
  }, []);

  // Release the background passive listener (and its mic stream) on unmount.
  // The main unmount cleanup only disposes the streaming provider; without
  // this, navigating away while passively listening would leak an open mic.
  useEffect(() => () => {
    if (passiveListenerRef.current) {
      passiveListenerRef.current.dispose();
      passiveListenerRef.current = null;
    }
  }, []);

  // ── Keep a running passive listener's threshold in sync ──
  //
  // The PassiveListener reads template.threshold live on every capture, and it
  // holds the SAME object as wakeWordTemplateRef.current. Mutating it here lets
  // the user retune sensitivity with the slider and have the background
  // listener pick it up immediately — no re-arm, no reload. This is what makes
  // the slider the single source of truth for both the settings test and the
  // live detector (they used to diverge: the test used the live store value
  // while the listener used a value frozen into the template at save time).
  useEffect(() => {
    if (wakeWordTemplateRef.current) {
      wakeWordTemplateRef.current.threshold = wakeWordThreshold;
    }
  }, [wakeWordThreshold]);

  // ── Auto-arm passive wake-word listening ──
  //
  // Nothing else in the app starts passive mode, so without this effect the
  // "always-on" wake word never actually listens — the toggle flips a config
  // bit but the mic is never opened. Here we reconcile the desired state:
  // when wake word is enabled and a template is loaded, listen passively in
  // the background whenever voice is idle (voiceState stays "idle" — the
  // button is unaffected). On detection enterPassiveMode routes to
  // startRecording; when that turn ends voiceState returns to "idle" and this
  // effect re-arms — the always-on cycle (idle+listening → wake → record →
  // idle+listening). Disabling the toggle (or a start failure) tears it down.
  useEffect(() => {
    // Clear the failure latch whenever the feature is off so a later
    // re-enable gets a fresh start attempt.
    if (!voiceEnabled || !wakeWordEnabled) {
      passiveStartBlockedRef.current = false;
    }
    const action = decidePassiveArm({
      voiceEnabled,
      wakeWordEnabled,
      wakeWordConfigured: state.wakeWordConfigured,
      voiceState: state.voiceState,
      listenerActive: !!passiveListenerRef.current,
      startBlocked: passiveStartBlockedRef.current,
    });
    if (action === "enter") {
      void enterPassiveMode();
    } else if (action === "exit") {
      exitPassiveMode();
    }
  }, [voiceEnabled, wakeWordEnabled, state.wakeWordConfigured, state.voiceState, enterPassiveMode, exitPassiveMode]);

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
    dismissRejection,
    retryWithoutFilter,
    enterPassiveMode,
    exitPassiveMode,
  };
}
