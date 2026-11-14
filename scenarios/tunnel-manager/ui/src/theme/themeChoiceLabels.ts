import { strings } from "../consts/strings";
import type { ThemeChoice } from "./themeContext";

/**
 * Static theme-choice → translation-key map. Referencing each key explicitly
 * (rather than indexing `strings.theme.choice[choice]` dynamically) keeps the
 * `strings/no-unused-keys` lint rule able to see every catalog key as used.
 */
type ThemeChoiceKey = (typeof strings.theme.choice)[keyof typeof strings.theme.choice];

export const THEME_CHOICE_LABEL: Record<ThemeChoice, ThemeChoiceKey> = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};
