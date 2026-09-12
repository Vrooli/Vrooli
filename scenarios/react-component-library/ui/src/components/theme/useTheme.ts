import { useContext } from "react";

import { ThemeContext } from "./theme-context";
import type { ThemeContextValue } from "./theme-context";

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used inside <ThemeProvider>");
  }
  return ctx;
}
