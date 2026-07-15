import { useQuery } from "@tanstack/react-query";
import { AlertCircle, GitCompare, ListChecks, Package } from "lucide-react";
import type { ReactNode } from "react";
import { Link, useParams } from "react-router-dom";

import { fetchTemplateDetail } from "../api/templateDomain";
import { DefinitionList } from "../components/detail/DefinitionList";
import { DetailError, DetailLoading } from "../components/detail/DetailStates";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailSection } from "../components/detail/DetailSection";
import { EmptyState } from "../components/ui/empty-state";
import { StatusBadge } from "../components/ui/status-badge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import {
  debtStatusTone,
  driftTone,
  kindLabel,
  modeLabel,
  runStatusTone,
} from "../lib/templateLabels";
import { formatTimestamp } from "../lib/time";

const TITLE_ID = "template-detail-heading";

export function TemplateDetailPage() {
  const { t } = useTranslation();
  const { templateId = "" } = useParams();
  const id = decodeURIComponent(templateId);
  const query = useQuery({
    queryKey: ["template-detail", id] as const,
    queryFn: () => fetchTemplateDetail(id),
  });

  if (query.isLoading) {
    return <DetailLoading testId={selectors.templateDetail.loading} />;
  }
  if (query.isError || !query.data) {
    return <DetailError testId={selectors.templateDetail.error} />;
  }

  const { template, runs, drift, debt } = query.data;
  const lag = template.versionLag?.lagCount ?? 0;

  return (
    <section
      data-testid={selectors.pages.templateDetail}
      aria-labelledby={TITLE_ID}
      className="flex flex-col gap-4"
    >
      <DetailPageHeader
        testId={selectors.templateDetail.header}
        backTo="/"
        backLabel={t(strings.detail.back.dashboard)}
        title={template.displayName || template.id}
        titleId={TITLE_ID}
        status={{ label: template.status || "—", tone: lag > 0 ? "warning" : "success" }}
        subtitle={`${kindLabel(template.kind)} · ${template.id}`}
      />

      <DetailSection
        testId={selectors.templateDetail.overview}
        title={t(strings.templateDetail.overview)}
        icon={<Package aria-hidden className="h-4 w-4" />}
        hideDivider
      >
        <DefinitionList
          items={[
            { label: t(strings.templateDetail.fields.kind), value: kindLabel(template.kind) },
            { label: t(strings.templateDetail.fields.version), value: template.version },
            { label: t(strings.templateDetail.fields.status), value: template.status },
            {
              label: t(strings.templateDetail.fields.latestVersion),
              value: template.versionLag?.latestVersion || template.version,
            },
            { label: t(strings.templateDetail.fields.lag), value: String(lag) },
            { label: t(strings.templateDetail.fields.updatedAt), value: formatTimestamp(template.updatedAt) },
            {
              label: t(strings.templateDetail.fields.tags),
              value: template.tags.length > 0 ? template.tags.join(", ") : "",
              full: true,
            },
            { label: t(strings.templateDetail.fields.manifestPath), value: template.manifestPath, full: true },
            { label: t(strings.templateDetail.fields.sourcePath), value: template.sourcePath, full: true },
          ]}
        />
      </DetailSection>

      <DetailSection
        testId={selectors.templateDetail.runs}
        title={t(strings.templateDetail.runs)}
        icon={<ListChecks aria-hidden className="h-4 w-4" />}
        storageKey={`template-runs-${id}`}
        defaultOpen={false}
        headerAction={<StatusBadge tone="neutral">{String(runs.length)}</StatusBadge>}
      >
        {runs.length === 0 ? (
          <EmptyState title={t(strings.templateDetail.runsEmpty)} />
        ) : (
          <ul className="grid gap-2">
            {runs.map((run) => (
              <LinkRow
                key={run.id}
                to={`/runs/${encodeURIComponent(run.id)}`}
                testId={selectors.templateDetail.runLink({ id: run.id })}
                primary={run.id}
                secondary={`${modeLabel(run.mode)} · ${run.findings.length} · ${run.trigger || "—"}`}
                badge={run.status}
                tone={runStatusTone(run.status)}
              />
            ))}
          </ul>
        )}
      </DetailSection>

      <DetailSection
        testId={selectors.templateDetail.drift}
        title={t(strings.templateDetail.drift)}
        icon={<GitCompare aria-hidden className="h-4 w-4" />}
        storageKey={`template-drift-${id}`}
        defaultOpen={false}
        headerAction={<StatusBadge tone="neutral">{String(drift.length)}</StatusBadge>}
      >
        {drift.length === 0 ? (
          <EmptyState title={t(strings.templateDetail.driftEmpty)} />
        ) : (
          <ul className="grid gap-2">
            {drift.map((snapshot) => (
              <li
                key={snapshot.id}
                className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-panel border border-app-border px-3 py-2"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{snapshot.target}</p>
                  <p className="truncate text-xs text-app-muted-foreground">
                    {snapshot.driftCount} · {formatTimestamp(snapshot.capturedAt)}
                  </p>
                </div>
                <StatusBadge tone={driftTone(snapshot.driftCount)}>{snapshot.status}</StatusBadge>
              </li>
            ))}
          </ul>
        )}
      </DetailSection>

      <DetailSection
        testId={selectors.templateDetail.debt}
        title={t(strings.templateDetail.debt)}
        icon={<AlertCircle aria-hidden className="h-4 w-4" />}
        storageKey={`template-debt-${id}`}
        defaultOpen={false}
        headerAction={<StatusBadge tone={debt.length > 0 ? "warning" : "success"}>{String(debt.length)}</StatusBadge>}
      >
        {debt.length === 0 ? (
          <EmptyState title={t(strings.templateDetail.debtEmpty)} />
        ) : (
          <ul className="grid gap-2">
            {debt.map((entry) => (
              <LinkRow
                key={entry.key}
                to={`/debt/${encodeURIComponent(entry.key)}`}
                testId={selectors.templateDetail.debtLink({ key: entry.key })}
                primary={entry.title || entry.key}
                secondary={`${entry.key} · ${entry.severity}`}
                badge={entry.status}
                tone={debtStatusTone(entry.status)}
              />
            ))}
          </ul>
        )}
      </DetailSection>
    </section>
  );
}

function LinkRow({
  to,
  testId,
  primary,
  secondary,
  badge,
  tone,
}: {
  to: string;
  testId: string;
  primary: string;
  secondary: string;
  badge: string;
  tone: "success" | "warning" | "danger" | "info" | "neutral";
}): ReactNode {
  return (
    <li className="min-w-0">
      <Link
        to={to}
        data-testid={testId}
        className="grid min-h-11 min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-panel border border-app-border px-3 py-2 transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
      >
        <span className="min-w-0">
          <span className="block truncate text-sm font-medium">{primary}</span>
          <span className="block truncate text-xs text-app-muted-foreground">{secondary}</span>
        </span>
        <StatusBadge tone={tone}>{badge}</StatusBadge>
      </Link>
    </li>
  );
}
