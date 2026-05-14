import { useCallback, useEffect, useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { toErrorInfo } from "../../lib/errors";
import { getTTSStatus, getTTSSummarizeConfig, updateTTSSummarizeConfig, updateTTSConfig } from "../../api/tts";
import type { TTSSummarizeConfig } from "../../api/tts";
import { useTextToSpeech } from "../../hooks/useTextToSpeech";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";

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

  // Summarization config state
  const [summarizeConfig, setSummarizeConfig] = useState<TTSSummarizeConfig | null>(null);
  const [summarizeError, setSummarizeError] = useState<string | null>(null);

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
      const status = await getTTSStatus();
      setAutoTtsEnabled(status.config.autoEnabled);
      setTtsBackendPreference(status.config.backend);
      setKokoroVoice(status.config.kokoroVoice);
      setKokoroSpeed(status.config.kokoroSpeed);
      setHookRegistered(status.hookRegistered);
      setHookCode(status.hookCode ?? null);
      setHookReason(status.hookReason);
      setHookSettingsPath(status.hookSettingsPath ?? null);
      setKokoroCapabilityLabel(status.kokoroCapabilityLabel ?? null);
      setLastHookRoutingSummary(status.lastHookRouting
        ? `${status.lastHookRouting.appended ? t(strings.settings.voiceOutputSection.appended) : t(strings.settings.voiceOutputSection.skipped)}: ${status.lastHookRouting.reason}`
        : null);
      setLastTailerRoutingSummary(status.lastTailerRouting
        ? `${status.lastTailerRouting.appended ? t(strings.settings.voiceOutputSection.appended) : t(strings.settings.voiceOutputSection.skipped)}: ${status.lastTailerRouting.reason}`
        : null);
      setLastHookAckSummary(status.lastHookAck
        ? `${status.lastHookAck.stage}${status.lastHookAck.backend ? ` via ${status.lastHookAck.backend}` : ""}${status.lastHookAck.message ? `: ${status.lastHookAck.message}` : ""}`
        : null);
      setLastTailerAckSummary(status.lastTailerAck
        ? `${status.lastTailerAck.stage}${status.lastTailerAck.backend ? ` via ${status.lastTailerAck.backend}` : ""}${status.lastTailerAck.message ? `: ${status.lastTailerAck.message}` : ""}`
        : null);
      setLastPlaybackSummary(status.lastPlaybackEvent
        ? `${status.lastPlaybackEvent.stage}${status.lastPlaybackEvent.backend ? ` via ${status.lastPlaybackEvent.backend}` : ""}${status.lastPlaybackEvent.message ? `: ${status.lastPlaybackEvent.message}` : ""}`
        : null);
      // Load summarization config alongside TTS status
      try {
        const sumCfg = await getTTSSummarizeConfig();
        setSummarizeConfig(sumCfg);
      } catch (sumError) {
        setSummarizeError(toErrorInfo(sumError).message);
      }
    } catch (statusErrorValue) {
      setStatusError(toErrorInfo(statusErrorValue).message);
    } finally {
      setStatusLoading(false);
    }
  }, [setAutoTtsEnabled, setKokoroSpeed, setKokoroVoice, setTtsBackendPreference, t]);

  useEffect(() => {
    void loadTtsStatus();
  }, [loadTtsStatus]);

  const persistTtsConfig = useCallback(async (patch: Parameters<typeof updateTTSConfig>[0]) => {
    setSaveError(null);
    try {
      const updated = await updateTTSConfig(patch);
      setAutoTtsEnabled(updated.autoEnabled);
      setTtsBackendPreference(updated.backend);
      setKokoroVoice(updated.kokoroVoice);
      setKokoroSpeed(updated.kokoroSpeed);
      await refresh();
      await loadTtsStatus();
    } catch (persistError) {
      setSaveError(toErrorInfo(persistError).message);
    }
  }, [loadTtsStatus, refresh, setAutoTtsEnabled, setKokoroSpeed, setKokoroVoice, setTtsBackendPreference]);

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
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow={t(strings.settings.voiceOutputSection.eyebrow)}
        title={t(strings.settings.voiceOutputSection.title)}
        description={t(strings.settings.voiceOutputSection.description)}
      />

      <SettingsCard className="space-y-4">
        {statusError && (
          <div className="text-xs text-wc-error-detail">{t(strings.settings.voiceOutputSection.statusLoadFailed, { message: statusError })}</div>
        )}

        <SettingsRow
          label={t(strings.settings.voiceOutputSection.activeBackend)}
          hint={t(strings.settings.voiceOutputSection.activeBackendHint, { label: preferenceLabel, reason: backendReason })}
          control={<span data-testid="tts-backend-indicator" className={`text-xs font-medium ${backendColor}`}>{backendLabel}</span>}
        />

        <SettingsRow
          label={t(strings.settings.voiceOutputSection.backendPreference)}
          hint={t(strings.settings.voiceOutputSection.backendPreferenceHint)}
          control={(
            <select
              data-testid="tts-backend-select"
              className="rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
              value={ttsBackendPreference}
              onChange={(event) => {
                const next = event.target.value as "auto" | "kokoro" | "browser";
                setTtsBackendPreference(next);
                void persistTtsConfig({ backend: next });
              }}
            >
              <option value="auto">{t(strings.settings.voiceOutputSection.backendPreferenceAutoOption)}</option>
              <option value="kokoro">{t(strings.settings.voiceOutputSection.preferenceKokoro)}</option>
              <option value="browser">{t(strings.settings.voiceOutputSection.preferenceBrowser)}</option>
            </select>
          )}
        />

        <SettingsRow
          label={t(strings.settings.voiceOutputSection.autoSpeak)}
          hint={t(strings.settings.voiceOutputSection.autoSpeakHint)}
          control={(
            <SettingsToggle
              testId="auto-tts-toggle"
              checked={autoTtsEnabled}
              onClick={() => {
                const next = !autoTtsEnabled;
                setAutoTtsEnabled(next);
                void persistTtsConfig({ autoEnabled: next });
              }}
            />
          )}
        />

        <SettingsRow
          label={t(strings.settings.voiceOutputSection.startMuted)}
          hint={t(strings.settings.voiceOutputSection.startMutedHint)}
          control={(
            <SettingsToggle
              testId="start-muted-toggle"
              checked={startMutedOnLoad}
              onClick={() => setStartMutedOnLoad(!startMutedOnLoad)}
            />
          )}
        />

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
            <RefreshCw className="mr-1 h-3.5 w-3.5" />
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
      </SettingsCard>

      <SettingsCard className="space-y-2 text-[11px] text-wc-text-faint">
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
      </SettingsCard>

      {(backend === "kokoro" || ttsBackendPreference === "kokoro") && (
        <SettingsCard className="space-y-4">
          <SettingsRow
            label={t(strings.settings.voiceOutputSection.kokoroVoice)}
            hint={t(strings.settings.voiceOutputSection.kokoroVoiceHint)}
            control={(
              <select
                data-testid="kokoro-voice-select"
                className="max-w-[180px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                value={kokoroVoice}
                onChange={(event) => {
                  const next = event.target.value;
                  setKokoroVoice(next);
                  void persistTtsConfig({ kokoroVoice: next });
                }}
              >
                {ttsVoices.map((voice) => (
                  <option key={voice.id} value={voice.id}>{voice.name}</option>
                ))}
              </select>
            )}
          />

          <SettingsRow
            label={t(strings.settings.voiceOutputSection.kokoroSpeed)}
            hint={t(strings.settings.voiceOutputSection.kokoroSpeedHint)}
            control={(
              <div className="flex items-center gap-2">
                <input
                  data-testid="kokoro-speed-slider"
                  type="range"
                  min="0.5"
                  max="4"
                  step="0.1"
                  value={kokoroSpeed}
                  onChange={(event) => {
                    const next = parseFloat(event.target.value);
                    setKokoroSpeed(next);
                    void persistTtsConfig({ kokoroSpeed: next });
                  }}
                  className="w-24 accent-[rgb(var(--wc-accent))]"
                />
                <span className="w-7 text-right text-xs text-wc-text-muted">{kokoroSpeed.toFixed(1)}</span>
              </div>
            )}
          />
        </SettingsCard>
      )}

      {/* Summarization settings */}
      <SettingsSectionIntro
        eyebrow={t(strings.settings.voiceOutputSection.summarizationEyebrow)}
        title={t(strings.settings.voiceOutputSection.summarizationTitle)}
        description={t(strings.settings.voiceOutputSection.summarizationDescription)}
      />

      <SettingsCard className="space-y-4">
        {summarizeError && (
          <div className="text-xs text-wc-error-detail">{t(strings.settings.voiceOutputSection.summarizationLoadFailed, { message: summarizeError })}</div>
        )}

        <SettingsRow
          label={t(strings.settings.voiceOutputSection.summarizeToggle)}
          hint={t(strings.settings.voiceOutputSection.summarizeToggleHint)}
          control={(
            <SettingsToggle
              testId="summarize-toggle"
              checked={summarizeConfig?.enabled ?? false}
              onClick={() => {
                const next = !(summarizeConfig?.enabled ?? false);
                setSummarizeConfig((prev) => prev ? { ...prev, enabled: next } : null);
                void updateTTSSummarizeConfig({ enabled: next })
                  .then((updated) => setSummarizeConfig(updated))
                  .catch((err) => setSummarizeError(toErrorInfo(err).message));
              }}
            />
          )}
        />

        <SettingsRow
          label={t(strings.settings.voiceOutputSection.wordThreshold)}
          hint={t(strings.settings.voiceOutputSection.wordThresholdHint)}
          control={(
            <div className="flex items-center gap-2">
              <input
                data-testid="summarize-threshold"
                type="number"
                min={100}
                max={10000}
                step={100}
                value={summarizeConfig?.charThreshold ?? 500}
                onChange={(e) => {
                  const val = Math.max(100, parseInt(e.target.value, 10) || 500);
                  setSummarizeConfig((prev) => prev ? { ...prev, charThreshold: val } : null);
                }}
                onBlur={() => {
                  void updateTTSSummarizeConfig({ charThreshold: summarizeConfig?.charThreshold ?? 500 })
                    .then((updated) => setSummarizeConfig(updated))
                    .catch((err) => setSummarizeError(toErrorInfo(err).message));
                }}
                className="w-24 rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
              />
              <span className="text-xs text-wc-text-faint">{t(strings.settings.voiceOutputSection.chars)}</span>
            </div>
          )}
        />

        <SettingsRow
          label={t(strings.settings.voiceOutputSection.summarizationLevel)}
          hint={t(strings.settings.voiceOutputSection.summarizationLevelHint)}
          control={(
            <select
              data-testid="summarize-level-select"
              className="rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
              value={summarizeConfig?.level ?? "moderate"}
              onChange={(e) => {
                const next = e.target.value as "light" | "moderate" | "heavy";
                setSummarizeConfig((prev) => prev ? { ...prev, level: next } : null);
                void updateTTSSummarizeConfig({ level: next })
                  .then((updated) => setSummarizeConfig(updated))
                  .catch((err) => setSummarizeError(toErrorInfo(err).message));
              }}
            >
              <option value="light">{t(strings.settings.voiceOutputSection.levelLightOption)}</option>
              <option value="moderate">{t(strings.settings.voiceOutputSection.levelModerateOption)}</option>
              <option value="heavy">{t(strings.settings.voiceOutputSection.levelHeavyOption)}</option>
            </select>
          )}
        />

        <SettingsRow
          label={t(strings.settings.voiceOutputSection.model)}
          hint={t(strings.settings.voiceOutputSection.modelHint)}
          control={(
            <input
              data-testid="summarize-model"
              type="text"
              value={summarizeConfig?.model ?? "qwen3:1.7b"}
              onChange={(e) => {
                setSummarizeConfig((prev) => prev ? { ...prev, model: e.target.value } : null);
              }}
              onBlur={() => {
                void updateTTSSummarizeConfig({ model: summarizeConfig?.model ?? "qwen3:1.7b" })
                  .then((updated) => setSummarizeConfig(updated))
                  .catch((err) => setSummarizeError(toErrorInfo(err).message));
              }}
              className="w-36 rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
            />
          )}
        />
      </SettingsCard>

      {backend === "browser" && (
        <SettingsCard className="space-y-4">
          <SettingsRow
            label={t(strings.settings.voiceOutputSection.browserVoice)}
            hint={t(strings.settings.voiceOutputSection.browserVoiceHint)}
            control={(
              <select
                data-testid="tts-voice-select"
                className="max-w-[180px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                value={ttsVoice}
                onChange={(event) => setTtsVoice(event.target.value)}
              >
                <option value="">{t(strings.settings.voiceOutputSection.systemDefault)}</option>
                {ttsVoices.map((voice) => (
                  <option key={voice.id} value={voice.id}>{voice.name}</option>
                ))}
              </select>
            )}
          />

          <SettingsRow
            label={t(strings.settings.voiceOutputSection.browserRate)}
            hint={t(strings.settings.voiceOutputSection.browserRateHint)}
            control={(
              <div className="flex items-center gap-2">
                <input
                  data-testid="tts-rate-slider"
                  type="range"
                  min="0.5"
                  max="2"
                  step="0.1"
                  value={ttsRate}
                  onChange={(event) => setTtsRate(parseFloat(event.target.value))}
                  className="w-24 accent-[rgb(var(--wc-accent))]"
                />
                <span className="w-7 text-right text-xs text-wc-text-muted">{ttsRate.toFixed(1)}</span>
              </div>
            )}
          />

          <SettingsRow
            label={t(strings.settings.voiceOutputSection.browserPitch)}
            hint={t(strings.settings.voiceOutputSection.browserPitchHint)}
            control={(
              <div className="flex items-center gap-2">
                <input
                  data-testid="tts-pitch-slider"
                  type="range"
                  min="0.5"
                  max="2"
                  step="0.1"
                  value={ttsPitch}
                  onChange={(event) => setTtsPitch(parseFloat(event.target.value))}
                  className="w-24 accent-[rgb(var(--wc-accent))]"
                />
                <span className="w-7 text-right text-xs text-wc-text-muted">{ttsPitch.toFixed(1)}</span>
              </div>
            )}
          />
        </SettingsCard>
      )}
    </div>
  );
}
