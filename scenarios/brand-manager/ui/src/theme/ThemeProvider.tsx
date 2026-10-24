import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

export type ThemeChoice = "light" | "dark" | "system";

const STORAGE_KEY = "vrooli.theme";

interface ThemeContextValue {
  /** The user's stated choice (light/dark/system). */
  choice: ThemeChoice;
  /** The currently-applied theme; `system` resolves to light or dark via media query. */
  resolved: "light" | "dark";
  setTheme: (choice: ThemeChoice) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

const readStoredChoice = (): ThemeChoice => {
  if (typeof window === "undefined") return "system";
  const stored = window.localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "system";
};

// matchMedia is absent under SSR and some test environments, but the DOM lib
// types it as always-present — read it through an optional-typed view of
// globalThis so the presence check is a real branch, not a lint-flagged
// always-falsy condition.
type MatchMediaHost = { matchMedia?: (query: string) => MediaQueryList };

const resolveChoice = (choice: ThemeChoice): "light" | "dark" => {
  if (choice === "light" || choice === "dark") return choice;
  const host = globalThis as MatchMediaHost;
  if (!host.matchMedia) return "light";
  return host.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
};

const applyTheme = (resolved: "light" | "dark", choice: ThemeChoice) => {
  if (typeof document === "undefined") return;
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
    const host = globalThis as MatchMediaHost;
    if (choice !== "system" || !host.matchMedia) return undefined;
    const mq = host.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => setResolved(mq.matches ? "dark" : "light");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [choice]);

  const setTheme = useCallback((next: ThemeChoice) => {
    setChoice(next);
    setResolved(resolveChoice(next));
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, next);
    }
  }, []);

  const value = useMemo<ThemeContextValue>(() => ({ choice, resolved, setTheme }), [choice, resolved, setTheme]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be called inside <ThemeProvider>");
  }
  return ctx;
}
