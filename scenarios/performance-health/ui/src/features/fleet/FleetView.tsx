import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { AlertTriangle, RefreshCw } from "lucide-react";

import { Button } from "../../components/ui/button";
import { EmptyState, ErrorState, LoadingState, Skeleton } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { perfClient, type FleetScenarioEntry } from "../../api/perf";
import { TIER_LABEL_KEY, fleetTierKey, formatMs, tierChipClass } from "../perf/format";
import { useScenario } from "../perf/scenarioContextValue";

const FLEET_QUERY_KEY = ["fleet-scan", "dashboard"] as const;

/**
 * "Fleet perf dashboard" workflow. Calls FleetService.ScanFleet and renders the
 * cross-scenario offender views the backend computes:
 *
 *   - headline counters (scenarios / no-budget / regressed)
 *   - tier distribution
 *   - offenders without a perf budget
 *   - slowest builds (sorted by total build time)
 *   - regressed scenarios
 *   - scan errors (scenarios that couldn't be graded)
 *
 * Clicking a scenario selects it for the per-scenario workflows.
 */
export function FleetView() {
  const { t } = useTranslation();
  const { setScenario } = useScenario();

  const query = useQuery({
    queryKey: FLEET_QUERY_KEY,
    queryFn: () => perfClient.scanFleet({}),
  });

  const data = query.data;
  const entries = data?.entries ?? [];
  const tiers = data?.tierDistribution ?? [];
  const scanErrors = data?.errors ?? [];

  const noBudget = entries.filter((e) => !e.hasBudget);
  const regressed = entries.filter((e) => e.regressed);
  const slowest = [...entries]
    .map((e) => ({ entry: e, total: Number(e.goBuildMs) + Number(e.uiBuildMs) }))
    .filter((x) => x.total > 0)
    .sort((a, b) => b.total - a.total)
    .slice(0, 10);

  return (
    <section
      data-testid={selectors.pages.fleet}
      aria-labelledby="fleet-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-wrap items-center gap-3">
        <div className="flex flex-col gap-1">
          <h2 id="fleet-heading" className="text-2xl font-semibold">
            {t(strings.fleet.title)}
          </h2>
          <p className="text-app-muted-foreground">{t(strings.fleet.description)}</p>
        </div>
        <Button
          data-testid={selectors.fleet.refreshButton}
          variant="outline"
          size="sm"
          className="ms-auto"
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
          aria-label={t(strings.fleet.refresh)}
        >
          <RefreshCw
            aria-hidden="true"
            className={["h-4 w-4", query.isFetching ? "animate-spin" : ""].join(" ")}
          />
        </Button>
      </header>

      {query.isLoading && (
        <LoadingState
          testId={selectors.fleet.loading}
          title={t(strings.fleet.loadingTitle)}
          skeleton={
            <div className="flex flex-col gap-4">
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-20 w-full" />
                ))}
              </div>
              <Skeleton className="h-40 w-full" />
            </div>
          }
        />
      )}
      {query.error && (
        <ErrorState
          testId={selectors.fleet.error}
          title={t(strings.fleet.errorTitle)}
          message={errorMessage(query.error, t)}
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
        />
      )}

      {data && !query.error && entries.length === 0 && scanErrors.length === 0 && (
        <EmptyState
          testId={selectors.fleet.empty}
          title={t(strings.fleet.emptyTitle)}
          message={t(strings.fleet.empty)}
        />
      )}

      {data && (entries.length > 0 || scanErrors.length > 0) && (
        <>
          <dl
            data-testid={selectors.fleet.summary}
            className="grid grid-cols-2 gap-3 sm:grid-cols-3"
          >
            <Stat
              testId={selectors.fleet.summaryScenarios}
              label={t(strings.fleet.summary.scenarios)}
              value={data.scenarioCount}
            />
            <Stat
              testId={selectors.fleet.summaryNoBudget}
              label={t(strings.fleet.summary.noBudget)}
              value={data.noBudgetCount}
            />
            <Stat
              testId={selectors.fleet.summaryRegressed}
              label={t(strings.fleet.summary.regressed)}
              value={data.regressedCount}
            />
          </dl>

          <TierDistributionCard tiers={tiers} />

          <OffenderTable
            testId={selectors.fleet.slowest}
            title={t(strings.fleet.slowest.title)}
            emptyLabel={t(strings.fleet.slowest.empty)}
            rows={slowest.map((x) => x.entry)}
            onSelect={setScenario}
            showBuild
          />

          <OffenderTable
            testId={selectors.fleet.regressed}
            title={t(strings.fleet.regressedSection.title)}
            emptyLabel={t(strings.fleet.regressedSection.empty)}
            rows={regressed}
            onSelect={setScenario}
            showReason
          />

          <OffenderTable
            testId={selectors.fleet.noBudget}
            title={t(strings.fleet.noBudgetSection.title)}
            emptyLabel={t(strings.fleet.noBudgetSection.empty)}
            rows={noBudget}
            onSelect={setScenario}
          />

          {scanErrors.length > 0 && (
            <section
              data-testid={selectors.fleet.errors}
              aria-label={t(strings.fleet.errors.title)}
              className="rounded-panel border border-app-danger/40 bg-app-danger/5 p-4"
            >
              <h3 className="flex items-center gap-2 text-sm font-medium text-app-danger">
                <AlertTriangle aria-hidden="true" className="h-4 w-4" />
                {t(strings.fleet.errors.title)}
              </h3>
              <ul className="mt-3 flex flex-col gap-1 text-sm">
                {scanErrors.map((e) => (
                  <li key={e.scenario} className="flex flex-wrap gap-2">
                    <span className="font-medium text-app-foreground">{e.scenario}</span>
                    <span className="text-app-muted-foreground">{e.reason}</span>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </>
      )}
    </section>
  );
}

function Stat({ testId, label, value }: { testId: string; label: string; value: number }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-4">
      <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</dt>
      <dd data-testid={testId} className="mt-2 text-2xl font-semibold tabular-nums">
        {value}
      </dd>
    </div>
  );
}

function TierDistributionCard({
  tiers,
}: {
  tiers: { tier: string; scenarioCount: number }[];
}) {
  const { t } = useTranslation();
  if (tiers.length === 0) return null;
  return (
    <section
      data-testid={selectors.fleet.tiers}
      aria-label={t(strings.fleet.tiers.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-medium text-app-muted-foreground">
        {t(strings.fleet.tiers.title)}
      </h3>
      <ul className="mt-3 flex flex-wrap gap-3">
        {tiers.map((d) => {
          const key = fleetTierKey(d.tier);
          return (
            <li
              key={d.tier}
              data-testid={selectors.fleet.tierRow({ tier: d.tier })}
              className="flex items-center gap-2 text-sm"
            >
              <span
                className={[
                  "rounded-control px-1.5 py-0.5 text-xs font-semibold uppercase",
                  tierChipClass(key),
                ].join(" ")}
              >
                {t(TIER_LABEL_KEY[key])}
              </span>
              <span className="tabular-nums text-app-muted-foreground">{d.scenarioCount}</span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

function OffenderTable({
  testId,
  title,
  emptyLabel,
  rows,
  onSelect,
  showBuild,
  showReason,
}: {
  testId: string;
  title: string;
  emptyLabel: string;
  rows: FleetScenarioEntry[];
  onSelect: (scenario: string) => void;
  showBuild?: boolean;
  showReason?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={testId}
      aria-label={title}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-medium text-app-muted-foreground">{title}</h3>
      {rows.length === 0 ? (
        <p className="mt-3 text-app-muted-foreground">{emptyLabel}</p>
      ) : (
        <div className="mt-3 overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="text-xs uppercase tracking-wide text-app-muted-foreground">
                <th scope="col" className="px-2 py-1 text-start font-medium">
                  {t(strings.fleet.col.scenario)}
                </th>
                <th scope="col" className="px-2 py-1 text-start font-medium">
                  {t(strings.fleet.col.tier)}
                </th>
                {showBuild && (
                  <>
                    <th scope="col" className="px-2 py-1 text-end font-medium">
                      {t(strings.fleet.col.goBuild)}
                    </th>
                    <th scope="col" className="px-2 py-1 text-end font-medium">
                      {t(strings.fleet.col.uiBuild)}
                    </th>
                  </>
                )}
                {showReason && (
                  <th scope="col" className="px-2 py-1 text-start font-medium">
                    {t(strings.fleet.col.reason)}
                  </th>
                )}
              </tr>
            </thead>
            <tbody>
              {rows.map((e) => {
                const key = fleetTierKey(e.tier);
                return (
                  <tr
                    key={e.scenario}
                    data-testid={selectors.fleet.scenarioRow({ scenario: e.scenario })}
                    className="border-t border-app-border"
                  >
                    <td className="px-2 py-1.5">
                      <Link
                        to="/audit"
                        onClick={() => onSelect(e.scenario)}
                        className="font-medium text-app-primary underline"
                      >
                        {e.scenario}
                      </Link>
                      {e.degradedReason && (
                        <span
                          className="ms-2 text-xs text-app-warning"
                          title={e.degradedReason}
                        >
                          ⚠
                        </span>
                      )}
                    </td>
                    <td className="px-2 py-1.5">
                      <span
                        className={[
                          "rounded-control px-1.5 py-0.5 text-xs font-semibold uppercase",
                          tierChipClass(key),
                        ].join(" ")}
                      >
                        {t(TIER_LABEL_KEY[key])}
                      </span>
                    </td>
                    {showBuild && (
                      <>
                        <td className="px-2 py-1.5 text-end tabular-nums">
                          {formatMs(e.goBuildMs)}
                        </td>
                        <td className="px-2 py-1.5 text-end tabular-nums">
                          {formatMs(e.uiBuildMs)}
                        </td>
                      </>
                    )}
                    {showReason && (
                      <td className="px-2 py-1.5 text-app-muted-foreground">
                        {e.degradedReason || t(strings.fleet.regressedSection.flagged)}
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
