import type { UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatStatsChartDate } from "../../lib/dateTime";

const W = 820;
const H = 200;
const PAD = 10;

function ms(value: string): number | undefined {
  const at = value ? Date.parse(value) : Number.NaN;
  return Number.isNaN(at) ? undefined : at;
}

/** A span a usage mode was on, shaded behind the readings. */
export interface ModeSpan {
  id: string;
  label: string;
  startedAt: string;
  /** Empty while the mode has no end time. */
  endsAt: string;
}

/**
 * Used percent this window from provider readings, the projected run-out band
 * and the burn that would last to the reset. RCL TrendSpark draws evenly spaced
 * values only, so this time-scaled chart stays local.
 */
export function UsageWindowChart({ forecast, compact = false, modes = [] }: { forecast: UsageWindowForecast; compact?: boolean; modes?: readonly ModeSpan[] }) {
  const points = forecast.series.map((point) => ({ at: ms(point.at), used: point.usedPercent })).filter((point): point is { at: number; used: number } => point.at !== undefined);
  const first = points[0];
  const last = points[points.length - 1];
  if (!first || !last || points.length < 2) return null;
  const start = ms(forecast.windowStart) ?? first.at;
  const reset = ms(forecast.resetsAt) ?? last.at;
  const span = Math.max(1, reset - start);
  const x = (at: number) => PAD + ((Math.min(Math.max(at, start), reset) - start) / span) * (W - 2 * PAD);
  const y = (used: number) => H - PAD - (Math.min(100, Math.max(0, used)) / 100) * (H - 2 * PAD);
  const bandStart = ms(forecast.exhaustionBandStart);
  const bandEnd = ms(forecast.exhaustionBandEnd) ?? reset;
  const shaded = modes.flatMap((mode) => {
    const from = ms(mode.startedAt);
    const to = ms(mode.endsAt) ?? reset;
    return from === undefined || to <= start || from >= reset ? [] : [{ mode, from: Math.max(from, start), to: Math.min(to, reset) }];
  });

  return (
    <figure className="m-0 flex flex-col gap-1.5" data-testid="usage-window-chart">
      <svg viewBox={`0 0 ${W} ${H}`} className={compact ? "h-28 w-full" : "h-40 w-full sm:h-52"} preserveAspectRatio="none" role="img" aria-label={`Used percent this window, ${Math.round(last.used)}% at the newest reading, with the projected run-out band and the reset${shaded.length > 0 ? `, and ${shaded.length} usage mode ${shaded.length === 1 ? "period" : "periods"} shaded` : ""}`}>
        {shaded.map(({ mode, from, to }) => <rect key={mode.id} data-testid="window-mode-band" x={x(from)} y={PAD} width={Math.max(2, x(to) - x(from))} height={H - 2 * PAD} className="fill-muted-foreground/20" />)}
        {[0, 50, 100].map((level) => <line key={level} x1={PAD} x2={W - PAD} y1={y(level)} y2={y(level)} className="stroke-border" strokeWidth={1} vectorEffect="non-scaling-stroke" />)}
        {bandStart !== undefined ? <polygon points={`${x(last.at)},${y(last.used)} ${x(bandStart)},${y(100)} ${x(bandEnd)},${y(100)}`} className="fill-destructive/20" /> : null}
        <line x1={x(last.at)} y1={y(last.used)} x2={x(reset)} y2={y(100)} className="stroke-muted-foreground" strokeWidth={1.5} strokeDasharray="5 5" vectorEffect="non-scaling-stroke" />
        <polyline fill="none" className="stroke-primary" strokeWidth={2.5} vectorEffect="non-scaling-stroke" points={points.map((point) => `${x(point.at)},${y(point.used)}`).join(" ")} />
        <line x1={x(reset)} x2={x(reset)} y1={y(100)} y2={y(0)} className="stroke-foreground" strokeWidth={1.5} vectorEffect="non-scaling-stroke" />
      </svg>
      <div className="flex justify-between text-xs text-muted-foreground" aria-hidden="true">
        <span>{formatStatsChartDate(new Date(start))}</span>
        <span>reset {formatStatsChartDate(new Date(reset))}</span>
      </div>
      {compact ? null : <figcaption className="text-xs text-muted-foreground">Solid: provider readings. Shaded: projected run-out band. Dashed: the burn rate that would last to the reset.{shaded.length > 0 ? ` Grey bands: ${shaded.map(({ mode }) => mode.label).join(", ")} was on.` : ""}</figcaption>}
    </figure>
  );
}
