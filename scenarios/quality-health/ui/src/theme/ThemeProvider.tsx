import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { ThemeContext, type ThemeChoice, type ThemeContextValue } from "./themeContext";

const STORAGE_KEY = "vrooli.theme";

interface RuntimeWindow {
  localStorage: Storage;
  matchMedia?: Window["matchMedia"];
}

const runtimeWindow = (): RuntimeWindow | undefined => {
  const root = globalThis as unknown as { window?: RuntimeWindow };
  return root.window;
};

const runtimeDocument = (): Document | undefined => {
  const root = globalThis as unknown as { document?: Document };
  return root.document;
};

const readStoredChoice = (): ThemeChoice => {
  const win = runtimeWindow();
  if (!win) return "system";
  const stored = win.localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "system";
};

const resolveChoice = (choice: ThemeChoice): "light" | "dark" => {
  if (choice === "light" || choice === "dark") return choice;
  const win = runtimeWindow();
  if (!win?.matchMedia) return "light";
  return win.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
};

const applyTheme = (resolved: "light" | "dark", choice: ThemeChoice) => {
  const doc = runtimeDocument();
  if (!doc) return;
  // `system` clears the attribute so the CSS @media fallback in tokens.css
  // owns resolution. Explicit choices write the attribute.
  if (choice === "system") {
    doc.documentElement.removeAttribute("data-theme");
  } else {
    doc.documentElement.setAttribute("data-theme", resolved);
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
    const win = runtimeWindow();
    if (!win?.matchMedia) return undefined;
    if (choice !== "system") return undefined;
    const mq = win.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => setResolved(mq.matches ? "dark" : "light");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [choice]);

  const setTheme = useCallback((next: ThemeChoice) => {
    setChoice(next);
    setResolved(resolveChoice(next));
    const win = runtimeWindow();
    if (win) {
      win.localStorage.setItem(STORAGE_KEY, next);
    }
  }, []);

  const value = useMemo<ThemeContextValue>(() => ({ choice, resolved, setTheme }), [choice, resolved, setTheme]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}
