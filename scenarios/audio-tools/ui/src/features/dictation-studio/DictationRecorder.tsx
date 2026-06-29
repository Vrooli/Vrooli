import { useEffect, useRef, useState } from "react";
import { Mic, Square } from "lucide-react";

import { VoiceStreamProvider, MicReadinessIndicator } from "../../audio-integration";
import { Button } from "../../components/ui/button";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { useMicPermission } from "../diagnostics/useMicPermission";
import { extractPcm16FromWav, pcm16DurationMs } from "./audioWav";

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

// DictationRecorder reuses VoiceStreamProvider — the same WebSocket capture
// path the diagnostics LiveTry uses — to record one turn, surface the batch
// transcript, and hand the retained PCM (via getLastTurnAudio) up for
// corpus storage. It never touches MediaRecorder directly.
export function DictationRecorder({ onCaptured }: Props) {
  const { t } = useTranslation();
  const providerRef = useRef<VoiceStreamProvider | null>(null);
  const [recording, setRecording] = useState(false);
  const [partial, setPartial] = useState("");
  const [finalText, setFinalText] = useState("");
  const [error, setError] = useState("");
  const [captureMissing, setCaptureMissing] = useState(false);
  const [capturedSeconds, setCapturedSeconds] = useState<number | null>(null);
  const micPermission = useMicPermission();

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
      onCaptured({ audio: pcm, durationMs, sampleRateHz, transcript: text });
    } catch {
      setCaptureMissing(true);
    }
  };

  function ensureProvider(): VoiceStreamProvider {
    if (providerRef.current === null) {
      const p = new VoiceStreamProvider();
      p.onResult = (text) => {
        setPartial("");
        setFinalText(text);
        setRecording(false);
        void captureTurn(text);
      };
      p.onError = (msg) => {
        setError(msg);
        setRecording(false);
      };
      p.onPartial = (text) => setPartial(text);
      providerRef.current = p;
    }
    return providerRef.current;
  }

  useEffect(() => {
    return () => {
      providerRef.current?.dispose();
      providerRef.current = null;
    };
  }, []);

  const start = async () => {
    setError("");
    setPartial("");
    setFinalText("");
    setCaptureMissing(false);
    setCapturedSeconds(null);
    try {
      await ensureProvider().start();
      setRecording(true);
    } catch (e) {
      setError((e as Error).message || t(strings.dictationStudio.saveError));
    }
  };

  const stop = () => {
    providerRef.current?.stop();
  };

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.recordHint)}</p>
      <div className="flex items-center gap-3">
        <Button
          type="button"
          data-testid={selectors.dictationStudio.recordStart}
          onClick={() => (recording ? stop() : void start())}
          aria-pressed={recording}
        >
          {recording ? (
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
        <MicReadinessIndicator state={micPermission} />
      </div>

      {recording ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.recording)}</p>
      ) : null}
      {error ? <p className="text-sm text-app-danger">{error}</p> : null}
      {captureMissing ? (
        <p className="text-sm text-app-warning">{t(strings.dictationStudio.captureMissing)}</p>
      ) : null}
      {capturedSeconds !== null ? (
        <p className="text-xs text-app-success">
          {t(strings.dictationStudio.captureRetained, { seconds: capturedSeconds })}
        </p>
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
            {t(strings.dictationStudio.transcribing)}
          </p>
          <p className="whitespace-pre-wrap font-mono text-sm">{finalText}</p>
        </div>
      ) : null}
    </div>
  );
}
