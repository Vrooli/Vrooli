import { useCallback, useEffect, useRef, useState } from "react";
import {
  AlertCircle,
  CheckCircle,
  ChevronDown,
  ChevronRight,
  Circle,
  Keyboard,
  Mic,
  RefreshCw,
  RotateCcw,
} from "lucide-react";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { fetchCapabilities, getVoiceStreamConfig, toErrorInfo, type CapabilityState, type VoiceStreamConfig, updateVoiceStreamConfig } from "../../lib/api";
import { formatShortcutFromEvent } from "../../lib/shortcutParser";
import { Button } from "../ui/button";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";

export default function VoiceInputSection() {
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

  const [recordingShortcut, setRecordingShortcut] = useState(false);
  const [voiceCaps, setVoiceCaps] = useState<CapabilityState[]>([]);
  const [voiceCapsLoading, setVoiceCapsLoading] = useState(false);
  const [voiceCapsError, setVoiceCapsError] = useState<string | null>(null);
  const [micPermission, setMicPermission] = useState<"granted" | "denied" | "prompt" | "unknown">("unknown");
  const [micRequesting, setMicRequesting] = useState(false);
  const [vsConfig, setVsConfig] = useState<VoiceStreamConfig | null>(null);
  const [vsConfigLoading, setVsConfigLoading] = useState(false);
  const [vsConfigError, setVsConfigError] = useState<string | null>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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
  }, []);

  useEffect(() => {
    if (!advancedOpen) return;
    const signal = { cancelled: false };
    void loadVoiceStreamConfig(signal);
    return () => {
      signal.cancelled = true;
    };
  }, [advancedOpen, loadVoiceStreamConfig]);

  useEffect(() => {
    return () => {
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
    };
  }, []);

  const handleVsConfigChange = useCallback((patch: Partial<VoiceStreamConfig>) => {
    setVsConfig((current) => (current ? { ...current, ...patch } : null));
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
  }, []);

  const resetVsConfig = useCallback(async () => {
    try {
      const updated = await updateVoiceStreamConfig({
        flushIntervalMs: 500,
        minDeltaBytes: 4096,
        overlapBytes: 2048,
      });
      setVsConfig(updated);
      setVsConfigError(null);
    } catch (error) {
      setVsConfigError(toErrorInfo(error).message);
    }
  }, []);

  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow="Speech Recognition"
        title="Voice input"
        description="Control how voice capture works, validate available backends, and tune streaming for responsiveness."
      />

      <SettingsCard className="space-y-4">
        <SettingsRow
          label="Voice input"
          hint="Enable speech-to-text controls throughout the workspace."
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
              label="Auto-stop on silence"
              hint="Stop recording automatically when speech ends in tap mode."
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
                label="Silence timeout"
                hint="How long to wait after speech stops."
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
                    <span className="w-9 text-right text-xs text-wc-text-muted">
                      {(vadSilenceTimeoutMs / 1000).toFixed(1)}s
                    </span>
                  </div>
                )}
              />
            )}

            <SettingsRow
              label="Language"
              hint="Use auto-detect unless recognition quality is better with a fixed language."
              control={(
                <select
                  data-testid="voice-language-select"
                  className="rounded-lg border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary"
                  value={voiceLanguage}
                  onChange={(event) => setVoiceLanguage(event.target.value)}
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
              )}
            />
          </>
        )}

        <SettingsRow
          label="Shortcut"
          hint="Capture a keyboard combo to start voice input quickly."
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
                  Press a key combo...
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
                title="Change shortcut"
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
            <div className="text-sm font-medium text-wc-text-secondary">Backend availability</div>
            <div className="text-[11px] text-wc-text-muted">
              Confirm which recognition engines are ready in the current environment.
            </div>
          </div>
          <Button
            data-testid="voice-caps-refresh"
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => void loadVoiceCaps()}
            title="Refresh"
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
              Web Speech API
            </span>
            <span className={hasWebSpeech ? "ml-auto text-green-400" : "ml-auto text-wc-text-faint"}>
              {hasWebSpeech ? "available" : "unavailable"}
            </span>
          </div>
        </div>

        {!voiceCapsLoading && voiceCaps.every((capability) => capability.status !== "available") && !hasWebSpeech && (
          <p className="text-[11px] text-amber-400">
            No transcription backend available. Install Whisper or use a Chromium-based browser for Web Speech API.
          </p>
        )}
      </SettingsCard>

      <SettingsCard className="space-y-3">
        <div>
          <div className="text-sm font-medium text-wc-text-secondary">Microphone access</div>
          <div className="text-[11px] text-wc-text-muted">
            The browser must allow microphone access before voice recording can start.
          </div>
        </div>

        <div className="flex items-center gap-2 text-xs">
          {micPermission === "granted" ? (
            <>
              <CheckCircle className="h-3.5 w-3.5 shrink-0 text-green-400" />
              <span className="text-wc-text-secondary">Permission granted</span>
            </>
          ) : micPermission === "denied" ? (
            <>
              <AlertCircle className="h-3.5 w-3.5 shrink-0 text-red-400" />
              <span className="text-red-400">Permission denied</span>
            </>
          ) : micPermission === "prompt" ? (
            <>
              <Circle className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" />
              <span className="text-wc-text-faint">Not yet requested</span>
            </>
          ) : (
            <>
              <Circle className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" />
              <span className="text-wc-text-faint">Unknown</span>
            </>
          )}
        </div>

        {micPermission === "denied" && (
          <p className="text-[11px] text-wc-text-faint">
            Microphone access was blocked. Click the lock or site-settings icon in your browser&apos;s address bar to allow microphone access, then refresh.
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
            <Mic className="mr-1 h-3.5 w-3.5" />
            {micRequesting ? "Requesting..." : "Allow microphone"}
          </Button>
        )}
      </SettingsCard>

      {voiceEnabled && (
        <SettingsCard className="space-y-3">
          <button
            data-testid="advanced-streaming-toggle"
            className="flex w-full items-center gap-1 text-left text-[11px] font-semibold uppercase tracking-[0.22em] text-wc-text-muted"
            onClick={() => setAdvancedOpen(!advancedOpen)}
          >
            {advancedOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
            Advanced streaming
          </button>

          {advancedOpen && (
            <div className="space-y-4">
              {vsConfigLoading && (
                <div className="py-2 text-center text-xs text-wc-text-faint">Loading...</div>
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
                    label="Flush interval"
                    hint="How often audio is sent to Whisper."
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
                        <span className="w-12 text-right text-xs text-wc-text-muted">{vsConfig.flushIntervalMs}ms</span>
                      </div>
                    )}
                  />

                  <SettingsRow
                    label="Min chunk size"
                    hint="Minimum buffered audio before a partial transcript fires."
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
                        <span className="w-12 text-right text-xs text-wc-text-muted">
                          {(vsConfig.minDeltaBytes / 1024).toFixed(1)}KB
                        </span>
                      </div>
                    )}
                  />

                  <SettingsRow
                    label="Audio overlap"
                    hint="Keep a small trailing overlap so word boundaries survive chunking."
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
                        <span className="w-12 text-right text-xs text-wc-text-muted">
                          {(vsConfig.overlapBytes / 1024).toFixed(1)}KB
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
                    <RotateCcw className="mr-1 h-3.5 w-3.5" />
                    Reset to defaults
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
