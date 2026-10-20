import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { ProjectWorkspace } from "../features/projects/ProjectWorkspace";
import { useTranslation } from "../i18n";

/** The local project index is the first product surface. */
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
      <div className="grid gap-space-md xl:grid-cols-[minmax(0,1fr)_minmax(18rem,0.32fr)]">
        <ProjectWorkspace />
        <HealthCard />
      </div>
    </section>
  );
}
