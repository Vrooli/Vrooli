import { useState, useEffect, useRef, useCallback } from "react";
import { PcmVoiceStreamProvider } from "./pcmVoiceStreamProvider";
import { buildVoiceActivitySnapshot, IDLE_VOICE_ACTIVITY, voiceActivitySnapshotsEqual } from "./voice/activity";
import { createVadRefs, createVadRefsFromCache, extractCacheableFloor, loadNoiseFloorCache, saveNoiseFloorCache, vadTick, VAD_FLOOR_CACHE_MAX_AGE_MS, VAD_MIN_SILENCE_THRESHOLD } from "./voice/vad";
import { advanceMeterEnvelope, LEVEL_ANALYSER_FFT_SIZE, LEVEL_TICK_MS, meterLevelFromEnvelope } from "./voice/audioUtils";
import { getSharedAudioContext, ensureRunningSharedAudioContext, suspendSharedAudioContext, armIdleSuspend, keepAudioContextAwake } from "./voice/sharedAudioContext";
import { installMicLifecycleCleanup, subscribeMicLeases, getActiveMicLeases } from "./voice/micOwnership";
import { VoiceCaptureController } from "./voice/voiceCaptureController";
import { decideMicLifecycle, isStandaloneDisplayMode, selectStaleLeases } from "./voice/micLifecyclePolicy";
import type { LifecycleReleaseScope, MicReleaseReason, MicLeaseSnapshot } from "./voice/micOwnership";
import { setServerVadState, resetServerVadState, useServerVadStateStore, SERVER_VAD_STALE_MS } from "./voice/useServerVadStateStore";
import { decideAutoStop } from "./voice/autoStopDecision";
import { decidePassiveArm } from "./voice/passiveArmDecision";
import { decidePersistentMode, PERSISTENT_STREAMING_UNAVAILABLE_MESSAGE } from "./voice/persistentModeDecision";
import { TranscriptBuffer } from "./transcriptBuffer";
import { WHISPER_FAILED_SENTINEL } from "./voice/types";
import type { TranscriptionProvider, VoiceBackend, VoiceInputState, VoiceMode, VoiceSegment, VoiceRejection, CommandSuggestion, StartRecordingOpts } from "./voice/types";
import type { AudioFeatures, WakeWordEngine, WakeWordTemplate } from "./voice/wakeword";
import type { VoiceCoreServices } from "./voice/services";


const INITIAL_STATE: VoiceInputState = {
  supported: false,
  backend: "none",
  voiceState: "idle",
  error: null,
  audioLevel: 0,
  voiceActivity: IDLE_VOICE_ACTIVITY,
  fallbackNotice: null,
  streamingDegradationNotice: null,
  partialTranscript: "",
  voiceMode: "one-shot",
  segments: [],
  commandSuggestion: null,
  rejectedAudio: null,
  speakerVerificationEnabled: false,
  speakerProfileConfigured: false,
  turnDiagnostic: null,
  wakeWordConfigured: false,
  passiveListeningActive: false,
  staleLiveMicLease: false,
};

function isStreamingProvider(provider: TranscriptionProvider | null): provider is PcmVoiceStreamProvider {
  return provider instanceof PcmVoiceStreamProvider ||
    (provider !== null && typeof (provider as PcmVoiceStreamProvider).sendSegmentBoundary === "function");
}

/**
 * Retention TTL for a rejection's audio blob. After this timeout fires the
 * rejection is auto-dismissed and the retained audio is released. Chosen
 * long enough for a distracted user to come back, short enough that a
 * forgotten banner does not pin memory for the whole session.
 */
const REJECTION_RETENTION_TTL_MS = 5 * 60 * 1000;

/** Audio level (0..1) below which the mic is treated as silent by the no-audio
 *  watchdog. Not strict zero: a wedged AudioContext or muted track produces
 *  tiny non-zero noise, which the old ===0 check missed. */
const NO_AUDIO_LEVEL_THRESHOLD = 0.01;

/**
 * Stream status codes are a mixed vocabulary: some report a condition a person
 * should act on, most are protocol bookkeeping between the browser and the
 * speech backend. Treating the whole vocabulary as user-facing — with a
 * deny-list of the few codes known to be noise — made every newly added
 * protocol code a user-visible notice by default. `processed_acknowledgement`
 * arrives per acknowledged wire batch, so that default produced a banner that
 * appeared and vanished several times a second while the operator was talking.
 *
 * The rule is now the other way round: a status reaches the operator only when
 * its emitter supplied copy written for a human (see `dispatchStreamMessage`,
 * which no longer invents any). These two sets cover the codes that need
 * behaviour beyond that default.
 */
const SILENT_STREAM_STATUS = new Set([
  // Pure bookkeeping. These carry sequence numbers and provider identity, and
  // must never reach the operator even if a future server build adds text.
  "processed_acknowledgement",
  "provider_identity",
]);

/** Codes meaning "whatever was wrong is fine now" — they retire the notice. */
const NOTICE_CLEARING_STREAM_STATUS = new Set([
  "stream_connected",
  "transcription_complete",
  "buffered_recovery_completed",
  "mic_reacquired",
  "mic_unmuted",
]);

/** Codes that additionally record a streaming-quality degradation. */
const DEGRADATION_STREAM_STATUS = new Set([
  "backend_degraded",
  "reconnect_exhausted",
  "buffered_recovery",
]);

/** Generate a stable id for a new rejection. Opaque to consumers. */
function generateRejectionId(): string {
  return `rej-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

/** Convert capture-start failures into an actionable, privacy-safe UI error. */
function describeProviderStartFailure(error: unknown): string {
  if (typeof DOMException !== "undefined" && error instanceof DOMException) {
    if (error.name === "NotAllowedError" || error.name === "PermissionDeniedError") {
      return "Microphone permission was denied; allow microphone access and try again.";
    }
    if (error.name === "NotFoundError" || error.name === "DevicesNotFoundError") {
      return "No microphone device is available; connect a microphone and try again.";
    }
    if (error.name === "NotReadableError" || error.name === "TrackStartError") {
      return "The microphone could not be opened; it may be in use by another application.";
    }
  }
  if (error instanceof Error && error.message.trim()) return error.message.trim();
  return "Voice input could not start; check microphone access and try again.";
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
  readonly services: VoiceCoreServices;
  // What today comes from useWorkspaceStore in web-console:
  voiceEnabled: boolean;
  voiceLanguage: string;
  vadSilenceTimeoutMs: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
  /** Live wake-word match sensitivity (DTW threshold). Single source of truth
   *  for both the settings test and the passive listener — see the threshold
   *  sync effect below. */
  wakeWordThreshold?: number;
  segmentSilenceMs: number;
  /** Retained for host compatibility; the shared core owns capture timing. */
  lowLatencyVoice?: boolean;
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
  const { services,
    voiceEnabled,
    voiceLanguage,
    vadSilenceTimeoutMs,
    persistentMode,
    wakeWordEnabled,
    wakeWordThreshold = 0.7,
    segmentSilenceMs,
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
  // Honest passive state: driven by real passive-listener activity, not by a
  // workflow state. voiceState stays "idle" while passively listening (so
  // tap-to-talk still works); the mic control reads this instead.
  const isPassive = state.passiveListeningActive;
  /** True when mic is active in either mode (excludes passive). */
  const isActive = isRecording || isListening;

  const providerRef = useRef<TranscriptionProvider | null>(null);
  // Mirror of the workflow state for non-React callbacks (the registry lease
  // subscription) that must read the latest value without re-subscribing.
  const voiceStateRef = useRef<VoiceInputState["voiceState"]>(state.voiceState);
  voiceStateRef.current = state.voiceState;
  // Hook-local capture teardown, wired after stopLevelMonitor is defined. The
  // controller calls this on every provider shutdown so capture machinery and
  // workflow ownership always tear down together. Idempotent.
  const captureTeardownRef = useRef<(reason: MicReleaseReason) => void>(() => {});
  // Single authority for provider replacement / disposal / shutdown / stale-lease
  // recovery + the start-cancellation generation token. Wraps providerRef so
  // reads stay `providerRef.current`; only sanctioned mutations go through it.
  const captureControllerRef = useRef<VoiceCaptureController | null>(null);
  if (!captureControllerRef.current) {
    captureControllerRef.current = new VoiceCaptureController(providerRef, {
      onCaptureTeardown: (reason) => captureTeardownRef.current(reason),
    });
  }
  const controller = captureControllerRef.current;
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

  // Wake word engine and template refs
  const wakeWordEngineRef = useRef<WakeWordEngine | null>(null);
  const wakeWordTemplateRef = useRef<WakeWordTemplate | null>(null);
  const passiveListenerRef = useRef<InstanceType<VoiceCoreServices["PassiveListener"]> | null>(null);
  /** Latches true after a passive-listener start fails (e.g. mic permission
   *  denied) so the auto-arm effect does not retry-storm getUserMedia every
   *  idle render. Cleared when the wake-word toggle (or voiceEnabled) flips. */
  const passiveStartBlockedRef = useRef(false);

  // Segment tracking for persistent mode
  const segmentsRef = useRef<VoiceSegment[]>([]);
  /**
   * True once any usable text has been delivered for the current turn — via a
   * segment-final (persistent mode) or a non-empty `final`. Reset at the start
   * of each turn. When a turn ends with this still false and no speaker
   * rejection pending, the turn was a silent loss: we surface a recoverable
   * "couldn't transcribe" banner instead of dropping the audio silently.
   */
  const turnDeliveredTextRef = useRef(false);
  /** Single owner for replaceable interim text and durable transcript bounds. */
  const transcriptBufferRef = useRef(new TranscriptBuffer());
  const dismissedFallbackNoticeRef = useRef<string | null>(null);
  /**
   * Coalesced partial RENDER, decoupled from TranscriptBuffer's exact state
   * used for tail recovery. A high partial rate must not jank the main thread and
   * re-introduce client-side backpressure, so interim partial text is throttled
   * to one paint per animation frame; durable segment-finals still render
   * immediately. pendingPartialRenderRef holds the latest text awaiting paint;
   * partialRenderRafRef is the scheduled frame handle (0 = none).
   */
  const pendingPartialRenderRef = useRef<string | null>(null);
  const partialRenderRafRef = useRef<number>(0);

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
  const endCaptureRef = useRef<((opts?: { reason?: "auto" | "user" }) => void) | null>(null);
  const vadSilenceTimeoutRef = useRef(vadSilenceTimeoutMs);
  vadSilenceTimeoutRef.current = vadSilenceTimeoutMs;

  const noAudioTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  /** Guards against concurrent startRecording calls during async startup. */
  const startingRef = useRef(false);
  /** Latest document-hidden state, read by non-React paths (timers, reconcile)
   *  to keep background mic acquisition from racing a hidden tab. */
  const documentHiddenRef = useRef(
    typeof document !== "undefined" && document.visibilityState === "hidden",
  );
  /** Holds the latest passive-arm reconcile so the visibility effect (which has
   *  a narrow dep list) can call it on becoming visible without going stale. */
  const reconcilePassiveRef = useRef<(() => void) | null>(null);
  /** True only after an explicit mic-control intent in the current visible page
   *  session. Mount/visibility alone must never acquire the microphone. */
  const micIntentArmedRef = useRef(false);
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

    // Touch services.getVoiceStreamConfig so any host-side prefetch / mocked surface
    // still resolves; the response itself is opaque to the core (host owns
    // the store).
    services.getVoiceStreamConfig().catch(() => { /* host owns the store, ignore */ });

    // Load wake word template and initialize engine. The template persists RAW
    // audio; the passive listener matches on MFCC features, so re-derive them
    // here via the shared helper (features are never persisted — an engine
    // upgrade re-extracts from the stored audio with no re-enrollment).
    services.getWakeWordConfig()
      .then(async (cfg) => {
        if (!cfg.configured || !cfg.template) return;
        const engine = wakeWordEngineRef.current ?? services.createWakeWordEngine();
        wakeWordEngineRef.current = engine;
        const samples = (
          await Promise.all(
            cfg.template.samples.map((s) => services.bytesToFeatures(s.audio, engine).catch(() => null)),
          )
        ).filter((f): f is AudioFeatures => f !== null);
        if (samples.length === 0) return;
        // Derive score calibration from the enrollment set (how consistent the
        // user's own takes are with each other). Like the MFCC features, this is
        // re-derived on every load and never persisted — see EngineCalibration.
        const calibration = engine.calibrate?.(samples) ?? null;
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

  /** Handle a segment-final transcript in persistent mode. */
  const handleSegmentFinal = useCallback((text: string, segmentIndex: number) => {
    if (!text.trim()) return;
    const transcript = transcriptBufferRef.current.segmentFinal(text, segmentIndex);
    if (!transcript.accepted) return;
    // A recognized segment (dictation OR command) means this turn produced
    // usable output — it is not a silent loss. Record it before either branch.
    turnDeliveredTextRef.current = true;

    // Check for command match (text-based, no prefix needed — wake word detected at audio level)
    const parsed = parseCommandRef.current(text);
    if (parsed) {
      console.info("[voice] Command detected via host parser");
      setState((s) => ({ ...s, commandSuggestion: parsed, partialTranscript: transcript.interimText }));
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
      partialTranscript: transcript.interimText,
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

  /** Cancel any scheduled coalesced partial render and drop the pending text.
   *  Called on every turn-terminal path so a queued frame can never paint stale
   *  partial text after the turn cleared it. Stable identity (no deps). */
  const cancelPartialRender = useCallback(() => {
    if (partialRenderRafRef.current !== 0) {
      if (typeof cancelAnimationFrame === "function") {
        cancelAnimationFrame(partialRenderRafRef.current);
      }
      partialRenderRafRef.current = 0;
    }
    pendingPartialRenderRef.current = null;
  }, []);

  /** Session counter for diagnostic logging — helps correlate log lines
   *  across multiple recording sessions within the same component mount. */
  const sessionCountRef = useRef(0);

  const startLevelMonitor = useCallback(async (stream: MediaStream) => {
    try {
      // Use the shared AudioContext singleton, resumed lazily here (inside the
      // real capture path) — not eagerly on the first arbitrary tap. ensureRunning
      // resumes it and cancels any idle-suspend armed by a prior turn.
      // DOC: docs/internal/VOICE-LATENCY.md#audiocontext-lifecycle
      // Heal a wedged context (suspended/interrupted that won't resume) by
      // rebuilding it, rather than reusing a dead one that yields a flat level
      // meter forever. Non-fatal: capture does not depend on this context.
      const ctx = await ensureRunningSharedAudioContext();
      audioCtxRef.current = ctx;
      if (ctx.state !== "running") {
        console.warn("[voice] S%d AudioContext still %s after ensureRunning (level meter may be flat)",
          sessionCountRef.current, ctx.state);
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
      const { analyser, nodes } = services.createAudioFilterChain(ctx, source);
      analyserRef.current = analyser;
      audioNodesRef.current = [source, ...nodes];

      // Read the WHOLE time-domain window. This used to allocate
      // `frequencyBinCount` (half of fftSize), so even the short window that
      // was configured was only half-read. `|| LEVEL_ANALYSER_FFT_SIZE` keeps
      // test doubles that leave fftSize at 0 out of a zero-length buffer.
      const data = new Uint8Array(analyser.fftSize || LEVEL_ANALYSER_FFT_SIZE);
      lastTickRef.current = 0;
      levelMonitorActiveRef.current = true;
      /** Counts non-throttled ticks in this session for early diagnostic logging. */
      let tickCount = 0;
      /** Previous audio-source health, so a mid-capture flip (track died/muted or
       *  the AudioContext left "running") is logged the instant it happens — not
       *  up to ~10s later on the next periodic tick. A silent audio source is the
       *  root cause of a VAD auto-stop that otherwise looks like "the mic just
       *  stopped for no reason". */
      let prevSrcHealthy = true;
      /** Set once we've suppressed an auto-stop because the audio source wasn't
       *  delivering samples (muted / suspended). Prevents per-tick log spam;
       *  reset when the source recovers. */
      let unhealthyStopSuppressed = false;
      /** Meter envelope, carried across ticks. Session-local: a fresh capture
       *  starts from silence rather than inheriting the last turn's loudness. */
      let meterEnvelope = 0;

      const tick = () => {
        // Zombie guard: if stopLevelMonitor was called (e.g. VAD-triggered
        // stop from within this tick), do NOT reschedule. Without this,
        // zombie ticks accumulate and starve real level monitors.
        if (!levelMonitorActiveRef.current) return;

        // Throttle to ~15 Hz -- audio analysis doesn't need 60 fps.
        const now = performance.now();
        const sinceLastTick = lastTickRef.current > 0 ? now - lastTickRef.current : LEVEL_TICK_MS;
        if (sinceLastTick < LEVEL_TICK_MS) {
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
        // RAW absolute reading. The no-audio watchdog below is calibrated to
        // this scale, so it must not become the display value.
        const audioLevel = Math.min(1, rms * 4);
        audioLevelRef.current = audioLevel;

        // DISPLAY reading. A meter and a watchdog want different things: the
        // watchdog asks "is anything arriving at all", the meter asks "how loud
        // is this against the room". Sharing one number meant the meter
        // inherited the watchdog's fixed absolute scale and sat near the floor
        // on any normal microphone. This one follows an envelope and is
        // measured against the VAD's adaptive noise floor.
        meterEnvelope = advanceMeterEnvelope(meterEnvelope, rms, sinceLastTick);
        const meterLevel = meterLevelFromEnvelope(
          meterEnvelope,
          Math.max(vadRef.current.silenceThreshold, VAD_MIN_SILENCE_THRESHOLD),
        );

        // Audio-source health, evaluated every tick so a mid-capture failure is
        // caught immediately. `muted === true` means no samples flow even though
        // readyState stays "live" (the analyser reads silence and the VAD will
        // auto-stop) — see streamHealth.isTrackUsable.
        const srcTracks = stream.getTracks();
        const trackAlive = srcTracks.every((t) => t.readyState === "live");
        const trackMuted = srcTracks.some((t) => t.muted);
        const srcHealthy = trackAlive && !trackMuted && ctx.state === "running";

        tickCount++;
        // Log the instant the source goes unhealthy (or recovers) — this is the
        // decisive triage line for "the mic abruptly stopped mid-speech".
        if (srcHealthy !== prevSrcHealthy) {
          const fn = srcHealthy ? console.info : console.warn;
          fn(`[voice] S${sessionId} tick#${tickCount} audio-source ${srcHealthy ? "RECOVERED" : "WENT SILENT"}: rms=${rms.toFixed(4)}, ctx.state=${ctx.state}, trackAlive=${trackAlive}, trackMuted=${trackMuted}, vadState=${vadActiveRef.current ? vadRef.current.state : "inactive"}`);
          prevSrcHealthy = srcHealthy;
          if (srcHealthy) unhealthyStopSuppressed = false;
        }
        // Periodic heartbeat: first 5 ticks + every 150th (~10s) for diagnostics.
        if (tickCount <= 5 || tickCount % 150 === 0) {
          console.info(`[voice] S${sessionId} tick#${tickCount}: rms=${rms.toFixed(4)}, meter=${meterLevel.toFixed(2)}, floor=${vadRef.current.silenceThreshold.toFixed(4)}, ctx.state=${ctx.state}, trackAlive=${trackAlive}, trackMuted=${trackMuted}, vadState=${vadActiveRef.current ? vadRef.current.state : "inactive"}`);
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
              const sp = provider as PcmVoiceStreamProvider;
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
              (provider as PcmVoiceStreamProvider).sendSegmentBoundary();
              const silenceDuration = Date.now() - vadRef.current.silenceStart;
              console.info("[voice] Segment boundary sent to backend, silenceDuration=%dms, segmentSilenceMs=%d",
                silenceDuration, vadRef.current.segmentSilenceMs);
            }
          } else if (result === "stop") {
            const srcTracks = stream.getTracks();
            const srcAlive = srcTracks.every((t) => t.readyState === "live");
            const srcMuted = srcTracks.some((t) => t.muted);
            console.info(`[voice] S${sessionCountRef.current} VAD client-stop: silenceElapsed=${vadNow - vadRef.current.silenceStart}ms, timeout=${vadSilenceTimeoutRef.current}ms, rms=${rms.toFixed(4)}, speechThresh=${vadRef.current.speechThreshold.toFixed(4)}, silenceThresh=${vadRef.current.silenceThreshold.toFixed(4)}, trackAlive=${srcAlive}, trackMuted=${srcMuted}, ctx=${ctx.state}`);
            // Persistent mode: treat as one final segment boundary then reset.
            // One-shot mode is handled below via decideAutoStop — keeps the
            // server-VAD SSOT precedence centralised.
            if (persistentModeRef.current) {
              const provider = providerRef.current;
              if (provider && "sendSegmentBoundary" in provider) {
                (provider as PcmVoiceStreamProvider).sendSegmentBoundary();
              }
              vadRef.current.state = "waitingForSpeech";
              vadRef.current.recordingStart = vadNow;
              vadRef.current.segmentBoundaryEmitted = false;
            }
          } else if (result === "no-speech") {
            console.info(`[voice] S${sessionCountRef.current} VAD no-speech after ${vadNow - vadRef.current.recordingStart}ms, rms=${rms.toFixed(4)}`);
            vadActiveRef.current = false;
            vadRef.current.state = "idle";
            endCaptureRef.current?.({ reason: "auto" });
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
              console.info(`[voice] S${sessionCountRef.current} auto-stop source=${verdict.source} serverAge=${serverAge}ms serverSilence=${serverSnap.silenceElapsedMs}/${serverSnap.silenceTimeoutMs}ms clientResult=${result ?? "null"} rms=${rms.toFixed(4)} trackAlive=${trackAlive} trackMuted=${trackMuted} ctx=${ctx.state}`);
              // A CLIENT-VAD silence verdict is only trustworthy when the audio
              // source is actually delivering samples. Under kyutai/passthrough
              // the server emits no VAD, so client VAD is the SOLE stop authority
              // (decideAutoStop §2) — if the analyser reads silence because the
              // track muted or the AudioContext suspended, that "silence" is an
              // ARTIFACT, not a real pause. Suppressing the false stop keeps the
              // turn alive (the track handlers / no-audio watchdog own true
              // terminal loss) and we try to wake a suspended context.
              if (verdict.source === "client-fallback" && !srcHealthy) {
                if (ctx.state === "suspended") void ctx.resume().catch(() => {});
                if (!unhealthyStopSuppressed) {
                  console.warn("[voice] S%d auto-stop SUPPRESSED: client-VAD silence but audio source not delivering (trackAlive=%s trackMuted=%s ctx=%s) — artifact, not a real pause; keeping the turn open",
                    sessionCountRef.current, trackAlive, trackMuted, ctx.state);
                  unhealthyStopSuppressed = true;
                }
              } else {
                endCaptureRef.current?.({ reason: "auto" });
              }
            }
          }
        }

        if (!levelMonitorActiveRef.current) return;

        // `audioLevel` on the snapshot is the display value; `rms` stays raw so
        // consumers that want the underlying signal (and the thresholds it is
        // judged against) still have it.
        const voiceActivity = buildVoiceActivitySnapshot({
          vadActive: vadActiveRef.current,
          vad: vadRef.current,
          rms,
          audioLevel: meterLevel,
          nowMs: vadNow,
          silenceTimeoutMs: vadSilenceTimeoutRef.current,
          voiceMode: persistentModeRef.current ? "persistent" : "one-shot",
        });
        setState((s) => {
          if (Math.abs(s.audioLevel - meterLevel) < 0.01 && voiceActivitySnapshotsEqual(s.voiceActivity, voiceActivity)) {
            return s;
          }
          return { ...s, audioLevel: meterLevel, voiceActivity };
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

  // Capture-machinery teardown invoked by the controller after every provider
  // shutdown. Resets timers, VAD, the level monitor, and the cue-session guard
  // (without playing a stop cue — shutdown is not a user-initiated stop). Pure
  // machinery reset: callers set `voiceState` to the appropriate value. Idempotent.
  captureTeardownRef.current = (_reason: MicReleaseReason) => {
    if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    cueSessionActiveRef.current = false;
    stopLevelMonitor();
    // The capture session is over — release the iOS audio session shortly after,
    // instead of holding a running-but-idle AudioContext forever (which keeps
    // other apps' audio ducked). Deferred so a trailing stop-cue can finish; a
    // new capture resumes and cancels this. DOC: #audiocontext-lifecycle
    armIdleSuspend();
  };

  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
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
          cause: "speaker-rejected",
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

  /**
   * Surface a recoverable banner for a turn that ended with NO usable text and
   * no speaker rejection — the "silently lost everything I said" case. If the
   * provider retained the turn's audio, offer a one-tap retry that re-runs the
   * full audio through the plain HTTP batch path (this recovers messages the
   * streaming path drops when a mid-turn reconnect loses server-side context).
   * If no audio was retained, fall back to a transient mic-button error so the
   * loss is at least visible instead of silent.
   */
  const surfaceEmptyTranscript = useCallback(() => {
    const audio = providerRef.current?.getLastTurnAudio() ?? null;
    if (!audio) {
      setState((s) => ({ ...s, error: "No speech detected — tap to try again" }));
      return;
    }
    const id = generateRejectionId();
    const rejection: VoiceRejection = {
      kind: "retryable",
      cause: "empty-transcript",
      id,
      blob: audio.blob,
      mimeType: audio.mimeType,
      durationMs: audio.durationMs,
      score: 0,
      threshold: 0,
      createdAt: Date.now(),
      status: "idle",
    };
    rejectedAudioRef.current = rejection;
    setState((s) => ({ ...s, rejectedAudio: rejection }));

    // Same retention TTL as a speaker rejection — release the audio if the
    // user neither retries nor dismisses.
    if (rejectionTtlTimerRef.current) clearTimeout(rejectionTtlTimerRef.current);
    rejectionTtlTimerRef.current = setTimeout(() => {
      if (rejectedAudioRef.current?.id === id) {
        providerRef.current?.disposeLastTurn();
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
  // DOC: docs/internal/VOICE-LATENCY.md#audiocontext-lifecycle
  useEffect(() => {
    if (!voiceEnabled) {
      setState((s) => ({ ...s, supported: false, backend: "none" }));
      return;
    }

    // Show button immediately -- optimistic default assumes Whisper.
    setState((s) => ({ ...s, supported: true, backend: "whisper" }));

    // NOTE: we deliberately do NOT pre-create/resume the AudioContext on an
    // arbitrary first gesture. Resuming it activates the iOS audio session and
    // interrupts other apps' audio even when the user never uses voice; the
    // context is instead resumed lazily inside the real capture/cue gesture.
    // DOC: docs/internal/VOICE-LATENCY.md#audiocontext-lifecycle

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
            controller.ensure(() => new services.PcmVoiceStreamProvider());
            if (isStreamingProvider(providerRef.current)) {
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

      console.info("[voice] Backend: none (durable audio path unavailable)");
      setState((s) => s.voiceState !== "idle"
        ? { ...s, supported: false, backend: "none", error: "Durable audio path unavailable" }
        : { ...s, supported: false, backend: "none", error: "Durable audio path unavailable", fallbackNotice: "Voice input is unavailable because audio-tools cannot be reached." });
    })();

    // Background capability refresh — keeps the snapshot warm so
    // startRecording() never needs to await a network call.
    bgRefreshInterval = setInterval(() => {
      capabilityCheckRef.current().catch(() => {});
    }, 25_000);

    return () => {
      cancelled = true;
      if (bgRefreshInterval) clearInterval(bgRefreshInterval);
      // Single-authority shutdown: disposes the provider (releasing its mic
      // lease), cancels any in-flight start, and runs capture teardown (clears
      // the cue-session guard WITHOUT playing a stop cue — unmount is a
      // lifecycle event, not a user-initiated stop, so a chime would mislead).
      controller.shutdown("unmount");
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

  const prepareRecording = useCallback(() => {
    micIntentArmedRef.current = true;
    if (!voiceEnabled || documentHiddenRef.current || isActiveRef.current || startingRef.current) return;
    // Mic-control intent: reconcile passive wake-word arming. We do NOT pre-warm
    // (hold) the mic here — the recorder acquires it on press. Holding the mic
    // idle is the audio-session/ducking anti-pattern we removed with low-latency.
    reconcilePassiveRef.current?.();
  }, [voiceEnabled]);

  const startRecording = useCallback(async (startOpts?: StartRecordingOpts) => {
    micIntentArmedRef.current = true;
    if (state.voiceState !== "idle" || startingRef.current) return;
    startingRef.current = true;
    stopRequestedRef.current = false;
    // Generation token for this start. A lifecycle shutdown (tab hidden, unmount)
    // calls controller.cancelStarts(), invalidating the token; when the async
    // start resolves we compare and release any late-acquired lease instead of
    // entering the recording state. DOC: plan Phase 6 — preparing/start cancel.
    const startToken = controller.beginStart();
    sessionCountRef.current++;
    // New turn: no text delivered yet. Drives the silent-loss guard in onResult.
    turnDeliveredTextRef.current = false;
    // No transcript state carried across turns.
    transcriptBufferRef.current.reset();
    // Tear down background wake-word listening (if armed) so the recorder owns
    // the mic cleanly. Wake-word detection also funnels through here and has
    // already disposed its listener, so this is a no-op in that path. The
    // auto-arm effect re-arms passive listening once this turn returns to idle.
    if (passiveListenerRef.current) {
      passiveListenerRef.current.dispose("owner-replaced");
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
    dismissedFallbackNoticeRef.current = null;
    setState((s) => ({
      ...s,
      voiceState: "preparing",
      error: null,
      fallbackNotice: null,
      streamingDegradationNotice: null,
    }));

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
      keepAudioContextAwake(); // real capture — cancel any pending idle-suspend
      const ctx = getSharedAudioContext();
      if (ctx.state !== "running") {
        console.info("[voice] S%d Resuming AudioContext (state=%s) in gesture context", sessionCountRef.current, ctx.state);
        ctx.resume().catch(() => {});
      }
    } catch { /* AudioContext unavailable */ }

    // Determine the mode for this session. Persistent mode is an explicit
    // long-form reliability contract: it must never silently become a
    // one-shot/buffered turn when the durable streaming path is unavailable.
    const isPersistent = persistentModeRef.current;

    // A user can press the control before the mount probe's promise resolves.
    // Resolve that probe on the explicit long-form start path so the initial
    // `false` ref value is never mistaken for a real streaming outage.
    if (!capCheckResolvedRef.current) {
      try {
        const probe = await capabilityCheckRef.current();
        streamingAvailableRef.current = probe.whisperHealthy && (probe.streamingAvailable ?? true);
        capCheckResolvedRef.current = true;
        if (!probe.whisperHealthy) {
          setState((s) => ({ ...s, supported: false, backend: "none", error: "Durable audio path unavailable" }));
        }
      } catch {
        capCheckResolvedRef.current = true;
        streamingAvailableRef.current = false;
      }
    }

    if (!controller.isCurrentStart(startToken)) {
      controller.shutdown("hidden");
      setState((s) => s.voiceState === "preparing" ? { ...s, voiceState: "idle" } : s);
      return;
    }

    const persistentDecision = decidePersistentMode(isPersistent, backendRef.current, streamingAvailableRef.current);
    if (!persistentDecision.allowed) {
      const reason = persistentDecision.reason ?? PERSISTENT_STREAMING_UNAVAILABLE_MESSAGE;
      console.warn("[voice] %s", reason);
      controller.shutdown("provider-error");
      setState((s) => ({
        ...s,
        voiceState: "idle",
        error: reason,
        fallbackNotice: reason,
        streamingDegradationNotice: "Streaming is unavailable. No buffered or one-shot fallback was selected.",
      }));
      return;
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
          controller.shutdown("provider-error");
          setState((s) => ({ ...s, supported: false, backend: "none", error: "Durable audio path unavailable", fallbackNotice: "Voice input is unavailable because audio-tools cannot be reached." }));
        } else {
          streamingAvailableRef.current = probe.streamingAvailable ?? true;
          setState((s) => ({ ...s, supported: true, backend: "whisper", error: null, fallbackNotice: null }));
        }
      }).catch(() => {});

      // Lazily create the durable provider on first use. All assignment goes
      // through the controller so a failed provider can never be orphaned.
      if (!providerRef.current) {
        if (backendRef.current === "whisper") {
          controller.set(streamingAvailableRef.current
            ? new services.PcmVoiceStreamProvider()
            : new services.WhisperProvider());
          console.info("[voice] Provider:", streamingAvailableRef.current ? "PCMVoiceStreamV2" : "WhisperHTTP");
        } else {
          const reason = "Voice input is unavailable because audio-tools cannot be reached.";
          setState((s) => ({ ...s, supported: false, voiceState: "idle", backend: "none", error: reason, fallbackNotice: reason }));
          return;
        }
      }

      // Set language from opts. Provider assignment flows through the controller
      // (an opaque function call), so re-assert non-null here for the type system
      // and as a defensive guard against the unreachable backend === "none" path.
      const provider = providerRef.current;
      if (!provider) {
        setState((s) => s.voiceState === "preparing" ? { ...s, voiceState: "idle" } : s);
        return;
      }
      const langCode = voiceLanguage === "auto" ? "" : (voiceLanguage.split("-")[0] ?? "en");
      if ("language" in provider) provider.language = langCode;

      // Wire up segment-final handler for persistent mode
      if (isStreamingProvider(provider)) {
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
        console.info("[voice] S%d onResult: %d chars, vadActive=%s, vadState=%s, delivered=%s",
          sessionCountRef.current, text.length, vadActiveRef.current, vadRef.current.state, turnDeliveredTextRef.current);
        // Capture before surfacePendingRejection() consumes the pending flag:
        // a speaker rejection owns this turn's banner, so we must not also
        // raise an empty-transcript banner for the same turn.
        const hadPendingRejection = pendingRejectionRef.current !== null;
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
        // A queued coalesced partial render must not repaint stale interim text
        // after the turn cleared it.
        cancelPartialRender();
        setState((s) => ({
          ...s,
          voiceState: "idle",
          error: null,
          audioLevel: 0,
          voiceActivity: IDLE_VOICE_ACTIVITY,
          fallbackNotice: null,
          partialTranscript: "",
          segments: [],
        }));
        dismissedFallbackNoticeRef.current = null;
        // Turn ended — surface any pending rejection as a persistent banner.
        // At this point the streaming provider has already snapshotted its
        // retained audio (see VoiceStreamProvider.stop), so
        // getLastTurnAudio() returns the full turn for the retry action.
        surfacePendingRejection();
        // In persistent mode, segment-finals deliver text incrementally.
        // The final message contains only the un-segmented tail (speech
        // after the last segment boundary). Deliver it if non-empty.
        const deliveredResult = transcriptBufferRef.current.result(text);
        if (deliveredResult) {
          onTranscriptRef.current(deliveredResult);
          turnDeliveredTextRef.current = true;
        }

        // Promote a teardown-raced partial exactly once. TranscriptBuffer owns
        // the committed-prefix cursor, so normal segment-finals cannot be
        // appended again here.
        const promoted = transcriptBufferRef.current.promoteTurnEnd();
        if (promoted !== null) {
          onTranscriptRef.current(promoted);
          turnDeliveredTextRef.current = true;
        }

        // Silent-loss guard: the turn ended but nothing was ever delivered
        // (empty final, empty HTTP fallback, or a mid-turn reconnect that lost
        // server context). Unless speaker verification already owns this turn's
        // banner, surface a recoverable "couldn't transcribe" notice so the
        // user isn't left staring at an idle mic with their message gone.
        if (!turnDeliveredTextRef.current && !hadPendingRejection) {
          surfaceEmptyTranscript();
        }

        // controller.shutdown runs capture teardown, which stops the mic tracks
        // and arms the idle AudioContext suspend — so the audio session is fully
        // released after the turn (no held mic, no ducking).
        controller.shutdown("manual-stop");
      };
      provider.onError = (error) => {
        // A provider error ENDS the turn (mic stops). Log the triggering reason
        // so an abrupt mid-speech stop is attributable to the actual cause
        // (backend error frame, mic-track loss, encoder failure) rather than
        // looking like a spontaneous stop.
        console.warn("[voice] S%d provider.onError → ending turn: %s",
          sessionCountRef.current, error);
        // Clear cue session without playing stop cue — errors are not normal
        // recording stops. Playing a pleasant "done" chime after an error
        // would be misleading.
        cueSessionActiveRef.current = false;
        if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
        vadActiveRef.current = false;
        vadRef.current.state = "idle";
        stopLevelMonitor();
        cancelPartialRender();
        // Error ends the turn: if speaker verification rejected segments
        // during this turn, surface the banner so the user can still retry
        // with whatever audio was retained.
        surfacePendingRejection();

        // Whisper failed after retry. Refuse the weaker browser-only path;
        // release the mic and preserve an explicit terminal reason.
        if (error === WHISPER_FAILED_SENTINEL) {
          controller.shutdown("provider-error");
          setState((s) => ({
            ...s,
            voiceState: "idle",
            error: "Durable transcription path failed; audio was retained for recovery",
            audioLevel: 0,
            voiceActivity: IDLE_VOICE_ACTIVITY,
            backend: "none",
            supported: false,
            fallbackNotice: "Voice input stopped because the durable audio path failed. Retry after audio-tools recovers.",
          }));
          return;
        }

        // Generic error terminal path: dispose the errored provider (releasing
        // its mic lease) before returning the UI to idle.
        controller.shutdown("provider-error");
        setState((s) => ({
          ...s,
          voiceState: "idle",
          error,
          audioLevel: 0,
          voiceActivity: IDLE_VOICE_ACTIVITY,
          fallbackNotice: null,
        }));
        dismissedFallbackNoticeRef.current = null;
      };
      if (provider.onPartial !== undefined) {
        provider.onPartial = (text) => {
          dismissedFallbackNoticeRef.current = null;
          const transcript = transcriptBufferRef.current.partial(text);
          // A partial proves the stream is alive, so it retires any streaming
          // notice — but only when one is actually up. Writing `fallbackNotice:
          // null` unconditionally allocated a new state object per partial and,
          // paired with a status that re-set it, drove the notice on and off at
          // the partial rate.
          const applyPartial = (interimText: string) => (s: VoiceInputState): VoiceInputState =>
            s.partialTranscript === interimText && s.fallbackNotice === null
              ? s
              : { ...s, partialTranscript: interimText, fallbackNotice: null };
          // Coalesce the RENDER to one paint per frame: at a high partial rate,
          // a setState per partial janks the main thread and re-introduces
          // client-side backpressure. The latest pending text wins.
          setState(applyPartial(transcript.interimText));
          pendingPartialRenderRef.current = transcript.interimText;
          if (typeof requestAnimationFrame !== "function") {
            setState(applyPartial(transcript.interimText));
            return;
          }
          if (partialRenderRafRef.current === 0) {
            partialRenderRafRef.current = requestAnimationFrame(() => {
              partialRenderRafRef.current = 0;
              const pending = pendingPartialRenderRef.current;
              pendingPartialRenderRef.current = null;
              if (pending !== null) {
                setState(applyPartial(pending));
              }
            });
          }
        };
      }
      if (provider.onStatus !== undefined) {
		provider.onStatus = ({ code, message }: { code: string; message: string }) => {
          if (SILENT_STREAM_STATUS.has(code)) return;

          if (DEGRADATION_STREAM_STATUS.has(code)) {
            setState((s) => ({
              ...s,
              streamingDegradationNotice: message || "Streaming degraded — buffered mode is active for this transcription.",
            }));
          }
          if (NOTICE_CLEARING_STREAM_STATUS.has(code)) {
            setState((s) => (s.fallbackNotice ? { ...s, fallbackNotice: null } : s));
            dismissedFallbackNoticeRef.current = null;
            return;
          }
          // No copy means the emitter had nothing to tell the operator.
          if (!message) return;
          if (dismissedFallbackNoticeRef.current === message) return;
          // Re-asserting the same notice is not a change; returning `s`
          // unchanged keeps a repeating status from re-rendering the host.
          setState((s) => (s.fallbackNotice === message ? s : { ...s, fallbackNotice: message }));
        };
      }
      if (provider.onDiagnostic !== undefined) {
        provider.onDiagnostic = (diagnostic) => {
          setState((s) => ({ ...s, turnDiagnostic: diagnostic }));
        };
      }

      // The provider acquires its own mic stream on start (single owner, fresh
      // getUserMedia). We no longer inject a pre-warmed stream — holding the mic
      // idle to save start latency was the audio-session/ducking anti-pattern.
      const providerStartTime = Date.now();
      try {
        await provider.start();
      } catch (error: unknown) {
        // Provider startup owns the microphone acquisition boundary. A
        // rejected getUserMedia/capture setup promise must never leave the
        // shared hook stuck in `preparing` (which otherwise looks like a
        // hung microphone and also prevents the diagnostic channel from
        // reaching a terminal state).
        const message = describeProviderStartFailure(error);
        console.warn("[voice] S%d provider.start() failed after %dms: %s",
          sessionCountRef.current, Date.now() - providerStartTime, message);
        provider.onError?.(message);
        if (!provider.onError) {
          controller.shutdown("provider-error");
          setState((s) => ({
            ...s,
            voiceState: "idle",
            audioLevel: 0,
            voiceActivity: IDLE_VOICE_ACTIVITY,
            error: message,
            fallbackNotice: null,
          }));
        }
        return;
      }
      console.info("[voice] Provider.start() took %dms (includes getUserMedia)", Date.now() - providerStartTime);

      // ── Late-resolve cancellation (Phase 6) ──
      // A lifecycle shutdown (tab hidden / unmount) may have fired while we were
      // awaiting start(). getUserMedia can have already returned a live lease, so
      // simply not entering the recording state would leak the mic. Compare the
      // generation token; if stale, shut the provider down (which releases the
      // just-acquired lease) and bail before showing any capture UI.
      if (!controller.isCurrentStart(startToken)) {
        console.info("[voice] S%d start resolved stale (cancelled during preparing) — releasing mic", sessionCountRef.current);
        controller.shutdown("hidden");
        setState((s) => s.voiceState === "preparing" ? { ...s, voiceState: "idle" } : s);
        return;
      }

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
            console.info(`[voice] Noise floor cache: loaded (age=${Math.round(cacheAge / 1000)}s, floor=${(cached.silenceThreshold / 1.5).toFixed(4)})`);
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
        services.playRecordingStartCue();
        startLevelMonitor(stream);

        // Warn if no audio detected after 2s (catches dead/muted mics).
        // Use a small threshold, not strict ===0: a wedged AudioContext or a
        // muted track yields near-silence that is not bit-exactly zero, which
        // is why the old ===0 guard silently missed the "stuck mic" state and
        // showed no error. When we can tell the track is muted, say so —
        // that's the actionable cause (device taken by another app / changed).
        if (noAudioTimerRef.current) clearTimeout(noAudioTimerRef.current);
        noAudioTimerRef.current = setTimeout(() => {
          if (audioLevelRef.current < NO_AUDIO_LEVEL_THRESHOLD) {
            const muted = (stream.getAudioTracks?.() ?? []).some((t) => t.muted);
            const message = muted
              ? "Microphone is muted or in use by another app — reload to recover"
              : "No audio detected — check your microphone";
            setState((s) => (s.voiceState === "recording" || s.voiceState === "listening")
              ? { ...s, error: message }
              : s);
            console.warn(`[voice] No audio after 2s (level=${audioLevelRef.current.toFixed(4)}, muted=${muted})`);
          }
        }, 2000);

        // If stop was requested during async start, abort immediately.
        // The start cue already played above, so play the matching stop cue
        // to keep the pair balanced.
        if (stopRequestedRef.current) {
          stopRequestedRef.current = false;
          if (cueSessionActiveRef.current) {
            cueSessionActiveRef.current = false;
            services.playRecordingStopCue();
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
          // Single-authority shutdown: disposes the provider (releasing its mic
          // lease) and runs capture teardown. The stop cue already played above.
          controller.shutdown("manual-stop");
          return;
        }
      } else {
        setState((s) => s.voiceState === "preparing" ? { ...s, voiceState: "idle" } : s);
      }
    } finally {
      startingRef.current = false;
    }
  }, [state.voiceState, state.backend, voiceLanguage, startLevelMonitor, stopLevelMonitor, handleSegmentFinal, surfacePendingRejection, cancelPartialRender]);

  const endCapture = useCallback((opts?: { reason?: "auto" | "user" }) => {
    const reason = opts?.reason ?? "user";
    // If start is in progress, signal it to abort after completing
    if (startingRef.current) {
      stopRequestedRef.current = true;
      console.info("[voice] S%d endCapture: deferred (start in progress)", sessionCountRef.current);
      return;
    }

    const provider = providerRef.current;
    if (!provider || !isActive) {
      console.warn("[voice] S%d endCapture: no-op (provider=%s, isActive=%s)",
        sessionCountRef.current, !!provider, isActive);
      return;
    }

    console.info("[voice] S%d %s stopped (reason=%s)", sessionCountRef.current, isListening ? "Listening" : "Recording", reason);
    // Only play the stop cue if a cue session is active (start cue was played).
    // This prevents the stop sound from firing during cleanup, error recovery,
    // or any other path that disposes the provider without a preceding start cue.
    if (cueSessionActiveRef.current) {
      cueSessionActiveRef.current = false;
      services.playRecordingStopCue();
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
      console.info(`[voice] S${sessionCountRef.current} Noise floor saved (speech=${floor.speechThreshold.toFixed(4)}, silence=${floor.silenceThreshold.toFixed(4)})`);
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

  const stopRecording = useCallback((opts?: { reason?: "auto" | "user" }) => {
    endCapture(opts ?? { reason: "user" });
  }, [endCapture]);

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
    if (provider.onStatus !== undefined) provider.onStatus = null;
    if (isStreamingProvider(provider)) {
      provider.onSegmentFinal = null;
      provider.onSegmentAccepted = null;
      provider.onSegmentRejected = null;
      provider.onSpeakerStatus = null;
    }
    // Single-authority shutdown disposes the provider (releasing its mic lease)
    // and runs capture teardown. Callbacks were nulled above so dispose is silent.
    controller.shutdown("manual-stop");
    cancelPartialRender();

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
  }, [isTranscribing, stopLevelMonitor, cancelPartialRender]);

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

  const dismissFallbackNotice = useCallback(() => {
    setState((s) => {
      dismissedFallbackNoticeRef.current = s.fallbackNotice;
      return s.fallbackNotice ? { ...s, fallbackNotice: null } : s;
    });
  }, []);

  /**
   * Retry transcription of the retained audio. The exact path depends on why
   * the banner appeared (`rejection.cause`): a speaker-rejected turn re-runs
   * with the verification filter bypassed; an empty-transcript turn re-runs
   * the full audio through the plain batch endpoint. No-op if the current
   * rejection has no retained audio (explanatory kind) or a retry is already
   * in flight.
   *
   * On success the transcript is delivered via the normal `onTranscript`
   * callback and the banner is dismissed. On failure the banner flips to
   * `status: "failed"` with an error message; the user can retry or dismiss.
   *
   * Named `retryWithoutFilter` for historical reasons (the speaker-rejection
   * path was first); it now serves both causes.
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
      // Pick the retry path by cause: an empty-transcript turn re-runs the
      // FULL retained audio through the plain HTTP batch endpoint (recovering
      // what a dropped streaming session lost); a speaker-rejected turn
      // re-runs with the verification filter bypassed for this one request.
      const text = current.cause === "empty-transcript"
        ? await services.transcribeAudio(current.blob, lang)
        : await services.transcribeAudioBypassFilter(current.blob, lang);
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

  // Keep the ref in sync so all async end triggers use the same capture funnel.
  endCaptureRef.current = endCapture;

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
    // Never open a background mic while hidden (iOS-PWA background-mic leak);
    // the visibility handler re-arms on becoming visible.
    if (documentHiddenRef.current) return;
    if (passiveListenerRef.current) {
      passiveListenerRef.current.dispose("owner-replaced");
      passiveListenerRef.current = null;
    }

    // Pick up the latest sensitivity before arming (the user may have moved the
    // slider since the template loaded). The sync effect keeps it current after.
    wakeWordTemplateRef.current.threshold = wakeWordThresholdRef.current;

    // Reuse the shared AudioContext. Passive listening is a real audio need, so
    // keep it awake (cancel any idle-suspend) and resume it — the analyser reads
    // silence on a suspended context, so passive VAD would never fire otherwise.
    keepAudioContextAwake();
    const sharedCtx = audioCtxRef.current ?? getSharedAudioContext();
    audioCtxRef.current = sharedCtx;
    if (sharedCtx.state === "suspended") sharedCtx.resume().catch(() => {});

    const listener = new services.PassiveListener({
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
        listener.dispose("owner-replaced");
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
      // Fires whenever the mic lease is released for ANY reason (our dispose,
      // the OS revoking the device, or page-hidden emergency cleanup). Drives
      // the honest passive state false so the mic control never shows idle while
      // a stream was live.
      onMicReleased: () => {
        if (passiveListenerRef.current === listener) passiveListenerRef.current = null;
        setState((s) => (s.passiveListeningActive ? { ...s, passiveListeningActive: false } : s));
      },
    });

    passiveListenerRef.current = listener;
    await listener.start();
    // Reflect successful arming in honest UI state. If start failed, onError /
    // onMicReleased already cleared the ref, so this guard skips the update.
    if (passiveListenerRef.current === listener) {
      setState((s) => (s.passiveListeningActive ? s : { ...s, passiveListeningActive: true }));
    }
  }, [startRecording]);

  /** Stop background passive listening (does not touch voiceState). */
  const exitPassiveMode = useCallback(() => {
    if (passiveListenerRef.current) {
      // dispose() releases the lease → onMicReleased clears the ref + state.
      passiveListenerRef.current.dispose("manual-stop");
      passiveListenerRef.current = null;
    }
  }, []);

  // Release the background passive listener (and its mic stream) on unmount.
  // The main unmount cleanup only disposes the streaming provider; without
  // this, navigating away while passively listening would leak an open mic.
  useEffect(() => () => {
    if (passiveListenerRef.current) {
      passiveListenerRef.current.dispose("unmount");
      passiveListenerRef.current = null;
    }
    // Drop any scheduled coalesced partial render so it can't paint after unmount.
    cancelPartialRender();
  }, [cancelPartialRender]);

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
  // Nothing else in the app starts passive mode, so without this the "always-on"
  // wake word never actually listens — the toggle flips a config bit but the mic
  // is never opened. We reconcile the desired state: when wake word is enabled,
  // a template is loaded, voice is idle, AND the document is visible, listen
  // passively in the background (voiceState stays "idle" — tap-to-talk still
  // works; the honest `passiveListeningActive` flag drives the mic control). On
  // detection enterPassiveMode routes to startRecording; when that turn ends
  // voiceState returns to "idle" and we re-arm. Disabling the toggle (or a start
  // failure) tears it down. The reconcile is also called by the visibility
  // handler on becoming visible (lease release does not re-run React effects).
  const reconcilePassive = useCallback(() => {
    // Clear the failure latch whenever the feature is off so a later re-enable
    // gets a fresh start attempt.
    if (!voiceEnabled || !wakeWordEnabledRef.current) {
      passiveStartBlockedRef.current = false;
    }
    const action = decidePassiveArm({
      voiceEnabled,
      wakeWordEnabled: wakeWordEnabledRef.current,
      wakeWordConfigured: state.wakeWordConfigured,
      voiceState: state.voiceState,
      listenerActive: !!passiveListenerRef.current,
      startBlocked: passiveStartBlockedRef.current,
      documentVisible: micIntentArmedRef.current && !documentHiddenRef.current,
    });
    if (action === "enter") {
      void enterPassiveMode();
    } else if (action === "exit") {
      exitPassiveMode();
    }
  }, [voiceEnabled, state.wakeWordConfigured, state.voiceState, enterPassiveMode, exitPassiveMode]);
  reconcilePassiveRef.current = reconcilePassive;

  useEffect(() => {
    if (micIntentArmedRef.current) reconcilePassive();
  }, [voiceEnabled, wakeWordEnabled, reconcilePassive]);

  // ── Page-lifecycle mic cleanup + coordinated re-arm ──
  // DOC: docs/internal/VOICE-LATENCY.md#visibility-based-mic-lifecycle
  //
  // One handler for ALL mic owners:
  //   - Install the registry's privacy backstop, which releases every
  //     non-active lease on tab-hidden and ALL leases on pagehide/freeze even
  //     if React cleanup never runs (mobile PWA close). Passive leases reset
  //     their owners via onRelease.
  //   - Suspend the shared AudioContext on background so the iOS audio session
  //     is released (a running-but-idle context keeps other apps' audio ducked).
  //   - For iOS-PWA privacy, an ACTIVE recording is stopped on hidden (it is not
  //     a registry lease concern — stopping it keeps UI state honest and
  //     releases the mic). Surfaced via a transient notice.
  //   - On becoming visible, do not re-arm passive listening or the audio
  //     session; visibility alone is not a mic intent.
  useEffect(() => {
    if (!voiceEnabled) return;
    // Platform-aware backstop: a standalone/PWA releases ALL leases on hidden
    // (iOS keeps the OS mic indicator on otherwise); a desktop tab releases only
    // non-active leases and lets the controller stop the active recording. The
    // resolver is read on every event so display-mode can change at runtime.
    const resolveScope = (event: "hidden" | "pagehide" | "freeze"): LifecycleReleaseScope =>
      decideMicLifecycle({ event, standalonePwa: isStandaloneDisplayMode() }).release === "all"
        ? "all"
        : "non-active";
    const uninstallBackstop = installMicLifecycleCleanup(resolveScope);

    const onVisibility = () => {
      const hidden = typeof document !== "undefined" && document.visibilityState === "hidden";
      documentHiddenRef.current = hidden;
      if (hidden) {
        // Abort any in-flight start so a getUserMedia that resolves after we go
        // hidden releases its lease instead of entering recording (Phase 6).
        micIntentArmedRef.current = false;
        controller.cancelStarts();
        if (isActiveRef.current) {
          console.info("[voice] Visibility: hidden during active recording — stopping (privacy)");
          endCaptureRef.current?.({ reason: "auto" });
          if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
          setState((s) => ({ ...s, fallbackNotice: "Recording stopped — app moved to background" }));
          fallbackTimerRef.current = setTimeout(() => {
            setState((s) => (s.fallbackNotice ? { ...s, fallbackNotice: null } : s));
          }, 5000);
        }
        // Passive leases are released by the backstop; their onRelease callbacks
        // reset passiveListeningActive. Release the audio session too: suspend
        // the shared AudioContext so backgrounding stops holding other apps'
        // audio. It resumes on demand on the next in-gesture voice/cue use.
        suspendSharedAudioContext();
      } else {
        // Visibility alone must not acquire the microphone or the audio session.
        // A new explicit mic-control intent will re-arm passive if needed.
      }
    };

    document.addEventListener("visibilitychange", onVisibility);
    documentHiddenRef.current = typeof document !== "undefined" && document.visibilityState === "hidden";
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      uninstallBackstop();
    };
  }, [voiceEnabled]);

  // ── Registry-driven UI honesty + stale-lease self-healing (Phase 4) ──
  // DOC: docs/internal/INVARIANTS.md#voice-input-ui-invariants
  //
  // Hardware truth is the mic ownership registry, not `voiceState`. Subscribe to
  // lease acquire/release: if a live lease exists that the workflow should not
  // be holding (UI idle/off but a provider/prewarm/passive stream is still live),
  // flip the honest `staleLiveMicLease` flag AND self-heal by releasing it. The
  // re-entrancy guard is required because recovery releases leases, which
  // re-notifies this same listener synchronously.
  useEffect(() => {
    if (!voiceEnabled) return;
    const recovery = { inProgress: false };
    const evaluate = (snapshots: MicLeaseSnapshot[]) => {
      // `voiceStateRef` is render-driven and lags a synchronous `setState`. While
      // a start is in flight, getUserMedia can return a recording lease before
      // the "preparing" render commits — treat an in-flight start as "preparing"
      // so a legitimate fresh lease is never mistaken for an orphan and recovered.
      const effectiveState = startingRef.current ? "preparing" : voiceStateRef.current;
      const staleInput = {
        leases: snapshots.map((s) => ({ id: s.id, owner: s.owner })),
        voiceState: effectiveState,
        passiveListenerActive: !!passiveListenerRef.current,
      };
      const hasStale = selectStaleLeases(staleInput).length > 0;
      setState((s) => (s.staleLiveMicLease === hasStale ? s : { ...s, staleLiveMicLease: hasStale }));
      if (hasStale && !recovery.inProgress) {
        recovery.inProgress = true;
        try {
          controller.recoverStaleLeases({
            voiceState: effectiveState,
            passiveListenerActive: !!passiveListenerRef.current,
            reason: "invariant-violation",
          });
        } finally {
          recovery.inProgress = false;
        }
      }
    };
    evaluate(getActiveMicLeases());
    return subscribeMicLeases(evaluate);
  }, [voiceEnabled, state.voiceState]);

  // User-triggered "release microphone" recovery (mic control affordance). Frees
  // any orphaned lease and tears down active capture; safe to call anytime.
  const releaseMicrophone = useCallback(() => {
    controller.recoverStaleLeases({
      voiceState: voiceStateRef.current,
      passiveListenerActive: !!passiveListenerRef.current,
      reason: "recovery",
    });
    controller.shutdown("recovery");
    setState((s) => ({
      ...s,
      voiceState: "idle",
      staleLiveMicLease: false,
      audioLevel: 0,
      voiceActivity: IDLE_VOICE_ACTIVITY,
      partialTranscript: "",
    }));
  }, [controller]);

  /** Returns a privacy-safe diagnostic JSON document, never audio or text. */
  const exportTurnDiagnostic = useCallback((): string | null => {
    const provider = providerRef.current;
    return provider?.exportDiagnostic?.() ?? null;
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
    prepareRecording,
    startRecording,
    stopRecording,
    cancelTranscription,
    dismissCommandSuggestion,
    dismissFallbackNotice,
    dismissRejection,
    retryWithoutFilter,
    enterPassiveMode,
    exitPassiveMode,
    releaseMicrophone,
    exportTurnDiagnostic,
  };
}
