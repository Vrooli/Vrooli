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
} from "../../api/voice";
import { fetchCapabilities, type CapabilityState } from "../../api/capabilities";
import { VOICE_COMMANDS } from "../../hooks/voice/commands";
import {
  createWakeWordEngine,
  MIN_ENROLLMENT_SAMPLES,
  MAX_ENROLLMENT_SAMPLES,
  useWakeWordTest,
  type AudioFeatures,
  type WakeWordTemplate,
} from "../../domains/audio";
import { formatShortcutFromEvent } from "../../lib/shortcutParser";
import { Button } from "../ui/button";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";

export default function VoiceInputSection() {
  const { t } = useTranslation();
  const voiceEnabled = useWorkspaceStore((state) => state.voiceEnabled);
  const setVoiceEnabled = useWorkspaceStore((state) => state.setVoiceEnabled);
  const voiceShortcut = useWorkspaceStore((state) => state.voiceShortcut);
  const setVoiceShortcut = useWorkspaceStore((state) => state.setVoiceShortcut);
  const vadAutoStop = useWorkspaceStore((state) => state.vadAutoStop);
  const setVadAutoStop = useWorkspaceStore((state) => state.setVadAutoStop);
  const vadSilenceTimeoutMs = useWorkspaceStore((state) => state.vadSilenceTimeoutMs);
  const setVadSilenceTimeoutMs = useWorkspaceStore((state) => state.setVadSilenceTimeoutMs);
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
  const wwStreamRef = useRef<MediaStream | null>(null);
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
  const enrollmentStreamRef = useRef<MediaStream | null>(null);
  const enrollmentTickerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const enrollmentStopTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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
        wwSamplesRef.current = [...config.template.samples];
        // No audio blobs available for previously saved samples
        wwAudioBlobsRef.current = config.template.samples.map(() => null);
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
    wwChunksRef.current = [];
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      wwStreamRef.current = stream;
      const recorder = new MediaRecorder(stream, {
        mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus") ? "audio/webm;codecs=opus" : "audio/webm",
      });
      wwRecorderRef.current = recorder;
      recorder.ondataavailable = (e) => { if (e.data.size > 0) wwChunksRef.current.push(e.data); };
      recorder.onerror = () => { setWwRecordingIdx(null); setWakeWordError(t(strings.settings.voiceInputSection.recordingFailed)); };
      recorder.onstop = async () => {
        wwStreamRef.current?.getTracks().forEach((track) => track.stop());
        wwStreamRef.current = null;
        const blob = new Blob(wwChunksRef.current, { type: "audio/webm" });
        if (blob.size === 0) { setWwRecordingIdx(null); setWakeWordError(t(strings.settings.voiceInputSection.recordingEmpty)); return; }
        // Decode audio and extract MFCC features
        try {
          const arrayBuf = await blob.arrayBuffer();
          const audioCtx = new AudioContext({ sampleRate: 16000 });
          const decoded = await audioCtx.decodeAudioData(arrayBuf);
          const pcm = decoded.getChannelData(0);
          await audioCtx.close();
          const features = wwEngineRef.current.extractFeatures(pcm, 16000);
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
      wwStreamRef.current?.getTracks().forEach((track) => track.stop());
      if (wwPlaybackRef.current) { wwPlaybackRef.current.pause(); wwPlaybackRef.current = null; }
    };
  }, []);

  const requestMicPermission = useCallback(async () => {
    setMicRequesting(true);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      stream.getTracks().forEach((track) => track.stop());
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
        setStorePersistentMode(config.persistentMode);
        setStoreWakeWordEnabled(config.wakeWordEnabled ?? false);
        setStoreSegmentSilenceMs(config.segmentSilenceMs || 1500);
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
  }, [setStorePersistentMode, setStoreWakeWordEnabled, setStoreSegmentSilenceMs]);

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
      enrollmentStreamRef.current?.getTracks().forEach((track) => track.stop());
    };
  }, []);

  const handleVsConfigChange = useCallback((patch: Partial<VoiceStreamConfig>) => {
    setVsConfig((current) => (current ? { ...current, ...patch } : null));
    // Update workspace store immediately for reactive consumers (useVoiceInput)
    if (patch.persistentMode !== undefined) setStorePersistentMode(patch.persistentMode);
    if (patch.wakeWordEnabled !== undefined) setStoreWakeWordEnabled(patch.wakeWordEnabled);
    if (patch.segmentSilenceMs !== undefined) setStoreSegmentSilenceMs(patch.segmentSilenceMs);
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
  }, [setStorePersistentMode, setStoreWakeWordEnabled, setStoreSegmentSilenceMs]);

  const saveWakeWord = useCallback(async () => {
    const samples = wwSamplesRef.current.filter((s): s is AudioFeatures => s !== null);
    if (samples.length < MIN_ENROLLMENT_SAMPLES) {
      setWakeWordError(t(strings.settings.voiceInputSection.minSamplesNeeded, { min: MIN_ENROLLMENT_SAMPLES }));
      return;
    }
    setWakeWordError(null);
    const threshold = useWorkspaceStore.getState().wakeWordThreshold;
    const template: WakeWordTemplate = {
      samples,
      label: wwLabel.trim() || "Hey Vrooli",
      threshold,
      updatedAt: new Date().toISOString(),
    };
    try {
      const updated = await updateWakeWordConfig(template);
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
        wakeWordThreshold: 0.65,
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
    enrollmentChunksRef.current = [];
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      enrollmentStreamRef.current = stream;
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
        setEnrollmentState("error");
        setEnrollmentMessage(t(strings.settings.voiceInputSection.enrollmentRecordingFailed));
      };
      recorder.onstop = () => {
        const blob = new Blob(enrollmentChunksRef.current, { type: "audio/webm" });
        enrollmentStreamRef.current?.getTracks().forEach((track) => track.stop());
        enrollmentStreamRef.current = null;
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
  }, [loadSpeakerStatus, profileDisplayName, reEnrollTargetId, stopEnrollmentRecording, t]);

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
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow={t(strings.settings.voiceInputSection.eyebrow)}
        title={t(strings.settings.voiceInputSection.title)}
        description={t(strings.settings.voiceInputSection.description)}
      />

      <SettingsCard className="space-y-4">
        <SettingsRow
          label={t(strings.settings.voiceInputSection.voiceInputLabel)}
          hint={t(strings.settings.voiceInputSection.voiceInputHint)}
          control={(
            <SettingsToggle
              testId="voice-enabled-toggle"
              checked={voiceEnabled}
              onClick={() => setVoiceEnabled(!voiceEnabled)}
            />
          )}
        />

        {voiceEnabled && (
          <>
            <SettingsRow
              label={t(strings.settings.voiceInputSection.autoStopLabel)}
              hint={t(strings.settings.voiceInputSection.autoStopHint)}
              control={(
                <SettingsToggle
                  testId="vad-auto-stop-toggle"
                  checked={vadAutoStop}
                  onClick={() => setVadAutoStop(!vadAutoStop)}
                />
              )}
            />

            {vadAutoStop && (
              <SettingsRow
                label={t(strings.settings.voiceInputSection.silenceTimeoutLabel)}
                hint={t(strings.settings.voiceInputSection.silenceTimeoutHint)}
                control={(
                  <div className="flex items-center gap-2">
                    <input
                      data-testid="vad-silence-timeout-slider"
                      type="range"
                      min={1000}
                      max={5000}
                      step={250}
                      value={vadSilenceTimeoutMs}
                      onChange={(event) => setVadSilenceTimeoutMs(Number(event.target.value))}
                      className="w-24 accent-wc-accent"
                    />
                    <span className="w-9 text-end text-xs text-wc-text-muted">
                      {t(strings.settings.voiceInputSection.secondsShort, { value: (vadSilenceTimeoutMs / 1000).toFixed(1) })}
                    </span>
                  </div>
                )}
              />
            )}

            <SettingsRow
              label={t(strings.settings.voiceInputSection.languageLabel)}
              hint={t(strings.settings.voiceInputSection.languageHint)}
              control={(
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
              )}
            />
          </>
        )}
      </SettingsCard>

      {voiceEnabled && (
        <SettingsCard className="space-y-4">
          <SettingsSectionIntro
            eyebrow={t(strings.settings.voiceInputSection.persistentEyebrow)}
            title={t(strings.settings.voiceInputSection.persistentTitle)}
            description={t(strings.settings.voiceInputSection.persistentDescription)}
          />

          {vsConfig && (
            <>
              <SettingsRow
                label={t(strings.settings.voiceInputSection.persistentModeLabel)}
                hint={t(strings.settings.voiceInputSection.persistentModeHint)}
                control={(
                  <SettingsToggle
                    testId="persistent-mode-toggle"
                    checked={vsConfig.persistentMode}
                    onClick={() => handleVsConfigChange({ persistentMode: !vsConfig.persistentMode })}
                  />
                )}
              />

              {vsConfig.persistentMode && (
                <>
                  <SettingsRow
                    label={t(strings.settings.voiceInputSection.wakeWordLabel)}
                    hint={t(strings.settings.voiceInputSection.wakeWordHint)}
                    control={(
                      <SettingsToggle
                        testId="wake-word-toggle"
                        checked={vsConfig.wakeWordEnabled}
                        onClick={() => {
                          if (!vsConfig.wakeWordEnabled && !wakeWordConfig?.configured) {
                            setWakeWordError(t(strings.settings.voiceInputSection.wakeWordRequireSaveFirst));
                            return;
                          }
                          handleVsConfigChange({ wakeWordEnabled: !vsConfig.wakeWordEnabled });
                        }}
                      />
                    )}
                  />

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

                    <div className="space-y-1.5">
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

                    <SettingsRow
                      label={t(strings.settings.voiceInputSection.sensitivityLabel)}
                      hint={t(strings.settings.voiceInputSection.sensitivityHint)}
                      control={(
                        <div className="flex items-center gap-2">
                          <input
                            data-testid="wake-word-threshold-slider"
                            type="range"
                            min={0.1}
                            max={0.95}
                            step={0.05}
                            value={useWorkspaceStore.getState().wakeWordThreshold}
                            onChange={(e) => {
                              const val = Number(e.target.value);
                              useWorkspaceStore.getState().setWakeWordThreshold(val);
                              handleVsConfigChange({ wakeWordThreshold: val });
                            }}
                            className="w-24 accent-wc-accent"
                          />
                          <span className="w-10 text-end text-xs text-wc-text-muted">
                            {useWorkspaceStore.getState().wakeWordThreshold.toFixed(2)}
                          </span>
                        </div>
                      )}
                    />

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
                          <div className="space-y-1">
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
                                style={{ left: `${useWorkspaceStore.getState().wakeWordThreshold * 100}%` }}
                                title={t(strings.settings.voiceInputSection.thresholdTooltip, { threshold: useWorkspaceStore.getState().wakeWordThreshold.toFixed(2) })}
                              />
                            </div>
                          </div>
                        )}

                        {wakeWordTest.state.history.length > 1 && (
                          <div className="space-y-1">
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
                                      style={{ left: `${useWorkspaceStore.getState().wakeWordThreshold * 100}%` }}
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

                  <SettingsRow
                    label={t(strings.settings.voiceInputSection.segmentSilenceLabel)}
                    hint={t(strings.settings.voiceInputSection.segmentSilenceHint)}
                    control={(
                      <div className="flex items-center gap-2">
                        <input
                          data-testid="segment-silence-slider"
                          type="range"
                          min={800}
                          max={3000}
                          step={100}
                          value={vsConfig.segmentSilenceMs}
                          onChange={(event) => handleVsConfigChange({ segmentSilenceMs: Number(event.target.value) })}
                          className="w-24 accent-wc-accent"
                        />
                        <span className="w-9 text-end text-xs text-wc-text-muted">
                          {t(strings.settings.voiceInputSection.secondsShort, { value: (vsConfig.segmentSilenceMs / 1000).toFixed(1) })}
                        </span>
                      </div>
                    )}
                  />

                  <div className="space-y-1.5">
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
        </SettingsCard>
      )}

      {voiceEnabled && (
        <SettingsCard className="space-y-4">
          <SettingsSectionIntro
            eyebrow={t(strings.settings.voiceInputSection.performanceEyebrow)}
            title={t(strings.settings.voiceInputSection.performanceTitle)}
            description={t(strings.settings.voiceInputSection.performanceDescription)}
          />
          <SettingsRow
            label={t(strings.settings.voiceInputSection.lowLatencyLabel)}
            hint={t(strings.settings.voiceInputSection.lowLatencyHint)}
            control={(
              <SettingsToggle
                testId="low-latency-voice-toggle"
                checked={useWorkspaceStore.getState().lowLatencyVoice}
                onClick={() => {
                  const store = useWorkspaceStore.getState();
                  store.setLowLatencyVoice(!store.lowLatencyVoice);
                }}
              />
            )}
          />
        </SettingsCard>
      )}

      {voiceEnabled && (
        <SettingsCard className="space-y-4">
          <SettingsSectionIntro
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

          <SettingsRow
            label={t(strings.settings.voiceInputSection.resourceStatusLabel)}
            hint={speakerLoading
              ? t(strings.settings.voiceInputSection.resourceStatusChecking)
              : speakerStatus?.capability === "available"
                ? (speakerStatus.resourceReady ? t(strings.settings.voiceInputSection.resourceReady) : t(strings.settings.voiceInputSection.resourceReachable))
                : t(strings.settings.voiceInputSection.resourceUnavailable)}
            control={(
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
                  className="h-8 w-8"
                  onClick={() => void loadSpeakerStatus()}
                  title={t(strings.settings.voiceInputSection.refreshSpeakerTitle)}
                >
                  <RefreshCw className={`h-3.5 w-3.5 text-wc-text-faint ${speakerLoading ? "animate-spin" : ""}`} />
                </Button>
              </div>
            )}
          />

          <SettingsRow
            label={t(strings.settings.voiceInputSection.useSpeakerLabel)}
            hint={t(strings.settings.voiceInputSection.useSpeakerHint)}
            control={(
              <SettingsToggle
                testId="speaker-verification-toggle"
                checked={speakerStatus?.config.enabled ?? false}
                onClick={() => {
                  const next = !(speakerStatus?.config.enabled ?? false);
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
            )}
          />

          <div className="space-y-1.5">
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

          <SettingsRow
            label={t(strings.settings.voiceInputSection.modeLabel)}
            hint={t(strings.settings.voiceInputSection.modeHint)}
            control={(
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
            )}
          />

          <SettingsRow
            label={t(strings.settings.voiceInputSection.thresholdLabel)}
            hint={t(strings.settings.voiceInputSection.thresholdHint)}
            control={(
              <div className="flex items-center gap-2">
                <input
                  data-testid="speaker-threshold-slider"
                  type="range"
                  min={0.1}
                  max={0.99}
                  step={0.01}
                  value={speakerStatus?.config.threshold ?? 0.35}
                  onChange={(event) => {
                    void persistSpeakerConfig({ threshold: Number(event.target.value) });
                  }}
                  className="w-24 accent-wc-accent"
                />
                <span className="w-10 text-end text-xs text-wc-text-muted">
                  {(speakerStatus?.config.threshold ?? 0.35).toFixed(2)}
                </span>
              </div>
            )}
          />

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
        </SettingsCard>
      )}

      <SettingsCard className="space-y-4">
        <SettingsRow
          label={t(strings.settings.voiceInputSection.shortcutLabel)}
          hint={t(strings.settings.voiceInputSection.shortcutHint)}
          control={(
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
                className="h-8 w-8"
                onClick={() => setRecordingShortcut(true)}
                title={t(strings.settings.voiceInputSection.changeShortcut)}
              >
                <Keyboard className="h-3.5 w-3.5 text-wc-text-faint" />
              </Button>
            </div>
          )}
        />
      </SettingsCard>

      <SettingsCard className="space-y-3">
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

        <div className="space-y-2">
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
      </SettingsCard>

      <SettingsCard className="space-y-3">
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
      </SettingsCard>

      {voiceEnabled && (
        <SettingsCard className="space-y-3">
          <button
            data-testid="advanced-streaming-toggle"
            className="flex w-full items-center gap-1 text-start text-[11px] font-semibold uppercase tracking-[0.22em] text-wc-text-muted"
            onClick={() => setAdvancedOpen(!advancedOpen)}
          >
            {advancedOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
            {t(strings.settings.voiceInputSection.advancedStreaming)}
          </button>

          {advancedOpen && (
            <div className="space-y-4">
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
                  <SettingsRow
                    label={t(strings.settings.voiceInputSection.flushIntervalLabel)}
                    hint={t(strings.settings.voiceInputSection.flushIntervalHint)}
                    control={(
                      <div className="flex items-center gap-2">
                        <input
                          data-testid="vs-flush-interval"
                          type="range"
                          min={100}
                          max={5000}
                          step={50}
                          value={vsConfig.flushIntervalMs}
                          onChange={(event) => handleVsConfigChange({ flushIntervalMs: Number(event.target.value) })}
                          className="w-24 accent-wc-accent"
                        />
                        <span className="w-12 text-end text-xs text-wc-text-muted">{t(strings.settings.voiceInputSection.msSuffix, { value: vsConfig.flushIntervalMs })}</span>
                      </div>
                    )}
                  />

                  <SettingsRow
                    label={t(strings.settings.voiceInputSection.minChunkLabel)}
                    hint={t(strings.settings.voiceInputSection.minChunkHint)}
                    control={(
                      <div className="flex items-center gap-2">
                        <input
                          data-testid="vs-min-delta"
                          type="range"
                          min={512}
                          max={32768}
                          step={512}
                          value={vsConfig.minDeltaBytes}
                          onChange={(event) => handleVsConfigChange({ minDeltaBytes: Number(event.target.value) })}
                          className="w-24 accent-wc-accent"
                        />
                        <span className="w-12 text-end text-xs text-wc-text-muted">
                          {t(strings.settings.voiceInputSection.kbSuffix, { value: (vsConfig.minDeltaBytes / 1024).toFixed(1) })}
                        </span>
                      </div>
                    )}
                  />

                  <SettingsRow
                    label={t(strings.settings.voiceInputSection.overlapLabel)}
                    hint={t(strings.settings.voiceInputSection.overlapHint)}
                    control={(
                      <div className="flex items-center gap-2">
                        <input
                          data-testid="vs-overlap"
                          type="range"
                          min={0}
                          max={16384}
                          step={256}
                          value={vsConfig.overlapBytes}
                          onChange={(event) => handleVsConfigChange({ overlapBytes: Number(event.target.value) })}
                          className="w-24 accent-wc-accent"
                        />
                        <span className="w-12 text-end text-xs text-wc-text-muted">
                          {t(strings.settings.voiceInputSection.kbSuffix, { value: (vsConfig.overlapBytes / 1024).toFixed(1) })}
                        </span>
                      </div>
                    )}
                  />

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
        </SettingsCard>
      )}
    </div>
  );
}
