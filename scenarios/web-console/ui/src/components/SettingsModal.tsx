import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import {
  X,
  GripHorizontal,
  Plus,
  Trash2,
  Save,
  AlertCircle,
  LayoutGrid,
  LayoutList,
  Keyboard,
  CheckCircle,
  Circle,
  RefreshCw,
  Mic,
  ChevronDown,
  ChevronRight,
  RotateCcw,
} from "lucide-react";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import { Button } from "./ui/button";
import IntegrationsPanel from "./IntegrationsPanel";
import HeaderColorPicker from "./appearance/HeaderColorPicker";
import ThemePicker from "./appearance/ThemePicker";
import FontSizeStepper from "./appearance/FontSizeStepper";
import {
  type ShortcutProfile,
  type ShortcutEntry,
  type CapabilityState,
  listShortcutProfiles,
  upsertShortcutProfile,
  deleteShortcutProfile,
  fetchCapabilities,
  toErrorInfo,
  type VoiceStreamConfig,
  getVoiceStreamConfig,
  updateVoiceStreamConfig,
  updateTTSConfig,
  getTTSStatus,
} from "../lib/api";
import { formatShortcutFromEvent } from "../lib/shortcutParser";
import { useTextToSpeech } from "../hooks/useTextToSpeech";

function ShortcutEditor({
  profile,
  onSave,
  onDelete,
}: {
  profile: ShortcutProfile;
  onSave: (p: ShortcutProfile) => void;
  onDelete: (id: string) => void;
}) {
  const [entries, setEntries] = useState<ShortcutEntry[]>(profile.shortcuts);
  const [name, setName] = useState(profile.name);
  const [dirty, setDirty] = useState(false);

  const updateEntry = (
    idx: number,
    field: keyof ShortcutEntry,
    value: string,
  ) => {
    setEntries((prev) =>
      prev.map((e, i) => (i === idx ? { ...e, [field]: value } : e)),
    );
    setDirty(true);
  };

  const addEntry = () => {
    setEntries((prev) => [...prev, { label: "", command: "" }]);
    setDirty(true);
  };

  const removeEntry = (idx: number) => {
    setEntries((prev) => prev.filter((_, i) => i !== idx));
    setDirty(true);
  };

  return (
    <div
      data-testid={`shortcut-profile-${profile.id}`}
      className="rounded-lg border border-wc-default bg-wc-surface-input p-3"
    >
      <div className="flex items-center justify-between mb-2">
        <input
          data-testid={`profile-name-${profile.id}`}
          className="bg-transparent text-sm font-medium text-wc-text-primary border-b border-transparent focus:border-wc-accent outline-none"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setDirty(true);
          }}
        />
        <div className="flex gap-1">
          {dirty && (
            <Button
              data-testid={`profile-save-${profile.id}`}
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={() => {
                onSave({ ...profile, name, shortcuts: entries });
                setDirty(false);
              }}
              title="Save changes"
            >
              <Save className="h-3 w-3 text-green-400" />
            </Button>
          )}
          <Button
            data-testid={`profile-delete-${profile.id}`}
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={() => onDelete(profile.id)}
            title="Delete profile"
          >
            <Trash2 className="h-3 w-3 text-wc-text-faint hover:text-wc-error-detail" />
          </Button>
        </div>
      </div>

      <div className="space-y-1.5">
        {entries.map((entry, idx) => (
          <div key={idx} className="flex items-center gap-1.5">
            <input
              data-testid={`entry-label-${profile.id}-${idx}`}
              placeholder="Label"
              className="flex-1 rounded border border-wc-default bg-wc-surface-base px-2 py-0.5 text-xs text-wc-text-primary outline-none focus:border-wc-accent"
              value={entry.label}
              onChange={(e) => updateEntry(idx, "label", e.target.value)}
            />
            <input
              data-testid={`entry-command-${profile.id}-${idx}`}
              placeholder="Command"
              className="flex-[2] rounded border border-wc-default bg-wc-surface-base px-2 py-0.5 text-xs font-mono text-wc-text-primary outline-none focus:border-wc-accent"
              value={entry.command}
              onChange={(e) => updateEntry(idx, "command", e.target.value)}
            />
            <Button
              variant="ghost"
              size="icon"
              className="h-5 w-5 shrink-0"
              onClick={() => removeEntry(idx)}
            >
              <Trash2 className="h-2.5 w-2.5 text-wc-text-faint" />
            </Button>
          </div>
        ))}
      </div>

      <Button
        variant="ghost"
        size="sm"
        className="mt-1.5 text-xs text-wc-text-faint"
        onClick={addEntry}
      >
        <Plus className="mr-1 h-3 w-3" /> Add shortcut
      </Button>
    </div>
  );
}

function TtsSettingsSection() {
  const ttsVoice = useWorkspaceStore((s) => s.ttsVoice);
  const setTtsVoice = useWorkspaceStore((s) => s.setTtsVoice);
  const ttsRate = useWorkspaceStore((s) => s.ttsRate);
  const setTtsRate = useWorkspaceStore((s) => s.setTtsRate);
  const ttsPitch = useWorkspaceStore((s) => s.ttsPitch);
  const setTtsPitch = useWorkspaceStore((s) => s.setTtsPitch);
  const autoTtsEnabled = useWorkspaceStore((s) => s.autoTtsEnabled);
  const setAutoTtsEnabled = useWorkspaceStore((s) => s.setAutoTtsEnabled);
  const ttsBackendPreference = useWorkspaceStore((s) => s.ttsBackendPreference);
  const setTtsBackendPreference = useWorkspaceStore((s) => s.setTtsBackendPreference);
  const kokoroVoice = useWorkspaceStore((s) => s.kokoroVoice);
  const setKokoroVoice = useWorkspaceStore((s) => s.setKokoroVoice);
  const kokoroSpeed = useWorkspaceStore((s) => s.kokoroSpeed);
  const setKokoroSpeed = useWorkspaceStore((s) => s.setKokoroSpeed);
  const [statusLoading, setStatusLoading] = useState(true);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [hookRegistered, setHookRegistered] = useState(false);
  const [hookReason, setHookReason] = useState("Checking Claude hook status…");
  const [hookSettingsPath, setHookSettingsPath] = useState<string | null>(null);
  const [lastDeliverySummary, setLastDeliverySummary] = useState<string | null>(null);
  const [kokoroCapabilityLabel, setKokoroCapabilityLabel] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [testState, setTestState] = useState<"idle" | "running" | "success" | "error">("idle");
  const [testMessage, setTestMessage] = useState<string | null>(null);

  const ttsSettings = useMemo(
    () => ({
      voice: ttsVoice,
      rate: ttsRate,
      pitch: ttsPitch,
      kokoroVoice,
      kokoroSpeed,
      backendPreference: ttsBackendPreference,
    }),
    [ttsVoice, ttsRate, ttsPitch, kokoroVoice, kokoroSpeed, ttsBackendPreference],
  );
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
  } = useTextToSpeech(ttsSettings);

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
      setHookReason(status.hookReason);
      setHookSettingsPath(status.hookSettingsPath ?? null);
      setKokoroCapabilityLabel(status.kokoroCapabilityLabel ?? null);
      if (status.lastDelivery) {
        const sourceLabel = status.lastDelivery.source.split("_").join(" ");
        const summary = `${status.lastDelivery.delivered ? "Delivered" : "Skipped"} via ${sourceLabel}: ${status.lastDelivery.reason}`;
        setLastDeliverySummary(summary);
      } else {
        setLastDeliverySummary(null);
      }
    } catch (err) {
      setStatusError(toErrorInfo(err).message);
    } finally {
      setStatusLoading(false);
    }
  }, [
    setAutoTtsEnabled,
    setKokoroSpeed,
    setKokoroVoice,
    setTtsBackendPreference,
  ]);

  useEffect(() => {
    loadTtsStatus();
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
    } catch (err) {
      setSaveError(toErrorInfo(err).message);
    }
  }, [
    loadTtsStatus,
    refresh,
    setAutoTtsEnabled,
    setKokoroSpeed,
    setKokoroVoice,
    setTtsBackendPreference,
  ]);

  const runTtsTest = useCallback(async () => {
    setTestState("running");
    setTestMessage(null);
    try {
      await refresh();
      await loadTtsStatus();
      await testSpeak();
      setTestState("success");
      setTestMessage("Test playback succeeded.");
    } catch (err) {
      setTestState("error");
      setTestMessage(toErrorInfo(err).message);
    } finally {
      await loadTtsStatus();
    }
  }, [loadTtsStatus, refresh, testSpeak]);

  return (
    <section>
      <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
        Voice Output (TTS)
      </h3>
      <div className="rounded-lg border border-wc-default bg-wc-surface-input p-3 space-y-3">
        {statusError && (
          <div className="text-[11px] text-wc-error-detail">
            Failed to load TTS status: {statusError}
          </div>
        )}

        {/* Backend indicator */}
        <div className="flex items-center justify-between">
          <span className="text-sm text-wc-text-secondary">Active backend</span>
          <span data-testid="tts-backend-indicator" className={`text-xs font-medium ${backendColor}`}>{backendLabel}</span>
        </div>
        <p className="text-[11px] text-wc-text-faint">
          Preference: {preferenceLabel}. {backendReason}
        </p>

        {/* Backend preference */}
        <div className="flex items-center justify-between">
          <span className="text-sm text-wc-text-secondary">Backend preference</span>
          <select
            data-testid="tts-backend-select"
            className="text-xs bg-wc-surface-base border border-wc-default rounded px-1.5 py-0.5 text-wc-text-primary"
            value={ttsBackendPreference}
            onChange={(e) => {
              const next = e.target.value as "auto" | "kokoro" | "browser";
              setTtsBackendPreference(next);
              persistTtsConfig({ backend: next });
            }}
          >
            <option value="auto">Auto (best available)</option>
            <option value="kokoro">Kokoro only</option>
            <option value="browser">Browser only</option>
          </select>
        </div>

        {/* Auto-speak toggle */}
        <div className="flex items-center justify-between">
          <div className="flex flex-col">
            <span className="text-sm text-wc-text-secondary">Auto-speak AI responses</span>
            <span className="text-[10px] text-wc-text-muted">Automatically speak AI assistant responses</span>
          </div>
          <button
            data-testid="auto-tts-toggle"
            role="switch"
            aria-checked={autoTtsEnabled}
            className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
              autoTtsEnabled ? "bg-wc-accent" : "bg-wc-surface-base"
            }`}
            onClick={() => {
              const next = !autoTtsEnabled;
              setAutoTtsEnabled(next);
              persistTtsConfig({ autoEnabled: next });
            }}
          >
            <span
              className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                autoTtsEnabled ? "translate-x-[18px]" : "translate-x-[3px]"
              }`}
            />
          </button>
        </div>

        <div className="grid grid-cols-[1fr_auto] items-center gap-2">
          <div>
            <div className="text-sm text-wc-text-secondary">Runtime checks</div>
            <div className="text-[11px] text-wc-text-faint">
              {statusLoading ? "Refreshing TTS diagnostics…" : "Refresh backend and hook status."}
            </div>
          </div>
          <Button
            data-testid="tts-refresh"
            variant="outline"
            size="sm"
            className="h-7 px-2"
            onClick={async () => {
              await refresh();
              await loadTtsStatus();
            }}
          >
            <RefreshCw className="mr-1 h-3 w-3" />
            Refresh
          </Button>
        </div>

        <div className="grid grid-cols-[1fr_auto] items-center gap-2">
          <div>
            <div className="text-sm text-wc-text-secondary">Test TTS</div>
            <div className="text-[11px] text-wc-text-faint">
              Play a short sample using the current backend decision.
            </div>
          </div>
          <Button
            data-testid="tts-test-button"
            variant="outline"
            size="sm"
            className="h-7 px-2"
            disabled={testState === "running" || isSpeaking}
            onClick={runTtsTest}
          >
            {testState === "running" || isSpeaking ? "Testing…" : "Test"}
          </Button>
        </div>

        {testMessage && (
          <div className={`text-[11px] ${testState === "error" ? "text-wc-error-detail" : "text-green-400"}`}>
            {testMessage}
          </div>
        )}

        {saveError && (
          <div className="text-[11px] text-wc-error-detail">
            Failed to save TTS settings: {saveError}
          </div>
        )}

        {error && (
          <div className="text-[11px] text-wc-error-detail">
            Playback error: {error}
          </div>
        )}

        <div className="rounded border border-wc-default bg-wc-surface-base px-2 py-2 text-[11px] text-wc-text-faint space-y-1">
          <div>
            Claude hook:{" "}
            <span className={hookRegistered ? "text-green-400" : "text-wc-error-detail"}>
              {hookRegistered ? "Registered" : "Not registered"}
            </span>
            {" "}· {hookReason}
          </div>
          <div>
            Kokoro: {kokoroCapabilityLabel ?? "Status unavailable"}
          </div>
          <div>
            Browser audio: {browserAudioReady ? "Ready" : "Blocked until you interact with the page"}
          </div>
          <div>
            Last delivery: {lastDeliverySummary ?? "No auto-TTS delivery has been recorded yet"}
          </div>
          <div>
            Last successful playback: {lastSuccessfulAt ? `${new Date(lastSuccessfulAt).toLocaleString()} via ${lastSuccessfulBackend === "kokoro" ? "Kokoro" : lastSuccessfulBackend === "browser" ? "Browser" : "Unknown"}` : "None in this tab yet"}
          </div>
          {hookSettingsPath && (
            <div className="break-all">
              Hook settings file: {hookSettingsPath}
            </div>
          )}
        </div>

        {/* Kokoro-specific settings */}
        {(backend === "kokoro" || ttsBackendPreference === "kokoro") && (
          <>
            <div className="flex items-center justify-between">
              <span className="text-sm text-wc-text-secondary">Kokoro voice</span>
              <select
                data-testid="kokoro-voice-select"
                className="text-xs bg-wc-surface-base border border-wc-default rounded px-1.5 py-0.5 text-wc-text-primary max-w-[180px]"
                value={kokoroVoice}
                onChange={(e) => {
                  const next = e.target.value;
                  setKokoroVoice(next);
                  persistTtsConfig({ kokoroVoice: next });
                }}
              >
                {ttsVoices.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-wc-text-secondary">Speed</span>
              <div className="flex items-center gap-2">
                <input
                  data-testid="kokoro-speed-slider"
                  type="range"
                  min="0.5"
                  max="4"
                  step="0.1"
                  value={kokoroSpeed}
                  onChange={(e) => {
                    const next = parseFloat(e.target.value);
                    setKokoroSpeed(next);
                    persistTtsConfig({ kokoroSpeed: next });
                  }}
                  className="w-24 accent-[rgb(var(--wc-accent))]"
                />
                <span className="text-xs text-wc-text-muted w-7 text-right">{kokoroSpeed.toFixed(1)}</span>
              </div>
            </div>
          </>
        )}

        {/* Browser-specific settings */}
        {backend === "browser" && (
          <>
            <div className="flex items-center justify-between">
              <span className="text-sm text-wc-text-secondary">Voice</span>
              <select
                data-testid="tts-voice-select"
                className="text-xs bg-wc-surface-base border border-wc-default rounded px-1.5 py-0.5 text-wc-text-primary max-w-[180px]"
                value={ttsVoice}
                onChange={(e) => setTtsVoice(e.target.value)}
              >
                <option value="">System default</option>
                {ttsVoices.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-wc-text-secondary">Rate</span>
              <div className="flex items-center gap-2">
                <input
                  data-testid="tts-rate-slider"
                  type="range"
                  min="0.5"
                  max="2"
                  step="0.1"
                  value={ttsRate}
                  onChange={(e) => setTtsRate(parseFloat(e.target.value))}
                  className="w-24 accent-[rgb(var(--wc-accent))]"
                />
                <span className="text-xs text-wc-text-muted w-7 text-right">{ttsRate.toFixed(1)}</span>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-wc-text-secondary">Pitch</span>
              <div className="flex items-center gap-2">
                <input
                  data-testid="tts-pitch-slider"
                  type="range"
                  min="0.5"
                  max="2"
                  step="0.1"
                  value={ttsPitch}
                  onChange={(e) => setTtsPitch(parseFloat(e.target.value))}
                  className="w-24 accent-[rgb(var(--wc-accent))]"
                />
                <span className="text-xs text-wc-text-muted w-7 text-right">{ttsPitch.toFixed(1)}</span>
              </div>
            </div>
          </>
        )}

        {backend === "none" && (
          <p className="text-[11px] text-wc-text-faint">
            No TTS backend is currently usable. {backendReason}
          </p>
        )}
      </div>
    </section>
  );
}

export default function SettingsModal() {
  const settingsModalOpen = useWorkspaceStore((s) => s.settingsModalOpen);
  const setSettingsModalOpen = useWorkspaceStore((s) => s.setSettingsModalOpen);
  const isMinimapVisible = useWorkspaceStore((s) => s.isMinimapVisible);
  const setMinimapVisible = useWorkspaceStore((s) => s.setMinimapVisible);
  const displayMode = useWorkspaceStore((s) => s.displayMode);
  const setDisplayMode = useWorkspaceStore((s) => s.setDisplayMode);
  const toolbarLayout = useWorkspaceStore((s) => s.toolbarLayout);
  const setToolbarLayout = useWorkspaceStore((s) => s.setToolbarLayout);
  const defaultHeaderColor = useWorkspaceStore((s) => s.defaultHeaderColor);
  const defaultThemeId = useWorkspaceStore((s) => s.defaultThemeId);
  const defaultFontSize = useWorkspaceStore((s) => s.defaultFontSize);
  const setDefaultHeaderColor = useWorkspaceStore((s) => s.setDefaultHeaderColor);
  const setDefaultThemeId = useWorkspaceStore((s) => s.setDefaultThemeId);
  const setDefaultFontSize = useWorkspaceStore((s) => s.setDefaultFontSize);
  const voiceEnabled = useWorkspaceStore((s) => s.voiceEnabled);
  const setVoiceEnabled = useWorkspaceStore((s) => s.setVoiceEnabled);
  const voiceShortcut = useWorkspaceStore((s) => s.voiceShortcut);
  const setVoiceShortcut = useWorkspaceStore((s) => s.setVoiceShortcut);
  const vadAutoStop = useWorkspaceStore((s) => s.vadAutoStop);
  const setVadAutoStop = useWorkspaceStore((s) => s.setVadAutoStop);
  const vadSilenceTimeoutMs = useWorkspaceStore((s) => s.vadSilenceTimeoutMs);
  const setVadSilenceTimeoutMs = useWorkspaceStore((s) => s.setVadSilenceTimeoutMs);
  const voiceLanguage = useWorkspaceStore((s) => s.voiceLanguage);
  const setVoiceLanguage = useWorkspaceStore((s) => s.setVoiceLanguage);
  const [recordingShortcut, setRecordingShortcut] = useState(false);

  // Voice capability status
  const [voiceCaps, setVoiceCaps] = useState<CapabilityState[]>([]);
  const [voiceCapsLoading, setVoiceCapsLoading] = useState(false);
  const [voiceCapsError, setVoiceCapsError] = useState<string | null>(null);
  const hasWebSpeech = typeof window !== "undefined" &&
    !!("SpeechRecognition" in window || "webkitSpeechRecognition" in window);

  // Microphone permission state
  const [micPermission, setMicPermission] = useState<"granted" | "denied" | "prompt" | "unknown">("unknown");
  const [micRequesting, setMicRequesting] = useState(false);

  // Advanced voice streaming config (server-side pipeline tuning)
  const [vsConfig, setVsConfig] = useState<VoiceStreamConfig | null>(null);
  const [vsConfigLoading, setVsConfigLoading] = useState(false);
  const [vsConfigError, setVsConfigError] = useState<string | null>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const checkMicPermission = useCallback(async () => {
    try {
      const result = await navigator.permissions.query({ name: "microphone" as PermissionName });
      setMicPermission(result.state as "granted" | "denied" | "prompt");
      // Listen for changes (e.g. user changes permission in browser settings)
      result.onchange = () => setMicPermission(result.state as "granted" | "denied" | "prompt");
    } catch {
      // Permissions API not supported — try getUserMedia to check
      setMicPermission("unknown");
    }
  }, []);

  useEffect(() => {
    if (!settingsModalOpen) return;
    checkMicPermission();
  }, [settingsModalOpen, checkMicPermission]);

  const requestMicPermission = useCallback(async () => {
    setMicRequesting(true);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      stream.getTracks().forEach((t) => t.stop());
      setMicPermission("granted");
      // Re-initialize voice input by toggling voiceEnabled
      if (voiceEnabled) {
        setVoiceEnabled(false);
        // Small delay so the useEffect cleanup runs before re-enabling
        setTimeout(() => setVoiceEnabled(true), 50);
      }
    } catch {
      setMicPermission("denied");
    } finally {
      setMicRequesting(false);
    }
  }, [voiceEnabled, setVoiceEnabled]);

  const loadVoiceCaps = useCallback(async (signal?: { cancelled: boolean }) => {
    setVoiceCapsLoading(true);
    setVoiceCapsError(null);
    try {
      const data = await fetchCapabilities();
      if (signal?.cancelled) return;
      setVoiceCaps(data.capabilities);
    } catch (err) {
      if (signal?.cancelled) return;
      setVoiceCapsError(toErrorInfo(err).message);
    } finally {
      if (!signal?.cancelled) setVoiceCapsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!settingsModalOpen) return;
    const signal = { cancelled: false };
    loadVoiceCaps(signal);
    return () => { signal.cancelled = true; };
  }, [settingsModalOpen, loadVoiceCaps]);

  // Fetch voice streaming config when advanced section is expanded
  const loadVoiceStreamConfig = useCallback(async (signal?: { cancelled: boolean }) => {
    setVsConfigLoading(true);
    setVsConfigError(null);
    try {
      const cfg = await getVoiceStreamConfig();
      if (!signal?.cancelled) setVsConfig(cfg);
    } catch (err) {
      if (!signal?.cancelled) setVsConfigError(toErrorInfo(err).message);
    } finally {
      if (!signal?.cancelled) setVsConfigLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!advancedOpen) return;
    const signal = { cancelled: false };
    loadVoiceStreamConfig(signal);
    return () => { signal.cancelled = true; };
  }, [advancedOpen, loadVoiceStreamConfig]);

  // Debounced save: update local state immediately, PUT after 500ms idle
  const handleVsConfigChange = useCallback(
    (patch: Partial<VoiceStreamConfig>) => {
      setVsConfig((prev) => (prev ? { ...prev, ...patch } : null));
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(async () => {
        try {
          const updated = await updateVoiceStreamConfig(patch);
          setVsConfig(updated);
          setVsConfigError(null);
        } catch (err) {
          setVsConfigError(toErrorInfo(err).message);
        }
      }, 500);
    },
    [],
  );

  const resetVsConfig = useCallback(async () => {
    try {
      const updated = await updateVoiceStreamConfig({
        flushIntervalMs: 500,
        minDeltaBytes: 4096,
        overlapBytes: 2048,
      });
      setVsConfig(updated);
      setVsConfigError(null);
    } catch (err) {
      setVsConfigError(toErrorInfo(err).message);
    }
  }, []);

  const { elementRef, floatingStyle, pointerHandlers, handleClickCapture } =
    useDraggablePosition({
      isActive: settingsModalOpen,
      storageKey: "wc-settings-pos",
      defaultPosition: () => {
        if (typeof window === "undefined") return { x: 100, y: 100 };
        return {
          x: Math.max(12, (window.innerWidth - 448) / 2),
          y: Math.max(12, window.innerHeight * 0.1),
        };
      },
    });

  // Shortcut profiles state
  const [profiles, setProfiles] = useState<ShortcutProfile[]>([]);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [profileLoading, setProfileLoading] = useState(true);

  const loadProfiles = useCallback(
    async (signal?: { cancelled: boolean }) => {
      setProfileLoading(true);
      try {
        const data = await listShortcutProfiles();
        if (signal?.cancelled) return;
        setProfiles(data);
        setProfileError(null);
      } catch (err) {
        if (signal?.cancelled) return;
        setProfileError(toErrorInfo(err).message);
      } finally {
        if (!signal?.cancelled) setProfileLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    if (!settingsModalOpen) return;
    const signal = { cancelled: false };
    loadProfiles(signal);
    return () => {
      signal.cancelled = true;
    };
  }, [settingsModalOpen, loadProfiles]);

  const handleSaveProfile = useCallback(
    async (profile: ShortcutProfile) => {
      try {
        const updated = await upsertShortcutProfile({
          id: profile.id,
          scope: profile.scope,
          name: profile.name,
          shortcuts: profile.shortcuts,
        });
        setProfiles((prev) =>
          prev.map((p) => (p.id === updated.id ? updated : p)),
        );
      } catch (err) {
        setProfileError(toErrorInfo(err).message);
        loadProfiles();
      }
    },
    [loadProfiles],
  );

  const handleDeleteProfile = useCallback(
    async (id: string) => {
      try {
        await deleteShortcutProfile(id);
        setProfiles((prev) => prev.filter((p) => p.id !== id));
      } catch (err) {
        setProfileError(toErrorInfo(err).message);
        loadProfiles();
      }
    },
    [loadProfiles],
  );

  const handleCreateProfile = useCallback(async () => {
    try {
      const newProfile = await upsertShortcutProfile({
        id: `profile-${Date.now()}`,
        scope: "workspace",
        name: "New Profile",
        shortcuts: [{ label: "List files", command: "ls -la" }],
      });
      setProfiles((prev) => [...prev, newProfile]);
    } catch (err) {
      setProfileError(toErrorInfo(err).message);
    }
  }, []);

  if (!settingsModalOpen) return null;

  const close = () => setSettingsModalOpen(false);

  return (
    <>
      {/* Backdrop */}
      <div
        data-testid="settings-backdrop"
        className="fixed inset-0 z-40 bg-wc-backdrop"
        onClick={close}
      />

      {/* Modal */}
      <div
        ref={(node) => { elementRef.current = node; }}
        data-testid="settings-modal"
        className="fixed left-0 top-0 z-50 w-[28rem] max-w-[calc(100vw-24px)] max-h-[80vh] overflow-hidden rounded-lg border border-wc-default bg-wc-surface-raised shadow-2xl flex flex-col"
        style={floatingStyle}
        onPointerDown={(e) => {
          const target = e.target as HTMLElement | null;
          const isOnHandle = Boolean(target?.closest("[data-drag-handle]"));
          const isOnControl = Boolean(target?.closest("button, a, input, textarea, select"));
          if (isOnHandle && !isOnControl) {
            pointerHandlers.onPointerDown(e);
          }
        }}
        onPointerMove={pointerHandlers.onPointerMove}
        onPointerUp={pointerHandlers.onPointerUp}
        onPointerCancel={pointerHandlers.onPointerCancel}
        onClickCapture={handleClickCapture}
      >
        {/* Drag handle header */}
        <div
          data-drag-handle
          className="flex items-center justify-between px-4 py-2 border-b border-wc-default cursor-grab active:cursor-grabbing select-none touch-none"
        >
          <div className="flex items-center gap-2">
            <GripHorizontal className="h-4 w-4 text-wc-text-faint" />
            <h2 className="text-sm font-semibold text-wc-text-primary">
              Settings
            </h2>
          </div>
          <Button
            data-testid="settings-close"
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={close}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto p-4 space-y-5">
          {/* Section 1: Shortcut Profiles */}
          <section>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">
                Shortcut Profiles
              </h3>
              <Button
                data-testid="create-profile"
                variant="outline"
                size="sm"
                className="text-xs"
                onClick={handleCreateProfile}
              >
                <Plus className="mr-1 h-3 w-3" /> New Profile
              </Button>
            </div>

            {profileError && (
              <div
                data-testid="settings-error"
                className="flex items-center gap-2 rounded border border-wc-error bg-wc-error-surface px-2 py-1.5 text-xs text-wc-error-detail mb-2"
              >
                <AlertCircle className="h-3 w-3 shrink-0" />
                {profileError}
              </div>
            )}

            {profileLoading ? (
              <div className="text-center text-xs text-wc-text-faint py-2">
                Loading...
              </div>
            ) : profiles.length === 0 ? (
              <div className="text-center text-xs text-wc-text-faint py-2">
                No shortcut profiles configured
              </div>
            ) : (
              <div className="space-y-2">
                {profiles.map((profile) => (
                  <ShortcutEditor
                    key={profile.id}
                    profile={profile}
                    onSave={handleSaveProfile}
                    onDelete={handleDeleteProfile}
                  />
                ))}
              </div>
            )}
          </section>

          {/* Section 2: Layout */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
              Layout
            </h3>
            <div className="rounded-lg border border-wc-default bg-wc-surface-input p-3 space-y-3">
              {/* Display Mode */}
              <div className="flex items-center justify-between">
                <span className="text-sm text-wc-text-secondary">Layout</span>
                <div className="flex items-center gap-1">
                  <Button
                    data-testid="display-mode-grid"
                    variant={displayMode === "grid" ? "default" : "outline"}
                    size="sm"
                    className="h-7 px-2"
                    onClick={() => setDisplayMode("grid")}
                  >
                    <LayoutGrid className="h-3.5 w-3.5 mr-1" />
                    Grid
                  </Button>
                  <Button
                    data-testid="display-mode-tabs"
                    variant={displayMode === "tabs" ? "default" : "outline"}
                    size="sm"
                    className="h-7 px-2"
                    onClick={() => setDisplayMode("tabs")}
                  >
                    <LayoutList className="h-3.5 w-3.5 mr-1" />
                    Tabs
                  </Button>
                </div>
              </div>

              {/* Mobile Toolbar Layout */}
              <div className="flex items-center justify-between">
                <span className="text-sm text-wc-text-secondary">Mobile toolbar</span>
                <div className="flex items-center gap-1">
                  <Button
                    data-testid="toolbar-layout-compact"
                    variant={toolbarLayout === "compact" ? "default" : "outline"}
                    size="sm"
                    className="h-7 px-2"
                    onClick={() => setToolbarLayout("compact")}
                  >
                    Compact
                  </Button>
                  <Button
                    data-testid="toolbar-layout-expanded"
                    variant={toolbarLayout === "expanded" ? "default" : "outline"}
                    size="sm"
                    className="h-7 px-2"
                    onClick={() => setToolbarLayout("expanded")}
                  >
                    Expanded
                  </Button>
                </div>
              </div>

              {/* Minimap Toggle (only relevant in grid mode) */}
              {displayMode === "grid" && (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-wc-text-secondary">Minimap</span>
                  <button
                    data-testid="minimap-toggle"
                    role="switch"
                    aria-checked={isMinimapVisible}
                    className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                      isMinimapVisible ? "bg-wc-accent" : "bg-wc-surface-base"
                    }`}
                    onClick={() => setMinimapVisible(!isMinimapVisible)}
                  >
                    <span
                      className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                        isMinimapVisible ? "translate-x-[18px]" : "translate-x-[3px]"
                      }`}
                    />
                  </button>
                </div>
              )}
            </div>
          </section>

          {/* Section: Voice Input */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
              Voice Input
            </h3>
            <div className="rounded-lg border border-wc-default bg-wc-surface-input p-3 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-wc-text-secondary">Enabled</span>
                <button
                  data-testid="voice-enabled-toggle"
                  role="switch"
                  aria-checked={voiceEnabled}
                  className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                    voiceEnabled ? "bg-wc-accent" : "bg-wc-surface-base"
                  }`}
                  onClick={() => setVoiceEnabled(!voiceEnabled)}
                >
                  <span
                    className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                      voiceEnabled ? "translate-x-[18px]" : "translate-x-[3px]"
                    }`}
                  />
                </button>
              </div>
              {voiceEnabled && (
                <>
                  <div className="flex items-center justify-between">
                    <div className="flex flex-col">
                      <span className="text-sm text-wc-text-secondary">Auto-stop on silence</span>
                      <span className="text-[10px] text-wc-text-muted">Stop recording when you stop speaking (tap mode only)</span>
                    </div>
                    <button
                      data-testid="vad-auto-stop-toggle"
                      role="switch"
                      aria-checked={vadAutoStop}
                      className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                        vadAutoStop ? "bg-wc-accent" : "bg-wc-surface-base"
                      }`}
                      onClick={() => setVadAutoStop(!vadAutoStop)}
                    >
                      <span
                        className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                          vadAutoStop ? "translate-x-[18px]" : "translate-x-[3px]"
                        }`}
                      />
                    </button>
                  </div>
                  {vadAutoStop && (
                    <div className="flex items-center justify-between">
                      <div className="flex flex-col">
                        <span className="text-sm text-wc-text-secondary">Silence timeout</span>
                        <span className="text-[10px] text-wc-text-muted">How long to wait after speech stops</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <input
                          data-testid="vad-silence-timeout-slider"
                          type="range"
                          min={1000}
                          max={5000}
                          step={250}
                          value={vadSilenceTimeoutMs}
                          onChange={(e) => setVadSilenceTimeoutMs(Number(e.target.value))}
                          className="w-20 accent-wc-accent"
                        />
                        <span className="text-xs text-wc-text-muted w-8 text-right">{(vadSilenceTimeoutMs / 1000).toFixed(1)}s</span>
                      </div>
                    </div>
                  )}
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-wc-text-secondary">Language</span>
                    <select
                      data-testid="voice-language-select"
                      className="text-xs bg-wc-surface-base border border-wc-default rounded px-1.5 py-0.5 text-wc-text-primary"
                      value={voiceLanguage}
                      onChange={(e) => setVoiceLanguage(e.target.value)}
                    >
                      <option value="auto">Auto-detect</option>
                      <option value="en-US">English (US)</option>
                      <option value="en-GB">English (UK)</option>
                      <option value="es-ES">Spanish</option>
                      <option value="fr-FR">French</option>
                      <option value="de-DE">German</option>
                      <option value="zh-CN">Chinese (Simplified)</option>
                      <option value="ja-JP">Japanese</option>
                      <option value="ko-KR">Korean</option>
                      <option value="pt-BR">Portuguese (Brazil)</option>
                      <option value="hi-IN">Hindi</option>
                    </select>
                  </div>
                </>
              )}
              <div className="flex items-center justify-between">
                <span className="text-sm text-wc-text-secondary">Shortcut</span>
                <div className="flex items-center gap-1.5">
                  {recordingShortcut ? (
                    <span
                      data-testid="voice-shortcut-recording"
                      className="text-xs font-mono text-wc-accent bg-wc-surface-base px-2 py-0.5 rounded border border-wc-accent animate-pulse"
                      tabIndex={0}
                      onKeyDown={(e) => {
                        // Ignore lone modifier presses
                        if (["Control", "Alt", "Shift", "Meta"].includes(e.key)) return;
                        e.preventDefault();
                        const shortcut = formatShortcutFromEvent(e.nativeEvent);
                        setVoiceShortcut(shortcut);
                        setRecordingShortcut(false);
                      }}
                      onBlur={() => setRecordingShortcut(false)}
                      ref={(el) => el?.focus()}
                    >
                      Press a key combo...
                    </span>
                  ) : (
                    <span
                      data-testid="voice-shortcut-display"
                      className="text-xs font-mono text-wc-text-muted bg-wc-surface-base px-2 py-0.5 rounded"
                    >
                      {voiceShortcut}
                    </span>
                  )}
                  <Button
                    data-testid="voice-shortcut-change"
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6"
                    onClick={() => setRecordingShortcut(true)}
                    title="Change shortcut"
                  >
                    <Keyboard className="h-3 w-3 text-wc-text-faint" />
                  </Button>
                </div>
              </div>

              {/* Backend status */}
              <div className="border-t border-wc-default pt-2 mt-1">
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-[11px] font-medium text-wc-text-muted uppercase tracking-wider">Backends</span>
                  <Button
                    data-testid="voice-caps-refresh"
                    variant="ghost"
                    size="icon"
                    className="h-5 w-5"
                    onClick={() => loadVoiceCaps()}
                    title="Refresh"
                  >
                    <RefreshCw className={`h-3 w-3 text-wc-text-faint ${voiceCapsLoading ? "animate-spin" : ""}`} />
                  </Button>
                </div>
                {voiceCapsError && (
                  <div className="flex items-center gap-1.5 text-[11px] text-wc-error-detail mb-1">
                    <AlertCircle className="h-3 w-3 shrink-0" />
                    {voiceCapsError}
                  </div>
                )}
                <div className="space-y-1">
                  {voiceCaps.filter((c) => c.features.includes("voice-input")).map((cap) => (
                    <div key={cap.id} className="flex items-center gap-1.5 text-[11px]">
                      {cap.status === "available" ? (
                        <CheckCircle className="h-3 w-3 shrink-0 text-green-400" />
                      ) : (
                        <Circle className="h-3 w-3 shrink-0 text-wc-text-faint" />
                      )}
                      <span className={cap.status === "available" ? "text-wc-text-secondary" : "text-wc-text-faint"}>
                        {cap.name}
                      </span>
                      <span className={`ml-auto ${cap.status === "available" ? "text-green-400" : "text-wc-text-faint"}`}>
                        {cap.status}
                      </span>
                    </div>
                  ))}
                  <div className="flex items-center gap-1.5 text-[11px]">
                    {hasWebSpeech ? (
                      <CheckCircle className="h-3 w-3 shrink-0 text-green-400" />
                    ) : (
                      <Circle className="h-3 w-3 shrink-0 text-wc-text-faint" />
                    )}
                    <span className={hasWebSpeech ? "text-wc-text-secondary" : "text-wc-text-faint"}>
                      Web Speech API
                    </span>
                    <span className={`ml-auto ${hasWebSpeech ? "text-green-400" : "text-wc-text-faint"}`}>
                      {hasWebSpeech ? "available" : "unavailable"}
                    </span>
                  </div>
                </div>
                {!voiceCapsLoading && voiceCaps.every((c) => c.status !== "available") && !hasWebSpeech && (
                  <p className="mt-1.5 text-[10px] text-amber-400">
                    No transcription backend available. Install Whisper or use a Chromium-based browser for Web Speech API.
                  </p>
                )}
              </div>

              {/* Microphone permission status */}
              <div className="border-t border-wc-default pt-2 mt-1">
                <span className="text-[11px] font-medium text-wc-text-muted uppercase tracking-wider">Microphone</span>
                <div className="mt-1.5 flex items-center gap-1.5 text-[11px]">
                  {micPermission === "granted" ? (
                    <>
                      <CheckCircle className="h-3 w-3 shrink-0 text-green-400" />
                      <span className="text-wc-text-secondary">Permission granted</span>
                    </>
                  ) : micPermission === "denied" ? (
                    <>
                      <AlertCircle className="h-3 w-3 shrink-0 text-red-400" />
                      <span className="text-red-400">Permission denied</span>
                    </>
                  ) : micPermission === "prompt" ? (
                    <>
                      <Circle className="h-3 w-3 shrink-0 text-wc-text-faint" />
                      <span className="text-wc-text-faint">Not yet requested</span>
                    </>
                  ) : (
                    <>
                      <Circle className="h-3 w-3 shrink-0 text-wc-text-faint" />
                      <span className="text-wc-text-faint">Unknown</span>
                    </>
                  )}
                </div>
                {micPermission === "denied" && (
                  <p className="mt-1 text-[10px] text-wc-text-faint">
                    Microphone access was blocked. Click the lock/site-settings icon in your browser&apos;s address bar to allow microphone access, then refresh.
                  </p>
                )}
                {(micPermission === "prompt" || micPermission === "unknown") && (
                  <Button
                    data-testid="mic-request-permission"
                    variant="outline"
                    size="sm"
                    className="mt-1.5 text-[11px] h-6"
                    onClick={requestMicPermission}
                    disabled={micRequesting}
                  >
                    <Mic className="mr-1 h-3 w-3" />
                    {micRequesting ? "Requesting..." : "Allow microphone"}
                  </Button>
                )}
              </div>

              {/* Advanced streaming config (server-side pipeline tuning) */}
              {voiceEnabled && (
                <div className="border-t border-wc-default pt-2 mt-1">
                  <button
                    data-testid="advanced-streaming-toggle"
                    className="flex items-center gap-1 text-[11px] font-medium text-wc-text-muted uppercase tracking-wider w-full"
                    onClick={() => setAdvancedOpen(!advancedOpen)}
                  >
                    {advancedOpen ? (
                      <ChevronDown className="h-3 w-3" />
                    ) : (
                      <ChevronRight className="h-3 w-3" />
                    )}
                    Advanced Streaming
                  </button>
                  {advancedOpen && (
                    <div className="mt-2 space-y-3">
                      {vsConfigLoading && (
                        <div className="text-center text-xs text-wc-text-faint py-1">Loading...</div>
                      )}
                      {vsConfigError && (
                        <div className="flex items-center gap-1.5 text-[11px] text-wc-error-detail">
                          <AlertCircle className="h-3 w-3 shrink-0" />
                          {vsConfigError}
                        </div>
                      )}
                      {vsConfig && (
                        <>
                          <div className="flex items-center justify-between">
                            <div className="flex flex-col">
                              <span className="text-sm text-wc-text-secondary">Flush interval</span>
                              <span className="text-[10px] text-wc-text-muted">How often audio is sent to Whisper</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <input
                                data-testid="vs-flush-interval"
                                type="range"
                                min={100}
                                max={5000}
                                step={50}
                                value={vsConfig.flushIntervalMs}
                                onChange={(e) => handleVsConfigChange({ flushIntervalMs: Number(e.target.value) })}
                                className="w-20 accent-wc-accent"
                              />
                              <span className="text-xs text-wc-text-muted w-12 text-right">{vsConfig.flushIntervalMs}ms</span>
                            </div>
                          </div>
                          <div className="flex items-center justify-between">
                            <div className="flex flex-col">
                              <span className="text-sm text-wc-text-secondary">Min chunk size</span>
                              <span className="text-[10px] text-wc-text-muted">Min audio before a partial fires</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <input
                                data-testid="vs-min-delta"
                                type="range"
                                min={512}
                                max={32768}
                                step={512}
                                value={vsConfig.minDeltaBytes}
                                onChange={(e) => handleVsConfigChange({ minDeltaBytes: Number(e.target.value) })}
                                className="w-20 accent-wc-accent"
                              />
                              <span className="text-xs text-wc-text-muted w-12 text-right">{(vsConfig.minDeltaBytes / 1024).toFixed(1)}KB</span>
                            </div>
                          </div>
                          <div className="flex items-center justify-between">
                            <div className="flex flex-col">
                              <span className="text-sm text-wc-text-secondary">Audio overlap</span>
                              <span className="text-[10px] text-wc-text-muted">Trailing overlap for word continuity</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <input
                                data-testid="vs-overlap"
                                type="range"
                                min={0}
                                max={16384}
                                step={256}
                                value={vsConfig.overlapBytes}
                                onChange={(e) => handleVsConfigChange({ overlapBytes: Number(e.target.value) })}
                                className="w-20 accent-wc-accent"
                              />
                              <span className="text-xs text-wc-text-muted w-12 text-right">{(vsConfig.overlapBytes / 1024).toFixed(1)}KB</span>
                            </div>
                          </div>
                          <Button
                            data-testid="vs-reset-defaults"
                            variant="ghost"
                            size="sm"
                            className="text-[11px] text-wc-text-faint"
                            onClick={resetVsConfig}
                          >
                            <RotateCcw className="mr-1 h-3 w-3" />
                            Reset to defaults
                          </Button>
                        </>
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>
          </section>

          {/* Section: Voice Output (TTS) */}
          <TtsSettingsSection />

          {/* Section 3: Appearance Defaults */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
              New Pane Defaults
            </h3>
            <div className="rounded-lg border border-wc-default bg-wc-surface-input p-3 space-y-4">
              <HeaderColorPicker
                currentColor={defaultHeaderColor}
                onSelectColor={setDefaultHeaderColor}
                testIdPrefix="defaults"
              />
              <ThemePicker
                currentThemeId={defaultThemeId}
                onSelectTheme={setDefaultThemeId}
                testIdPrefix="defaults"
              />
              <FontSizeStepper
                currentSize={defaultFontSize}
                onChangeSize={setDefaultFontSize}
                testIdPrefix="defaults"
              />
            </div>
          </section>

          {/* Section 4: Integrations */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
              Integrations
            </h3>
            <div className="rounded-lg border border-wc-default bg-wc-surface-input p-3">
              <IntegrationsPanel open={settingsModalOpen} />
            </div>
          </section>
        </div>
      </div>
    </>
  );
}
