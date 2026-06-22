import { strings } from "../consts/strings";
import type { ThemeChoice } from "./ThemeProvider";

/**
 * Static map of theme choice → typed translation key. Explicit `strings.*`
 * accessors (not a computed `strings.theme.choice[c]`) so the no-unused-keys
 * lint rule sees every leaf referenced. Consumed by both the top-bar select and
 * the settings-page radio group so the two surfaces never drift.
 */
export const THEME_CHOICE_LABEL: Record<
  ThemeChoice,
  (typeof strings.theme.choice)[keyof typeof strings.theme.choice]
> = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};
