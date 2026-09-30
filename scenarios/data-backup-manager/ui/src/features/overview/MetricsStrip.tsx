import { Coins } from "lucide-react";

import { AsyncSection } from "../../components/AsyncSection";
import { useRunStats } from "../../hooks/useRuns";
import { formatBytes } from "../../lib/format";
import { formatNumber } from "../../i18n/format";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Backup economics — the one-glance answer to "what is this actually costing me
 * on disk?" Physical (deduped + compressed) bytes vs the logical total, with the
 * dedup ratio. The physical figure is an approximate repo-size delta measured
 * around each run (labeled as such); when no run in the window could measure
 * repo growth we say so rather than show a misleading 0× ratio.
 */
export function MetricsStrip() {
  const { t } = useTranslation();
  const { data, isLoading, isError, refetch } = useRunStats();

  const physical = data ? Number(data.totalPhysicalBytes) : 0;
  const ratio = data?.dedupRatio ?? 0;
  const window = data ? Number(data.window) : 0;
  const hasPhysical = physical > 0 && ratio > 0;

  return (
    <section data-testid={selectors.overview.metrics} className="flex flex-col gap-2">
      <h2 className="flex items-center gap-1.5 text-sm font-semibold text-app-foreground">
        <Coins className="size-4 text-app-muted-foreground" aria-hidden />
        {t(strings.overview.metricsHeading)}
      </h2>
      <AsyncSection isLoading={isLoading} isError={isError} onRetry={() => void refetch()}>
        <div className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-3">
          {hasPhysical ? (
            <p className="text-sm text-app-foreground">
              {t(strings.overview.metricsDedup, {
                physical: formatBytes(physical),
                ratio: formatNumber(ratio, { maximumFractionDigits: 1 }),
              })}{" "}
              <span className="text-xs text-app-muted-foreground">
                ({t(strings.overview.metricsDedupApprox)})
              </span>
            </p>
          ) : (
            <p className="text-sm text-app-muted-foreground">{t(strings.overview.metricsDedupEmpty)}</p>
          )}
          <p className="text-xs text-app-muted-foreground">
            {t(strings.overview.metricsWindow, { count: window })}
          </p>
        </div>
      </AsyncSection>
    </section>
  );
}
