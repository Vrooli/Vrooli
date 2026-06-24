import type { DecisionTraceEntry } from '@/types/api';

/** formatRealizedDelta renders an iteration's realized weighted-score change. */
export function formatRealizedDelta(entry: DecisionTraceEntry): string {
  const delta = entry.realized_delta;
  if (delta === undefined) return 'pending';
  if (delta > 0) return `−${delta.toFixed(1)} (improved)`;
  if (delta < 0) return `+${Math.abs(delta).toFixed(1)} (regressed)`;
  return '0.0 (no change)';
}

/** topDimensions returns the heaviest open dimensions for an iteration. */
export function topDimensions(entry: DecisionTraceEntry, limit = 3): Array<[string, number]> {
  const scores = entry.dimension_scores ?? {};
  return Object.entries(scores)
    .filter(([, v]) => v > 0)
    .sort((a, b) => (b[1] !== a[1] ? b[1] - a[1] : a[0].localeCompare(b[0])))
    .slice(0, limit);
}
