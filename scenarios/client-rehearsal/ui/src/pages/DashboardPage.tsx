import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { RehearsalWorkspace } from "../features/rehearsals/RehearsalWorkspace";
import { useTranslation } from "../i18n";

export function DashboardPage() {
  const { t } = useTranslation();
  return (
    <section data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-space-md">
      <PageHeader headingId="dashboard-heading" title={t(strings.pages.dashboard.title)} description={t(strings.pages.dashboard.description)} testId={selectors.pages.dashboardHeader} />
      <RehearsalWorkspace />
      <HealthCard />
    </section>
  );
}
