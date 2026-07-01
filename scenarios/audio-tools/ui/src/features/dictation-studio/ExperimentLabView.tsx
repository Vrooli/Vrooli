import { useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from "react";
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
import { listClips, type ClipMeta } from "../../services/corpus";
import {
  cancelExperiment,
  compareExperiments,
  getExperiment,
  getExperimentReport,
  listExperiments,
  startExperiment,
  streamExperimentEvents,
  waitExperiment,
  type ExperimentEventRow,
  type ExperimentReportRow,
  type ExperimentRow,
  type StartExperimentInput,
} from "../../services/experiment";
import { getSpeakerStatus } from "../../services/speakerAdmin";
import { EvalReportTable } from "./EvalReportTable";

const strategyOptions = ["batch", "vad_segment", "overlap_agree"] as const;

function pct(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

function ms(value: number): string {
  return String(Math.round(value));
}

function isTerminal(status: ExperimentRow["status"]): boolean {
  return status === "succeeded" || status === "failed" || status === "canceled";
}
function hasSweepDurations(input: StartExperimentInput): boolean {
  return input.sweepDurationsCsv
    .split(",")
    .map((part) => part.trim())
    .some(Boolean);
}

function StatusLabel({ status }: { status: ExperimentRow["status"] }) {
  const { t } = useTranslation();
  switch (status) {
    case "queued":
      return <>{t(strings.dictationStudio.statusQueued)}</>;
    case "running":
      return <>{t(strings.dictationStudio.statusRunning)}</>;
    case "succeeded":
      return <>{t(strings.dictationStudio.statusSucceeded)}</>;
    case "failed":
      return <>{t(strings.dictationStudio.statusFailed)}</>;
    case "canceled":
      return <>{t(strings.dictationStudio.statusCanceled)}</>;
    default:
      return <>{t(strings.dictationStudio.statusUnspecified)}</>;
  }
}

function StrategyName({ kind }: { kind: string }) {
  const { t } = useTranslation();
  switch (kind) {
    case "batch":
      return <>{t(strings.dictationStudio.strategyBatch)}</>;
    case "vad_segment":
      return <>{t(strings.dictationStudio.strategyVadSegment)}</>;
    case "overlap_agree":
      return <>{t(strings.dictationStudio.strategyOverlapAgree)}</>;
    default:
      return <>{kind}</>;
  }
}

function defaultInput(): StartExperimentInput {
  return {
    name: "Dictation experiment",
    clipIds: [],
    strategies: [...strategyOptions],
    realtimeRepeats: 0,
    latencyTailSeconds: 8,
    chunkMs: 100,
    seed: 42,
    longForm: true,
    targetDurationSeconds: 180,
    gapMs: 5000,
    tagContains: "",
    sweepDurationsCsv: "",
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
  const [activeExperiment, setActiveExperiment] = useState<ExperimentRow | null>(null);
  const [liveEvent, setLiveEvent] = useState<ExperimentEventRow | null>(null);
  const [streamError, setStreamError] = useState("");
  const [report, setReport] = useState<ExperimentReportRow | null>(null);
  const [compareSelected, setCompareSelected] = useState<string[]>([]);
  const [compareRows, setCompareRows] = useState<ExperimentReportRow[]>([]);

  const history = useQuery({
    queryKey: ["experiments", "list"],
    queryFn: () => listExperiments(),
  });

  const start = useMutation({
    mutationFn: () => startExperiment(input),
    onSuccess: (row) => {
      setSelectedId(row.id);
      setActiveExperiment(row);
      setReport(null);
      setStreamError("");
      setLiveEvent({
        experimentId: row.id,
        status: row.status,
        progress: row.status === "queued" ? 0 : 1,
        message: row.status,
        at: row.createdAt,
      });
      void qc.invalidateQueries({ queryKey: ["experiments", "list"] });
      pushToast({ title: t(strings.dictationStudio.experimentStarted) });
    },
  });

  const wait = useMutation({
    mutationFn: (id: string) => waitExperiment(id),
    onSuccess: ({ experiment }) => {
      if (experiment) {
        setSelectedId(experiment.id);
        setActiveExperiment(experiment);
      }
      void qc.invalidateQueries({ queryKey: ["experiments", "list"] });
    },
  });

  const cancel = useMutation({
    mutationFn: (id: string) => cancelExperiment(id),
    onSuccess: (experiment) => {
      if (experiment) setActiveExperiment(experiment);
      void qc.invalidateQueries({ queryKey: ["experiments", "list"] });
    },
  });

  const loadReport = useMutation({
    mutationFn: (id: string) => getExperimentReport(id),
    onSuccess: (row) => {
      setReport(row);
      if (row.experiment) {
        setSelectedId(row.experiment.id);
        setActiveExperiment(row.experiment);
      }
    },
  });

  const compare = useMutation({
    mutationFn: (ids: string[]) => compareExperiments(ids),
    onSuccess: setCompareRows,
  });
  const qcRef = useRef(qc);
  const loadReportRef = useRef(loadReport.mutate);
  const tRef = useRef(t);
  qcRef.current = qc;
  loadReportRef.current = loadReport.mutate;
  tRef.current = t;

  const selected = useMemo(
    () => history.data?.find((row) => row.id === selectedId) ?? report?.experiment ?? activeExperiment,
    [activeExperiment, history.data, report, selectedId],
  );
  const selectedExperimentId = selected?.id ?? "";
  const selectedExperimentStatus = selected?.status ?? "unspecified";

  const selectedLiveEvent = liveEvent?.experimentId === selectedId ? liveEvent : null;

  const toggleCompare = (id: string) =>
    setCompareSelected((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id],
    );

  useEffect(() => {
    if (!selectedExperimentId || isTerminal(selectedExperimentStatus)) return;
    const controller = new AbortController();
    let closed = false;
    let fallbackTimer: number | null = null;

    const handleTerminal = (id: string) => {
      void qcRef.current.invalidateQueries({ queryKey: ["experiments", "list"] });
      loadReportRef.current(id);
    };

    const startFallbackPolling = () => {
      if (closed || fallbackTimer) return;
      fallbackTimer = window.setInterval(() => {
        void getExperiment(selectedExperimentId)
          .then(({ experiment }) => {
            if (!experiment || closed) return;
            setActiveExperiment(experiment);
            setLiveEvent((current) => ({
              experimentId: experiment.id,
              status: experiment.status,
              progress: isTerminal(experiment.status) ? 100 : current?.experimentId === experiment.id ? current.progress : 0,
              message: isTerminal(experiment.status)
                ? tRef.current(strings.dictationStudio.liveComplete)
                : tRef.current(strings.dictationStudio.livePolling),
              at: experiment.finishedAt || experiment.startedAt || experiment.createdAt,
            }));
            if (isTerminal(experiment.status)) {
              handleTerminal(experiment.id);
              if (fallbackTimer) window.clearInterval(fallbackTimer);
              fallbackTimer = null;
            }
          })
          .catch((error: unknown) => {
            if (!closed) setStreamError(error instanceof Error ? error.message : String(error));
          });
      }, 2500);
    };

    setStreamError("");
    let terminalSeen = false;
    void streamExperimentEvents(
      selectedExperimentId,
      (event) => {
        if (closed) return;
        setLiveEvent(event);
        if (isTerminal(event.status)) {
          terminalSeen = true;
          handleTerminal(event.experimentId);
          controller.abort();
        }
      },
      controller.signal,
    )
      .then(() => {
        if (closed || controller.signal.aborted || terminalSeen) return;
        setStreamError(tRef.current(strings.dictationStudio.liveStreamClosed));
        startFallbackPolling();
      })
      .catch((error: unknown) => {
        if (closed || controller.signal.aborted) return;
        setStreamError(error instanceof Error ? error.message : String(error));
        startFallbackPolling();
      });

    return () => {
      closed = true;
      controller.abort();
      if (fallbackTimer) window.clearInterval(fallbackTimer);
    };
  }, [selectedExperimentId, selectedExperimentStatus]);

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
          compareSelected={compareSelected}
          onToggleCompare={toggleCompare}
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
        {selected && !isTerminal(selected.status) ? (
          <LiveExperimentProgress event={selectedLiveEvent} fallbackMessage={streamError} status={selected.status} />
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
          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              data-testid={selectors.dictationStudio.compareExperiments}
              disabled={compare.isPending || compareSelected.length < 2}
              onClick={() => compare.mutate(compareSelected)}
            >
              {compare.isPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : null}
              {t(strings.dictationStudio.compareRunSelected, { count: compareSelected.length })}
            </Button>
            {compareSelected.length < 2 ? (
              <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.compareSelectHint)}</span>
            ) : null}
          </div>
          {compare.isError ? <ApiErrorState error={compare.error} title={t(strings.dictationStudio.compareError)} onRetry={() => compare.mutate(compareSelected)} /> : null}
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
  const speakerStatus = useQuery({
    queryKey: ["speaker", "status"],
    queryFn: getSpeakerStatus,
  });
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
                <StrategyName kind={kind} />
              </Button>
            ))}
          </div>
        </fieldset>

        <fieldset className="flex flex-col gap-2 rounded-control border border-app-border p-3 lg:col-span-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.safetyGateLabel)}</legend>
          <div className="grid gap-2 sm:grid-cols-2">
            <NumberField
              testId={selectors.dictationStudio.experimentDroppedSpanThreshold}
              label={t(strings.dictationStudio.droppedSpanThresholdLabel)}
              value={input.droppedSpanThresholdWords}
              min={0}
              onChange={(value) => set("droppedSpanThresholdWords", value)}
            />
          </div>
          <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.droppedSpanThresholdHint)}</p>
        </fieldset>

        <ClipPicker selected={input.clipIds} onChange={(ids) => set("clipIds", ids)} />

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2 lg:col-span-2">
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
          <NumberField testId={selectors.dictationStudio.experimentTargetDuration} label={t(strings.dictationStudio.targetDurationLabel)} value={input.targetDurationSeconds} min={0} onChange={(value) => set("targetDurationSeconds", value)} />
          <NumberField testId={selectors.dictationStudio.experimentGapMs} label={t(strings.dictationStudio.gapMsLabel)} value={input.gapMs} min={0} onChange={(value) => set("gapMs", value)} />
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.tagFilterLabel)}
            <Input data-testid={selectors.dictationStudio.experimentTagContains} value={input.tagContains} onChange={(event) => set("tagContains", event.currentTarget.value)} />
          </label>
          <label className="flex flex-col gap-1 text-xs sm:col-span-2">
            {t(strings.dictationStudio.sweepDurationsLabel)}
            <Input
              data-testid={selectors.dictationStudio.experimentSweepDurations}
              value={input.sweepDurationsCsv}
              onChange={(event) => set("sweepDurationsCsv", event.currentTarget.value)}
              placeholder={t(strings.dictationStudio.sweepPlaceholder)}
            />
            <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.sweepDurationsHint)}</span>
          </label>
        </fieldset>

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.hyperparamsLabel)}</legend>
          <div className="flex flex-col gap-1">
            <NumberField testId={selectors.dictationStudio.experimentRealtimeRepeats} label={t(strings.dictationStudio.repeatsLabel)} value={input.realtimeRepeats} min={0} max={20} onChange={(value) => set("realtimeRepeats", value)} />
            {input.realtimeRepeats === 0 ? <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.latencyNotMeasured)}</p> : null}
          </div>
          <NumberField testId={selectors.dictationStudio.experimentLatencyTailSeconds} label={t(strings.dictationStudio.latencyTailLabel)} value={input.latencyTailSeconds} min={0} onChange={(value) => set("latencyTailSeconds", value)} />
          <NumberField testId={selectors.dictationStudio.experimentOverlapMaxWindow} label={t(strings.dictationStudio.maxWindowLabel)} value={input.overlapMaxWindowMs} min={0} onChange={(value) => set("overlapMaxWindowMs", value)} />
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
            <Select
              data-testid={selectors.dictationStudio.experimentSpeakerProfile}
              value={input.speakerTargetProfileId}
              onChange={(event) => set("speakerTargetProfileId", event.currentTarget.value)}
            >
              <option value="">{t(strings.dictationStudio.speakerProfileNone)}</option>
              {(speakerStatus.data?.profiles ?? []).map((profile) => (
                <option key={profile.id} value={profile.id}>
                  {profile.displayName || profile.id} ({profile.clipCount})
                </option>
              ))}
            </Select>
            {speakerStatus.isPending ? <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.speakerProfilesLoading)}</span> : null}
            {speakerStatus.isError ? <span className="text-xs text-app-warning">{t(strings.dictationStudio.speakerProfilesError)}</span> : null}
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
          <NumberField label={t(strings.speakerAdmin.configThreshold)} value={input.speakerThreshold} step={0.05} min={0} max={1} onChange={(value) => set("speakerThreshold", value)} />
          <label className="flex items-center gap-2 text-xs sm:col-span-2">
            <input type="checkbox" checked={input.speakerAblation} onChange={(event) => set("speakerAblation", event.currentTarget.checked)} />
            {t(strings.dictationStudio.speakerAblationLabel)}
          </label>
        </fieldset>

        <details className="rounded-control border border-app-border p-3 lg:col-span-2">
          <summary data-testid={selectors.dictationStudio.experimentAdvanced} className="cursor-pointer text-xs font-medium">
            {t(strings.dictationStudio.advancedLabel)}
          </summary>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <div className="flex flex-col gap-1">
              <NumberField testId={selectors.dictationStudio.experimentChunkMs} label={t(strings.dictationStudio.chunkMsLabel)} value={input.chunkMs} min={0} onChange={(value) => set("chunkMs", value)} />
              <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.chunkMsHint)}</span>
            </div>
            <div className="flex flex-col gap-1">
              <NumberField testId={selectors.dictationStudio.experimentOverlapMaxStall} label={t(strings.dictationStudio.overlapMaxStallLabel)} value={input.overlapMaxStallRejects} min={-1} onChange={(value) => set("overlapMaxStallRejects", value)} />
              <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.overlapMaxStallHint)}</span>
            </div>
            <NumberField testId={selectors.dictationStudio.experimentOverlapWindow} label={t(strings.dictationStudio.overlapWindowLabel)} value={input.overlapWindowMs} min={0} onChange={(value) => set("overlapWindowMs", value)} />
            <NumberField testId={selectors.dictationStudio.experimentOverlapCommitRuns} label={t(strings.dictationStudio.overlapCommitRunsLabel)} value={input.overlapCommitRuns} min={0} onChange={(value) => set("overlapCommitRuns", value)} />
            <NumberField testId={selectors.dictationStudio.experimentVadSilence} label={t(strings.dictationStudio.vadSilenceLabel)} value={input.vadSilenceMs} min={0} onChange={(value) => set("vadSilenceMs", value)} />
            <label className="flex items-center gap-2 text-xs sm:col-span-2">
              <input
                data-testid={selectors.dictationStudio.experimentSpeakerFallback}
                type="checkbox"
                checked={input.speakerFallback}
                onChange={(event) => set("speakerFallback", event.currentTarget.checked)}
              />
              {t(strings.dictationStudio.speakerFallbackLabel)}
            </label>
          </div>
        </details>
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <Button
          type="button"
          data-testid={selectors.dictationStudio.startExperiment}
          disabled={pending || input.strategies.length === 0 || (!input.longForm && input.clipIds.length === 0 && !hasSweepDurations(input))}
          onClick={onStart}
        >
          {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <FlaskConical className="h-4 w-4" aria-hidden="true" />}
          {t(strings.dictationStudio.startExperiment)}
        </Button>
        <span className="text-xs text-app-muted-foreground">
          {!input.longForm && input.clipIds.length === 0 && !hasSweepDurations(input)
            ? t(strings.dictationStudio.startInputRequired)
            : t(strings.dictationStudio.builderSafetyHint)}
        </span>
      </div>
    </Panel>
  );
}

function ClipPicker({ selected, onChange }: { selected: string[]; onChange: (ids: string[]) => void }) {
  const { t } = useTranslation();
  const clips = useQuery({
    queryKey: ["corpus", "clips"],
    queryFn: () => listClips(),
  });
  const all = clips.data ?? [];
  const selectedSet = new Set(selected);

  const toggle = (id: string) =>
    onChange(selectedSet.has(id) ? selected.filter((item) => item !== id) : [...selected, id]);
  const selectAll = () => onChange(all.map((clip) => clip.id));
  const clear = () => onChange([]);

  return (
    <fieldset data-testid={selectors.dictationStudio.clipPicker} className="flex flex-col gap-2 rounded-control border border-app-border p-3 lg:col-span-2">
      <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.clipPickerLabel)}</legend>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.clipPickerHint)}</span>
        <span data-testid={selectors.dictationStudio.clipPickerCount} className="text-xs font-medium text-app-foreground">
          {t(strings.dictationStudio.clipPickerSelected, { selected: selected.length, total: all.length })}
        </span>
      </div>
      {clips.isPending ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.clipPickerLoading)}</p>
      ) : clips.isError ? (
        <ApiErrorState error={clips.error} title={t(strings.dictationStudio.clipPickerError)} onRetry={() => void clips.refetch()} />
      ) : all.length === 0 ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.clipPickerEmpty)}</p>
      ) : (
        <>
          <div className="flex flex-wrap gap-2">
            <Button type="button" size="sm" variant="outline" data-testid={selectors.dictationStudio.clipPickerSelectAll} onClick={selectAll}>
              {t(strings.dictationStudio.clipPickerSelectAll)}
            </Button>
            <Button type="button" size="sm" variant="ghost" data-testid={selectors.dictationStudio.clipPickerClear} onClick={clear}>
              {t(strings.dictationStudio.clipPickerClear)}
            </Button>
          </div>
          <div className="flex max-h-48 flex-col gap-1 overflow-y-auto">
            {all.map((clip) => (
              <ClipPickerRow key={clip.id} clip={clip} checked={selectedSet.has(clip.id)} onToggle={() => toggle(clip.id)} />
            ))}
          </div>
        </>
      )}
    </fieldset>
  );
}

function ClipPickerRow({ clip, checked, onToggle }: { clip: ClipMeta; checked: boolean; onToggle: () => void }) {
  return (
    <label className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 rounded-control px-1 py-1 text-xs hover:bg-app-surface-muted">
      <input
        type="checkbox"
        data-testid={selectors.dictationStudio.clipPick({ id: clip.id })}
        checked={checked}
        onChange={onToggle}
      />
      <span className="max-w-full truncate font-medium text-app-foreground">{clip.referenceText || clip.id}</span>
      <span className="text-app-muted-foreground">
        {clip.id}
        {clip.tags.length > 0 ? ` · ${clip.tags.join(", ")}` : ""}
        {clip.durationMs > 0 ? ` · ${(clip.durationMs / 1000).toFixed(1)}s` : ""}
      </span>
    </label>
  );
}

function NumberField({
  label,
  value,
  onChange,
  testId,
  step = 1,
  min,
  max,
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
  testId?: string;
  step?: number;
  min?: number;
  max?: number;
}) {
  const clamp = (n: number): number => {
    let next = n;
    if (typeof min === "number") next = Math.max(min, next);
    if (typeof max === "number") next = Math.min(max, next);
    return next;
  };
  return (
    <label className="flex flex-col gap-1 text-xs">
      {label}
      <Input
        data-testid={testId}
        type="number"
        step={step}
        min={min}
        max={max}
        value={value}
        onChange={(event) => {
          const raw = event.currentTarget.value;
          // Empty resets to the lower bound (or 0); a non-empty value that
          // does not parse is ignored rather than silently coerced to 0,
          // so invalid keystrokes don't hide the operator's real intent.
          if (raw === "") {
            onChange(clamp(min ?? 0));
            return;
          }
          const parsed = Number(raw);
          if (Number.isNaN(parsed)) return;
          onChange(clamp(parsed));
        }}
      />
    </label>
  );
}

function ExperimentHistory({
  rows,
  pending,
  error,
  selectedId,
  compareSelected,
  onToggleCompare,
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
  compareSelected: string[];
  onToggleCompare: (id: string) => void;
  onSelect: (id: string) => void;
  onWait: (id: string) => void;
  onCancel: (id: string) => void;
  onReport: (id: string) => void;
  onRetry: () => void;
  actionPending: boolean;
}) {
  const { t } = useTranslation();
  const [confirmCancelId, setConfirmCancelId] = useState("");
  if (pending) return <LoadingRows rows={3} label={t(strings.dictationStudio.historyTitle)} />;
  if (error) return <ApiErrorState error={error} title={t(strings.dictationStudio.historyError)} onRetry={onRetry} />;
  if (rows.length === 0) return <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.historyEmpty)}</p>;

  const compareSet = new Set(compareSelected);

  return (
    <div className="overflow-x-auto">
      <Table>
        <THead>
          <TR>
            <TH>{t(strings.dictationStudio.compareSelectHeader)}</TH>
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
                <input
                  type="checkbox"
                  aria-label={t(strings.dictationStudio.compareSelectHeader)}
                  data-testid={selectors.dictationStudio.experimentCompare({ id: row.id })}
                  checked={compareSet.has(row.id)}
                  onChange={() => onToggleCompare(row.id)}
                />
              </TD>
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
                    confirmCancelId === row.id ? (
                      <>
                        <span className="self-center text-xs text-app-muted-foreground">{t(strings.dictationStudio.cancelConfirmPrompt)}</span>
                        <Button
                          type="button"
                          size="sm"
                          variant="destructive"
                          data-testid={selectors.dictationStudio.experimentCancelConfirm({ id: row.id })}
                          disabled={actionPending}
                          onClick={() => {
                            onCancel(row.id);
                            setConfirmCancelId("");
                          }}
                        >
                          {t(strings.dictationStudio.cancelConfirm)}
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          data-testid={selectors.dictationStudio.experimentCancelDismiss({ id: row.id })}
                          onClick={() => setConfirmCancelId("")}
                        >
                          {t(strings.dictationStudio.cancelDismiss)}
                        </Button>
                      </>
                    ) : (
                      <>
                        <Button type="button" size="sm" variant="outline" data-testid={selectors.dictationStudio.experimentWait({ id: row.id })} disabled={actionPending} onClick={() => onWait(row.id)}>
                          {t(strings.dictationStudio.wait)}
                        </Button>
                        <Button type="button" size="sm" variant="ghost" data-testid={selectors.dictationStudio.experimentCancel({ id: row.id })} disabled={actionPending} onClick={() => setConfirmCancelId(row.id)}>
                          {t(strings.dictationStudio.cancel)}
                        </Button>
                      </>
                    )
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

function LiveExperimentProgress({
  event,
  fallbackMessage,
  status,
}: {
  event: ExperimentEventRow | null;
  fallbackMessage: string;
  status: ExperimentRow["status"];
}) {
  const { t } = useTranslation();
  const progress = Math.max(0, Math.min(100, event?.progress ?? (status === "queued" ? 0 : 1)));
  const message = fallbackMessage
    ? t(strings.dictationStudio.liveFallback)
    : event?.message || (status === "queued" ? t(strings.dictationStudio.liveQueued) : t(strings.dictationStudio.liveRunning));

  return (
    <div data-testid={selectors.dictationStudio.experimentLiveProgress} className="mb-4 rounded-control border border-app-border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
        <div className="flex items-center gap-2">
          <StatusBadge status={event?.status ?? status} />
          <span className="font-medium">{message}</span>
        </div>
        <span className="text-xs text-app-muted-foreground">{progress}%</span>
      </div>
      <div
        className="mt-2 h-2 overflow-hidden rounded-full bg-app-surface-muted"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={progress}
      >
        <div className="h-full bg-app-accent" style={{ width: `${progress}%` }} />
      </div>
      {fallbackMessage ? <p className="mt-2 text-xs text-app-muted-foreground">{fallbackMessage}</p> : null}
    </div>
  );
}

function StatusBadge({ status }: { status: ExperimentRow["status"] }) {
  const variant = status === "succeeded" ? "success" : status === "failed" || status === "canceled" ? "danger" : status === "running" ? "info" : "neutral";
  return (
    <Badge variant={variant}>
      <StatusLabel status={status} />
    </Badge>
  );
}

function ExperimentRecipeSummary({ row }: { row: ExperimentRow | null }) {
  const { t } = useTranslation();
  if (!row) return null;
  const r = row.recipe;
  const hasRealizedInput = r.realizedClipIds.length > 0 || r.realizedDurationMs > 0;
  const conditionRows = [
    ...r.augmentationConditions.map((condition) => ({
      id: condition.id,
      title: [condition.kind, condition.source].filter(Boolean).join(" / ") || condition.id,
      skipped: condition.skipped,
      note: condition.note,
    })),
    ...r.speakerConditions.map((condition) => ({
      id: condition.id,
      title: [condition.extraction ? t(strings.dictationStudio.speakerExtractionLabel) : "", condition.verification ? t(strings.dictationStudio.speakerVerificationLabel) : ""]
        .filter(Boolean)
        .join(" / ") || condition.id,
      skipped: condition.skipped,
      note: condition.note,
    })),
  ];
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
        <div>
          {r.longFormEnabled ? `${r.targetDurationSeconds}s · ${r.gapMs}ms` : t(strings.common.dash)}
          {r.sweepDurationsSeconds.length > 0 ? ` · ${r.sweepDurationsSeconds.join("/")}s` : ""}
        </div>
      </div>
      <div>
        <div className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.augmentationLabel)}</div>
        <div>{[...r.noiseTypes, ...r.competingVoiceIds].join(", ") || t(strings.common.dash)}</div>
      </div>
      {hasRealizedInput ? (
        <div className="md:col-span-4 grid gap-2 rounded-control border border-app-border bg-app-surface-muted/40 p-2 text-xs sm:grid-cols-2">
          <div>
            <div className="font-medium text-app-foreground">{t(strings.dictationStudio.realizedClipsLabel)}</div>
            <div className="break-all text-app-muted-foreground">{r.realizedClipIds.join(", ") || t(strings.common.dash)}</div>
          </div>
          <div>
            <div className="font-medium text-app-foreground">{t(strings.dictationStudio.realizedDurationLabel)}</div>
            <div className="text-app-muted-foreground">
              {t(strings.dictationStudio.realizedDurationValue, { seconds: (r.realizedDurationMs / 1000).toFixed(1) })}
            </div>
          </div>
        </div>
      ) : null}
      {conditionRows.length > 0 ? (
        <div data-testid={selectors.dictationStudio.experimentConditions} className="md:col-span-4 rounded-control border border-app-border p-2">
          <div className="text-xs font-semibold">{t(strings.dictationStudio.realizedConditionsTitle)}</div>
          <ul className="mt-2 space-y-1 text-xs text-app-muted-foreground">
            {conditionRows.map((condition) => (
              <li key={condition.id} className="flex flex-wrap items-center gap-2">
                <Badge variant={condition.skipped ? "warning" : "neutral"}>
                  {condition.skipped ? t(strings.dictationStudio.conditionSkipped) : t(strings.dictationStudio.conditionApplied)}
                </Badge>
                <span className="font-medium text-app-foreground">{condition.title}</span>
                {condition.note ? <span>{condition.note}</span> : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {row.error ? <p className="text-xs text-app-danger md:col-span-4">{row.error}</p> : null}
      {r.realizedReference ? (
        <p className="max-h-24 overflow-auto text-xs text-app-muted-foreground md:col-span-4">{r.realizedReference}</p>
      ) : null}
    </div>
  );
}

function recipeDiffLines(rows: ExperimentReportRow[]): string[] {
  if (rows.length < 2) return [];
  const maps = rows.map((row) => recipeFields(row.experiment?.recipe));
  const keys = Array.from(new Set(maps.flatMap((map) => Object.keys(map)))).sort();
  return keys.flatMap((key) => {
    const first = maps[0]?.[key] ?? "";
    if (maps.every((map) => (map[key] ?? "") === first)) return [];
    if (rows.length === 2) return [`${key}: ${valueOrDash(maps[0]?.[key])} -> ${valueOrDash(maps[1]?.[key])}`];
    return [`${key}: ${maps.map((map, index) => `${shortExperimentLabel(rows[index]?.experiment)}=${valueOrDash(map[key])}`).join(", ")}`];
  });
}

function recipeFields(recipe?: ExperimentRow["recipe"]): Record<string, string> {
  if (!recipe) return {};
  const fields: Record<string, string> = {
    clip_ids: recipe.clipIds.join(","),
    realtime_repeats: String(recipe.realtimeRepeats),
    chunk_ms: String(recipe.chunkMs),
    seed: String(recipe.seed),
    latency_tail_seconds: String(recipe.latencyTailSeconds),
    dropped_span_threshold_words: String(recipe.droppedSpanThresholdWords),
    "long_form.enabled": String(recipe.longFormEnabled),
    "long_form.target_duration_seconds": String(recipe.targetDurationSeconds),
    "long_form.gap_ms": String(recipe.gapMs),
    "long_form.tag_contains": recipe.tagContains,
    "long_form.sweep_durations_seconds": recipe.sweepDurationsSeconds.join(","),
    "augmentation.noise_types": recipe.noiseTypes.join(","),
    "augmentation.snr_db": recipe.snrDb.join(","),
    "augmentation.competing_voice_ids": recipe.competingVoiceIds.join(","),
    "augmentation.competing_text": recipe.competingText,
    "speaker.target_profile_id": recipe.speakerTargetProfileId,
    "speaker.extraction_enabled": String(recipe.speakerExtraction),
    "speaker.verification_enabled": String(recipe.speakerVerification),
    "speaker.verification_mode": recipe.speakerMode,
    "speaker.threshold": String(recipe.speakerThreshold),
    "speaker.ablation_enabled": String(recipe.speakerAblation),
  };
  for (const strategy of recipe.strategyDetails ?? []) {
    const key = `strategy.${strategy.kind || strategy.label || "unknown"}`;
    fields[`${key}.overlap_max_window_ms`] = String(strategy.overlapMaxWindowMs);
    fields[`${key}.overlap_max_stall_rejects`] = String(strategy.overlapMaxStallRejects);
    fields[`${key}.overlap_window_ms`] = String(strategy.overlapWindowMs);
    fields[`${key}.overlap_commit_runs`] = String(strategy.overlapCommitRuns);
    fields[`${key}.vad_silence_ms`] = String(strategy.vadSilenceMs);
  }
  return fields;
}

function alignedStrategyKeys(rows: ExperimentReportRow[]): string[] {
  return Array.from(new Set(rows.flatMap((row) => row.report.perStrategy.map(strategyAlignmentKey)))).sort();
}

function strategyMetricCell(row: ExperimentReportRow, key: string): string {
  const strategy = row.report.perStrategy.find((item) => strategyAlignmentKey(item) === key);
  if (!strategy) return "-";
  const p95 = row.report.latencyMeasured ? ms(strategy.finalizationLatencyP95Ms) : "-";
  return `${pct(strategy.wer)} / ${p95}`;
}

function strategyAlignmentKey(strategy: ExperimentReportRow["report"]["perStrategy"][number]): string {
  return strategy.strategy.split("/")[0] || strategy.label.split("/")[0] || "-";
}

function shortExperimentLabel(row: ExperimentRow | null | undefined): string {
  return row?.name || row?.id || "-";
}

function valueOrDash(value: string | undefined): string {
  return value && value.trim() ? value : "-";
}

function CompareResults({ rows }: { rows: ExperimentReportRow[] }) {
  const { t } = useTranslation();
  const recipeDiffs = recipeDiffLines(rows);
  const strategyKeys = alignedStrategyKeys(rows);
  return (
    <div data-testid={selectors.dictationStudio.compareResults} className="space-y-4 overflow-x-auto">
      <div className="min-w-full">
        <Table>
          <THead>
            <TR>
              <TH>{t(strings.dictationStudio.colName)}</TH>
              <TH>{t(strings.dictationStudio.colStatus)}</TH>
              <TH>{t(strings.dictationStudio.compareColWinner)}</TH>
              <TH>{t(strings.dictationStudio.colWer)}</TH>
              <TH>{t(strings.dictationStudio.compareColP95)}</TH>
              <TH>{t(strings.dictationStudio.compareColSafety)}</TH>
            </TR>
          </THead>
          <TBody>
            {rows.map((row) => {
              const winner = row.report.summary?.winnerStrategy;
              const strategy = row.report.perStrategy.find((item) => item.strategy === winner) ?? row.report.perStrategy[0];
              const latency = row.report.latencyMeasured;
              const quality = row.report.qualityMeasured;
              return (
                <TR key={row.experiment?.id ?? row.report.summary?.recommendation ?? "compare-row"}>
                  <TD>{row.experiment?.name ?? row.experiment?.id ?? t(strings.common.dash)}</TD>
                  <TD>{row.experiment ? <StatusBadge status={row.experiment.status} /> : t(strings.common.dash)}</TD>
                  <TD>{strategy ? <StrategyName kind={strategy.strategy} /> : t(strings.common.dash)}</TD>
                  <TD>{strategy && quality ? pct(strategy.wer) : t(strings.common.dash)}</TD>
                  <TD>{strategy && latency ? ms(strategy.finalizationLatencyP95Ms) : t(strings.common.dash)}</TD>
                  <TD>
                    {strategy?.safety ? (
                      <span className={strategy.safety.passed ? "text-app-success" : "text-app-danger"}>
                        {strategy.safety.passed ? t(strings.dictationStudio.safetySafe) : t(strings.dictationStudio.safetyUnsafe)}
                      </span>
                    ) : (
                      t(strings.dictationStudio.safetyMeasured)
                    )}
                  </TD>
                </TR>
              );
            })}
          </TBody>
        </Table>
      </div>
      {recipeDiffs.length > 0 ? (
        <div className="rounded-control border border-app-border p-3">
          <h3 className="text-sm font-semibold">{t(strings.dictationStudio.compareRecipeDiffTitle)}</h3>
          <ul className="mt-2 space-y-1 text-xs text-app-muted-foreground">
            {recipeDiffs.map((line) => (
              <li key={line} className="break-all">
                {line}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {strategyKeys.length > 0 ? (
        <div>
          <h3 className="mb-2 text-sm font-semibold">{t(strings.dictationStudio.compareStrategyAlignmentTitle)}</h3>
          <Table>
            <THead>
              <TR>
                <TH>{t(strings.dictationStudio.colStrategy)}</TH>
                {rows.map((row) => (
                  <TH key={row.experiment?.id ?? shortExperimentLabel(row.experiment)}>{shortExperimentLabel(row.experiment)}</TH>
                ))}
              </TR>
            </THead>
            <TBody>
              {strategyKeys.map((key) => (
                <TR key={key}>
                  <TD>
                    <StrategyName kind={key} />
                  </TD>
                  {rows.map((row) => (
                    <TD key={`${key}-${row.experiment?.id ?? shortExperimentLabel(row.experiment)}`}>{strategyMetricCell(row, key)}</TD>
                  ))}
                </TR>
              ))}
            </TBody>
          </Table>
        </div>
      ) : null}
    </div>
  );
}
