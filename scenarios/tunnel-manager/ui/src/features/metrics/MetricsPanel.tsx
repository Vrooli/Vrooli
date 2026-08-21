import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { MetricsSample } from "@vrooli/proto-types/tunnel-manager/v1/tunnel/tunnel_pb";
import type { ProbeResult, RouteClassification } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";
import { FailureClass } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

import { Button } from "../../components/ui/button";
import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { tunnelClient } from "../../api/tunnel";
import { probesClient } from "../../api/probes";
import {
  probeKindLabel,
  probeStatusLabel,
  probeStatusTone,
  failureClassLabel,
  failureClassTone,
} from "./labels";

const METRICS_KEY = ["metrics"] as const;
const PROBES_KEY = ["probes"] as const;
const CLASSIFY_KEY = ["probes-classify"] as const;

function sampleTime(sample: MetricsSample): number {
  return sample.scrapedAt ? timestampDate(sample.scrapedAt).getTime() : 0;
}

function latestSample(samples: MetricsSample[]): MetricsSample | undefined {
  return [...samples].sort((a, b) => sampleTime(b) - sampleTime(a))[0];
}

function latestProbe(probes: ProbeResult[]): ProbeResult | undefined {
  return [...probes].sort((a, b) => {
    const bTime = b.createdAt ? timestampDate(b.createdAt).getTime() : 0;
    const aTime = a.createdAt ? timestampDate(a.createdAt).getTime() : 0;
    return bTime - aTime;
  })[0];
}

/**
 * MetricsPanel pairs the tunnel-wide Prometheus metrics time-series with the
 * per-route probe history and reachability classification. Operators read the
 * tunnel signal (HA connections / errors / RTT) and drill into which specific
 * route is failing and why.
 */
export function MetricsPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const metricsQuery = useQuery({ queryKey: METRICS_KEY, queryFn: () => tunnelClient.listMetrics({}) });
  const probesQuery = useQuery({ queryKey: PROBES_KEY, queryFn: () => probesClient.listProbes({}) });
  const classifyQuery = useQuery({ queryKey: CLASSIFY_KEY, queryFn: () => probesClient.classify({}) });

  const scrapeMutation = useMutation({
    mutationFn: () => tunnelClient.scrape({}),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: METRICS_KEY }),
  });

  const runProbesMutation = useMutation({
    mutationFn: () => probesClient.runProbes({}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: PROBES_KEY });
      void queryClient.invalidateQueries({ queryKey: CLASSIFY_KEY });
    },
  });

  const samples = metricsQuery.data?.samples ?? [];
  const probes = probesQuery.data?.results ?? [];
  const classifications = classifyQuery.data?.classifications ?? [];
  const latest = latestSample(samples);
  const latestProbeResult = latestProbe(probes);
  const unhealthyRoutes = classifications.filter(
    (cls) => cls.classification !== FailureClass.HEALTHY && cls.classification !== FailureClass.UNSPECIFIED,
  );
  const metricsLoading = metricsQuery.isLoading || probesQuery.isLoading || classifyQuery.isLoading;
  const metricsError = metricsQuery.error || probesQuery.error || classifyQuery.error;
  const experienceState = metricsLoading ? "loading" : metricsError && samples.length === 0 && probes.length === 0 ? "error" : samples.length === 0 && probes.length === 0 ? "empty" : metricsError ? "partial" : "ready";

  return (
    <div className="flex flex-col gap-8">
      <section data-testid={selectors.metrics.panel} data-experience-surface="metrics-results" data-experience-state={experienceState} className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t(strings.metrics.heading)}</h2>
          <Button
            data-testid={selectors.metrics.scrapeButton}
            disabled={scrapeMutation.isPending}
            onClick={() => scrapeMutation.mutate()}
          >
            {t(strings.metrics.scrapeButton)}
          </Button>
        </div>
        {(scrapeMutation.isError || scrapeMutation.isSuccess) && (
          <p
            data-testid={selectors.metrics.scrapeActionFeedback}
            role={scrapeMutation.isError ? "alert" : undefined}
            className={`text-sm ${scrapeMutation.isError ? "text-app-danger" : "text-app-success"}`}
          >
            {scrapeMutation.isError ? t(strings.metrics.scrapeError) : t(strings.metrics.scrapeSuccess)}
          </p>
        )}

        <div data-testid={selectors.metrics.summary} className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <SummaryStat
            label={t(strings.metrics.latestHaConnections)}
            value={latest ? latest.haConnections : t(strings.common.notAvailable)}
            tone={latest && latest.haConnections < 4 ? "warning" : "success"}
          />
          <SummaryStat
            label={t(strings.metrics.latestRequestErrors)}
            value={latest ? latest.requestErrors : t(strings.common.notAvailable)}
            tone={latest && latest.requestErrors > 0 ? "warning" : "success"}
          />
          <SummaryStat
            label={t(strings.metrics.latestRtt)}
            value={latest ? latest.smoothedRttMs.toFixed(1) : t(strings.common.notAvailable)}
          />
          <SummaryStat
            label={t(strings.metrics.latestScrape)}
            value={
              latest?.scrapedAt
                ? formatDate(timestampDate(latest.scrapedAt), { dateStyle: "short", timeStyle: "short" })
                : t(strings.common.never)
            }
          />
        </div>

        <QueryState
          isLoading={metricsQuery.isLoading}
          error={metricsQuery.error}
          isEmpty={samples.length === 0}
          loadingLabel={t(strings.metrics.loading)}
          errorLabel={t(strings.metrics.error)}
          emptyLabel={t(strings.metrics.empty)}
        >
          <div className="overflow-x-auto rounded-panel border border-app-border">
            <table data-testid={selectors.metrics.table} className="w-full text-left text-sm">
              <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
                <tr>
                  <th className="px-3 py-2">{t(strings.metrics.colTime)}</th>
                  <th className="px-3 py-2">{t(strings.metrics.colHaConnections)}</th>
                  <th className="px-3 py-2">{t(strings.metrics.colActiveStreams)}</th>
                  <th className="px-3 py-2">{t(strings.metrics.colRequestErrors)}</th>
                  <th className="px-3 py-2">{t(strings.metrics.colRtt)}</th>
                </tr>
              </thead>
              <tbody>
                {samples.map((sample: MetricsSample) => (
                  <tr
                    key={sample.id}
                    data-testid={selectors.metrics.row}
                    className="border-b border-app-border last:border-0 tabular-nums"
                  >
                    <td className="px-3 py-2">
                      {sample.scrapedAt
                        ? formatDate(timestampDate(sample.scrapedAt), { dateStyle: "short", timeStyle: "medium" })
                        : "—"}
                    </td>
                    <td className="px-3 py-2">{sample.haConnections}</td>
                    <td className="px-3 py-2">{sample.activeStreams}</td>
                    <td className="px-3 py-2">{sample.requestErrors}</td>
                    <td className="px-3 py-2">{sample.smoothedRttMs.toFixed(1)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </QueryState>
      </section>

      <section data-testid={selectors.metrics.probesPanel} className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t(strings.metrics.probesHeading)}</h2>
          <Button
            data-testid={selectors.metrics.runProbesButton}
            disabled={runProbesMutation.isPending}
            onClick={() => runProbesMutation.mutate()}
          >
            {t(strings.metrics.runProbesButton)}
          </Button>
        </div>
        {(runProbesMutation.isError || runProbesMutation.isSuccess) && (
          <p
            data-testid={selectors.metrics.probesActionFeedback}
            role={runProbesMutation.isError ? "alert" : undefined}
            className={`text-sm ${runProbesMutation.isError ? "text-app-danger" : "text-app-success"}`}
          >
            {runProbesMutation.isError
              ? t(strings.metrics.probesErrorAction)
              : t(strings.metrics.probesSuccess)}
          </p>
        )}

        <div className="grid gap-3 sm:grid-cols-3">
          <SummaryStat label={t(strings.metrics.classifiedRoutes)} value={classifyQuery.error ? t(strings.common.notAvailable) : classifications.length} testId={selectors.metrics.classifiedCount} />
          <SummaryStat
            label={t(strings.metrics.routesNeedingAttention)}
            value={unhealthyRoutes.length}
            tone={unhealthyRoutes.length > 0 ? "danger" : "success"}
            testId={selectors.metrics.classCount}
          />
          <SummaryStat
            label={t(strings.metrics.latestProbe)}
            value={
              latestProbeResult?.createdAt
                ? formatDate(timestampDate(latestProbeResult.createdAt), { dateStyle: "short", timeStyle: "short" })
                : t(strings.common.never)
            }
          />
        </div>

        <p
          data-testid={selectors.metrics.limitation}
          className="rounded-panel border border-app-border bg-app-surface-muted px-3 py-2 text-sm text-app-muted-foreground"
        >
          {t(strings.metrics.classificationScope)}
        </p>

        <QueryState
          isLoading={classifyQuery.isLoading}
          error={classifyQuery.error}
          onRetry={() => void classifyQuery.refetch()}
          loadingLabel={t(strings.metrics.loading)}
          errorLabel={t(strings.overview.classificationUnavailable)}
        >
        {classifications.length > 0 && (
          <div data-testid={selectors.metrics.classification} className="flex flex-col gap-2">
            <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
              {t(strings.metrics.classificationHeading)}
            </h3>
            <ul className="flex flex-wrap gap-2">
              {classifications.map((cls: RouteClassification) => (
                <li
                  key={cls.subdomain}
                  className="flex max-w-full flex-wrap items-center gap-2 rounded-panel border border-app-border bg-app-surface px-3 py-2 text-sm"
                >
                  <span className="font-medium">{cls.subdomain}</span>
                  <StatusBadge tone={failureClassTone(cls.classification)}>
                    {t(failureClassLabel(cls.classification))}
                  </StatusBadge>
                  {cls.assessment && (
                    <span className="text-app-muted-foreground">{cls.assessment}</span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}
        </QueryState>

        <QueryState
          isLoading={probesQuery.isLoading}
          error={probesQuery.error}
          isEmpty={probes.length === 0}
          errorLabel={t(strings.metrics.probesError)}
          emptyLabel={t(strings.metrics.probesEmpty)}
        >
          <div className="overflow-x-auto rounded-panel border border-app-border">
            <table data-testid={selectors.metrics.probesTable} className="w-full text-left text-sm">
              <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
                <tr>
                  <th className="px-3 py-2">{t(strings.metrics.colTime)}</th>
                  <th className="px-3 py-2">{t(strings.metrics.colRoute)}</th>
                  <th className="px-3 py-2">{t(strings.metrics.colKind)}</th>
                  <th className="px-3 py-2">{t(strings.metrics.colStatus)}</th>
                  <th className="px-3 py-2">{t(strings.metrics.colLatency)}</th>
                </tr>
              </thead>
              <tbody>
                {probes.map((probe: ProbeResult) => (
                  <tr
                    key={probe.id}
                    data-testid={selectors.metrics.probesRow}
                    className="border-b border-app-border last:border-0"
                  >
                    <td className="px-3 py-2">
                      {probe.createdAt
                        ? formatDate(timestampDate(probe.createdAt), { dateStyle: "short", timeStyle: "medium" })
                        : "—"}
                    </td>
                    <td className="px-3 py-2 font-medium">{probe.subdomain}</td>
                    <td className="px-3 py-2">{t(probeKindLabel(probe.kind))}</td>
                    <td className="px-3 py-2">
                      <StatusBadge tone={probeStatusTone(probe.status)}>
                        {t(probeStatusLabel(probe.status))}
                      </StatusBadge>
                    </td>
                    <td className="px-3 py-2 tabular-nums">{probe.latencyMs}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </QueryState>
      </section>
    </div>
  );
}

function SummaryStat({
  label,
  value,
  tone = "neutral",
  testId,
}: {
  label: string;
  value: string | number;
  tone?: "success" | "warning" | "danger" | "neutral";
  testId?: string;
}) {
  const toneClass = {
    success: "text-app-success",
    warning: "text-app-warning",
    danger: "text-app-danger",
    neutral: "text-app-foreground",
  }[tone];

  return (
    <div data-testid={testId} className="rounded-panel border border-app-border bg-app-surface p-3">
      <div className="text-xs uppercase text-app-muted-foreground">{label}</div>
      <div className={`mt-1 text-lg font-semibold tabular-nums ${toneClass}`}>{value}</div>
    </div>
  );
}
