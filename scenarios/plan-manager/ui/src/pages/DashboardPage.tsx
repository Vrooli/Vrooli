import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ClipboardList, FlaskConical, GaugeCircle, ListChecks } from "lucide-react";

import { listEntries } from "../api/log";
import { StatusBadge } from "../components/StatusBadge";
import { Card, SectionPanel } from "../components/Surfaces";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { usePlansList } from "../features/plans/usePlans";
import { useTranslation } from "../i18n";
import { planStatusDescriptor } from "../lib/planStatus";
import {
  FindingTriage,
  LogEntryType,
  PlanStatus,
  StalenessTier,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

/** A single stat tile. */
function StatCard({ label, value }: { label: string; value: number | string }) {
  return (
    <Card className="flex flex-col gap-1">
      <p className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</p>
      <p className="text-2xl font-semibold text-app-foreground">{value}</p>
    </Card>
  );
}

/**
 * Dashboard — the console home. Replaces the template stat placeholders with
 * real plan-manager surfaces: counts (total / active / stale plans, candidate
 * findings), the most recent plans, and quick links into the boards. The health
 * card stays as the canonical system-status surface.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const plans = usePlansList();
  const candidates = useQuery({
    queryKey: ["triage", "candidates"],
    queryFn: async () =>
      (await listEntries({ type: LogEntryType.FINDING, triage: FindingTriage.CANDIDATE })).entries,
  });

  const all = plans.data ?? [];
  const active = all.filter((p) => p.status === PlanStatus.ACTIVE).length;
  const stale = all.filter((p) =>
    p.references.some(
      (r) =>
        r.staleness === StalenessTier.LIGHTLY_STALE ||
        r.staleness === StalenessTier.DEFINITELY_STALE,
    ),
  ).length;
  const recent = [...all]
    .filter((p) => p.status !== PlanStatus.ARCHIVED)
    .sort((a, b) => (b.updatedAt > a.updatedAt ? 1 : -1))
    .slice(0, 5);

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="dashboard-heading" className="text-2xl font-semibold">
          {t(strings.pages.dashboard.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      </header>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label={t(strings.pages.dashboard.statPlans)} value={all.length} />
        <StatCard label={t(strings.pages.dashboard.statActive)} value={active} />
        <StatCard label={t(strings.pages.dashboard.statStale)} value={stale} />
        <StatCard
          label={t(strings.pages.dashboard.statCandidates)}
          value={candidates.data?.length ?? 0}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <SectionPanel title={t(strings.pages.dashboard.recentHeading)} headingId="dashboard-recent-heading">
            {recent.length === 0 ? (
              <p className="text-sm text-app-muted-foreground">{t(strings.pages.plans.empty)}</p>
            ) : (
              <ul className="flex flex-col gap-1">
                {recent.map((plan) => (
                  <li key={plan.id}>
                    <Link
                      to={`/plans/${plan.id}`}
                      className="flex items-center justify-between gap-2 rounded-control px-2 py-2 text-sm hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
                    >
                      <span className="min-w-0 truncate font-medium text-app-foreground">
                        {plan.title}
                      </span>
                      <StatusBadge descriptor={planStatusDescriptor(plan.status)} />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </SectionPanel>
        </div>

        <div className="flex flex-col gap-4">
          <SectionPanel title={t(strings.pages.dashboard.quickLinksHeading)} headingId="dashboard-links-heading">
            <nav aria-label={t(strings.pages.dashboard.quickLinksHeading)} className="flex flex-col gap-1">
              <Link
                to="/plans"
                className="flex items-center gap-2 rounded-control px-2 py-2 text-sm hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
              >
                <ClipboardList aria-hidden="true" className="h-4 w-4 text-app-muted-foreground" />
                {t(strings.layout.nav.plans)}
              </Link>
              <Link
                to="/execution"
                className="flex items-center gap-2 rounded-control px-2 py-2 text-sm hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
              >
                <ListChecks aria-hidden="true" className="h-4 w-4 text-app-muted-foreground" />
                {t(strings.layout.nav.execution)}
              </Link>
              <Link
                to="/validation"
                className="flex items-center gap-2 rounded-control px-2 py-2 text-sm hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
              >
                <FlaskConical aria-hidden="true" className="h-4 w-4 text-app-muted-foreground" />
                {t(strings.layout.nav.validation)}
              </Link>
              <Link
                to="/velocity"
                className="flex items-center gap-2 rounded-control px-2 py-2 text-sm hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
              >
                <GaugeCircle aria-hidden="true" className="h-4 w-4 text-app-muted-foreground" />
                {t(strings.layout.nav.velocity)}
              </Link>
            </nav>
          </SectionPanel>
          <HealthCard />
        </div>
      </div>
    </section>
  );
}
