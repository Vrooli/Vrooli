import { strings } from "../consts/strings";
import type { ThemeChoice } from "./ThemeProvider";

/** Canonical theme-choice order, shared by the top-bar select and settings page. */
export const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Static label-key map per theme choice. Declared with explicit member access
 * (not `strings.theme.choice[c]`) so the `strings/no-unused-keys` audit sees a
 * concrete callsite for each key.
 */
type ThemeLabelKey = (typeof strings.theme.choice)[keyof typeof strings.theme.choice];

export const THEME_LABEL: Record<ThemeChoice, ThemeLabelKey> = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};
