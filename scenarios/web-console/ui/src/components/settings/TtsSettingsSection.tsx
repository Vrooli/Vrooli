import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { NumberField } from "@vrooli/react-component-library/NumberField";
import { Button } from "../ui/button";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { toErrorInfo } from "../../lib/errors";
import {
  getTTSConfig,
  updateTTSConfig,
} from "../../audio-integration";
import type { TTSVoiceInfo } from "../../audio-integration";
import { getTTSHookStatus, updateTTSHookConfig } from "../../api/ttsHook";
import { useTextToSpeech } from "../../hooks/useTextToSpeech";
import { SettingsSlider, SettingsToggle } from "./primitives";
import { useSummarizeSettings } from "./useSummarizeSettings";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/0.1.4";

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
  const [statusError, setStatusError] = useState<string | null>(null);
  const [hookRegistered, setHookRegistered] = useState(false);
  const [hookCode, setHookCode] = useState<string | null>(null);
  const [hookReason, setHookReason] = useState("Checking Claude hook status…");
  const [hookSettingsPath, setHookSettingsPath] = useState<string | null>(null);
  const [lastHookRoutingSummary, setLastHookRoutingSummary] = useState<string | null>(null);
  const [lastTailerRoutingSummary, setLastTailerRoutingSummary] = useState<string | null>(null);
  const [lastHookAckSummary, setLastHookAckSummary] = useState<string | null>(null);
  const [lastTailerAckSummary, setLastTailerAckSummary] = useState<string | null>(null);
  const [lastPlaybackSummary, setLastPlaybackSummary] = useState<string | null>(null);
  const [kokoroCapabilityLabel, setKokoroCapabilityLabel] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [testState, setTestState] = useState<"idle" | "running" | "success" | "error">("idle");
  const [testMessage, setTestMessage] = useState<string | null>(null);

  const summarizeSettings = useSummarizeSettings();

  const ttsSettings = useMemo(() => ({
    voice: ttsVoice,
    rate: ttsRate,
    pitch: ttsPitch,
    kokoroVoice,
    kokoroSpeed,
    backendPreference: ttsBackendPreference,
  }), [kokoroSpeed, kokoroVoice, ttsBackendPreference, ttsPitch, ttsRate, ttsVoice]);

  const {
    backend,
    voices: ttsVoices,
    backendReason,
    browserAudioReady,
    refresh,
    testSpeak,
    isSpeaking,
    error,
    lastSuccessfulAt,
    lastSuccessfulBackend,
  } = useTextToSpeech(ttsSettings, { source: "settings_test" });

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
        if (voiceCfg.defaultVoice) setKokoroVoice(voiceCfg.defaultVoice);
        if (voiceCfg.defaultSpeed > 0) setKokoroSpeed(voiceCfg.defaultSpeed);
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
    } catch (statusErrorValue) {
      setStatusError(toErrorInfo(statusErrorValue).message);
    } finally {
      setStatusLoading(false);
    }
  }, [setAutoTtsEnabled, setKokoroSpeed, setKokoroVoice, setStartMutedOnLoad, setTtsBackendPreference, t]);

  useEffect(() => {
    void loadTtsStatus();
  }, [loadTtsStatus]);

  const persistHookConfig = useCallback(async (patch: { autoEnabled?: boolean; backend?: "auto" | "kokoro" | "browser"; startMuted?: boolean }) => {
    setSaveError(null);
    try {
      const updated = await updateTTSHookConfig(patch);
      setAutoTtsEnabled(updated.autoEnabled);
      setTtsBackendPreference(updated.backend);
      setStartMutedOnLoad(updated.startMuted);
      await refresh();
      await loadTtsStatus();
    } catch (persistError) {
      setSaveError(toErrorInfo(persistError).message);
    }
  }, [loadTtsStatus, refresh, setAutoTtsEnabled, setStartMutedOnLoad, setTtsBackendPreference]);

  const persistVoiceConfig = useCallback(async (patch: { defaultVoice?: string; defaultSpeed?: number }) => {
    setSaveError(null);
    try {
      const updated = await updateTTSConfig(patch);
      if (updated.defaultVoice) setKokoroVoice(updated.defaultVoice);
      if (updated.defaultSpeed > 0) setKokoroSpeed(updated.defaultSpeed);
      await refresh();
    } catch (persistError) {
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
    } catch (testError) {
      setTestState("error");
      setTestMessage(toErrorInfo(testError).message);
    } finally {
      await loadTtsStatus();
    }
  }, [loadTtsStatus, refresh, testSpeak, t]);

  return (
    <SettingsList>
      <SettingsList.Intro
        eyebrow={t(strings.settings.voiceOutputSection.eyebrow)}
        title={t(strings.settings.voiceOutputSection.title)}
        description={t(strings.settings.voiceOutputSection.description)}
      />

      <SettingsList.Group>
        {statusError && (
          <div className="text-xs text-wc-error-detail">{t(strings.settings.voiceOutputSection.statusLoadFailed, { message: statusError })}</div>
        )}

        <SettingsList.Row
          label={t(strings.settings.voiceOutputSection.activeBackend)}
          hint={t(strings.settings.voiceOutputSection.activeBackendHint, { label: preferenceLabel, reason: backendReason })} control="compact">{<span data-testid="tts-backend-indicator" className={`text-xs font-medium ${backendColor}`}>{backendLabel}</span>}</SettingsList.Row>

        <SettingsList.Row
          label={t(strings.settings.voiceOutputSection.backendPreference)}
          hint={t(strings.settings.voiceOutputSection.backendPreferenceHint)} control="wide">{(
            <select
              data-testid="tts-backend-select"
              className="rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
              value={ttsBackendPreference}
              onChange={(event) => {
                const next = event.target.value as "auto" | "kokoro" | "browser";
                setTtsBackendPreference(next);
                void persistHookConfig({ backend: next });
              }}
            >
              <option value="auto">{t(strings.settings.voiceOutputSection.backendPreferenceAutoOption)}</option>
              <option value="kokoro">{t(strings.settings.voiceOutputSection.preferenceKokoro)}</option>
              <option value="browser">{t(strings.settings.voiceOutputSection.preferenceBrowser)}</option>
            </select>
          )}</SettingsList.Row>

        <SettingsList.Row
          label={t(strings.settings.voiceOutputSection.autoSpeak)}
          hint={t(strings.settings.voiceOutputSection.autoSpeakHint)} control="compact">{(
            <SettingsToggle
              testId="auto-tts-toggle"
              checked={autoTtsEnabled}
              onCheckedChange={(next) => {
                setAutoTtsEnabled(next);
                void persistHookConfig({ autoEnabled: next });
              }}
            />
          )}</SettingsList.Row>

        <SettingsList.Row
          label={t(strings.settings.voiceOutputSection.startMuted)}
          hint={t(strings.settings.voiceOutputSection.startMutedHint)} control="compact">{(
            <SettingsToggle
              testId="start-muted-toggle"
              checked={startMutedOnLoad}
              onCheckedChange={(next) => {
                setStartMutedOnLoad(next);
                void persistHookConfig({ startMuted: next });
              }}
            />
          )}</SettingsList.Row>

        <div className="grid gap-2 sm:grid-cols-[1fr_auto] sm:items-center">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">{t(strings.settings.voiceOutputSection.runtimeChecks)}</div>
            <div className="text-[11px] text-wc-text-muted">
              {statusLoading ? t(strings.settings.voiceOutputSection.runtimeChecksRefreshing) : t(strings.settings.voiceOutputSection.runtimeChecksHint)}
            </div>
          </div>
          <Button
            data-testid="tts-refresh"
            variant="outline"
            size="sm"
            className="h-8 px-3 text-xs"
            onClick={async () => {
              await refresh();
              await loadTtsStatus();
            }}
          >
            <RefreshCw className="me-1 h-3.5 w-3.5" />
            {t(strings.settings.voiceOutputSection.refresh)}
          </Button>
        </div>

        <div className="grid gap-2 sm:grid-cols-[1fr_auto] sm:items-center">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">{t(strings.settings.voiceOutputSection.testTts)}</div>
            <div className="text-[11px] text-wc-text-muted">
              {t(strings.settings.voiceOutputSection.testTtsHint)}
            </div>
          </div>
          <Button
            data-testid="tts-test-button"
            variant="outline"
            size="sm"
            className="h-8 px-3 text-xs"
            disabled={testState === "running" || isSpeaking}
            onClick={() => void runTtsTest()}
          >
            {testState === "running" || isSpeaking ? t(strings.settings.voiceOutputSection.testing) : t(strings.settings.voiceOutputSection.test)}
          </Button>
        </div>

        {testMessage && (
          <div className={`text-xs ${testState === "error" ? "text-wc-error-detail" : "text-green-400"}`}>
            {testMessage}
          </div>
        )}

        {saveError && (
          <div className="text-xs text-wc-error-detail">{t(strings.settings.voiceOutputSection.saveFailed, { message: saveError })}</div>
        )}

        {error && (
          <div className="text-xs text-wc-error-detail">{t(strings.settings.voiceOutputSection.playbackError, { message: error })}</div>
        )}
      </SettingsList.Group>

      <SettingsList.Group className="text-[11px] text-wc-text-faint">
        <div>
          {t(strings.settings.voiceOutputSection.claudeHookPrefix)}
          <span className={hookRegistered ? "text-green-400" : "text-wc-error-detail"}>
            {hookRegistered ? t(strings.settings.voiceOutputSection.registered) : t(strings.settings.voiceOutputSection.notRegistered)}
          </span>{" "}
          · {hookReason}
        </div>
        {hookCode && <div>{t(strings.settings.voiceOutputSection.hookStatusCode, { code: hookCode })}</div>}
        <div>{t(strings.settings.voiceOutputSection.kokoroStatusPrefix, { label: kokoroCapabilityLabel ?? t(strings.settings.voiceOutputSection.kokoroStatusUnavailable) })}</div>
        <div>{t(strings.settings.voiceOutputSection.browserAudioPrefix)}{browserAudioReady ? t(strings.settings.voiceOutputSection.browserAudioReady) : t(strings.settings.voiceOutputSection.browserAudioBlocked)}</div>
        <div>{t(strings.settings.voiceOutputSection.lastHookRouting, { summary: lastHookRoutingSummary ?? t(strings.settings.voiceOutputSection.lastHookRoutingNone) })}</div>
        <div>{t(strings.settings.voiceOutputSection.lastHookAck, { summary: lastHookAckSummary ?? t(strings.settings.voiceOutputSection.lastHookAckNone) })}</div>
        <div>{t(strings.settings.voiceOutputSection.lastTailerRouting, { summary: lastTailerRoutingSummary ?? t(strings.settings.voiceOutputSection.lastTailerRoutingNone) })}</div>
        <div>{t(strings.settings.voiceOutputSection.lastTailerAck, { summary: lastTailerAckSummary ?? t(strings.settings.voiceOutputSection.lastTailerAckNone) })}</div>
        <div>
          {t(strings.settings.voiceOutputSection.lastPlayback, {
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
          })}
        </div>
        {hookSettingsPath && <div className="break-all">{t(strings.settings.voiceOutputSection.hookSettingsPath, { path: hookSettingsPath })}</div>}
      </SettingsList.Group>

      {(backend === "kokoro" || ttsBackendPreference === "kokoro") && (
        <SettingsList.Group>
          <SettingsList.Row
            label={t(strings.settings.voiceOutputSection.kokoroVoice)}
            hint={t(strings.settings.voiceOutputSection.kokoroVoiceHint)} control="wide">{(
              <select
                data-testid="kokoro-voice-select"
                className="max-w-[180px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                value={kokoroVoice}
                onChange={(event) => {
                  const next = event.target.value;
                  setKokoroVoice(next);
                  void persistVoiceConfig({ defaultVoice: next });
                }}
              >
                {ttsVoices.map((voice: TTSVoiceInfo) => (
                  <option key={voice.id} value={voice.id}>{voice.name}</option>
                ))}
              </select>
            )}</SettingsList.Row>

          <SettingsList.Row
            label={t(strings.settings.voiceOutputSection.kokoroSpeed)}
            hint={t(strings.settings.voiceOutputSection.kokoroSpeedHint)} control="wide">{(
              <SettingsSlider
                testId="kokoro-speed-slider"
                value={kokoroSpeed}
                onCommit={(next) => {
                  setKokoroSpeed(next);
                  void persistVoiceConfig({ defaultSpeed: next });
                }}
                min={0.5}
                max={4}
                step={0.1}
                defaultMarker={1}
                formatValue={(value) => value.toFixed(1)}
              />
            )}</SettingsList.Row>
        </SettingsList.Group>
      )}

      {/* Summarization — sourced from / persisted to audio-tools via audio-integration. */}
      <SettingsList.Intro
        eyebrow={t(strings.settings.voiceOutputSection.summarizationEyebrow)}
        title={t(strings.settings.voiceOutputSection.summarizationTitle)}
        description={t(strings.settings.voiceOutputSection.summarizationDescription)}
      />

      <SettingsList.Group>
        {summarizeSettings.error && (
          <div className="text-xs text-wc-error-detail">{t(strings.settings.voiceOutputSection.summarizationLoadFailed, { message: summarizeSettings.error })}</div>
        )}

        <SettingsList.Row
          label={t(strings.settings.voiceOutputSection.summarizeToggle)}
          hint={t(strings.settings.voiceOutputSection.summarizeToggleHint)} control="compact">{(
            <SettingsToggle
              testId="summarize-toggle"
              checked={summarizeSettings.config?.enabled ?? false}
              onCheckedChange={(next) => {
                void summarizeSettings.save({ enabled: next });
              }}
            />
          )}</SettingsList.Row>

        <SettingsList.Row
          label={t(strings.settings.voiceOutputSection.wordThreshold)}
          hint={t(strings.settings.voiceOutputSection.wordThresholdHint)} control="compact">{(
            /* Previously a `type="number"` whose onChange ran
               `Math.max(100, parseInt(...) || 500)`: the floor was enforced,
               the declared 10000 ceiling never was, and any draft parsing to 0
               silently jumped to 500. NumberField enforces both bounds on
               every path and commits on blur rather than per keystroke. */
            <NumberField
              testId="summarize-threshold"
              label={t(strings.settings.voiceOutputSection.wordThreshold)}
              value={summarizeSettings.config?.charThreshold ?? 500}
              onChange={(next) => {
                summarizeSettings.setConfig((prev) => prev ? { ...prev, charThreshold: next } : null);
                void summarizeSettings.save({ charThreshold: next });
              }}
              min={100}
              max={10 * 1000}
              step={100}
              unit={t(strings.settings.voiceOutputSection.chars)}
              size="sm"
            />
          )}</SettingsList.Row>

        <SettingsList.Row
          label={t(strings.settings.voiceOutputSection.summarizationLevel)}
          hint={t(strings.settings.voiceOutputSection.summarizationLevelHint)} control="wide">{(
            <select
              data-testid="summarize-level-select"
              className="rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
              value={summarizeSettings.config?.level ?? "moderate"}
              onChange={(e) => {
                const next = e.target.value as "light" | "moderate" | "heavy";
                void summarizeSettings.save({ level: next });
              }}
            >
              <option value="light">{t(strings.settings.voiceOutputSection.levelLightOption)}</option>
              <option value="moderate">{t(strings.settings.voiceOutputSection.levelModerateOption)}</option>
              <option value="heavy">{t(strings.settings.voiceOutputSection.levelHeavyOption)}</option>
            </select>
          )}</SettingsList.Row>

        <SettingsList.Row
          label={t(strings.settings.voiceOutputSection.model)}
          hint={t(strings.settings.voiceOutputSection.modelHint)} control="wide">{(
            <select
              data-testid="summarize-model-select"
              className="max-w-[220px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
              disabled={summarizeSettings.loading || summarizeSettings.saving || summarizeSettings.models.length === 0}
              value={summarizeSettings.config?.model ?? ""}
              onChange={(e) => void summarizeSettings.save({ model: e.target.value })}
            >
              {summarizeSettings.models.map((model) => (
                <option key={model.id} value={model.id}>
                  {model.displayName}{model.installed ? "" : " (not installed)"}{model.reasoning ? " (reasoning)" : ""}
                </option>
              ))}
              {summarizeSettings.config?.model && !summarizeSettings.models.some((model) => model.id === summarizeSettings.config?.model) && (
                <option value={summarizeSettings.config.model}>{summarizeSettings.config.model}</option>
              )}
            </select>
          )}</SettingsList.Row>

        {summarizeSettings.selectedModel && (
          <div className="rounded-md border border-wc-default bg-wc-surface-base px-3 py-2 text-[11px] text-wc-text-muted">
            <div className="flex items-center justify-between gap-3">
              <span className="font-medium text-wc-text-secondary">{summarizeSettings.selectedModel.statusLabel}</span>
              {summarizeSettings.selectedModel.parameterSize && (
                <span className="text-wc-text-faint">{summarizeSettings.selectedModel.parameterSize}</span>
              )}
            </div>
            {summarizeSettings.selectedModel.reasoning && (
              <div data-testid="summarize-model-reasoning-warning" className="mt-1 flex gap-1.5 text-wc-error-detail">
                <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                <span>Reasoning models are slower and are not recommended for TTS summaries.</span>
              </div>
            )}
            {!summarizeSettings.selectedModel.installed && summarizeSettings.selectedModel.pullCommand && (
              <div className="mt-1 break-all font-mono text-[10px] text-wc-text-secondary" data-testid="summarize-model-pull-command">
                {summarizeSettings.selectedModel.pullCommand}
              </div>
            )}
            {summarizeSettings.selectedModel.notes && (
              <div className="mt-1">{summarizeSettings.selectedModel.notes}</div>
            )}
          </div>
        )}

        <SettingsList.Row
          label="Timeout"
          hint="Maximum time to wait for local summarization." control="compact">{(
            <NumberField
              testId="summarize-timeout"
              label="Timeout"
              value={summarizeSettings.config?.timeoutSeconds ?? 120}
              onChange={(next) => {
                summarizeSettings.setConfig((prev) => prev ? { ...prev, timeoutSeconds: next } : null);
                void summarizeSettings.save({ timeoutSeconds: next });
              }}
              min={15}
              max={300}
              step={5}
              unit="sec"
              size="sm"
            />
          )}</SettingsList.Row>
      </SettingsList.Group>

      {backend === "browser" && (
        <SettingsList.Group>
          <SettingsList.Row
            label={t(strings.settings.voiceOutputSection.browserVoice)}
            hint={t(strings.settings.voiceOutputSection.browserVoiceHint)} control="wide">{(
              <select
                data-testid="tts-voice-select"
                className="max-w-[180px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                value={ttsVoice}
                onChange={(event) => setTtsVoice(event.target.value)}
              >
                <option value="">{t(strings.settings.voiceOutputSection.systemDefault)}</option>
                {ttsVoices.map((voice: TTSVoiceInfo) => (
                  <option key={voice.id} value={voice.id}>{voice.name}</option>
                ))}
              </select>
            )}</SettingsList.Row>

          <SettingsList.Row
            label={t(strings.settings.voiceOutputSection.browserRate)}
            hint={t(strings.settings.voiceOutputSection.browserRateHint)} control="wide">{(
              <SettingsSlider
                testId="tts-rate-slider"
                value={ttsRate}
                onCommit={setTtsRate}
                min={0.5}
                max={2}
                step={0.1}
                defaultMarker={1}
                formatValue={(value) => value.toFixed(1)}
              />
            )}</SettingsList.Row>

          <SettingsList.Row
            label={t(strings.settings.voiceOutputSection.browserPitch)}
            hint={t(strings.settings.voiceOutputSection.browserPitchHint)} control="wide">{(
              <SettingsSlider
                testId="tts-pitch-slider"
                value={ttsPitch}
                onCommit={setTtsPitch}
                min={0.5}
                max={2}
                step={0.1}
                defaultMarker={1}
                formatValue={(value) => value.toFixed(1)}
              />
            )}</SettingsList.Row>
        </SettingsList.Group>
      )}
    </SettingsList>
  );
}
