import { timestampDate } from "@bufbuild/protobuf/wkt";

import type { ScoreSnapshot, TrendSummary } from "../../api/scoring";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";

interface TrendCardProps {
  snapshots: ScoreSnapshot[];
  trend?: TrendSummary;
}

function formatSignedDelta(delta: number): string {
  if (delta > 0) return `+${delta}`;
  return `${delta}`;
}

export function TrendCard({ snapshots, trend }: TrendCardProps) {
  const { t } = useTranslation();
  const chronological = [...snapshots].reverse();

  return (
    <section
      data-testid={selectors.scoring.trend.card}
      aria-label={t(strings.scoring.trend.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.scoring.trend.title)}
      </h3>
      {trend ? (
        <div data-testid={selectors.scoring.trend.delta} className="mt-2 flex flex-wrap items-baseline gap-2">
          <span className="text-2xl font-semibold">{formatSignedDelta(trend.delta)}</span>
          <span className="text-sm text-app-muted-foreground">
            {t(strings.scoring.trend.since, {
              score: trend.previousScore,
              date: trend.previousCalculatedAt
                ? formatDate(timestampDate(trend.previousCalculatedAt), { dateStyle: "medium" })
                : t(strings.scoring.trend.unknownDate),
            })}
          </span>
        </div>
      ) : (
        <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.scoring.trend.empty)}</p>
      )}
      <div data-testid={selectors.scoring.trend.series} className="mt-4 grid min-h-28 grid-cols-12 items-end gap-1">
        {chronological.slice(-12).map((snapshot) => (
          <div key={`${snapshot.digest}-${snapshot.calculatedAt?.seconds ?? 0}`} className="flex h-28 items-end">
            <div
              className="w-full rounded-t-sm bg-emerald-500"
              style={{ height: `${Math.max(6, Math.min(100, snapshot.score))}%` }}
              title={`${snapshot.score}/100`}
            />
          </div>
        ))}
      </div>
    </section>
  );
}
