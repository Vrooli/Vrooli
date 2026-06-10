import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { ScoreDashboard } from "../features/scoring/ScoreDashboard";
import { useTranslation } from "../i18n";

/**
 * Dashboard / home page: the scoring dashboard (this scenario's primary
 * surface) plus the API health card.
 */
export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <ScoreDashboard />
      <div className="max-w-md">
        <HealthCard />
      </div>
    </section>
  );
}
