import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { runEval, type EvalReportData } from "../../services/corpus";

const DASH = "—";

function pct(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}
function ratio(value: number): string {
  return value.toFixed(2);
}
function seconds(value: number): string {
  return value.toFixed(1);
}
function ms(value: number): string {
  return String(Math.round(value));
}

// EvalReportView replays the corpus through every strategy and renders the
// quality-vs-latency comparison table. Latency columns degrade to a dash
// when the run did not pace clips in real time (latency_measured = false).
export function EvalReportView() {
  const { t } = useTranslation();
  const [repeats, setRepeats] = useState(3);

  const run = useMutation({
    mutationFn: () => runEval({ realtimeRepeats: repeats }),
  });

  const report = run.data;

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.reportHint)}</p>

      <div className="flex flex-wrap items-end gap-3">
        <label htmlFor="eval-repeats" className="flex flex-col gap-1 text-xs">
          {t(strings.dictationStudio.repeatsLabel)}
          <Input
            id="eval-repeats"
            data-testid={selectors.dictationStudio.repeatsInput}
            type="number"
            min={0}
            max={20}
            className="w-28"
            value={repeats}
            onChange={(e) => setRepeats(Math.max(0, Math.min(20, Number(e.currentTarget.value) || 0)))}
          />
        </label>
        <Button
          type="button"
          data-testid={selectors.dictationStudio.runEval}
          disabled={run.isPending}
          onClick={() => run.mutate()}
        >
          {run.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              {t(strings.dictationStudio.running)}
            </>
          ) : (
            t(strings.dictationStudio.runEval)
          )}
        </Button>
      </div>
      <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.repeatsHint)}</p>

      {run.isError ? (
        <ApiErrorState error={run.error} title={t(strings.dictationStudio.reportError)} onRetry={() => run.mutate()} />
      ) : null}

      {report ? (
        <ReportTable report={report} />
      ) : !run.isError ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.reportEmpty)}</p>
      ) : null}
    </div>
  );
}

function ReportTable({ report }: { report: EvalReportData }) {
  const { t } = useTranslation();
  const latency = report.latencyMeasured;
  return (
    <div className="flex flex-col gap-2">
      {!report.qualityMeasured ? (
        <p className="text-xs text-app-warning">{t(strings.dictationStudio.qualityNotMeasured)}</p>
      ) : null}
      {!latency ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.latencyNotMeasured)}</p>
      ) : null}
      <Table data-testid={selectors.dictationStudio.evalTable}>
        <THead>
          <TR>
            <TH>{t(strings.dictationStudio.colStrategy)}</TH>
            <TH>{t(strings.dictationStudio.colWer)}</TH>
            <TH>{t(strings.dictationStudio.colRtf)}</TH>
            <TH>{t(strings.dictationStudio.colWhisperCalls)}</TH>
            <TH>{t(strings.dictationStudio.colAudioSeconds)}</TH>
            <TH>{t(strings.dictationStudio.colP50)}</TH>
            <TH>{t(strings.dictationStudio.colP95)}</TH>
            <TH>{t(strings.dictationStudio.colRevisions)}</TH>
          </TR>
        </THead>
        <TBody>
          {report.perStrategy.map((row) => (
            <TR key={row.strategy} data-testid={selectors.dictationStudio.evalRow({ strategy: row.strategy })}>
              <TD className="font-medium">{row.label}</TD>
              <TD>{report.qualityMeasured ? pct(row.wer) : DASH}</TD>
              <TD>{ratio(row.rtf)}</TD>
              <TD>{String(row.whisperCalls)}</TD>
              <TD>{seconds(row.whisperAudioSeconds)}</TD>
              <TD>{latency ? ms(row.finalizationLatencyP50Ms) : DASH}</TD>
              <TD>{latency ? ms(row.finalizationLatencyP95Ms) : DASH}</TD>
              <TD>{String(row.partialRevisions)}</TD>
            </TR>
          ))}
        </TBody>
      </Table>
    </div>
  );
}
