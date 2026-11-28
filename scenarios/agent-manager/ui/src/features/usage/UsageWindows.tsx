import { Gauge } from "lucide-react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Progress } from "@vrooli/react-component-library/Progress/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { ListUsageModesResponse, UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { MISSING } from "../../lib/format";
import { DataState } from "../../components/feedback/DataState";
import { dormantWindowLine, formatAgo, formatAt, formatRate, formatServerText, isStale, usageStatus, windowKey, windowTitle } from "./format";
import { useUsageModes } from "./api";
import { runnerForProvider } from "./modes";
import { UsageWindowChart, type ModeSpan } from "./UsageWindowChart";

const HORIZONS = ["1h", "6h", "24h"] as const;
const PROGRESS_TONES = { danger: "danger", warning: "warning", success: "success" } as const;

export function progressTone(status: string) {
  const tone = usageStatus(status).tone;
  return tone in PROGRESS_TONES ? PROGRESS_TONES[tone as keyof typeof PROGRESS_TONES] : "accent";
}

/** The projection in one sentence, or why there is none. */
function projectionSentence(forecast: UsageWindowForecast): string {
  if (forecast.status === "unknown" || !forecast.hasPercent) return forecast.statusReason ? formatServerText(forecast.statusReason) : "Not enough provider readings to project this window.";
  if (forecast.status === "on_track" || forecast.status === "idle") return `Lasts until the reset at the current burn (${formatRate(forecast.centralPercentPerHour)}).`;
  const band = forecast.exhaustionBandEnd ? `${formatAt(forecast.exhaustionBandStart)} – ${formatAt(forecast.exhaustionBandEnd)}` : `after ${formatAt(forecast.exhaustionBandStart)}`;
  return `Projected to run out ${band}; the reset is ${formatAt(forecast.resetsAt)}.`;
}

function hasReading(forecast: UsageWindowForecast): boolean {
  return forecast.sampleCount !== 0n && !!forecast.observedAt;
}

function WindowTile({ forecast, siblings, selected, onSelect }: { forecast: UsageWindowForecast; siblings: UsageWindowForecast[]; selected: boolean; onSelect: (key: string) => void }) {
  const status = usageStatus(forecast.status);
  const read = hasReading(forecast);
  const idle = forecast.status === "idle";
  const lastUse = idle && forecast.observedAt ? ` · last use ${formatAgo(forecast.observedAt)}` : "";
  return (
    <button
      type="button"
      aria-pressed={selected}
      data-testid="usage-window"
      data-selected={selected}
      onClick={() => onSelect(windowKey(forecast))}
      className={`flex min-h-11 min-w-0 flex-col gap-2 rounded-lg border p-3 text-left transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${selected ? "border-primary bg-primary/5 ring-1 ring-primary" : "border-border bg-card"}`}
    >
      <span className="flex items-start justify-between gap-2">
        <span className="min-w-0 truncate text-sm font-semibold">{windowTitle(forecast, siblings)}</span>
        <StatusBadge tone={read ? status.tone : "neutral"}>{read ? status.short : "No reading"}</StatusBadge>
      </span>
      {read && forecast.hasPercent ? (
        <>
          <span className="flex flex-wrap items-baseline gap-x-2">
            <span className="text-2xl font-bold tabular-nums">{Math.round(forecast.usedPercent)}%</span>
            <span className="text-xs text-muted-foreground">resets {formatAt(forecast.resetsAt)} · read {formatAgo(forecast.observedAt)}{lastUse}</span>
          </span>
          <Progress size="sm" value={forecast.usedPercent} max={100} tone={progressTone(forecast.status)} showValue={false} label={`${Math.round(forecast.usedPercent)} percent of the ${windowTitle(forecast, siblings)} limit used`} />
        </>
      ) : (
        <span className="text-xs text-muted-foreground">{read ? forecast.standing || MISSING : "Readings appear after the provider reports its quota on the next agent reply."}</span>
      )}
      {idle ? <span className="text-xs text-muted-foreground">Idle, lasts at this pace</span> : null}
    </button>
  );
}

/**
 * Every mode that reaches this provider, ended ones included: the recorded
 * periods (an ended period carries the time it ended) plus the active modes,
 * which win for the same id because their expiry is the live one.
 */
export function modeSpans(modes: ListUsageModesResponse | undefined, runner: string): ModeSpan[] {
  const spans = new Map<string, ModeSpan>();
  for (const period of modes?.periods ?? []) {
    if (period.runner && period.runner !== runner) continue;
    spans.set(period.id, { id: period.id, label: period.label || "A usage mode", startedAt: period.startedAt, endsAt: period.endedAt || period.expiresAt });
  }
  for (const status of modes?.modes ?? []) {
    const mode = status.mode;
    if (!mode || (mode.scope?.runner && mode.scope.runner !== runner)) continue;
    spans.set(mode.id, { id: mode.id, label: mode.label || "A usage mode", startedAt: mode.startedAt, endsAt: mode.expiresAt });
  }
  return [...spans.values()];
}

/** The selected window in full: projection, chart, burn per horizon and the notes that qualify them. */
export function WindowDetail({ forecast, siblings }: { forecast: UsageWindowForecast; siblings: UsageWindowForecast[] }) {
  const modes = useUsageModes();
  if (!hasReading(forecast)) return null;
  const stale = isStale(forecast.observedAt);
  const rates = new Map(forecast.burn.map((rate) => [rate.horizon, rate]));
  const spans = modeSpans(modes.data, runnerForProvider(forecast.provider));
  return (
    <Card data-testid="usage-window-detail" className="flex min-w-0 flex-col gap-3 p-4 sm:p-5">
      <h3 className="text-base font-semibold">{windowTitle(forecast, siblings)} · projection</h3>
      <p className="text-sm">
        <strong>{projectionSentence(forecast)}</strong>
        {forecast.hasPercent && forecast.lastingPercentPerHour > 0 && forecast.status !== "on_track" && forecast.status !== "idle" ? ` To last until the reset, burn at most ${formatRate(forecast.lastingPercentPerHour)}.` : ""}
      </p>
      <UsageWindowChart forecast={forecast} modes={spans} />
      <dl className="grid grid-cols-2 gap-3 border-t border-border pt-3 sm:grid-cols-4">
        {HORIZONS.map((horizon) => {
          const rate = rates.get(horizon);
          return (
            <div key={horizon}>
              <dt className="text-xs text-muted-foreground">Last {horizon}</dt>
              <dd className="text-base font-semibold tabular-nums">{rate?.available ? formatRate(rate.percentPerHour) : MISSING}</dd>
              {rate?.stale ? <dd className="text-xs text-muted-foreground">no reading in {horizon}</dd> : null}
            </div>
          );
        })}
        <div>
          <dt className="text-xs text-muted-foreground">Lasts to reset</dt>
          <dd className="text-base font-semibold tabular-nums">{forecast.lastingPercentPerHour > 0 ? `≤ ${formatRate(forecast.lastingPercentPerHour)}` : MISSING}</dd>
        </div>
      </dl>
      {forecast.lastReadingPercent > 0 && Math.round(forecast.lastReadingPercent) < Math.round(forecast.usedPercent) ? (
        <p className="text-xs text-muted-foreground">The newest reading says {Math.round(forecast.lastReadingPercent)}%; it lags the highest reading this window ({Math.round(forecast.usedPercent)}%), which the forecast uses.</p>
      ) : null}
      {forecast.calibrationWindows >= 3n ? <p className="text-xs text-muted-foreground">Projection calibrated on {forecast.calibrationWindows.toString()} past windows (×{forecast.calibrationFactor.toFixed(2)}).</p> : null}
      {stale ? <p role="note" className="text-xs text-warning">The newest reading is {formatAgo(forecast.observedAt)}, so this projection may be out of date. It updates after the next provider reading.</p> : null}
      {forecast.earlyReset ? <p className="text-xs text-muted-foreground">This window started before the previous one's scheduled reset ({formatAt(forecast.previousResetsAt)}). The forecast restarts whenever the provider resets.</p> : null}
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

interface UsageWindowsProps {
  windows: UsageWindowForecast[];
  loading: boolean;
  error: unknown;
  onRetry: () => void;
  selected: string;
  onSelect: (key: string) => void;
}

/** Subscription windows as one compact row of selectable tiles; dead windows collapse to a line. */
export function UsageWindows({ windows, loading, error, onRetry, selected, onSelect }: UsageWindowsProps) {
  const dormant = windows.filter((window) => window.dormant);
  return (
    <DataState
      subject="Usage forecast"
      loading={loading}
      error={error}
      isEmpty={windows.length === 0}
      onRetry={onRetry}
      empty={<EmptyState icon={<Gauge aria-hidden="true" />} title="No provider readings yet" description="Readings appear once a provider reports its quota after an agent reply. Missing readings are shown as unknown, never as zero." />}
    >
      <div className="space-y-2">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {windows.filter((window) => !dormant.includes(window)).map((window) => <WindowTile key={windowKey(window)} forecast={window} siblings={windows} selected={windowKey(window) === selected} onSelect={onSelect} />)}
        </div>
        <DormantWindows windows={dormant} siblings={windows} />
      </div>
    </DataState>
  );
}
