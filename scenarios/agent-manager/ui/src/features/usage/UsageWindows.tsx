import { Gauge } from "lucide-react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Progress } from "@vrooli/react-component-library/Progress/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { DataState } from "../../components/feedback/DataState";
import { useUsageForecast } from "./api";
import { dormantWindowLine, formatAgo, formatAt, formatRate, formatServerText, isDormantWindow, isStale, UNKNOWN, usageStatus, windowKey, windowTitle } from "./format";
import { UsageWindowChart } from "./UsageWindowChart";

const HORIZONS = ["1h", "6h", "24h"] as const;
const PROGRESS_TONES = { danger: "danger", warning: "warning", success: "success" } as const;

export function progressTone(status: string) {
  const tone = usageStatus(status).tone;
  return tone in PROGRESS_TONES ? PROGRESS_TONES[tone as keyof typeof PROGRESS_TONES] : "accent";
}

/** The projection in one sentence, or why there is none. */
function projectionSentence(forecast: UsageWindowForecast): string {
  if (forecast.status === "unknown" || !forecast.hasPercent) return forecast.statusReason ? formatServerText(forecast.statusReason) : "Not enough provider readings to project this window.";
  if (forecast.status === "on_track") return `Lasts until the reset at the current burn (${formatRate(forecast.centralPercentPerHour)}).`;
  const band = forecast.exhaustionBandEnd ? `${formatAt(forecast.exhaustionBandStart)} – ${formatAt(forecast.exhaustionBandEnd)}` : `after ${formatAt(forecast.exhaustionBandStart)}`;
  return `Projected to run out ${band}; the reset is ${formatAt(forecast.resetsAt)}.`;
}

function WindowCard({ forecast, siblings }: { forecast: UsageWindowForecast; siblings: UsageWindowForecast[] }) {
  const status = usageStatus(forecast.status);
  const noReading = forecast.sampleCount === 0n || !forecast.observedAt;
  const stale = !noReading && isStale(forecast.observedAt);
  const rates = new Map(forecast.burn.map((rate) => [rate.horizon, rate]));
  return (
    <Card data-testid="usage-window" className="flex min-w-0 flex-col gap-3 p-4 sm:p-5">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="text-base font-semibold">{windowTitle(forecast, siblings)}</h3>
          <p className="text-sm text-muted-foreground">
            {noReading ? "No provider reading yet" : `Provider reading ${formatAgo(forecast.observedAt)}`}
            {forecast.resetsAt ? ` · resets ${formatAt(forecast.resetsAt)}` : ""}
          </p>
        </div>
        <StatusBadge tone={noReading ? "neutral" : status.tone}>{noReading ? "No reading yet" : status.label}</StatusBadge>
      </div>
      {noReading ? (
        <p className="text-sm text-muted-foreground">{forecast.statusReason ? formatServerText(forecast.statusReason) : "Readings appear after the provider reports its quota on the next agent reply."}</p>
      ) : (
        <>
          {forecast.hasPercent ? (
            <>
              <div className="flex items-baseline gap-2">
                <span className="text-4xl font-bold tabular-nums">{Math.round(forecast.usedPercent)}%</span>
                <span className="text-sm text-muted-foreground">used · {Math.max(0, 100 - Math.round(forecast.usedPercent))}% left</span>
              </div>
              <Progress value={forecast.usedPercent} max={100} tone={progressTone(forecast.status)} showValue={false} label={`${Math.round(forecast.usedPercent)} percent of the ${forecast.window} limit used`} />
            </>
          ) : (
            <p className="text-sm font-medium">{forecast.standing || UNKNOWN}</p>
          )}
          <p className="text-sm">
            <strong>{projectionSentence(forecast)}</strong>
            {forecast.hasPercent && forecast.lastingPercentPerHour > 0 && forecast.status !== "on_track" ? ` To last until the reset, burn at most ${formatRate(forecast.lastingPercentPerHour)}.` : ""}
          </p>
          <UsageWindowChart forecast={forecast} />
          <dl className="grid grid-cols-2 gap-3 border-t border-border pt-3 sm:grid-cols-4">
            {HORIZONS.map((horizon) => {
              const rate = rates.get(horizon);
              return (
                <div key={horizon}>
                  <dt className="text-xs text-muted-foreground">Last {horizon}</dt>
                  <dd className="text-base font-semibold tabular-nums">{rate?.available ? formatRate(rate.percentPerHour) : UNKNOWN}</dd>
                  {rate?.stale ? <dd className="text-xs text-muted-foreground">no reading in {horizon}</dd> : null}
                </div>
              );
            })}
            <div>
              <dt className="text-xs text-muted-foreground">Lasts to reset</dt>
              <dd className="text-base font-semibold tabular-nums">{forecast.lastingPercentPerHour > 0 ? `≤ ${formatRate(forecast.lastingPercentPerHour)}` : UNKNOWN}</dd>
            </div>
          </dl>
          {forecast.lastReadingPercent > 0 && Math.round(forecast.lastReadingPercent) < Math.round(forecast.usedPercent) ? (
            <p className="text-xs text-muted-foreground">The newest reading says {Math.round(forecast.lastReadingPercent)}%; it lags the highest reading this window ({Math.round(forecast.usedPercent)}%), which the forecast uses.</p>
          ) : null}
          {forecast.calibrationWindows >= 3n ? <p className="text-xs text-muted-foreground">Projection calibrated on {forecast.calibrationWindows.toString()} past windows (×{forecast.calibrationFactor.toFixed(2)}).</p> : null}
          {stale ? <p role="note" className="text-xs text-warning">The newest reading is {formatAgo(forecast.observedAt)}, so this projection may be out of date. It updates after the next provider reading.</p> : null}
          {forecast.earlyReset ? <p className="text-xs text-muted-foreground">This window started before the previous one's scheduled reset ({formatAt(forecast.previousResetsAt)}). The forecast restarts whenever the provider resets.</p> : null}
        </>
      )}
    </Card>
  );
}

/** Windows with no reading in their current period: one muted line each. */
export function DormantWindows({ windows, siblings = windows }: { windows: UsageWindowForecast[]; siblings?: UsageWindowForecast[] }) {
  if (windows.length === 0) return null;
  return (
    <ul className="m-0 list-none space-y-1 p-0 text-sm text-muted-foreground" aria-label="Windows without a current reading">
      {windows.map((window) => <li key={windowKey(window)} data-testid="usage-window-dormant">{dormantWindowLine(window, siblings)}</li>)}
    </ul>
  );
}

/** Subscription windows: one card per provider window from UsageForecast; dead windows collapse to a line. */
export function UsageWindows() {
  const forecast = useUsageForecast();
  const windows = forecast.data?.windows ?? [];
  const dormant = windows.filter((window) => isDormantWindow(window));
  return (
    <DataState
      subject="Usage forecast"
      loading={forecast.isLoading}
      error={forecast.error}
      isEmpty={windows.length === 0}
      onRetry={() => void forecast.refetch()}
      empty={<EmptyState icon={<Gauge aria-hidden="true" />} title="No provider readings yet" description="Readings appear once a provider reports its quota after an agent reply. Missing readings are shown as unknown, never as zero." />}
    >
      <div className="space-y-3">
        <div className="grid gap-4 lg:grid-cols-2">
          {windows.filter((window) => !dormant.includes(window)).map((window) => <WindowCard key={windowKey(window)} forecast={window} siblings={windows} />)}
        </div>
        <DormantWindows windows={dormant} siblings={windows} />
      </div>
    </DataState>
  );
}
