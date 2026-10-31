/**
 * A per-launch context limit: how much conversation a new session keeps before
 * its harness compacts it. It is applied to the launch command (an environment
 * prefix or an argument) so it lasts for that session only and no harness
 * setting is saved. "standard" adds nothing, so a usage mode or the harness
 * default still applies.
 */
export type ContextLimitChoice = "keep_long" | "standard" | "tight";

export const CONTEXT_LIMIT_CHOICES: readonly ContextLimitChoice[] = ["keep_long", "standard", "tight"];

/** Tokens each choice asks the harness for. Claude ignores anything under 100k, so tight stays above it. */
export const CONTEXT_LIMIT_TOKENS: Record<Exclude<ContextLimitChoice, "standard">, number> = { keep_long: 1_000_000, tight: 150_000 };

const CODEX_ARG = "model_auto_compact_token_limit";
const CLAUDE_ENV = "CLAUDE_CODE_AUTO_COMPACT_WINDOW";

type Harness = "claude" | "codex";

/** Which harness a launch command starts, when it is a plain, simple invocation. */
export function launchHarness(command: string): Harness | null {
  const text = command.trim();
  if (/^vrooli\s+agent\s+launch\b/.test(text)) {
    const runner = /--runner[=\s]+(\S+)/.exec(text)?.[1];
    return runner === "claude" || runner === "claude-code" ? "claude" : runner === "codex" ? "codex" : null;
  }
  const first = text.split(/\s+/)[0] ?? "";
  return first === "claude" ? "claude" : first === "codex" ? "codex" : null;
}

/** Whether a limit can be applied to this command. Shell pipelines and wrappers are left alone. */
export function canLimitContext(command: string): boolean {
  return launchHarness(command) !== null && !/[;&|`$(]/.test(command);
}

/** True when the command already states its own limit, which then wins. */
export function hasExplicitLimit(command: string): boolean {
  return command.includes(CLAUDE_ENV) || command.includes(CODEX_ARG);
}

/**
 * The command with the chosen limit, or the command unchanged for "standard",
 * for a command that already sets one, or for one that is not a plain Claude or
 * Codex launch.
 */
export function withContextLimit(command: string, choice: ContextLimitChoice): string {
  if (choice === "standard" || !canLimitContext(command) || hasExplicitLimit(command)) return command;
  const tokens = String(CONTEXT_LIMIT_TOKENS[choice]);
  const text = command.trim();
  const harness = launchHarness(text);
  if (harness === "claude") return `${CLAUDE_ENV}=${tokens} ${text}`;
  if (/^vrooli\s+agent\s+launch\b/.test(text)) return `${text} --arg=-c --arg=${CODEX_ARG}=${tokens}`;
  const [binary = "codex", ...rest] = text.split(/\s+/);
  return [binary, "-c", `${CODEX_ARG}=${tokens}`, ...rest].join(" ");
}
