/**
 * ThemeProvider — light / dark / system.
 *
 * Source of truth is React state; the resolved value (light or dark) is
 * mirrored onto `<html>` as `class="dark"` plus `data-resolved-theme`, so
 * the design-tokens.css selectors flip in lockstep. System mode listens to
 * `prefers-color-scheme` so OS-level toggles flow through without a reload.
 */
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { ThemeContext, type ResolvedTheme, type Theme } from "./theme-context";

const STORAGE_KEY = "react-component-library.theme.v1";

function readStorage(): Theme {
  if (typeof window === "undefined") return "system";
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (raw === "light" || raw === "dark" || raw === "system") return raw;
  } catch {
    /* ignore */
  }
  return "system";
}

function readSystem(): ResolvedTheme {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return "light";
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyResolved(resolved: ResolvedTheme): void {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  root.setAttribute("data-resolved-theme", resolved);
  if (resolved === "dark") root.classList.add("dark");
  else root.classList.remove("dark");
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => readStorage());
  const [systemResolved, setSystemResolved] = useState<ResolvedTheme>(() => readSystem());

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") return;
    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (e: MediaQueryListEvent) => setSystemResolved(e.matches ? "dark" : "light");
    mql.addEventListener("change", handler);
    setSystemResolved(mql.matches ? "dark" : "light");
    return () => mql.removeEventListener("change", handler);
  }, []);

  const resolved: ResolvedTheme = theme === "system" ? systemResolved : theme;

  useEffect(() => {
    applyResolved(resolved);
  }, [resolved]);

  const setTheme = useCallback((next: Theme) => {
    setThemeState(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, next);
    } catch {
      /* ignore */
    }
  }, []);

  const value = useMemo(
    () => ({ theme, resolved, setTheme }),
    [theme, resolved, setTheme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}
