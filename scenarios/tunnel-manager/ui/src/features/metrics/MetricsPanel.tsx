import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { MetricsSample } from "@vrooli/proto-types/tunnel-manager/v1/tunnel/tunnel_pb";
import type { ProbeResult, RouteClassification } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

import { Button } from "../../components/ui/button";
import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { tunnelClient } from "../../api/tunnel";
import { probesClient } from "../../api/probes";
import { probeKindLabel, probeStatusLabel, probeStatusTone, failureClassLabel, failureClassTone } from "./labels";

const METRICS_KEY = ["metrics"] as const;
const PROBES_KEY = ["probes"] as const;
const CLASSIFY_KEY = ["probes-classify"] as const;

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

  return (
    <div className="flex flex-col gap-8">
      <section data-testid={selectors.metrics.panel} className="flex flex-col gap-3">
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

        {classifications.length > 0 && (
          <div data-testid={selectors.metrics.classification} className="flex flex-col gap-2">
            <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
              {t(strings.metrics.classificationHeading)}
            </h3>
            <ul className="flex flex-wrap gap-2">
              {classifications.map((cls: RouteClassification) => (
                <li key={cls.subdomain} className="flex items-center gap-2 text-sm">
                  <span className="font-medium">{cls.subdomain}</span>
                  <StatusBadge tone={failureClassTone(cls.classification)}>
                    {t(failureClassLabel(cls.classification))}
                  </StatusBadge>
                </li>
              ))}
            </ul>
          </div>
        )}

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
