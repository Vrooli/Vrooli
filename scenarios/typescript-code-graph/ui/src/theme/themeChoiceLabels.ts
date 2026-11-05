import { strings } from "../consts/strings";
import type { ThemeChoice } from "./ThemeProvider";

/**
 * Static map from a theme choice to its i18n key. Defined once and shared by
 * the TopBar select and the Settings radios so the per-leaf keys
 * (`theme.choice.light` …) are referenced via a static accessor — the
 * `strings/no-unused-keys` audit can't see dynamic `strings.theme.choice[c]`
 * indexing, and would otherwise flag the leaves as dead.
 */
export const THEME_CHOICE_LABELS = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} as const satisfies Record<ThemeChoice, string>;
