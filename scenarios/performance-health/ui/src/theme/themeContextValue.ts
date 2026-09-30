import { createContext, useContext } from "react";

import { strings } from "../consts/strings";

export type ThemeChoice = "light" | "dark" | "system";

/**
 * Static map of theme choice → typed translation key. Explicit `strings.*`
 * accessors (not a computed `strings.theme.choice[c]`) so the no-unused-keys
 * lint rule can see each key is referenced. Shared by TopBar and SettingsPage.
 *
 * Lives in this component-free module (alongside the context + hook) so
 * `ThemeProvider.tsx` only exports the provider *component* and keeps
 * fast-refresh working.
 */
export const THEME_CHOICE_LABEL_KEY: Record<
  ThemeChoice,
  (typeof strings.theme.choice)[keyof typeof strings.theme.choice]
> = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};

export interface ThemeContextValue {
  /** The user's stated choice (light/dark/system). */
  choice: ThemeChoice;
  /** The currently-applied theme; `system` resolves to light or dark via media query. */
  resolved: "light" | "dark";
  setTheme: (choice: ThemeChoice) => void;
}

export const ThemeContext = createContext<ThemeContextValue | null>(null);

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be called inside <ThemeProvider>");
  }
  return ctx;
}
