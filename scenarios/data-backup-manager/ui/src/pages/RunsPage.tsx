import { useState } from "react";
import { ChevronDown, ChevronRight, History } from "lucide-react";

import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/EmptyState";
import { AsyncSection } from "../components/AsyncSection";
import { StatusChip } from "../components/ui/status-chip";
import { usePlans } from "../hooks/usePlans";
import { useRuns } from "../hooks/useRuns";
import type { Run } from "../api/runs";
import { outcomeMeta, runStatusMeta, triggerSlug } from "../lib/status";
import { OUTCOME_STRINGS, RUN_STATUS_STRINGS, TRIGGER_STRINGS } from "../consts/statusStrings";
import { formatAge, formatBytes, formatDuration } from "../lib/format";
import { tsToDate } from "../lib/proto";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

function RunRow({ run, planName }: { run: Run; planName: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const meta = runStatusMeta(run.status);
  const Chevron = open ? ChevronDown : ChevronRight;

  return (
    <li data-testid={selectors.runs.row({ id: run.id })} className="flex flex-col">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex flex-wrap items-center gap-x-3 gap-y-2 p-3 text-start hover:bg-app-surface-muted"
      >
        <Chevron aria-hidden="true" className="h-4 w-4 shrink-0 text-app-muted-foreground" />
        <span className="min-w-0 flex-1 truncate text-sm font-medium text-app-foreground">{planName}</span>
        <span className="text-xs text-app-muted-foreground">
          {t(TRIGGER_STRINGS[triggerSlug(run.trigger)])}
        </span>
        <StatusChip tone={meta.tone} labelKey={RUN_STATUS_STRINGS[meta.slug]} />
        <span className="text-xs text-app-muted-foreground">
          {formatAge(tsToDate(run.startedAt), t(strings.common.never))}
        </span>
        <span className="text-xs text-app-muted-foreground">
          {formatDuration(tsToDate(run.startedAt), tsToDate(run.finishedAt))}
        </span>
      </button>

      {open && (
        <div className="border-t border-app-border bg-app-surface-muted/40 px-3 py-2">
          {(run.failureCode || run.nextAction || run.preflightIncidents.length > 0) && (
            <div className="mb-3 rounded-panel border border-app-danger/40 bg-app-danger/5 p-3 text-xs">
              {run.failureCode && (
                <p className="font-mono font-semibold text-app-danger" data-testid="run-failure-code">
                  {run.failureCode}
                  {run.failureCategory ? ` (${run.failureCategory})` : ""}
                </p>
              )}
              {run.nextAction && (
                <p className="mt-1 text-app-foreground" data-testid="run-next-action">
                  {t(strings.runs.nextAction)}: {run.nextAction}
                </p>
              )}
              {run.preflightIncidents.map((incident) => (
                <p key={`${incident.code}-${incident.scope}`} className="mt-1 text-app-danger">
                  {incident.code}: {incident.message}
                  {incident.nextAction ? ` — ${incident.nextAction}` : ""}
                </p>
              ))}
            </div>
          )}
          <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
            {t(strings.runs.outcomesHeading)}
          </p>
          <ul className="flex flex-col divide-y divide-app-border">
            {run.outcomes.map((o) => {
              const om = outcomeMeta(o.status);
              return (
                <li
                  key={`${o.targetId}-${o.destinationId}`}
                  data-testid={selectors.runs.outcomeRow({ targetId: o.targetId })}
                  className="flex flex-wrap items-center gap-x-3 gap-y-1 py-2 text-xs"
                >
                  <span className="min-w-0 flex-1 truncate font-mono text-app-foreground">{o.targetId}</span>
                  <StatusChip tone={om.tone} labelKey={OUTCOME_STRINGS[om.slug]} />
                  <span className="text-app-muted-foreground">{formatBytes(o.bytes)}</span>
                  {o.snapshotId && <span className="truncate font-mono text-app-muted-foreground">{o.snapshotId}</span>}
                  {o.error && <span className="w-full break-words text-app-danger">{o.error}</span>}
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </li>
  );
}

/**
 * Runs surface — plan-execution history. A run that partially failed renders as
 * an amber "partial failure" with a per-target breakdown when expanded
 * (cap-blocked targets are distinct from failures), never a flat red "failed".
 * In-flight runs refresh live via the polling in useRuns.
 */
export function RunsPage() {
  const { t } = useTranslation();
  const { data, isLoading, isError, refetch } = useRuns();
  const plans = usePlans();
  const runs = data ?? [];
  const planName = (id: string) => plans.data?.find((p) => p.id === id)?.name ?? id;

  return (
    <section data-testid={selectors.pages.runs} aria-labelledby="runs-heading" className="flex flex-col gap-6">
      <div id="runs-heading">
        <PageHeader title={t(strings.layout.nav.runs)} subtitle={t(strings.runs.subtitle)} />
      </div>

      <AsyncSection
        isLoading={isLoading}
        isError={isError}
        isEmpty={runs.length === 0}
        onRetry={() => void refetch()}
        emptyState={
          <EmptyState icon={History} title={t(strings.runs.empty)} description={t(strings.runs.emptyHint)} />
        }
      >
        <ul data-testid={selectors.runs.table} className="flex flex-col divide-y divide-app-border rounded-panel border border-app-border bg-app-surface">
          {runs.map((run) => (
            <RunRow key={run.id} run={run} planName={planName(run.planId)} />
          ))}
        </ul>
      </AsyncSection>
    </section>
  );
}

export default RunsPage;
