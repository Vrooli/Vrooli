import { useState } from "react";
import { Loader2, Send } from "lucide-react";
import { Panel } from "../../../components/ui/panel";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Select } from "../../../components/ui/select";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

export function TranscodeTryIt() {
  const { t } = useTranslation();
  const [file, setFile] = useState<File | null>(null);
  const [format, setFormat] = useState("wav");
  const [busy, setBusy] = useState(false);
  const [downloadUrl, setDownloadUrl] = useState<string>("");
  const [error, setError] = useState<string>("");

  const run = async () => {
    if (!file) return;
    setBusy(true);
    setError("");
    if (downloadUrl) URL.revokeObjectURL(downloadUrl);
    const fd = new FormData();
    fd.append("audio", file);
    fd.append("output_format", format);
    const { uploadFile } = await import("../../../api/client");
    const res = await uploadFile("/api/v1/audio/transcode", fd);
    setBusy(false);
    if (!res.ok) {
      setError(t(strings.diagnostics.transcodeFailed, { status: res.status }));
      return;
    }
    const blob = await res.blob();
    setDownloadUrl(URL.createObjectURL(blob));
  };

  return (
    <Panel title={t(strings.diagnostics.transcodeTitle)} description={t(strings.diagnostics.transcodeDescription)}>
      <div className="flex flex-col gap-3">
        <Input type="file" accept="audio/*" onChange={(e) => setFile(e.currentTarget.files?.[0] ?? null)} aria-label={t(strings.diagnostics.audioFileLabel)} />
        <div className="flex flex-wrap items-end gap-3">
          <label htmlFor="transcode-format" className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            {t(strings.diagnostics.transcodeOutputFormatLabel)}
            <Select id="transcode-format" value={format} onChange={(e) => setFormat(e.currentTarget.value)} className="w-32">
              <option value="wav">{t(strings.diagnostics.formatWav)}</option>
              <option value="mp3">{t(strings.diagnostics.formatMp3)}</option>
              <option value="flac">{t(strings.diagnostics.formatFlac)}</option>
              <option value="ogg">{t(strings.diagnostics.formatOgg)}</option>
            </Select>
          </label>
          <Button onClick={() => void run()} disabled={!file || busy}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Send className="h-4 w-4" aria-hidden="true" />}
            {t(strings.diagnostics.transcodeAction)}
          </Button>
        </div>
        {error ? <p className="text-sm text-app-danger">{error}</p> : null}
        {downloadUrl ? (
          <a href={downloadUrl} download={`transcoded.${format}`} className="text-sm text-app-primary underline">
            {t(strings.diagnostics.transcodeDownload)}
          </a>
        ) : null}
      </div>
    </Panel>
  );
}
