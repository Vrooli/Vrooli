import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { ThemeContext, type ThemeChoice, type ThemeContextValue } from "./ThemeContext";

export type { ThemeChoice };

const STORAGE_KEY = "vrooli.theme";

const readStoredChoice = (): ThemeChoice => {
  const stored = window.localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "system";
};

const getMatchMedia = (): Window["matchMedia"] | undefined => {
  const maybeMatchMedia: unknown = Reflect.get(window, "matchMedia");
  if (typeof maybeMatchMedia !== "function") return undefined;
  return (query: string) => maybeMatchMedia.call(window, query) as MediaQueryList;
};

const resolveChoice = (choice: ThemeChoice): "light" | "dark" => {
  if (choice === "light" || choice === "dark") return choice;
  const matchMedia = getMatchMedia();
  if (!matchMedia) return "light";
  return matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
};

const applyTheme = (resolved: "light" | "dark", choice: ThemeChoice) => {
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
  const [resolved, setResolved] = useState<"light" | "dark">(() => resolveChoice(initialChoice ?? readStoredChoice()));

  useEffect(() => {
    applyTheme(resolved, choice);
  }, [resolved, choice]);

  useEffect(() => {
    const matchMedia = getMatchMedia();
    if (!matchMedia) return undefined;
    if (choice !== "system") return undefined;
    const mq = matchMedia("(prefers-color-scheme: dark)");
    const handler = () => setResolved(mq.matches ? "dark" : "light");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [choice]);

  const setTheme = useCallback((next: ThemeChoice) => {
    setChoice(next);
    setResolved(resolveChoice(next));
    window.localStorage.setItem(STORAGE_KEY, next);
  }, []);

  const value = useMemo<ThemeContextValue>(() => ({ choice, resolved, setTheme }), [choice, resolved, setTheme]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}
