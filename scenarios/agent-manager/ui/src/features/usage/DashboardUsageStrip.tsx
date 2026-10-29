import { Link } from "react-router-dom";
import { Progress } from "@vrooli/react-component-library/Progress/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { useUsageForecast, useUsageRecommendations } from "./api";
import { formatDayRange, isDormantWindow, usageStatus, windowKey, windowTitle } from "./format";
import { DormantWindows, progressTone } from "./UsageWindows";

/**
 * One window per row: chip, name, bar and percent on the first line (fits a
 * 390 px phone); run-out and reset wrap under it on a phone and sit beside it
 * on wider screens.
 */
function StripLine({ forecast, recommendations, siblings }: { forecast: UsageWindowForecast; recommendations: number; siblings: UsageWindowForecast[] }) {
  const status = usageStatus(forecast.status);
  const noReading = forecast.sampleCount === 0n || !forecast.observedAt;
  const atRisk = forecast.status === "will_exhaust" || forecast.status === "at_risk";
  const used = Math.round(forecast.usedPercent);
  const showBar = !noReading && forecast.hasPercent;
  return (
    <Link
      to="/usage"
      data-testid="dashboard-usage-line"
      data-status={forecast.status}
      aria-label={`Open Usage: ${windowTitle(forecast, siblings)}, ${noReading ? "no reading yet" : status.label.toLowerCase()}`}
      className={`flex min-h-11 min-w-0 flex-col gap-1 rounded-lg border px-3 py-2 text-sm no-underline hover:bg-muted/40 sm:flex-row sm:flex-wrap sm:items-center sm:gap-x-4 ${atRisk ? "border-destructive/40 bg-destructive/5" : "border-border bg-card"}`}
    >
      <span className="flex min-w-0 items-center gap-2 sm:flex-[1_1_24rem]">
        <StatusBadge className="shrink-0" tone={noReading ? "neutral" : status.tone}>{noReading ? "No reading" : status.short}</StatusBadge>
        <span className="min-w-0 truncate font-semibold">{windowTitle(forecast, siblings)}</span>
        {showBar ? (
          <>
            <Progress className="min-w-0 flex-1" style={{ minWidth: 0 }} size="sm" showValue={false} value={forecast.usedPercent} tone={progressTone(forecast.status)} label={`${used} percent used`} />
            <span className="shrink-0 font-semibold tabular-nums">{used}%</span>
          </>
        ) : null}
      </span>
      {!showBar ? (
        <span className="text-muted-foreground">{noReading ? "Readings appear after the next reply." : forecast.standing}</span>
      ) : atRisk ? (
        <span>Runs out <strong>{formatDayRange(forecast.exhaustionBandStart || forecast.projectedExhaustionAt, forecast.exhaustionBandEnd)}</strong> · resets {formatDayRange(forecast.resetsAt)}</span>
      ) : (
        <span className="text-muted-foreground">resets {formatDayRange(forecast.resetsAt)}</span>
      )}
      {atRisk && recommendations > 0 ? <span className="font-medium text-primary sm:ml-auto">{recommendations} {recommendations === 1 ? "recommendation" : "recommendations"} →</span> : null}
    </Link>
  );
}

/** One line per provider window above the dashboard; nothing until the forecast answers with windows. */
export function DashboardUsageStrip() {
  const forecast = useUsageForecast();
  const recommendations = useUsageRecommendations();
  const windows = forecast.data?.windows ?? [];
  if (windows.length === 0) return null;
  const dormant = windows.filter((window) => isDormantWindow(window));
  const perProvider = (provider: string) => (recommendations.data?.groups ?? []).filter((group) => group.provider === provider).length;
  return (
    <section aria-label="Subscription usage" data-testid="dashboard-usage-strip" className="flex min-w-0 flex-col gap-2">
      {windows.filter((window) => !dormant.includes(window)).map((window) => <StripLine key={windowKey(window)} forecast={window} recommendations={perProvider(window.provider)} siblings={windows} />)}
      <DormantWindows windows={dormant} siblings={windows} />
    </section>
  );
}
