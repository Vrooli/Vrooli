import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, RefreshCw } from "lucide-react";

import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { Button } from "../../../components/ui/button";
import { Panel } from "../../../components/ui/panel";
import { pushToast } from "../../../components/ui/toast";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { listClips } from "../../../services/corpus";
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
} from "../../../services/experiment";
import { getSpeakerStatus } from "../../../services/speakerAdmin";
import { CompareResults } from "../CompareResults";
import { ExperimentBuilder } from "../ExperimentBuilder";
import { ExperimentHistory } from "../ExperimentHistory";
import { defaultInput, isTerminal } from "../ExperimentLabFormat";
import { ExperimentResults, LiveExperimentProgress } from "../ExperimentResults";
import { useExperimentStream } from "../useExperimentStream";

export function ExperimentLabView() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [input, setInput] = useState<StartExperimentInput>(() => defaultInput());
  const [selectedId, setSelectedId] = useState("");
  const [activeExperiment, setActiveExperiment] = useState<ExperimentRow | null>(null);
  const [report, setReport] = useState<ExperimentReportRow | null>(null);
  const [compareSelected, setCompareSelected] = useState<string[]>([]);
  const [compareRows, setCompareRows] = useState<ExperimentReportRow[]>([]);

  const history = useQuery({
    queryKey: ["experiments", "list"],
    queryFn: () => listExperiments(),
  });
  const clips = useQuery({
    queryKey: ["corpus", "clips"],
    queryFn: () => listClips(),
  });
  const speakerStatus = useQuery({
    queryKey: ["speaker", "status"],
    queryFn: getSpeakerStatus,
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

  const selected = useMemo(
    () => history.data?.find((row) => row.id === selectedId) ?? report?.experiment ?? activeExperiment,
    [activeExperiment, history.data, report, selectedId],
  );
  const selectedExperimentId = selected?.id ?? "";
  const selectedExperimentStatus = selected?.status ?? "unspecified";

  const { liveEvent, setLiveEvent, streamError, setStreamError } = useExperimentStream({
    selectedExperimentId,
    selectedExperimentStatus,
    selectedId,
    queryClient: qc,
    loadReport: loadReport.mutate,
    messages: {
      complete: t(strings.dictationStudio.liveComplete),
      polling: t(strings.dictationStudio.livePolling),
      streamClosed: t(strings.dictationStudio.liveStreamClosed),
    },
    setActiveExperiment,
  });

  const toggleCompare = (id: string) =>
    setCompareSelected((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id],
    );

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.08fr)_minmax(360px,0.92fr)]">
      <ExperimentBuilder
        input={input}
        setInput={setInput}
        pending={start.isPending}
        onStart={() => start.mutate()}
        clips={clips}
        speakerStatus={speakerStatus}
      />

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
          <LiveExperimentProgress event={liveEvent} fallbackMessage={streamError} status={selected.status} />
        ) : null}
        <ExperimentResults
          report={report}
          selected={selected}
          loadReportPending={loadReport.isPending}
          onLoadReport={(id) => loadReport.mutate(id)}
        />
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
