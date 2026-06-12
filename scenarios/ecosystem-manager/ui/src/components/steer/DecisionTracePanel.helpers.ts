import type { DecisionTraceEntry } from '@/types/api';

/** sumCounts totals a per-dimension count map. */
export function sumCounts(counts?: Record<string, number>): number {
  if (!counts) return 0;
  return Object.values(counts).reduce((a, b) => a + b, 0);
}

/** formatRealizedDelta renders an iteration's realized weighted-score change. */
export function formatRealizedDelta(entry: DecisionTraceEntry): string {
  const delta = entry.realized_delta;
  if (delta === undefined) return 'pending';
  if (delta > 0) return `−${delta.toFixed(1)} (improved)`;
  if (delta < 0) return `+${Math.abs(delta).toFixed(1)} (regressed)`;
  return '0.0 (no change)';
}

/** formatPredictedReduction renders an iteration's forward reduction estimate. */
export function formatPredictedReduction(entry: DecisionTraceEntry): string | null {
  const p = entry.predicted_reduction;
  if (p === undefined || p === 0) return null;
  return p > 0 ? `−${p.toFixed(1)}` : `+${Math.abs(p).toFixed(1)}`;
}

/**
 * calibrationError returns the running mean absolute error between predicted and
 * realized reduction across iterations where both are known, plus the sample
 * count. Null when there is no comparable iteration yet. A large error means the
 * bandit's forward estimate is poorly calibrated to outcomes.
 */
export function calibrationError(entries: DecisionTraceEntry[]): { mae: number; n: number } | null {
  let sum = 0;
  let n = 0;
  for (const e of entries) {
    const p = e.predicted_reduction;
    const r = e.realized_delta;
    if (p === undefined || p === 0) continue;
    if (r === undefined) continue;
    sum += Math.abs(p - r);
    n += 1;
  }
  if (n === 0) return null;
  return { mae: sum / n, n };
}

/** dtvVerdictClass maps a DTV fitness verdict to its badge styling. */
export function dtvVerdictClass(verdict: string): string {
  switch (verdict) {
    case 'green':
      return 'bg-emerald-500/15 text-emerald-600';
    case 'yellow':
      return 'bg-amber-500/15 text-amber-600';
    case 'red':
      return 'bg-destructive/15 text-destructive';
    default:
      return 'bg-muted text-muted-foreground';
  }
}

/** degradedGateLabel renders a human-readable cause for the degraded-gate badge. */
export function degradedGateLabel(cause: string): string {
  switch (cause) {
    case 'dtv_unavailable':
      return 'DTV unavailable';
    case 'all_red':
      return 'all candidate skills DTV-red';
    default:
      return cause;
  }
}

/** topDimensions returns the heaviest open dimensions for an iteration. */
export function topDimensions(entry: DecisionTraceEntry, limit = 3): Array<[string, number]> {
  const scores = entry.dimension_scores ?? {};
  return Object.entries(scores)
    .filter(([, v]) => v > 0)
    .sort((a, b) => (b[1] !== a[1] ? b[1] - a[1] : a[0].localeCompare(b[0])))
    .slice(0, limit);
}
