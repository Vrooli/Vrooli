import { useEffect, useState } from "react";
import { Loader2, Send } from "lucide-react";
import { Panel } from "../../components/ui/panel";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { Select } from "../../components/ui/select";
import { Tabs } from "../../components/ui/tabs";
import { Badge } from "../../components/ui/badge";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { summarize, type ProviderTrace } from "../../services/diagnostics";
import { synthesize, listVoices } from "../../services/tts";
import type { ApiError } from "../../api/client";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { TranscribeTryIt } from "./TranscribeTryIt";

interface TraceEntry extends ProviderTrace {
  capability: string;
  emittedAt: number;
}

export function DiagnosticsPage() {
  const { t } = useTranslation();
  const [trace, setTrace] = useState<TraceEntry[]>([]);
  const recordTrace = (capability: string, tr: ProviderTrace) =>
    setTrace((prev) => [{ ...tr, capability, emittedAt: Date.now() }, ...prev].slice(0, 10));

  return (
    <div className="flex max-w-6xl flex-col gap-4 md:gap-6">
      <PageHeader title={t(strings.diagnostics.title)} description={t(strings.diagnostics.description)} />

      <div className="grid gap-4 md:grid-cols-[1fr,22rem]">
        <Tabs
          ariaLabel={t(strings.diagnostics.tablistLabel)}
          items={[
            { value: "transcribe", label: t(strings.diagnostics.tabTranscribe) },
            { value: "synthesize", label: t(strings.diagnostics.tabSynthesize) },
            { value: "summarize", label: t(strings.diagnostics.tabSummarize) },
            { value: "transcode", label: t(strings.diagnostics.tabTranscode) },
          ]}
        >
          {(active) => {
            if (active === "transcribe") return <TranscribeTryIt onTrace={(tr) => recordTrace("stt", tr)} />;
            if (active === "synthesize") return <SynthesizeTryIt onTrace={(tr) => recordTrace("tts", tr)} />;
            if (active === "summarize") return <SummarizeTryIt onTrace={(tr) => recordTrace("summarize", tr)} />;
            if (active === "transcode") return <TranscodeTryIt />;
            return null;
          }}
        </Tabs>

        <ProviderTraceCard entries={trace} />
      </div>
    </div>
  );
}

/* ----------------------------- Try-its --------------------------------- */

function SummarizeTryIt({ onTrace }: { onTrace: (t: ProviderTrace) => void }) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [level, setLevel] = useState<"light" | "moderate" | "heavy">("moderate");
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState("");
  const [error, setError] = useState<ApiError | null>(null);

  const run = async () => {
    if (!text.trim()) return;
    setBusy(true);
    setError(null);
    const r = await summarize(text, level);
    setBusy(false);
    if (!r.ok) {
      setError(r.error);
      return;
    }
    setResult(r.data.text);
    onTrace(r.data.trace);
  };

  return (
    <Panel title={t(strings.diagnostics.summarizeTitle)} description={t(strings.diagnostics.summarizeDescription)}>
      <div className="flex flex-col gap-3">
        <Textarea
          value={text}
          onChange={(e) => setText(e.currentTarget.value)}
          placeholder={t(strings.diagnostics.summarizeInputPlaceholder)}
          rows={6}
          aria-label={t(strings.diagnostics.summarizeInputLabel)}
        />
        <div className="flex flex-wrap items-end gap-3">
          <label
            htmlFor="summarize-level"
            className="flex flex-col gap-1 text-xs text-app-muted-foreground"
          >
            {t(strings.diagnostics.levelLabel)}
            <Select
              id="summarize-level"
              value={level}
              onChange={(e) => setLevel(e.currentTarget.value as "light" | "moderate" | "heavy")}
              className="w-40"
            >
              <option value="light">{t(strings.diagnostics.levelLight)}</option>
              <option value="moderate">{t(strings.diagnostics.levelModerate)}</option>
              <option value="heavy">{t(strings.diagnostics.levelHeavy)}</option>
            </Select>
          </label>
          <Button onClick={() => void run()} disabled={!text.trim() || busy}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Send className="h-4 w-4" aria-hidden="true" />}
            {t(strings.diagnostics.summarizeAction)}
          </Button>
        </div>
        {error ? <ApiErrorState error={error} /> : null}
        {result ? (
          <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm whitespace-pre-wrap">
            {result}
          </div>
        ) : null}
      </div>
    </Panel>
  );
}

function SynthesizeTryIt({ onTrace }: { onTrace: (t: ProviderTrace) => void }) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [voice, setVoice] = useState("voice.feminine.warm");
  const [voices, setVoices] = useState<{ id: string; name: string }[]>([]);
  const [busy, setBusy] = useState(false);
  const [audioUrl, setAudioUrl] = useState<string>("");
  const [error, setError] = useState<ApiError | null>(null);

  useEffect(() => {
    let cancelled = false;
    void listVoices().then((r) => {
      if (cancelled) return;
      if (r.ok && r.data.length > 0) setVoices(r.data);
    });
    return () => { cancelled = true; };
  }, []);

  const run = async () => {
    if (!text.trim()) return;
    setBusy(true);
    setError(null);
    if (audioUrl) URL.revokeObjectURL(audioUrl);
    const r = await synthesize(text, voice, 1.0, "wav");
    setBusy(false);
    if (!r.ok) {
      setError(r.error);
      return;
    }
    const buf = r.data.audio.slice().buffer;
    const blob = new Blob([buf], { type: r.data.contentType || "audio/wav" });
    setAudioUrl(URL.createObjectURL(blob));
    onTrace({ providerTier: r.data.providerTier, providerId: r.data.providerId, modelId: r.data.modelId, latencyMs: r.data.latencyMs });
  };

  return (
    <Panel title={t(strings.diagnostics.synthesizeTitle)} description={t(strings.diagnostics.synthesizeDescription)}>
      <div className="flex flex-col gap-3">
        <Textarea value={text} onChange={(e) => setText(e.currentTarget.value)} rows={4} placeholder="Type text to synthesize…" />
        <div className="flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            Voice
            <Select value={voice} onChange={(e) => setVoice(e.currentTarget.value)} className="w-56">
              {(voices.length ? voices : [{ id: voice, name: voice }]).map((v) => (
                <option key={v.id} value={v.id}>{v.name}</option>
              ))}
            </Select>
          </label>
          <Button onClick={() => void run()} disabled={!text.trim() || busy}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Send className="h-4 w-4" aria-hidden="true" />}
            Synthesize
          </Button>
        </div>
        {error ? <ApiErrorState error={error} /> : null}
        {audioUrl ? <audio controls src={audioUrl} className="w-full" /> : null}
      </div>
    </Panel>
  );
}

function TranscodeTryIt() {
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
    const { uploadFile } = await import("../../api/client");
    const res = await uploadFile("/api/v1/audio/transcode", fd);
    setBusy(false);
    if (!res.ok) {
      setError(`Transcode failed (${res.status})`);
      return;
    }
    const blob = await res.blob();
    setDownloadUrl(URL.createObjectURL(blob));
  };

  return (
    <Panel title={t(strings.diagnostics.transcodeTitle)} description={t(strings.diagnostics.transcodeDescription)}>
      <div className="flex flex-col gap-3">
        <Input type="file" accept="audio/*" onChange={(e) => setFile(e.currentTarget.files?.[0] ?? null)} />
        <div className="flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            Output format
            <Select value={format} onChange={(e) => setFormat(e.currentTarget.value)} className="w-32">
              <option value="wav">wav</option>
              <option value="mp3">mp3</option>
              <option value="flac">flac</option>
              <option value="ogg">ogg</option>
            </Select>
          </label>
          <Button onClick={() => void run()} disabled={!file || busy}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Send className="h-4 w-4" aria-hidden="true" />}
            Transcode
          </Button>
        </div>
        {error ? <p className="text-sm text-app-danger">{error}</p> : null}
        {downloadUrl ? (
          <a href={downloadUrl} download={`transcoded.${format}`} className="text-sm text-app-primary underline">
            Download transcoded file
          </a>
        ) : null}
      </div>
    </Panel>
  );
}

/* ----------------------------- Trace ----------------------------------- */

function ProviderTraceCard({ entries }: { entries: TraceEntry[] }) {
  const { t } = useTranslation();
  return (
    <aside className="flex flex-col gap-2">
      <Panel title={t(strings.diagnostics.traceTitle)} description={t(strings.diagnostics.traceDescription)} bodyless>
        {entries.length === 0 ? (
          <div className="p-4 text-sm text-app-muted-foreground">{t(strings.diagnostics.traceEmpty)}</div>
        ) : (
          <ul className="divide-y divide-app-border">
            {entries.map((e) => (
              <li key={`${e.emittedAt}-${e.providerId}`} className="flex flex-col gap-1 px-4 py-2 text-xs">
                <div className="flex items-center gap-2">
                  <Badge variant="info">{e.capability}</Badge>
                  <span className="font-mono">{e.providerTier}</span>
                  <span className="text-app-muted-foreground">{e.providerId}</span>
                </div>
                <div className="flex items-center justify-between text-app-muted-foreground">
                  <span className="font-mono">{e.modelId || t(strings.common.dash)}</span>
                  <span>{t(strings.common.millisSuffix, { ms: Math.round(e.latencyMs) })}</span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Panel>
    </aside>
  );
}
