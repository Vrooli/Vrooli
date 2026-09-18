import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { MealsPage } from "../features/meals/MealsPage";
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
      <MealsPage />
    </section>
  );
}
