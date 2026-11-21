import { createContext, useContext } from "react";

export type ThemeChoice = "light" | "dark" | "system";

export interface ThemeContextValue {
  /** The user's stated choice (light/dark/system). */
  choice: ThemeChoice;
  /** The currently-applied theme; `system` resolves to light or dark via media query. */
  resolved: "light" | "dark";
  setTheme: (choice: ThemeChoice) => void;
}

/**
 * Theme context + `useTheme` hook live here (not in `ThemeProvider.tsx`) so the
 * component file exports *only* the `ThemeProvider` component. That keeps
 * react-refresh's "only export components" contract intact for fast refresh,
 * while consumers (and tests) import the hook/type from this module.
 */
export const ThemeContext = createContext<ThemeContextValue | null>(null);

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be called inside <ThemeProvider>");
  }
  return ctx;
}
