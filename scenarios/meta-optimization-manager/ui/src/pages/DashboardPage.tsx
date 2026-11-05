import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { ReadinessBoard } from "../features/readiness/ReadinessBoard";
import { useTranslation } from "../i18n";

/**
 * Readiness scoreboard — the operator console home. Shows per-projection
 * coverage + denominator-confidence + the latest empirical trial trend, with the
 * API health card alongside.
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
      <ReadinessBoard />
      <HealthCard />
    </section>
  );
}
