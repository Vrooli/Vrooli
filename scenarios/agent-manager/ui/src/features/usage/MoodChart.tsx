import { formatDayRange } from "./format";
import type { MoodPoint } from "./api";

const W = 600;
const H = 150;
const PAD = 12;

function ms(value: string | undefined): number | undefined {
  const at = value ? Date.parse(value) : Number.NaN;
  return Number.isNaN(at) ? undefined : at;
}

/** The value axis: 0 to 1 for a probability, widened when a label's own scale reaches outside it. */
export function valueDomain(points: readonly MoodPoint[]): { low: number; high: number } {
  const values = points.flatMap((point) => [point.mean, point.proven_mean]).filter((value): value is number => typeof value === "number");
  return { low: Math.min(0, ...values), high: Math.max(1, ...values) };
}

/** Runs of consecutive buckets that have a value: a bucket with none breaks the line instead of drawing a false zero. */
function runs(points: readonly MoodPoint[], pick: (point: MoodPoint) => number | null | undefined): MoodPoint[][] {
  const out: MoodPoint[][] = [];
  let current: MoodPoint[] = [];
  for (const point of points) {
    if (typeof pick(point) === "number") current.push(point);
    else if (current.length > 0) { out.push(current); current = []; }
  }
  if (current.length > 0) out.push(current);
  return out;
}

/**
 * One message-scope label's mean per time bucket. The solid line is the mean
 * over messages from a proven model; the dashed line is the mean over every
 * message, which is unproven evidence until the two agree. Time-scaled, so it
 * stays local like EvidenceChart: RCL TrendSpark spaces values evenly. Dots mark
 * each bucket, because a person's messages are sparse and a lone bucket would
 * otherwise draw nothing. The table under the chart carries every number.
 */
export function MoodChart({ points, since, until, label }: { points: readonly MoodPoint[]; since?: string; until?: string; label: string }) {
  const times = points.map((point) => ms(point.start)).filter((value): value is number => value !== undefined);
  const start = ms(since) ?? Math.min(...times);
  const end = ms(until) ?? Math.max(...times);
  if (points.length === 0 || times.length === 0 || !(end > start)) return null;
  const { low, high } = valueDomain(points);
  const x = (at: number) => PAD + ((Math.min(Math.max(at, start), end) - start) / (end - start)) * (W - 2 * PAD);
  const y = (value: number) => H - PAD - ((value - low) / (high - low)) * (H - 2 * PAD);
  const at = (point: MoodPoint) => x((ms(point.start) ?? start) + 12 * 3600_000);
  const provenCount = points.filter((point) => typeof point.proven_mean === "number").length;
  const summary = `${label} per day: ${provenCount} day${provenCount === 1 ? "" : "s"} with a proven-model mean (solid) and ${points.filter((point) => typeof point.mean === "number").length} with a mean over all models (dashed, unproven). The table below lists every value.`;
  return (
    <figure className="m-0 flex flex-col gap-1" data-testid="mood-chart">
      <svg viewBox={`0 0 ${W} ${H}`} className="h-auto w-full" role="img" aria-label={summary}>
        {[low, (low + high) / 2, high].map((level) => <line key={level} x1={PAD} x2={W - PAD} y1={y(level)} y2={y(level)} className="stroke-border" strokeWidth={1} vectorEffect="non-scaling-stroke" />)}
        {runs(points, (point) => point.mean).map((run) => (
          <polyline key={run[0]?.start} data-testid="mood-line-all" fill="none" className="stroke-muted-foreground" strokeWidth={2} strokeDasharray="2 4" vectorEffect="non-scaling-stroke" points={run.map((point) => `${at(point)},${y(point.mean ?? 0)}`).join(" ")} />
        ))}
        {points.filter((point) => typeof point.mean === "number").map((point) => <circle key={`all-${point.start}`} cx={at(point)} cy={y(point.mean ?? 0)} r={3} className="fill-background stroke-muted-foreground" strokeWidth={1.5} vectorEffect="non-scaling-stroke" />)}
        {runs(points, (point) => point.proven_mean).map((run) => (
          <polyline key={run[0]?.start} data-testid="mood-line-proven" fill="none" className="stroke-primary" strokeWidth={2.5} vectorEffect="non-scaling-stroke" points={run.map((point) => `${at(point)},${y(point.proven_mean ?? 0)}`).join(" ")} />
        ))}
        {points.filter((point) => typeof point.proven_mean === "number").map((point) => <circle key={`proven-${point.start}`} data-testid="mood-dot-proven" cx={at(point)} cy={y(point.proven_mean ?? 0)} r={3.5} className="fill-primary" />)}
      </svg>
      <div className="flex justify-between text-xs text-muted-foreground" aria-hidden="true">
        <span>{formatDayRange(new Date(start).toISOString())}</span>
        <span>{Number(low.toFixed(1))} to {Number(high.toFixed(1))}</span>
        <span>{formatDayRange(new Date(end).toISOString())}</span>
      </div>
    </figure>
  );
}
