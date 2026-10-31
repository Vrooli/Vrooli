import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { getVelocity } from "../../api/execution";
import { AsyncBoundary } from "../../components/AsyncBoundary";
import { PlanSelect } from "../../components/PlanSelect";
import { SectionPanel } from "../../components/Surfaces";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { type StringKey } from "../../consts/stringKey";
import { formatDate, formatNumber } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { bigintToNumber } from "../../lib/planStatus";
import { Completeness, type VelocityPoint } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const COMPLETENESS_LABELS: Record<Completeness, StringKey> = {
  [Completeness.UNSPECIFIED]: strings.pages.execution.completenessUnspecified,
  [Completeness.FULL]: strings.pages.execution.completenessFull,
  [Completeness.PARTIAL]: strings.pages.execution.completenessPartial,
};

/**
 * Tokens-per-run bar chart, drawn as inline SVG (no chart dependency). bigint
 * wire values are converted via `bigintToNumber` before any arithmetic so we
 * never render a raw bigint or overflow. The chart is decorative; the table
 * below it carries the same data accessibly.
 */
function TokensChart({ points }: { points: VelocityPoint[] }) {
  const { t } = useTranslation();
  const values = points.map((p) => bigintToNumber(p.tokens));
  const max = Math.max(1, ...values);
  const width = 100;
  const barGap = 2;
  const barWidth = points.length > 0 ? (width - barGap * (points.length - 1)) / points.length : width;

  return (
    <figure className="flex flex-col gap-2">
      <svg
        data-testid={selectors.velocity.chart}
        viewBox={`0 0 ${width} 40`}
        preserveAspectRatio="none"
        role="img"
        aria-label={t(strings.pages.velocity.chartLabel)}
        className="h-32 w-full rounded-control border border-app-border bg-app-surface-muted"
      >
        {points.map((point, i) => {
          const value = values[i] ?? 0;
          const barHeight = (value / max) * 36;
          const x = i * (barWidth + barGap);
          return (
            <rect
              key={point.id}
              x={x}
              y={40 - barHeight}
              width={barWidth}
              height={barHeight}
              className="fill-app-primary"
            >
              <title>{`${t(strings.pages.velocity.runLabel)} ${i + 1}: ${formatNumber(value)}`}</title>
            </rect>
          );
        })}
      </svg>
      <figcaption className="text-xs text-app-muted-foreground">
        {t(strings.pages.velocity.chartLabel)}
      </figcaption>
    </figure>
  );
}

/**
 * VelocityBoard — per-plan velocity trends. Pick a plan, then chart tokens per
 * run and tabulate wall-time, tokens, iterations, and completeness across runs.
 */
export function VelocityBoard() {
  const { t } = useTranslation();
  const [planId, setPlanId] = useState("");

  const velocity = useQuery({
    queryKey: ["velocity", planId],
    queryFn: () => getVelocity(planId),
    enabled: planId.length > 0,
  });

  const points = velocity.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      <SectionPanel title={t(strings.pages.velocity.planLabel)} headingId="velocity-plan-heading">
        <PlanSelect
          value={planId}
          onChange={setPlanId}
          label={t(strings.pages.velocity.planLabel)}
          testId={selectors.velocity.planSelect}
        />
      </SectionPanel>

      {planId.length > 0 ? (
        <AsyncBoundary
          isLoading={velocity.isLoading}
          error={velocity.error}
          isEmpty={points.length === 0}
          testIdPrefix={selectors.velocity.table}
          emptyLabel={t(strings.pages.velocity.empty)}
        >
          <SectionPanel title={t(strings.pages.velocity.title)} headingId="velocity-data-heading">
            <TokensChart points={points} />
            <div className="overflow-x-auto">
              <table
                data-testid={selectors.velocity.table}
                className="w-full min-w-[36rem] border-collapse text-sm"
              >
                <caption className="sr-only">{t(strings.pages.velocity.title)}</caption>
                <thead>
                  <tr className="border-b border-app-border text-left text-xs uppercase tracking-wide text-app-muted-foreground">
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t(strings.pages.velocity.runLabel)}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t(strings.pages.velocity.wallTime)}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t(strings.pages.velocity.tokens)}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t(strings.pages.velocity.iterations)}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t(strings.pages.velocity.completeness)}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t(strings.pages.velocity.recordedAt)}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {points.map((point) => (
                    <tr key={point.id} className="border-b border-app-border last:border-0">
                      <th scope="row" className="px-3 py-2 text-left font-mono text-xs text-app-foreground">
                        {point.runId || "—"}
                      </th>
                      <td className="px-3 py-2 text-app-muted-foreground">
                        {t(strings.pages.velocity.seconds, {
                          count: bigintToNumber(point.wallTimeSeconds),
                        })}
                      </td>
                      <td className="px-3 py-2 text-app-muted-foreground">
                        {formatNumber(bigintToNumber(point.tokens))}
                      </td>
                      <td className="px-3 py-2 text-app-muted-foreground">{point.iterations}</td>
                      <td className="px-3 py-2 text-app-muted-foreground">
                        {t(COMPLETENESS_LABELS[point.completeness])}
                      </td>
                      <td className="px-3 py-2 text-app-muted-foreground">
                        {point.recordedAt
                          ? formatDate(new Date(point.recordedAt), { dateStyle: "medium" })
                          : t(strings.common.unknownDate)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </SectionPanel>
        </AsyncBoundary>
      ) : null}
    </div>
  );
}
