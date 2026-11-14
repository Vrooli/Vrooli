// Pure helpers for usage modes: what a window's provider maps to, which end
// times a start can offer, and the not-measured effect figure.
import type { UsageAction, UsageMode, UsageModeStatus, UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatPoints } from "../../lib/format";
import { formatAt, friendlyLabel } from "./format";

export type ModeRunner = "claude" | "codex";
export type ModeDuration = "five_hour" | "weekly" | "time" | "off";

const PROVIDER_RUNNER: Record<string, ModeRunner> = { anthropic: "claude", openai: "codex" };
export const RUNNER_NAMES: Record<ModeRunner, string> = { claude: "Claude", codex: "Codex" };

export function runnerForProvider(provider: string): ModeRunner | "" {
  return PROVIDER_RUNNER[provider] ?? "";
}

/** The `until` the start procedure takes for a duration choice; hours apply to "time". */
export function untilValue(duration: ModeDuration, hours: number): string {
  switch (duration) {
    case "five_hour": return "5h-reset";
    case "weekly": return "weekly-reset";
    case "off": return "off";
    default: return `${hours}h`;
  }
}

export const DURATION_LABELS: Record<ModeDuration, string> = {
  five_hour: "Until 5-hour reset",
  weekly: "Until weekly reset",
  time: "For a set time",
  off: "Until I turn it off",
};

export const TIME_HOURS = [1, 2, 4, 8, 24, 72] as const;

/** The runner's current window of one length, from the forecast. */
export function windowFor(windows: readonly UsageWindowForecast[], runner: ModeRunner | "", minutes: bigint): UsageWindowForecast | undefined {
  if (!runner) return undefined;
  return windows.find((w) => runnerForProvider(w.provider) === runner && w.windowMinutes === minutes && !!w.resetsAt && new Date(w.resetsAt).getTime() > Date.now());
}

/** Why a reset-based duration cannot be chosen, or "" when it can. */
export function durationBlock(duration: ModeDuration, runner: ModeRunner | "", windows: readonly UsageWindowForecast[]): string {
  if (duration !== "five_hour" && duration !== "weekly") return "";
  if (!runner) return "Pick one coding agent under Advanced to follow its reset.";
  const found = windowFor(windows, runner, duration === "five_hour" ? 300n : 10080n);
  if (!found) return `${RUNNER_NAMES[runner]} has no ${duration === "five_hour" ? "5-hour" : "weekly"} reading, so its reset time is unknown.`;
  return "";
}

export function durationDetail(duration: ModeDuration, runner: ModeRunner | "", windows: readonly UsageWindowForecast[]): string {
  const block = durationBlock(duration, runner, windows);
  if (block) return block;
  if (duration === "off") return "We ask you daily whether to keep it on.";
  if (duration === "time") return "Choose how long.";
  const found = windowFor(windows, runner, duration === "five_hour" ? 300n : 10080n);
  return found ? `Resets ${formatAt(found.resetsAt)}` : "";
}

/** The effect range from numbers the recommendations already report; undefined when none exists. */
export interface EffectRange {
  /** Points of the window a change might still save, at most. */
  highPoints: number;
}

export function effectOfAction(action: UsageAction): EffectRange | undefined {
  const effect = action.effect;
  const candidates = [action.recoverablePoints, effect?.unit === "points" ? (effect.basis === "range" ? effect.upper : effect.value) : 0].filter((v) => v > 0);
  return candidates.length > 0 ? { highPoints: Math.max(...candidates) } : undefined;
}

export function effectText(effect: EffectRange | undefined, windowName: string): string {
  if (!effect) return `No figure exists for ${windowName} yet. Nothing is measured either way.`;
  return `Might save somewhere between 0 and about ${formatPoints(effect.highPoints)} of ${windowName} before it resets.`;
}

/** A mode's scope in words: "Codex", "Claude and Codex", with role or profile when set. */
export function scopeText(mode: UsageMode): string {
  const runner = mode.scope?.runner ?? "";
  const base = runner ? (RUNNER_NAMES[runner as ModeRunner] ?? runner) : "Claude and Codex";
  const extras = [mode.scope?.role && `role ${mode.scope.role}`, mode.scope?.profile && `profile ${mode.scope.profile}`].filter(Boolean);
  return extras.length > 0 ? `${base}, ${extras.join(", ")}` : base;
}

export function modeEnds(mode: UsageMode): string {
  return mode.untilKind === "off" || !mode.expiresAt ? "until you turn it off" : `until ${formatAt(mode.expiresAt)}`;
}

export function secondsLeftText(status: UsageModeStatus): string {
  if (status.mode?.untilKind === "off") return "no end time";
  const minutes = Math.max(0, Math.round(Number(status.secondsLeft) / 60));
  if (minutes < 60) return `${minutes} min left`;
  const hours = Math.floor(minutes / 60);
  return hours < 48 ? `${hours} h ${minutes % 60} min left` : `${Math.floor(hours / 24)} days left`;
}

/** Extend choices a banner offers for a mode: only those that can be honoured. */
export interface ExtendChoice {
  id: "weekly" | "two_hours" | "off";
  label: string;
  /** What the ExtendUsageMode call is asked to extend to: weekly-reset, a duration, or off. */
  to: string;
}

export function extendChoices(mode: UsageMode, windows: readonly UsageWindowForecast[]): ExtendChoice[] {
  const choices: ExtendChoice[] = [];
  const runner = mode.scope?.runner as ModeRunner | "";
  const weekly = windowFor(windows, runner, 10080n);
  if (weekly && (mode.untilKind === "time" || mode.untilKind === "window_reset") && (!mode.expiresAt || new Date(weekly.resetsAt) > new Date(mode.expiresAt))) {
    choices.push({ id: "weekly", label: "To weekly reset", to: "weekly-reset" });
  }
  if (mode.untilKind !== "off") {
    choices.push({ id: "two_hours", label: "2 more hours", to: "2h" });
    choices.push({ id: "off", label: "Until I turn it off", to: "off" });
  }
  return choices;
}

/** Who started a mode: an email or name as is; a bare subject id shortened, never shown whole. */
export function starterText(startedBy: string): string {
  if (!startedBy) return "an operator";
  return startedBy.includes("@") ? startedBy : `operator ${friendlyLabel(startedBy)}`;
}
