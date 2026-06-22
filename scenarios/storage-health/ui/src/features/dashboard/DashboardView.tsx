import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { RefreshCw } from "lucide-react";

import { Button } from "../../components/ui/button";
import { EmptyState, ErrorState, LoadingState, Skeleton } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { storageClient, type ScanFleetResponse } from "../../api/storage";
import { IsolationBadge } from "../storage/format";
import { relativeFromNow } from "../storage/relativeFromNow";

const SCORECARD_PREVIEW_LIMIT = 4;

/**
 * Safety-first dashboard. Backed by the persisted `GetInventory` snapshot (a
 * "Scan now" action triggers a live `ScanFleet`). The isolation-unready count is
 * the headline number, rendered in the alert color when > 0; every stat links
 * into the matching `/fleet` view.
 */
export function DashboardView() {
  const { t } = useTranslation();

  const query = useQuery<ScanFleetResponse>({
    queryKey: ["dashboard-inventory"],
    queryFn: () => storageClient.getInventory(),
  });

  const scanMutation = useQuery<ScanFleetResponse>({
    queryKey: ["dashboard-scan"],
    queryFn: () => storageClient.scanFleet({}),
    enabled: false,
  });

  const data = scanMutation.data ?? query.data;
  const scanning = scanMutation.isFetching;
  const unready = [...(data?.entries ?? [])]
    .filter((e) => !e.isolationReady)
    .sort((a, b) => a.scenario.localeCompare(b.scenario));

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-wrap items-start gap-3">
        <div className="flex flex-col gap-1">
          <h2 id="dashboard-heading" className="text-2xl font-semibold">
            {t(strings.pages.dashboard.title)}
          </h2>
          <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
        </div>
        <Button
          data-testid={selectors.dashboard.scanButton}
          variant="outline"
          size="sm"
          className="ms-auto"
          onClick={() => void scanMutation.refetch()}
          disabled={scanning}
        >
          <RefreshCw aria-hidden="true" className={["me-1 h-4 w-4", scanning ? "animate-spin" : ""].join(" ")} />
          {scanning ? t(strings.common.scanning) : t(strings.common.scanNow)}
        </Button>
      </header>

      {query.isLoading && (
        <LoadingState
          testId={selectors.dashboard.loading}
          title={t(strings.dashboard.loadingTitle)}
          skeleton={
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-20 w-full" />
              ))}
            </div>
          }
        />
      )}

      {query.error && (
        <ErrorState
          testId={selectors.dashboard.error}
          title={t(strings.common.errorTitle)}
          message={errorMessage(query.error, t)}
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
        />
      )}

      {data && !query.error && data.scenarioCount === 0 && !scanning && (
        <EmptyState
          testId={selectors.dashboard.empty}
          title={t(strings.dashboard.empty.title)}
          message={t(strings.dashboard.empty.message)}
          actionLabel={t(strings.dashboard.empty.action)}
          onAction={() => void scanMutation.refetch()}
        />
      )}

      {data && !query.error && data.scenarioCount > 0 && (
        <>
          <div data-testid={selectors.dashboard.band} className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <StatLink
              testId={selectors.dashboard.statScenarios}
              to="/fleet?view=all"
              label={t(strings.dashboard.stat.scenarios)}
              value={data.scenarioCount}
            />
            <StatLink
              testId={selectors.dashboard.statIsolationUnready}
              to="/fleet?view=isolation"
              label={t(strings.dashboard.stat.isolationUnready)}
              value={data.isolationUnreadyCount}
              alert={data.isolationUnreadyCount > 0}
            />
            <StatLink
              testId={selectors.dashboard.statNoBackup}
              to="/fleet?view=no-backup"
              label={t(strings.dashboard.stat.noBackup)}
              value={data.noBackupCount}
            />
            <StatLink
              testId={selectors.dashboard.statFindings}
              to="/fleet?view=all"
              label={t(strings.dashboard.stat.findings)}
              value={data.findingCount}
            />
          </div>

          <section
            data-testid={selectors.dashboard.scorecard}
            aria-label={t(strings.dashboard.scorecard.title)}
            className="rounded-panel border border-app-border bg-app-surface p-4"
          >
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-medium text-app-muted-foreground">
                {t(strings.dashboard.scorecard.title)}
              </h3>
              <Link to="/fleet?view=isolation" className="ms-auto text-sm text-app-primary underline">
                {t(strings.dashboard.scorecard.viewAll)}
              </Link>
            </div>
            {unready.length === 0 ? (
              <p className="mt-3 text-sm text-app-muted-foreground">
                {t(strings.dashboard.scorecard.empty)}
              </p>
            ) : (
              <ul className="mt-3 flex flex-col gap-2">
                {unready.slice(0, SCORECARD_PREVIEW_LIMIT).map((e) => (
                  <li key={e.scenario} className="flex flex-wrap items-center gap-2 text-sm">
                    <Link
                      to={`/validate?scenario=${encodeURIComponent(e.scenario)}`}
                      className="font-medium text-app-primary underline"
                    >
                      {e.scenario}
                    </Link>
                    <IsolationBadge ready={false} />
                    <span className="text-app-muted-foreground">{e.isolationReason}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section
            data-testid={selectors.dashboard.engines}
            aria-label={t(strings.dashboard.engines.title)}
            className="rounded-panel border border-app-border bg-app-surface p-4"
          >
            <h3 className="text-sm font-medium text-app-muted-foreground">
              {t(strings.dashboard.engines.title)}
            </h3>
            {data.engineDistribution.length === 0 ? (
              <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.dashboard.engines.empty)}</p>
            ) : (
              <ul className="mt-3 flex flex-col gap-2">
                {data.engineDistribution.map((d) => {
                  const max = Math.max(...data.engineDistribution.map((x) => x.scenarioCount), 1);
                  return (
                    <li key={d.engine} className="flex items-center gap-3 text-sm">
                      <span className="w-24 shrink-0 text-app-foreground">{d.engine}</span>
                      <div className="h-2 flex-1 overflow-hidden rounded-control bg-app-surface-muted">
                        <div className="h-full bg-app-primary" style={{ width: `${(d.scenarioCount / max) * 100}%` }} />
                      </div>
                      <span className="w-8 shrink-0 text-end tabular-nums text-app-muted-foreground">
                        {d.scenarioCount}
                      </span>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>

          <section
            data-testid={selectors.dashboard.freshness}
            aria-label={t(strings.dashboard.freshness.title)}
            className="rounded-panel border border-app-border bg-app-surface p-4"
          >
            <h3 className="text-sm font-medium text-app-muted-foreground">
              {t(strings.dashboard.freshness.title)}
            </h3>
            <p className="mt-2 text-sm text-app-foreground">
              {data.scannedAt
                ? t(strings.dashboard.freshness.scannedAt, { when: relativeFromNow(data.scannedAt) })
                : t(strings.dashboard.freshness.never)}
            </p>
          </section>
        </>
      )}
    </section>
  );
}

function StatLink({
  testId,
  to,
  label,
  value,
  alert,
}: {
  testId: string;
  to: string;
  label: string;
  value: number;
  alert?: boolean;
}) {
  return (
    <Link
      to={to}
      className="rounded-panel border border-app-border bg-app-surface p-4 transition-colors hover:bg-app-surface-muted"
    >
      <span className="block text-xs uppercase tracking-wide text-app-muted-foreground">{label}</span>
      <span
        data-testid={testId}
        className={[
          "mt-2 block text-2xl font-semibold tabular-nums",
          alert ? "text-app-danger" : "text-app-foreground",
        ].join(" ")}
      >
        {value}
      </span>
    </Link>
  );
}
