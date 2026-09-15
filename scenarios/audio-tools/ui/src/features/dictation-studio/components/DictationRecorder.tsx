import { useEffect, useRef, useState } from "react";
import { RotateCcw, X } from "lucide-react";

import { PcmVoiceStreamProvider, MicReadinessIndicator } from "../../../audio-integration";
import { VoiceInputButton } from "@vrooli/react-component-library/VoiceInputButton/4";
import { Button } from "../../../components/ui/button";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { useMicPermission } from "../../diagnostics/useMicPermission";
import { extractPcm16FromWav, pcm16DurationMs } from "../audioWav";
import { joinTranscriptText, type StreamTurnDiagnostic } from "@vrooli/audio-capture-browser";

type VirtualCaptureShape = "burst" | "chunked";

function virtualCaptureConfig(): { url: string; shape: VirtualCaptureShape; targetSamples: number } | null {
  if (typeof window === "undefined") return null;
  const params = new URLSearchParams(window.location.search);
  if (params.get("stt_test_mode") !== "1" || params.get("stt_capture_source") !== "virtual") return null;
  const url = params.get("stt_corpus_url");
  if (!url) return null;
  const shape = params.get("stt_capture_shape") === "chunked" ? "chunked" : "burst";
  const targetSamples = Number(params.get("stt_virtual_samples") ?? "0");
  if (!Number.isSafeInteger(targetSamples) || targetSamples <= 0) return null;
  return { url, shape, targetSamples };
}

function emptyMediaStream(): MediaStream {
  return { getTracks: () => [] } as unknown as MediaStream;
}

async function createVirtualCapture(
  config: { url: string; shape: VirtualCaptureShape; targetSamples: number },
  onFrame: (samples: Float32Array, sampleRate: number) => void,
): Promise<{ stop(): void }> {
  const response = await fetch(config.url, { credentials: "same-origin" });
  if (!response.ok) throw new Error(`virtual corpus request failed (${response.status})`);
  const { pcm, sampleRateHz } = extractPcm16FromWav(await response.arrayBuffer());
  if (pcm.byteLength === 0 || sampleRateHz <= 0) throw new Error("virtual corpus clip is empty");
  const view = new DataView(pcm.buffer, pcm.byteOffset, pcm.byteLength);
  const baseSamples = pcm.byteLength / 2;
  const frameSamples = config.shape === "chunked" ? 1_600 : baseSamples;
  let stopped = false;
  let offset = 0;
  const pump = () => {
    if (stopped) return;
    // Yield between bounded batches. The provider must be able to open the
    // WebSocket and receive processed acknowledgements while a large
    // accelerated turn is being journaled; generating the whole turn in one
    // synchronous callback queues every IndexedDB append ahead of compaction.
    for (let batch = 0; batch < 256 && offset < config.targetSamples; batch += 1) {
      const length = Math.min(frameSamples, config.targetSamples - offset);
      const samples = new Float32Array(length);
      for (let index = 0; index < length; index += 1) {
        samples[index] = view.getInt16(((offset + index) % baseSamples) * 2, true) / 32_768;
      }
      onFrame(samples, sampleRateHz);
      offset += length;
    }
    if (offset < config.targetSamples) setTimeout(pump, 0);
  };
  setTimeout(pump, 0);
  return { stop: () => { stopped = true; } };
}

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
  /** Receives streamed text even when the bounded replay copy is unavailable. */
  onTranscript?: (text: string) => void;
}

type RecorderState = "idle" | "preparing" | "recording" | "transcribing" | "captured" | "failed" | "cancelled";

// DictationRecorder reuses PcmVoiceStreamProvider — the same WebSocket capture
// path the diagnostics LiveTry uses — to record one turn, surface the batch
// transcript, and hand the retained PCM (via getLastTurnAudio) up for
// corpus storage. It never touches MediaRecorder directly.
export function DictationRecorder({ onCaptured, onTranscript }: Props) {
  const { t } = useTranslation();
  const providerRef = useRef<PcmVoiceStreamProvider | null>(null);
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
  const committedTextRef = useRef("");
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
    if (!stream || !AudioContextCtor || virtualCaptureConfig()) return;

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
      // A bounded whole-turn replay copy is deliberately not retained forever.
      // The stream may still have delivered a complete transcript, so do not
      // leave the recorder in "transcribing" with no recoverable outcome.
      setState("failed");
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

  function ensureProvider(): PcmVoiceStreamProvider {
    if (providerRef.current === null) {
      const virtual = virtualCaptureConfig();
      const p = new PcmVoiceStreamProvider(virtual ? {
        getUserMedia: async () => emptyMediaStream(),
        captureFactory: (_stream, onFrame) => createVirtualCapture(virtual, onFrame),
        // The accelerated source is not a realtime/device claim. Preserve
        // exact sample-range accounting while reducing browser persistence
        // overhead from 100 ms to 1 s wire batches. Real microphone and BAS
        // fake-media runs retain the package's production default.
        wireBatchSamples: 16_000,
      } : undefined);
      p.onResult = (text) => {
        setPartial("");
        // Passthrough providers may commit durable segments before sending
        // the terminal envelope. The envelope's final text is intentionally
        // empty in that case; preserve the committed composer text instead of
        // replacing it with an empty string.
        if (text.trim()) {
          committedTextRef.current = text;
          setFinalText(text);
        } else {
          setFinalText(committedTextRef.current);
        }
        stopLevelMonitor();
        const transcript = text.trim() || committedTextRef.current;
        onTranscript?.(transcript);
        void captureTurn(transcript);
      };
      p.onError = (msg) => {
        setError(msg);
        setState("failed");
        stopLevelMonitor();
      };
      p.onPartial = (text) => setPartial(text);
      p.onSegmentFinal = (text) => {
        committedTextRef.current = joinTranscriptText(committedTextRef.current, text);
        setFinalText(committedTextRef.current);
      };

      p.onStatus = ({ code, message }) => {
        // Processed coverage is a durability signal consumed by the
        // diagnostic ledger. It must not replace the user-facing connection
        // state while a deterministic or real microphone turn is running.
        if (code !== "processed_acknowledgement") setStreamStatus(message);
      };
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
    setError("");
    // Surface the attempt immediately. Some backends emit their first status
    // only after media negotiation, but the UI must not look inert while that
    // negotiation is in progress.
    setStreamStatus("Connecting to speech stream…");
    setPartial("");
    setFinalText("");
    committedTextRef.current = "";
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
        <VoiceInputButton
          data-testid={selectors.dictationStudio.recordStart}
          state={state === "recording" ? "recording" : state === "preparing" ? "preparing" : state === "transcribing" ? "transcribing" : state === "failed" ? "error" : "idle"}
          level={audioLevel}
          aria-label={state === "recording" ? t(strings.dictationStudio.recordStop) : t(strings.dictationStudio.recordStart)}
          disabled={state === "preparing" || state === "transcribing"}
          onStart={() => void start()}
          onStop={stop}
        />
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

      <div className="flex flex-col gap-1" data-testid={selectors.dictationStudio.recordState} data-recorder-state={state}>
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
            data-retained-bytes={String(diagnostic.retainedBytes)}
            data-first-partial-latency-ms={diagnostic.firstPartialLatencyMs === null ? "" : String(diagnostic.firstPartialLatencyMs)}
            data-committed-text-lag-ms={diagnostic.committedTextLagMs === null ? "" : String(diagnostic.committedTextLagMs)}
          />
          {diagnostic.state !== "preparing" && diagnostic.state !== "recording" && diagnostic.capturedSequence >= 0 && diagnostic.processedSequence + 1 >= diagnostic.capturedSequence ? <span className="sr-only" data-testid={selectors.dictationStudio.turnProcessedReady} /> : null}
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
          <p className="whitespace-pre-wrap" data-testid={selectors.dictationStudio.interimTranscript}>{partial}</p>
        </div>
      ) : null}
      {finalText ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.dictationStudio.finalLabel)}
          </p>
          <p className="whitespace-pre-wrap font-mono text-sm" data-testid={selectors.dictationStudio.finalTranscript}>{finalText}</p>
        </div>
      ) : null}
    </div>
  );
}
