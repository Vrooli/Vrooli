import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { ProtoHealthPanel } from "../features/proto-health/ProtoHealthPanel";
import { useTranslation } from "../i18n";

/**
 * Dashboard / home page. Composes the health card plus stat placeholders.
 * Replace the cards with real surfaces when the scenario grows them.
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
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
        <ProtoHealthPanel />
        <HealthCard />
      </div>
    </section>
  );
}
