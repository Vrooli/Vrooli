import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageModePeriod, UsageOutcomeRow, UsageParityRow, UsageRange } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatCount, formatPoints, MISSING } from "../../lib/format";
import { formatShare } from "./format";
import { formatRatio, formatWholePercent, invertRange, known, pathName, ratioRangeText, sessionCoverage } from "./evidence";

const VERDICT = {
  cheaper: { tone: "success", text: "Cheaper" },
  more_expensive: { tone: "danger", text: "Costs more" },
  about_same: { tone: "neutral", text: "About the same" },
  inconclusive: { tone: "neutral", text: "Inconclusive" },
  baseline: { tone: "info", text: "Baseline" },
  not_enough_data: { tone: "neutral", text: "Not enough data" },
} as const;

/** "7.2 to 11 pts" from a range of points; MISSING when the sample cannot support it. */
function pointsRange(range: UsageRange | undefined): string {
  return known(range) ? `${formatPoints(range.low)} to ${formatPoints(range.high)}` : MISSING;
}

function percentRange(range: UsageRange | undefined): string {
  return known(range) ? `${formatWholePercent(range.value)} (${formatWholePercent(range.low)} to ${formatWholePercent(range.high)})` : MISSING;
}

function minutesRange(range: UsageRange | undefined): string {
  return known(range) ? `${Math.round(range.value)} min (${Math.round(range.low)} to ${Math.round(range.high)})` : MISSING;
}

/** Small horizontal interval, scaled to the widest high edge in the table. */
function IntervalBar({ range, top }: { range: UsageRange | undefined; top: number }) {
  if (!known(range) || top <= 0) return <span aria-hidden="true" />;
  const at = (v: number) => `${Math.min(100, Math.max(0, (v / top) * 100))}%`;
  return (
    <span aria-hidden="true" className="relative mt-1 block h-2 w-full rounded bg-muted">
      <i className="absolute top-0 block h-2 rounded bg-primary/50" style={{ left: at(range.low), width: `max(2px, calc(${at(range.high)} - ${at(range.low)}))` }} />
      <i className="absolute top-[-2px] block h-3 w-0.5 bg-foreground" style={{ left: at(range.value) }} />
    </span>
  );
}

const READING_WORDS: Record<string, string> = {
  cheaper: "cost less",
  more_expensive: "cost more",
  about_same: "about the same",
  inconclusive: "inconclusive",
};

/** The comparison under a verdict: a range only reads as "cheaper" when the verdict says so. */
function verdictDetail(row: UsageOutcomeRow) {
  if (known(row.cheaperBy)) {
    if (row.verdict === "cheaper") return `${formatRatio(row.cheaperBy.value)} less than ${row.baseline} (${ratioRangeText(row.cheaperBy)})`;
    const inverse = invertRange(row.cheaperBy);
    if (row.verdict === "more_expensive") return `${formatRatio(inverse?.value)} more than ${row.baseline} (${ratioRangeText(inverse)})`;
    return `${ratioRangeText(inverse)} of ${row.baseline}`;
  }
  const reading = READING_WORDS[row.completedReading];
  if (reading && known(row.cheaperByCompleted)) {
    return `Per completed only, not a verdict: ${reading} against ${row.completedBaseline} (${ratioRangeText(invertRange(row.cheaperByCompleted))} of it)`;
  }
  return null;
}

/** Cost, use, rework and run length for each model on each path, every figure with its range. */
export function ModelComparison({ rows }: { rows: readonly UsageOutcomeRow[] }) {
  const shown = rows.filter((row) => Number(row.units) > 0).sort((a, b) => Number(b.status === "enough") - Number(a.status === "enough") || a.stratum.localeCompare(b.stratum) || Number(b.units - a.units));
  const top = Math.max(0, ...shown.map((row) => (known(row.pointsPerAccepted) ? row.pointsPerAccepted.high : 0)));
  if (shown.length === 0) return <p className="m-0 text-sm text-muted-foreground">No work units in this period.</p>;
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[52rem] text-left text-sm tabular-nums" data-testid="model-comparison">
        <caption className="sr-only">Cost per accepted outcome, the weaker cost per completed unit, rework and run length by model and path, with 90% ranges</caption>
        <thead className="text-xs text-muted-foreground">
          <tr>
            <th scope="col" className="py-1 pr-3 font-medium">Model</th>
            <th scope="col" className="py-1 pr-3 font-medium">Accepted of units</th>
            <th scope="col" className="py-1 pr-3 font-medium">Cost per accepted outcome</th>
            <th scope="col" className="py-1 pr-3 font-medium">Cost per completed (weaker)</th>
            <th scope="col" className="py-1 pr-3 font-medium">Rework</th>
            <th scope="col" className="py-1 pr-3 font-medium">Active time per run</th>
            <th scope="col" className="py-1 pr-3 font-medium">Against the same path's baseline</th>
          </tr>
        </thead>
        <tbody>
          {shown.map((row) => {
            const verdict = VERDICT[row.verdict as keyof typeof VERDICT] ?? VERDICT.not_enough_data;
            return (
              <tr key={`${row.runner}/${row.stratum}/${row.key}`} className="border-t border-border align-top" data-testid="model-comparison-row">
                <th scope="row" className="max-w-[12rem] break-words py-2 pr-3 font-medium">{row.key}<div className="text-xs font-normal text-muted-foreground">{row.runner}, {pathName(row.stratum)}</div></th>
                <td className="py-2 pr-3">{formatCount(row.accepted)} of {formatCount(row.units)}<div className="text-xs text-muted-foreground">{Number(row.completed) > 0 ? `${formatCount(row.completed)} only finished, ` : ""}{formatWholePercent(row.unknownRate)} unknown</div></td>
                <td className="min-w-[9rem] py-2 pr-3">{pointsRange(row.pointsPerAccepted)}<IntervalBar range={row.pointsPerAccepted} top={top} />{known(row.pointsPerAccepted) ? null : <div className="text-xs text-muted-foreground">{row.statusReason ? row.statusReason.replace(/^(\d+) accepted outcomes;/, (_m, n: string) => `${n} accepted ${n === "1" ? "outcome" : "outcomes"};`) : "not enough data"}</div>}</td>
                <td className="min-w-[7rem] py-2 pr-3">{known(row.pointsPerCompleted) ? pointsRange(row.pointsPerCompleted) : MISSING}</td>
                <td className="py-2 pr-3">{percentRange(row.reworkRate)}{row.stratum === "headless" ? <div className="text-xs text-muted-foreground">retries only: no operator to correct it</div> : null}</td>
                <td className="py-2 pr-3">{minutesRange(row.activeMinutesPerRun)}</td>
                <td className="py-2 pr-3"><StatusBadge tone={verdict.tone}>{verdict.text}</StatusBadge><div className="text-xs text-muted-foreground">{verdictDetail(row)}</div></td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function periodLabel(key: string, periods: readonly UsageModePeriod[]): string {
  if (key === "no_mode") return "No mode on";
  if (key === "not_changed") return "Mode on, model not lowered";
  const period = periods.find((p) => p.id === key);
  return period?.label || (period?.runner ? `Mode (${period.runner})` : `Mode ${key}`);
}

/** Quality in plain words: work with a mode on against work without, by mode. */
export function QualityByMode({ rows, periods }: { rows: readonly UsageOutcomeRow[]; periods: readonly UsageModePeriod[] }) {
  const shown = rows.filter((row) => Number(row.units) > 0);
  if (shown.length === 0) return <p className="m-0 text-sm text-muted-foreground">No work units in this period.</p>;
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[26rem] text-left text-sm tabular-nums" data-testid="quality-by-mode">
        <caption className="sr-only">Quality measures with a usage mode on and off</caption>
        <thead className="text-xs text-muted-foreground">
          <tr>
            <th scope="col" className="py-1 pr-3 font-medium">Work</th>
            <th scope="col" className="py-1 pr-3 font-medium">Redone</th>
            <th scope="col" className="py-1 pr-3 font-medium">You stepped in</th>
            <th scope="col" className="py-1 pr-3 font-medium">Unknown</th>
          </tr>
        </thead>
        <tbody>
          {shown.map((row) => (
            <tr key={`${row.runner}/${row.stratum}/${row.key}`} className="border-t border-border align-top" data-testid="quality-row">
              <th scope="row" className="py-2 pr-3 font-medium">{periodLabel(row.key, periods)}<div className="text-xs font-normal text-muted-foreground">{row.runner}, {pathName(row.stratum)}, {formatCount(row.units)} units</div></th>
              <td className="py-2 pr-3">{percentRange(row.reworkRate)}</td>
              <td className="py-2 pr-3">{row.stratum === "headless" ? "No operator on this path" : `${formatCount(row.corrected)} of ${formatCount(row.units)}`}</td>
              <td className="py-2 pr-3">{formatWholePercent(row.unknownRate)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** How much of the work each kind of session covers: usage data and known outcomes. */
export function SessionCoverageList({ rows }: { rows: readonly UsageParityRow[] }) {
  const coverage = sessionCoverage(rows);
  return (
    <dl className="m-0 grid grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-2 text-sm" data-testid="session-coverage">
      {coverage.map((type) => (
        <div key={type.id} className="contents" data-testid="session-coverage-row">
          <dt className="min-w-0">{type.label}<div className="text-xs text-muted-foreground">{formatCount(type.runs)} runs, {formatCount(type.eligible)} counted</div></dt>
          <dd className="m-0 text-right tabular-nums">{type.usageShare === undefined ? MISSING : `${formatShare(type.usageShare)} have usage data`}<div className="text-xs text-muted-foreground">{type.outcomeShare === undefined ? "outcome coverage not known" : `${formatShare(type.outcomeShare)} have an outcome`}</div></dd>
        </div>
      ))}
    </dl>
  );
}

/** Operator signals arrive with another measure; until it is defined the view says so. */
export function OperatorSignals({ defined }: { defined: boolean | undefined }) {
  return (
    <p className="m-0 text-sm text-muted-foreground" data-testid="operator-signals">
      {defined ? "The operator-signals measure exists, but this view does not read it yet." : "Operator signals (terse replies, repeated corrections) are not available yet."}
    </p>
  );
}
