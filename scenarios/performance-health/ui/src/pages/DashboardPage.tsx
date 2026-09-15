import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Activity, BarChart3, FileSearch, Layers, ShieldCheck, Wand2 } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { Button } from "../components/ui/button";
import { EmptyState, ErrorState, Skeleton } from "../components/ui/state";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";
import { perfClient } from "../api/perf";

/** Union of every leaf translation-key string in the typed `strings` registry. */
type LeafKeys<T> = T extends string ? T : T extends object ? LeafKeys<T[keyof T]> : never;
type TKey = LeafKeys<typeof strings>;

interface WorkflowLink {
  to: string;
  icon: LucideIcon;
  titleKey: TKey;
  descKey: TKey;
}

const WORKFLOWS: WorkflowLink[] = [
  { to: "/audit", icon: Activity, titleKey: strings.audit.title, descKey: strings.audit.description },
  {
    to: "/trends",
    icon: BarChart3,
    titleKey: strings.trends.title,
    descKey: strings.trends.description,
  },
  { to: "/fleet", icon: Layers, titleKey: strings.fleet.title, descKey: strings.fleet.description },
  { to: "/trace", icon: FileSearch, titleKey: strings.trace.title, descKey: strings.trace.description },
  {
    to: "/readiness",
    icon: Wand2,
    titleKey: strings.readiness.title,
    descKey: strings.readiness.description,
  },
  {
    to: "/budgets",
    icon: ShieldCheck,
    titleKey: strings.budgets.title,
    descKey: strings.budgets.description,
  },
];

/**
 * Overview / home. Surfaces a live fleet snapshot (from ScanFleet) and direct
 * entry points into every core performance workflow, so a first-time user
 * immediately understands what the product does and where to start.
 */
export function DashboardPage() {
  const { t } = useTranslation();

  const fleet = useQuery({
    queryKey: ["fleet-scan", "overview"],
    queryFn: () => perfClient.scanFleet({}),
    staleTime: 60_000,
  });

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="dashboard-heading" className="text-2xl font-semibold">
          {t(strings.pages.dashboard.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      </header>

      <section aria-labelledby="dashboard-snapshot-heading" className="flex flex-col gap-3">
        <h3
          id="dashboard-snapshot-heading"
          className="text-sm font-semibold uppercase tracking-wide text-app-muted-foreground"
        >
          {t(strings.pages.dashboard.snapshotTitle)}
        </h3>

        {fleet.isLoading && (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-20 w-full" />
            ))}
          </div>
        )}

        {fleet.error && (
          <ErrorState
            title={t(strings.pages.dashboard.snapshotError)}
            message={errorMessage(fleet.error, t)}
            onRetry={() => void fleet.refetch()}
            retrying={fleet.isFetching}
          />
        )}

        {fleet.data && !fleet.error && fleet.data.scenarioCount === 0 && (
          <EmptyState
            icon={Layers}
            message={t(strings.pages.dashboard.snapshotEmpty)}
            actionSlot={
              <Button asChild size="sm">
                <Link to="/fleet">{t(strings.pages.dashboard.snapshotCta)}</Link>
              </Button>
            }
          />
        )}

        {fleet.data && !fleet.error && fleet.data.scenarioCount > 0 && (
          <dl
            data-testid={selectors.pages.overviewSnapshot}
            className="grid grid-cols-2 gap-3 sm:grid-cols-3"
          >
            <Stat label={t(strings.fleet.summary.scenarios)} value={fleet.data.scenarioCount} />
            <Stat label={t(strings.fleet.summary.noBudget)} value={fleet.data.noBudgetCount} />
            <Stat label={t(strings.fleet.summary.regressed)} value={fleet.data.regressedCount} />
          </dl>
        )}
      </section>

      <h3
        id="dashboard-workflows-heading"
        className="text-sm font-semibold uppercase tracking-wide text-app-muted-foreground"
      >
        {t(strings.pages.dashboard.workflowsTitle)}
      </h3>
      <div
        aria-labelledby="dashboard-workflows-heading"
        className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
      >
        {WORKFLOWS.map((w) => {
          const Icon = w.icon;
          return (
            <Link
              key={w.to}
              to={w.to}
              data-testid={selectors.pages.workflowCard({ to: w.to })}
              className="group flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4 transition-colors hover:border-app-primary"
            >
              <Icon aria-hidden="true" className="h-5 w-5 text-app-primary" />
              <span className="font-medium text-app-foreground group-hover:text-app-primary">
                {t(w.titleKey)}
              </span>
              <span className="text-sm text-app-muted-foreground">{t(w.descKey)}</span>
            </Link>
          );
        })}
      </div>
    </section>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-4">
      <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</dt>
      <dd className="mt-2 text-2xl font-semibold tabular-nums">{value}</dd>
    </div>
  );
}
