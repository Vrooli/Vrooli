import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { buildVoiceActivitySnapshot, IDLE_VOICE_ACTIVITY, voiceActivitySnapshotsEqual, VAD_AUTO_STOP_VISUAL_GRACE_MS } from "./activity";
import { advanceMeterEnvelope, LEVEL_ANALYSER_FFT_SIZE, LEVEL_TICK_MS, meterLevelFromEnvelope } from "./audioUtils";
import { ensureRunningSharedAudioContext } from "./sharedAudioContext";
import { decideAutoStop, decideAutoStopRing } from "./autoStopDecision";
import { vadTick, VAD_MIN_SILENCE_THRESHOLD, type VadRefs } from "./vad";
import { useServerVadStateStore, SERVER_VAD_STALE_MS } from "./useServerVadStateStore";
import type { VoiceActivitySnapshot, VoiceBackend, VoiceInputState } from "./types";
import type { VoiceCoreServices } from "./services";

type Ref<T> = MutableRefObject<T>;
export interface LevelMonitorController {
  start(stream: MediaStream): Promise<void>;
  stop(): void;
  readonly audioLevel: number;
  audioContext: AudioContext | null;
}
interface Dependencies {
  services: Pick<VoiceCoreServices, "createAudioFilterChain">;
  setState: Dispatch<SetStateAction<VoiceInputState>>;
  sessionCountRef: Ref<number>;
  vadRef: Ref<VadRefs>;
  vadActiveRef: Ref<boolean>;
  vadSilenceTimeoutRef: Ref<number>;
  persistentModeRef: Ref<boolean>;
  backendRef: Ref<VoiceBackend>;
  endCaptureRef: Ref<((opts?: { reason?: "auto" | "user" }) => void) | null>;
}
export function createLevelMonitorController(deps: Dependencies): LevelMonitorController {
  let audioCtx: AudioContext | null = null;
  let analyser: AnalyserNode | null = null;
  let audioNodes: AudioNode[] = [];
  let raf = 0;
  let lastTick = 0;
  let currentAudioLevel = 0;
  let active = false;
  let generation = 0;
  const { services, setState, sessionCountRef, vadRef, vadActiveRef, vadSilenceTimeoutRef, persistentModeRef, backendRef, endCaptureRef } = deps;
  const start = async (stream: MediaStream) => {
    const startGeneration = ++generation;
    try {
      // Use the shared AudioContext singleton, resumed lazily here (inside the
      // real capture path) — not eagerly on the first arbitrary tap. ensureRunning
      // resumes it and cancels any idle-suspend armed by a prior turn.
      // DOC: docs/internal/VOICE-LATENCY.md#audiocontext-lifecycle
      // Heal a wedged context (suspended/interrupted that won't resume) by
      // rebuilding it, rather than reusing a dead one that yields a flat level
      // meter forever. Non-fatal: capture does not depend on this context.
      const ctx = await ensureRunningSharedAudioContext();
      // Context resume/rebuild is asynchronous. Teardown can run while it is
      // pending, so never attach audio nodes for an obsolete capture session.
      if (startGeneration !== generation) return;
      audioCtx = ctx;
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
      for (const node of audioNodes) {
        try { node.disconnect(); } catch { /* already disconnected */ }
      }
      audioNodes = [];

      const source = ctx.createMediaStreamSource(stream);
      let chain: ReturnType<VoiceCoreServices["createAudioFilterChain"]>;
      try {
        chain = services.createAudioFilterChain(ctx, source);
      } catch (error) {
        // The source is created before the filter chain. If a browser or
        // injected filter factory throws, it has not yet been added to
        // audioNodes, so stop() cannot release it during teardown.
        try { source.disconnect(); } catch { /* source may not have connected */ }
        throw error;
      }
      const { analyser: nextAnalyser, nodes } = chain;
      analyser = nextAnalyser;
      audioNodes = [source, ...nodes];

      // Read the WHOLE time-domain window. This used to allocate
      // `frequencyBinCount` (half of fftSize), so even the short window that
      // was configured was only half-read. `|| LEVEL_ANALYSER_FFT_SIZE` keeps
      // test doubles that leave fftSize at 0 out of a zero-length buffer.
      const data = new Uint8Array(analyser.fftSize || LEVEL_ANALYSER_FFT_SIZE);
      lastTick = 0;
      active = true;
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
        if (!active) return;

        // Throttle to ~15 Hz -- audio analysis doesn't need 60 fps.
        const now = performance.now();
        const sinceLastTick = lastTick > 0 ? now - lastTick : LEVEL_TICK_MS;
        if (sinceLastTick < LEVEL_TICK_MS) {
          raf = requestAnimationFrame(tick);
          return;
        }
        lastTick = now;

        analyser!.getByteTimeDomainData(data);
        let sum = 0;
        for (let i = 0; i < data.length; i++) {
          const v = ((data[i] ?? 128) - 128) / 128;
          sum += v * v;
        }
        const rms = Math.sqrt(sum / data.length);
        // RAW absolute reading. The no-audio watchdog below is calibrated to
        // this scale, so it must not become the display value.
        const audioLevel = Math.min(1, rms * 4);
        currentAudioLevel = audioLevel;

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
          }
          if (result === "segment-boundary") {
            // Client VAD drives local turn timing. Server segmentation and
            // silence handling remain owned by the stream strategy.
          } else if (result === "stop") {
            const srcTracks = stream.getTracks();
            const srcAlive = srcTracks.every((t) => t.readyState === "live");
            const srcMuted = srcTracks.some((t) => t.muted);
            console.info(`[voice] S${sessionCountRef.current} VAD client-stop: silenceElapsed=${vadNow - vadRef.current.silenceStart}ms, timeout=${vadSilenceTimeoutRef.current}ms, rms=${rms.toFixed(4)}, speechThresh=${vadRef.current.speechThreshold.toFixed(4)}, silenceThresh=${vadRef.current.silenceThreshold.toFixed(4)}, trackAlive=${srcAlive}, trackMuted=${srcMuted}, ctx=${ctx.state}`);
            // Persistent mode resets its local VAD window after each silence
            // boundary. Server segmentation is driven by the active strategy.
            if (persistentModeRef.current) {
              vadRef.current.state = "waitingForSpeech";
              vadRef.current.recordingStart = vadNow;
              vadRef.current.segmentBoundaryEmitted = false;
            }
          } else if (result === "no-speech") {
            const tracks = stream.getAudioTracks?.() ?? stream.getTracks();
            const trackUnavailable = tracks.length === 0
              || tracks.some((track) => track.readyState !== "live" || track.muted);
            const monitorUnavailable = ctx.state !== "running";
            const sourceUnavailable = trackUnavailable || monitorUnavailable;
            const terminalMessage = sourceUnavailable
              ? "Microphone audio became unavailable — tap to try again"
              : "No speech detected";
            console.info(`[voice] S${sessionCountRef.current} VAD no-speech after ${vadNow - vadRef.current.recordingStart}ms, rms=${rms.toFixed(4)}, sourceUnavailable=${sourceUnavailable}, trackUnavailable=${trackUnavailable}, ctx=${ctx.state}`);
            vadActiveRef.current = false;
            vadRef.current.state = "idle";
            endCaptureRef.current?.({ reason: "auto" });
            // `endCapture` stops the provider asynchronously. Keep the UI's
            // workflow state ahead of that callback so the next tap is a new
            // start, rather than being interpreted as a second stop. Whisper
            // still needs its final transcription settlement; other providers
            // can return directly to idle.
            setState((s) => ({
              ...s,
              voiceState: backendRef.current === "whisper" ? "transcribing" : "idle",
              error: terminalMessage,
              audioLevel: 0,
              voiceActivity: IDLE_VOICE_ACTIVITY,
              partialTranscript: "",
            }));
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

        if (!active) return;

        // `audioLevel` on the snapshot is the display value; `rms` stays raw so
        // consumers that want the underlying signal (and the thresholds it is
        // judged against) still have it.
        const clientActivity = buildVoiceActivitySnapshot({
          vadActive: vadActiveRef.current,
          vad: vadRef.current,
          rms,
          audioLevel: meterLevel,
          nowMs: vadNow,
          silenceTimeoutMs: vadSilenceTimeoutRef.current,
          voiceMode: persistentModeRef.current ? "persistent" : "one-shot",
        });
        // Reconcile the countdown with the SAME authority that will actually
        // stop the turn. The snapshot's own autoStopProgress is the client RMS
        // timer, which is not the operative clock whenever the server is
        // running VAD — so under a server-VAD strategy the ring stayed hidden
        // for the entire countdown and the turn ended with no warning at all.
        // decideAutoStopRing applies decideAutoStop's precedence; publishing its
        // verdict here means every surface that renders the snapshot gets the
        // right ring without knowing which side owns the decision.
        const ringNowPerf = typeof performance !== "undefined" && typeof performance.now === "function"
          ? performance.now()
          : Date.now();
        const ring = decideAutoStopRing({
          isRecording: vadActiveRef.current && !persistentModeRef.current,
          serverVad: useServerVadStateStore.getState(),
          voiceActivity: clientActivity,
          nowPerf: ringNowPerf,
          staleTickMs: SERVER_VAD_STALE_MS,
          visualGraceMs: VAD_AUTO_STOP_VISUAL_GRACE_MS,
        });
        const voiceActivity: VoiceActivitySnapshot = ring.visible === clientActivity.autoStopVisible
          && ring.progress === clientActivity.autoStopProgress
          ? clientActivity
          : { ...clientActivity, autoStopVisible: ring.visible, autoStopProgress: ring.progress };
        setState((s) => {
          if (Math.abs(s.audioLevel - meterLevel) < 0.01 && voiceActivitySnapshotsEqual(s.voiceActivity, voiceActivity)) {
            return s;
          }
          return { ...s, audioLevel: meterLevel, voiceActivity };
        });

        raf = requestAnimationFrame(tick);
      };
      raf = requestAnimationFrame(tick);
    } catch (err) {
      console.error("[voice] S%d startLevelMonitor FAILED:", sessionCountRef.current, err);
    }
  };
  const stop = () => {
    // Invalidate an ensureRunningSharedAudioContext() still in flight.
    generation++;
    active = false;
    cancelAnimationFrame(raf);
    raf = 0;
    lastTick = 0;
    currentAudioLevel = 0;
    analyser = null;
    // Disconnect all audio nodes to prevent zombie node accumulation
    for (const node of audioNodes) {
      try { node.disconnect(); } catch { /* already disconnected */ }
    }
    audioNodes = [];
    setState((s) => (s.audioLevel === 0 && voiceActivitySnapshotsEqual(s.voiceActivity, IDLE_VOICE_ACTIVITY)
      ? s
      : { ...s, audioLevel: 0, voiceActivity: IDLE_VOICE_ACTIVITY }));
  };
  return { start, stop, get audioLevel() { return currentAudioLevel; }, get audioContext() { return audioCtx; }, set audioContext(value: AudioContext | null) { audioCtx = value; } };
}
