import { useQuery } from "@tanstack/react-query";
import { AlertCircle, ListChecks, Zap } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { fetchValidationRun } from "../api/templateDomain";
import { DefinitionList } from "../components/detail/DefinitionList";
import { DetailError, DetailLoading } from "../components/detail/DetailStates";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailSection } from "../components/detail/DetailSection";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.1.0";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { modeLabel, runStatusTone, severityTone } from "../lib/templateLabels";
import { formatDuration, formatTimestamp } from "../lib/time";

const TITLE_ID = "run-detail-heading";

export function ValidationRunDetailPage() {
  const { t } = useTranslation();
  const { runId = "" } = useParams();
  const id = decodeURIComponent(runId);
  const query = useQuery({
    queryKey: ["validation-run", id] as const,
    queryFn: () => fetchValidationRun(id),
  });

  if (query.isLoading) {
    return <DetailLoading testId={selectors.runDetail.loading} />;
  }
  if (query.isError || !query.data) {
    return <DetailError testId={selectors.runDetail.error} />;
  }

  const run = query.data;

  return (
    <section
      data-testid={selectors.pages.runDetail}
      aria-labelledby={TITLE_ID}
      className="flex flex-col gap-4"
    >
      <DetailPageHeader
        testId={selectors.runDetail.header}
        backTo="/"
        backLabel={t(strings.detail.back.dashboard)}
        title={run.id}
        titleId={TITLE_ID}
        status={{ label: run.status || "—", tone: runStatusTone(run.status) }}
        subtitle={
          <Link to={`/templates/${encodeURIComponent(run.templateId)}`} className="underline-offset-2 hover:underline">
            {run.templateId}
          </Link>
        }
      />

      <DetailSection
        testId={selectors.runDetail.overview}
        title={t(strings.runDetail.overview)}
        icon={<Zap aria-hidden className="h-4 w-4" />}
        hideDivider
      >
        <DefinitionList
          items={[
            {
              label: t(strings.runDetail.fields.template),
              value: (
                <Link to={`/templates/${encodeURIComponent(run.templateId)}`} className="underline-offset-2 hover:underline">
                  {run.templateId}
                </Link>
              ),
            },
            { label: t(strings.runDetail.fields.mode), value: modeLabel(run.mode) },
            { label: t(strings.runDetail.fields.status), value: run.status },
            { label: t(strings.runDetail.fields.trigger), value: run.trigger },
            { label: t(strings.runDetail.fields.target), value: run.target },
            { label: t(strings.runDetail.fields.startedAt), value: formatTimestamp(run.startedAt) },
            { label: t(strings.runDetail.fields.finishedAt), value: formatTimestamp(run.finishedAt) },
            { label: t(strings.runDetail.fields.duration), value: formatDuration(run.startedAt, run.finishedAt) },
          ]}
        />
      </DetailSection>

      <DetailSection
        testId={selectors.runDetail.phases}
        title={t(strings.runDetail.phases)}
        icon={<ListChecks aria-hidden className="h-4 w-4" />}
      >
        {run.phaseResults.length === 0 ? (
          <EmptyState title={t(strings.runDetail.phasesEmpty)} />
        ) : (
          <ul className="grid gap-2">
            {run.phaseResults.map((phase) => (
              <li
                key={phase.phase}
                className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 rounded-panel border border-app-border px-3 py-2"
              >
                <span className="min-w-0 truncate text-sm font-medium">{phase.phase}</span>
                <span className="tabular-nums text-xs text-app-muted-foreground">
                  {phase.findingCount} {t(strings.runDetail.columns.findingCount)}
                </span>
                <StatusBadge tone={runStatusTone(phase.status)}>{phase.status}</StatusBadge>
              </li>
            ))}
          </ul>
        )}
      </DetailSection>

      <DetailSection
        testId={selectors.runDetail.findings}
        title={t(strings.runDetail.findings)}
        icon={<AlertCircle aria-hidden className="h-4 w-4" />}
        storageKey="run-findings"
        defaultOpen={false}
        headerAction={
          <StatusBadge tone={run.findings.length > 0 ? "warning" : "success"}>
            {String(run.findings.length)}
          </StatusBadge>
        }
      >
        {run.findings.length === 0 ? (
          <EmptyState title={t(strings.runDetail.findingsEmpty)} />
        ) : (
          <ul className="grid gap-2">
            {run.findings.map((finding) => (
              <li key={finding.key} className="min-w-0 rounded-panel border border-app-border px-3 py-2">
                <div className="flex min-w-0 flex-wrap items-center justify-between gap-2">
                  <span className="min-w-0 break-words text-sm font-semibold">{finding.key}</span>
                  <StatusBadge tone={severityTone(finding.severity)}>{finding.severity}</StatusBadge>
                </div>
                {finding.summary && <p className="mt-1 text-sm text-app-muted-foreground">{finding.summary}</p>}
                {finding.source && (
                  <p className="mt-1 text-xs text-app-muted-foreground">
                    {t(strings.runDetail.columns.source)}: {finding.source}
                  </p>
                )}
              </li>
            ))}
          </ul>
        )}
      </DetailSection>
    </section>
  );
}
