import { ShieldCheck } from "lucide-react";

import { AsyncSection } from "../components/AsyncSection";
import { EmptyState } from "../components/EmptyState";
import { PageHeader } from "../components/PageHeader";
import { Button } from "../components/ui/button";
import { StatusChip } from "../components/ui/status-chip";
import { useDrills, useRunDrill } from "../hooks/useDrills";
import { usePlans } from "../hooks/usePlans";
import { useTargets } from "../hooks/useTargets";
import { useDestinations } from "../hooks/useDestinations";
import { DrillStatus } from "../api/drills";
import { tsToDate } from "../lib/proto";
import { formatAge } from "../lib/format";
import { weeklyProofUnits, type WeeklyProofState } from "../features/drills/weeklyProof";
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

const WEEKLY_PROOF_LABELS: Record<WeeklyProofState, StringKey> = {
  verified: strings.drills.weeklyVerified,
  missing: strings.drills.weeklyMissing,
  pending: strings.drills.weeklyPending,
  failed: strings.drills.weeklyFailed,
  stale: strings.drills.weeklyStale,
  retired: strings.drills.weeklyRetired,
  manual: strings.drills.weeklyManual,
  disabled: strings.drills.weeklyDisabled,
};

const WEEKLY_PROOF_STYLES: Record<WeeklyProofState, string> = {
  verified: "border-app-success/40 bg-app-success/10 text-app-success",
  missing: "border-app-border bg-app-muted/30 text-app-muted-foreground",
  pending: "border-app-info/40 bg-app-info/10 text-app-info",
  failed: "border-app-danger/40 bg-app-danger/10 text-app-danger",
  stale: "border-app-warning/40 bg-app-warning/10 text-app-warning",
  retired: "border-app-border bg-app-muted/30 text-app-muted-foreground",
  manual: "border-app-border bg-app-muted/30 text-app-muted-foreground",
  disabled: "border-app-border bg-app-muted/30 text-app-muted-foreground",
};

export function DrillsPage() {
  const { t } = useTranslation();
  const plans = usePlans();
  const drills = useDrills();
  const targets = useTargets();
  const destinations = useDestinations();
  const run = useRunDrill();
  const records = drills.data ?? [];
  const proofUnits = weeklyProofUnits(plans.data ?? [], records);
  const eligibleProofUnits = proofUnits.filter((unit) => unit.state !== "disabled" && unit.state !== "manual");
  const verifiedProofCount = eligibleProofUnits.filter((unit) => unit.state === "verified").length;

  return (
    <section data-testid={selectors.pages.drills} aria-labelledby="drills-heading" className="flex flex-col gap-6">
      <PageHeader title={t(strings.layout.nav.drills)} subtitle={t(strings.drills.subtitle)} />
      <AsyncSection isLoading={plans.isLoading || drills.isLoading || targets.isLoading || destinations.isLoading} isError={plans.isError || drills.isError || targets.isError || destinations.isError} isEmpty={plans.data?.length === 0 && records.length === 0} onRetry={() => { void plans.refetch(); void drills.refetch(); void targets.refetch(); void destinations.refetch(); }} emptyState={<EmptyState icon={ShieldCheck} title={t(strings.drills.empty)} description={t(strings.drills.emptyHint)} />}>
        <div className="flex flex-col gap-3">
          <section aria-labelledby="weekly-proof-heading" className="rounded-panel border border-app-border bg-app-surface p-4">
            <h2 id="weekly-proof-heading" className="font-medium text-app-foreground">{t(strings.drills.weeklyTitle)}</h2>
            <p role="status" className="mt-1 text-sm text-app-muted-foreground">
              {eligibleProofUnits.length === 0
                ? t(strings.drills.weeklyNoPairs)
                : <>
                    <span className={`font-medium ${verifiedProofCount === eligibleProofUnits.length ? "text-app-success" : "text-app-warning"}`}>
                      {t(verifiedProofCount === eligibleProofUnits.length ? strings.drills.weeklyAllGreen : strings.drills.weeklyIncomplete)}
                    </span>
                    <span className="ml-2">{t(strings.drills.weeklySummary, { green: verifiedProofCount, total: eligibleProofUnits.length })}</span>
                  </>}
            </p>
          </section>
          {(plans.data ?? []).map((plan) => {
            const units = plan.targetIds.flatMap((targetId) => plan.destinationIds.map((destinationId) => ({ targetId, destinationId })));
            return (
              <article key={plan.id} className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
                <div>
                  <p className="font-medium text-app-foreground">{plan.name}</p>
                  <p className="text-xs text-app-muted-foreground">{t(strings.drills.planHint, { schedule: plan.recoveryDrillSchedule || t(strings.drills.manual) })}</p>
                </div>
                <div className="flex flex-col gap-2">
                  {units.map(({ targetId, destinationId }) => {
                    const latest = records.find((record) => record.planId === plan.id && record.targetId === targetId && record.destinationId === destinationId);
                    const meta = latest ? drillStatus(latest.status) : undefined;
                    const proof = proofUnits.find((unit) => unit.planId === plan.id && unit.targetId === targetId && unit.destinationId === destinationId);
                    const target = (targets.data ?? []).find((candidate) => candidate.id === targetId);
                    const destination = (destinations.data ?? []).find((candidate) => candidate.id === destinationId);
                    return (
                      <div key={`${targetId}:${destinationId}`} className="flex flex-wrap items-center gap-3 rounded-md border border-app-border/70 px-3 py-2">
                        <p className="min-w-0 flex-1 text-xs text-app-muted-foreground">{target?.name ?? targetId} [{targetId}] → {destination?.name ?? destinationId} [{destinationId}] · {latest ? formatAge(tsToDate(latest.requestedAt), t(strings.common.never)) : t(strings.drills.neverRun)}</p>
                        {proof && <span role="status" className={`rounded-full border px-2 py-1 text-xs ${WEEKLY_PROOF_STYLES[proof.state]}`}>{t(WEEKLY_PROOF_LABELS[proof.state])}</span>}
                        {meta && <StatusChip labelKey={meta.labelKey} tone={meta.tone} />}
                        {latest?.notificationState && <span className="text-xs text-app-muted-foreground">{t(latest.notificationState === "pending" ? strings.drills.notificationPending : latest.notificationState === "sent" ? strings.drills.notificationSent : strings.drills.notificationUntracked)}</span>}
                        <Button size="sm" variant="outline" disabled={run.isPending} onClick={() => run.mutate({ planId: plan.id, targetId, destinationId })}>{t(strings.drills.run)}</Button>
                      </div>
                    );
                  })}
                  {units.length === 0 && <p className="text-xs text-app-muted-foreground">{t(strings.drills.noUnits)}</p>}
                </div>
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
