import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { Button } from "../../components/ui/button";
import { EmptyState, ErrorState, LoadingState, Skeleton } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { perfClient, type StartupMeasurement, type TrendSample } from "../../api/perf";
import { useScenario } from "../perf/scenarioContextValue";
import { ScenarioPicker } from "../perf/ScenarioPicker";
import { formatBytes, formatMs, formatMsFloat, formatTimestamp } from "../perf/format";
import { Sparkline } from "./Sparkline";

/**
 * "Scenario perf over time" workflow. Plots the perf trend series from GetTrend
 * (build time, bundle size, LCP, p95, slowest-component) and the startup trend
 * from GetStartupTrend, oldest → newest. Each metric shows its latest value and
 * a sparkline. All real data — the series are exactly what the backend persists.
 */
export function TrendsView() {
  const { t } = useTranslation();
  const { scenario } = useScenario();

  const trend = useQuery({
    queryKey: ["trend", scenario],
    queryFn: () => perfClient.getTrend({ scenario }),
  });
  const startup = useQuery({
    queryKey: ["startup-trend", scenario],
    queryFn: () => perfClient.getStartupTrend({ scenario }),
  });

  // Backend returns newest → oldest; reverse so sparklines read left→right in time.
  const samples: TrendSample[] = [...(trend.data?.samples ?? [])].reverse();
  const measurements: StartupMeasurement[] = [...(startup.data?.measurements ?? [])].reverse();

  return (
    <section
      data-testid={selectors.pages.trends}
      aria-labelledby="trends-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-3">
        <h2 id="trends-heading" className="text-2xl font-semibold">
          {t(strings.trends.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.trends.description)}</p>
        <ScenarioPicker />
      </header>

      {trend.error && (
        <ErrorState
          testId={selectors.trends.error}
          title={t(strings.trends.errorTitle)}
          message={errorMessage(trend.error, t)}
          onRetry={() => {
            void trend.refetch();
            void startup.refetch();
          }}
          retrying={trend.isFetching}
        />
      )}
      {trend.isLoading && (
        <LoadingState
          title={t(strings.trends.loadingTitle)}
          skeleton={
            <div className="grid gap-4 lg:grid-cols-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-28 w-full" />
              ))}
            </div>
          }
        />
      )}

      {trend.data && !trend.error && samples.length === 0 && measurements.length === 0 && (
        <EmptyState
          testId={selectors.trends.empty}
          title={t(strings.trends.emptyTitle)}
          message={t(strings.trends.empty)}
          actionSlot={
            <Button asChild size="sm">
              <Link to="/audit">{t(strings.trends.emptyCta)}</Link>
            </Button>
          }
        />
      )}

      {(samples.length > 0 || measurements.length > 0) && (
        <div data-testid={selectors.trends.charts} className="grid gap-4 lg:grid-cols-2">
          <MetricCard
            testId={selectors.trends.cardGoBuild}
            title={t(strings.trends.metric.goBuild)}
            latest={formatMs(samples.at(-1)?.goBuildMs)}
            values={samples.map((s) => Number(s.goBuildMs))}
          />
          <MetricCard
            testId={selectors.trends.cardUiBuild}
            title={t(strings.trends.metric.uiBuild)}
            latest={formatMs(samples.at(-1)?.uiBuildMs)}
            values={samples.map((s) => Number(s.uiBuildMs))}
          />
          <MetricCard
            testId={selectors.trends.cardBundle}
            title={t(strings.trends.metric.bundle)}
            latest={formatBytes(samples.at(-1)?.bundleBytes)}
            values={samples.map((s) => Number(s.bundleBytes))}
          />
          <MetricCard
            testId={selectors.trends.cardLcp}
            title={t(strings.trends.metric.lcp)}
            latest={formatMs(samples.at(-1)?.lcpMs)}
            values={samples.map((s) => Number(s.lcpMs))}
          />
          <MetricCard
            testId={selectors.trends.cardP95}
            title={t(strings.trends.metric.p95)}
            latest={formatMs(samples.at(-1)?.p95Ms)}
            values={samples.map((s) => Number(s.p95Ms))}
          />
          <MetricCard
            testId={selectors.trends.cardComponent}
            title={t(strings.trends.metric.component)}
            latest={
              samples.at(-1)?.slowestComponent
                ? `${samples.at(-1)?.slowestComponent} (${formatMsFloat(samples.at(-1)?.slowestComponentAvgMs)})`
                : "—"
            }
            values={samples.map((s) => s.slowestComponentAvgMs)}
          />
          <MetricCard
            testId={selectors.trends.cardStartup}
            title={t(strings.trends.metric.startup)}
            latest={formatMs(measurements.at(-1)?.timeToHealthyMs)}
            values={measurements.map((m) => Number(m.timeToHealthyMs))}
          />
        </div>
      )}

      {samples.length > 0 && (
        <SamplesTable samples={samples} />
      )}
    </section>
  );
}

function MetricCard({
  testId,
  title,
  latest,
  values,
}: {
  testId: string;
  title: string;
  latest: string;
  values: number[];
}) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={testId}
      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <p className="text-xs uppercase tracking-wide text-app-muted-foreground">{title}</p>
      <p className="text-2xl font-semibold tabular-nums">{latest}</p>
      <div className="text-app-primary">
        <Sparkline values={values} label={t(strings.trends.sparklineLabel, { metric: title })} />
      </div>
    </div>
  );
}

function SamplesTable({ samples }: { samples: TrendSample[] }) {
  const { t } = useTranslation();
  // Newest first in the table for scanability.
  const rows = [...samples].reverse();
  return (
    <section
      data-testid={selectors.trends.samples}
      aria-label={t(strings.trends.samplesTitle)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-medium text-app-muted-foreground">
        {t(strings.trends.samplesTitle)}
      </h3>
      <div className="mt-3 overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="text-xs uppercase tracking-wide text-app-muted-foreground">
              <th scope="col" className="px-2 py-1 text-start font-medium">
                {t(strings.trends.col.captured)}
              </th>
              <th scope="col" className="px-2 py-1 text-end font-medium">
                {t(strings.trends.metric.goBuild)}
              </th>
              <th scope="col" className="px-2 py-1 text-end font-medium">
                {t(strings.trends.metric.uiBuild)}
              </th>
              <th scope="col" className="px-2 py-1 text-end font-medium">
                {t(strings.trends.metric.lcp)}
              </th>
              <th scope="col" className="px-2 py-1 text-start font-medium">
                {t(strings.trends.col.note)}
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((s, i) => (
              <tr key={`${s.capturedAt}:${i}`} className="border-t border-app-border">
                <td className="px-2 py-1.5 whitespace-nowrap">{formatTimestamp(s.capturedAt)}</td>
                <td className="px-2 py-1.5 text-end tabular-nums">{formatMs(s.goBuildMs)}</td>
                <td className="px-2 py-1.5 text-end tabular-nums">{formatMs(s.uiBuildMs)}</td>
                <td className="px-2 py-1.5 text-end tabular-nums">{formatMs(s.lcpMs)}</td>
                <td className="px-2 py-1.5 text-app-muted-foreground">{s.note || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
