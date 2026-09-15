import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { useStats } from "./controllers/useAnalyticsController";

export interface StatsPanelProps {
  scenario: string;
}

export function StatsPanel({ scenario }: StatsPanelProps) {
  const { t } = useTranslation();
  const stats = useStats(scenario);

  if (stats.isPending) {
    return (
      <div data-testid={selectors.features.analytics.stats.loading}>
        <LoadingState label={t(strings.pages.targetAnalytics.statsLoading)} />
      </div>
    );
  }
  if (stats.isError) {
    return (
      <div data-testid={selectors.features.analytics.stats.error}>
        <ErrorState
          title={t(strings.pages.targetAnalytics.statsError)}
          message={stats.error instanceof Error ? stats.error.message : String(stats.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void stats.refetch();
          }}
        />
      </div>
    );
  }

  const s = stats.data.stats;
  const conflictsDetected = s ? Number(s.conflictsDetected) : 0;
  const conflictsResolved = s ? Number(s.conflictsResolved) : 0;
  const conflictsForce = s ? Number(s.conflictsForceResolved) : 0;
  const placementsAuto = s ? Number(s.placementsAuto) : 0;
  const placementsSuggest = s ? Number(s.placementsSuggest) : 0;
  const overrides = s ? Number(s.overrides) : 0;
  const verdictRate = s ? s.verdictSuccessRate : 0;
  const suppressed = s ? s.verdictSuccessRateSuppressed : true;
  const verdictCount = s ? Number(s.verdictObservationCount) : 0;

  return (
    <dl
      data-testid={selectors.features.analytics.stats.root}
      className="grid gap-2 sm:grid-cols-2 md:grid-cols-3"
    >
      <Stat label={t(strings.pages.targetAnalytics.statsConflictsDetected)} value={conflictsDetected} />
      <Stat label={t(strings.pages.targetAnalytics.statsConflictsResolved)} value={conflictsResolved} />
      <Stat label={t(strings.pages.targetAnalytics.statsConflictsForce)} value={conflictsForce} />
      <Stat label={t(strings.pages.targetAnalytics.statsPlacementsAuto)} value={placementsAuto} />
      <Stat label={t(strings.pages.targetAnalytics.statsPlacementsSuggest)} value={placementsSuggest} />
      <Stat label={t(strings.pages.targetAnalytics.statsOverrides)} value={overrides} />
      <div className="rounded-panel border border-app-border bg-app-surface p-3">
        <dt className="text-xs uppercase text-app-muted-foreground">
          {t(strings.pages.targetAnalytics.statsVerdictSuccess)}
        </dt>
        <dd className="mt-1 text-lg font-semibold">
          {suppressed ? (
            <span
              data-testid={selectors.features.analytics.stats.suppressed}
              className="text-sm text-app-muted-foreground"
            >
              {t(strings.pages.targetAnalytics.statsVerdictSuppressed, { count: verdictCount })}
            </span>
          ) : (
            `${Math.round(verdictRate * 100)}%`
          )}
        </dd>
      </div>
    </dl>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-3">
      <dt className="text-xs uppercase text-app-muted-foreground">{label}</dt>
      <dd className="mt-1 text-lg font-semibold">{value}</dd>
    </div>
  );
}
