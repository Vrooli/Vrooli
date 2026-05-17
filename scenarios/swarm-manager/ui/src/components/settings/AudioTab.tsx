// AudioTab — surfaces audio-tools-owned config (voice / speed / summarize)
// alongside swarm-manager-local prefs (auto-speak), plus discovery /
// mic-readiness diagnostics.
//
// Voice + speed + summarize are persisted by audio-tools' own settings
// API (shared across adopters). The auto-speak toggle is persisted
// scenario-locally in useAudioPrefs (localStorage); see comment in that
// hook for why we don't add a Settings proto field today.

import { useEffect, useState } from "react";
import type { MicReadinessIndicatorProps } from "../../audio-integration";
import { Loader2 } from "lucide-react";
import {
  MicReadinessIndicator,
  getTTSConfig,
  updateTTSConfig,
  getTTSVoices,
  getTTSSummarizeConfig,
  updateTTSSummarizeConfig,
  useAudioToolsUnavailableReason,
  type TTSConfig,
  type TTSVoiceInfo,
  type TTSSummarizeConfig,
} from "../../audio-integration";
import { useAudioPrefs } from "../../hooks/useAudioPrefs";
import { selectors } from "../../consts/selectors";

interface AudioTabProps {
  testId?: string;
}

export function AudioTab({ testId }: AudioTabProps) {
  const unavailableReason = useAudioToolsUnavailableReason();
  const unavailable = Boolean(unavailableReason);
  const [audioPrefs, setAudioPrefs] = useAudioPrefs();

  const [voices, setVoices] = useState<TTSVoiceInfo[]>([]);
  const [ttsConfig, setTtsConfig] = useState<TTSConfig | null>(null);
  const [summarizeConfig, setSummarizeConfig] = useState<TTSSummarizeConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [micState, setMicState] = useState<MicReadinessIndicatorProps["state"]>("unknown");

  useEffect(() => {
    if (typeof navigator === "undefined" || !navigator.permissions) return;
    let cancelled = false;
    navigator.permissions
      .query({ name: "microphone" as PermissionName })
      .then((status) => {
        if (cancelled) return;
        const mapped: MicReadinessIndicatorProps["state"] =
          status.state === "granted" ? "granted" : status.state === "denied" ? "denied" : "prompt";
        setMicState(mapped);
        status.onchange = () => {
          if (cancelled) return;
          const s: MicReadinessIndicatorProps["state"] =
            status.state === "granted" ? "granted" : status.state === "denied" ? "denied" : "prompt";
          setMicState(s);
        };
      })
      .catch(() => { /* permissions API unavailable; leave unknown */ });
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    if (unavailable) return;
    let cancelled = false;
    setLoading(true);
    Promise.all([getTTSVoices(), getTTSConfig(), getTTSSummarizeConfig()])
      .then(([v, c, s]) => {
        if (cancelled) return;
        setVoices(v);
        setTtsConfig(c);
        setSummarizeConfig(s);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [unavailable]);

  const saveTtsConfig = async (patch: Partial<TTSConfig>) => {
    try {
      const updated = await updateTTSConfig(patch);
      setTtsConfig(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const saveSummarizeConfig = async (patch: Partial<TTSSummarizeConfig>) => {
    try {
      const updated = await updateTTSSummarizeConfig(patch);
      setSummarizeConfig(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <div className="space-y-6" data-testid={testId}>
      {unavailable && (
        <div
          className="rounded border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-300"
          data-testid={selectors.settings.audioUnavailableBanner}
        >
          Audio features are unavailable: <code className="rounded bg-amber-900/40 px-1">{unavailableReason}</code>. Voice input,
          agent speech, and TTS settings will be re-enabled when audio-tools is reachable.
        </div>
      )}

      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-slate-300">Microphone</h3>
        <MicReadinessIndicator state={micState} />
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-slate-300">Auto-speak agent replies</h3>
        <label className="flex items-center gap-3 text-sm text-slate-300">
          <input
            type="checkbox"
            checked={audioPrefs.autoSpeak}
            disabled={unavailable}
            onChange={(e) => setAudioPrefs({ autoSpeak: e.target.checked })}
            data-testid={selectors.settings.audioAutoSpeak}
          />
          Automatically speak new assistant messages
        </label>
        <p className="text-xs text-slate-500">
          When enabled, new assistant messages in Session Details are read aloud as they arrive. Click the speaker icon on any
          message to start or stop manually.
        </p>
      </section>

      {loading && (
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <Loader2 className="h-4 w-4 animate-spin" /> Loading audio-tools configuration...
        </div>
      )}

      {error && (
        <div className="rounded border border-red-500/40 bg-red-500/10 p-3 text-sm text-red-300">
          {error}
        </div>
      )}

      {!unavailable && ttsConfig && (
        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-slate-300">Voice</h3>
          <label className="flex flex-col gap-1 text-sm text-slate-300">
            Default voice
            <select
              value={ttsConfig.defaultVoice}
              onChange={(e) => void saveTtsConfig({ defaultVoice: e.target.value })}
              className="rounded border border-slate-700 bg-slate-900 px-2 py-1 text-sm text-slate-200"
              data-testid={selectors.settings.audioVoice}
            >
              {voices.length === 0 && <option value="">(no voices reported)</option>}
              {voices.map((v) => (
                <option key={v.id} value={v.id}>{v.name || v.id}</option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm text-slate-300">
            Speed: {ttsConfig.defaultSpeed.toFixed(2)}×
            <input
              type="range"
              min={0.5}
              max={2.0}
              step={0.05}
              value={ttsConfig.defaultSpeed}
              onChange={(e) => void saveTtsConfig({ defaultSpeed: Number(e.target.value) })}
              data-testid={selectors.settings.audioSpeed}
            />
          </label>
        </section>
      )}

      {!unavailable && summarizeConfig && (
        <section className="space-y-3" data-testid={selectors.settings.audioSummarize}>
          <h3 className="text-sm font-semibold text-slate-300">Summarize before speaking</h3>
          <label className="flex items-center gap-3 text-sm text-slate-300">
            <input
              type="checkbox"
              checked={summarizeConfig.enabled}
              onChange={(e) => void saveSummarizeConfig({ enabled: e.target.checked })}
            />
            Summarize long replies before TTS
          </label>
          <label className="flex flex-col gap-1 text-sm text-slate-300">
            Level
            <select
              value={summarizeConfig.level}
              onChange={(e) => void saveSummarizeConfig({ level: e.target.value as TTSSummarizeConfig["level"] })}
              className="rounded border border-slate-700 bg-slate-900 px-2 py-1 text-sm text-slate-200"
            >
              <option value="light">Light</option>
              <option value="moderate">Moderate</option>
              <option value="heavy">Heavy</option>
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm text-slate-300">
            Character threshold
            <input
              type="number"
              min={100}
              step={100}
              value={summarizeConfig.charThreshold}
              onChange={(e) => void saveSummarizeConfig({ charThreshold: Number(e.target.value) })}
              className="w-32 rounded border border-slate-700 bg-slate-900 px-2 py-1 text-sm text-slate-200"
            />
          </label>
        </section>
      )}
    </div>
  );
}
