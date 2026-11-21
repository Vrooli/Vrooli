import { lazy, Suspense, useState } from "react";
import { Link } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useRecentTargets } from "../features/targets/hooks/useRecentTargets";
import { useTranslation } from "../i18n";

const ActiveSnapshotsPanel = lazy(() =>
  import("../features/targets/ActiveSnapshotsPanel").then((m) => ({ default: m.ActiveSnapshotsPanel })),
);
const HealthCard = lazy(() => import("../features/health/HealthCard").then((m) => ({ default: m.HealthCard })));
const RecentTargetsList = lazy(() =>
  import("../features/targets/RecentTargetsList").then((m) => ({ default: m.RecentTargetsList })),
);

/**
 * Overview — the cartographer landing page. Composes:
 *   - Recent targets (localStorage-backed)
 *   - Active snapshots (live `ListGraphSnapshots` call)
 *   - System health (the existing HealthCard)
 */
export function OverviewPage() {
  const { t } = useTranslation();
  const { recent, remove } = useRecentTargets();
  const [showSnapshots, setShowSnapshots] = useState(false);
  const [showHealth, setShowHealth] = useState(false);

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
        <Link
          to="/targets/new"
          className="inline-flex h-10 items-center justify-center rounded-control bg-app-primary px-4 py-2 text-sm font-medium text-app-primary-foreground transition-colors hover:bg-app-primary/90"
        >
          {t(strings.pages.overview.startExtraction)}
        </Link>
      </header>

      <div className="grid gap-6 lg:grid-cols-2">
        <section aria-labelledby="overview-recent-heading" className="flex flex-col gap-3">
          <h3 id="overview-recent-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.overview.recentTargetsHeading)}
          </h3>
          {recent.length === 0 ? (
            <div
              data-testid={selectors.features.targets.recent.empty}
              role="status"
              className="flex flex-col items-center justify-center gap-2 rounded-panel border border-dashed border-app-border bg-app-surface p-8 text-center text-app-muted-foreground backdrop-blur-sm"
            >
              <p className="text-sm font-semibold text-app-foreground">
                {t(strings.targets.recent.emptyTitle)}
              </p>
              <p className="max-w-sm text-sm">{t(strings.targets.recent.emptyDescription)}</p>
            </div>
          ) : (
            <Suspense fallback={null}>
              <RecentTargetsList recent={recent} onRemove={remove} />
            </Suspense>
          )}
        </section>

        <section aria-labelledby="overview-snapshots-heading" className="flex flex-col gap-3">
          <h3 id="overview-snapshots-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.overview.activeSnapshotsHeading)}
          </h3>
          {showSnapshots ? (
            <Suspense fallback={null}>
              <ActiveSnapshotsPanel />
            </Suspense>
          ) : (
            <div className="flex min-h-28 items-center justify-center rounded-panel border border-app-border bg-app-surface">
              <button
                type="button"
                className="inline-flex h-10 items-center justify-center rounded-control border border-app-border px-4 py-2 text-sm font-medium text-app-foreground transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
                onClick={() => setShowSnapshots(true)}
              >
                {t(strings.pages.overview.loadSnapshots)}
              </button>
            </div>
          )}
        </section>
      </div>

      <section aria-labelledby="overview-health-heading" className="flex flex-col gap-3">
        <h3 id="overview-health-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.overview.healthHeading)}
        </h3>
        {showHealth ? (
          <Suspense fallback={null}>
            <HealthCard />
          </Suspense>
        ) : (
          <div className="flex min-h-24 items-center justify-center rounded-panel border border-app-border bg-app-surface">
            <button
              type="button"
              className="inline-flex h-10 items-center justify-center rounded-control border border-app-border px-4 py-2 text-sm font-medium text-app-foreground transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
              onClick={() => setShowHealth(true)}
            >
              {t(strings.pages.overview.checkHealth)}
            </button>
          </div>
        )}
      </section>
    </section>
  );
}
