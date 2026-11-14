import { ShieldCheck } from "lucide-react";

import { AsyncSection } from "../components/AsyncSection";
import { EmptyState } from "../components/EmptyState";
import { PageHeader } from "../components/PageHeader";
import { Button } from "../components/ui/button";
import { StatusChip } from "../components/ui/status-chip";
import { useDrills, useRunDrill } from "../hooks/useDrills";
import { usePlans } from "../hooks/usePlans";
import { DrillStatus } from "../api/drills";
import { tsToDate } from "../lib/proto";
import { formatAge } from "../lib/format";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import type { StringKey } from "../consts/strings";
import { useTranslation } from "../i18n";

function drillStatus(status: DrillStatus): { labelKey: StringKey; tone: "neutral" | "info" | "success" | "danger" } {
  switch (status) {
    case DrillStatus.VERIFIED: return { labelKey: strings.status.drill.verified, tone: "success" };
    case DrillStatus.FAILED: return { labelKey: strings.status.drill.failed, tone: "danger" };
    case DrillStatus.RUNNING: return { labelKey: strings.status.drill.running, tone: "info" };
    default: return { labelKey: strings.status.drill.requested, tone: "neutral" };
  }
}

export function DrillsPage() {
  const { t } = useTranslation();
  const plans = usePlans();
  const drills = useDrills();
  const run = useRunDrill();
  const records = drills.data ?? [];

  return (
    <section data-testid={selectors.pages.drills} aria-labelledby="drills-heading" className="flex flex-col gap-6">
      <PageHeader title={t(strings.layout.nav.drills)} subtitle={t(strings.drills.subtitle)} />
      <AsyncSection isLoading={plans.isLoading || drills.isLoading} isError={plans.isError || drills.isError} isEmpty={plans.data?.length === 0 && records.length === 0} onRetry={() => { void plans.refetch(); void drills.refetch(); }} emptyState={<EmptyState icon={ShieldCheck} title={t(strings.drills.empty)} description={t(strings.drills.emptyHint)} />}>
        <div className="flex flex-col gap-3">
          {(plans.data ?? []).map((plan) => {
            const latest = records.find((record) => record.planId === plan.id);
            const meta = latest ? drillStatus(latest.status) : undefined;
            return (
              <article key={plan.id} className="flex flex-wrap items-center gap-3 rounded-panel border border-app-border bg-app-surface p-4">
                <div className="min-w-0 flex-1">
                  <p className="font-medium text-app-foreground">{plan.name}</p>
                  <p className="text-xs text-app-muted-foreground">{t(strings.drills.planHint, { schedule: plan.recoveryDrillSchedule || t(strings.drills.manual) })}</p>
                  {latest && <p className="mt-1 text-xs text-app-muted-foreground">{latest.targetId} → {latest.destinationId} · {formatAge(tsToDate(latest.requestedAt), t(strings.common.never))}</p>}
                </div>
                {meta && <StatusChip labelKey={meta.labelKey} tone={meta.tone} />}
                <Button size="sm" variant="outline" disabled={run.isPending} onClick={() => run.mutate({ planId: plan.id, targetId: plan.targetIds[0], destinationId: plan.destinationIds[0] })}>{t(strings.drills.run)}</Button>
              </article>
            );
          })}
          {records.length > 0 && <p className="text-xs text-app-muted-foreground">{t(strings.drills.evidenceHint)}</p>}
        </div>
      </AsyncSection>
    </section>
  );
}

export default DrillsPage;
