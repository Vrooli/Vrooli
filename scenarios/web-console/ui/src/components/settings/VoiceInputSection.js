import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AlertCircle, CheckCircle, ChevronDown, ChevronRight, Circle, Keyboard, Mic, Play, RefreshCw, RotateCcw, Square, Trash2, UserRound, } from "lucide-react";
import { strings } from "../../consts/strings";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { toErrorInfo } from "../../lib/errors";
import { clearSpeakerVerificationProfile, deleteWakeWordConfig, deleteSpeakerVerificationProfile, enrollSpeakerVerificationProfile, getSpeakerVerificationStatus, getVoiceStreamConfig, getWakeWordConfig, removeSpeakerVerificationProfile, updateWakeWordConfig, updateSpeakerVerificationConfig, updateVoiceStreamConfig, } from "../../audio-integration";
import { fetchCapabilities } from "../../api/capabilities";
import { probeWhisperHealth } from "../../audio-integration";
import { PcmVoiceStreamProvider, WhisperProvider, WebSpeechProvider } from "../../audio-integration";
import { getSharedAudioContext } from "../../audio-integration/hooks/voice/sharedAudioContext";
import { createAudioFilterChain } from "../../audio-integration/hooks/voice/audioUtils";
import { acquireMicStream, releaseMicLease } from "../../audio-integration/hooks/voice/micOwnership";
import { VOICE_COMMANDS } from "../../hooks/voice/commands";
import { bytesToFeatures, createWakeWordEngine, DEFAULT_WAKE_WORD_THRESHOLD, MIN_ENROLLMENT_SAMPLES, MAX_ENROLLMENT_SAMPLES, useWakeWordTest, WAKE_WORD_AUDIO_CONSTRAINTS, } from "../../audio-integration";
import { formatShortcutFromEvent } from "../../lib/shortcutParser";
import { Button } from "../ui/button";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";
/** Lease reasons meaning the page/OS pulled the mic, so an in-flight settings
 *  capture must be cancelled (not processed/uploaded). */
const LIFECYCLE_CANCEL_REASONS = new Set(["hidden", "pagehide", "freeze", "ended"]);
export default function VoiceInputSection() {
    const { t } = useTranslation();
    const voiceEnabled = useWorkspaceStore((state) => state.voiceEnabled);
    const setVoiceEnabled = useWorkspaceStore((state) => state.setVoiceEnabled);
    const voiceShortcut = useWorkspaceStore((state) => state.voiceShortcut);
    const setVoiceShortcut = useWorkspaceStore((state) => state.setVoiceShortcut);
    const vadAutoStop = useWorkspaceStore((state) => state.vadAutoStop);
    const setVadAutoStop = useWorkspaceStore((state) => state.setVadAutoStop);
    const vadSilenceTimeoutMs = useWorkspaceStore((state) => state.vadSilenceTimeoutMs);
    const setStoreVadSilenceTimeoutMs = useWorkspaceStore((state) => state.setVadSilenceTimeoutMs);
    const voiceLanguage = useWorkspaceStore((state) => state.voiceLanguage);
    const setVoiceLanguage = useWorkspaceStore((state) => state.setVoiceLanguage);
    const setStorePersistentMode = useWorkspaceStore((state) => state.setPersistentMode);
    const setStoreWakeWordEnabled = useWorkspaceStore((state) => state.setWakeWordEnabled);
    const setStoreSegmentSilenceMs = useWorkspaceStore((state) => state.setSegmentSilenceMs);
    const [recordingShortcut, setRecordingShortcut] = useState(false);
    const [voiceCaps, setVoiceCaps] = useState([]);
    const [voiceCapsLoading, setVoiceCapsLoading] = useState(false);
    const [voiceCapsError, setVoiceCapsError] = useState(null);
    const [micPermission, setMicPermission] = useState("unknown");
    const [micRequesting, setMicRequesting] = useState(false);
    const [vsConfig, setVsConfig] = useState(null);
    const [vsConfigLoading, setVsConfigLoading] = useState(false);
    const [vsConfigError, setVsConfigError] = useState(null);
    const [speakerStatus, setSpeakerStatus] = useState(null);
    const [speakerLoading, setSpeakerLoading] = useState(false);
    const [speakerError, setSpeakerError] = useState(null);
    const [enrollmentState, setEnrollmentState] = useState("idle");
    const [enrollmentSeconds, setEnrollmentSeconds] = useState(0);
    // Live mic level (0..1) shown as a meter while enrolling, so the user can
    // see their voice is being captured. Driven by the same AnalyserNode the
    // streaming mic uses (createAudioFilterChain) — its absence was why
    // enrollment "showed no volume".
    const [enrollmentLevel, setEnrollmentLevel] = useState(0);
    const [enrollmentMessage, setEnrollmentMessage] = useState(null);
    const [profileDisplayName, setProfileDisplayName] = useState("My Voice");
    const [reEnrollTargetId, setReEnrollTargetId] = useState(null);
    const [advancedOpen, setAdvancedOpen] = useState(false);
    // Wake word state
    const [wakeWordConfig, setWakeWordConfig] = useState(null);
    const [, setWakeWordLoading] = useState(false);
    const [wakeWordError, setWakeWordError] = useState(null);
    const [wwRecordingIdx, setWwRecordingIdx] = useState(null);
    const [wwRecordingSeconds, setWwRecordingSeconds] = useState(0);
    const [wwPlayingIdx, setWwPlayingIdx] = useState(null);
    const [wwTestResult, setWwTestResult] = useState(null);
    const [wwLabel, setWwLabel] = useState("Hey Vrooli");
    const wwSamplesRef = useRef([]);
    const wwAudioBlobsRef = useRef([]);
    const wwRecorderRef = useRef(null);
    const wwLeaseRef = useRef(null);
    /** Set when the mic lease is pulled by page/OS lifecycle so onstop skips
     *  feature extraction of the cancelled capture. */
    const wwCancelledRef = useRef(false);
    const wwChunksRef = useRef([]);
    const wwTickerRef = useRef(null);
    const wwStopTimerRef = useRef(null);
    const wwEngineRef = useRef(createWakeWordEngine());
    const wwPlaybackRef = useRef(null);
    // Live wake word test hook
    const wwTestSamples = wwSamplesRef.current.filter((s) => s !== null);
    const wakeWordTest = useWakeWordTest({
        engine: wwEngineRef.current,
        samples: wwTestSamples,
        threshold: useWorkspaceStore.getState().wakeWordThreshold,
        disabled: wwRecordingIdx !== null,
    });
    const saveTimerRef = useRef(null);
    const enrollmentChunksRef = useRef([]);
    const enrollmentRecorderRef = useRef(null);
    const enrollmentLeaseRef = useRef(null);
    /** Set when the mic lease is pulled by page/OS lifecycle so onstop skips
     *  the enroll upload of the cancelled capture. */
    const enrollmentCancelledRef = useRef(false);
    const enrollmentTickerRef = useRef(null);
    const enrollmentStopTimerRef = useRef(null);
    // Web Audio graph used purely for the live level meter + Chrome's render
    // keepalive (see createAudioFilterChain). Recording itself still uses the
    // RAW mic stream so enrollment embeddings match the streaming verification
    // path's audio characteristics. Disconnected in teardownEnrollmentAudio.
    const enrollmentSourceRef = useRef(null);
    const enrollmentNodesRef = useRef([]);
    const enrollmentAnalyserRef = useRef(null);
    const enrollmentLevelTimerRef = useRef(null);
    const hasWebSpeech = typeof window !== "undefined" &&
        Boolean("SpeechRecognition" in window || "webkitSpeechRecognition" in window);
    const checkMicPermission = useCallback(async () => {
        try {
            const result = await navigator.permissions.query({ name: "microphone" });
            setMicPermission(result.state);
            result.onchange = () => setMicPermission(result.state);
        }
        catch {
            setMicPermission("unknown");
        }
    }, []);
    useEffect(() => {
        void checkMicPermission();
    }, [checkMicPermission]);
    // ── Wake word config loading ──
    const loadWakeWordConfig = useCallback(async (signal) => {
        setWakeWordLoading(true);
        setWakeWordError(null);
        try {
            const config = await getWakeWordConfig();
            if (signal?.cancelled)
                return;
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
                const features = await Promise.all(persisted.map((s) => bytesToFeatures(s.audio, wwEngineRef.current).catch(() => null)));
                if (signal?.cancelled)
                    return;
                wwSamplesRef.current = features;
                wwAudioBlobsRef.current = blobs;
            }
        }
        catch (error) {
            if (!signal?.cancelled)
                setWakeWordError(toErrorInfo(error).message);
        }
        finally {
            if (!signal?.cancelled)
                setWakeWordLoading(false);
        }
    }, []);
    useEffect(() => {
        if (!voiceEnabled)
            return;
        const signal = { cancelled: false };
        void loadWakeWordConfig(signal);
        return () => { signal.cancelled = true; };
    }, [voiceEnabled, loadWakeWordConfig]);
    const stopWwRecording = useCallback(() => {
        if (wwTickerRef.current) {
            clearInterval(wwTickerRef.current);
            wwTickerRef.current = null;
        }
        if (wwStopTimerRef.current) {
            clearTimeout(wwStopTimerRef.current);
            wwStopTimerRef.current = null;
        }
        if (wwRecorderRef.current?.state === "recording")
            wwRecorderRef.current.stop();
    }, []);
    const startWwRecording = useCallback(async (slotIdx) => {
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
                    if (!LIFECYCLE_CANCEL_REASONS.has(reason))
                        return;
                    // Page/OS pulled the mic mid-capture → cancel; onstop must not extract.
                    wwCancelledRef.current = true;
                    if (wwTickerRef.current) {
                        clearInterval(wwTickerRef.current);
                        wwTickerRef.current = null;
                    }
                    if (wwStopTimerRef.current) {
                        clearTimeout(wwStopTimerRef.current);
                        wwStopTimerRef.current = null;
                    }
                    if (wwRecorderRef.current?.state === "recording")
                        wwRecorderRef.current.stop();
                    setWwRecordingIdx(null);
                },
            });
            wwLeaseRef.current = lease;
            const recorder = new MediaRecorder(lease.stream, {
                mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus") ? "audio/webm;codecs=opus" : "audio/webm",
            });
            wwRecorderRef.current = recorder;
            recorder.ondataavailable = (e) => { if (e.data.size > 0)
                wwChunksRef.current.push(e.data); };
            recorder.onerror = () => {
                releaseMicLease(wwLeaseRef.current, "provider-error");
                wwLeaseRef.current = null;
                setWwRecordingIdx(null);
                setWakeWordError(t(strings.settings.voiceInputSection.recordingFailed));
            };
            recorder.onstop = async () => {
                releaseMicLease(wwLeaseRef.current, "manual-stop");
                wwLeaseRef.current = null;
                if (wwCancelledRef.current) {
                    wwCancelledRef.current = false;
                    setWwRecordingIdx(null);
                    return;
                }
                const blob = new Blob(wwChunksRef.current, { type: "audio/webm" });
                if (blob.size === 0) {
                    setWwRecordingIdx(null);
                    setWakeWordError(t(strings.settings.voiceInputSection.recordingEmpty));
                    return;
                }
                // Decode audio and extract MFCC features via the shared helper — the
                // same decode path used on load, so re-derived features match exactly.
                try {
                    const bytes = new Uint8Array(await blob.arrayBuffer());
                    const features = await bytesToFeatures(bytes, wwEngineRef.current);
                    const next = [...wwSamplesRef.current];
                    while (next.length <= slotIdx)
                        next.push(null);
                    next[slotIdx] = features;
                    wwSamplesRef.current = next;
                    const blobs = [...wwAudioBlobsRef.current];
                    while (blobs.length <= slotIdx)
                        blobs.push(null);
                    blobs[slotIdx] = blob;
                    wwAudioBlobsRef.current = blobs;
                }
                catch (err) {
                    setWakeWordError(t(strings.settings.voiceInputSection.featureExtractionFailed, { error: String(err) }));
                }
                setWwRecordingIdx(null);
            };
            recorder.start(250);
            wwTickerRef.current = setInterval(() => setWwRecordingSeconds((v) => v + 1), 1000);
            wwStopTimerRef.current = setTimeout(() => stopWwRecording(), 5000);
        }
        catch (error) {
            setWwRecordingIdx(null);
            setWakeWordError(toErrorInfo(error).message);
        }
    }, [stopWwRecording, t]);
    const playWwSample = useCallback((idx) => {
        const blob = wwAudioBlobsRef.current[idx];
        if (!blob)
            return;
        if (wwPlaybackRef.current) {
            wwPlaybackRef.current.pause();
            wwPlaybackRef.current = null;
        }
        const url = URL.createObjectURL(blob);
        const audio = new Audio(url);
        wwPlaybackRef.current = audio;
        setWwPlayingIdx(idx);
        audio.onended = () => { setWwPlayingIdx(null); URL.revokeObjectURL(url); };
        audio.play().catch(() => setWwPlayingIdx(null));
    }, []);
    const removeWwSample = useCallback((idx) => {
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
            if (wwTickerRef.current)
                clearInterval(wwTickerRef.current);
            if (wwStopTimerRef.current)
                clearTimeout(wwStopTimerRef.current);
            releaseMicLease(wwLeaseRef.current, "unmount");
            wwLeaseRef.current = null;
            if (wwPlaybackRef.current) {
                wwPlaybackRef.current.pause();
                wwPlaybackRef.current = null;
            }
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
        }
        catch {
            setMicPermission("denied");
        }
        finally {
            setMicRequesting(false);
        }
    }, [setVoiceEnabled, voiceEnabled]);
    const loadVoiceCaps = useCallback(async (signal) => {
        setVoiceCapsLoading(true);
        setVoiceCapsError(null);
        try {
            const data = await fetchCapabilities();
            if (signal?.cancelled)
                return;
            setVoiceCaps(data.capabilities);
        }
        catch (error) {
            if (signal?.cancelled)
                return;
            setVoiceCapsError(toErrorInfo(error).message);
        }
        finally {
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
    const loadVoiceStreamConfig = useCallback(async (signal) => {
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
        }
        catch (error) {
            if (!signal?.cancelled) {
                setVsConfigError(toErrorInfo(error).message);
            }
        }
        finally {
            if (!signal?.cancelled) {
                setVsConfigLoading(false);
            }
        }
    }, [setStorePersistentMode, setStoreWakeWordEnabled, setStoreSegmentSilenceMs, setStoreVadSilenceTimeoutMs]);
    const loadSpeakerStatus = useCallback(async (signal) => {
        setSpeakerLoading(true);
        setSpeakerError(null);
        try {
            const status = await getSpeakerVerificationStatus();
            if (!signal?.cancelled) {
                setSpeakerStatus(status);
            }
        }
        catch (error) {
            if (!signal?.cancelled) {
                setSpeakerError(toErrorInfo(error).message);
            }
        }
        finally {
            if (!signal?.cancelled)
                setSpeakerLoading(false);
        }
    }, []);
    // Load voice config when voice is enabled (needed for persistent mode settings
    // and advanced streaming section)
    useEffect(() => {
        if (!voiceEnabled)
            return;
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
            if (enrollmentTickerRef.current)
                clearInterval(enrollmentTickerRef.current);
            if (enrollmentStopTimerRef.current)
                clearTimeout(enrollmentStopTimerRef.current);
            if (enrollmentLevelTimerRef.current)
                clearInterval(enrollmentLevelTimerRef.current);
            try {
                enrollmentSourceRef.current?.disconnect();
            }
            catch { /* noop */ }
            for (const node of enrollmentNodesRef.current) {
                try {
                    node.disconnect();
                }
                catch { /* noop */ }
            }
            releaseMicLease(enrollmentLeaseRef.current, "unmount");
            enrollmentLeaseRef.current = null;
        };
    }, []);
    const handleVsConfigChange = useCallback((patch) => {
        setVsConfig((current) => (current ? { ...current, ...patch } : null));
        // Update workspace store immediately for reactive consumers (useVoiceInput).
        // Both vadSilenceMs and segmentSilenceMs patches map onto the same two
        // store fields: the silence-timeout slider, the segment-silence slider,
        // and the server's vad_silence_ms collapse to a single source of truth.
        if (patch.persistentMode !== undefined)
            setStorePersistentMode(patch.persistentMode);
        if (patch.wakeWordEnabled !== undefined)
            setStoreWakeWordEnabled(patch.wakeWordEnabled);
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
            }
            catch (error) {
                setVsConfigError(toErrorInfo(error).message);
            }
        }, 500);
    }, [setStorePersistentMode, setStoreWakeWordEnabled, setStoreSegmentSilenceMs, setStoreVadSilenceTimeoutMs]);
    const saveWakeWord = useCallback(async () => {
        // We persist RAW audio, not MFCC features — collect the recorded blob for
        // every slot that has both a blob and successfully-extracted features.
        const blobs = [];
        for (let i = 0; i < wwAudioBlobsRef.current.length; i++) {
            const blob = wwAudioBlobsRef.current[i];
            if (blob && wwSamplesRef.current[i])
                blobs.push(blob);
        }
        const featureCount = wwSamplesRef.current.filter((s) => s !== null).length;
        if (blobs.length < MIN_ENROLLMENT_SAMPLES) {
            // Distinguish "not enough samples" from "samples present but their audio
            // is missing" (only possible for a slot that never captured a blob).
            setWakeWordError(featureCount >= MIN_ENROLLMENT_SAMPLES
                ? t(strings.settings.voiceInputSection.wakeWordReRecordNeeded)
                : t(strings.settings.voiceInputSection.minSamplesNeeded, { min: MIN_ENROLLMENT_SAMPLES }));
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
        }
        catch (error) {
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
        }
        catch (error) {
            setWakeWordError(toErrorInfo(error).message);
        }
    }, [handleVsConfigChange]);
    const testWakeWord = useCallback(async () => {
        const samples = wwSamplesRef.current.filter((s) => s !== null);
        if (samples.length < 2) {
            setWwTestResult(t(strings.settings.voiceInputSection.testNeedSamples));
            return;
        }
        const threshold = useWorkspaceStore.getState().wakeWordThreshold;
        let matches = 0;
        let total = 0;
        for (let i = 0; i < samples.length; i++) {
            const target = samples[i];
            if (!target)
                continue;
            const others = samples.filter((_, j) => j !== i);
            const result = wwEngineRef.current.compareBest(target, others, threshold);
            total++;
            if (result.isMatch)
                matches++;
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
        }
        catch (error) {
            setVsConfigError(toErrorInfo(error).message);
        }
    }, []);
    const persistSpeakerConfig = useCallback(async (patch) => {
        setSpeakerError(null);
        try {
            const updated = await updateSpeakerVerificationConfig(patch);
            setSpeakerStatus((current) => current ? { ...current, config: updated } : current);
            await loadSpeakerStatus();
        }
        catch (error) {
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
        }
        catch { /* already disconnected */ }
        for (const node of enrollmentNodesRef.current) {
            try {
                node.disconnect();
            }
            catch { /* already disconnected */ }
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
            if (ctx.state !== "running")
                ctx.resume().catch(() => { });
        }
        catch { /* AudioContext unavailable */ }
        try {
            const lease = await acquireMicStream("speaker-enrollment", { audio: true }, {
                onRelease: (reason) => {
                    if (!LIFECYCLE_CANCEL_REASONS.has(reason))
                        return;
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
                if (ctx.state !== "running")
                    await ctx.resume();
                const source = ctx.createMediaStreamSource(stream);
                const { analyser, nodes } = createAudioFilterChain(ctx, source);
                enrollmentSourceRef.current = source;
                enrollmentNodesRef.current = nodes;
                enrollmentAnalyserRef.current = analyser;
                const timeDomain = new Uint8Array(analyser.frequencyBinCount);
                enrollmentLevelTimerRef.current = setInterval(() => {
                    const a = enrollmentAnalyserRef.current;
                    if (!a)
                        return;
                    a.getByteTimeDomainData(timeDomain);
                    let sum = 0;
                    for (let i = 0; i < timeDomain.length; i++) {
                        const v = ((timeDomain[i] ?? 128) - 128) / 128;
                        sum += v * v;
                    }
                    const rms = Math.sqrt(sum / timeDomain.length);
                    setEnrollmentLevel(Math.min(1, rms * 4));
                }, 100);
            }
            catch { /* level meter is best-effort; recording proceeds without it */ }
            const recorder = new MediaRecorder(stream, {
                mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
                    ? "audio/webm;codecs=opus"
                    : "audio/webm",
                // Match the streaming bitrate so enrollment and verification embeddings
                // are extracted from audio with the same codec characteristics.
                audioBitsPerSecond: 48000,
            });
            enrollmentRecorderRef.current = recorder;
            recorder.ondataavailable = (event) => {
                if (event.data.size > 0)
                    enrollmentChunksRef.current.push(event.data);
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
                if (enrollmentCancelledRef.current) {
                    enrollmentCancelledRef.current = false;
                    setEnrollmentState("idle");
                    return;
                }
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
        }
        catch (error) {
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
        }
        catch (error) {
            setSpeakerError(toErrorInfo(error).message);
        }
    }, [loadSpeakerStatus]);
    const removeProfile = useCallback(async (profileId) => {
        setSpeakerError(null);
        try {
            const updated = await removeSpeakerVerificationProfile(profileId);
            setSpeakerStatus((current) => current ? {
                ...current,
                config: updated,
                profileConfigured: updated.profileIds.length > 0,
            } : current);
            await loadSpeakerStatus();
        }
        catch (error) {
            setSpeakerError(toErrorInfo(error).message);
        }
    }, [loadSpeakerStatus]);
    const deleteProfile = useCallback(async (profileId) => {
        setSpeakerError(null);
        try {
            const updated = await deleteSpeakerVerificationProfile(profileId);
            setSpeakerStatus((current) => current ? {
                ...current,
                config: updated,
                profileConfigured: updated.profileIds.length > 0,
            } : current);
            await loadSpeakerStatus();
        }
        catch (error) {
            setSpeakerError(toErrorInfo(error).message);
        }
    }, [loadSpeakerStatus]);
    return (_jsxs("div", { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.voiceInputSection.eyebrow), title: t(strings.settings.voiceInputSection.title), description: t(strings.settings.voiceInputSection.description) }), _jsxs(SettingsCard, { className: "space-y-4", children: [_jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.voiceInputLabel), hint: t(strings.settings.voiceInputSection.voiceInputHint), control: (_jsx(SettingsToggle, { testId: "voice-enabled-toggle", checked: voiceEnabled, onClick: () => setVoiceEnabled(!voiceEnabled) })) }), voiceEnabled && (_jsxs(_Fragment, { children: [_jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.autoStopLabel), hint: t(strings.settings.voiceInputSection.autoStopHint), control: (_jsx(SettingsToggle, { testId: "vad-auto-stop-toggle", checked: vadAutoStop, onClick: () => setVadAutoStop(!vadAutoStop) })) }), vadAutoStop && (_jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.silenceTimeoutLabel), hint: t(strings.settings.voiceInputSection.silenceTimeoutHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "vad-silence-timeout-slider", type: "range", min: 200, max: 3000, step: 100, value: vadSilenceTimeoutMs, 
                                            // Write-through: audio-tools' stt_stream_config.vad_silence_ms
                                            // is the single source of truth. Range matches the
                                            // server-side validation [200, 3000] in stream_config.go.
                                            onChange: (event) => handleVsConfigChange({ vadSilenceMs: Number(event.target.value) }), disabled: !vsConfig || vsConfigLoading, className: "w-24 accent-wc-accent disabled:opacity-50" }), _jsx("span", { className: "w-9 text-end text-xs text-wc-text-muted", children: t(strings.settings.voiceInputSection.secondsShort, { value: (vadSilenceTimeoutMs / 1000).toFixed(1) }) })] })) })), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.languageLabel), hint: t(strings.settings.voiceInputSection.languageHint), control: (_jsxs("select", { "data-testid": "voice-language-select", className: "rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", value: voiceLanguage, onChange: (event) => setVoiceLanguage(event.target.value), children: [_jsx("option", { value: "auto", children: t(strings.settings.voiceInputSection.languageAuto) }), _jsx("option", { value: "en-US", children: t(strings.settings.voiceInputSection.langEnUs) }), _jsx("option", { value: "en-GB", children: t(strings.settings.voiceInputSection.langEnGb) }), _jsx("option", { value: "es-ES", children: t(strings.settings.voiceInputSection.langEsEs) }), _jsx("option", { value: "fr-FR", children: t(strings.settings.voiceInputSection.langFrFr) }), _jsx("option", { value: "de-DE", children: t(strings.settings.voiceInputSection.langDeDe) }), _jsx("option", { value: "zh-CN", children: t(strings.settings.voiceInputSection.langZhCn) }), _jsx("option", { value: "ja-JP", children: t(strings.settings.voiceInputSection.langJaJp) }), _jsx("option", { value: "ko-KR", children: t(strings.settings.voiceInputSection.langKoKr) }), _jsx("option", { value: "pt-BR", children: t(strings.settings.voiceInputSection.langPtBr) }), _jsx("option", { value: "hi-IN", children: t(strings.settings.voiceInputSection.langHiIn) })] })) })] }))] }), voiceEnabled && (_jsxs(SettingsCard, { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.voiceInputSection.persistentEyebrow), title: t(strings.settings.voiceInputSection.persistentTitle), description: t(strings.settings.voiceInputSection.persistentDescription) }), vsConfig && (_jsxs(_Fragment, { children: [_jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.persistentModeLabel), hint: t(strings.settings.voiceInputSection.persistentModeHint), control: (_jsx(SettingsToggle, { testId: "persistent-mode-toggle", checked: vsConfig.persistentMode, onClick: () => handleVsConfigChange({ persistentMode: !vsConfig.persistentMode }) })) }), vsConfig.persistentMode && (_jsxs(_Fragment, { children: [_jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.segmentSilenceLabel), hint: t(strings.settings.voiceInputSection.segmentSilenceHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "segment-silence-slider", type: "range", min: 200, max: 3000, step: 100, 
                                                    // Reads from vad_silence_ms (with legacy fall-back),
                                                    // writes to vad_silence_ms. Both this slider and
                                                    // the silence-timeout slider above map to the same
                                                    // server field — they are two UI views on one knob.
                                                    value: vsConfig.vadSilenceMs > 0 ? vsConfig.vadSilenceMs : vsConfig.segmentSilenceMs, onChange: (event) => handleVsConfigChange({ vadSilenceMs: Number(event.target.value) }), disabled: vsConfigLoading, className: "w-24 accent-wc-accent disabled:opacity-50" }), _jsx("span", { className: "w-9 text-end text-xs text-wc-text-muted", children: t(strings.settings.voiceInputSection.secondsShort, { value: ((vsConfig.vadSilenceMs > 0 ? vsConfig.vadSilenceMs : vsConfig.segmentSilenceMs) / 1000).toFixed(1) }) })] })) }), _jsxs("div", { className: "space-y-1.5", children: [_jsx("div", { className: "text-xs font-medium text-wc-text-secondary", children: t(strings.settings.voiceInputSection.voiceCommandsTitle) }), _jsx("div", { className: "text-[11px] text-wc-text-muted mb-1", children: t(strings.settings.voiceInputSection.voiceCommandsHint) }), _jsx("div", { className: "grid grid-cols-2 gap-1", children: VOICE_COMMANDS.map((cmd) => (_jsxs("div", { className: "text-[10px] text-wc-text-faint", children: [_jsx("span", { className: "text-wc-text-muted", children: cmd.description }), " — ", cmd.patterns[0]] }, cmd.id))) })] })] }))] })), !vsConfig && !vsConfigLoading && (_jsx("div", { className: "text-xs text-wc-text-faint", children: t(strings.settings.voiceInputSection.voiceConfigUnavailable) }))] })), voiceEnabled && vsConfig && (_jsxs(SettingsCard, { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.voiceInputSection.wakeWordEyebrow), title: t(strings.settings.voiceInputSection.wakeWordTitle), description: t(strings.settings.voiceInputSection.wakeWordSectionDescription) }), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.wakeWordLabel), hint: t(strings.settings.voiceInputSection.wakeWordHint), control: (_jsx(SettingsToggle, { testId: "wake-word-toggle", checked: vsConfig.wakeWordEnabled, onClick: () => {
                                if (!vsConfig.wakeWordEnabled && !wakeWordConfig?.configured) {
                                    setWakeWordError(t(strings.settings.voiceInputSection.wakeWordRequireSaveFirst));
                                    return;
                                }
                                handleVsConfigChange({ wakeWordEnabled: !vsConfig.wakeWordEnabled });
                            } })) }), _jsxs("div", { className: "rounded-xl border border-wc-default bg-wc-surface-base/60 p-3 space-y-3", children: [_jsxs("div", { className: "flex items-center gap-2 text-sm font-medium text-wc-text-secondary", children: [_jsx(Mic, { className: "h-4 w-4" }), t(strings.settings.voiceInputSection.wakeWordRecordingTitle)] }), _jsx("p", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.voiceInputSection.wakeWordRecordingHelp, { min: MIN_ENROLLMENT_SAMPLES, max: MAX_ENROLLMENT_SAMPLES }) }), _jsxs("div", { className: "flex items-center gap-2", children: [_jsx("span", { className: "text-[11px] text-wc-text-muted shrink-0", children: t(strings.settings.voiceInputSection.labelColon) }), _jsx("input", { "data-testid": "wake-word-label", type: "text", value: wwLabel, onChange: (e) => setWwLabel(e.target.value), className: "w-36 rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", placeholder: t(strings.settings.voiceInputSection.wakeWordLabelPlaceholder) })] }), _jsx("div", { className: "space-y-1.5", children: Array.from({ length: MAX_ENROLLMENT_SAMPLES }, (_, i) => {
                                    const sample = wwSamplesRef.current[i] ?? null;
                                    const hasSample = sample != null;
                                    const hasBlob = wwAudioBlobsRef.current[i] != null;
                                    const isRecording = wwRecordingIdx === i;
                                    const isPlaying = wwPlayingIdx === i;
                                    return (_jsxs("div", { className: "flex items-center gap-2 h-8", children: [_jsxs("span", { className: "w-16 text-[11px] text-wc-text-muted", children: [t(strings.settings.voiceInputSection.sample, { n: i + 1 }), i < MIN_ENROLLMENT_SAMPLES ? "" : t(strings.settings.voiceInputSection.sampleOptionalSuffix)] }), hasSample && sample ? (_jsxs(_Fragment, { children: [_jsx(CheckCircle, { className: "h-3.5 w-3.5 text-green-400 shrink-0" }), _jsx("span", { className: "text-[10px] text-green-400", children: t(strings.settings.voiceInputSection.secondsShort, { value: sample.durationSec.toFixed(1) }) }), hasBlob && (_jsx(Button, { variant: "ghost", size: "icon", className: "h-6 w-6", title: t(strings.settings.voiceInputSection.playSample), onClick: () => isPlaying ? undefined : playWwSample(i), disabled: isPlaying, children: _jsx(Play, { className: "h-3 w-3 text-wc-text-faint" }) })), _jsx(Button, { variant: "ghost", size: "icon", className: "h-6 w-6", title: t(strings.settings.voiceInputSection.reRecord), onClick: () => { removeWwSample(i); void startWwRecording(i); }, disabled: wwRecordingIdx !== null, children: _jsx(RotateCcw, { className: "h-3 w-3 text-wc-text-faint" }) }), _jsx(Button, { variant: "ghost", size: "icon", className: "h-6 w-6", title: t(strings.settings.voiceInputSection.remove), onClick: () => removeWwSample(i), children: _jsx(Trash2, { className: "h-3 w-3 text-wc-text-faint" }) })] })) : isRecording ? (_jsxs(Button, { variant: "outline", size: "sm", className: "h-6 px-2 text-[10px]", onClick: stopWwRecording, children: [_jsx(Square, { className: "me-1 h-3 w-3" }), t(strings.settings.voiceInputSection.stopSeconds, { seconds: wwRecordingSeconds })] })) : (_jsxs(Button, { variant: "outline", size: "sm", className: "h-6 px-2 text-[10px]", onClick: () => void startWwRecording(i), disabled: wwRecordingIdx !== null, children: [_jsx(Mic, { className: "me-1 h-3 w-3" }), t(strings.settings.voiceInputSection.record)] }))] }, i));
                                }) }), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.sensitivityLabel), hint: t(strings.settings.voiceInputSection.sensitivityHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "wake-word-threshold-slider", type: "range", min: 0.1, max: 0.95, step: 0.05, value: useWorkspaceStore.getState().wakeWordThreshold, onChange: (e) => {
                                                const val = Number(e.target.value);
                                                useWorkspaceStore.getState().setWakeWordThreshold(val);
                                                handleVsConfigChange({ wakeWordThreshold: val });
                                            }, className: "w-24 accent-wc-accent" }), _jsx("span", { className: "w-10 text-end text-xs text-wc-text-muted", children: useWorkspaceStore.getState().wakeWordThreshold.toFixed(2) })] })) }), wakeWordError && (_jsxs("div", { className: "flex items-center gap-2 text-xs text-wc-error-detail", children: [_jsx(AlertCircle, { className: "h-3.5 w-3.5 shrink-0" }), wakeWordError] })), wwTestResult && (_jsx("div", { className: "text-xs text-wc-text-muted", children: wwTestResult })), _jsxs("div", { className: "flex flex-wrap items-center gap-2", children: [_jsx(Button, { "data-testid": "wake-word-save", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", onClick: () => void saveWakeWord(), disabled: wwSamplesRef.current.filter((s) => s != null).length < MIN_ENROLLMENT_SAMPLES || wwRecordingIdx !== null, children: t(strings.settings.voiceInputSection.saveWakeWord) }), _jsx(Button, { "data-testid": "wake-word-test", variant: "ghost", size: "sm", className: "h-8 px-3 text-xs text-wc-text-faint", onClick: () => void testWakeWord(), disabled: wwSamplesRef.current.filter((s) => s != null).length < 2, children: t(strings.settings.voiceInputSection.testCrossMatch) }), wakeWordConfig?.configured && (_jsx(Button, { "data-testid": "wake-word-delete", variant: "ghost", size: "sm", className: "h-8 px-3 text-xs text-wc-text-faint", onClick: () => void deleteWakeWord(), children: t(strings.settings.voiceInputSection.deleteWakeWord) }))] })] }), _jsxs("div", { className: "rounded-xl border border-wc-default bg-wc-surface-base/60 p-3 space-y-3", children: [_jsxs("div", { className: "flex items-center gap-2 text-sm font-medium text-wc-text-secondary", children: [_jsx(Play, { className: "h-4 w-4" }), t(strings.settings.voiceInputSection.liveTestTitle)] }), _jsx("p", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.voiceInputSection.liveTestHelp) }), wwTestSamples.length === 0 ? (_jsx("p", { className: "text-[11px] text-wc-text-faint italic", children: t(strings.settings.voiceInputSection.liveTestEmpty) })) : (_jsxs(_Fragment, { children: [_jsxs("div", { className: "flex items-center gap-3", children: [_jsxs(Button, { "data-testid": "wake-word-live-test", variant: "outline", size: "sm", className: `h-10 px-4 text-xs select-none ${wakeWordTest.state.status === "recording" ? "border-wc-accent text-wc-accent animate-pulse" : ""}`, disabled: wwRecordingIdx !== null || wakeWordTest.state.status === "comparing", onMouseDown: () => wakeWordTest.startRecording(), onMouseUp: () => wakeWordTest.stopRecording(), onMouseLeave: () => { if (wakeWordTest.state.status === "recording")
                                                    wakeWordTest.stopRecording(); }, onTouchStart: (e) => { e.preventDefault(); wakeWordTest.startRecording(); }, onTouchEnd: (e) => { e.preventDefault(); wakeWordTest.stopRecording(); }, children: [_jsx(Mic, { className: "me-1.5 h-3.5 w-3.5" }), wakeWordTest.state.status === "idle" || wakeWordTest.state.status === "result"
                                                        ? t(strings.settings.voiceInputSection.holdToTest)
                                                        : wakeWordTest.state.status === "recording"
                                                            ? t(strings.settings.voiceInputSection.recordingSecondsLabel, { seconds: wakeWordTest.state.recordingSeconds })
                                                            : t(strings.settings.voiceInputSection.comparing)] }), wakeWordTest.state.history.length > 0 && (_jsx("button", { "data-testid": "wake-word-clear-history", className: "text-[10px] text-wc-text-faint hover:text-wc-text-muted underline", onClick: () => wakeWordTest.clearHistory(), children: t(strings.settings.voiceInputSection.clearHistory) }))] }), wakeWordTest.state.error && (_jsxs("div", { className: "flex items-center gap-2 text-xs text-wc-error-detail", children: [_jsx(AlertCircle, { className: "h-3.5 w-3.5 shrink-0" }), wakeWordTest.state.error] })), wakeWordTest.state.currentResult && (_jsxs("div", { className: "space-y-1", children: [_jsxs("div", { className: "flex items-center gap-2 text-xs", children: [_jsx("span", { className: wakeWordTest.state.currentResult.isMatch ? "text-green-500 font-medium" : "text-red-500 font-medium", children: wakeWordTest.state.currentResult.isMatch ? t(strings.settings.voiceInputSection.liveTestMatch) : t(strings.settings.voiceInputSection.liveTestReject) }), _jsx("span", { className: "text-wc-text-muted", children: t(strings.settings.voiceInputSection.liveTestScore, { score: wakeWordTest.state.currentResult.score.toFixed(3) }) })] }), _jsxs("div", { className: "relative h-3 w-full rounded-full bg-wc-surface-overlay overflow-hidden", children: [_jsx("div", { className: `h-full rounded-full transition-all ${wakeWordTest.state.currentResult.isMatch ? "bg-green-500/70" : "bg-red-500/70"}`, style: { width: `${Math.min(wakeWordTest.state.currentResult.score * 100, 100)}%` } }), _jsx("div", { className: "absolute top-0 h-full w-0.5 bg-wc-text-secondary", style: { left: `${useWorkspaceStore.getState().wakeWordThreshold * 100}%` }, title: t(strings.settings.voiceInputSection.thresholdTooltip, { threshold: useWorkspaceStore.getState().wakeWordThreshold.toFixed(2) }) })] })] })), wakeWordTest.state.history.length > 1 && (_jsxs("div", { className: "space-y-1", children: [_jsx("div", { className: "text-[10px] text-wc-text-faint font-medium", children: t(strings.settings.voiceInputSection.recentAttempts) }), _jsx("div", { className: "max-h-32 overflow-y-auto space-y-1", children: wakeWordTest.state.history.map((attempt, i) => (_jsxs("div", { className: `flex items-center gap-2 text-[10px] ${i === 0 ? "opacity-100" : "opacity-70"}`, children: [_jsx("span", { className: `w-9 font-medium ${attempt.isMatch ? "text-green-500" : "text-red-500"}`, children: attempt.isMatch ? t(strings.settings.voiceInputSection.attemptPass) : t(strings.settings.voiceInputSection.attemptFail) }), _jsxs("div", { className: "relative h-1.5 flex-1 rounded-full bg-wc-surface-overlay overflow-hidden", children: [_jsx("div", { className: `h-full rounded-full ${attempt.isMatch ? "bg-green-500/60" : "bg-red-500/60"}`, style: { width: `${Math.min(attempt.score * 100, 100)}%` } }), _jsx("div", { className: "absolute top-0 h-full w-px bg-wc-text-faint", style: { left: `${useWorkspaceStore.getState().wakeWordThreshold * 100}%` } })] }), _jsx("span", { className: "w-10 text-end text-wc-text-faint", children: attempt.score.toFixed(2) })] }, attempt.timestamp))) })] }))] }))] })] })), voiceEnabled && (_jsxs(SettingsCard, { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.voiceInputSection.speakerEyebrow), title: t(strings.settings.voiceInputSection.speakerTitle), description: t(strings.settings.voiceInputSection.speakerDescription) }), speakerError && (_jsxs("div", { className: "flex items-center gap-2 text-xs text-wc-error-detail", children: [_jsx(AlertCircle, { className: "h-3.5 w-3.5 shrink-0" }), speakerError] })), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.resourceStatusLabel), hint: speakerLoading
                            ? t(strings.settings.voiceInputSection.resourceStatusChecking)
                            : speakerStatus?.capability === "available"
                                ? (speakerStatus.resourceReady ? t(strings.settings.voiceInputSection.resourceReady) : t(strings.settings.voiceInputSection.resourceReachable))
                                : t(strings.settings.voiceInputSection.resourceUnavailable), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("span", { className: `text-xs font-medium ${speakerStatus?.capability === "available" && speakerStatus?.resourceReady
                                        ? "text-green-400"
                                        : "text-wc-text-faint"}`, children: speakerStatus?.capability === "available" && speakerStatus?.resourceReady ? t(strings.settings.voiceInputSection.ready) : t(strings.settings.voiceInputSection.unavailable) }), _jsx(Button, { "data-testid": "speaker-refresh", variant: "ghost", size: "icon", className: "h-8 w-8", onClick: () => void loadSpeakerStatus(), title: t(strings.settings.voiceInputSection.refreshSpeakerTitle), children: _jsx(RefreshCw, { className: `h-3.5 w-3.5 text-wc-text-faint ${speakerLoading ? "animate-spin" : ""}` }) })] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.useSpeakerLabel), hint: t(strings.settings.voiceInputSection.useSpeakerHint), control: (_jsx(SettingsToggle, { testId: "speaker-verification-toggle", checked: speakerStatus?.config.enabled ?? false, onClick: () => {
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
                            } })) }), _jsxs("div", { className: "space-y-1.5", children: [_jsx("div", { className: "text-xs font-medium text-wc-text-secondary", children: t(strings.settings.voiceInputSection.activeProfilesTitle) }), _jsx("div", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.voiceInputSection.activeProfilesHint) }), (speakerStatus?.config.profileIds ?? []).length > 0 ? (_jsx("div", { className: "flex flex-wrap gap-1.5", "data-testid": "speaker-active-profiles", children: (speakerStatus?.config.profileIds ?? []).map((id) => {
                                    const profile = speakerStatus?.profiles?.find((p) => p.id === id);
                                    return (_jsxs("span", { className: "inline-flex items-center gap-1 rounded-full border border-wc-default bg-wc-surface-base px-2.5 py-0.5 text-[11px] text-wc-text-primary", children: [profile?.display_name ?? id, _jsx("button", { className: "ms-0.5 text-wc-text-faint hover:text-wc-text-primary", title: t(strings.settings.voiceInputSection.removeProfileTitle, { name: profile?.display_name ?? id }), onClick: () => void removeProfile(id), children: "\u00D7" })] }, id));
                                }) })) : (_jsx("div", { className: "text-[11px] text-wc-text-faint", children: t(strings.settings.voiceInputSection.noActiveProfiles) }))] }), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.modeLabel), hint: t(strings.settings.voiceInputSection.modeHint), control: (_jsxs("select", { "data-testid": "speaker-mode-select", className: "rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", value: speakerStatus?.config.mode ?? "filter", onChange: (event) => {
                                void persistSpeakerConfig({ mode: event.target.value });
                            }, children: [_jsx("option", { value: "filter", children: t(strings.settings.voiceInputSection.modeFilter) }), _jsx("option", { value: "advisory", children: t(strings.settings.voiceInputSection.modeAdvisory) }), _jsx("option", { value: "off", children: t(strings.settings.voiceInputSection.modeOff) })] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.thresholdLabel), hint: t(strings.settings.voiceInputSection.thresholdHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "speaker-threshold-slider", type: "range", min: 0.1, max: 0.99, step: 0.01, value: speakerStatus?.config.threshold ?? 0.35, onChange: (event) => {
                                        void persistSpeakerConfig({ threshold: Number(event.target.value) });
                                    }, className: "w-24 accent-wc-accent" }), _jsx("span", { className: "w-10 text-end text-xs text-wc-text-muted", children: (speakerStatus?.config.threshold ?? 0.35).toFixed(2) })] })) }), _jsxs("div", { className: "rounded-xl border border-wc-default bg-wc-surface-base/60 p-3", children: [_jsxs("div", { className: "flex items-center gap-2 text-sm font-medium text-wc-text-secondary", children: [_jsx(UserRound, { className: "h-4 w-4" }), t(strings.settings.voiceInputSection.enrollment)] }), _jsx("p", { className: "mt-1 text-[11px] text-wc-text-muted", children: t(strings.settings.voiceInputSection.enrollmentHelp) }), _jsx("p", { className: "mt-1.5 rounded-lg bg-wc-surface-base/80 px-2.5 py-1.5 text-xs italic text-wc-text-primary", children: t(strings.settings.voiceInputSection.enrollmentScript) }), _jsxs("div", { className: "mt-3 flex flex-wrap items-center gap-2", children: [_jsx("input", { "data-testid": "speaker-display-name", type: "text", value: profileDisplayName, onChange: (event) => setProfileDisplayName(event.target.value), className: "w-36 rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", placeholder: t(strings.settings.voiceInputSection.profileNamePlaceholder) }), enrollmentState === "recording" ? (_jsxs(_Fragment, { children: [_jsxs(Button, { "data-testid": "speaker-enrollment-stop", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", onClick: stopEnrollmentRecording, children: [_jsx(Square, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.voiceInputSection.stopSeconds, { seconds: enrollmentSeconds })] }), _jsx("div", { "data-testid": "speaker-enrollment-level", className: "h-2 w-24 overflow-hidden rounded-full bg-wc-surface-base", role: "meter", "aria-label": t(strings.settings.voiceInputSection.addVoiceProfile), "aria-valuenow": Math.round(enrollmentLevel * 100), children: _jsx("div", { className: "h-full rounded-full bg-wc-accent transition-[width] duration-100", style: { width: `${Math.min(enrollmentLevel * 100, 100)}%` } }) })] })) : (_jsxs(Button, { "data-testid": "speaker-enrollment-start", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", onClick: () => {
                                            setReEnrollTargetId(null);
                                            void startEnrollmentRecording();
                                        }, disabled: speakerStatus?.capability !== "available" || !speakerStatus?.resourceReady || enrollmentState === "uploading", children: [_jsx(Mic, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.voiceInputSection.addVoiceProfile)] })), _jsx(Button, { "data-testid": "speaker-clear-profile", variant: "ghost", size: "sm", className: "h-8 px-3 text-xs text-wc-text-faint", onClick: () => void clearSpeakerBinding(), disabled: !(speakerStatus?.config.profileIds?.length), children: t(strings.settings.voiceInputSection.clearAllProfiles) })] }), enrollmentMessage && (_jsx("div", { className: `mt-2 text-xs ${enrollmentState === "error" ? "text-wc-error-detail" : "text-wc-text-muted"}`, children: enrollmentMessage })), speakerStatus?.profiles?.length ? (_jsxs("div", { className: "mt-3 space-y-1.5", children: [_jsx("div", { className: "text-[11px] font-medium text-wc-text-secondary", children: t(strings.settings.voiceInputSection.enrolledProfilesTitle) }), speakerStatus.profiles.map((profile) => {
                                        const isActive = speakerStatus.config.profileIds?.includes(profile.id);
                                        return (_jsxs("div", { className: "flex items-center justify-between rounded-lg border border-wc-default bg-wc-surface-base/40 px-2.5 py-1.5", children: [_jsxs("div", { className: "min-w-0", children: [_jsxs("div", { className: "flex items-center gap-1.5 text-xs text-wc-text-primary", children: [profile.display_name, isActive && (_jsx("span", { className: "rounded bg-green-400/15 px-1.5 py-0 text-[10px] text-green-400", children: t(strings.settings.voiceInputSection.profileActive) }))] }), _jsxs("div", { className: "text-[10px] text-wc-text-faint", children: [t(strings.settings.voiceInputSection.enrollmentSecondsLabel, { seconds: profile.enrollment_audio_seconds.toFixed(1) }), profile.notes ? ` · ${profile.notes}` : ""] })] }), _jsxs("div", { className: "flex items-center gap-1 shrink-0 ms-2", children: [!isActive && (_jsx(Button, { variant: "ghost", size: "icon", className: "h-6 w-6", title: t(strings.settings.voiceInputSection.addToActiveTitle), onClick: () => {
                                                                const ids = [...(speakerStatus.config.profileIds ?? []), profile.id];
                                                                void persistSpeakerConfig({ profileIds: ids, enabled: true });
                                                            }, children: _jsx(CheckCircle, { className: "h-3 w-3 text-wc-text-faint" }) })), _jsx(Button, { variant: "ghost", size: "icon", className: "h-6 w-6", title: t(strings.settings.voiceInputSection.reEnrollTitle, { name: profile.display_name }), onClick: () => {
                                                                setProfileDisplayName(profile.display_name);
                                                                setReEnrollTargetId(profile.id);
                                                                void startEnrollmentRecording();
                                                            }, disabled: speakerStatus?.capability !== "available" || !speakerStatus?.resourceReady || enrollmentState === "recording" || enrollmentState === "uploading", children: _jsx(Mic, { className: "h-3 w-3 text-wc-text-faint" }) }), _jsx(Button, { variant: "ghost", size: "icon", className: "h-6 w-6", title: t(strings.settings.voiceInputSection.deleteProfileTitle, { name: profile.display_name }), onClick: () => void deleteProfile(profile.id), children: _jsx(Trash2, { className: "h-3 w-3 text-wc-text-faint hover:text-red-400" }) })] })] }, profile.id));
                                    })] })) : null] })] })), _jsx(SettingsCard, { className: "space-y-4", children: _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.shortcutLabel), hint: t(strings.settings.voiceInputSection.shortcutHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [recordingShortcut ? (_jsx("span", { "data-testid": "voice-shortcut-recording", className: "rounded-lg border border-wc-accent bg-wc-surface-base px-2 py-1 font-mono text-xs text-wc-accent animate-pulse", tabIndex: 0, onKeyDown: (event) => {
                                    if (["Control", "Alt", "Shift", "Meta"].includes(event.key))
                                        return;
                                    event.preventDefault();
                                    const shortcut = formatShortcutFromEvent(event.nativeEvent);
                                    setVoiceShortcut(shortcut);
                                    setRecordingShortcut(false);
                                }, onBlur: () => setRecordingShortcut(false), ref: (element) => element?.focus(), children: t(strings.settings.voiceInputSection.pressKeyCombo) })) : (_jsx("span", { "data-testid": "voice-shortcut-display", className: "rounded-lg bg-wc-surface-base px-2 py-1 font-mono text-xs text-wc-text-muted", children: voiceShortcut })), _jsx(Button, { "data-testid": "voice-shortcut-change", variant: "ghost", size: "icon", className: "h-8 w-8", onClick: () => setRecordingShortcut(true), title: t(strings.settings.voiceInputSection.changeShortcut), children: _jsx(Keyboard, { className: "h-3.5 w-3.5 text-wc-text-faint" }) })] })) }) }), _jsxs(SettingsCard, { className: "space-y-3", children: [_jsxs("div", { className: "flex items-center justify-between", children: [_jsxs("div", { children: [_jsx("div", { className: "text-sm font-medium text-wc-text-secondary", children: t(strings.settings.voiceInputSection.backendAvailability) }), _jsx("div", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.voiceInputSection.backendAvailabilityHint) })] }), _jsx(Button, { "data-testid": "voice-caps-refresh", variant: "ghost", size: "icon", className: "h-8 w-8", onClick: () => void loadVoiceCaps(), title: t(strings.settings.voiceInputSection.refreshTitle), children: _jsx(RefreshCw, { className: `h-3.5 w-3.5 text-wc-text-faint ${voiceCapsLoading ? "animate-spin" : ""}` }) })] }), voiceCapsError && (_jsxs("div", { className: "flex items-center gap-2 text-xs text-wc-error-detail", children: [_jsx(AlertCircle, { className: "h-3.5 w-3.5 shrink-0" }), voiceCapsError] })), _jsxs("div", { className: "space-y-2", children: [voiceCaps.filter((capability) => capability.features.includes("voice-input")).map((capability) => (_jsxs("div", { className: "flex items-center gap-2 text-xs", children: [capability.status === "available" ? (_jsx(CheckCircle, { className: "h-3.5 w-3.5 shrink-0 text-green-400" })) : (_jsx(Circle, { className: "h-3.5 w-3.5 shrink-0 text-wc-text-faint" })), _jsx("span", { className: capability.status === "available" ? "text-wc-text-secondary" : "text-wc-text-faint", children: capability.name }), _jsx("span", { className: capability.status === "available" ? "ml-auto text-green-400" : "ml-auto text-wc-text-faint", children: capability.status })] }, capability.id))), _jsxs("div", { className: "flex items-center gap-2 text-xs", children: [hasWebSpeech ? (_jsx(CheckCircle, { className: "h-3.5 w-3.5 shrink-0 text-green-400" })) : (_jsx(Circle, { className: "h-3.5 w-3.5 shrink-0 text-wc-text-faint" })), _jsx("span", { className: hasWebSpeech ? "text-wc-text-secondary" : "text-wc-text-faint", children: t(strings.settings.voiceInputSection.webSpeechApi) }), _jsx("span", { className: hasWebSpeech ? "ml-auto text-green-400" : "ml-auto text-wc-text-faint", children: hasWebSpeech ? t(strings.settings.voiceInputSection.available) : t(strings.settings.voiceInputSection.unavailable) })] })] }), !voiceCapsLoading && voiceCaps.every((capability) => capability.status !== "available") && !hasWebSpeech && (_jsx("p", { className: "text-[11px] text-amber-400", children: t(strings.settings.voiceInputSection.noBackendWarning) }))] }), _jsxs(SettingsCard, { className: "space-y-3", children: [_jsxs("div", { children: [_jsx("div", { className: "text-sm font-medium text-wc-text-secondary", children: t(strings.settings.voiceInputSection.micAccess) }), _jsx("div", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.voiceInputSection.micAccessHint) })] }), _jsx("div", { className: "flex items-center gap-2 text-xs", children: micPermission === "granted" ? (_jsxs(_Fragment, { children: [_jsx(CheckCircle, { className: "h-3.5 w-3.5 shrink-0 text-green-400" }), _jsx("span", { className: "text-wc-text-secondary", children: t(strings.settings.voiceInputSection.permissionGranted) })] })) : micPermission === "denied" ? (_jsxs(_Fragment, { children: [_jsx(AlertCircle, { className: "h-3.5 w-3.5 shrink-0 text-red-400" }), _jsx("span", { className: "text-red-400", children: t(strings.settings.voiceInputSection.permissionDenied) })] })) : micPermission === "prompt" ? (_jsxs(_Fragment, { children: [_jsx(Circle, { className: "h-3.5 w-3.5 shrink-0 text-wc-text-faint" }), _jsx("span", { className: "text-wc-text-faint", children: t(strings.settings.voiceInputSection.permissionNotRequested) })] })) : (_jsxs(_Fragment, { children: [_jsx(Circle, { className: "h-3.5 w-3.5 shrink-0 text-wc-text-faint" }), _jsx("span", { className: "text-wc-text-faint", children: t(strings.settings.voiceInputSection.permissionUnknown) })] })) }), micPermission === "denied" && (_jsx("p", { className: "text-[11px] text-wc-text-faint", children: t(strings.settings.voiceInputSection.micBlockedHelp) })), (micPermission === "prompt" || micPermission === "unknown") && (_jsxs(Button, { "data-testid": "mic-request-permission", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", onClick: () => void requestMicPermission(), disabled: micRequesting, children: [_jsx(Mic, { className: "me-1 h-3.5 w-3.5" }), micRequesting ? t(strings.settings.voiceInputSection.requesting) : t(strings.settings.voiceInputSection.allowMicrophone)] }))] }), voiceEnabled && _jsx(TestMicrophoneCard, {}), voiceEnabled && (_jsxs(SettingsCard, { className: "space-y-3", children: [_jsxs("button", { "data-testid": "advanced-streaming-toggle", className: "flex w-full items-center gap-1 text-start text-[11px] font-semibold uppercase tracking-[0.22em] text-wc-text-muted", onClick: () => setAdvancedOpen(!advancedOpen), children: [advancedOpen ? _jsx(ChevronDown, { className: "h-3.5 w-3.5" }) : _jsx(ChevronRight, { className: "h-3.5 w-3.5" }), t(strings.settings.voiceInputSection.advancedStreaming)] }), advancedOpen && (_jsxs("div", { className: "space-y-4", children: [vsConfigLoading && (_jsx("div", { className: "py-2 text-center text-xs text-wc-text-faint", children: t(strings.settings.voiceInputSection.loading) })), vsConfigError && (_jsxs("div", { className: "flex items-center gap-2 text-xs text-wc-error-detail", children: [_jsx(AlertCircle, { className: "h-3.5 w-3.5 shrink-0" }), vsConfigError] })), vsConfig && (_jsxs(_Fragment, { children: [_jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.flushIntervalLabel), hint: t(strings.settings.voiceInputSection.flushIntervalHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "vs-flush-interval", type: "range", min: 100, max: 5000, step: 50, value: vsConfig.flushIntervalMs, onChange: (event) => handleVsConfigChange({ flushIntervalMs: Number(event.target.value) }), className: "w-24 accent-wc-accent" }), _jsx("span", { className: "w-12 text-end text-xs text-wc-text-muted", children: t(strings.settings.voiceInputSection.msSuffix, { value: vsConfig.flushIntervalMs }) })] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.minChunkLabel), hint: t(strings.settings.voiceInputSection.minChunkHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "vs-min-delta", type: "range", min: 512, max: 32768, step: 512, value: vsConfig.minDeltaBytes, onChange: (event) => handleVsConfigChange({ minDeltaBytes: Number(event.target.value) }), className: "w-24 accent-wc-accent" }), _jsx("span", { className: "w-12 text-end text-xs text-wc-text-muted", children: t(strings.settings.voiceInputSection.kbSuffix, { value: (vsConfig.minDeltaBytes / 1024).toFixed(1) }) })] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceInputSection.overlapLabel), hint: t(strings.settings.voiceInputSection.overlapHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "vs-overlap", type: "range", min: 0, max: 16384, step: 256, value: vsConfig.overlapBytes, onChange: (event) => handleVsConfigChange({ overlapBytes: Number(event.target.value) }), className: "w-24 accent-wc-accent" }), _jsx("span", { className: "w-12 text-end text-xs text-wc-text-muted", children: t(strings.settings.voiceInputSection.kbSuffix, { value: (vsConfig.overlapBytes / 1024).toFixed(1) }) })] })) }), _jsxs(Button, { "data-testid": "vs-reset-defaults", variant: "ghost", size: "sm", className: "h-8 px-3 text-xs text-wc-text-faint", onClick: () => void resetVsConfig(), children: [_jsx(RotateCcw, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.voiceInputSection.resetDefaults)] })] }))] }))] }))] }));
}
function detectedBackendLabel(t, b) {
    switch (b) {
        case "loading": return t(strings.settings.voiceInputSection.testMicDetecting);
        case "whisper-stream": return t(strings.settings.voiceInputSection.testMicBackendWhisperStream);
        case "whisper-http": return t(strings.settings.voiceInputSection.testMicBackendWhisperHttp);
        case "web-speech": return t(strings.settings.voiceInputSection.testMicBackendWebSpeech);
        case "none": return t(strings.settings.voiceInputSection.testMicBackendNone);
    }
}
function TestMicrophoneCard() {
    const { t: tRaw } = useTranslation();
    const t = tRaw;
    const [detected, setDetected] = useState("loading");
    const [recording, setRecording] = useState(false);
    const [remaining, setRemaining] = useState(0);
    const [transcribing, setTranscribing] = useState(false);
    const [result, setResult] = useState(null);
    const [error, setError] = useState(null);
    const providerRef = useRef(null);
    const detect = useCallback(async () => {
        setDetected("loading");
        try {
            const probe = await probeWhisperHealth();
            if (probe.whisperHealthy) {
                setDetected(probe.streamingAvailable ? "whisper-stream" : "whisper-http");
                return;
            }
        }
        catch {
            // fall through to web-speech check
        }
        const Ctor = window.SpeechRecognition
            ?? window.webkitSpeechRecognition;
        setDetected(Ctor ? "web-speech" : "none");
    }, []);
    useEffect(() => { void detect(); }, [detect]);
    const runTest = useCallback(async () => {
        setError(null);
        setResult(null);
        const backend = detected;
        let provider;
        let providerUsed;
        if (backend === "whisper-stream") {
            provider = new PcmVoiceStreamProvider();
            providerUsed = t(strings.settings.voiceInputSection.testMicBackendWhisperStream);
        }
        else if (backend === "whisper-http") {
            provider = new WhisperProvider();
            providerUsed = t(strings.settings.voiceInputSection.testMicBackendWhisperHttp);
        }
        else if (backend === "web-speech") {
            provider = new WebSpeechProvider();
            providerUsed = t(strings.settings.voiceInputSection.testMicBackendWebSpeech);
        }
        else {
            setError(t(strings.settings.voiceInputSection.testMicBackendNone));
            return;
        }
        providerRef.current = provider;
        let finalText = "";
        provider.onResult = (text) => { finalText = text; };
        const startedAt = performance.now();
        try {
            setRecording(true);
            setRemaining(3);
            await provider.start();
            const tick = setInterval(() => setRemaining((r) => (r > 0 ? r - 1 : 0)), 1000);
            await new Promise((resolve) => setTimeout(resolve, 3000));
            clearInterval(tick);
            setRecording(false);
            setTranscribing(true);
            provider.stop();
            // Wait briefly for onResult to fire (Whisper HTTP path is async).
            await new Promise((resolve) => setTimeout(resolve, 1500));
            const elapsedMs = Math.round(performance.now() - startedAt);
            setResult({ providerUsed, transcript: finalText, elapsedMs });
        }
        catch (err) {
            setError(toErrorInfo(err).message);
            try {
                provider.stop();
            }
            catch { /* ignore */ }
        }
        finally {
            setRecording(false);
            setTranscribing(false);
            setRemaining(0);
            providerRef.current = null;
        }
    }, [detected, t]);
    const busy = recording || transcribing || detected === "loading";
    return (_jsxs(SettingsCard, { className: "space-y-3", children: [_jsxs("div", { className: "flex items-center justify-between", children: [_jsx("div", { className: "text-[11px] font-semibold uppercase tracking-[0.22em] text-wc-text-muted", children: t(strings.settings.voiceInputSection.testMicHeading) }), _jsx(Button, { "data-testid": "mic-test-refresh-detection", variant: "ghost", size: "sm", className: "h-7 px-2 text-xs", onClick: () => void detect(), disabled: busy, children: _jsx(RefreshCw, { className: "h-3.5 w-3.5" }) })] }), _jsx("p", { className: "text-xs text-wc-text-faint", children: t(strings.settings.voiceInputSection.testMicHint) }), _jsxs("div", { className: "flex items-center justify-between text-xs", children: [_jsx("span", { className: "text-wc-text-muted", children: t(strings.settings.voiceInputSection.testMicDetected) }), _jsx("span", { "data-testid": "mic-test-detected-backend", className: "font-medium text-wc-text-primary", children: detectedBackendLabel(t, detected) })] }), _jsxs(Button, { "data-testid": "mic-test-record", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", onClick: () => void runTest(), disabled: busy || detected === "none", children: [_jsx(Mic, { className: "me-1 h-3.5 w-3.5" }), recording
                        ? t(strings.settings.voiceInputSection.testMicRecording, { remaining })
                        : transcribing
                            ? t(strings.settings.voiceInputSection.testMicTranscribing)
                            : t(strings.settings.voiceInputSection.testMicRecord)] }), error && (_jsxs("div", { className: "flex items-start gap-2 text-xs text-wc-error-detail", children: [_jsx(AlertCircle, { className: "mt-0.5 h-3.5 w-3.5 shrink-0" }), _jsxs("span", { children: [t(strings.settings.voiceInputSection.testMicError), ": ", error] })] })), result && (_jsxs("div", { "data-testid": "mic-test-result", className: "space-y-1 rounded-md border border-wc-default bg-wc-surface-base p-2 text-xs", children: [_jsxs("div", { className: "flex justify-between", children: [_jsx("span", { className: "text-wc-text-muted", children: t(strings.settings.voiceInputSection.testMicProviderUsed) }), _jsx("span", { className: "font-medium text-wc-text-primary", children: result.providerUsed })] }), _jsxs("div", { className: "flex justify-between", children: [_jsx("span", { className: "text-wc-text-muted", children: t(strings.settings.voiceInputSection.testMicElapsed) }), _jsx("span", { className: "text-wc-text-primary", children: t(strings.settings.voiceInputSection.testMicMsSuffix, { value: result.elapsedMs }) })] }), _jsxs("div", { className: "space-y-0.5", children: [_jsx("div", { className: "text-wc-text-muted", children: t(strings.settings.voiceInputSection.testMicTranscript) }), _jsx("div", { className: "rounded bg-wc-surface-raised p-1.5 text-wc-text-primary", children: result.transcript || _jsx("span", { className: "text-wc-text-faint", children: t(strings.settings.voiceInputSection.testMicNoTranscript) }) })] })] }))] }));
}
