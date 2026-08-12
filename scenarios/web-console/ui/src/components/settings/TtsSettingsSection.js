import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { toErrorInfo } from "../../lib/errors";
import { getTTSConfig, updateTTSConfig, } from "../../audio-integration";
import { getTTSHookStatus, updateTTSHookConfig } from "../../api/ttsHook";
import { useTextToSpeech } from "../../hooks/useTextToSpeech";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";
import { useSummarizeSettings } from "./useSummarizeSettings";
// TtsSettingsSection split-of-concerns:
//   - voice / speed / response-format / summarization knobs → audio-integration
//     (the audio-tools scenario owns the canonical TTSConfig + summarize knobs).
//   - autoEnabled / backend preference / startMuted / Claude-hook routing
//     status → web-console-internal /api/v1/tts-hook/* (Claude-hook routing
//     is a web-console concern, not an audio-tools concern, as is the
//     workspace-store preference triple).
//
// audio-tools' TTSConfig exposes defaultVoice / defaultSpeed; the UI maps
// kokoroVoice <-> defaultVoice and kokoroSpeed <-> defaultSpeed for backward
// compatibility with the workspace-store fields and the existing test seams.
export default function TtsSettingsSection() {
    const { t } = useTranslation();
    const ttsVoice = useWorkspaceStore((state) => state.ttsVoice);
    const setTtsVoice = useWorkspaceStore((state) => state.setTtsVoice);
    const ttsRate = useWorkspaceStore((state) => state.ttsRate);
    const setTtsRate = useWorkspaceStore((state) => state.setTtsRate);
    const ttsPitch = useWorkspaceStore((state) => state.ttsPitch);
    const setTtsPitch = useWorkspaceStore((state) => state.setTtsPitch);
    const autoTtsEnabled = useWorkspaceStore((state) => state.autoTtsEnabled);
    const setAutoTtsEnabled = useWorkspaceStore((state) => state.setAutoTtsEnabled);
    const startMutedOnLoad = useWorkspaceStore((state) => state.startMutedOnLoad);
    const setStartMutedOnLoad = useWorkspaceStore((state) => state.setStartMutedOnLoad);
    const ttsBackendPreference = useWorkspaceStore((state) => state.ttsBackendPreference);
    const setTtsBackendPreference = useWorkspaceStore((state) => state.setTtsBackendPreference);
    const kokoroVoice = useWorkspaceStore((state) => state.kokoroVoice);
    const setKokoroVoice = useWorkspaceStore((state) => state.setKokoroVoice);
    const kokoroSpeed = useWorkspaceStore((state) => state.kokoroSpeed);
    const setKokoroSpeed = useWorkspaceStore((state) => state.setKokoroSpeed);
    const [statusLoading, setStatusLoading] = useState(true);
    const [statusError, setStatusError] = useState(null);
    const [hookRegistered, setHookRegistered] = useState(false);
    const [hookCode, setHookCode] = useState(null);
    const [hookReason, setHookReason] = useState("Checking Claude hook status…");
    const [hookSettingsPath, setHookSettingsPath] = useState(null);
    const [lastHookRoutingSummary, setLastHookRoutingSummary] = useState(null);
    const [lastTailerRoutingSummary, setLastTailerRoutingSummary] = useState(null);
    const [lastHookAckSummary, setLastHookAckSummary] = useState(null);
    const [lastTailerAckSummary, setLastTailerAckSummary] = useState(null);
    const [lastPlaybackSummary, setLastPlaybackSummary] = useState(null);
    const [kokoroCapabilityLabel, setKokoroCapabilityLabel] = useState(null);
    const [saveError, setSaveError] = useState(null);
    const [testState, setTestState] = useState("idle");
    const [testMessage, setTestMessage] = useState(null);
    const summarizeSettings = useSummarizeSettings();
    const ttsSettings = useMemo(() => ({
        voice: ttsVoice,
        rate: ttsRate,
        pitch: ttsPitch,
        kokoroVoice,
        kokoroSpeed,
        backendPreference: ttsBackendPreference,
    }), [kokoroSpeed, kokoroVoice, ttsBackendPreference, ttsPitch, ttsRate, ttsVoice]);
    const { backend, voices: ttsVoices, backendReason, browserAudioReady, refresh, testSpeak, isSpeaking, error, lastSuccessfulAt, lastSuccessfulBackend, } = useTextToSpeech(ttsSettings, { source: "settings_test" });
    const backendLabel = backend === "kokoro"
        ? t(strings.settings.voiceOutputSection.backendKokoro)
        : backend === "browser"
            ? t(strings.settings.voiceOutputSection.backendBrowser)
            : t(strings.settings.voiceOutputSection.backendUnavailable);
    const backendColor = backend === "kokoro" ? "text-green-400" : backend === "browser" ? "text-yellow-400" : "text-wc-text-faint";
    const preferenceLabel = ttsBackendPreference === "auto"
        ? t(strings.settings.voiceOutputSection.preferenceAuto)
        : ttsBackendPreference === "kokoro"
            ? t(strings.settings.voiceOutputSection.preferenceKokoro)
            : t(strings.settings.voiceOutputSection.preferenceBrowser);
    const loadTtsStatus = useCallback(async () => {
        setStatusLoading(true);
        setStatusError(null);
        try {
            const [hookStatus, voiceCfg] = await Promise.all([
                getTTSHookStatus(),
                getTTSConfig().catch(() => null),
            ]);
            setAutoTtsEnabled(hookStatus.config.autoEnabled);
            setTtsBackendPreference(hookStatus.config.backend);
            setStartMutedOnLoad(hookStatus.config.startMuted);
            if (voiceCfg) {
                if (voiceCfg.defaultVoice)
                    setKokoroVoice(voiceCfg.defaultVoice);
                if (voiceCfg.defaultSpeed > 0)
                    setKokoroSpeed(voiceCfg.defaultSpeed);
            }
            setHookRegistered(hookStatus.hookRegistered);
            setHookCode(hookStatus.hookCode ?? null);
            setHookReason(hookStatus.hookReason);
            setHookSettingsPath(hookStatus.hookSettingsPath ?? null);
            setKokoroCapabilityLabel(hookStatus.audioToolsCapabilityLabel ?? null);
            setLastHookRoutingSummary(hookStatus.lastHookRouting
                ? `${hookStatus.lastHookRouting.appended ? t(strings.settings.voiceOutputSection.appended) : t(strings.settings.voiceOutputSection.skipped)}: ${hookStatus.lastHookRouting.reason}`
                : null);
            setLastTailerRoutingSummary(hookStatus.lastTailerRouting
                ? `${hookStatus.lastTailerRouting.appended ? t(strings.settings.voiceOutputSection.appended) : t(strings.settings.voiceOutputSection.skipped)}: ${hookStatus.lastTailerRouting.reason}`
                : null);
            setLastHookAckSummary(hookStatus.lastHookAck
                ? `${hookStatus.lastHookAck.stage}${hookStatus.lastHookAck.backend ? ` via ${hookStatus.lastHookAck.backend}` : ""}${hookStatus.lastHookAck.message ? `: ${hookStatus.lastHookAck.message}` : ""}`
                : null);
            setLastTailerAckSummary(hookStatus.lastTailerAck
                ? `${hookStatus.lastTailerAck.stage}${hookStatus.lastTailerAck.backend ? ` via ${hookStatus.lastTailerAck.backend}` : ""}${hookStatus.lastTailerAck.message ? `: ${hookStatus.lastTailerAck.message}` : ""}`
                : null);
            setLastPlaybackSummary(hookStatus.lastPlaybackEvent
                ? `${hookStatus.lastPlaybackEvent.stage}${hookStatus.lastPlaybackEvent.backend ? ` via ${hookStatus.lastPlaybackEvent.backend}` : ""}${hookStatus.lastPlaybackEvent.message ? `: ${hookStatus.lastPlaybackEvent.message}` : ""}`
                : null);
        }
        catch (statusErrorValue) {
            setStatusError(toErrorInfo(statusErrorValue).message);
        }
        finally {
            setStatusLoading(false);
        }
    }, [setAutoTtsEnabled, setKokoroSpeed, setKokoroVoice, setStartMutedOnLoad, setTtsBackendPreference, t]);
    useEffect(() => {
        void loadTtsStatus();
    }, [loadTtsStatus]);
    const persistHookConfig = useCallback(async (patch) => {
        setSaveError(null);
        try {
            const updated = await updateTTSHookConfig(patch);
            setAutoTtsEnabled(updated.autoEnabled);
            setTtsBackendPreference(updated.backend);
            setStartMutedOnLoad(updated.startMuted);
            await refresh();
            await loadTtsStatus();
        }
        catch (persistError) {
            setSaveError(toErrorInfo(persistError).message);
        }
    }, [loadTtsStatus, refresh, setAutoTtsEnabled, setStartMutedOnLoad, setTtsBackendPreference]);
    const persistVoiceConfig = useCallback(async (patch) => {
        setSaveError(null);
        try {
            const updated = await updateTTSConfig(patch);
            if (updated.defaultVoice)
                setKokoroVoice(updated.defaultVoice);
            if (updated.defaultSpeed > 0)
                setKokoroSpeed(updated.defaultSpeed);
            await refresh();
        }
        catch (persistError) {
            setSaveError(toErrorInfo(persistError).message);
        }
    }, [refresh, setKokoroSpeed, setKokoroVoice]);
    const runTtsTest = useCallback(async () => {
        setTestState("running");
        setTestMessage(null);
        try {
            await refresh();
            await loadTtsStatus();
            await testSpeak();
            setTestState("success");
            setTestMessage(t(strings.settings.voiceOutputSection.testSucceeded));
        }
        catch (testError) {
            setTestState("error");
            setTestMessage(toErrorInfo(testError).message);
        }
        finally {
            await loadTtsStatus();
        }
    }, [loadTtsStatus, refresh, testSpeak, t]);
    return (_jsxs("div", { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.voiceOutputSection.eyebrow), title: t(strings.settings.voiceOutputSection.title), description: t(strings.settings.voiceOutputSection.description) }), _jsxs(SettingsCard, { className: "space-y-4", children: [statusError && (_jsx("div", { className: "text-xs text-wc-error-detail", children: t(strings.settings.voiceOutputSection.statusLoadFailed, { message: statusError }) })), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.activeBackend), hint: t(strings.settings.voiceOutputSection.activeBackendHint, { label: preferenceLabel, reason: backendReason }), control: _jsx("span", { "data-testid": "tts-backend-indicator", className: `text-xs font-medium ${backendColor}`, children: backendLabel }) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.backendPreference), hint: t(strings.settings.voiceOutputSection.backendPreferenceHint), control: (_jsxs("select", { "data-testid": "tts-backend-select", className: "rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", value: ttsBackendPreference, onChange: (event) => {
                                const next = event.target.value;
                                setTtsBackendPreference(next);
                                void persistHookConfig({ backend: next });
                            }, children: [_jsx("option", { value: "auto", children: t(strings.settings.voiceOutputSection.backendPreferenceAutoOption) }), _jsx("option", { value: "kokoro", children: t(strings.settings.voiceOutputSection.preferenceKokoro) }), _jsx("option", { value: "browser", children: t(strings.settings.voiceOutputSection.preferenceBrowser) })] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.autoSpeak), hint: t(strings.settings.voiceOutputSection.autoSpeakHint), control: (_jsx(SettingsToggle, { testId: "auto-tts-toggle", checked: autoTtsEnabled, onClick: () => {
                                const next = !autoTtsEnabled;
                                setAutoTtsEnabled(next);
                                void persistHookConfig({ autoEnabled: next });
                            } })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.startMuted), hint: t(strings.settings.voiceOutputSection.startMutedHint), control: (_jsx(SettingsToggle, { testId: "start-muted-toggle", checked: startMutedOnLoad, onClick: () => {
                                const next = !startMutedOnLoad;
                                setStartMutedOnLoad(next);
                                void persistHookConfig({ startMuted: next });
                            } })) }), _jsxs("div", { className: "grid gap-2 sm:grid-cols-[1fr_auto] sm:items-center", children: [_jsxs("div", { children: [_jsx("div", { className: "text-sm font-medium text-wc-text-secondary", children: t(strings.settings.voiceOutputSection.runtimeChecks) }), _jsx("div", { className: "text-[11px] text-wc-text-muted", children: statusLoading ? t(strings.settings.voiceOutputSection.runtimeChecksRefreshing) : t(strings.settings.voiceOutputSection.runtimeChecksHint) })] }), _jsxs(Button, { "data-testid": "tts-refresh", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", onClick: async () => {
                                    await refresh();
                                    await loadTtsStatus();
                                }, children: [_jsx(RefreshCw, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.voiceOutputSection.refresh)] })] }), _jsxs("div", { className: "grid gap-2 sm:grid-cols-[1fr_auto] sm:items-center", children: [_jsxs("div", { children: [_jsx("div", { className: "text-sm font-medium text-wc-text-secondary", children: t(strings.settings.voiceOutputSection.testTts) }), _jsx("div", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.voiceOutputSection.testTtsHint) })] }), _jsx(Button, { "data-testid": "tts-test-button", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", disabled: testState === "running" || isSpeaking, onClick: () => void runTtsTest(), children: testState === "running" || isSpeaking ? t(strings.settings.voiceOutputSection.testing) : t(strings.settings.voiceOutputSection.test) })] }), testMessage && (_jsx("div", { className: `text-xs ${testState === "error" ? "text-wc-error-detail" : "text-green-400"}`, children: testMessage })), saveError && (_jsx("div", { className: "text-xs text-wc-error-detail", children: t(strings.settings.voiceOutputSection.saveFailed, { message: saveError }) })), error && (_jsx("div", { className: "text-xs text-wc-error-detail", children: t(strings.settings.voiceOutputSection.playbackError, { message: error }) }))] }), _jsxs(SettingsCard, { className: "space-y-2 text-[11px] text-wc-text-faint", children: [_jsxs("div", { children: [t(strings.settings.voiceOutputSection.claudeHookPrefix), _jsx("span", { className: hookRegistered ? "text-green-400" : "text-wc-error-detail", children: hookRegistered ? t(strings.settings.voiceOutputSection.registered) : t(strings.settings.voiceOutputSection.notRegistered) }), " ", "\u00B7 ", hookReason] }), hookCode && _jsx("div", { children: t(strings.settings.voiceOutputSection.hookStatusCode, { code: hookCode }) }), _jsx("div", { children: t(strings.settings.voiceOutputSection.kokoroStatusPrefix, { label: kokoroCapabilityLabel ?? t(strings.settings.voiceOutputSection.kokoroStatusUnavailable) }) }), _jsxs("div", { children: [t(strings.settings.voiceOutputSection.browserAudioPrefix), browserAudioReady ? t(strings.settings.voiceOutputSection.browserAudioReady) : t(strings.settings.voiceOutputSection.browserAudioBlocked)] }), _jsx("div", { children: t(strings.settings.voiceOutputSection.lastHookRouting, { summary: lastHookRoutingSummary ?? t(strings.settings.voiceOutputSection.lastHookRoutingNone) }) }), _jsx("div", { children: t(strings.settings.voiceOutputSection.lastHookAck, { summary: lastHookAckSummary ?? t(strings.settings.voiceOutputSection.lastHookAckNone) }) }), _jsx("div", { children: t(strings.settings.voiceOutputSection.lastTailerRouting, { summary: lastTailerRoutingSummary ?? t(strings.settings.voiceOutputSection.lastTailerRoutingNone) }) }), _jsx("div", { children: t(strings.settings.voiceOutputSection.lastTailerAck, { summary: lastTailerAckSummary ?? t(strings.settings.voiceOutputSection.lastTailerAckNone) }) }), _jsx("div", { children: t(strings.settings.voiceOutputSection.lastPlayback, {
                            summary: lastPlaybackSummary ?? (lastSuccessfulAt
                                ? t(strings.settings.voiceOutputSection.playbackTimestamp, {
                                    time: new Date(lastSuccessfulAt).toLocaleString(),
                                    backend: lastSuccessfulBackend === "kokoro"
                                        ? t(strings.settings.voiceOutputSection.backendKokoro)
                                        : lastSuccessfulBackend === "browser"
                                            ? t(strings.settings.voiceOutputSection.backendBrowser)
                                            : t(strings.settings.voiceOutputSection.backendUnknown),
                                })
                                : t(strings.settings.voiceOutputSection.lastPlaybackNone)),
                        }) }), hookSettingsPath && _jsx("div", { className: "break-all", children: t(strings.settings.voiceOutputSection.hookSettingsPath, { path: hookSettingsPath }) })] }), (backend === "kokoro" || ttsBackendPreference === "kokoro") && (_jsxs(SettingsCard, { className: "space-y-4", children: [_jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.kokoroVoice), hint: t(strings.settings.voiceOutputSection.kokoroVoiceHint), control: (_jsx("select", { "data-testid": "kokoro-voice-select", className: "max-w-[180px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", value: kokoroVoice, onChange: (event) => {
                                const next = event.target.value;
                                setKokoroVoice(next);
                                void persistVoiceConfig({ defaultVoice: next });
                            }, children: ttsVoices.map((voice) => (_jsx("option", { value: voice.id, children: voice.name }, voice.id))) })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.kokoroSpeed), hint: t(strings.settings.voiceOutputSection.kokoroSpeedHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "kokoro-speed-slider", type: "range", min: "0.5", max: "4", step: "0.1", value: kokoroSpeed, onChange: (event) => {
                                        const next = parseFloat(event.target.value);
                                        setKokoroSpeed(next);
                                        void persistVoiceConfig({ defaultSpeed: next });
                                    }, className: "w-24 accent-[rgb(var(--wc-accent))]" }), _jsx("span", { className: "w-7 text-end text-xs text-wc-text-muted", children: kokoroSpeed.toFixed(1) })] })) })] })), _jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.voiceOutputSection.summarizationEyebrow), title: t(strings.settings.voiceOutputSection.summarizationTitle), description: t(strings.settings.voiceOutputSection.summarizationDescription) }), _jsxs(SettingsCard, { className: "space-y-4", children: [summarizeSettings.error && (_jsx("div", { className: "text-xs text-wc-error-detail", children: t(strings.settings.voiceOutputSection.summarizationLoadFailed, { message: summarizeSettings.error }) })), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.summarizeToggle), hint: t(strings.settings.voiceOutputSection.summarizeToggleHint), control: (_jsx(SettingsToggle, { testId: "summarize-toggle", checked: summarizeSettings.config?.enabled ?? false, onClick: () => {
                                const next = !(summarizeSettings.config?.enabled ?? false);
                                void summarizeSettings.save({ enabled: next });
                            } })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.wordThreshold), hint: t(strings.settings.voiceOutputSection.wordThresholdHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "summarize-threshold", type: "number", min: 100, max: 10000, step: 100, value: summarizeSettings.config?.charThreshold ?? 500, onChange: (e) => {
                                        const val = Math.max(100, parseInt(e.target.value, 10) || 500);
                                        summarizeSettings.setConfig((prev) => prev ? { ...prev, charThreshold: val } : null);
                                    }, onBlur: () => {
                                        void summarizeSettings.save({ charThreshold: summarizeSettings.config?.charThreshold ?? 500 });
                                    }, className: "w-24 rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary" }), _jsx("span", { className: "text-xs text-wc-text-faint", children: t(strings.settings.voiceOutputSection.chars) })] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.summarizationLevel), hint: t(strings.settings.voiceOutputSection.summarizationLevelHint), control: (_jsxs("select", { "data-testid": "summarize-level-select", className: "rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", value: summarizeSettings.config?.level ?? "moderate", onChange: (e) => {
                                const next = e.target.value;
                                void summarizeSettings.save({ level: next });
                            }, children: [_jsx("option", { value: "light", children: t(strings.settings.voiceOutputSection.levelLightOption) }), _jsx("option", { value: "moderate", children: t(strings.settings.voiceOutputSection.levelModerateOption) }), _jsx("option", { value: "heavy", children: t(strings.settings.voiceOutputSection.levelHeavyOption) })] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.model), hint: t(strings.settings.voiceOutputSection.modelHint), control: (_jsxs("select", { "data-testid": "summarize-model-select", className: "max-w-[220px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", disabled: summarizeSettings.loading || summarizeSettings.saving || summarizeSettings.models.length === 0, value: summarizeSettings.config?.model ?? "", onChange: (e) => void summarizeSettings.save({ model: e.target.value }), children: [summarizeSettings.models.map((model) => (_jsxs("option", { value: model.id, children: [model.displayName, model.installed ? "" : " (not installed)", model.reasoning ? " (reasoning)" : ""] }, model.id))), summarizeSettings.config?.model && !summarizeSettings.models.some((model) => model.id === summarizeSettings.config?.model) && (_jsx("option", { value: summarizeSettings.config.model, children: summarizeSettings.config.model }))] })) }), summarizeSettings.selectedModel && (_jsxs("div", { className: "rounded-md border border-wc-default bg-wc-surface-base px-3 py-2 text-[11px] text-wc-text-muted", children: [_jsxs("div", { className: "flex items-center justify-between gap-3", children: [_jsx("span", { className: "font-medium text-wc-text-secondary", children: summarizeSettings.selectedModel.statusLabel }), summarizeSettings.selectedModel.parameterSize && (_jsx("span", { className: "text-wc-text-faint", children: summarizeSettings.selectedModel.parameterSize }))] }), summarizeSettings.selectedModel.reasoning && (_jsxs("div", { "data-testid": "summarize-model-reasoning-warning", className: "mt-1 flex gap-1.5 text-wc-error-detail", children: [_jsx(AlertTriangle, { className: "mt-0.5 h-3.5 w-3.5 shrink-0" }), _jsx("span", { children: "Reasoning models are slower and are not recommended for TTS summaries." })] })), !summarizeSettings.selectedModel.installed && summarizeSettings.selectedModel.pullCommand && (_jsx("div", { className: "mt-1 break-all font-mono text-[10px] text-wc-text-secondary", "data-testid": "summarize-model-pull-command", children: summarizeSettings.selectedModel.pullCommand })), summarizeSettings.selectedModel.notes && (_jsx("div", { className: "mt-1", children: summarizeSettings.selectedModel.notes }))] })), _jsx(SettingsRow, { label: "Timeout", hint: "Maximum time to wait for local summarization.", control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "summarize-timeout", type: "number", min: 15, max: 300, step: 5, value: summarizeSettings.config?.timeoutSeconds ?? 120, onChange: (event) => {
                                        const next = Math.min(300, Math.max(15, parseInt(event.target.value, 10) || 120));
                                        summarizeSettings.setConfig((prev) => prev ? { ...prev, timeoutSeconds: next } : null);
                                    }, onBlur: () => {
                                        void summarizeSettings.save({ timeoutSeconds: summarizeSettings.config?.timeoutSeconds ?? 120 });
                                    }, className: "w-20 rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary" }), _jsx("span", { className: "text-xs text-wc-text-faint", children: "sec" })] })) })] }), backend === "browser" && (_jsxs(SettingsCard, { className: "space-y-4", children: [_jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.browserVoice), hint: t(strings.settings.voiceOutputSection.browserVoiceHint), control: (_jsxs("select", { "data-testid": "tts-voice-select", className: "max-w-[180px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary", value: ttsVoice, onChange: (event) => setTtsVoice(event.target.value), children: [_jsx("option", { value: "", children: t(strings.settings.voiceOutputSection.systemDefault) }), ttsVoices.map((voice) => (_jsx("option", { value: voice.id, children: voice.name }, voice.id)))] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.browserRate), hint: t(strings.settings.voiceOutputSection.browserRateHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "tts-rate-slider", type: "range", min: "0.5", max: "2", step: "0.1", value: ttsRate, onChange: (event) => setTtsRate(parseFloat(event.target.value)), className: "w-24 accent-[rgb(var(--wc-accent))]" }), _jsx("span", { className: "w-7 text-end text-xs text-wc-text-muted", children: ttsRate.toFixed(1) })] })) }), _jsx(SettingsRow, { label: t(strings.settings.voiceOutputSection.browserPitch), hint: t(strings.settings.voiceOutputSection.browserPitchHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": "tts-pitch-slider", type: "range", min: "0.5", max: "2", step: "0.1", value: ttsPitch, onChange: (event) => setTtsPitch(parseFloat(event.target.value)), className: "w-24 accent-[rgb(var(--wc-accent))]" }), _jsx("span", { className: "w-7 text-end text-xs text-wc-text-muted", children: ttsPitch.toFixed(1) })] })) })] }))] }));
}
