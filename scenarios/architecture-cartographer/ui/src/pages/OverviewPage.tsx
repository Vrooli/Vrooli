import { Link } from "react-router-dom";

import { Button } from "../components/ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { ActiveSnapshotsPanel } from "../features/targets/ActiveSnapshotsPanel";
import { RecentTargetsList } from "../features/targets/RecentTargetsList";
import { useRecentTargets } from "../features/targets/hooks/useRecentTargets";
import { useTranslation } from "../i18n";

/**
 * Overview — the cartographer landing page. Composes:
 *   - Recent targets (localStorage-backed)
 *   - Active snapshots (live `ListGraphSnapshots` call)
 *   - System health (the existing HealthCard)
 */
export function OverviewPage() {
  const { t } = useTranslation();
  const { recent, remove } = useRecentTargets();

  return (
    <section
      data-testid={selectors.pages.overview}
      aria-labelledby="overview-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 id="overview-heading" className="text-2xl font-semibold">
            {t(strings.pages.overview.title)}
          </h2>
          <p className="text-app-muted-foreground">{t(strings.pages.overview.description)}</p>
        </div>
        <Button asChild>
          <Link to="/targets/new">{t(strings.pages.overview.startExtraction)}</Link>
        </Button>
      </header>

      <div className="grid gap-6 lg:grid-cols-2">
        <section aria-labelledby="overview-recent-heading" className="flex flex-col gap-3">
          <h3 id="overview-recent-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.overview.recentTargetsHeading)}
          </h3>
          <RecentTargetsList recent={recent} onRemove={remove} />
        </section>

        <section aria-labelledby="overview-snapshots-heading" className="flex flex-col gap-3">
          <h3 id="overview-snapshots-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.overview.activeSnapshotsHeading)}
          </h3>
          <ActiveSnapshotsPanel />
        </section>
      </div>

      <section aria-labelledby="overview-health-heading" className="flex flex-col gap-3">
        <h3 id="overview-health-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.overview.healthHeading)}
        </h3>
        <HealthCard />
      </section>
    </section>
  );
}
