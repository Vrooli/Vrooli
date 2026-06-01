/**
 * Decision Trace panel — the controller's glass box.
 *
 * Renders the closed-loop controller's per-iteration reasoning for a
 * scenario-improver task: open findings by dimension (state) → chosen skill
 * (decision) → rationale → realized weighted-score delta after the run.
 * See docs/concepts/CONTROL-MODEL.md ("Transparency").
 */
import { useAutoSteerDecisionTrace, useAutoSteerEffectiveness } from '@/hooks/useAutoSteer';
import type { DecisionTraceEntry, EffectivenessRow } from '@/types/api';

/** sumCounts totals a per-dimension count map. */
export function sumCounts(counts?: Record<string, number>): number {
  if (!counts) return 0;
  return Object.values(counts).reduce((a, b) => a + b, 0);
}

/** formatRealizedDelta renders an iteration's realized weighted-score change. */
export function formatRealizedDelta(entry: DecisionTraceEntry): string {
  const delta = entry.realized_delta;
  if (delta === undefined || delta === null) return 'pending';
  if (delta > 0) return `−${delta.toFixed(1)} (improved)`;
  if (delta < 0) return `+${Math.abs(delta).toFixed(1)} (regressed)`;
  return '0.0 (no change)';
}

/** formatPredictedReduction renders an iteration's forward reduction estimate. */
export function formatPredictedReduction(entry: DecisionTraceEntry): string | null {
  const p = entry.predicted_reduction;
  if (p === undefined || p === null || p === 0) return null;
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
    if (p === undefined || p === null || p === 0) continue;
    if (r === undefined || r === null) continue;
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

  const calibration = calibrationError(entries);

  return (
    <ol className="space-y-3" role="list" aria-label="Controller decision trace">
      {calibration && (
        <li role="presentation" className="list-none">
          <p
            className="rounded-md border border-border/60 bg-muted/40 px-2 py-1 text-[11px] text-muted-foreground"
            data-testid="calibration-indicator"
            aria-label={`Bandit calibration: mean absolute prediction error ${calibration.mae.toFixed(1)} over ${calibration.n} iterations`}
          >
            Bandit calibration: mean |predicted − realized| = {calibration.mae.toFixed(1)} over {calibration.n}{' '}
            iteration{calibration.n === 1 ? '' : 's'}
          </p>
        </li>
      )}
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
                aria-label={`Realized delta ${formatRealizedDelta(entry)}${
                  formatPredictedReduction(entry) ? `, predicted ${formatPredictedReduction(entry)}` : ''
                }`}
              >
                {formatPredictedReduction(entry) && (
                  <span data-testid="trace-predicted" className="mr-2">
                    pred Δ {formatPredictedReduction(entry)}
                  </span>
                )}
                <span data-testid="trace-realized">Δ {formatRealizedDelta(entry)}</span>
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

            {(() => {
              const closed = sumCounts(entry.closed_by_dimension);
              const introduced = sumCounts(entry.introduced_by_dimension);
              const showFlow = closed > 0 || introduced > 0 || (entry.tokens_used ?? 0) > 0;
              if (!showFlow) return null;
              return (
                <p className="mt-1 text-[11px] text-muted-foreground" data-testid="trace-flow">
                  {closed > 0 && <span>closed {closed}</span>}
                  {introduced > 0 && <span>{closed > 0 ? ', ' : ''}introduced {introduced}</span>}
                  {(entry.tokens_used ?? 0) > 0 && (
                    <span>{closed > 0 || introduced > 0 ? ' · ' : ''}{entry.tokens_used} tok</span>
                  )}
                </p>
              );
            })()}

            {(entry.regressed || entry.veto_applied || entry.halt_reason) && (
              <ul className="mt-2 flex flex-wrap gap-1.5" aria-label="Controller flags">
                {entry.regressed && (
                  <li className="rounded bg-destructive/15 px-1.5 py-0.5 text-[11px] text-destructive" data-testid="flag-regressed">
                    regressed
                  </li>
                )}
                {entry.veto_applied && (
                  <li className="rounded bg-destructive/15 px-1.5 py-0.5 text-[11px] text-destructive" data-testid="flag-veto">
                    regression veto
                  </li>
                )}
                {entry.halt_reason && (
                  <li className="rounded bg-amber-500/15 px-1.5 py-0.5 text-[11px] text-amber-600" data-testid="flag-halt">
                    halt: {entry.halt_reason}
                  </li>
                )}
              </ul>
            )}

            {entry.gate_degraded_cause && (
              <p
                className="mt-2 rounded border border-destructive/40 bg-destructive/15 px-2 py-1 text-[11px] font-medium text-destructive"
                role="alert"
                data-testid="gate-degraded"
              >
                ⚠ Degraded gate ({degradedGateLabel(entry.gate_degraded_cause)}) — proceeded with the
                least-bad skill; remaining iteration budget halved. Review this iteration.
              </p>
            )}

            {(entry.dtv_verdict || (entry.dtv_excluded && Object.keys(entry.dtv_excluded).length > 0) || entry.dtv_degraded) && (
              <ul className="mt-2 flex flex-wrap gap-1.5" aria-label="DTV fitness">
                {entry.dtv_verdict && (
                  <li
                    className={`rounded px-1.5 py-0.5 text-[11px] ${dtvVerdictClass(entry.dtv_verdict)}`}
                    data-testid="dtv-verdict"
                  >
                    DTV: {entry.dtv_verdict}
                    {(entry.dtv_prior ?? 0) > 0 && ` · prior ${entry.dtv_prior?.toFixed(2)}`}
                  </li>
                )}
                {entry.dtv_excluded &&
                  Object.entries(entry.dtv_excluded).map(([skill, reason]) => (
                    <li
                      key={skill}
                      className="rounded bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground"
                      data-testid="dtv-excluded"
                    >
                      gated {skill} ({reason})
                    </li>
                  ))}
                {entry.dtv_gate_override && (
                  <li className="rounded bg-amber-500/15 px-1.5 py-0.5 text-[11px] text-amber-600" data-testid="dtv-override">
                    all-red override
                  </li>
                )}
                {entry.dtv_degraded && (
                  <li className="rounded bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground" data-testid="dtv-degraded">
                    DTV degraded → P1
                  </li>
                )}
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

interface EffectivenessTableProps {
  rows: EffectivenessRow[];
}

/** Presentational, network-free effectiveness ledger — "which skills work". */
export function EffectivenessTable({ rows }: EffectivenessTableProps) {
  if (rows.length === 0) {
    return (
      <p className="text-sm text-muted-foreground" data-testid="effectiveness-empty">
        No effectiveness data yet — the ledger fills as steered runs complete iterations.
      </p>
    );
  }
  return (
    <table className="w-full text-left text-xs" aria-label="Skill effectiveness ledger">
      <thead className="text-muted-foreground">
        <tr>
          <th className="py-1 pr-2 font-medium">Skill</th>
          <th className="py-1 pr-2 font-medium">Dimension</th>
          <th className="py-1 pr-2 font-medium" title="closed − introduced">Net</th>
          <th className="py-1 pr-2 font-medium">Runs</th>
          <th className="py-1 pr-2 font-medium" title="net findings per 1000 tokens">Efficacy</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr key={`${r.skill_id}:${r.dimension}`} className="border-t border-border/40">
            <td className="py-1 pr-2 font-medium text-primary">{r.skill_id}</td>
            <td className="py-1 pr-2">{r.dimension}</td>
            <td className={`py-1 pr-2 ${r.net_closed < 0 ? 'text-destructive' : ''}`}>
              {r.net_closed > 0 ? `+${r.net_closed}` : r.net_closed}
            </td>
            <td className="py-1 pr-2">{r.total_runs}</td>
            <td className="py-1 pr-2">{r.expected_efficacy_per_ktok.toFixed(2)}/khtok</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

interface EffectivenessPanelProps {
  skill?: string;
  dimension?: string;
}

/** Data-fetching wrapper around EffectivenessTable. */
export function EffectivenessPanel({ skill, dimension }: EffectivenessPanelProps) {
  const { data: rows = [], isLoading, isError } = useAutoSteerEffectiveness({ skill, dimension });
  return (
    <section aria-label="Skill effectiveness" className="space-y-2">
      <h4 className="text-sm font-semibold">Skill effectiveness</h4>
      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading effectiveness ledger…</p>
      ) : isError ? (
        <p className="text-sm text-destructive">Failed to load the effectiveness ledger.</p>
      ) : (
        <EffectivenessTable rows={rows} />
      )}
    </section>
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
