import { useCallback, useEffect, useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import { Button } from "../ui/button";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { getTTSStatus, getTTSSummarizeConfig, updateTTSSummarizeConfig, toErrorInfo, updateTTSConfig } from "../../lib/api";
import type { TTSSummarizeConfig } from "../../lib/api";
import { useTextToSpeech } from "../../hooks/useTextToSpeech";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";

export default function TtsSettingsSection() {
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

  const backendLabel = backend === "kokoro" ? "Kokoro" : backend === "browser" ? "Browser" : "Unavailable";
  const backendColor = backend === "kokoro" ? "text-green-400" : backend === "browser" ? "text-yellow-400" : "text-wc-text-faint";
  const preferenceLabel = ttsBackendPreference === "auto"
    ? "Auto"
    : ttsBackendPreference === "kokoro"
      ? "Kokoro only"
      : "Browser only";

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
        ? `${status.lastHookRouting.appended ? "Appended" : "Skipped"}: ${status.lastHookRouting.reason}`
        : null);
      setLastTailerRoutingSummary(status.lastTailerRouting
        ? `${status.lastTailerRouting.appended ? "Appended" : "Skipped"}: ${status.lastTailerRouting.reason}`
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
  }, [setAutoTtsEnabled, setKokoroSpeed, setKokoroVoice, setTtsBackendPreference]);

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
      setTestMessage("Test playback succeeded.");
    } catch (testError) {
      setTestState("error");
      setTestMessage(toErrorInfo(testError).message);
    } finally {
      await loadTtsStatus();
    }
  }, [loadTtsStatus, refresh, testSpeak]);

  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow="Speech Synthesis"
        title="Voice output"
        description="Choose the playback backend, verify hook status, and tune the current speaking experience."
      />

      <SettingsCard className="space-y-4">
        {statusError && (
          <div className="text-xs text-wc-error-detail">Failed to load TTS status: {statusError}</div>
        )}

        <SettingsRow
          label="Active backend"
          hint={`Preference: ${preferenceLabel}. ${backendReason}`}
          control={<span data-testid="tts-backend-indicator" className={`text-xs font-medium ${backendColor}`}>{backendLabel}</span>}
        />

        <SettingsRow
          label="Backend preference"
          hint="Auto picks the best available option for this environment."
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
              <option value="auto">Auto (best available)</option>
              <option value="kokoro">Kokoro only</option>
              <option value="browser">Browser only</option>
            </select>
          )}
        />

        <SettingsRow
          label="Auto-speak AI responses"
          hint="Read assistant responses aloud automatically."
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
          label="Start muted on app load"
          hint="When enabled, audio is muted on app load. Tap the speaker icon to unmute."
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
            <div className="text-sm font-medium text-wc-text-secondary">Runtime checks</div>
            <div className="text-[11px] text-wc-text-muted">
              {statusLoading ? "Refreshing TTS diagnostics…" : "Refresh backend and hook status."}
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
            Refresh
          </Button>
        </div>

        <div className="grid gap-2 sm:grid-cols-[1fr_auto] sm:items-center">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">Test TTS</div>
            <div className="text-[11px] text-wc-text-muted">
              Play a short sample using the current backend decision.
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
            {testState === "running" || isSpeaking ? "Testing…" : "Test"}
          </Button>
        </div>

        {testMessage && (
          <div className={`text-xs ${testState === "error" ? "text-wc-error-detail" : "text-green-400"}`}>
            {testMessage}
          </div>
        )}

        {saveError && (
          <div className="text-xs text-wc-error-detail">Failed to save TTS settings: {saveError}</div>
        )}

        {error && (
          <div className="text-xs text-wc-error-detail">Playback error: {error}</div>
        )}
      </SettingsCard>

      <SettingsCard className="space-y-2 text-[11px] text-wc-text-faint">
        <div>
          Claude hook:{" "}
          <span className={hookRegistered ? "text-green-400" : "text-wc-error-detail"}>
            {hookRegistered ? "Registered" : "Not registered"}
          </span>{" "}
          · {hookReason}
        </div>
        {hookCode && <div>Hook status code: {hookCode}</div>}
        <div>Kokoro: {kokoroCapabilityLabel ?? "Status unavailable"}</div>
        <div>Browser audio: {browserAudioReady ? "Ready" : "Blocked until you interact with the page"}</div>
        <div>Last Claude hook routing: {lastHookRoutingSummary ?? "No Claude hook routing has been recorded yet"}</div>
        <div>Last Claude terminal ack: {lastHookAckSummary ?? "No Claude terminal acknowledgment has been recorded yet"}</div>
        <div>Last Codex tailer routing: {lastTailerRoutingSummary ?? "No Codex tailer routing has been recorded yet"}</div>
        <div>Last Codex terminal ack: {lastTailerAckSummary ?? "No Codex terminal acknowledgment has been recorded yet"}</div>
        <div>
          Last playback event: {lastPlaybackSummary ?? (lastSuccessfulAt
            ? `${new Date(lastSuccessfulAt).toLocaleString()} via ${lastSuccessfulBackend === "kokoro" ? "Kokoro" : lastSuccessfulBackend === "browser" ? "Browser" : "Unknown"}`
            : "None recorded yet")}
        </div>
        {hookSettingsPath && <div className="break-all">Hook settings file: {hookSettingsPath}</div>}
      </SettingsCard>

      {(backend === "kokoro" || ttsBackendPreference === "kokoro") && (
        <SettingsCard className="space-y-4">
          <SettingsRow
            label="Kokoro voice"
            hint="Pick the Kokoro voice to use when Kokoro is active."
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
            label="Speed"
            hint="Adjust Kokoro playback speed."
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
        eyebrow="Summarization"
        title="Long response summarization"
        description="Summarize long AI responses via Ollama before TTS playback. Full text is always preserved in the Messages view."
      />

      <SettingsCard className="space-y-4">
        {summarizeError && (
          <div className="text-xs text-wc-error-detail">Failed to load summarize config: {summarizeError}</div>
        )}

        <SettingsRow
          label="Summarize long responses"
          hint="When enabled, responses longer than the word threshold are summarized before being read aloud."
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
          label="Word threshold"
          hint="Responses with fewer characters than this are read in full (not summarized)."
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
              <span className="text-xs text-wc-text-faint">chars</span>
            </div>
          )}
        />

        <SettingsRow
          label="Summarization level"
          hint="Light preserves more detail; heavy gives a brief spoken overview."
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
              <option value="light">Light (~60% of original)</option>
              <option value="moderate">Moderate (~40% of original)</option>
              <option value="heavy">Heavy (2-3 sentences)</option>
            </select>
          )}
        />

        <SettingsRow
          label="Model"
          hint="Ollama model used for summarization."
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
            label="Voice"
            hint="Choose the browser speech synthesis voice."
            control={(
              <select
                data-testid="tts-voice-select"
                className="max-w-[180px] rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                value={ttsVoice}
                onChange={(event) => setTtsVoice(event.target.value)}
              >
                <option value="">System default</option>
                {ttsVoices.map((voice) => (
                  <option key={voice.id} value={voice.id}>{voice.name}</option>
                ))}
              </select>
            )}
          />

          <SettingsRow
            label="Rate"
            hint="Adjust browser speech speed."
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
            label="Pitch"
            hint="Adjust browser speech pitch."
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
