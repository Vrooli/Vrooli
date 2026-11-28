import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2, Send } from "lucide-react";
import { Panel } from "../../../components/ui/panel";
import { Button } from "../../../components/ui/button";
import { Textarea } from "@vrooli/react-component-library/Textarea/1.1.1";
import { Select } from "../../../components/ui/select";
import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { listSummarizeModels, summarize, type ProviderTrace } from "../../../services/diagnostics";
import type { ApiError } from "../../../api/client";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

export function SummarizeTryIt({ onTrace }: { onTrace: (t: ProviderTrace) => void }) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [level, setLevel] = useState<"light" | "moderate" | "heavy">("moderate");
  const [model, setModel] = useState("summarize.audio_tools");
  const models = useQuery({
    queryKey: ["models", "summarize"],
    queryFn: listSummarizeModels,
  });
  const modelRows = useMemo(() => models.data?.ok ? models.data.data : [], [models.data]);

  useEffect(() => {
    if (modelRows.length === 0) return;
    if (modelRows.some((candidate) => candidate.id === model)) return;
    const defaultModel = modelRows.find((candidate) => candidate.id === "summarize.audio_tools")
      ?? modelRows.find((candidate) => candidate.defaultEligible)
      ?? modelRows[0];
    if (defaultModel) setModel(defaultModel.id);
  }, [model, modelRows]);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState("");
  const [error, setError] = useState<ApiError | null>(null);

  const run = async () => {
    if (!text.trim()) return;
    setBusy(true);
    setError(null);
    const r = await summarize(text, level, model);
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
          <label htmlFor="summarize-model" className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            {t(strings.diagnostics.summarizeModelLabel)}
            <Select
              id="summarize-model"
              aria-describedby="summarize-model-notes"
              value={model}
              onChange={(e) => setModel(e.currentTarget.value)}
              className="w-48"
              disabled={modelRows.length === 0}
            >
              {modelRows.map((candidate) => (
                <option key={candidate.id} value={candidate.id}>{candidate.name}</option>
              ))}
            </Select>
          </label>
          <span className="max-w-sm text-xs text-app-muted-foreground" id="summarize-model-notes">
            {modelRows.find((candidate) => candidate.id === model)?.notes}
          </span>
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
