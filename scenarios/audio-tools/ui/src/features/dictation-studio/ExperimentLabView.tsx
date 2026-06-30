import { useMemo, useState, type Dispatch, type SetStateAction } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FlaskConical, Loader2, RefreshCw } from "lucide-react";

import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Panel } from "../../components/ui/panel";
import { Select } from "../../components/ui/select";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { Textarea } from "../../components/ui/textarea";
import { pushToast } from "../../components/ui/toast";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  cancelExperiment,
  compareExperiments,
  getExperimentReport,
  listExperiments,
  startExperiment,
  waitExperiment,
  type ExperimentReportRow,
  type ExperimentRow,
  type StartExperimentInput,
} from "../../services/experiment";
import { EvalReportTable } from "./EvalReportView";

const strategyOptions = ["batch", "vad_segment", "overlap_agree"] as const;

function pct(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

function isTerminal(status: ExperimentRow["status"]): boolean {
  return status === "succeeded" || status === "failed" || status === "canceled";
}

function defaultInput(): StartExperimentInput {
  return {
    name: "Dictation experiment",
    strategies: [...strategyOptions],
    realtimeRepeats: 0,
    chunkMs: 100,
    seed: 42,
    longForm: true,
    targetDurationSeconds: 180,
    gapMs: 5000,
    tagContains: "",
    overlapMaxStallRejects: -1,
    overlapWindowMs: 0,
    overlapCommitRuns: 0,
    overlapMaxWindowMs: 25000,
    vadSilenceMs: 0,
    noiseTypesCsv: "",
    snrDbCsv: "12",
    competingVoicesCsv: "",
    competingText: "",
    speakerTargetProfileId: "",
    speakerExtraction: false,
    speakerVerification: false,
    speakerMode: "filter",
    speakerThreshold: 0.5,
    speakerFallback: true,
    speakerAblation: false,
    droppedSpanThresholdWords: 4,
  };
}

export function ExperimentLabView() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [input, setInput] = useState<StartExperimentInput>(() => defaultInput());
  const [selectedId, setSelectedId] = useState("");
  const [report, setReport] = useState<ExperimentReportRow | null>(null);
  const [compareIds, setCompareIds] = useState("");
  const [compareRows, setCompareRows] = useState<ExperimentReportRow[]>([]);

  const history = useQuery({
    queryKey: ["experiments", "list"],
    queryFn: () => listExperiments(),
  });

  const start = useMutation({
    mutationFn: () => startExperiment(input),
    onSuccess: (row) => {
      setSelectedId(row.id);
      void qc.invalidateQueries({ queryKey: ["experiments", "list"] });
      pushToast({ title: t(strings.dictationStudio.experimentStarted) });
    },
  });

  const wait = useMutation({
    mutationFn: (id: string) => waitExperiment(id),
    onSuccess: ({ experiment }) => {
      if (experiment) setSelectedId(experiment.id);
      void qc.invalidateQueries({ queryKey: ["experiments", "list"] });
    },
  });

  const cancel = useMutation({
    mutationFn: (id: string) => cancelExperiment(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["experiments", "list"] });
    },
  });

  const loadReport = useMutation({
    mutationFn: (id: string) => getExperimentReport(id),
    onSuccess: (row) => {
      setReport(row);
      if (row.experiment) setSelectedId(row.experiment.id);
    },
  });

  const compare = useMutation({
    mutationFn: () => compareExperiments(compareIds.split(",").map((id) => id.trim()).filter(Boolean)),
    onSuccess: setCompareRows,
  });

  const selected = useMemo(
    () => history.data?.find((row) => row.id === selectedId) ?? report?.experiment ?? null,
    [history.data, report, selectedId],
  );

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.08fr)_minmax(360px,0.92fr)]">
      <ExperimentBuilder input={input} setInput={setInput} pending={start.isPending} onStart={() => start.mutate()} />

      <Panel
        title={t(strings.dictationStudio.historyTitle)}
        description={t(strings.dictationStudio.historyHint)}
        actions={
          <Button
            type="button"
            variant="outline"
            size="sm"
            data-testid={selectors.dictationStudio.refreshExperiments}
            onClick={() => void history.refetch()}
          >
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            {t(strings.dictationStudio.refresh)}
          </Button>
        }
      >
        <ExperimentHistory
          rows={history.data ?? []}
          pending={history.isPending}
          error={history.error}
          selectedId={selectedId}
          onSelect={setSelectedId}
          onWait={(id) => wait.mutate(id)}
          onCancel={(id) => cancel.mutate(id)}
          onReport={(id) => loadReport.mutate(id)}
          onRetry={() => void history.refetch()}
          actionPending={wait.isPending || cancel.isPending || loadReport.isPending}
        />
      </Panel>

      <Panel title={t(strings.dictationStudio.resultsTitle)} description={t(strings.dictationStudio.resultsHint)} className="xl:col-span-2">
        {loadReport.isError ? (
          <ApiErrorState error={loadReport.error} title={t(strings.dictationStudio.reportError)} onRetry={() => selectedId && loadReport.mutate(selectedId)} />
        ) : null}
        {report ? (
          <div data-testid={selectors.dictationStudio.experimentResults} className="flex flex-col gap-4">
            <ExperimentRecipeSummary row={report.experiment} />
            <EvalReportTable report={report.report} />
          </div>
        ) : selected ? (
          <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-app-muted-foreground">
            <span>{t(strings.dictationStudio.resultsSelectHint)}</span>
            <Button type="button" onClick={() => loadReport.mutate(selected.id)} disabled={loadReport.isPending}>
              {loadReport.isPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <FlaskConical className="h-4 w-4" aria-hidden="true" />}
              {t(strings.dictationStudio.loadReport)}
            </Button>
          </div>
        ) : (
          <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.resultsEmpty)}</p>
        )}
      </Panel>

      <Panel title={t(strings.dictationStudio.compareTitle)} description={t(strings.dictationStudio.compareHint)} className="xl:col-span-2">
        <div className="flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.compareIdsLabel)}
            <Input
              data-testid={selectors.dictationStudio.compareIds}
              value={compareIds}
              onChange={(event) => setCompareIds(event.currentTarget.value)}
              placeholder={t(strings.dictationStudio.comparePlaceholder)}
            />
          </label>
          <div>
            <Button
              type="button"
              data-testid={selectors.dictationStudio.compareExperiments}
              disabled={compare.isPending || compareIds.split(",").filter(Boolean).length < 2}
              onClick={() => compare.mutate()}
            >
              {compare.isPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : null}
              {t(strings.dictationStudio.compareRun)}
            </Button>
          </div>
          {compare.isError ? <ApiErrorState error={compare.error} title={t(strings.dictationStudio.compareError)} onRetry={() => compare.mutate()} /> : null}
          {compareRows.length > 0 ? <CompareResults rows={compareRows} /> : null}
        </div>
      </Panel>
    </div>
  );
}

function ExperimentBuilder({
  input,
  setInput,
  pending,
  onStart,
}: {
  input: StartExperimentInput;
  setInput: Dispatch<SetStateAction<StartExperimentInput>>;
  pending: boolean;
  onStart: () => void;
}) {
  const { t } = useTranslation();
  const set = <K extends keyof StartExperimentInput>(key: K, value: StartExperimentInput[K]) =>
    setInput((current) => ({ ...current, [key]: value }));
  const toggleStrategy = (kind: string) =>
    setInput((current) => ({
      ...current,
      strategies: current.strategies.includes(kind)
        ? current.strategies.filter((item) => item !== kind)
        : [...current.strategies, kind],
    }));

  return (
    <Panel title={t(strings.dictationStudio.builderTitle)} description={t(strings.dictationStudio.builderHint)}>
      <div className="grid gap-4 lg:grid-cols-2">
        <label className="flex flex-col gap-1 text-xs lg:col-span-2">
          {t(strings.dictationStudio.experimentNameLabel)}
          <Input data-testid={selectors.dictationStudio.experimentName} value={input.name} onChange={(event) => set("name", event.currentTarget.value)} />
        </label>

        <fieldset className="flex flex-col gap-2 rounded-control border border-app-border p-3">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.strategiesLabel)}</legend>
          <div className="flex flex-wrap gap-2">
            {strategyOptions.map((kind) => (
              <Button
                key={kind}
                type="button"
                variant={input.strategies.includes(kind) ? "default" : "outline"}
                aria-pressed={input.strategies.includes(kind)}
                onClick={() => toggleStrategy(kind)}
              >
                {kind}
              </Button>
            ))}
          </div>
        </fieldset>

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.longFormLabel)}</legend>
          <label className="flex items-center gap-2 text-xs sm:col-span-2">
            <input
              data-testid={selectors.dictationStudio.experimentLongForm}
              type="checkbox"
              checked={input.longForm}
              onChange={(event) => set("longForm", event.currentTarget.checked)}
            />
            {t(strings.dictationStudio.longFormEnabled)}
          </label>
          <NumberField testId={selectors.dictationStudio.experimentSeed} label={t(strings.dictationStudio.seedLabel)} value={input.seed} onChange={(value) => set("seed", value)} />
          <NumberField testId={selectors.dictationStudio.experimentTargetDuration} label={t(strings.dictationStudio.targetDurationLabel)} value={input.targetDurationSeconds} onChange={(value) => set("targetDurationSeconds", value)} />
          <NumberField testId={selectors.dictationStudio.experimentGapMs} label={t(strings.dictationStudio.gapMsLabel)} value={input.gapMs} onChange={(value) => set("gapMs", value)} />
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.tagFilterLabel)}
            <Input data-testid={selectors.dictationStudio.experimentTagContains} value={input.tagContains} onChange={(event) => set("tagContains", event.currentTarget.value)} />
          </label>
        </fieldset>

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.hyperparamsLabel)}</legend>
          <NumberField testId={selectors.dictationStudio.experimentRealtimeRepeats} label={t(strings.dictationStudio.repeatsLabel)} value={input.realtimeRepeats} onChange={(value) => set("realtimeRepeats", value)} />
          <NumberField testId={selectors.dictationStudio.experimentOverlapMaxWindow} label={t(strings.dictationStudio.maxWindowLabel)} value={input.overlapMaxWindowMs} onChange={(value) => set("overlapMaxWindowMs", value)} />
        </fieldset>

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.augmentationLabel)}</legend>
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.noiseTypesLabel)}
            <Input data-testid={selectors.dictationStudio.experimentNoiseTypes} value={input.noiseTypesCsv} onChange={(event) => set("noiseTypesCsv", event.currentTarget.value)} placeholder={t(strings.dictationStudio.noisePlaceholder)} />
          </label>
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.snrLabel)}
            <Input data-testid={selectors.dictationStudio.experimentSnrDb} value={input.snrDbCsv} onChange={(event) => set("snrDbCsv", event.currentTarget.value)} placeholder={t(strings.dictationStudio.snrPlaceholder)} />
          </label>
          <label className="flex flex-col gap-1 text-xs sm:col-span-2">
            {t(strings.dictationStudio.competingVoicesLabel)}
            <Input data-testid={selectors.dictationStudio.experimentCompetingVoices} value={input.competingVoicesCsv} onChange={(event) => set("competingVoicesCsv", event.currentTarget.value)} />
          </label>
          <label className="flex flex-col gap-1 text-xs sm:col-span-2">
            {t(strings.dictationStudio.competingTextLabel)}
            <Textarea rows={2} value={input.competingText} onChange={(event) => set("competingText", event.currentTarget.value)} />
          </label>
        </fieldset>

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.speakerDimensionLabel)}</legend>
          <label className="flex flex-col gap-1 text-xs sm:col-span-2">
            {t(strings.dictationStudio.targetProfileLabel)}
            <Input data-testid={selectors.dictationStudio.experimentSpeakerProfile} value={input.speakerTargetProfileId} onChange={(event) => set("speakerTargetProfileId", event.currentTarget.value)} />
          </label>
          <label className="flex items-center gap-2 text-xs">
            <input type="checkbox" checked={input.speakerExtraction} onChange={(event) => set("speakerExtraction", event.currentTarget.checked)} />
            {t(strings.dictationStudio.speakerExtractionLabel)}
          </label>
          <label className="flex items-center gap-2 text-xs">
            <input type="checkbox" checked={input.speakerVerification} onChange={(event) => set("speakerVerification", event.currentTarget.checked)} />
            {t(strings.dictationStudio.speakerVerificationLabel)}
          </label>
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.speakerModeLabel)}
            <Select value={input.speakerMode} onChange={(event) => set("speakerMode", event.currentTarget.value as StartExperimentInput["speakerMode"])}>
              <option value="filter">{t(strings.speakerAdmin.modeFilter)}</option>
              <option value="advisory">{t(strings.speakerAdmin.modeAdvisory)}</option>
              <option value="off">{t(strings.speakerAdmin.modeOff)}</option>
            </Select>
          </label>
          <NumberField label={t(strings.speakerAdmin.configThreshold)} value={input.speakerThreshold} step={0.05} onChange={(value) => set("speakerThreshold", value)} />
          <label className="flex items-center gap-2 text-xs sm:col-span-2">
            <input type="checkbox" checked={input.speakerAblation} onChange={(event) => set("speakerAblation", event.currentTarget.checked)} />
            {t(strings.dictationStudio.speakerAblationLabel)}
          </label>
        </fieldset>
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <Button
          type="button"
          data-testid={selectors.dictationStudio.startExperiment}
          disabled={pending || input.strategies.length === 0}
          onClick={onStart}
        >
          {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <FlaskConical className="h-4 w-4" aria-hidden="true" />}
          {t(strings.dictationStudio.startExperiment)}
        </Button>
        <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.builderSafetyHint)}</span>
      </div>
    </Panel>
  );
}

function NumberField({
  label,
  value,
  onChange,
  testId,
  step = 1,
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
  testId?: string;
  step?: number;
}) {
  return (
    <label className="flex flex-col gap-1 text-xs">
      {label}
      <Input
        data-testid={testId}
        type="number"
        step={step}
        value={value}
        onChange={(event) => onChange(Number(event.currentTarget.value) || 0)}
      />
    </label>
  );
}

function ExperimentHistory({
  rows,
  pending,
  error,
  selectedId,
  onSelect,
  onWait,
  onCancel,
  onReport,
  onRetry,
  actionPending,
}: {
  rows: ExperimentRow[];
  pending: boolean;
  error: Error | null;
  selectedId: string;
  onSelect: (id: string) => void;
  onWait: (id: string) => void;
  onCancel: (id: string) => void;
  onReport: (id: string) => void;
  onRetry: () => void;
  actionPending: boolean;
}) {
  const { t } = useTranslation();
  if (pending) return <LoadingRows rows={3} label={t(strings.dictationStudio.historyTitle)} />;
  if (error) return <ApiErrorState error={error} title={t(strings.dictationStudio.historyError)} onRetry={onRetry} />;
  if (rows.length === 0) return <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.historyEmpty)}</p>;

  return (
    <div className="overflow-x-auto">
      <Table>
        <THead>
          <TR>
            <TH>{t(strings.dictationStudio.colName)}</TH>
            <TH>{t(strings.dictationStudio.colStatus)}</TH>
            <TH>{t(strings.dictationStudio.colRecipe)}</TH>
            <TH>{t(strings.dictationStudio.colActions)}</TH>
          </TR>
        </THead>
        <TBody>
          {rows.map((row) => (
            <TR
              key={row.id}
              data-testid={selectors.dictationStudio.experimentRow({ id: row.id })}
              className={row.id === selectedId ? "bg-app-surface-muted" : undefined}
            >
              <TD>
                <button type="button" className="text-left font-medium text-app-foreground underline-offset-2 hover:underline" onClick={() => onSelect(row.id)}>
                  {row.name || row.id}
                </button>
                <div className="text-xs text-app-muted-foreground">{row.id}</div>
              </TD>
              <TD><StatusBadge status={row.status} /></TD>
              <TD className="text-xs text-app-muted-foreground">
                {row.recipe.strategies.join(", ") || t(strings.dictationStudio.recipeDefault)} · {row.recipe.longFormEnabled ? `${row.recipe.targetDurationSeconds}s` : t(strings.dictationStudio.recipeClips)}
              </TD>
              <TD>
                <div className="flex flex-wrap gap-1">
                  {!isTerminal(row.status) ? (
                    <>
                      <Button type="button" size="sm" variant="outline" data-testid={selectors.dictationStudio.experimentWait({ id: row.id })} disabled={actionPending} onClick={() => onWait(row.id)}>
                        {t(strings.dictationStudio.wait)}
                      </Button>
                      <Button type="button" size="sm" variant="ghost" data-testid={selectors.dictationStudio.experimentCancel({ id: row.id })} disabled={actionPending} onClick={() => onCancel(row.id)}>
                        {t(strings.dictationStudio.cancel)}
                      </Button>
                    </>
                  ) : null}
                  <Button type="button" size="sm" variant="ghost" data-testid={selectors.dictationStudio.experimentReport({ id: row.id })} disabled={actionPending} onClick={() => onReport(row.id)}>
                    {t(strings.dictationStudio.report)}
                  </Button>
                </div>
              </TD>
            </TR>
          ))}
        </TBody>
      </Table>
    </div>
  );
}

function StatusBadge({ status }: { status: ExperimentRow["status"] }) {
  const variant = status === "succeeded" ? "success" : status === "failed" || status === "canceled" ? "danger" : status === "running" ? "info" : "neutral";
  return <Badge variant={variant}>{status}</Badge>;
}

function ExperimentRecipeSummary({ row }: { row: ExperimentRow | null }) {
  const { t } = useTranslation();
  if (!row) return null;
  const r = row.recipe;
  return (
    <div className="grid gap-3 rounded-control border border-app-border p-3 text-sm md:grid-cols-4">
      <div>
        <div className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.colStatus)}</div>
        <StatusBadge status={row.status} />
      </div>
      <div>
        <div className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.strategiesLabel)}</div>
        <div>{r.strategies.join(", ") || t(strings.dictationStudio.recipeDefault)}</div>
      </div>
      <div>
        <div className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.longFormLabel)}</div>
        <div>{r.longFormEnabled ? `${r.targetDurationSeconds}s · ${r.gapMs}ms` : t(strings.common.dash)}</div>
      </div>
      <div>
        <div className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.augmentationLabel)}</div>
        <div>{[...r.noiseTypes, ...r.competingVoiceIds].join(", ") || t(strings.common.dash)}</div>
      </div>
      {row.error ? <p className="text-xs text-app-danger md:col-span-4">{row.error}</p> : null}
      {r.realizedReference ? (
        <p className="max-h-24 overflow-auto text-xs text-app-muted-foreground md:col-span-4">{r.realizedReference}</p>
      ) : null}
    </div>
  );
}

function CompareResults({ rows }: { rows: ExperimentReportRow[] }) {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.dictationStudio.compareResults} className="overflow-x-auto">
      <Table>
        <THead>
          <TR>
            <TH>{t(strings.dictationStudio.colName)}</TH>
            <TH>{t(strings.dictationStudio.recommendationTitle)}</TH>
            <TH>{t(strings.dictationStudio.colWer)}</TH>
            <TH>{t(strings.dictationStudio.colStatus)}</TH>
          </TR>
        </THead>
        <TBody>
          {rows.map((row) => {
            const winner = row.report.summary?.winnerStrategy;
            const strategy = row.report.perStrategy.find((item) => item.strategy === winner) ?? row.report.perStrategy[0];
            return (
              <TR key={row.experiment?.id ?? row.report.summary?.recommendation ?? "compare-row"}>
                <TD>{row.experiment?.name ?? row.experiment?.id ?? t(strings.common.dash)}</TD>
                <TD>{row.report.summary?.recommendation ?? t(strings.common.dash)}</TD>
                <TD>{strategy ? pct(strategy.wer) : t(strings.common.dash)}</TD>
                <TD>{row.experiment ? <StatusBadge status={row.experiment.status} /> : t(strings.common.dash)}</TD>
              </TR>
            );
          })}
        </TBody>
      </Table>
    </div>
  );
}
