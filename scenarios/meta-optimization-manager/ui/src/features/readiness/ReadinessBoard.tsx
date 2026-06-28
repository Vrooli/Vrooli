import { useQuery } from "@tanstack/react-query";

import { coverageClient } from "../../api/coverage";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { confidenceLabel, pct, projectionLabel } from "../labels";

/**
 * ReadinessBoard renders the per-projection coverage scoreboard (OT-P0-001):
 * each projection's now/in-reach/missing counts, coverage %, and
 * denominator-confidence, plus the latest empirical trial trend. Degrades
 * per-projection: an unavailable owner shows its honest reason inline.
 */
export function ReadinessBoard() {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ["coverage", "status"],
    queryFn: () => coverageClient.getStatus({}),
  });

  if (isLoading) {
    return (
      <p data-testid={selectors.readiness.loading} className="text-app-muted-foreground">
        {t(strings.common.loading)}
      </p>
    );
  }
  if (error) {
    return (
      <p data-testid={selectors.readiness.error} className="text-red-400">
        {t(strings.common.error)}
      </p>
    );
  }
  const projections = data?.projections ?? [];
  if (projections.length === 0) {
    return (
      <p data-testid={selectors.readiness.empty} className="text-app-muted-foreground">
        {t(strings.pages.dashboard.empty)}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <h3 className="text-lg font-semibold">{t(strings.pages.dashboard.projectionHeading)}</h3>
      <div className="grid gap-4 md:grid-cols-3">
        {projections.map((p) => (
          <article
            key={projectionLabel(p.projection)}
            data-testid={selectors.readiness.projection}
            className="rounded-panel border border-app-border bg-app-surface p-4"
          >
            <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
              {projectionLabel(p.projection)}
            </p>
            <p className="mt-1 text-3xl font-semibold">
              {p.available ? (
                <>
                  {pct(p.coverageRatio)}
                  <span className="ms-1 text-sm font-normal text-app-muted-foreground">
                    % {t(strings.pages.dashboard.coverageLabel)}
                  </span>
                </>
              ) : (
                // The live numerator join failed; a coverage % here would read
                // as "measured" when it was not. Show a dash; the honest reason
                // renders below.
                <span className="text-app-muted-foreground" title={p.unavailableReason}>
                  {"—"}
                </span>
              )}
            </p>
            {p.available && (
              <p className="mt-2 text-sm text-app-muted-foreground">
                {t(strings.pages.dashboard.cellsLabel, {
                  now: p.nowCount,
                  total: p.totalCells,
                  inReach: p.inReachCount,
                  missing: p.missingCount,
                })}
              </p>
            )}
            <p className="mt-1 text-xs text-app-muted-foreground">
              {t(strings.pages.dashboard.confidenceLabel)}: {confidenceLabel(p.denominatorConfidence)}
            </p>
            {!p.available && (
              <p className="mt-2 text-xs text-amber-500">
                {t(strings.pages.dashboard.unavailableReason, { reason: p.unavailableReason })}
              </p>
            )}
          </article>
        ))}
      </div>
      <TrialTrend trend={data?.latestTrialTrend} />
    </div>
  );
}

function TrialTrend({ trend }: { trend?: { successRate: number; medianTokens: bigint } }) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={selectors.readiness.trend}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
        {t(strings.pages.dashboard.trendHeading)}
      </p>
      {trend ? (
        <p className="mt-2 text-sm">
          {t(strings.pages.dashboard.trendValue, {
            rate: pct(trend.successRate),
            tokens: Number(trend.medianTokens),
          })}
        </p>
      ) : (
        <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.pages.dashboard.noTrend)}</p>
      )}
    </div>
  );
}
