import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FleetPanel } from "../features/fleet/FleetPanel";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";

/**
 * Dashboard / home page. Composes the API health card and the fleet panel —
 * the Phase-1 control-plane surface showing the owner's trusted nodes with live
 * presence. Later phases add dispatch, run history, and pairing surfaces.
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
      <div className="grid gap-4 md:grid-cols-2">
        <HealthCard />
        <FleetPanel />
      </div>
    </section>
  );
}
