// Pure logic for the Usage evidence view: which state the data supports and
// the plain-language sentences generated from it. Every figure comes from
// `usage outcomes`; a value the API did not give is said to be unknown, never
// filled in.
import type { UsageModePeriod, UsageOutcomeRow, UsageOutcomesResponse, UsageParityRow, UsageRange } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { MISSING } from "../../lib/format";

/** How a compared set of work was assigned to the models, as recorded per unit. */
export type EvidenceDesign = "observational" | "randomized" | "day_block_randomized";

export type EvidenceState = "no_mode" | "not_enough_data" | EvidenceDesign;

export const EVIDENCE_LABELS: Record<EvidenceState, string> = {
  no_mode: "No mode run yet",
  not_enough_data: "Not enough data",
  observational: "Observational",
  randomized: "Randomized",
  day_block_randomized: "Randomized by day",
};

/** The smallest accepted-outcome count the API needs before it gives a row a verdict. */
export const ENOUGH_ACCEPTED = 5;

/** "3.6x": one decimal under 10, whole above; MISSING when the value is not a finite number. */
export function formatRatio(value: number | undefined): string {
  if (value === undefined || !Number.isFinite(value)) return MISSING;
  if (value > 0 && value < 0.1) return "<0.1x";
  return `${value >= 10 ? Math.round(value) : value.toFixed(1)}x`;
}

/** "28%" from a 0 to 1 share. */
export function formatWholePercent(share: number | undefined): string {
  return share === undefined || !Number.isFinite(share) ? MISSING : `${Math.round(share * 100)}%`;
}

export function known(range: UsageRange | undefined): range is UsageRange {
  return !!range && range.available;
}

/** "1.1x to 1.9x", or MISSING when the range is unknown. */
export function ratioRangeText(range: UsageRange | undefined): string {
  return known(range) ? `${formatRatio(range.low)} to ${formatRatio(range.high)}` : MISSING;
}

/** The ratio inverted: a "cheaper by" range read from the other side. Needs a positive low edge. */
export function invertRange(range: UsageRange | undefined): UsageRange | undefined {
  if (!known(range) || range.low <= 0 || range.value <= 0 || range.high <= 0) return undefined;
  return { ...range, value: 1 / range.value, low: 1 / range.high, high: 1 / range.low };
}

/** A model or role key as it reads in a sentence. */
export function rowName(key: string): string {
  return key === "no_mode" ? "No mode" : key;
}

/** The path a row's work ran on, as it reads in a sentence. */
export function pathName(stratum: string): string {
  if (stratum === "headless") return "headless runs";
  return stratum === "operator" ? "operator sessions" : "all work";
}

/** Rows of one key across paths, summed, for views that list keys rather than compare them. */
export function rowsByKey(rows: readonly UsageOutcomeRow[]): UsageOutcomeRow[] {
  const merged = new Map<string, UsageOutcomeRow>();
  for (const row of rows) {
    const id = `${row.runner}/${row.key}`;
    const seen = merged.get(id);
    merged.set(id, seen ? { ...seen, units: seen.units + row.units, accepted: seen.accepted + row.accepted, completed: seen.completed + row.completed } : row);
  }
  return [...merged.values()];
}

/** Rows compared with a baseline and given a verdict. */
export function comparedRows(rows: readonly UsageOutcomeRow[]): UsageOutcomeRow[] {
  return rows.filter((row) => row.status === "enough" && row.verdict !== "baseline" && row.verdict !== "not_enough_data" && known(row.cheaperBy));
}

/** The comparison that carries the headline: the compared row backed by the most work units. */
export function headlineRow(rows: readonly UsageOutcomeRow[]): UsageOutcomeRow | undefined {
  return [...comparedRows(rows)].sort((a, b) => Number(b.units - a.units))[0];
}

export interface EvidenceView {
  state: EvidenceState;
  /** One or two sentences, generated from the data; the first thing on the page. */
  conclusion: string;
  /** What the strength label means for this result. */
  strength: string;
  headline?: UsageOutcomeRow;
}

function acceptedTotal(rows: readonly UsageOutcomeRow[]): number {
  return rows.reduce((sum, row) => sum + Number(row.accepted), 0);
}

const ACCEPTANCE_SOURCES: Record<string, string> = {
  operator_approval: "your approval",
  validation: "a passing validation",
  agent_verdict: "the agent's own verdict",
};

/** What acceptance rests on for a row, and what stayed out of it. */
export function acceptanceClause(row: UsageOutcomeRow): string {
  const sources = row.evidence.flatMap((count) => (ACCEPTANCE_SOURCES[count.source] ? [ACCEPTANCE_SOURCES[count.source]] : []));
  const rests = sources.length > 0 ? `Accepted here means ${sources.join(" or ")}${row.evidence.some((count) => count.source === "operator_approval") ? "" : ", not your review"}.` : "";
  const completed = Number(row.completed);
  const finished = completed > 0 ? ` ${completed} more ${completed === 1 ? "unit" : "units"} only finished; acceptance is unknown and they do not count as accepted.` : "";
  return `${rests}${finished}`.trim();
}

function unknownClause(row: UsageOutcomeRow | undefined, response: UsageOutcomesResponse): string {
  const units = row ? Number(row.units) : Number(response.units);
  const unknown = row ? row.unknownRate : units > 0 ? Number(response.unknownUnits) / units : undefined;
  return units > 0 && unknown !== undefined ? `${formatWholePercent(unknown)} of outcomes are unknown` : "how many outcomes are unknown is not known";
}

/** The sentence for a compared row; ratios are of cost per accepted outcome, and they describe what was seen. */
export function comparisonSentence(row: UsageOutcomeRow): string {
  const cheaper = row.cheaperBy;
  const subject = row.key;
  const path = pathName(row.stratum);
  const consumption = known(row.consumptionRatio) ? row.consumptionRatio.value : undefined;
  const used = consumption === undefined || (consumption > 0.9 && consumption < 1.1) ? "" : `, using ${formatRatio(consumption)} the reference tokens`;
  const inverse = invertRange(cheaper);
  if (row.verdict === "cheaper") {
    return `On ${path}, ${subject} cost about ${formatRatio(cheaper?.value)} less per accepted outcome (range ${ratioRangeText(cheaper)}) than ${row.baseline}${used}`;
  }
  if (row.verdict === "more_expensive") {
    return `On ${path}, ${subject} cost about ${formatRatio(inverse?.value)} more per accepted outcome (range ${ratioRangeText(inverse)}) than ${row.baseline}${used}`;
  }
  if (row.verdict === "about_same") {
    return `On ${path}, ${subject} and ${row.baseline} cost about the same per accepted outcome: ${subject} cost ${ratioRangeText(inverse)} of ${row.baseline}, all inside the 30% margin`;
  }
  return `Inconclusive on ${path}: ${subject} cost ${ratioRangeText(inverse)} of ${row.baseline} per accepted outcome, too wide to call cheaper, dearer or the same`;
}

const STRENGTH: Record<EvidenceDesign, string> = {
  observational: "Compares models on the work each happened to run. Harder or easier work may differ between them, so this is an association to look into, not what the model caused.",
  randomized: "Each launch drew its model by chance, recorded per run, so nothing but chance chose which model got which work. It is still a range, and cost per accepted outcome is only as good as how acceptance was recorded.",
  day_block_randomized: "Whole days drew their model by chance, so a day counts once, however many sessions it held, and you saw the label when you launched. It is still a range, and cost per accepted outcome is only as good as how acceptance was recorded.",
};

function designOf(value: string): EvidenceDesign {
  return value === "randomized" || value === "day_block_randomized" ? value : "observational";
}

/** The state the data supports and its conclusion, in plain words. */
export function evidenceView(response: UsageOutcomesResponse, windowLabel: string): EvidenceView {
  const rows = response.rows;
  const headline = headlineRow(rows);
  const design = designOf(response.design);
  if (headline) {
    return {
      state: design,
      conclusion: `${comparisonSentence(headline)}; ${unknownClause(headline, response)}.`,
      strength: `${STRENGTH[design]} ${acceptanceClause(headline)}`.trim(),
      headline,
    };
  }
  const accepted = acceptedTotal(rows);
  const completed = Number(response.completedUnits);
  const finished = completed > 0 ? ` ${completed} more ${completed === 1 ? "unit" : "units"} only finished, with acceptance unknown.` : "";
  if (response.modePeriods.length === 0) {
    return {
      state: "no_mode",
      conclusion: `No usage mode ran in the ${windowLabel}, so there is no before and after to compare.`,
      strength: `${accepted > 0 ? `${accepted} accepted ${accepted === 1 ? "outcome" : "outcomes"} exist across models, but no model has the ${ENOUGH_ACCEPTED} on one path needed to be compared.` : "No accepted outcomes are on record yet."}${finished}`,
    };
  }
  return {
    state: "not_enough_data",
    conclusion: `Too early to say. ${accepted} accepted ${accepted === 1 ? "outcome" : "outcomes"} so far in the ${windowLabel}; a model needs ${ENOUGH_ACCEPTED} on one path to be compared, and ${unknownClause(undefined, response)}.`,
    strength: `No verdict is given until two models on one path have enough accepted outcomes.${finished}`,
  };
}

/** A price advantage, what it was spent on, and the cost ratio the two multiply to. */
export interface BreakEven {
  model: string;
  baseline: string;
  stratum: string;
  /** Realized: how many times less the model cost per reference token on these units. */
  advantage: number;
  /** Extra reference tokens per accepted outcome, with its range, on the same units. */
  used: UsageRange;
  /** Advantage over use: cost per accepted outcome, with the range the verdict reads. */
  cheaperBy: UsageRange;
  verdict: string;
  /** The model's price apart from its mix, at one token mix; undefined when unpriced. */
  modelPrice?: number;
  /** "price_book", "tier_prior", "fallback", "mixed". */
  modelPriceSource: string;
  /** True when a price the points or the ratio rest on is a prior, not a price-book entry. */
  prior: boolean;
}

export function breakEven(row: UsageOutcomeRow | undefined): BreakEven | undefined {
  if (!row || !(row.breakEvenRatio > 0) || !known(row.consumptionRatio) || !known(row.cheaperBy)) return undefined;
  return {
    model: row.key,
    baseline: row.baseline,
    stratum: row.stratum,
    advantage: row.breakEvenRatio,
    used: row.consumptionRatio,
    cheaperBy: row.cheaperBy,
    verdict: row.verdict,
    modelPrice: row.modelPriceRatio > 0 ? row.modelPriceRatio : undefined,
    modelPriceSource: row.modelPriceSource,
    prior: row.priceSource !== "price_book",
  };
}

/** A ratio as "3.6x less", or "1.4x more" for a value under one. */
function lessOrMore(ratio: number): string {
  return ratio >= 1 ? `${formatRatio(ratio)} less` : `${formatRatio(1 / ratio)} more`;
}

/** The break-even sentence: realized price, model price, use and what they multiply to, on one set of units. */
export function breakEvenSentence(b: BreakEven): string {
  const price = b.modelPrice === undefined
    ? "No model price is on record for this pair."
    : `The model price alone is ${lessOrMore(b.modelPrice)} at one fixed token mix (${b.modelPriceSource === "price_book" ? "from the price book" : "a prior from its tier, not a list price"}).`;
  return `On ${pathName(b.stratum)}, ${b.model} cost ${lessOrMore(b.advantage)} per reference token than ${b.baseline} on these units, its price and its token mix together. ${price} It used ${formatRatio(b.used.value)} the reference tokens per accepted outcome (range ${ratioRangeText(b.used)}), so it stays cheaper while that stays under ${formatRatio(b.advantage)}.`;
}

/** Sessions by how they started, for the coverage card. */
export const SESSION_TYPES = [
  { id: "headless", label: "Headless runs", paths: ["am_headless"] },
  { id: "interactive", label: "Interactive sessions", paths: ["am_interactive"] },
  { id: "imported", label: "Imported sessions", paths: ["imported", "imported_web_console"] },
] as const;

export interface SessionCoverage {
  id: string;
  label: string;
  runs: number;
  eligible: number;
  /** Share of eligible runs that have usage data; undefined when none were eligible. */
  usageShare?: number;
  /** Share of eligible runs with a recorded outcome. */
  outcomeShare?: number;
}

function measureShare(rows: readonly UsageParityRow[], name: string): number | undefined {
  let numerator = 0;
  let denominator = 0;
  for (const row of rows) {
    const measure = row.measures.find((m) => m.name === name);
    if (!measure) continue;
    numerator += Number(measure.numerator);
    denominator += Number(measure.denominator);
  }
  return denominator > 0 ? numerator / denominator : undefined;
}

/** Coverage by session type, summed over the paths and harnesses that make each type. */
export function sessionCoverage(rows: readonly UsageParityRow[]): SessionCoverage[] {
  return SESSION_TYPES.map((type) => {
    const mine = rows.filter((row) => (type.paths as readonly string[]).includes(row.path));
    return {
      id: type.id,
      label: type.label,
      runs: mine.reduce((sum, row) => sum + Number(row.runs), 0),
      eligible: mine.reduce((sum, row) => sum + Number(row.eligibleRuns), 0),
      usageShare: measureShare(mine, "usage"),
      outcomeShare: measureShare(mine, "outcome"),
    };
  });
}

/** A mode period as a label: its own label, else the runner it covered. */
export function periodName(period: UsageModePeriod): string {
  return period.label || (period.runner ? `Usage mode (${period.runner})` : "Usage mode");
}
