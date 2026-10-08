import { createContext, useContext } from "react";

export type ThemeChoice = "light" | "dark" | "system";
export type ThemeContextValue = { choice: ThemeChoice; setTheme: (choice: ThemeChoice) => void };

export const ThemeContext = createContext<ThemeContextValue | null>(null);

export function useTheme() {
  const value = useContext(ThemeContext);
  if (!value) throw new Error("useTheme must be used inside ThemeProvider");
  return value;
}
