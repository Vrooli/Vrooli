import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";

/**
 * Home. The first thing an operator sees, and the one page the template
 * refuses to decide for you.
 *
 * The first surface answers the operator's source question: what is the
 * current readiness standing, and what action is safe next?
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
        <div data-testid="source-readiness-summary" className="flex flex-col gap-space-sm rounded-panel border border-border-subtle bg-surface p-space-md">
          <div className="flex items-center justify-between gap-space-sm">
            <h2 className="text-lg font-semibold">Source export readiness</h2>
            <span className="rounded-full bg-status-warning/15 px-space-sm py-1 text-sm text-status-warning">Not verified</span>
          </div>
          <p className="text-sm text-text-secondary">Select an immutable source snapshot to inspect closure, policy, archive integrity, and clean-build evidence.</p>
          <div className="grid gap-space-sm sm:grid-cols-3" aria-label="Source export gates">
            <div className="rounded-panel bg-surface-muted p-space-sm"><strong className="block">Closure</strong><span className="text-sm text-text-secondary">Awaiting source</span></div>
            <div className="rounded-panel bg-surface-muted p-space-sm"><strong className="block">Policy</strong><span className="text-sm text-text-secondary">Fail-closed</span></div>
            <div className="rounded-panel bg-surface-muted p-space-sm"><strong className="block">Publication</strong><span className="text-sm text-text-secondary">Human-only</span></div>
          </div>
        </div>
      </div>
    </section>
  );
}
