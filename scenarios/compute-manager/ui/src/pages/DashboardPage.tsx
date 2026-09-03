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
 * The home surface answers the operator's first question: is capacity healthy,
 * and what instances need attention?
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
        <div data-testid={selectors.pages.dashboardPlaceholder} className="flex">
          <EmptyState
            className="flex-1"
            title={t(strings.pages.dashboard.placeholderTitle)}
            description={t(strings.pages.dashboard.placeholderDescription)}
          />
        </div>
      </div>
    </section>
  );
}
