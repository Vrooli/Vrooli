import { Aperture, FlaskConical, WandSparkles } from "lucide-react";
import { NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { ProviderMaturityCard } from "../features/health/ProviderMaturityCard";
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
      <div className="grid gap-4 md:grid-cols-2">
        <HealthCard />
        <ProviderMaturityCard />
      </div>
      <nav aria-label={t(strings.layout.sidebarLabel)} className="grid gap-3 sm:grid-cols-3">
        <NavLink
          to="/brands"
          className="group flex min-h-20 items-center gap-3 rounded-panel border border-app-border bg-app-surface p-4 text-app-foreground transition-colors hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
        >
          <Aperture aria-hidden="true" className="h-5 w-5 shrink-0 text-app-primary" />
          <span className="font-medium">{t(strings.pages.brands.title)}</span>
        </NavLink>
        <NavLink
          to="/generation"
          className="group flex min-h-20 items-center gap-3 rounded-panel border border-app-border bg-app-surface p-4 text-app-foreground transition-colors hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
        >
          <WandSparkles aria-hidden="true" className="h-5 w-5 shrink-0 text-app-primary" />
          <span className="font-medium">{t(strings.pages.generation.title)}</span>
        </NavLink>
        <NavLink
          to="/apply"
          className="group flex min-h-20 items-center gap-3 rounded-panel border border-app-border bg-app-surface p-4 text-app-foreground transition-colors hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
        >
          <FlaskConical aria-hidden="true" className="h-5 w-5 shrink-0 text-app-primary" />
          <span className="font-medium">{t(strings.pages.apply.title)}</span>
        </NavLink>
      </nav>
    </section>
  );
}
