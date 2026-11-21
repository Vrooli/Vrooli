import { createContext } from "react";

export type ThemeChoice = "light" | "dark" | "system";

export interface ThemeContextValue {
  /** The user's stated choice (light/dark/system). */
  choice: ThemeChoice;
  /** The currently-applied theme; `system` resolves to light or dark via media query. */
  resolved: "light" | "dark";
  setTheme: (choice: ThemeChoice) => void;
}

/**
 * Shared between ThemeProvider (writes) and useTheme (reads). Lives in its
 * own module so both files keep single-purpose exports (react-refresh
 * requires component files to export only components).
 */
export const ThemeContext = createContext<ThemeContextValue | null>(null);
