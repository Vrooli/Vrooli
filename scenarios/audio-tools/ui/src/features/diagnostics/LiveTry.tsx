import { useEffect, useRef, useState } from "react";
import { Mic, Square } from "lucide-react";
import { VoiceStreamProvider, MicReadinessIndicator } from "../../audio-integration";
import { Button } from "../../components/ui/button";
import type { ProviderTrace } from "../../services/diagnostics";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { useMicPermission } from "./useMicPermission";

interface Props {
  onTrace: (t: ProviderTrace) => void;
}

// LiveTry streams audio via WebSocket through VoiceStreamProvider — the
// same path consumer scenarios adopt by copying audio-integration/. The lazy
// construction is deliberate: SSR and the first render must never touch
// MediaRecorder / AudioContext.
export function LiveTry({ onTrace }: Props) {
  const { t } = useTranslation();
  const providerRef = useRef<VoiceStreamProvider | null>(null);
  const [recording, setRecording] = useState(false);
  const [partial, setPartial] = useState("");
  const [finalText, setFinalText] = useState("");
  const [error, setError] = useState<string>("");
  const micPermission = useMicPermission();

  function ensureProvider(): VoiceStreamProvider {
    if (providerRef.current === null) {
      const p = new VoiceStreamProvider();
      p.onResult = (text) => {
        setPartial("");
        setFinalText(text);
        setRecording(false);
        // VoiceStreamProvider does not surface a typed ProviderTrace event
        // in the current proto shape; emit a placeholder so the trace card
        // still reflects that a live run completed.
        onTrace({ providerTier: "local", providerId: "voice-stream", modelId: "whisper", latencyMs: 0 });
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
    try {
      await ensureProvider().start();
      setRecording(true);
    } catch (e) {
      setError((e as Error).message || t(strings.diagnostics.liveStartFailed));
    }
  };

  const stop = () => {
    providerRef.current?.stop();
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <Button onClick={() => (recording ? stop() : void start())} aria-pressed={recording}>
          {recording ? (
            <>
              <Square className="h-4 w-4" aria-hidden="true" />
              {t(strings.diagnostics.liveStop)}
            </>
          ) : (
            <>
              <Mic className="h-4 w-4" aria-hidden="true" />
              {t(strings.diagnostics.liveStart)}
            </>
          )}
        </Button>
        <MicReadinessIndicator state={micPermission} />
      </div>
      {error ? <p className="text-sm text-app-danger">{error}</p> : null}
      {partial ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted/60 p-3 text-sm italic opacity-80">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.diagnostics.partialLabel)}</p>
          <p className="whitespace-pre-wrap">{partial}</p>
        </div>
      ) : null}
      {finalText ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.diagnostics.finalLabel)}</p>
          <p className="whitespace-pre-wrap font-mono text-sm">{finalText}</p>
        </div>
      ) : null}
    </div>
  );
}
