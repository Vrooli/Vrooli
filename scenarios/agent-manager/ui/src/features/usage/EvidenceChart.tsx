import type { UsageModePeriod, UsageModelSeries } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatPoints } from "../../lib/format";
import { formatDayRange } from "./format";
import { periodName } from "./evidence";

const W = 820;
const H = 180;
const PAD = 10;
/** More series than this stop being readable as lines on one chart; the rest stay in the table. */
export const MAX_SERIES = 3;
/** Small charts drawn when there are more models than MAX_SERIES. */
export const MAX_MULTIPLES = 6;

// Distinguished by dash as well as colour, so the lines stay apart without colour vision.
const STYLES = [
  { stroke: "stroke-primary", swatch: "bg-primary", dash: undefined },
  { stroke: "stroke-success", swatch: "bg-success", dash: "7 4" },
  { stroke: "stroke-warning", swatch: "bg-warning", dash: "2 4" },
] as const;

function ms(value: string): number | undefined {
  const at = value ? Date.parse(value) : Number.NaN;
  return Number.isNaN(at) ? undefined : at;
}

function total(series: UsageModelSeries): number {
  return series.buckets.reduce((sum, bucket) => sum + bucket.points, 0);
}

/** The series to draw, largest first, and how many were left to the table. */
export function drawnSeries(series: readonly UsageModelSeries[]): { drawn: UsageModelSeries[]; hidden: number; multiples: boolean } {
  const sorted = [...series].sort((a, b) => total(b) - total(a));
  // Past three models one chart is unreadable: each of the largest gets its own small chart on a shared scale.
  const multiples = sorted.length > MAX_SERIES;
  const limit = multiples ? MAX_MULTIPLES : MAX_SERIES;
  return { drawn: sorted.slice(0, limit), hidden: Math.max(0, sorted.length - limit), multiples };
}

interface Band { period: UsageModePeriod; from: number; to: number }

function Panel({ lines, bands, peak, start, end, label, tall }: { lines: readonly { s: UsageModelSeries; index: number }[]; bands: readonly Band[]; peak: number; start: number; end: number; label: string; tall: boolean }) {
  const span = end - start;
  const x = (at: number) => PAD + ((Math.min(Math.max(at, start), end) - start) / span) * (W - 2 * PAD);
  const y = (points: number) => H - PAD - (points / peak) * (H - 2 * PAD);
  return (
    <svg viewBox={`0 0 ${W} ${H}`} className={tall ? "h-40 w-full sm:h-48" : "h-24 w-full"} preserveAspectRatio="none" role="img" aria-label={label}>
      {bands.map((band) => <rect key={band.period.id} data-testid="evidence-mode-band" x={x(band.from)} y={PAD} width={Math.max(2, x(band.to) - x(band.from))} height={H - 2 * PAD} className="fill-muted-foreground/20" />)}
      {[0, 0.5, 1].map((level) => <line key={level} x1={PAD} x2={W - PAD} y1={y(peak * level)} y2={y(peak * level)} className="stroke-border" strokeWidth={1} vectorEffect="non-scaling-stroke" />)}
      {lines.map(({ s, index }) => {
        const style = STYLES[index % STYLES.length] ?? STYLES[0];
        const points = s.buckets.map((b) => `${x((ms(b.at) ?? start) + 12 * 3600_000)},${y(b.points)}`).join(" ");
        return <polyline key={s.model} data-testid="evidence-series" fill="none" className={style.stroke} strokeWidth={2.5} strokeDasharray={style.dash} vectorEffect="non-scaling-stroke" points={points} />;
      })}
    </svg>
  );
}

/**
 * Attributed quota points per day for each model, with the periods a usage
 * mode was on shaded behind them. Up to three models share one chart; more
 * become small charts on one scale. Time-scaled, so it stays local: RCL
 * TrendSpark spaces values evenly. The table under the chart carries every
 * number, for every model.
 */
export function EvidenceChart({ series, periods, windowStart, windowEnd }: { series: readonly UsageModelSeries[]; periods: readonly UsageModePeriod[]; windowStart: string; windowEnd: string }) {
  const start = ms(windowStart);
  const end = ms(windowEnd);
  const { drawn, hidden, multiples } = drawnSeries(series);
  const peak = Math.max(0, ...drawn.flatMap((s) => s.buckets.map((b) => b.points)));
  if (start === undefined || end === undefined || end <= start || drawn.length === 0 || peak <= 0) {
    return <p className="text-sm text-muted-foreground" data-testid="evidence-chart-empty">No attributed usage in this period, so there is nothing to draw.</p>;
  }
  const bands: Band[] = periods.flatMap((period) => {
    const from = ms(period.startedAt);
    const to = ms(period.endedAt) ?? ms(period.expiresAt) ?? end;
    return from === undefined ? [] : [{ period, from: Math.max(from, start), to: Math.min(to, end) }];
  }).filter((band) => band.to > band.from);
  const everyone = [...series].sort((a, b) => total(b) - total(a));
  const days = [...new Set(everyone.flatMap((s) => s.buckets.map((b) => b.at)))].sort();
  const lines = drawn.map((s, index) => ({ s, index }));
  const summary = `Quota points used per day by ${drawn.map((s) => s.model).join(", ")}, with ${bands.length} usage mode ${bands.length === 1 ? "period" : "periods"} shaded. The table below lists every value.`;
  return (
    <figure className="m-0 flex flex-col gap-2" data-testid="evidence-chart">
      {multiples ? (
        <div className="grid gap-3 sm:grid-cols-2" data-testid="evidence-multiples">
          {lines.map((line) => (
            <div key={line.s.model} className="min-w-0">
              <div className="truncate text-xs font-medium">{line.s.model} <span className="font-normal text-muted-foreground">{formatPoints(total(line.s))} in all</span></div>
              <Panel lines={[{ s: line.s, index: 0 }]} bands={bands} peak={peak} start={start} end={end} label={`${line.s.model}: ${summary}`} tall={false} />
            </div>
          ))}
        </div>
      ) : <Panel lines={lines} bands={bands} peak={peak} start={start} end={end} label={summary} tall />}
      <div className="flex justify-between text-xs text-muted-foreground" aria-hidden="true">
        <span>{formatDayRange(windowStart)}</span>
        <span>{multiples ? "same scale in every chart, " : ""}peak {formatPoints(peak)} a day</span>
        <span>{formatDayRange(windowEnd)}</span>
      </div>
      <ul className="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 p-0 text-xs" aria-label="Chart key">
        {multiples ? null : drawn.map((s, index) => <li key={s.model} className="inline-flex items-center gap-1.5"><i className={`inline-block h-1 w-5 rounded ${(STYLES[index] ?? STYLES[0]).swatch}`} aria-hidden="true" />{s.model}</li>)}
        {bands.length > 0 ? <li className="inline-flex items-center gap-1.5"><i className="inline-block h-3 w-5 rounded bg-muted-foreground/20" aria-hidden="true" />Mode on: {bands.map((b) => `${periodName(b.period)} (${formatDayRange(b.period.startedAt, b.period.endedAt || b.period.expiresAt)})`).join("; ")}</li> : <li className="text-muted-foreground">No mode was on in this period.</li>}
      </ul>
      {hidden > 0 ? <p className="m-0 text-xs text-muted-foreground" data-testid="evidence-chart-hidden">{hidden} smaller {hidden === 1 ? "model is" : "models are"} not drawn; {hidden === 1 ? "it is" : "they are"} in the table below.</p> : null}
      <details className="text-xs">
        <summary className="inline-flex min-h-11 cursor-pointer items-center text-primary sm:min-h-8">Show the numbers as a table</summary>
        <div className="overflow-x-auto">
          <table className="mt-1 w-full text-left tabular-nums" data-testid="evidence-chart-table">
            <caption className="sr-only">Quota points used per day by model</caption>
            <thead><tr><th scope="col" className="py-1 pr-3 font-medium">Day</th>{everyone.map((s) => <th key={s.model} scope="col" className="py-1 pr-3 font-medium">{s.model}</th>)}</tr></thead>
            <tbody>
              {days.map((day) => (
                <tr key={day} className="border-t border-border">
                  <th scope="row" className="py-1 pr-3 font-normal">{formatDayRange(day)}</th>
                  {everyone.map((s) => {
                    const bucket = s.buckets.find((b) => b.at === day);
                    return <td key={s.model} className="py-1 pr-3">{bucket ? formatPoints(bucket.points) : "0 pts"}</td>;
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </figure>
  );
}
