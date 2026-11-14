import { strings } from "../consts/strings";
import type { ThemeChoice } from "./ThemeProvider";

/** Canonical ordering of the theme choices offered in the UI. */
export const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Static map from a theme choice to its i18n key. Defined explicitly rather than
 * indexed dynamically (`strings.theme.choice[choice]`) so the unused-key linter
 * can see each leaf key is referenced — a computed index hides the leaves from
 * static analysis and trips `strings/no-unused-keys`.
 */
export const THEME_CHOICE_LABEL_KEYS = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} satisfies Record<ThemeChoice, string>;
