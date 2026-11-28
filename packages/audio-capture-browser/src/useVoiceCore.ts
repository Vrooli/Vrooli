import { useState, useEffect, useRef, useCallback } from "react";
import { PcmVoiceStreamProvider } from "./pcmVoiceStreamProvider";
import { preloadCanonicalPcmCapture } from "./pcmCapture";
import { IDLE_VOICE_ACTIVITY } from "./voice/activity";
import { createVadRefs, createVadRefsFromCache, extractCacheableFloor, loadNoiseFloorCache, saveNoiseFloorCache, VAD_FLOOR_CACHE_MAX_AGE_MS } from "./voice/vad";
import { getSharedAudioContext, suspendSharedAudioContext, armIdleSuspend, keepAudioContextAwake } from "./voice/sharedAudioContext";
import { createLevelMonitorController } from "./voice/levelMonitor";
import { VoiceCaptureController } from "./voice/voiceCaptureController";
import { installMicLeaseRecovery } from "./voice/micLeaseRecovery";
import type { MicReleaseReason } from "./voice/micOwnership";
import { installVoicePageLifecycle } from "./voice/pageLifecycle";
import { setServerVadState, resetServerVadState, useServerVadStateStore } from "./voice/useServerVadStateStore";
import { decidePassiveArm } from "./voice/passiveArmDecision";
import { decidePersistentMode, PERSISTENT_STREAMING_UNAVAILABLE_MESSAGE } from "./voice/persistentModeDecision";
import { TranscriptBuffer } from "./transcriptBuffer";
import { VoiceRejectionController } from "./voice/rejectionController";
import { StreamNoticeController } from "./voice/streamNoticeController";
import { createCaptureProvider, ensurePcmStreamProvider } from "./voice/providerOrchestration";
import { loadWakeWordTemplate } from "./voice/wakeWordTemplateLoader";
import { PartialTranscriptRenderer } from "./voice/partialTranscriptRenderer";
import { AudioCueSession } from "./voice/audioCueSession";
import { WHISPER_FAILED_SENTINEL } from "./voice/types";
import type { TranscriptionProvider, VoiceBackend, VoiceInputState, VoiceMode, VoiceSegment, CommandSuggestion, StartRecordingOpts } from "./voice/types";
import type { WakeWordEngine, WakeWordTemplate } from "./voice/wakeword";
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
    (provider !== null && typeof (provider as PcmVoiceStreamProvider).isStreamConnected === "function");
}

/** Audio level (0..1) below which the mic is treated as silent by the no-audio
 *  watchdog. Not strict zero: a wedged AudioContext or muted track produces
 *  tiny non-zero noise, which the old ===0 check missed. */
const NO_AUDIO_LEVEL_THRESHOLD = 0.01;

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
  /** Human-readable provider failure detail for an unavailable mic state. */
  capabilityReason?: string;
  /** Operator command that repairs or inspects the unavailable provider. */
  operatorCommand?: string;
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
  // Measure visible speech timing after React commits the transcript, not when
  // the stream callback arrives. This includes frame-coalesced partial renders
  // and durable segment commits, while counting one revision per React commit.
  useEffect(() => {
    if (!state.partialTranscript.trim() && state.segments.length === 0) return;
    const provider = providerRef.current;
    if (isStreamingProvider(provider)) provider.recordVisibleTranscriptRevision();
  }, [state.partialTranscript, state.segments]);

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
  const rejectionControllerRef = useRef<VoiceRejectionController | null>(null);
  if (!rejectionControllerRef.current) {
    rejectionControllerRef.current = new VoiceRejectionController({
      getProvider: () => providerRef.current,
      setState,
      getLanguage: () => voiceLanguageRef.current,
      transcribeAudio: (audio, language) => services.transcribeAudio(audio, language),
      transcribeAudioBypassFilter: (audio, language) => services.transcribeAudioBypassFilter(audio, language),
      onTranscript: (text) => onTranscriptRef.current(text),
    });
  }
  const rejectionController = rejectionControllerRef.current;
  const streamNoticeControllerRef = useRef<StreamNoticeController | null>(null);
  if (!streamNoticeControllerRef.current) {
    streamNoticeControllerRef.current = new StreamNoticeController(setState);
  }
  const streamNoticeController = streamNoticeControllerRef.current;
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
  /** Interim text is replaceable; the renderer owns frame-level coalescing. */
  const partialTranscriptRendererRef = useRef<PartialTranscriptRenderer | null>(null);
  if (!partialTranscriptRendererRef.current) {
    partialTranscriptRendererRef.current = new PartialTranscriptRenderer(setState);
  }
  const partialTranscriptRenderer = partialTranscriptRendererRef.current;

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
  const cueSessionRef = useRef<AudioCueSession | null>(null);
  if (!cueSessionRef.current) {
    cueSessionRef.current = new AudioCueSession({
      start: services.playRecordingStartCue,
      stop: services.playRecordingStopCue,
      fault: services.playRecordingFaultCue,
    });
  }

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

    // The persisted template contains raw samples; the loader re-derives
    // engine features and calibration at startup.
    loadWakeWordTemplate(
      services,
      wakeWordEngineRef.current,
      wakeWordThresholdRef.current,
      (engine) => { wakeWordEngineRef.current = engine; },
    )
      .then(({ template }) => {
        if (!template) return;
        // Threshold is live configuration, not the stale persisted value.
        wakeWordTemplateRef.current = template;
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

  /** Cancel queued interim UI work so stale text cannot paint after turn end. */
  const cancelPartialRender = useCallback(() => {
    partialTranscriptRenderer.cancel();
  }, [partialTranscriptRenderer]);

  /** Session counter for diagnostic logging — helps correlate log lines
   *  across multiple recording sessions within the same component mount. */
  const sessionCountRef = useRef(0);

  const levelMonitorRef = useRef<ReturnType<typeof createLevelMonitorController> | null>(null);
  if (!levelMonitorRef.current) {
    levelMonitorRef.current = createLevelMonitorController({
      services, setState, sessionCountRef, vadRef, vadActiveRef,
      vadSilenceTimeoutRef, persistentModeRef, backendRef, endCaptureRef,
    });
  }
  const levelMonitor = levelMonitorRef.current;
  const startLevelMonitor = levelMonitor.start;
  const stopLevelMonitor = levelMonitor.stop;


  // Capture-machinery teardown invoked by the controller after every provider
  // shutdown. Resets timers, VAD, the level monitor, and the cue-session guard
  // (without playing a stop cue — shutdown is not a user-initiated stop). Pure
  // machinery reset: callers set `voiceState` to the appropriate value. Idempotent.
  captureTeardownRef.current = (_reason: MicReleaseReason) => {
    if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    cueSessionRef.current?.clear();
    stopLevelMonitor();
    // The capture session is over — release the iOS audio session shortly after,
    // instead of holding a running-but-idle AudioContext forever (which keeps
    // other apps' audio ducked). Deferred so a trailing stop-cue can finish; a
    // new capture resumes and cancels this. DOC: #audiocontext-lifecycle
    armIdleSuspend();
  };

  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Retained-audio rejection, retry, expiry, and disposal state belongs to one
  // controller so timer races cannot diverge from the visible banner.

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

    // Warm the streaming transport immediately. This creates no microphone
    // lease and sends no audio; it only opens the provider's stream so
    // a slow capability probe cannot sit between the user's press and the
    // first streaming batch. The probe below still decides whether the host
    // advertises streaming and remains the source of capability truth.
    ensurePcmStreamProvider(controller, () => new services.PcmVoiceStreamProvider());
    if (isStreamingProvider(providerRef.current)) {
      const currentLanguage = voiceLanguageRef.current;
      const lang = currentLanguage === "auto" ? "" : (currentLanguage.split("-")[0] ?? "en");
      providerRef.current.preConnect(lang);
    }

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

          // Pre-connect the bidi stream so it's ready when the user presses
          // the mic button, eliminating 10-100ms of connection latency.
          // DOC: docs/internal/VOICE-LATENCY.md#websocket-pre-connection
          if (streamingAvailableRef.current) {
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
      // Readiness only — deliberately silent.
      //
      // This probe exists to pre-warm the provider, not to tell anyone
      // anything: nobody has asked for audio yet, so an unreachable
      // backend is a capability fact for the host's capability surface to
      // present when it becomes relevant, not a notice this hook may
      // raise. Setting `error` AND `fallbackNotice` here used to put two
      // rows in the host's notice region beside the host's own capability
      // banner — one unreachable backend, three separate notices to
      // dismiss, on every load.
      //
      // A turn already in flight is the exception: the user did ask, so
      // it still gets its error.
      setState((s) => s.voiceState !== "idle"
        ? { ...s, supported: false, backend: "none", error: "Durable audio path unavailable" }
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
      partialTranscriptRenderer.cancel();
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
      rejectionController.clearForUnmount();
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
    const startRequestedAtMs = typeof globalThis.performance?.now === "function"
      ? globalThis.performance.now()
      : Date.now();
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
    streamNoticeController.clearDismissed();
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
      // Begin the AudioWorklet module load while getUserMedia, journal restore,
      // and transport startup proceed. This is gesture-scoped and creates no
      // microphone lease or audio graph; createCanonicalPcmCapture reuses the
      // same in-flight load when the stream is ready.
      void preloadCanonicalPcmCapture(ctx).catch(() => {});
    } catch { /* AudioContext unavailable */ }

    // Determine the mode for this session. Persistent mode is an explicit
    // long-form reliability contract: it must never silently become a
    // one-shot/buffered turn when the durable streaming path is unavailable.
    const isPersistent = persistentModeRef.current;

    if (!controller.isCurrentStart(startToken)) {
      controller.shutdown("hidden");
      setState((s) => s.voiceState === "preparing" ? { ...s, voiceState: "idle" } : s);
      return;
    }

    // The mount probe is deliberately best-effort and may be slow while a
    // provider/resource manager wakes up. It must never sit between the
    // user's press and microphone acquisition: the hook already exposes an
    // optimistic durable voice capability and the stream itself is the
    // authoritative readiness check. If the probe has not resolved yet,
    // choose the streaming path and let a real startup error fail visibly.
    const streamingReady = capCheckResolvedRef.current ? streamingAvailableRef.current : true;
    const persistentDecision = decidePersistentMode(isPersistent, backendRef.current, streamingReady);
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
        // A prior turn's probe may finish after that turn has stopped and a
        // newer capture has started. Its health snapshot must not dispose or
        // overwrite the newer session's provider and UI state.
        if (!controller.isCurrentStart(startToken)) return;
        if (!probe.whisperHealthy) {
          controller.shutdown("provider-error");
          // One fact, one field. `error` is the unreachable-backend
          // channel; `fallbackNotice` is for degradation *during* a
          // turn (see the transport-failure path below). Writing both
          // made the host render the same sentence twice.
          setState((s) => ({ ...s, supported: false, backend: "none", error: "Durable audio path unavailable" }));
        } else {
          streamingAvailableRef.current = probe.streamingAvailable ?? true;
          setState((s) => ({ ...s, supported: true, backend: "whisper", error: null, fallbackNotice: null, capabilityReason: undefined, operatorCommand: undefined }));
        }
      }).catch(() => {});

      // Lazily create the durable provider on first use. All assignment goes
      // through the controller so a failed provider can never be orphaned.
      if (!providerRef.current) {
        if (backendRef.current === "whisper") {
          const selectedProvider = createCaptureProvider(backendRef.current, streamingReady, {
            pcmStream: () => new services.PcmVoiceStreamProvider(),
            whisper: () => new services.WhisperProvider(),
          });
          if (selectedProvider) controller.set(selectedProvider);
          console.info("[voice] Provider:", streamingReady ? "PCMVoiceStreamV2" : "WhisperHTTP");
        } else {
          const reason = "Voice input is unavailable because audio-tools cannot be reached.";
          setState((s) => ({ ...s, supported: false, voiceState: "idle", backend: "none", error: reason }));
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
      provider.markStartRequested?.(startRequestedAtMs);
      const langCode = voiceLanguage === "auto" ? "" : (voiceLanguage.split("-")[0] ?? "en");
      if ("language" in provider) provider.language = langCode;
      // The capability probe is intentionally asynchronous. If the user presses
      // the control before it has completed, admit the stream transport here so
      // the first voiced batch does not pay the stream setup. The provider
      // preserves this identity through start(), which makes the preconnect a
      // real handoff rather than an orphaned stream.
      if (isStreamingProvider(provider)) provider.preConnect(langCode);

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
          rejectionController.markPending(score, threshold);
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
        const hadPendingRejection = rejectionController.hasPending;
        // Clear cue session — the stop cue already played in stopRecording().
        // If onResult fires without stopRecording() (e.g. server-side stop),
        // we still clear the guard to prevent a stale cue on next session.
        cueSessionRef.current?.clear();
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
        streamNoticeController.clearDismissed();
        // Turn ended — surface any pending rejection as a persistent banner.
        // At this point the streaming provider has already snapshotted its
        // retained audio (see VoiceStreamProvider.stop), so
        // getLastTurnAudio() returns the full turn for the retry action.
        rejectionController.surfacePending();
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
          rejectionController.surfaceEmptyTranscript();
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
        // Errors are not normal recording stops, so the "done" chime would
        // misreport the outcome — but silence misreports it worse. The mic has
        // stopped and the countdown ring is gone; a speaker who is not watching
        // the screen has no way to learn the turn ended and keeps talking into a
        // dead capture. Play the distinct fault cue instead: it sounds wrong on
        // purpose, which is the accurate signal.
        cueSessionRef.current?.fault();
        if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
        vadActiveRef.current = false;
        vadRef.current.state = "idle";
        stopLevelMonitor();
        cancelPartialRender();
        // Error ends the turn: if speaker verification rejected segments
        // during this turn, surface the banner so the user can still retry
        // with whatever audio was retained.
        rejectionController.surfacePending();

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
        streamNoticeController.clearDismissed();
      };
      if (provider.onPartial !== undefined) {
        provider.onPartial = (text) => {
          streamNoticeController.clearDismissed();
          const transcript = transcriptBufferRef.current.partial(text);
          // The latest replaceable partial wins. The renderer schedules one
          // React state write per animation frame instead of one write per
          // provider event.
          partialTranscriptRenderer.schedule(transcript.interimText);
        };
      }
      if (provider.onStatus !== undefined) {
        provider.onStatus = ({ code, message }: { code: string; message: string }) => {
          streamNoticeController.handleStatus(code, message);
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
        cueSessionRef.current?.start();
        startLevelMonitor(stream);

        // Warn if no audio detected after 2s (catches dead/muted mics).
        // Use a small threshold, not strict ===0: a wedged AudioContext or a
        // muted track yields near-silence that is not bit-exactly zero, which
        // is why the old ===0 guard silently missed the "stuck mic" state and
        // showed no error. When we can tell the track is muted, say so —
        // that's the actionable cause (device taken by another app / changed).
        if (noAudioTimerRef.current) clearTimeout(noAudioTimerRef.current);
        noAudioTimerRef.current = setTimeout(() => {
          if (levelMonitor.audioLevel < NO_AUDIO_LEVEL_THRESHOLD) {
            const muted = (stream.getAudioTracks?.() ?? []).some((t) => t.muted);
            const message = muted
              ? "Microphone is muted or in use by another app — reload to recover"
              : "No audio detected — check your microphone";
            setState((s) => (s.voiceState === "recording" || s.voiceState === "listening")
              ? { ...s, error: message }
              : s);
            console.warn(`[voice] No audio after 2s (level=${levelMonitor.audioLevel.toFixed(4)}, muted=${muted})`);
          }
        }, 2000);

        // If stop was requested during async start, abort immediately.
        // The start cue already played above, so play the matching stop cue
        // to keep the pair balanced.
        if (stopRequestedRef.current) {
          stopRequestedRef.current = false;
          cueSessionRef.current?.stop();
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
  }, [state.voiceState, state.backend, voiceLanguage, startLevelMonitor, stopLevelMonitor, handleSegmentFinal, rejectionController, cancelPartialRender]);

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
    // Anchor stop-to-final-tail timing at the shared action, before the
    // user-tap 120 ms encoder settle window below.
    provider.markStopRequested?.();
    // Only play the stop cue if a cue session is active (start cue was played).
    // This prevents the stop sound from firing during cleanup, error recovery,
    // or any other path that disposes the provider without a preceding start cue.
    cueSessionRef.current?.stop();
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
    cueSessionRef.current?.clear();
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
    rejectionController.clearPending();
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
    rejectionController.dismiss();
  }, []);

  const dismissFallbackNotice = useCallback(() => {
    streamNoticeController.dismissFallback();
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
    await rejectionController.retry();
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
    const sharedCtx = levelMonitor.audioContext ?? getSharedAudioContext();
    levelMonitor.audioContext = sharedCtx;
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
    return installVoicePageLifecycle({
      onVisibilityChange: (hidden) => { documentHiddenRef.current = hidden; },
      cancelPendingStarts: () => {
        // A late getUserMedia result must release its lease instead of recording.
        micIntentArmedRef.current = false;
        controller.cancelStarts();
      },
      isActiveRecording: () => isActiveRef.current,
      stopActiveRecording: () => {
        console.info("[voice] Visibility: hidden during active recording — stopping (privacy)");
        endCaptureRef.current?.({ reason: "auto" });
        if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
        setState((s) => ({ ...s, fallbackNotice: "Recording stopped — app moved to background" }));
        fallbackTimerRef.current = setTimeout(() => {
          setState((s) => (s.fallbackNotice ? { ...s, fallbackNotice: null } : s));
        }, 5000);
      },
      suspendAudio: suspendSharedAudioContext,
    });
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
    return installMicLeaseRecovery({
      getVoiceState: () => voiceStateRef.current,
      isStarting: () => startingRef.current,
      isPassiveListenerActive: () => !!passiveListenerRef.current,
      onStaleState: (hasStale) => setState((s) =>
        s.staleLiveMicLease === hasStale ? s : { ...s, staleLiveMicLease: hasStale }),
      recover: (input) => controller.recoverStaleLeases(input),
    });
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
