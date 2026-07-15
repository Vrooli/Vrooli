import { useEffect, useRef, useState } from "react";
import { Mic, RotateCcw, Square, X } from "lucide-react";

import { VoiceStreamProvider, MicReadinessIndicator } from "../../audio-integration";
import { Button } from "../../components/ui/button";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { useMicPermission } from "../diagnostics/useMicPermission";
import { extractPcm16FromWav, pcm16DurationMs } from "./audioWav";
import type { StreamTurnDiagnostic } from "@vrooli/audio-capture-browser";
import { getSharedAudioContext } from "../../audio-integration/hooks/voice/sharedAudioContext";

export interface CapturedClip {
  /** raw signed-16-bit little-endian PCM */
  audio: Uint8Array;
  durationMs: number;
  sampleRateHz: number;
  /** the streamed batch transcript for this turn */
  transcript: string;
}

interface Props {
  onCaptured: (clip: CapturedClip) => void;
}

type RecorderState = "idle" | "preparing" | "recording" | "transcribing" | "captured" | "failed" | "cancelled";

// DictationRecorder reuses VoiceStreamProvider — the same WebSocket capture
// path the diagnostics LiveTry uses — to record one turn, surface the batch
// transcript, and hand the retained PCM (via getLastTurnAudio) up for
// corpus storage. It never touches MediaRecorder directly.
export function DictationRecorder({ onCaptured }: Props) {
  const { t } = useTranslation();
  const providerRef = useRef<VoiceStreamProvider | null>(null);
  const rafRef = useRef<number>(0);
  const audioNodesRef = useRef<AudioNode[]>([]);
  const audioLevelRef = useRef(0);
  const noAudioTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [state, setState] = useState<RecorderState>("idle");
  const [audioLevel, setAudioLevel] = useState(0);
  const [partial, setPartial] = useState("");
  const [finalText, setFinalText] = useState("");
  const [error, setError] = useState("");
  const [streamStatus, setStreamStatus] = useState("");
  const [captureMissing, setCaptureMissing] = useState(false);
  const [capturedSeconds, setCapturedSeconds] = useState<number | null>(null);
  const [noAudioDetected, setNoAudioDetected] = useState(false);
  const [diagnostic, setDiagnostic] = useState<StreamTurnDiagnostic | null>(null);
  const micPermission = useMicPermission();

  const stopLevelMonitor = () => {
    cancelAnimationFrame(rafRef.current);
    rafRef.current = 0;
    audioLevelRef.current = 0;
    setAudioLevel(0);
    for (const node of audioNodesRef.current) {
      try {
        node.disconnect();
      } catch {
        // Already disconnected.
      }
    }
    audioNodesRef.current = [];
    if (noAudioTimerRef.current) {
      clearTimeout(noAudioTimerRef.current);
      noAudioTimerRef.current = null;
    }
  };

  const startLevelMonitor = (stream: MediaStream | null) => {
    stopLevelMonitor();
    const win = window as Window & {
      AudioContext?: typeof AudioContext;
      webkitAudioContext?: typeof AudioContext;
    };
    const AudioContextCtor = win.AudioContext ?? win.webkitAudioContext;
    if (!stream || !AudioContextCtor) return;

    try {
      const ctx = new AudioContextCtor();
      void ctx.resume().catch(() => undefined);
      const source = ctx.createMediaStreamSource(stream);
      const analyser = ctx.createAnalyser();
      analyser.fftSize = 512;
      source.connect(analyser);
      audioNodesRef.current = [source, analyser];
      const data = new Uint8Array(analyser.frequencyBinCount);

      const tick = () => {
        analyser.getByteTimeDomainData(data);
        let sum = 0;
        for (const point of data) {
          const normalized = (point - 128) / 128;
          sum += normalized * normalized;
        }
        const nextLevel = Math.min(1, Math.sqrt(sum / data.length) * 4);
        audioLevelRef.current = nextLevel;
        setAudioLevel(nextLevel);
        rafRef.current = requestAnimationFrame(tick);
      };
      rafRef.current = requestAnimationFrame(tick);
      noAudioTimerRef.current = setTimeout(() => {
        if (audioLevelRef.current < 0.01) {
          setNoAudioDetected(true);
        }
      }, 2_000);
    } catch {
      stopLevelMonitor();
    }
  };

  const captureTurn = async (text: string) => {
    const provider = providerRef.current;
    const last = provider?.getLastTurnAudio() ?? null;
    if (!last) {
      setCaptureMissing(true);
      return;
    }
    try {
      const buf = await last.blob.arrayBuffer();
      const { pcm, sampleRateHz } = extractPcm16FromWav(buf);
      const durationMs = last.durationMs > 0 ? last.durationMs : pcm16DurationMs(pcm.byteLength, sampleRateHz);
      setCapturedSeconds(Math.round(durationMs / 100) / 10);
      setCaptureMissing(false);
      setState("captured");
      onCaptured({ audio: pcm, durationMs, sampleRateHz, transcript: text });
    } catch {
      setCaptureMissing(true);
      setState("failed");
    }
  };

  function ensureProvider(): VoiceStreamProvider {
    if (providerRef.current === null) {
      const p = new VoiceStreamProvider();
      p.onResult = (text) => {
        setPartial("");
        setFinalText(text);
        stopLevelMonitor();
        void captureTurn(text);
      };
      p.onError = (msg) => {
        setError(msg);
        setState("failed");
        stopLevelMonitor();
      };
      p.onPartial = (text) => setPartial(text);

      p.onStatus = ({ message }) => setStreamStatus(message);
      p.onDiagnostic = (next) => setDiagnostic(next);
      providerRef.current = p;
    }
    return providerRef.current;
  }

  useEffect(() => {
    return () => {
      providerRef.current?.dispose();
      providerRef.current = null;
      stopLevelMonitor();
    };
  }, []);

  const start = async () => {
    if (state === "preparing" || state === "recording" || state === "transcribing") return;
    // Dictation Studio does not use useVoiceCore, which normally resumes this
    // context in the initiating gesture. Do it before the first await so the
    // microphone graph is renderable in browsers that suspend new contexts.
    try {
      const context = getSharedAudioContext();
      if (context.state !== "running") void context.resume().catch(() => undefined);
    } catch {
      // Capture startup will surface an actionable error if Web Audio is unavailable.
    }
    setError("");
    setStreamStatus("");
    setPartial("");
    setFinalText("");
    setCaptureMissing(false);
    setCapturedSeconds(null);
    setNoAudioDetected(false);
    setDiagnostic(null);
    setState("preparing");
    try {
      const provider = ensureProvider();
      await provider.start();
      startLevelMonitor(provider.getStream());
      setState("recording");
    } catch (e) {
      setError((e as Error).message || t(strings.dictationStudio.saveError));
      setState("failed");
      stopLevelMonitor();
    }
  };

  const stop = () => {
    if (state !== "recording") return;
    setState("transcribing");
    stopLevelMonitor();
    providerRef.current?.stop();
  };

  const cancel = () => {
    if (state !== "transcribing") return;
    providerRef.current?.dispose();
    providerRef.current = null;
    setPartial("");
    setState("cancelled");
    stopLevelMonitor();
  };

  const active = state === "preparing" || state === "recording" || state === "transcribing";
  const exportDiagnostic = () => {
    const exported = providerRef.current?.exportDiagnostic();
    if (!exported || typeof document === "undefined") return;
    const blob = new Blob([exported], { type: "application/json" });
    const href = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = href;
    link.download = "dictation-turn-diagnostic.json";
    link.click();
    URL.revokeObjectURL(href);
  };
  const stateLabel =
    state === "preparing"
      ? t(strings.dictationStudio.preparing)
      : state === "recording"
        ? t(strings.dictationStudio.recording)
        : state === "transcribing"
          ? t(strings.dictationStudio.transcribing)
          : state === "captured"
            ? t(strings.dictationStudio.captured)
            : state === "cancelled"
              ? t(strings.dictationStudio.cancelled)
              : state === "failed"
                ? t(strings.dictationStudio.failed)
                : t(strings.dictationStudio.ready);

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.recordHint)}</p>
      <div className="flex items-center gap-3">
        <Button
          type="button"
          data-testid={selectors.dictationStudio.recordStart}
          onClick={() => (state === "recording" ? stop() : void start())}
          aria-pressed={active}
          disabled={state === "preparing" || state === "transcribing"}
        >
          {state === "recording" ? (
            <>
              <Square className="h-4 w-4" aria-hidden="true" />
              {t(strings.dictationStudio.recordStop)}
            </>
          ) : (
            <>
              <Mic className="h-4 w-4" aria-hidden="true" />
              {t(strings.dictationStudio.recordStart)}
            </>
          )}
        </Button>
        {state === "transcribing" ? (
          <Button
            type="button"
            variant="outline"
            data-testid={selectors.dictationStudio.recordCancel}
            onClick={cancel}
          >
            <X className="h-4 w-4" aria-hidden="true" />
            {t(strings.dictationStudio.cancelTranscription)}
          </Button>
        ) : null}
        {state === "failed" || state === "cancelled" ? (
          <Button type="button" variant="outline" onClick={() => void start()}>
            <RotateCcw className="h-4 w-4" aria-hidden="true" />
            {t(strings.dictationStudio.retryRecording)}
          </Button>
        ) : null}
        <MicReadinessIndicator state={micPermission} />
      </div>

      <div className="flex flex-col gap-1" data-testid={selectors.dictationStudio.recordState}>
        <p className="text-sm text-app-muted-foreground">{stateLabel}</p>
        <div
          className="h-2 w-full overflow-hidden rounded-full bg-app-surface-muted"
          role="meter"
          aria-label={t(strings.dictationStudio.audioLevel)}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.round(audioLevel * 100)}
          data-testid={selectors.dictationStudio.audioMeter}
        >
          <div className="h-full bg-app-primary transition-[width]" style={{ width: `${Math.round(audioLevel * 100)}%` }} />
        </div>
      </div>
      {noAudioDetected && state === "recording" ? (
        <p className="text-sm text-app-warning">{t(strings.dictationStudio.noAudioDetected)}</p>
      ) : null}
      {error ? <p className="text-sm text-app-danger" data-testid={selectors.dictationStudio.recordError}>{error}</p> : null}
      {streamStatus && !error ? <p className="text-sm text-app-muted-foreground" aria-live="polite" data-testid={selectors.dictationStudio.streamStatus}>{streamStatus}</p> : null}
      {captureMissing ? (
        <p className="text-sm text-app-warning">{t(strings.dictationStudio.captureMissing)}</p>
      ) : null}
      {capturedSeconds !== null ? (
        <p className="text-xs text-app-success">
          {t(strings.dictationStudio.captureRetained, { seconds: capturedSeconds })}
        </p>
      ) : null}
      {diagnostic ? (
        <details className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm" data-testid={selectors.dictationStudio.turnDetails}>
          <summary className="cursor-pointer font-medium">{t(strings.dictationStudio.turnDetails)}</summary>
          <span
            className="sr-only"
            data-testid={selectors.dictationStudio.turnCaptureStatus}
            data-has-captured-audio={diagnostic.capturedSequence >= 0 ? "true" : "false"}
          />
          <span
            className="sr-only"
            data-testid={selectors.dictationStudio.turnSentStatus}
            data-has-sent-audio={diagnostic.sentSequence >= 0 ? "true" : "false"}
          />
          <span
            className="sr-only"
            data-testid={selectors.dictationStudio.turnDoneStatus}
            data-done-sent={diagnostic.doneSent ? "true" : "false"}
          />
          {diagnostic.doneSent ? <span className="sr-only" data-testid={selectors.dictationStudio.turnDoneReady} /> : null}
          <span
            className="sr-only"
            data-testid={selectors.dictationStudio.turnProcessedStatus}
            data-has-processed-audio={diagnostic.processedSequence >= 0 ? "true" : "false"}
          />
          {diagnostic.processedSequence >= 0 ? <span className="sr-only" data-testid={selectors.dictationStudio.turnProcessedReady} /> : null}
          <dl className="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-xs">
            <dt className="text-app-muted-foreground">{t(strings.dictationStudio.turnState)}</dt><dd>{diagnostic.state}</dd>
            <dt className="text-app-muted-foreground">{t(strings.dictationStudio.turnDurability)}</dt><dd>{diagnostic.durability}</dd>
            <dt className="text-app-muted-foreground">{t(strings.dictationStudio.turnCapturedChunks)}</dt><dd>{Math.max(0, diagnostic.capturedSequence + 1)}</dd>
            <dt className="text-app-muted-foreground">{t(strings.dictationStudio.turnProcessedChunks)}</dt><dd>{Math.max(0, diagnostic.processedSequence + 1)}</dd>
            <dt className="text-app-muted-foreground">{t(strings.dictationStudio.turnTerminalOutcome)}</dt><dd>{diagnostic.terminalReason ?? t(strings.dictationStudio.turnInProgress)}</dd>
          </dl>
          <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.dictationStudio.turnDiagnosticPrivacy)}</p>
          <Button type="button" variant="outline" className="mt-2" data-testid={selectors.dictationStudio.exportDiagnostic} onClick={exportDiagnostic}>
            {t(strings.dictationStudio.exportDiagnostic)}
          </Button>
        </details>
      ) : null}

      {partial ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted/60 p-3 text-sm italic opacity-80">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.dictationStudio.partialLabel)}
          </p>
          <p className="whitespace-pre-wrap">{partial}</p>
        </div>
      ) : null}
      {finalText ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.dictationStudio.finalLabel)}
          </p>
          <p className="whitespace-pre-wrap font-mono text-sm">{finalText}</p>
        </div>
      ) : null}
    </div>
  );
}
