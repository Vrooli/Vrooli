import { useState } from "react";
import { Loader2, Upload } from "lucide-react";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { transcribe, type ProviderTrace } from "../../../services/diagnostics";
import type { ApiError } from "../../../api/client";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

interface Props {
  onTrace: (t: ProviderTrace) => void;
}

// FileTry uploads an audio sample from disk through the multipart REST
// endpoint. The same code path as OneshotTry, but without the
// MediaRecorder ceremony, for offline-recorded samples.
export function FileTry({ onTrace }: Props) {
  const { t } = useTranslation();
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<string>("");
  const [error, setError] = useState<ApiError | null>(null);

  const run = async () => {
    if (!file) return;
    setBusy(true);
    setError(null);
    const r = await transcribe(file);
    setBusy(false);
    if (!r.ok) {
      setError(r.error);
      return;
    }
    setResult(r.data.text);
    onTrace(r.data.trace);
  };

  return (
    <div className="flex flex-col gap-3">
      <Input
        type="file"
        accept="audio/*"
        onChange={(e) => setFile(e.currentTarget.files?.[0] ?? null)}
        aria-label={t(strings.diagnostics.audioFileLabel)}
      />
      <div>
        <Button onClick={() => void run()} disabled={!file || busy}>
          {busy ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : (
            <Upload className="h-4 w-4" aria-hidden="true" />
          )}
          {t(strings.diagnostics.transcribeAction)}
        </Button>
      </div>
      {error ? <ApiErrorState error={error} /> : null}
      {result ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.diagnostics.resultLabel)}
          </p>
          <p className="whitespace-pre-wrap font-mono text-sm">{result}</p>
        </div>
      ) : null}
    </div>
  );
}
