import { FlaskConical, Loader2 } from "lucide-react";

import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { type ExperimentEventRow, type ExperimentReportRow, type ExperimentRow } from "../../../services/experiment";
import { EvalReportTable } from "../EvalReportTable";
import { StatusBadge } from "../ExperimentLabShared";

export function ExperimentResults({
  report,
  selected,
  loadReportPending,
  onLoadReport,
}: {
  report: ExperimentReportRow | null;
  selected: ExperimentRow | null | undefined;
  loadReportPending: boolean;
  onLoadReport: (id: string) => void;
}) {
  const { t } = useTranslation();
  if (report) {
    return (
      <div data-testid={selectors.dictationStudio.experimentResults} className="flex flex-col gap-4">
        <ExperimentRecipeSummary row={report.experiment} />
        <EvalReportTable report={report.report} />
      </div>
    );
  }
  if (selected) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-app-muted-foreground">
        <span>{t(strings.dictationStudio.resultsSelectHint)}</span>
        <Button type="button" onClick={() => onLoadReport(selected.id)} disabled={loadReportPending}>
          {loadReportPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <FlaskConical className="h-4 w-4" aria-hidden="true" />}
          {t(strings.dictationStudio.loadReport)}
        </Button>
      </div>
    );
  }
  return <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.resultsEmpty)}</p>;
}

export function LiveExperimentProgress({
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
