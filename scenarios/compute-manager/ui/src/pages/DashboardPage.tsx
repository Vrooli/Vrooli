import { Compass } from "lucide-react";

import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";

/**
 * Home. The first thing an operator sees, and the one page the template
 * refuses to decide for you.
 *
 * Two things are on it today. `HealthCard` is real: a scenario-owned feature
 * built from library parts, polling the API this scenario ships. Keep it or
 * move it; every scenario has a health surface. The `EmptyState` beside it is
 * a placeholder that the orientation gate fails until you replace it with the
 * surface that answers this scenario's first question.
 */
export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-space-md">
      <PageHeader
        headingId="dashboard-heading"
        title={t(strings.pages.dashboard.title)}
        description={t(strings.pages.dashboard.description)}
        testId={selectors.pages.dashboardHeader}
      />
      <div className="grid gap-space-sm lg:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
        <HealthCard />
        {/* PLACEHOLDER:home-surface — delete this EmptyState with the real home surface. The design-decision gate stays red until this marker is gone. */}
        <div data-testid={selectors.pages.dashboardPlaceholder} className="flex">
          <EmptyState
            className="flex-1"
            icon={<Compass aria-hidden="true" />}
            title={t(strings.pages.dashboard.placeholderTitle)}
            description={t(strings.pages.dashboard.placeholderDescription)}
          />
        </div>
      </div>
    </section>
  );
}
