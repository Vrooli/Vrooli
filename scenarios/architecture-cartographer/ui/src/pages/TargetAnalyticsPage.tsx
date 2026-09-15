import { Navigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { BuildStatusDeltas } from "../features/analytics/BuildStatusDeltas";
import { EventsTimeline } from "../features/analytics/EventsTimeline";
import { PlacementsTable } from "../features/analytics/PlacementsTable";
import { StatsPanel } from "../features/analytics/StatsPanel";

export function TargetAnalyticsPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  if (scenario === null) return <Navigate to="/" replace />;

  return (
    <section
      data-testid={selectors.pages.targetAnalytics}
      aria-labelledby="target-analytics-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-analytics-heading" className="text-xl font-semibold">
          {t(strings.pages.targetAnalytics.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.targetAnalytics.description)}
        </p>
      </header>

      <section aria-labelledby="analytics-stats-heading" className="flex flex-col gap-2">
        <h4 id="analytics-stats-heading" className="text-lg font-semibold">
          {t(strings.pages.targetAnalytics.statsHeading)}
        </h4>
        <StatsPanel scenario={scenario} />
      </section>

      <section aria-labelledby="analytics-events-heading" className="flex flex-col gap-2">
        <h4 id="analytics-events-heading" className="text-lg font-semibold">
          {t(strings.pages.targetAnalytics.eventsHeading)}
        </h4>
        <EventsTimeline scenario={scenario} />
      </section>

      <section aria-labelledby="analytics-placements-heading" className="flex flex-col gap-2">
        <h4 id="analytics-placements-heading" className="text-lg font-semibold">
          {t(strings.pages.targetAnalytics.placementsHeading)}
        </h4>
        <PlacementsTable scenario={scenario} />
      </section>

      <section aria-labelledby="analytics-build-heading" className="flex flex-col gap-2">
        <h4 id="analytics-build-heading" className="text-lg font-semibold">
          {t(strings.pages.targetAnalytics.buildDeltasHeading)}
        </h4>
        <BuildStatusDeltas scenario={scenario} />
      </section>
    </section>
  );
}
