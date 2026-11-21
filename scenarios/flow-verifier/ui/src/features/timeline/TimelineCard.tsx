import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { fetchRuns, type RunRow } from "../../api/inventory";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";

const TIMELINE_KEY = (flowId?: string) =>
  flowId ? (["runs", "timeline", flowId] as const) : (["runs", "timeline"] as const);

interface TimelineCardProps {
  flowId?: string;
  limit?: number;
}

type DayBucket = {
  day: string;
  passed: number;
  failed: number;
  error: number;
};

function bucketRunsByDay(runs: RunRow[]): DayBucket[] {
  const map = new Map<string, DayBucket>();
  for (const r of runs) {
    const stamp = r.finishedAt || r.startedAt;
    if (!stamp) continue;
    const date = new Date(stamp);
    if (Number.isNaN(date.getTime())) continue;
    const day = date.toISOString().slice(0, 10);
    let bucket = map.get(day);
    if (!bucket) {
      bucket = { day, passed: 0, failed: 0, error: 0 };
      map.set(day, bucket);
    }
    if (r.status === "passed") bucket.passed += 1;
    else if (r.status === "failed") bucket.failed += 1;
    else bucket.error += 1;
  }
  return Array.from(map.values()).sort((a, b) => a.day.localeCompare(b.day));
}

export function TimelineCard({ flowId, limit = 200 }: TimelineCardProps = {}) {
  const { t } = useTranslation();
  const runsQuery = useQuery({
    queryKey: TIMELINE_KEY(flowId),
    queryFn: () => fetchRuns({ flowId, limit }),
  });

  const buckets = useMemo(
    () => bucketRunsByDay(runsQuery.data ?? []),
    [runsQuery.data],
  );

  return (
    <section
      data-testid="timeline-card"
      aria-label={t("timeline.title", { defaultValue: "Verification timeline" })}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h2 className="text-sm font-medium text-app-foreground">
        {t("timeline.title", { defaultValue: "Verification timeline" })}
      </h2>

      {runsQuery.isLoading && (
        <p data-testid="timeline-loading" className="mt-3 text-sm text-app-muted-foreground">
          {t("timeline.loading", { defaultValue: "Loading run history…" })}
        </p>
      )}

      {runsQuery.error && (
        <p data-testid="timeline-error" className="mt-3 text-sm text-app-danger">
          {errorMessage(runsQuery.error, t)}
        </p>
      )}

      {!runsQuery.isLoading && !runsQuery.error && buckets.length < 2 && (
        <p data-testid="timeline-empty" className="mt-3 text-sm text-app-muted-foreground">
          {t("timeline.empty", {
            defaultValue:
              "Run the verifier on at least two different days to see the trend.",
          })}
        </p>
      )}

      {!runsQuery.isLoading && !runsQuery.error && buckets.length >= 2 && (
        <div data-testid="timeline-chart" className="mt-3 h-64 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={buckets} margin={{ top: 8, right: 8, left: -8, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
              <XAxis dataKey="day" stroke="var(--color-muted-foreground)" fontSize={11} />
              <YAxis stroke="var(--color-muted-foreground)" fontSize={11} allowDecimals={false} />
              <Tooltip
                contentStyle={{
                  background: "var(--color-surface)",
                  border: "1px solid var(--color-border)",
                  color: "var(--color-foreground)",
                  fontSize: 12,
                }}
                labelStyle={{ color: "var(--color-foreground)" }}
              />
              <Legend wrapperStyle={{ fontSize: 12 }} />
              <Bar dataKey="passed" stackId="a" fill="var(--color-success)" />
              <Bar dataKey="failed" stackId="a" fill="var(--color-danger)" />
              <Bar dataKey="error" stackId="a" fill="var(--color-warning)" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}
    </section>
  );
}
