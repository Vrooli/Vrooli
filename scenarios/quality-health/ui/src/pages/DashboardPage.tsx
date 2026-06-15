import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
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
      <p className="text-sm font-medium uppercase text-app-muted-foreground">
        {t(strings.app.eyebrow)}
      </p>
      <p className="max-w-3xl text-app-muted-foreground">
        {t(strings.app.description)}
      </p>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="grid gap-4 md:grid-cols-3">
        <HealthCard />
        <div className="rounded-panel border border-app-border bg-app-surface p-4">
          <p className="text-xs uppercase text-app-muted-foreground">
            {t(strings.pages.dashboard.statPlaceholderLabel)}
          </p>
          <p className="mt-2 text-2xl font-semibold">—</p>
        </div>
        <div className="rounded-panel border border-app-border bg-app-surface p-4">
          <p className="text-xs uppercase text-app-muted-foreground">
            {t(strings.pages.dashboard.statPlaceholderLabel)}
          </p>
          <p className="mt-2 text-2xl font-semibold">—</p>
        </div>
      </div>
    </section>
  );
}
