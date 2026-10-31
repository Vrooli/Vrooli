// Pure logic for Learn mode: the plain-language verdicts, the time to a verdict
// and what a role's pace says about a comparison. Every figure comes from
// `usage experiment`; a value the API did not give is said to be unknown.
import type { UsageExperiment, UsageExperimentHorizon, UsageExperimentPlan, UsageOutcomeRow } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";

export type Verdict = "cheaper_looks_better" | "about_the_same" | "higher_looks_better" | "inconclusive" | "withheld" | "not_enough_data";

export const VERDICT_TEXT: Record<Verdict, string> = {
  cheaper_looks_better: "Cheaper looks better",
  about_the_same: "About the same",
  higher_looks_better: "Higher looks better",
  inconclusive: "Inconclusive",
  withheld: "Withheld",
  not_enough_data: "Not enough data",
};

export function verdictOf(value: string): Verdict {
  return value in VERDICT_TEXT ? (value as Verdict) : "not_enough_data";
}

/** Roles that usage modes never touch; the API refuses them and the setup page says why. */
export const EXEMPT_ROLES = ["code.fallback", "code.review", "judgment.supervision", "code.supervision"] as const;

export function isExemptRole(role: string): boolean {
  return (EXEMPT_ROLES as readonly string[]).includes(role);
}

/** Rows that are not roles: sessions and sub-agents carry no role to compare. */
const NOT_ROLES = new Set(["operator_session", "sub_agent", "unassigned", "unknown"]);

export interface LearnRole {
  key: string;
  /** claude or codex: the runner name the API takes. */
  runner: "claude" | "codex";
  /** Finished launches a week over the window the pace was read from. */
  perWeek: number;
  exempt: boolean;
}

export function runnerKey(runner: string): "claude" | "codex" | "" {
  const name = runner.toLowerCase();
  if (name.startsWith("claude")) return "claude";
  return name === "codex" ? "codex" : "";
}

/** Roles with managed runs, from `usage outcomes --by role` over `days`. */
export function learnRoles(rows: readonly UsageOutcomeRow[], days: number): LearnRole[] {
  const out: LearnRole[] = [];
  for (const row of rows) {
    const runner = runnerKey(row.runner);
    if (!runner || NOT_ROLES.has(row.key) || Number(row.units) === 0) continue;
    out.push({ key: row.key, runner, perWeek: (Number(row.units) / days) * 7, exempt: isExemptRole(row.key) });
  }
  return out.sort((a, b) => Number(a.exempt) - Number(b.exempt) || b.perWeek - a.perWeek || a.key.localeCompare(b.key));
}

/** "about 9 a week", "about 1 a week", "under 1 a week" for a pace. */
export function paceText(perWeek: number): string {
  if (perWeek < 0.5) return "under 1 a week";
  return `about ${Math.round(perWeek)} a week`;
}

/** Days as an operator plans: days, weeks, months, or over a number of years. */
export function humanDays(days: number): string {
  if (!(days > 0)) return "not reached";
  if (days < 14) return `${Math.ceil(days)} days`;
  if (days < 90) return `${Math.ceil(days / 7)} weeks`;
  if (days < 730) return `${Math.ceil(days / 30)} months`;
  return `over ${Math.floor(days / 365)} years`;
}

export function horizonText(horizon: UsageExperimentHorizon | undefined): string {
  if (!horizon || !horizon.reached) return "not within the range simulated (more than 700 launches per side)";
  const middle = humanDays(horizon.days);
  const lead = middle.startsWith("over ") ? middle : `about ${middle}`;
  return horizon.highDays > 0 ? `${lead}, likely ${humanDays(horizon.lowDays)} to ${humanDays(horizon.highDays)}` : `${lead}, at least ${humanDays(horizon.lowDays).replace(/^over /, "")} and the upper end is beyond the range simulated`;
}

export interface Headline {
  /** The sentence that leads the estimate, e.g. "Probably 3 to 6 weeks, and it could take longer." */
  text: string;
  /** True when no estimate could be made for at least one chosen role. */
  open: boolean;
  /** Position of the likely range on a 0..1 bar whose end is the longest upper bound. */
  bar?: { from: number; to: number };
}

/** The estimate for the chosen roles: the slowest role decides when every comparison has an answer. */
export function headline(plans: readonly UsageExperimentPlan[]): Headline {
  const detect = plans.map((plan) => plan.detect).filter((h): h is UsageExperimentHorizon => !!h);
  if (detect.length === 0) return { text: "Choose a role to see how long an answer takes.", open: true };
  const reached = detect.filter((h) => h.reached);
  if (reached.length < detect.length) return { text: "More than two years at this pace, so this role is too quiet to compare.", open: true };
  const slowest = reached.reduce((a, b) => (b.days > a.days ? b : a));
  const high = slowest.highDays > 0 ? slowest.highDays : slowest.days * 2;
  const low = slowest.lowDays;
  const long = slowest.days >= 730;
  const text = long ? `Probably ${humanDays(slowest.days)}, so a large effect is the only thing this could show.` : `Probably ${humanDays(low)} to ${humanDays(high)}, and it could take longer.`;
  return { text, open: long, bar: { from: Math.min(0.95, low / (high * 1.15)), to: Math.min(1, high / (high * 1.15)) } };
}

/** "about half", "about a fifth", "about N%" for a challenger share. */
export function shareWords(percent: number): string {
  if (percent === 50) return "about half";
  if (percent === 20) return "about a fifth";
  return `about ${percent}%`;
}

/**
 * Says when acceptance is partly assumed: some accepted outcomes rest only on the
 * agent's own "complete" verdict, with no approval or validation behind them.
 * Empty when none do.
 */
export function assumedAcceptanceText(experiment: Pick<UsageExperiment, "incumbent" | "challenger">): string {
  const arms = [experiment.incumbent, experiment.challenger].filter((arm): arm is NonNullable<typeof arm> => !!arm);
  const assumed = arms.reduce((sum, arm) => sum + Number(arm.assumedAccepted), 0);
  if (assumed === 0) return "";
  const accepted = arms.reduce((sum, arm) => sum + Number(arm.accepted), 0);
  return `Acceptance is partly assumed: ${assumed} of ${accepted} accepted outcomes rest only on the agent's own "complete" verdict, with no approval or validation behind them.`;
}

export function ratioText(value: number | undefined): string {
  return value === undefined || !Number.isFinite(value) ? "unknown" : `${value.toFixed(2)}x`;
}

/** One sentence per verdict that says what was seen and how sure it is. */
export function experimentSentence(experiment: UsageExperiment): string {
  const role = experiment.role;
  switch (verdictOf(experiment.verdict)) {
    case "cheaper_looks_better":
      return `For ${role}, the cheaper model looks better: it cost less per accepted outcome without more rework.`;
    case "about_the_same":
      return `For ${role}, the two models look about the same per accepted outcome.`;
    case "higher_looks_better":
      return `For ${role}, the higher model looks better: the cheaper one cost more per accepted outcome or needed more rework.`;
    case "inconclusive":
      return `For ${role}, the planned sample was read and it does not show whether the two differ.`;
    case "withheld":
      return `For ${role}, a verdict is withheld: the data cannot be trusted to settle it.`;
    default:
      return `For ${role}, there is not enough data yet.`;
  }
}
