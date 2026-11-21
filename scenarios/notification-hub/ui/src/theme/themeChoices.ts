import { strings } from "../consts/strings";
import type { ThemeChoice } from "./themeContext";

export const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

export const THEME_LABEL = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} as const satisfies Record<ThemeChoice, string>;
