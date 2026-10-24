import { strings } from "../consts/strings";
import type { ThemeChoice } from "./ThemeProvider";

/**
 * Label key per theme choice. Spelled out as literal accessors (rather than
 * `strings.theme.choice[choice]`) so the `no-unused-keys` lint rule sees a
 * callsite for every leaf — bracket indexing hides the leaf from its scan.
 */
export const THEME_CHOICE_LABEL = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} as const satisfies Record<ThemeChoice, string>;
