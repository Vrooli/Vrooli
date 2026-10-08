import { createContext } from "react";

export type Theme = "light" | "dark" | "system";
export type ResolvedTheme = "light" | "dark";

export interface ThemeContextValue {
  theme: Theme;
  resolved: ResolvedTheme;
  setTheme: (next: Theme) => void;
}

export const ThemeContext = createContext<ThemeContextValue | null>(null);
