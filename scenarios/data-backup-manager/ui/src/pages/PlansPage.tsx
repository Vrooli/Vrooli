import { useState } from "react";
import { CalendarClock, Pencil, Play, Plus, Trash2 } from "lucide-react";

import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/EmptyState";
import { AsyncSection } from "../components/AsyncSection";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { Button } from "../components/ui/button";
import { StatusChip } from "../components/ui/status-chip";
import { PlanFormDialog } from "../features/plans/PlanFormDialog";
import { CoverageBanner } from "../features/backup-coverage/CoverageBanner";
import { usePlans, useDeletePlan } from "../hooks/usePlans";
import { useRuns, useTriggerRun } from "../hooks/useRuns";
import type { Plan } from "../api/plans";
import { runStatusMeta } from "../lib/status";
import { RUN_STATUS_STRINGS } from "../consts/statusStrings";
import { formatAge } from "../lib/format";
import { tsToDate } from "../lib/proto";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/** One plan row: cadence + binding summary, the latest run outcome (a small
 * per-plan query), and the run-now / edit / delete actions. */
function PlanRow({ plan, onEdit, onDelete }: { plan: Plan; onEdit: () => void; onDelete: () => void }) {
  const { t } = useTranslation();
  const sharedRiskWarnings = Array.isArray(plan.sharedRiskWarnings) ? plan.sharedRiskWarnings : [];
  const { data } = useRuns(plan.id);
  const trigger = useTriggerRun();
  const latest = data?.[0];
  const latestMeta = latest ? runStatusMeta(latest.status) : undefined;

  return (
    <li
      data-testid={selectors.plans.row({ id: plan.id })}
      className="flex flex-wrap items-center gap-x-3 gap-y-2 p-3"
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-medium text-app-foreground">{plan.name}</span>
          {plan.enabled && <StatusChip tone="success" labelKey={strings.plans.enabled} />}
        </div>
        <p className="flex flex-wrap items-center gap-x-2 text-xs text-app-muted-foreground">
          <span className="font-mono">{plan.schedule}</span>
          <span>
            {t(strings.plans.summary, {
              targets: plan.targetIds.length,
              destinations: plan.destinationIds.length,
              keep: plan.retention?.keepLatest ?? 0,
            })}
          </span>
        </p>
        <p className="text-xs text-app-muted-foreground">
          {t(strings.plans.recoveryDrillSchedule)}: {plan.recoveryDrillSchedule || t(strings.drills.manual)}
        </p>
        {sharedRiskWarnings.map((warning) => (
          <p key={warning} role="status" className="text-xs text-app-warning">
            {t(strings.plans.sharedRiskWarning, { warning })}
          </p>
        ))}
      </div>

      <div className="flex items-center gap-2 text-xs text-app-muted-foreground">
        {latestMeta ? (
          <>
            <StatusChip tone={latestMeta.tone} labelKey={RUN_STATUS_STRINGS[latestMeta.slug]} />
            <span>{formatAge(tsToDate(latest?.startedAt), t(strings.common.never))}</span>
          </>
        ) : (
          <span>{t(strings.plans.noRuns)}</span>
        )}
      </div>

      <div className="flex items-center gap-2">
        <Button
          size="sm"
          data-testid={selectors.plans.runNowButton}
          disabled={trigger.isPending}
          onClick={() => trigger.mutate(plan.id)}
        >
          <Play aria-hidden="true" className="me-1.5 h-4 w-4" />
          {t(strings.plans.runNow)}
        </Button>
        <Button variant="outline" size="sm" aria-label={t(strings.plans.editTitle)} onClick={onEdit}>
          <Pencil aria-hidden="true" className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="sm"
          aria-label={t(strings.plans.delete)}
          data-testid={selectors.plans.deleteButton}
          onClick={onDelete}
        >
          <Trash2 aria-hidden="true" className="h-4 w-4" />
        </Button>
      </div>
    </li>
  );
}

/**
 * Plans surface — author the target↔destination bindings the scheduler runs,
 * and trigger on-demand runs. Run-now is the prominent per-row action; editing
 * opens the binding builder.
 */
export function PlansPage() {
  const { t } = useTranslation();
  const { data, isLoading, isError, refetch } = usePlans();
  const del = useDeletePlan();
  const plans = data ?? [];

  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Plan | null>(null);
  const [deleting, setDeleting] = useState<Plan | null>(null);

  const confirmDelete = () => {
    if (!deleting) return;
    del.mutate(deleting.id, { onSuccess: () => setDeleting(null) });
  };

  return (
    <section data-testid={selectors.pages.plans} aria-labelledby="plans-heading" className="flex flex-col gap-6">
      <div id="plans-heading">
        <PageHeader
          title={t(strings.layout.nav.plans)}
          subtitle={t(strings.plans.subtitle)}
          actions={
            <Button size="sm" data-testid={selectors.plans.createButton} onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden="true" className="me-1.5 h-4 w-4" />
              {t(strings.plans.create)}
            </Button>
          }
        />
      </div>

      <CoverageBanner />

      <AsyncSection
        isLoading={isLoading}
        isError={isError}
        isEmpty={plans.length === 0}
        onRetry={() => void refetch()}
        emptyState={
          <EmptyState
            icon={CalendarClock}
            title={t(strings.plans.empty)}
            description={t(strings.plans.emptyHint)}
            action={
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                {t(strings.plans.create)}
              </Button>
            }
          />
        }
      >
        <ul data-testid={selectors.plans.list} className="flex flex-col divide-y divide-app-border rounded-panel border border-app-border bg-app-surface">
          {plans.map((plan) => (
            <PlanRow key={plan.id} plan={plan} onEdit={() => setEditing(plan)} onDelete={() => setDeleting(plan)} />
          ))}
        </ul>
      </AsyncSection>

      <PlanFormDialog open={createOpen} onClose={() => setCreateOpen(false)} />
      {editing && <PlanFormDialog key={editing.id} open plan={editing} onClose={() => setEditing(null)} />}

      <ConfirmDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={confirmDelete}
        title={t(strings.plans.deleteTitle)}
        body={t(strings.plans.deleteBody, { name: deleting?.name ?? "" })}
        confirmLabel={t(strings.plans.delete)}
        danger
        busy={del.isPending}
      />
    </section>
  );
}

export default PlansPage;
