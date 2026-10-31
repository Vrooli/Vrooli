import { useContext } from "react";

import { ThemeContext, type ThemeContextValue } from "./ThemeContext";

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be called inside <ThemeProvider>");
  }
  return ctx;
}
