import { useQuery } from "@tanstack/react-query";

import { trialsClient } from "../../api/trials";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { pct, verdictLabel } from "../labels";

/**
 * TrialsBoard renders the empirical local-model gate (OT-P1-001): Guide-gate
 * coverage, the success-rate trend, and the most recent runs. Read-only — trials
 * run on explicit invocation via the CLI (`trials run`), never from the console.
 */
export function TrialsBoard() {
  const { t } = useTranslation();
  const coverage = useQuery({
    queryKey: ["trials", "coverage"],
    queryFn: () => trialsClient.getGateCoverage({}),
  });
  const history = useQuery({
    queryKey: ["trials", "history"],
    queryFn: () => trialsClient.getTrialHistory({}),
  });

  if (coverage.isLoading || history.isLoading) {
    return (
      <p data-testid={selectors.trials.loading} className="text-app-muted-foreground">
        {t(strings.common.loading)}
      </p>
    );
  }
  if (coverage.error || history.error) {
    return (
      <p data-testid={selectors.trials.error} className="text-red-400">
        {t(strings.common.error)}
      </p>
    );
  }

  const points = history.data?.points ?? [];
  const recent = history.data?.recentRuns ?? [];
  const gc = coverage.data;

  return (
    <div className="flex flex-col gap-6">
      <section
        data-testid={selectors.trials.coverage}
        aria-labelledby="trials-coverage-heading"
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 id="trials-coverage-heading" className="text-xs uppercase tracking-wide text-app-muted-foreground">
          {t(strings.pages.trials.coverageHeading)}
        </h3>
        <p className="mt-2 text-sm">
          {t(strings.pages.trials.coverageValue, {
            rate: gc ? pct(gc.gateCoverageRatio) : "0",
            withGate: gc?.guideTasksWithGate ?? 0,
            total: gc?.guideTasksTotal ?? 0,
          })}
        </p>
      </section>

      {points.length === 0 && recent.length === 0 ? (
        <p data-testid={selectors.trials.empty} className="text-app-muted-foreground">
          {t(strings.pages.trials.empty)}
        </p>
      ) : (
        <>
          <section aria-labelledby="trials-trend-heading" className="flex flex-col gap-3">
            <h3 id="trials-trend-heading" className="text-lg font-semibold">
              {t(strings.pages.trials.historyHeading)}
            </h3>
            <ul className="flex flex-col gap-2">
              {points.map((p, i) => (
                <li
                  key={i}
                  data-testid={selectors.trials.point}
                  className="rounded-panel border border-app-border bg-app-surface p-3 text-sm"
                >
                  {t(strings.pages.trials.historyPoint, {
                    date: p.at ? formatDate(timestampToDate(p.at), { dateStyle: "medium" }) : "—",
                    rate: pct(p.successRate),
                    tokens: Number(p.medianTokens),
                    runs: p.runCount,
                  })}
                </li>
              ))}
            </ul>
          </section>

          <section aria-labelledby="trials-recent-heading" className="flex flex-col gap-3">
            <h3 id="trials-recent-heading" className="text-lg font-semibold">
              {t(strings.pages.trials.recentHeading)}
            </h3>
            <ul className="flex flex-col gap-2">
              {recent.map((r) => (
                <li
                  key={r.id}
                  data-testid={selectors.trials.run}
                  className="rounded-panel border border-app-border bg-app-surface p-3 text-sm"
                >
                  <span className="rounded bg-app-border px-2 py-0.5 text-xs font-mono">
                    {verdictLabel(r.verdict)}
                  </span>{" "}
                  <span className="font-medium">{r.suite}</span>{" "}
                  <span className="text-app-muted-foreground">
                    {t(strings.pages.trials.runMetrics, {
                      tokens: Number(r.tokens),
                      ms: Number(r.durationMs),
                    })}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        </>
      )}
    </div>
  );
}

/** timestampToDate converts a protobuf Timestamp message to a JS Date. */
function timestampToDate(ts: { seconds: bigint; nanos: number }): Date {
  return new Date(Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000));
}
