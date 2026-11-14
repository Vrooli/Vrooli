import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ScenarioAuditWorkbench } from "../features/audit/ScenarioAuditWorkbench";
import { useTranslation } from "../i18n";

export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      aria-describedby="dashboard-description"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="sr-only">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p id="dashboard-description" className="sr-only">
        {t(strings.pages.dashboard.description)}
      </p>
      <ScenarioAuditWorkbench />
    </section>
  );
}
