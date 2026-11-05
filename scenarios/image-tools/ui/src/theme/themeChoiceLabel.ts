import { strings } from "../consts/strings";
import type { ThemeChoice } from "./ThemeProvider";

/**
 * The three theme choices in display order. Shared by the top-bar toggle and
 * the Settings page so the two never drift.
 */
export const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Literal label map for the theme switcher. Written with explicit
 * `strings.theme.choice.<key>` accessors (not `strings.theme.choice[choice]`)
 * so each catalog key has a static callsite the `strings/no-unused-keys` lint
 * can see. `satisfies` (not a `: Record<…, string>` annotation) preserves the
 * literal key-path types `t()` is typed against.
 */
export const THEME_CHOICE_LABEL = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} satisfies Record<ThemeChoice, string>;
