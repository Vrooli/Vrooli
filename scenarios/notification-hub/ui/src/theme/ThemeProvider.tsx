import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { readLocalPreference, writeLocalPreference } from "./preferenceStorage";
import { ThemeContext, type ThemeChoice, type ThemeContextValue } from "./themeContext";

const STORAGE_KEY = "vrooli.theme";

const readStoredChoice = (): ThemeChoice => {
  const stored = readLocalPreference("theme", STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "system";
};

const resolveChoice = (choice: ThemeChoice): "light" | "dark" => {
  if (choice === "light" || choice === "dark") return choice;
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
};

const applyTheme = (resolved: "light" | "dark", choice: ThemeChoice) => {
  if (typeof document === "undefined") return;
  // Generated design tokens consume the resolved value, including system mode.
  document.documentElement.setAttribute("data-resolved-theme", resolved);
  // `system` clears the attribute so the CSS @media fallback in tokens.css
  // owns resolution. Explicit choices write the attribute.
  if (choice === "system") {
    document.documentElement.removeAttribute("data-theme");
  } else {
    document.documentElement.setAttribute("data-theme", resolved);
  }
};

interface ThemeProviderProps {
  children: ReactNode;
  /** Test override — skips localStorage and media-query reads. */
  initialChoice?: ThemeChoice;
}

export function ThemeProvider({ children, initialChoice }: ThemeProviderProps) {
  const [choice, setChoice] = useState<ThemeChoice>(() => initialChoice ?? readStoredChoice());
  const [resolved, setResolved] = useState<"light" | "dark">(() => resolveChoice(initialChoice ?? choice));

  useEffect(() => {
    applyTheme(resolved, choice);
  }, [resolved, choice]);

  useEffect(() => {
    if (typeof window === "undefined") return undefined;
    if (choice !== "system") return undefined;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => setResolved(mq.matches ? "dark" : "light");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [choice]);

  const setTheme = useCallback((next: ThemeChoice) => {
    setChoice(next);
    setResolved(resolveChoice(next));
    writeLocalPreference("theme", STORAGE_KEY, next);
  }, []);

  const value = useMemo<ThemeContextValue>(() => ({ choice, resolved, setTheme }), [choice, resolved, setTheme]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}
