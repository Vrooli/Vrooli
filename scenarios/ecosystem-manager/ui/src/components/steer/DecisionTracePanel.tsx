/**
 * Decision Trace panel — the controller's glass box.
 *
 * Renders the closed-loop controller's per-iteration reasoning for a
 * scenario-improver task: open findings by dimension (state) → chosen skill
 * (decision) → rationale → realized weighted-score delta after the run.
 * See docs/concepts/CONTROL-MODEL.md ("Transparency").
 */
import { useAutoSteerDecisionTrace } from '@/hooks/useAutoSteer';
import type { DecisionTraceEntry } from '@/types/api';

/** formatRealizedDelta renders an iteration's realized weighted-score change. */
export function formatRealizedDelta(entry: DecisionTraceEntry): string {
  const delta = entry.realized_delta;
  if (delta === undefined || delta === null) return 'pending';
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

interface DecisionTraceListProps {
  entries: DecisionTraceEntry[];
}

/** Presentational, network-free list of decision-trace entries. */
export function DecisionTraceList({ entries }: DecisionTraceListProps) {
  if (entries.length === 0) {
    return (
      <p className="text-sm text-muted-foreground" data-testid="decision-trace-empty">
        No controller decisions recorded yet.
      </p>
    );
  }

  return (
    <ol className="space-y-3" role="list" aria-label="Controller decision trace">
      {entries.map((entry) => {
        const dims = topDimensions(entry);
        return (
          <li
            key={entry.iteration}
            role="listitem"
            aria-label={`Iteration ${entry.iteration}: selected ${entry.chosen_skill || 'no skill'}`}
            className="rounded-md border border-border/60 bg-card/40 p-3 backdrop-blur-sm"
          >
            <div className="flex items-center justify-between gap-2">
              <span className="font-medium">
                Iteration {entry.iteration}
                {entry.chosen_skill ? (
                  <>
                    {' → '}
                    <span className="text-primary">{entry.chosen_skill}</span>
                  </>
                ) : (
                  <span className="text-muted-foreground"> → (no actionable skill)</span>
                )}
              </span>
              <span
                className="text-xs text-muted-foreground"
                aria-label={`Realized delta ${formatRealizedDelta(entry)}`}
              >
                Δ {formatRealizedDelta(entry)}
              </span>
            </div>

            {entry.heaviest_dimension && (
              <p className="mt-1 text-xs text-muted-foreground">
                Heaviest open dimension: <span className="font-medium">{entry.heaviest_dimension}</span>
                {entry.score_before !== undefined && (
                  <>
                    {' · score '}
                    {entry.score_before.toFixed(1)}
                    {entry.score_after !== undefined && ` → ${entry.score_after.toFixed(1)}`}
                  </>
                )}
              </p>
            )}

            {dims.length > 0 && (
              <ul className="mt-2 flex flex-wrap gap-1.5" aria-label="Open findings by dimension">
                {dims.map(([dim, score]) => (
                  <li
                    key={dim}
                    className="rounded bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground"
                  >
                    {dim}: {score.toFixed(1)}
                  </li>
                ))}
              </ul>
            )}

            {entry.rationale && (
              <p className="mt-2 text-xs italic text-muted-foreground">{entry.rationale}</p>
            )}
          </li>
        );
      })}
    </ol>
  );
}

interface DecisionTracePanelProps {
  taskId: string;
}

/** Data-fetching wrapper around DecisionTraceList. */
export function DecisionTracePanel({ taskId }: DecisionTracePanelProps) {
  const { data: entries = [], isLoading, isError } = useAutoSteerDecisionTrace(taskId);

  return (
    <section aria-label="Decision trace" className="space-y-2">
      <h4 className="text-sm font-semibold">Decision trace</h4>
      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading decision trace…</p>
      ) : isError ? (
        <p className="text-sm text-destructive">Failed to load the decision trace.</p>
      ) : (
        <DecisionTraceList entries={entries} />
      )}
    </section>
  );
}
