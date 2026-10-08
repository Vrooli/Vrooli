import { lazy, Suspense } from "react";

import { PageHeader } from "../components/PageHeader";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const OverviewDashboard = lazy(() => import("./OverviewDashboard"));

/**
 * Overview — the operational landing surface. Posture banner first (is
 * everything protected, within cap, and verified?), then the storage strip and
 * the owner-grouped coverage grid. When nothing is set up yet, a single setup
 * call to action funnels the operator into destinations → plans.
 */
export function OverviewPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.overview}
      aria-labelledby="overview-heading"
      className="flex flex-col gap-6"
    >
      <div id="overview-heading">
        <PageHeader title={t(strings.overview.title)} subtitle={t(strings.overview.subtitle)} />
      </div>
      <Suspense fallback={<div role="status" aria-live="polite">{t(strings.common.loading)}</div>}>
        <OverviewDashboard />
      </Suspense>
    </section>
  );
}

export default OverviewPage;
