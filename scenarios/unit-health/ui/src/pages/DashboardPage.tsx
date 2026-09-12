import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ScenarioValidationWorkbench } from "../features/validation/ScenarioValidationWorkbench";
import { useTranslation } from "../i18n";

/**
 * Dashboard / home page. Hosts the validation workbench — the operator surface
 * for running a scenario's test-maturity validation. The heading/description
 * stay screen-reader-only because the workbench renders its own visible header.
 */
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
      <ScenarioValidationWorkbench />
    </section>
  );
}
