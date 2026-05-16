import { useState } from "react";
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
import { summarize, transcribe, type ProviderTrace } from "../../services/diagnostics";
import type { ApiError } from "../../api/client";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

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
            if (active === "synthesize") return <SynthesizeStub />;
            if (active === "summarize") return <SummarizeTryIt onTrace={(tr) => recordTrace("summarize", tr)} />;
            if (active === "transcode") return <TranscodeStub />;
            return null;
          }}
        </Tabs>

        <ProviderTraceCard entries={trace} />
      </div>
    </div>
  );
}

/* ----------------------------- Try-its --------------------------------- */

function TranscribeTryIt({ onTrace }: { onTrace: (t: ProviderTrace) => void }) {
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
    <Panel title={t(strings.diagnostics.transcribeTitle)} description={t(strings.diagnostics.transcribeDescription)}>
      <div className="flex flex-col gap-3">
        <Input
          type="file"
          accept="audio/*"
          onChange={(e) => setFile(e.currentTarget.files?.[0] ?? null)}
          aria-label={t(strings.diagnostics.audioFileLabel)}
        />
        <div>
          <Button onClick={() => void run()} disabled={!file || busy}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Send className="h-4 w-4" aria-hidden="true" />}
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
    </Panel>
  );
}

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

function SynthesizeStub() {
  const { t } = useTranslation();
  return (
    <Panel title={t(strings.diagnostics.synthesizeTitle)} description={t(strings.diagnostics.synthesizeDescription)}>
      <p className="text-sm text-app-muted-foreground">{t(strings.diagnostics.synthesizeFollowUp)}</p>
    </Panel>
  );
}

function TranscodeStub() {
  const { t } = useTranslation();
  return (
    <Panel title={t(strings.diagnostics.transcodeTitle)} description={t(strings.diagnostics.transcodeDescription)}>
      <p className="text-sm text-app-muted-foreground">{t(strings.diagnostics.transcodeFollowUp)}</p>
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
