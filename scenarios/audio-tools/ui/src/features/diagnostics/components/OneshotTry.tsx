import { useEffect, useRef, useState } from "react";
import { Loader2, Mic, Square } from "lucide-react";
import { MicReadinessIndicator } from "../../../audio-integration";
import { Button } from "../../../components/ui/button";
import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { transcribe, type ProviderTrace } from "../../../services/diagnostics";
import type { ApiError } from "../../../api/client";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import { useMicPermission } from "../useMicPermission";

interface Props {
  onTrace: (t: ProviderTrace) => void;
}

// OneshotTry records a short clip via MediaRecorder and uploads through
// the multipart REST endpoint (the declared multipart_upload exception),
// exercising the unary STT chain.
export function OneshotTry({ onTrace }: Props) {
  const { t } = useTranslation();
  const mediaRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<BlobPart[]>([]);
  const [recording, setRecording] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState("");
  const [error, setError] = useState<ApiError | string | null>(null);
  const micPermission = useMicPermission();

  useEffect(() => () => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
  }, []);

  const upload = async (blob: Blob) => {
    setBusy(true);
    const file = new File([blob], "oneshot.webm", { type: blob.type });
    const r = await transcribe(file);
    setBusy(false);
    if (!r.ok) {
      setError(r.error);
      return;
    }
    setResult(r.data.text);
    onTrace(r.data.trace);
  };

  const start = async () => {
    setError(null);
    setResult("");
    chunksRef.current = [];
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      const mr = new MediaRecorder(stream);
      mr.ondataavailable = (ev) => {
        if (ev.data.size > 0) chunksRef.current.push(ev.data);
      };
      mr.onstop = () => {
        const blob = new Blob(chunksRef.current, { type: mr.mimeType || "audio/webm" });
        streamRef.current?.getTracks().forEach((track) => track.stop());
        streamRef.current = null;
        void upload(blob);
      };
      mr.start();
      mediaRef.current = mr;
      setRecording(true);
    } catch (e) {
      setError((e as Error).message || t(strings.diagnostics.micPermissionDenied));
    }
  };

  const stop = () => {
    mediaRef.current?.stop();
    setRecording(false);
  };

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-app-muted-foreground">{t(strings.diagnostics.oneshotHint)}</p>
      <div className="flex items-center gap-3">
        <Button onClick={() => (recording ? stop() : void start())} disabled={busy} aria-pressed={recording}>
          {recording ? (
            <>
              <Square className="h-4 w-4" aria-hidden="true" />
              {t(strings.diagnostics.oneshotStopUpload)}
            </>
          ) : busy ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              {t(strings.diagnostics.oneshotUploading)}
            </>
          ) : (
            <>
              <Mic className="h-4 w-4" aria-hidden="true" />
              {t(strings.diagnostics.oneshotRecord)}
            </>
          )}
        </Button>
        <MicReadinessIndicator state={micPermission} />
      </div>
      {error ? (
        typeof error === "string" ? (
          <p className="text-sm text-app-danger">{error}</p>
        ) : (
          <ApiErrorState error={error} />
        )
      ) : null}
      {result ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.diagnostics.finalLabel)}</p>
          <p className="whitespace-pre-wrap font-mono text-sm">{result}</p>
        </div>
      ) : null}
    </div>
  );
}
