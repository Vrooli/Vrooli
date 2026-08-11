import { selectors } from "../consts/selectors";
import { WorkbenchPage } from "./WorkbenchPage";

/**
 * Dashboard / home page. Composes the health card plus stat placeholders.
 * Replace the cards with real surfaces when the scenario grows them.
 */
export function DashboardPage() {
  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <WorkbenchPage />
    </section>
  );
}
