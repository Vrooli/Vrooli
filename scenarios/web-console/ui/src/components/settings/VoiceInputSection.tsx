import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AlertCircle,
  CheckCircle,
  ChevronDown,
  ChevronRight,
  Circle,
  Keyboard,
  Mic,
  Play,
  RefreshCw,
  RotateCcw,
  Square,
  Trash2,
  UserRound,
} from "lucide-react";
import { strings } from "../../consts/strings";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { toErrorInfo } from "../../lib/errors";
import {
  clearSpeakerVerificationProfile,
  deleteWakeWordConfig,
  deleteSpeakerVerificationProfile,
  enrollSpeakerVerificationProfile,
  getSpeakerVerificationStatus,
  getVoiceStreamConfig,
  getWakeWordConfig,
  removeSpeakerVerificationProfile,
  updateWakeWordConfig,
  updateSpeakerVerificationConfig,
  updateVoiceStreamConfig,
  type SpeakerVerificationStatusResponse,
  type VoiceStreamConfig,
  type WakeWordConfig,
} from "../../audio-integration";
import { fetchCapabilities, type CapabilityState } from "../../api/capabilities";
import { probeWhisperHealth } from "../../audio-integration";
import { PcmVoiceStreamProvider, WhisperProvider } from "../../audio-integration";
import type { TranscriptionProvider } from "../../audio-integration";
import { getSharedAudioContext } from "../../audio-integration/hooks/voice/sharedAudioContext";
import { createAudioFilterChain } from "../../audio-integration/hooks/voice/audioUtils";
import { acquireMicStream, releaseMicLease, type MicLease, type MicReleaseReason } from "../../audio-integration/hooks/voice/micOwnership";
import { VOICE_COMMANDS } from "../../hooks/voice/commands";
import {
  bytesToFeatures,
  createWakeWordEngine,
  DEFAULT_WAKE_WORD_THRESHOLD,
  MIN_ENROLLMENT_SAMPLES,
  MAX_ENROLLMENT_SAMPLES,
  useWakeWordTest,
  WAKE_WORD_AUDIO_CONSTRAINTS,
  type AudioFeatures,
} from "../../audio-integration";
import { formatShortcutFromEvent } from "../../lib/shortcutParser";
import { Button } from "../ui/button";
import { SettingsSlider, SettingsToggle } from "./primitives";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";

/** Lease reasons meaning the page/OS pulled the mic, so an in-flight settings
 *  capture must be cancelled (not processed/uploaded). */
const LIFECYCLE_CANCEL_REASONS: ReadonlySet<MicReleaseReason> = new Set(["hidden", "pagehide", "freeze", "ended"]);

export default function VoiceInputSection() {
  const { t } = useTranslation();
  const voiceEnabled = useWorkspaceStore((state) => state.voiceEnabled);
  const setVoiceEnabled = useWorkspaceStore((state) => state.setVoiceEnabled);
  const voiceShortcut = useWorkspaceStore((state) => state.voiceShortcut);
  const setVoiceShortcut = useWorkspaceStore((state) => state.setVoiceShortcut);
  const vadAutoStop = useWorkspaceStore((state) => state.vadAutoStop);
  const setVadAutoStop = useWorkspaceStore((state) => state.setVadAutoStop);
  // Subscribed rather than read through getState(): the threshold is rendered
  // in two places, and a getState() read never re-renders when it changes.
  const wakeWordThreshold = useWorkspaceStore((state) => state.wakeWordThreshold);
  const setWakeWordThreshold = useWorkspaceStore((state) => state.setWakeWordThreshold);
  const vadSilenceTimeoutMs = useWorkspaceStore((state) => state.vadSilenceTimeoutMs);
  const setStoreVadSilenceTimeoutMs = useWorkspaceStore((state) => state.setVadSilenceTimeoutMs);
  const voiceLanguage = useWorkspaceStore((state) => state.voiceLanguage);
  const setVoiceLanguage = useWorkspaceStore((state) => state.setVoiceLanguage);
  const setStorePersistentMode = useWorkspaceStore((state) => state.setPersistentMode);
  const setStoreWakeWordEnabled = useWorkspaceStore((state) => state.setWakeWordEnabled);
  const setStoreSegmentSilenceMs = useWorkspaceStore((state) => state.setSegmentSilenceMs);

  const [recordingShortcut, setRecordingShortcut] = useState(false);
  const [voiceCaps, setVoiceCaps] = useState<CapabilityState[]>([]);
  const [voiceCapsLoading, setVoiceCapsLoading] = useState(false);
  const [voiceCapsError, setVoiceCapsError] = useState<string | null>(null);
  const [micPermission, setMicPermission] = useState<"granted" | "denied" | "prompt" | "unknown">("unknown");
  const [micRequesting, setMicRequesting] = useState(false);
  const [vsConfig, setVsConfig] = useState<VoiceStreamConfig | null>(null);
  const [vsConfigLoading, setVsConfigLoading] = useState(false);
  const [vsConfigError, setVsConfigError] = useState<string | null>(null);
  const [speakerStatus, setSpeakerStatus] = useState<SpeakerVerificationStatusResponse | null>(null);
  const [speakerLoading, setSpeakerLoading] = useState(false);
  const [speakerError, setSpeakerError] = useState<string | null>(null);
  const [enrollmentState, setEnrollmentState] = useState<"idle" | "recording" | "uploading" | "success" | "error">("idle");
  const [enrollmentSeconds, setEnrollmentSeconds] = useState(0);
  // Live mic level (0..1) shown as a meter while enrolling, so the user can
  // see their voice is being captured. Driven by the same AnalyserNode the
  // streaming mic uses (createAudioFilterChain) — its absence was why
  // enrollment "showed no volume".
  const [enrollmentLevel, setEnrollmentLevel] = useState(0);
  const [enrollmentMessage, setEnrollmentMessage] = useState<string | null>(null);
  const [profileDisplayName, setProfileDisplayName] = useState("My Voice");
  const [reEnrollTargetId, setReEnrollTargetId] = useState<string | null>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  // Wake word state
  const [wakeWordConfig, setWakeWordConfig] = useState<WakeWordConfig | null>(null);
  const [, setWakeWordLoading] = useState(false);
  const [wakeWordError, setWakeWordError] = useState<string | null>(null);
  const [wwRecordingIdx, setWwRecordingIdx] = useState<number | null>(null);
  const [wwRecordingSeconds, setWwRecordingSeconds] = useState(0);
  const [wwPlayingIdx, setWwPlayingIdx] = useState<number | null>(null);
  const [wwTestResult, setWwTestResult] = useState<string | null>(null);
  const [wwLabel, setWwLabel] = useState("Hey Vrooli");
  const wwSamplesRef = useRef<(AudioFeatures | null)[]>([]);
  const wwAudioBlobsRef = useRef<(Blob | null)[]>([]);
  const wwRecorderRef = useRef<MediaRecorder | null>(null);
  const wwLeaseRef = useRef<MicLease | null>(null);
  /** Set when the mic lease is pulled by page/OS lifecycle so onstop skips
   *  feature extraction of the cancelled capture. */
  const wwCancelledRef = useRef(false);
  const wwChunksRef = useRef<Blob[]>([]);
  const wwTickerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const wwStopTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wwEngineRef = useRef(createWakeWordEngine());
  const wwPlaybackRef = useRef<HTMLAudioElement | null>(null);

  // Live wake word test hook
  const wwTestSamples = wwSamplesRef.current.filter((s): s is AudioFeatures => s !== null);
  const wakeWordTest = useWakeWordTest({
    engine: wwEngineRef.current,
    samples: wwTestSamples,
    threshold: useWorkspaceStore.getState().wakeWordThreshold,
    disabled: wwRecordingIdx !== null,
  });

  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const enrollmentChunksRef = useRef<Blob[]>([]);
  const enrollmentRecorderRef = useRef<MediaRecorder | null>(null);
  const enrollmentLeaseRef = useRef<MicLease | null>(null);
  /** Set when the mic lease is pulled by page/OS lifecycle so onstop skips
   *  the enroll upload of the cancelled capture. */
  const enrollmentCancelledRef = useRef(false);
  const enrollmentTickerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const enrollmentStopTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Web Audio graph used purely for the live level meter + Chrome's render
  // keepalive (see createAudioFilterChain). Recording itself still uses the
  // RAW mic stream so enrollment embeddings match the streaming verification
  // path's audio characteristics. Disconnected in teardownEnrollmentAudio.
  const enrollmentSourceRef = useRef<MediaStreamAudioSourceNode | null>(null);
  const enrollmentNodesRef = useRef<AudioNode[]>([]);
  const enrollmentAnalyserRef = useRef<AnalyserNode | null>(null);
  const enrollmentLevelTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const hasWebSpeech = typeof window !== "undefined" &&
    Boolean("SpeechRecognition" in window || "webkitSpeechRecognition" in window);

  const checkMicPermission = useCallback(async () => {
    try {
      const result = await navigator.permissions.query({ name: "microphone" as PermissionName });
      setMicPermission(result.state as "granted" | "denied" | "prompt");
      result.onchange = () => setMicPermission(result.state as "granted" | "denied" | "prompt");
    } catch {
      setMicPermission("unknown");
    }
  }, []);

  useEffect(() => {
    void checkMicPermission();
  }, [checkMicPermission]);

  // ── Wake word config loading ──
  const loadWakeWordConfig = useCallback(async (signal?: { cancelled: boolean }) => {
    setWakeWordLoading(true);
    setWakeWordError(null);
    try {
      const config = await getWakeWordConfig();
      if (signal?.cancelled) return;
      setWakeWordConfig(config);
      if (config.template) {
        setWwLabel(config.template.label);
        const persisted = config.template.samples;
        // The template persists RAW audio. Rehydrate playable blobs from the
        // returned bytes, and re-derive MFCC features from that same audio via
        // the shared helper (features are never persisted). One failed clip
        // becomes a null slot rather than failing the whole load.
        // Copy into a fresh (non-shared) ArrayBuffer so the bytes are a valid
        // BlobPart regardless of the proto reader's backing buffer.
        const blobs = persisted.map((s) => {
          const copy = new Uint8Array(s.audio.length);
          copy.set(s.audio);
          return new Blob([copy], { type: s.mime });
        });
        const features = await Promise.all(
          persisted.map((s) => bytesToFeatures(s.audio, wwEngineRef.current).catch(() => null)),
        );
        if (signal?.cancelled) return;
        wwSamplesRef.current = features;
        wwAudioBlobsRef.current = blobs;
      }
    } catch (error) {
      if (!signal?.cancelled) setWakeWordError(toErrorInfo(error).message);
    } finally {
      if (!signal?.cancelled) setWakeWordLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!voiceEnabled) return;
    const signal = { cancelled: false };
    void loadWakeWordConfig(signal);
    return () => { signal.cancelled = true; };
  }, [voiceEnabled, loadWakeWordConfig]);

  const stopWwRecording = useCallback(() => {
    if (wwTickerRef.current) { clearInterval(wwTickerRef.current); wwTickerRef.current = null; }
    if (wwStopTimerRef.current) { clearTimeout(wwStopTimerRef.current); wwStopTimerRef.current = null; }
    if (wwRecorderRef.current?.state === "recording") wwRecorderRef.current.stop();
  }, []);

  const startWwRecording = useCallback(async (slotIdx: number) => {
    setWakeWordError(null);
    setWwTestResult(null);
    setWwRecordingIdx(slotIdx);
    setWwRecordingSeconds(0);
    wwCancelledRef.current = false;
    wwChunksRef.current = [];
    try {
      // Wake-word enrollment capture: pin the same constraints the settings test
      // and the passive listener use, so the channel matches at detection time.
      const lease = await acquireMicStream("wake-word-enrollment", { audio: WAKE_WORD_AUDIO_CONSTRAINTS }, {
        onRelease: (reason) => {
          if (!LIFECYCLE_CANCEL_REASONS.has(reason)) return;
          // Page/OS pulled the mic mid-capture → cancel; onstop must not extract.
          wwCancelledRef.current = true;
          if (wwTickerRef.current) { clearInterval(wwTickerRef.current); wwTickerRef.current = null; }
          if (wwStopTimerRef.current) { clearTimeout(wwStopTimerRef.current); wwStopTimerRef.current = null; }
          if (wwRecorderRef.current?.state === "recording") wwRecorderRef.current.stop();
          setWwRecordingIdx(null);
        },
      });
      wwLeaseRef.current = lease;
      const recorder = new MediaRecorder(lease.stream, {
        mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus") ? "audio/webm;codecs=opus" : "audio/webm",
      });
      wwRecorderRef.current = recorder;
      recorder.ondataavailable = (e) => { if (e.data.size > 0) wwChunksRef.current.push(e.data); };
      recorder.onerror = () => {
        releaseMicLease(wwLeaseRef.current, "provider-error");
        wwLeaseRef.current = null;
        setWwRecordingIdx(null);
        setWakeWordError(t(strings.settings.voiceInputSection.recordingFailed));
      };
      recorder.onstop = async () => {
        releaseMicLease(wwLeaseRef.current, "manual-stop");
        wwLeaseRef.current = null;
        if (wwCancelledRef.current) { wwCancelledRef.current = false; setWwRecordingIdx(null); return; }
        const blob = new Blob(wwChunksRef.current, { type: "audio/webm" });
        if (blob.size === 0) { setWwRecordingIdx(null); setWakeWordError(t(strings.settings.voiceInputSection.recordingEmpty)); return; }
        // Decode audio and extract MFCC features via the shared helper — the
        // same decode path used on load, so re-derived features match exactly.
        try {
          const bytes = new Uint8Array(await blob.arrayBuffer());
          const features = await bytesToFeatures(bytes, wwEngineRef.current);
          const next = [...wwSamplesRef.current];
          while (next.length <= slotIdx) next.push(null);
          next[slotIdx] = features;
          wwSamplesRef.current = next;
          const blobs = [...wwAudioBlobsRef.current];
          while (blobs.length <= slotIdx) blobs.push(null);
          blobs[slotIdx] = blob;
          wwAudioBlobsRef.current = blobs;
        } catch (err) {
          setWakeWordError(t(strings.settings.voiceInputSection.featureExtractionFailed, { error: String(err) }));
        }
        setWwRecordingIdx(null);
      };
      recorder.start(250);
      wwTickerRef.current = setInterval(() => setWwRecordingSeconds((v) => v + 1), 1000);
      wwStopTimerRef.current = setTimeout(() => stopWwRecording(), 5000);
    } catch (error) {
      setWwRecordingIdx(null);
      setWakeWordError(toErrorInfo(error).message);
    }
  }, [stopWwRecording, t]);

  const playWwSample = useCallback((idx: number) => {
    const blob = wwAudioBlobsRef.current[idx];
    if (!blob) return;
    if (wwPlaybackRef.current) { wwPlaybackRef.current.pause(); wwPlaybackRef.current = null; }
    const url = URL.createObjectURL(blob);
    const audio = new Audio(url);
    wwPlaybackRef.current = audio;
    setWwPlayingIdx(idx);
    audio.onended = () => { setWwPlayingIdx(null); URL.revokeObjectURL(url); };
    audio.play().catch(() => setWwPlayingIdx(null));
  }, []);

  const removeWwSample = useCallback((idx: number) => {
    const next = [...wwSamplesRef.current];
    next[idx] = null;
    wwSamplesRef.current = next;
    const blobs = [...wwAudioBlobsRef.current];
    blobs[idx] = null;
    wwAudioBlobsRef.current = blobs;
    // Force re-render by updating a derived piece of state
    setWwTestResult(null);
    setWakeWordError(null);
  }, []);

  // Cleanup wake word recording on unmount
  useEffect(() => {
    return () => {
      if (wwTickerRef.current) clearInterval(wwTickerRef.current);
      if (wwStopTimerRef.current) clearTimeout(wwStopTimerRef.current);
      releaseMicLease(wwLeaseRef.current, "unmount");
      wwLeaseRef.current = null;
      if (wwPlaybackRef.current) { wwPlaybackRef.current.pause(); wwPlaybackRef.current = null; }
    };
  }, []);

  const requestMicPermission = useCallback(async () => {
    setMicRequesting(true);
    try {
      // One-shot permission probe: acquire through the registry for ownership
      // visibility, then release immediately (tracks stopped).
      const lease = await acquireMicStream("mic-permission-probe", { audio: true });
      releaseMicLease(lease, "manual-stop");
      setMicPermission("granted");
      if (voiceEnabled) {
        setVoiceEnabled(false);
        setTimeout(() => setVoiceEnabled(true), 50);
      }
    } catch {
      setMicPermission("denied");
    } finally {
      setMicRequesting(false);
    }
  }, [setVoiceEnabled, voiceEnabled]);

  const loadVoiceCaps = useCallback(async (signal?: { cancelled: boolean }) => {
    setVoiceCapsLoading(true);
    setVoiceCapsError(null);
    try {
      const data = await fetchCapabilities();
      if (signal?.cancelled) return;
      setVoiceCaps(data.capabilities);
    } catch (error) {
      if (signal?.cancelled) return;
      setVoiceCapsError(toErrorInfo(error).message);
    } finally {
      if (!signal?.cancelled) {
        setVoiceCapsLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const signal = { cancelled: false };
    void loadVoiceCaps(signal);
    return () => {
      signal.cancelled = true;
    };
  }, [loadVoiceCaps]);

  const loadVoiceStreamConfig = useCallback(async (signal?: { cancelled: boolean }) => {
    setVsConfigLoading(true);
    setVsConfigError(null);
    try {
      const config = await getVoiceStreamConfig();
      if (!signal?.cancelled) {
        setVsConfig(config);
        // Hydrate workspace store from backend config so useVoiceInput picks up
        // persisted values without waiting for the user to toggle settings.
        // vad_silence_ms is the authoritative VAD knob; fall back to the
        // legacy segment_silence_ms only if it's unset on the server.
        setStorePersistentMode(config.persistentMode);
        setStoreWakeWordEnabled(config.wakeWordEnabled ?? false);
        // wake_word_threshold is the durable source of truth for match
        // sensitivity (shared by the test, the slider, and the passive
        // listener). Hydrate the store unless the server value is unset (0).
        if (config.wakeWordThreshold > 0) {
          useWorkspaceStore.getState().setWakeWordThreshold(config.wakeWordThreshold);
        }
        const silenceMs = config.vadSilenceMs > 0 ? config.vadSilenceMs : (config.segmentSilenceMs || 1500);
        setStoreSegmentSilenceMs(silenceMs);
        setStoreVadSilenceTimeoutMs(silenceMs);
      }
    } catch (error) {
      if (!signal?.cancelled) {
        setVsConfigError(toErrorInfo(error).message);
      }
    } finally {
      if (!signal?.cancelled) {
        setVsConfigLoading(false);
      }
    }
  }, [setStorePersistentMode, setStoreWakeWordEnabled, setStoreSegmentSilenceMs, setStoreVadSilenceTimeoutMs]);

  const loadSpeakerStatus = useCallback(async (signal?: { cancelled: boolean }) => {
    setSpeakerLoading(true);
    setSpeakerError(null);
    try {
      const status = await getSpeakerVerificationStatus();
      if (!signal?.cancelled) {
        setSpeakerStatus(status);
      }
    } catch (error) {
      if (!signal?.cancelled) {
        setSpeakerError(toErrorInfo(error).message);
      }
    } finally {
      if (!signal?.cancelled) setSpeakerLoading(false);
    }
  }, []);

  // Load voice config when voice is enabled (needed for persistent mode settings
  // and advanced streaming section)
  useEffect(() => {
    if (!voiceEnabled) return;
    const signal = { cancelled: false };
    void loadVoiceStreamConfig(signal);
    void loadSpeakerStatus(signal);
    return () => {
      signal.cancelled = true;
    };
  }, [voiceEnabled, loadSpeakerStatus, loadVoiceStreamConfig]);

  useEffect(() => {
    return () => {
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
      if (enrollmentTickerRef.current) clearInterval(enrollmentTickerRef.current);
      if (enrollmentStopTimerRef.current) clearTimeout(enrollmentStopTimerRef.current);
      if (enrollmentLevelTimerRef.current) clearInterval(enrollmentLevelTimerRef.current);
      try { enrollmentSourceRef.current?.disconnect(); } catch { /* noop */ }
      for (const node of enrollmentNodesRef.current) {
        try { node.disconnect(); } catch { /* noop */ }
      }
      releaseMicLease(enrollmentLeaseRef.current, "unmount");
      enrollmentLeaseRef.current = null;
    };
  }, []);

  const handleVsConfigChange = useCallback((patch: Partial<VoiceStreamConfig>) => {
    setVsConfig((current) => (current ? { ...current, ...patch } : null));
    // Update workspace store immediately for reactive consumers (useVoiceInput).
    // Both vadSilenceMs and segmentSilenceMs patches map onto the same two
    // store fields: the silence-timeout slider, the segment-silence slider,
    // and the server's vad_silence_ms collapse to a single source of truth.
    if (patch.persistentMode !== undefined) setStorePersistentMode(patch.persistentMode);
    if (patch.wakeWordEnabled !== undefined) setStoreWakeWordEnabled(patch.wakeWordEnabled);
    const silenceMs = patch.vadSilenceMs ?? patch.segmentSilenceMs;
    if (silenceMs !== undefined) {
      setStoreSegmentSilenceMs(silenceMs);
      setStoreVadSilenceTimeoutMs(silenceMs);
    }
    // Debounce the backend write
    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current);
    }
    saveTimerRef.current = setTimeout(async () => {
      try {
        const updated = await updateVoiceStreamConfig(patch);
        setVsConfig(updated);
        setVsConfigError(null);
      } catch (error) {
        setVsConfigError(toErrorInfo(error).message);
      }
    }, 500);
  }, [setStorePersistentMode, setStoreWakeWordEnabled, setStoreSegmentSilenceMs, setStoreVadSilenceTimeoutMs]);

  const saveWakeWord = useCallback(async () => {
    // We persist RAW audio, not MFCC features — collect the recorded blob for
    // every slot that has both a blob and successfully-extracted features.
    const blobs: Blob[] = [];
    for (let i = 0; i < wwAudioBlobsRef.current.length; i++) {
      const blob = wwAudioBlobsRef.current[i];
      if (blob && wwSamplesRef.current[i]) blobs.push(blob);
    }
    const featureCount = wwSamplesRef.current.filter((s): s is AudioFeatures => s !== null).length;
    if (blobs.length < MIN_ENROLLMENT_SAMPLES) {
      // Distinguish "not enough samples" from "samples present but their audio
      // is missing" (only possible for a slot that never captured a blob).
      setWakeWordError(
        featureCount >= MIN_ENROLLMENT_SAMPLES
          ? t(strings.settings.voiceInputSection.wakeWordReRecordNeeded)
          : t(strings.settings.voiceInputSection.minSamplesNeeded, { min: MIN_ENROLLMENT_SAMPLES }),
      );
      return;
    }
    setWakeWordError(null);
    const threshold = useWorkspaceStore.getState().wakeWordThreshold;
    try {
      const updated = await updateWakeWordConfig({
        label: wwLabel.trim() || "Hey Vrooli",
        threshold,
        samples: blobs.map((audio) => ({ audio, sampleRateHz: 16000 })),
      });
      setWakeWordConfig(updated);
      handleVsConfigChange({ wakeWordEnabled: true });
      setWwTestResult(t(strings.settings.voiceInputSection.wakeWordSaved));
    } catch (error) {
      setWakeWordError(toErrorInfo(error).message);
    }
  }, [wwLabel, handleVsConfigChange, t]);

  const deleteWakeWord = useCallback(async () => {
    setWakeWordError(null);
    try {
      const updated = await deleteWakeWordConfig();
      setWakeWordConfig(updated);
      wwSamplesRef.current = [];
      wwAudioBlobsRef.current = [];
      handleVsConfigChange({ wakeWordEnabled: false });
      setWwTestResult(null);
    } catch (error) {
      setWakeWordError(toErrorInfo(error).message);
    }
  }, [handleVsConfigChange]);

  const testWakeWord = useCallback(async () => {
    const samples = wwSamplesRef.current.filter((s): s is AudioFeatures => s !== null);
    if (samples.length < 2) { setWwTestResult(t(strings.settings.voiceInputSection.testNeedSamples)); return; }
    const threshold = useWorkspaceStore.getState().wakeWordThreshold;
    let matches = 0;
    let total = 0;
    for (let i = 0; i < samples.length; i++) {
      const target = samples[i];
      if (!target) continue;
      const others = samples.filter((_, j) => j !== i);
      const result = wwEngineRef.current.compareBest(target, others, threshold);
      total++;
      if (result.isMatch) matches++;
    }
    setWwTestResult(t(strings.settings.voiceInputSection.crossValidation, {
      matches,
      total,
      threshold: threshold.toFixed(2),
    }));
  }, [t]);

  const resetVsConfig = useCallback(async () => {
    try {
      const updated = await updateVoiceStreamConfig({
        flushIntervalMs: 500,
        minDeltaBytes: 4096,
        overlapBytes: 2048,
        persistentMode: false,
        wakeWordEnabled: false,
        wakeWordThreshold: DEFAULT_WAKE_WORD_THRESHOLD,
        segmentSilenceMs: 1500,
      });
      setVsConfig(updated);
      setVsConfigError(null);
    } catch (error) {
      setVsConfigError(toErrorInfo(error).message);
    }
  }, []);

  const persistSpeakerConfig = useCallback(async (patch: {
    enabled?: boolean;
    profileIds?: string[];
    threshold?: number;
    mode?: "off" | "filter" | "advisory";
    rejectBehavior?: "drop" | "show-muted";
    fallbackWithoutVerification?: boolean;
  }) => {
    setSpeakerError(null);
    try {
      const updated = await updateSpeakerVerificationConfig(patch);
      setSpeakerStatus((current) => current ? { ...current, config: updated } : current);
      await loadSpeakerStatus();
    } catch (error) {
      setSpeakerError(toErrorInfo(error).message);
    }
  }, [loadSpeakerStatus]);

  // Tear down the level-meter Web Audio graph and timers. Safe to call
  // multiple times. Does NOT stop the recorder/tracks — those are handled in
  // the recorder's onstop so the final webm cluster is flushed first.
  const teardownEnrollmentAudio = useCallback(() => {
    if (enrollmentLevelTimerRef.current) {
      clearInterval(enrollmentLevelTimerRef.current);
      enrollmentLevelTimerRef.current = null;
    }
    try {
      enrollmentSourceRef.current?.disconnect();
    } catch { /* already disconnected */ }
    for (const node of enrollmentNodesRef.current) {
      try { node.disconnect(); } catch { /* already disconnected */ }
    }
    enrollmentSourceRef.current = null;
    enrollmentNodesRef.current = [];
    enrollmentAnalyserRef.current = null;
    setEnrollmentLevel(0);
  }, []);

  const stopEnrollmentRecording = useCallback(() => {
    if (enrollmentTickerRef.current) {
      clearInterval(enrollmentTickerRef.current);
      enrollmentTickerRef.current = null;
    }
    if (enrollmentStopTimerRef.current) {
      clearTimeout(enrollmentStopTimerRef.current);
      enrollmentStopTimerRef.current = null;
    }
    if (enrollmentRecorderRef.current?.state === "recording") {
      enrollmentRecorderRef.current.stop();
    }
  }, []);

  const startEnrollmentRecording = useCallback(async () => {
    setEnrollmentMessage(null);
    setEnrollmentState("recording");
    setEnrollmentSeconds(0);
    setEnrollmentLevel(0);
    enrollmentCancelledRef.current = false;
    enrollmentChunksRef.current = [];

    // Resume the shared AudioContext synchronously inside this click gesture —
    // exactly what the streaming mic does in startRecording. Mobile browsers
    // suspend the context between gestures; without this the level meter (and,
    // on some setups, capture) sees only silence. Done before any await so the
    // gesture context is still active.
    try {
      const ctx = getSharedAudioContext();
      if (ctx.state !== "running") ctx.resume().catch(() => {});
    } catch { /* AudioContext unavailable */ }

    try {
      const lease = await acquireMicStream("speaker-enrollment", { audio: true }, {
        onRelease: (reason) => {
          if (!LIFECYCLE_CANCEL_REASONS.has(reason)) return;
          // Page/OS pulled the mic mid-enrollment → cancel; onstop must not upload.
          enrollmentCancelledRef.current = true;
          stopEnrollmentRecording();
          teardownEnrollmentAudio();
          setEnrollmentState("idle");
        },
      });
      enrollmentLeaseRef.current = lease;
      const stream = lease.stream;

      // Build the same analyser/filter graph the streaming mic uses. We do NOT
      // record its filteredStream — recording the RAW stream keeps enrollment
      // and verification embeddings on matching audio characteristics — but the
      // graph drives the live level meter and keeps Chrome's Web Audio renderer
      // alive (silent-gain keepalive in createAudioFilterChain).
      try {
        const ctx = getSharedAudioContext();
        if (ctx.state !== "running") await ctx.resume();
        const source = ctx.createMediaStreamSource(stream);
        const { analyser, nodes } = createAudioFilterChain(ctx, source);
        enrollmentSourceRef.current = source;
        enrollmentNodesRef.current = nodes;
        enrollmentAnalyserRef.current = analyser;
        const timeDomain = new Uint8Array(analyser.frequencyBinCount);
        enrollmentLevelTimerRef.current = setInterval(() => {
          const a = enrollmentAnalyserRef.current;
          if (!a) return;
          a.getByteTimeDomainData(timeDomain);
          let sum = 0;
          for (let i = 0; i < timeDomain.length; i++) {
            const v = ((timeDomain[i] ?? 128) - 128) / 128;
            sum += v * v;
          }
          const rms = Math.sqrt(sum / timeDomain.length);
          setEnrollmentLevel(Math.min(1, rms * 4));
        }, 100);
      } catch { /* level meter is best-effort; recording proceeds without it */ }

      const recorder = new MediaRecorder(stream, {
        mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
          ? "audio/webm;codecs=opus"
          : "audio/webm",
        // Match the streaming bitrate so enrollment and verification embeddings
        // are extracted from audio with the same codec characteristics.
        audioBitsPerSecond: 48_000,
      });
      enrollmentRecorderRef.current = recorder;
      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) enrollmentChunksRef.current.push(event.data);
      };
      recorder.onerror = () => {
        teardownEnrollmentAudio();
        releaseMicLease(enrollmentLeaseRef.current, "provider-error");
        enrollmentLeaseRef.current = null;
        setEnrollmentState("error");
        setEnrollmentMessage(t(strings.settings.voiceInputSection.enrollmentRecordingFailed));
      };
      recorder.onstop = () => {
        teardownEnrollmentAudio();
        const blob = new Blob(enrollmentChunksRef.current, { type: "audio/webm" });
        releaseMicLease(enrollmentLeaseRef.current, "manual-stop");
        enrollmentLeaseRef.current = null;
        if (enrollmentCancelledRef.current) { enrollmentCancelledRef.current = false; setEnrollmentState("idle"); return; }
        if (blob.size === 0) {
          setEnrollmentState("error");
          setEnrollmentMessage(t(strings.settings.voiceInputSection.enrollmentEmpty));
          return;
        }
        setEnrollmentState("uploading");
        const targetId = reEnrollTargetId
          ?? (profileDisplayName.trim() || "My Voice").toLowerCase().replace(/\s+/g, "-") + "-" + Date.now();
        void enrollSpeakerVerificationProfile({
          audioBlob: blob,
          profileId: targetId,
          displayName: profileDisplayName.trim() || "My Voice",
          addToActive: true,
          enable: true,
        }).then(async (result) => {
          setEnrollmentState("success");
          setReEnrollTargetId(null);
          setEnrollmentMessage(t(strings.settings.voiceInputSection.enrolled, { name: result.enrollment.display_name }));
          await loadSpeakerStatus();
        }).catch((error) => {
          setEnrollmentState("error");
          setReEnrollTargetId(null);
          setEnrollmentMessage(toErrorInfo(error).message);
        });
      };
      recorder.start(250);
      enrollmentTickerRef.current = setInterval(() => {
        setEnrollmentSeconds((value) => value + 1);
      }, 1000);
      enrollmentStopTimerRef.current = setTimeout(() => {
        stopEnrollmentRecording();
      }, 20000);
    } catch (error) {
      setEnrollmentState("error");
      setEnrollmentMessage(toErrorInfo(error).message);
    }
  }, [loadSpeakerStatus, profileDisplayName, reEnrollTargetId, stopEnrollmentRecording, teardownEnrollmentAudio, t]);

  const clearSpeakerBinding = useCallback(async () => {
    setSpeakerError(null);
    try {
      const updated = await clearSpeakerVerificationProfile();
      setSpeakerStatus((current) => current ? {
        ...current,
        config: updated,
        profileConfigured: false,
        profileExists: false,
      } : current);
      await loadSpeakerStatus();
    } catch (error) {
      setSpeakerError(toErrorInfo(error).message);
    }
  }, [loadSpeakerStatus]);

  const removeProfile = useCallback(async (profileId: string) => {
    setSpeakerError(null);
    try {
      const updated = await removeSpeakerVerificationProfile(profileId);
      setSpeakerStatus((current) => current ? {
        ...current,
        config: updated,
        profileConfigured: updated.profileIds.length > 0,
      } : current);
      await loadSpeakerStatus();
    } catch (error) {
      setSpeakerError(toErrorInfo(error).message);
    }
  }, [loadSpeakerStatus]);

  const deleteProfile = useCallback(async (profileId: string) => {
    setSpeakerError(null);
    try {
      const updated = await deleteSpeakerVerificationProfile(profileId);
      setSpeakerStatus((current) => current ? {
        ...current,
        config: updated,
        profileConfigured: updated.profileIds.length > 0,
      } : current);
      await loadSpeakerStatus();
    } catch (error) {
      setSpeakerError(toErrorInfo(error).message);
    }
  }, [loadSpeakerStatus]);

  return (
    <SettingsList>
      <SettingsList.Intro
        eyebrow={t(strings.settings.voiceInputSection.eyebrow)}
        title={t(strings.settings.voiceInputSection.title)}
        description={t(strings.settings.voiceInputSection.description)}
      />

      <SettingsList.Group>
        <SettingsList.Row
          label={t(strings.settings.voiceInputSection.voiceInputLabel)}
          hint={t(strings.settings.voiceInputSection.voiceInputHint)} control="compact">{(
            <SettingsToggle
              testId="voice-enabled-toggle"
              checked={voiceEnabled}
              onCheckedChange={setVoiceEnabled}
            />
          )}</SettingsList.Row>

        {voiceEnabled && (
          <>
            <SettingsList.Row
              label={t(strings.settings.voiceInputSection.autoStopLabel)}
              hint={t(strings.settings.voiceInputSection.autoStopHint)} control="compact">{(
                <SettingsToggle
                  testId="vad-auto-stop-toggle"
                  checked={vadAutoStop}
                  onCheckedChange={setVadAutoStop}
                />
              )}</SettingsList.Row>

            {vadAutoStop && (
              <SettingsList.Row
                label={t(strings.settings.voiceInputSection.silenceTimeoutLabel)}
                hint={t(strings.settings.voiceInputSection.silenceTimeoutHint)} control="wide">{(
                  // Write-through: audio-tools' stt_stream_config.vad_silence_ms
                  // is the single source of truth. Range matches the
                  // server-side validation [200, 3000] in stream_config.go.
                  // The write happens on commit, so one drag is one request.
                  <SettingsSlider
                    testId="vad-silence-timeout-slider"
                    value={vadSilenceTimeoutMs}
                    onCommit={(next) => handleVsConfigChange({ vadSilenceMs: next })}
                    min={200}
                    max={3000}
                    step={100}
                    disabled={!vsConfig || vsConfigLoading}
                    formatValue={(value) =>
                      t(strings.settings.voiceInputSection.secondsShort, { value: (value / 1000).toFixed(1) })
                    }
                  />
                )}</SettingsList.Row>
            )}

            <SettingsList.Row
              label={t(strings.settings.voiceInputSection.languageLabel)}
              hint={t(strings.settings.voiceInputSection.languageHint)} control="wide">{(
                <select
                  data-testid="voice-language-select"
                  className="rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                  value={voiceLanguage}
                  onChange={(event) => setVoiceLanguage(event.target.value)}
                >
                  <option value="auto">{t(strings.settings.voiceInputSection.languageAuto)}</option>
                  <option value="en-US">{t(strings.settings.voiceInputSection.langEnUs)}</option>
                  <option value="en-GB">{t(strings.settings.voiceInputSection.langEnGb)}</option>
                  <option value="es-ES">{t(strings.settings.voiceInputSection.langEsEs)}</option>
                  <option value="fr-FR">{t(strings.settings.voiceInputSection.langFrFr)}</option>
                  <option value="de-DE">{t(strings.settings.voiceInputSection.langDeDe)}</option>
                  <option value="zh-CN">{t(strings.settings.voiceInputSection.langZhCn)}</option>
                  <option value="ja-JP">{t(strings.settings.voiceInputSection.langJaJp)}</option>
                  <option value="ko-KR">{t(strings.settings.voiceInputSection.langKoKr)}</option>
                  <option value="pt-BR">{t(strings.settings.voiceInputSection.langPtBr)}</option>
                  <option value="hi-IN">{t(strings.settings.voiceInputSection.langHiIn)}</option>
                </select>
              )}</SettingsList.Row>
          </>
        )}
      </SettingsList.Group>

      {voiceEnabled && (
        <SettingsList.Group>
          <SettingsList.Intro
            eyebrow={t(strings.settings.voiceInputSection.persistentEyebrow)}
            title={t(strings.settings.voiceInputSection.persistentTitle)}
            description={t(strings.settings.voiceInputSection.persistentDescription)}
          />

          {vsConfig && (
            <>
              <SettingsList.Row
                label={t(strings.settings.voiceInputSection.persistentModeLabel)}
                hint={t(strings.settings.voiceInputSection.persistentModeHint)} control="compact">{(
                  <SettingsToggle
                    testId="persistent-mode-toggle"
                    checked={vsConfig.persistentMode}
                    onCheckedChange={(next) => handleVsConfigChange({ persistentMode: next })}
                  />
                )}</SettingsList.Row>

              {vsConfig.persistentMode && (
                <>
                  <SettingsList.Row
                    label={t(strings.settings.voiceInputSection.segmentSilenceLabel)}
                    hint={t(strings.settings.voiceInputSection.segmentSilenceHint)} control="wide">{(
                      // Reads from vad_silence_ms (with legacy fall-back),
                      // writes to vad_silence_ms. Both this slider and
                      // the silence-timeout slider above map to the same
                      // server field — they are two UI views on one knob.
                      <SettingsSlider
                        testId="segment-silence-slider"
                        value={vsConfig.vadSilenceMs > 0 ? vsConfig.vadSilenceMs : vsConfig.segmentSilenceMs}
                        onCommit={(next) => handleVsConfigChange({ vadSilenceMs: next })}
                        min={200}
                        max={3000}
                        step={100}
                        disabled={vsConfigLoading}
                        formatValue={(value) =>
                          t(strings.settings.voiceInputSection.secondsShort, { value: (value / 1000).toFixed(1) })
                        }
                      />
                    )}</SettingsList.Row>

                  <div>
                    <div className="text-xs font-medium text-wc-text-secondary">{t(strings.settings.voiceInputSection.voiceCommandsTitle)}</div>
                    <div className="text-[11px] text-wc-text-muted mb-1">
                      {t(strings.settings.voiceInputSection.voiceCommandsHint)}
                    </div>
                    <div className="grid grid-cols-2 gap-1">
                      {VOICE_COMMANDS.map((cmd) => (
                        <div key={cmd.id} className="text-[10px] text-wc-text-faint">
                          <span className="text-wc-text-muted">{cmd.description}</span>
                          {" — "}
                          {cmd.patterns[0]}
                        </div>
                      ))}
                    </div>
                  </div>
                </>
              )}
            </>
          )}

          {!vsConfig && !vsConfigLoading && (
            <div className="text-xs text-wc-text-faint">
              {t(strings.settings.voiceInputSection.voiceConfigUnavailable)}
            </div>
          )}
        </SettingsList.Group>
      )}

      {/* Wake word — independent of persistent mode. Works as a secondary
          trigger in any mode (one-shot or persistent); the mic button still
          behaves normally. */}
      {voiceEnabled && vsConfig && (
        <SettingsList.Group>
          <SettingsList.Intro
            eyebrow={t(strings.settings.voiceInputSection.wakeWordEyebrow)}
            title={t(strings.settings.voiceInputSection.wakeWordTitle)}
            description={t(strings.settings.voiceInputSection.wakeWordSectionDescription)}
          />

          <SettingsList.Row
            label={t(strings.settings.voiceInputSection.wakeWordLabel)}
            hint={t(strings.settings.voiceInputSection.wakeWordHint)} control="compact">{(
              <SettingsToggle
                testId="wake-word-toggle"
                checked={vsConfig.wakeWordEnabled}
                onCheckedChange={(next) => {
                  if (next && !wakeWordConfig?.configured) {
                    setWakeWordError(t(strings.settings.voiceInputSection.wakeWordRequireSaveFirst));
                    return;
                  }
                  handleVsConfigChange({ wakeWordEnabled: next });
                }}
              />
            )}</SettingsList.Row>

          <div className="rounded-xl border border-wc-default bg-wc-surface-base/60 p-3 space-y-3">
            <div className="flex items-center gap-2 text-sm font-medium text-wc-text-secondary">
              <Mic className="h-4 w-4" />
              {t(strings.settings.voiceInputSection.wakeWordRecordingTitle)}
            </div>
            <p className="text-[11px] text-wc-text-muted">
              {t(strings.settings.voiceInputSection.wakeWordRecordingHelp, { min: MIN_ENROLLMENT_SAMPLES, max: MAX_ENROLLMENT_SAMPLES })}
            </p>

            <div className="flex items-center gap-2">
              <span className="text-[11px] text-wc-text-muted shrink-0">{t(strings.settings.voiceInputSection.labelColon)}</span>
              <input
                data-testid="wake-word-label"
                type="text"
                value={wwLabel}
                onChange={(e) => setWwLabel(e.target.value)}
                className="w-36 rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                placeholder={t(strings.settings.voiceInputSection.wakeWordLabelPlaceholder)}
              />
            </div>

            <div>
              {Array.from({ length: MAX_ENROLLMENT_SAMPLES }, (_, i) => {
                const sample = wwSamplesRef.current[i] ?? null;
                const hasSample = sample != null;
                const hasBlob = wwAudioBlobsRef.current[i] != null;
                const isRecording = wwRecordingIdx === i;
                const isPlaying = wwPlayingIdx === i;
                return (
                  <div key={i} className="flex items-center gap-2 h-8">
                    <span className="w-16 text-[11px] text-wc-text-muted">
                      {t(strings.settings.voiceInputSection.sample, { n: i + 1 })}
                      {i < MIN_ENROLLMENT_SAMPLES ? "" : t(strings.settings.voiceInputSection.sampleOptionalSuffix)}
                    </span>
                    {hasSample && sample ? (
                      <>
                        <CheckCircle className="h-3.5 w-3.5 text-green-400 shrink-0" />
                        <span className="text-[10px] text-green-400">
                          {t(strings.settings.voiceInputSection.secondsShort, { value: sample.durationSec.toFixed(1) })}
                        </span>
                        {hasBlob && (
                          <Button
                            variant="ghost"
                            size="icon"
                            shape="square"
                            className="h-6 w-6"
                            title={t(strings.settings.voiceInputSection.playSample)}
                            onClick={() => isPlaying ? undefined : playWwSample(i)}
                            disabled={isPlaying}
                          >
                            <Play className="h-3 w-3 text-wc-text-faint" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="icon"
                          shape="square"
                          className="h-6 w-6"
                          title={t(strings.settings.voiceInputSection.reRecord)}
                          onClick={() => { removeWwSample(i); void startWwRecording(i); }}
                          disabled={wwRecordingIdx !== null}
                        >
                          <RotateCcw className="h-3 w-3 text-wc-text-faint" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          shape="square"
                          className="h-6 w-6"
                          title={t(strings.settings.voiceInputSection.remove)}
                          onClick={() => removeWwSample(i)}
                        >
                          <Trash2 className="h-3 w-3 text-wc-text-faint" />
                        </Button>
                      </>
                    ) : isRecording ? (
                      <Button
                        variant="outline"
                        size="sm"
                        shape="square"
                        className="h-6 px-2 text-[10px]"
                        onClick={stopWwRecording}
                      >
                        <Square className="me-1 h-3 w-3" />
                        {t(strings.settings.voiceInputSection.stopSeconds, { seconds: wwRecordingSeconds })}
                      </Button>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        shape="square"
                        className="h-6 px-2 text-[10px]"
                        onClick={() => void startWwRecording(i)}
                        disabled={wwRecordingIdx !== null}
                      >
                        <Mic className="me-1 h-3 w-3" />
                        {t(strings.settings.voiceInputSection.record)}
                      </Button>
                    )}
                  </div>
                );
              })}
            </div>

            <SettingsList.Row
              label={t(strings.settings.voiceInputSection.sensitivityLabel)}
              hint={t(strings.settings.voiceInputSection.sensitivityHint)} control="wide">{(
                <SettingsSlider
                  testId="wake-word-threshold-slider"
                  value={wakeWordThreshold}
                  onCommit={(next) => {
                    setWakeWordThreshold(next);
                    handleVsConfigChange({ wakeWordThreshold: next });
                  }}
                  min={0.1}
                  max={0.95}
                  step={0.05}
                  formatValue={(value) => value.toFixed(2)}
                />
              )}</SettingsList.Row>

            {wakeWordError && (
              <div className="flex items-center gap-2 text-xs text-wc-error-detail">
                <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                {wakeWordError}
              </div>
            )}
            {wwTestResult && (
              <div className="text-xs text-wc-text-muted">{wwTestResult}</div>
            )}

            <div className="flex flex-wrap items-center gap-2">
              <Button
                data-testid="wake-word-save"
                variant="outline"
                size="sm"
                className="h-8 px-3 text-xs"
                onClick={() => void saveWakeWord()}
                disabled={wwSamplesRef.current.filter((s) => s != null).length < MIN_ENROLLMENT_SAMPLES || wwRecordingIdx !== null}
              >
                {t(strings.settings.voiceInputSection.saveWakeWord)}
              </Button>
              <Button
                data-testid="wake-word-test"
                variant="ghost"
                size="sm"
                className="h-8 px-3 text-xs text-wc-text-faint"
                onClick={() => void testWakeWord()}
                disabled={wwSamplesRef.current.filter((s) => s != null).length < 2}
              >
                {t(strings.settings.voiceInputSection.testCrossMatch)}
              </Button>
              {wakeWordConfig?.configured && (
                <Button
                  data-testid="wake-word-delete"
                  variant="ghost"
                  size="sm"
                  className="h-8 px-3 text-xs text-wc-text-faint"
                  onClick={() => void deleteWakeWord()}
                >
                  {t(strings.settings.voiceInputSection.deleteWakeWord)}
                </Button>
              )}
            </div>
          </div>

          {/* Live wake word test */}
          <div className="rounded-xl border border-wc-default bg-wc-surface-base/60 p-3 space-y-3">
            <div className="flex items-center gap-2 text-sm font-medium text-wc-text-secondary">
              <Play className="h-4 w-4" />
              {t(strings.settings.voiceInputSection.liveTestTitle)}
            </div>
            <p className="text-[11px] text-wc-text-muted">
              {t(strings.settings.voiceInputSection.liveTestHelp)}
            </p>

            {wwTestSamples.length === 0 ? (
              <p className="text-[11px] text-wc-text-faint italic">{t(strings.settings.voiceInputSection.liveTestEmpty)}</p>
            ) : (
              <>
                <div className="flex items-center gap-3">
                  <Button
                    data-testid="wake-word-live-test"
                    variant="outline"
                    size="sm"
                    className={`h-10 px-4 text-xs select-none ${wakeWordTest.state.status === "recording" ? "border-wc-accent text-wc-accent animate-pulse" : ""}`}
                    disabled={wwRecordingIdx !== null || wakeWordTest.state.status === "comparing"}
                    onMouseDown={() => wakeWordTest.startRecording()}
                    onMouseUp={() => wakeWordTest.stopRecording()}
                    onMouseLeave={() => { if (wakeWordTest.state.status === "recording") wakeWordTest.stopRecording(); }}
                    onTouchStart={(e) => { e.preventDefault(); wakeWordTest.startRecording(); }}
                    onTouchEnd={(e) => { e.preventDefault(); wakeWordTest.stopRecording(); }}
                  >
                    <Mic className="me-1.5 h-3.5 w-3.5" />
                    {wakeWordTest.state.status === "idle" || wakeWordTest.state.status === "result"
                      ? t(strings.settings.voiceInputSection.holdToTest)
                      : wakeWordTest.state.status === "recording"
                        ? t(strings.settings.voiceInputSection.recordingSecondsLabel, { seconds: wakeWordTest.state.recordingSeconds })
                        : t(strings.settings.voiceInputSection.comparing)}
                  </Button>
                  {wakeWordTest.state.history.length > 0 && (
                    <button
                      data-testid="wake-word-clear-history"
                      className="text-[10px] text-wc-text-faint hover:text-wc-text-muted underline"
                      onClick={() => wakeWordTest.clearHistory()}
                    >
                      {t(strings.settings.voiceInputSection.clearHistory)}
                    </button>
                  )}
                </div>

                {wakeWordTest.state.error && (
                  <div className="flex items-center gap-2 text-xs text-wc-error-detail">
                    <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                    {wakeWordTest.state.error}
                  </div>
                )}

                {wakeWordTest.state.currentResult && (
                  <div>
                    <div className="flex items-center gap-2 text-xs">
                      <span className={wakeWordTest.state.currentResult.isMatch ? "text-green-500 font-medium" : "text-red-500 font-medium"}>
                        {wakeWordTest.state.currentResult.isMatch ? t(strings.settings.voiceInputSection.liveTestMatch) : t(strings.settings.voiceInputSection.liveTestReject)}
                      </span>
                      <span className="text-wc-text-muted">
                        {t(strings.settings.voiceInputSection.liveTestScore, { score: wakeWordTest.state.currentResult.score.toFixed(3) })}
                      </span>
                    </div>
                    <div className="relative h-3 w-full rounded-full bg-wc-surface-overlay overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all ${wakeWordTest.state.currentResult.isMatch ? "bg-green-500/70" : "bg-red-500/70"}`}
                        style={{ width: `${Math.min(wakeWordTest.state.currentResult.score * 100, 100)}%` }}
                      />
                      <div
                        className="absolute top-0 h-full w-0.5 bg-wc-text-secondary"
                        style={{ left: `${wakeWordThreshold * 100}%` }}
                        title={t(strings.settings.voiceInputSection.thresholdTooltip, { threshold: wakeWordThreshold.toFixed(2) })}
                      />
                    </div>
                  </div>
                )}

                {wakeWordTest.state.history.length > 1 && (
                  <div>
                    <div className="text-[10px] text-wc-text-faint font-medium">{t(strings.settings.voiceInputSection.recentAttempts)}</div>
                    <div className="max-h-32 overflow-y-auto space-y-1">
                      {wakeWordTest.state.history.map((attempt, i) => (
                        <div key={attempt.timestamp} className={`flex items-center gap-2 text-[10px] ${i === 0 ? "opacity-100" : "opacity-70"}`}>
                          <span className={`w-9 font-medium ${attempt.isMatch ? "text-green-500" : "text-red-500"}`}>
                            {attempt.isMatch ? t(strings.settings.voiceInputSection.attemptPass) : t(strings.settings.voiceInputSection.attemptFail)}
                          </span>
                          <div className="relative h-1.5 flex-1 rounded-full bg-wc-surface-overlay overflow-hidden">
                            <div
                              className={`h-full rounded-full ${attempt.isMatch ? "bg-green-500/60" : "bg-red-500/60"}`}
                              style={{ width: `${Math.min(attempt.score * 100, 100)}%` }}
                            />
                            <div
                              className="absolute top-0 h-full w-px bg-wc-text-faint"
                              style={{ left: `${wakeWordThreshold * 100}%` }}
                            />
                          </div>
                          <span className="w-10 text-end text-wc-text-faint">{attempt.score.toFixed(2)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        </SettingsList.Group>
      )}

      {voiceEnabled && (
        <SettingsList.Group>
          <SettingsList.Intro
            eyebrow={t(strings.settings.voiceInputSection.speakerEyebrow)}
            title={t(strings.settings.voiceInputSection.speakerTitle)}
            description={t(strings.settings.voiceInputSection.speakerDescription)}
          />

          {speakerError && (
            <div className="flex items-center gap-2 text-xs text-wc-error-detail">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" />
              {speakerError}
            </div>
          )}

          <SettingsList.Row
            label={t(strings.settings.voiceInputSection.resourceStatusLabel)}
            hint={speakerLoading
              ? t(strings.settings.voiceInputSection.resourceStatusChecking)
              : speakerStatus?.capability === "available"
                ? (speakerStatus.resourceReady ? t(strings.settings.voiceInputSection.resourceReady) : t(strings.settings.voiceInputSection.resourceReachable))
                : t(strings.settings.voiceInputSection.resourceUnavailable)} control="compact">{(
              <div className="flex items-center gap-2">
                <span className={`text-xs font-medium ${
                  speakerStatus?.capability === "available" && speakerStatus?.resourceReady
                    ? "text-green-400"
                    : "text-wc-text-faint"
                }`}>
                  {speakerStatus?.capability === "available" && speakerStatus?.resourceReady ? t(strings.settings.voiceInputSection.ready) : t(strings.settings.voiceInputSection.unavailable)}
                </span>
                <Button
                  data-testid="speaker-refresh"
                  variant="ghost"
                  size="icon"
                  shape="square"
                  className="h-8 w-8"
                  onClick={() => void loadSpeakerStatus()}
                  title={t(strings.settings.voiceInputSection.refreshSpeakerTitle)}
                >
                  <RefreshCw className={`h-3.5 w-3.5 text-wc-text-faint ${speakerLoading ? "animate-spin" : ""}`} />
                </Button>
              </div>
            )}</SettingsList.Row>

          <SettingsList.Row
            label={t(strings.settings.voiceInputSection.useSpeakerLabel)}
            hint={t(strings.settings.voiceInputSection.useSpeakerHint)} control="compact">{(
              <SettingsToggle
                testId="speaker-verification-toggle"
                checked={speakerStatus?.config.enabled ?? false}
                onCheckedChange={(next) => {
                  const ids = speakerStatus?.config.profileIds ?? [];
                  if (next && ids.length === 0 && !speakerStatus?.profiles?.length) {
                    setSpeakerError(t(strings.settings.voiceInputSection.enrollFirst));
                    return;
                  }
                  const profileIds = ids.length > 0
                    ? ids
                    : (speakerStatus?.profiles?.map((p) => p.id) ?? ["default"]);
                  void persistSpeakerConfig({ enabled: next, profileIds });
                }}
              />
            )}</SettingsList.Row>

          <div>
            <div className="text-xs font-medium text-wc-text-secondary">{t(strings.settings.voiceInputSection.activeProfilesTitle)}</div>
            <div className="text-[11px] text-wc-text-muted">
              {t(strings.settings.voiceInputSection.activeProfilesHint)}
            </div>
            {(speakerStatus?.config.profileIds ?? []).length > 0 ? (
              <div className="flex flex-wrap gap-1.5" data-testid="speaker-active-profiles">
                {(speakerStatus?.config.profileIds ?? []).map((id) => {
                  const profile = speakerStatus?.profiles?.find((p) => p.id === id);
                  return (
                    <span
                      key={id}
                      className="inline-flex items-center gap-1 rounded-full border border-wc-default bg-wc-surface-base px-2.5 py-0.5 text-[11px] text-wc-text-primary"
                    >
                      {profile?.display_name ?? id}
                      <button
                        className="ms-0.5 text-wc-text-faint hover:text-wc-text-primary"
                        title={t(strings.settings.voiceInputSection.removeProfileTitle, { name: profile?.display_name ?? id })}
                        onClick={() => void removeProfile(id)}
                      >
                        &times;
                      </button>
                    </span>
                  );
                })}
              </div>
            ) : (
              <div className="text-[11px] text-wc-text-faint">{t(strings.settings.voiceInputSection.noActiveProfiles)}</div>
            )}
          </div>

          <SettingsList.Row
            label={t(strings.settings.voiceInputSection.modeLabel)}
            hint={t(strings.settings.voiceInputSection.modeHint)} control="wide">{(
              <select
                data-testid="speaker-mode-select"
                className="rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                value={speakerStatus?.config.mode ?? "filter"}
                onChange={(event) => {
                  void persistSpeakerConfig({ mode: event.target.value as "off" | "filter" | "advisory" });
                }}
              >
                <option value="filter">{t(strings.settings.voiceInputSection.modeFilter)}</option>
                <option value="advisory">{t(strings.settings.voiceInputSection.modeAdvisory)}</option>
                <option value="off">{t(strings.settings.voiceInputSection.modeOff)}</option>
              </select>
            )}</SettingsList.Row>

          <SettingsList.Row
            label={t(strings.settings.voiceInputSection.thresholdLabel)}
            hint={t(strings.settings.voiceInputSection.thresholdHint)} control="wide">{(
              <SettingsSlider
                testId="speaker-threshold-slider"
                value={speakerStatus?.config.threshold ?? 0.35}
                onCommit={(next) => {
                  void persistSpeakerConfig({ threshold: next });
                }}
                min={0.1}
                max={0.99}
                step={0.01}
                formatValue={(value) => value.toFixed(2)}
              />
            )}</SettingsList.Row>

          <div className="rounded-xl border border-wc-default bg-wc-surface-base/60 p-3">
            <div className="flex items-center gap-2 text-sm font-medium text-wc-text-secondary">
              <UserRound className="h-4 w-4" />
              {t(strings.settings.voiceInputSection.enrollment)}
            </div>
            <p className="mt-1 text-[11px] text-wc-text-muted">
              {t(strings.settings.voiceInputSection.enrollmentHelp)}
            </p>
            <p className="mt-1.5 rounded-lg bg-wc-surface-base/80 px-2.5 py-1.5 text-xs italic text-wc-text-primary">
              {t(strings.settings.voiceInputSection.enrollmentScript)}
            </p>
            <div className="mt-3 flex flex-wrap items-center gap-2">
              <input
                data-testid="speaker-display-name"
                type="text"
                value={profileDisplayName}
                onChange={(event) => setProfileDisplayName(event.target.value)}
                className="w-36 rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                placeholder={t(strings.settings.voiceInputSection.profileNamePlaceholder)}
              />
              {enrollmentState === "recording" ? (
                <>
                  <Button
                    data-testid="speaker-enrollment-stop"
                    variant="outline"
                    size="sm"
                    className="h-8 px-3 text-xs"
                    onClick={stopEnrollmentRecording}
                  >
                    <Square className="me-1 h-3.5 w-3.5" />
                    {t(strings.settings.voiceInputSection.stopSeconds, { seconds: enrollmentSeconds })}
                  </Button>
                  {/* Live mic level: confirms the user's voice is being captured. */}
                  <div
                    data-testid="speaker-enrollment-level"
                    className="h-2 w-24 overflow-hidden rounded-full bg-wc-surface-base"
                    role="meter"
                    aria-label={t(strings.settings.voiceInputSection.addVoiceProfile)}
                    aria-valuenow={Math.round(enrollmentLevel * 100)}
                  >
                    <div
                      className="h-full rounded-full bg-wc-accent transition-[width] duration-100"
                      style={{ width: `${Math.min(enrollmentLevel * 100, 100)}%` }}
                    />
                  </div>
                </>
              ) : (
                <Button
                  data-testid="speaker-enrollment-start"
                  variant="outline"
                  size="sm"
                  className="h-8 px-3 text-xs"
                  onClick={() => {
                    setReEnrollTargetId(null);
                    void startEnrollmentRecording();
                  }}
                  disabled={speakerStatus?.capability !== "available" || !speakerStatus?.resourceReady || enrollmentState === "uploading"}
                >
                  <Mic className="me-1 h-3.5 w-3.5" />
                  {t(strings.settings.voiceInputSection.addVoiceProfile)}
                </Button>
              )}
              <Button
                data-testid="speaker-clear-profile"
                variant="ghost"
                size="sm"
                className="h-8 px-3 text-xs text-wc-text-faint"
                onClick={() => void clearSpeakerBinding()}
                disabled={!(speakerStatus?.config.profileIds?.length)}
              >
                {t(strings.settings.voiceInputSection.clearAllProfiles)}
              </Button>
            </div>
            {enrollmentMessage && (
              <div className={`mt-2 text-xs ${enrollmentState === "error" ? "text-wc-error-detail" : "text-wc-text-muted"}`}>
                {enrollmentMessage}
              </div>
            )}
            {speakerStatus?.profiles?.length ? (
              <div className="mt-3 space-y-1.5">
                <div className="text-[11px] font-medium text-wc-text-secondary">{t(strings.settings.voiceInputSection.enrolledProfilesTitle)}</div>
                {speakerStatus.profiles.map((profile) => {
                  const isActive = speakerStatus.config.profileIds?.includes(profile.id);
                  return (
                    <div
                      key={profile.id}
                      className="flex items-center justify-between rounded-lg border border-wc-default bg-wc-surface-base/40 px-2.5 py-1.5"
                    >
                      <div className="min-w-0">
                        <div className="flex items-center gap-1.5 text-xs text-wc-text-primary">
                          {profile.display_name}
                          {isActive && (
                            <span className="rounded bg-green-400/15 px-1.5 py-0 text-[10px] text-green-400">{t(strings.settings.voiceInputSection.profileActive)}</span>
                          )}
                        </div>
                        <div className="text-[10px] text-wc-text-faint">
                          {t(strings.settings.voiceInputSection.enrollmentSecondsLabel, { seconds: profile.enrollment_audio_seconds.toFixed(1) })}
                          {profile.notes ? ` · ${profile.notes}` : ""}
                        </div>
                      </div>
                      <div className="flex items-center gap-1 shrink-0 ms-2">
                        {!isActive && (
                          <Button
                            variant="ghost"
                            size="icon"
                            shape="square"
                            className="h-6 w-6"
                            title={t(strings.settings.voiceInputSection.addToActiveTitle)}
                            onClick={() => {
                              const ids = [...(speakerStatus.config.profileIds ?? []), profile.id];
                              void persistSpeakerConfig({ profileIds: ids, enabled: true });
                            }}
                          >
                            <CheckCircle className="h-3 w-3 text-wc-text-faint" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="icon"
                          shape="square"
                          className="h-6 w-6"
                          title={t(strings.settings.voiceInputSection.reEnrollTitle, { name: profile.display_name })}
                          onClick={() => {
                            setProfileDisplayName(profile.display_name);
                            setReEnrollTargetId(profile.id);
                            void startEnrollmentRecording();
                          }}
                          disabled={speakerStatus?.capability !== "available" || !speakerStatus?.resourceReady || enrollmentState === "recording" || enrollmentState === "uploading"}
                        >
                          <Mic className="h-3 w-3 text-wc-text-faint" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          shape="square"
                          className="h-6 w-6"
                          title={t(strings.settings.voiceInputSection.deleteProfileTitle, { name: profile.display_name })}
                          onClick={() => void deleteProfile(profile.id)}
                        >
                          <Trash2 className="h-3 w-3 text-wc-text-faint hover:text-red-400" />
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : null}
          </div>
        </SettingsList.Group>
      )}

      <SettingsList.Group>
        <SettingsList.Row
          label={t(strings.settings.voiceInputSection.shortcutLabel)}
          hint={t(strings.settings.voiceInputSection.shortcutHint)} control="compact">{(
            <div className="flex items-center gap-2">
              {recordingShortcut ? (
                <span
                  data-testid="voice-shortcut-recording"
                  className="rounded-lg border border-wc-accent bg-wc-surface-base px-2 py-1 font-mono text-xs text-wc-accent animate-pulse"
                  tabIndex={0}
                  onKeyDown={(event) => {
                    if (["Control", "Alt", "Shift", "Meta"].includes(event.key)) return;
                    event.preventDefault();
                    const shortcut = formatShortcutFromEvent(event.nativeEvent);
                    setVoiceShortcut(shortcut);
                    setRecordingShortcut(false);
                  }}
                  onBlur={() => setRecordingShortcut(false)}
                  ref={(element) => element?.focus()}
                >
                  {t(strings.settings.voiceInputSection.pressKeyCombo)}
                </span>
              ) : (
                <span
                  data-testid="voice-shortcut-display"
                  className="rounded-lg bg-wc-surface-base px-2 py-1 font-mono text-xs text-wc-text-muted"
                >
                  {voiceShortcut}
                </span>
              )}
              <Button
                data-testid="voice-shortcut-change"
                variant="ghost"
                size="icon"
                shape="square"
                className="h-8 w-8"
                onClick={() => setRecordingShortcut(true)}
                title={t(strings.settings.voiceInputSection.changeShortcut)}
              >
                <Keyboard className="h-3.5 w-3.5 text-wc-text-faint" />
              </Button>
            </div>
          )}</SettingsList.Row>
      </SettingsList.Group>

      <SettingsList.Group>
        <div className="flex items-center justify-between">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">{t(strings.settings.voiceInputSection.backendAvailability)}</div>
            <div className="text-[11px] text-wc-text-muted">
              {t(strings.settings.voiceInputSection.backendAvailabilityHint)}
            </div>
          </div>
          <Button
            data-testid="voice-caps-refresh"
            variant="ghost"
            size="icon"
            shape="square"
            className="h-8 w-8"
            onClick={() => void loadVoiceCaps()}
            title={t(strings.settings.voiceInputSection.refreshTitle)}
          >
            <RefreshCw className={`h-3.5 w-3.5 text-wc-text-faint ${voiceCapsLoading ? "animate-spin" : ""}`} />
          </Button>
        </div>

        {voiceCapsError && (
          <div className="flex items-center gap-2 text-xs text-wc-error-detail">
            <AlertCircle className="h-3.5 w-3.5 shrink-0" />
            {voiceCapsError}
          </div>
        )}

        <div>
          {voiceCaps.filter((capability) => capability.features.includes("voice-input")).map((capability) => (
            <div key={capability.id} className="flex items-center gap-2 text-xs">
              {capability.status === "available" ? (
                <CheckCircle className="h-3.5 w-3.5 shrink-0 text-green-400" />
              ) : (
                <Circle className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" />
              )}
              <span className={capability.status === "available" ? "text-wc-text-secondary" : "text-wc-text-faint"}>
                {capability.name}
              </span>
              <span className={capability.status === "available" ? "ml-auto text-green-400" : "ml-auto text-wc-text-faint"}>
                {capability.status}
              </span>
            </div>
          ))}

          <div className="flex items-center gap-2 text-xs">
            {hasWebSpeech ? (
              <CheckCircle className="h-3.5 w-3.5 shrink-0 text-green-400" />
            ) : (
              <Circle className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" />
            )}
            <span className={hasWebSpeech ? "text-wc-text-secondary" : "text-wc-text-faint"}>
              {t(strings.settings.voiceInputSection.webSpeechApi)}
            </span>
            <span className={hasWebSpeech ? "ml-auto text-green-400" : "ml-auto text-wc-text-faint"}>
              {hasWebSpeech ? t(strings.settings.voiceInputSection.available) : t(strings.settings.voiceInputSection.unavailable)}
            </span>
          </div>
        </div>

        {!voiceCapsLoading && voiceCaps.every((capability) => capability.status !== "available") && !hasWebSpeech && (
          <p className="text-[11px] text-amber-400">
            {t(strings.settings.voiceInputSection.noBackendWarning)}
          </p>
        )}
      </SettingsList.Group>

      <SettingsList.Group>
        <div>
          <div className="text-sm font-medium text-wc-text-secondary">{t(strings.settings.voiceInputSection.micAccess)}</div>
          <div className="text-[11px] text-wc-text-muted">
            {t(strings.settings.voiceInputSection.micAccessHint)}
          </div>
        </div>

        <div className="flex items-center gap-2 text-xs">
          {micPermission === "granted" ? (
            <>
              <CheckCircle className="h-3.5 w-3.5 shrink-0 text-green-400" />
              <span className="text-wc-text-secondary">{t(strings.settings.voiceInputSection.permissionGranted)}</span>
            </>
          ) : micPermission === "denied" ? (
            <>
              <AlertCircle className="h-3.5 w-3.5 shrink-0 text-red-400" />
              <span className="text-red-400">{t(strings.settings.voiceInputSection.permissionDenied)}</span>
            </>
          ) : micPermission === "prompt" ? (
            <>
              <Circle className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" />
              <span className="text-wc-text-faint">{t(strings.settings.voiceInputSection.permissionNotRequested)}</span>
            </>
          ) : (
            <>
              <Circle className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" />
              <span className="text-wc-text-faint">{t(strings.settings.voiceInputSection.permissionUnknown)}</span>
            </>
          )}
        </div>

        {micPermission === "denied" && (
          <p className="text-[11px] text-wc-text-faint">
            {t(strings.settings.voiceInputSection.micBlockedHelp)}
          </p>
        )}

        {(micPermission === "prompt" || micPermission === "unknown") && (
          <Button
            data-testid="mic-request-permission"
            variant="outline"
            size="sm"
            className="h-8 px-3 text-xs"
            onClick={() => void requestMicPermission()}
            disabled={micRequesting}
          >
            <Mic className="me-1 h-3.5 w-3.5" />
            {micRequesting ? t(strings.settings.voiceInputSection.requesting) : t(strings.settings.voiceInputSection.allowMicrophone)}
          </Button>
        )}
      </SettingsList.Group>

      {voiceEnabled && <TestMicrophoneCard />}

      {voiceEnabled && (
        <SettingsList.Group>
          <button
            data-testid="advanced-streaming-toggle"
            className="flex w-full items-center gap-1 text-start text-[11px] font-semibold uppercase tracking-[0.22em] text-wc-text-muted"
            onClick={() => setAdvancedOpen(!advancedOpen)}
          >
            {advancedOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
            {t(strings.settings.voiceInputSection.advancedStreaming)}
          </button>

          {advancedOpen && (
            <div>
              {vsConfigLoading && (
                <div className="py-2 text-center text-xs text-wc-text-faint">{t(strings.settings.voiceInputSection.loading)}</div>
              )}

              {vsConfigError && (
                <div className="flex items-center gap-2 text-xs text-wc-error-detail">
                  <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                  {vsConfigError}
                </div>
              )}

              {vsConfig && (
                <>
                  <SettingsList.Row
                    label={t(strings.settings.voiceInputSection.flushIntervalLabel)}
                    hint={t(strings.settings.voiceInputSection.flushIntervalHint)} control="wide">{(
                      <SettingsSlider
                        testId="vs-flush-interval"
                        value={vsConfig.flushIntervalMs}
                        onCommit={(next) => handleVsConfigChange({ flushIntervalMs: next })}
                        min={100}
                        max={5000}
                        step={50}
                        formatValue={(value) =>
                          t(strings.settings.voiceInputSection.msSuffix, { value })
                        }
                      />
                    )}</SettingsList.Row>

                  <SettingsList.Row
                    label={t(strings.settings.voiceInputSection.minChunkLabel)}
                    hint={t(strings.settings.voiceInputSection.minChunkHint)} control="wide">{(
                      <SettingsSlider
                        testId="vs-min-delta"
                        value={vsConfig.minDeltaBytes}
                        onCommit={(next) => handleVsConfigChange({ minDeltaBytes: next })}
                        min={512}
                        max={32768}
                        step={512}
                        formatValue={(value) =>
                          t(strings.settings.voiceInputSection.kbSuffix, { value: (value / 1024).toFixed(1) })
                        }
                      />
                    )}</SettingsList.Row>

                  <SettingsList.Row
                    label={t(strings.settings.voiceInputSection.overlapLabel)}
                    hint={t(strings.settings.voiceInputSection.overlapHint)} control="wide">{(
                      <SettingsSlider
                        testId="vs-overlap"
                        value={vsConfig.overlapBytes}
                        onCommit={(next) => handleVsConfigChange({ overlapBytes: next })}
                        min={0}
                        max={16384}
                        step={256}
                        formatValue={(value) =>
                          t(strings.settings.voiceInputSection.kbSuffix, { value: (value / 1024).toFixed(1) })
                        }
                      />
                    )}</SettingsList.Row>

                  <Button
                    data-testid="vs-reset-defaults"
                    variant="ghost"
                    size="sm"
                    className="h-8 px-3 text-xs text-wc-text-faint"
                    onClick={() => void resetVsConfig()}
                  >
                    <RotateCcw className="me-1 h-3.5 w-3.5" />
                    {t(strings.settings.voiceInputSection.resetDefaults)}
                  </Button>
                </>
              )}
            </div>
          )}
        </SettingsList.Group>
      )}
    </SettingsList>
  );
}

type DetectedBackend = "loading" | "whisper-stream" | "whisper-http" | "none";

function detectedBackendLabel(t: (key: string) => string, b: DetectedBackend): string {
  switch (b) {
    case "loading": return t(strings.settings.voiceInputSection.testMicDetecting);
    case "whisper-stream": return t(strings.settings.voiceInputSection.testMicBackendWhisperStream);
    case "whisper-http": return t(strings.settings.voiceInputSection.testMicBackendWhisperHttp);
    case "none": return t(strings.settings.voiceInputSection.testMicBackendNone);
  }
}

function TestMicrophoneCard() {
  const { t: tRaw } = useTranslation();
  const t = tRaw as unknown as (key: string, opts?: Record<string, unknown>) => string;
  const [detected, setDetected] = useState<DetectedBackend>("loading");
  const [recording, setRecording] = useState(false);
  const [remaining, setRemaining] = useState(0);
  const [transcribing, setTranscribing] = useState(false);
  const [result, setResult] = useState<{ providerUsed: string; transcript: string; elapsedMs: number } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const providerRef = useRef<TranscriptionProvider | null>(null);

  const detect = useCallback(async () => {
    setDetected("loading");
    try {
      const probe = await probeWhisperHealth();
      if (probe.whisperHealthy) {
        setDetected(probe.streamingAvailable ? "whisper-stream" : "whisper-http");
        return;
      }
      setDetected("none");
    } catch {
      setDetected("none");
    }
  }, []);

  useEffect(() => { void detect(); }, [detect]);

  const runTest = useCallback(async () => {
    setError(null);
    setResult(null);

    const backend = detected;
    let provider: TranscriptionProvider;
    let providerUsed: string;
    if (backend === "whisper-stream") {
      provider = new PcmVoiceStreamProvider();
      providerUsed = t(strings.settings.voiceInputSection.testMicBackendWhisperStream);
    } else if (backend === "whisper-http") {
      provider = new WhisperProvider();
      providerUsed = t(strings.settings.voiceInputSection.testMicBackendWhisperHttp);
    } else {
      setError(t(strings.settings.voiceInputSection.testMicBackendNone));
      return;
    }
    providerRef.current = provider;

    let finalText = "";
    provider.onResult = (text: string) => { finalText = text; };

    const startedAt = performance.now();
    try {
      setRecording(true);
      setRemaining(3);
      await provider.start();
      const tick = setInterval(() => setRemaining((r) => (r > 0 ? r - 1 : 0)), 1000);
      await new Promise<void>((resolve) => setTimeout(resolve, 3000));
      clearInterval(tick);
      setRecording(false);
      setTranscribing(true);
      provider.stop();
      // Wait briefly for onResult to fire (Whisper HTTP path is async).
      await new Promise<void>((resolve) => setTimeout(resolve, 1500));
      const elapsedMs = Math.round(performance.now() - startedAt);
      setResult({ providerUsed, transcript: finalText, elapsedMs });
    } catch (err) {
      setError(toErrorInfo(err).message);
      try { provider.stop(); } catch { /* ignore */ }
    } finally {
      setRecording(false);
      setTranscribing(false);
      setRemaining(0);
      providerRef.current = null;
    }
  }, [detected, t]);

  const busy = recording || transcribing || detected === "loading";

  return (
    <SettingsList.Group>
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-semibold uppercase tracking-[0.22em] text-wc-text-muted">
          {t(strings.settings.voiceInputSection.testMicHeading)}
        </div>
        <Button
          data-testid="mic-test-refresh-detection"
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={() => void detect()}
          disabled={busy}
        >
          <RefreshCw className="h-3.5 w-3.5" />
        </Button>
      </div>

      <p className="text-xs text-wc-text-faint">{t(strings.settings.voiceInputSection.testMicHint)}</p>

      <div className="flex items-center justify-between text-xs">
        <span className="text-wc-text-muted">{t(strings.settings.voiceInputSection.testMicDetected)}</span>
        <span data-testid="mic-test-detected-backend" className="font-medium text-wc-text-primary">
          {detectedBackendLabel(t, detected)}
        </span>
      </div>

      <Button
        data-testid="mic-test-record"
        variant="outline"
        size="sm"
        className="h-8 px-3 text-xs"
        onClick={() => void runTest()}
        disabled={busy || detected === "none"}
      >
        <Mic className="me-1 h-3.5 w-3.5" />
        {recording
          ? t(strings.settings.voiceInputSection.testMicRecording, { remaining })
          : transcribing
          ? t(strings.settings.voiceInputSection.testMicTranscribing)
          : t(strings.settings.voiceInputSection.testMicRecord)}
      </Button>

      {error && (
        <div className="flex items-start gap-2 text-xs text-wc-error-detail">
          <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span>{t(strings.settings.voiceInputSection.testMicError)}: {error}</span>
        </div>
      )}

      {result && (
        <div data-testid="mic-test-result" className="rounded-md border border-wc-default bg-wc-surface-base p-2 text-xs">
          <div className="flex justify-between">
            <span className="text-wc-text-muted">{t(strings.settings.voiceInputSection.testMicProviderUsed)}</span>
            <span className="font-medium text-wc-text-primary">{result.providerUsed}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-wc-text-muted">{t(strings.settings.voiceInputSection.testMicElapsed)}</span>
            <span className="text-wc-text-primary">{t(strings.settings.voiceInputSection.testMicMsSuffix, { value: result.elapsedMs })}</span>
          </div>
          <div>
            <div className="text-wc-text-muted">{t(strings.settings.voiceInputSection.testMicTranscript)}</div>
            <div className="rounded bg-wc-surface-raised p-1.5 text-wc-text-primary">
              {result.transcript || <span className="text-wc-text-faint">{t(strings.settings.voiceInputSection.testMicNoTranscript)}</span>}
            </div>
          </div>
        </div>
      )}
    </SettingsList.Group>
  );
}
