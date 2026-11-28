import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

export type ThemeChoice = "light" | "dark" | "system";

const STORAGE_KEY = "vrooli.theme";

interface ThemeContextValue {
  /** The user's stated choice (light/dark/system). */
  choice: ThemeChoice;
  /** The currently-applied theme; `system` resolves to light or dark via media query. */
  resolved: "light" | "dark";
  setTheme: (choice: ThemeChoice) => void;
  /** Whether the browser accepted access to durable preference storage. */
  storageAvailable: boolean;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

interface ThemePreference {
  choice: ThemeChoice;
  storageAvailable: boolean;
}

const readStoredPreference = (): ThemePreference => {
  if (typeof window === "undefined") return { choice: "system", storageAvailable: false };
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    const choice = stored === "light" || stored === "dark" || stored === "system" ? stored : "system";
    return { choice, storageAvailable: true };
  } catch {
    return { choice: "system", storageAvailable: false };
  }
};

const resolveChoice = (choice: ThemeChoice): "light" | "dark" => {
  if (choice === "light" || choice === "dark") return choice;
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
};

const applyTheme = (resolved: "light" | "dark", choice: ThemeChoice) => {
  if (typeof document === "undefined") return;
  // The design kit's dark palette keys on `data-resolved-theme` (see the
  // `[data-resolved-theme="dark"]` block in design-tokens.css), so the resolved
  // value is always written. `data-theme` records the user's choice for
  // anything that wants to know whether the theme is explicit or follows the
  // system; `system` clears it.
  document.documentElement.setAttribute("data-resolved-theme", resolved);
  document.documentElement.style.colorScheme = resolved;
  if (choice === "system") {
    document.documentElement.removeAttribute("data-theme");
  } else {
    document.documentElement.setAttribute("data-theme", resolved);
  }
};

interface ThemeProviderProps {
  children: ReactNode;
  /** Test override — skips stored preference reads. */
  initialChoice?: ThemeChoice;
}

export function ThemeProvider({ children, initialChoice }: ThemeProviderProps) {
  const [{ choice, storageAvailable }, setPreference] = useState<ThemePreference>(() =>
    initialChoice ? { choice: initialChoice, storageAvailable: true } : readStoredPreference(),
  );
  const [resolved, setResolved] = useState<"light" | "dark">(() => resolveChoice(choice));

  useEffect(() => {
    applyTheme(resolved, choice);
  }, [resolved, choice]);

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") return undefined;
    if (choice !== "system") return undefined;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => setResolved(mq.matches ? "dark" : "light");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [choice]);

  const setTheme = useCallback((next: ThemeChoice) => {
    setResolved(resolveChoice(next));
    let storageAvailable = false;
    if (typeof window !== "undefined") {
      try {
        window.localStorage.setItem(STORAGE_KEY, next);
        storageAvailable = true;
      } catch {
        // The in-memory choice remains usable in storage-restricted browsers.
      }
    }
    setPreference({ choice: next, storageAvailable });
  }, []);

  const value = useMemo<ThemeContextValue>(() => ({ choice, resolved, setTheme, storageAvailable }), [choice, resolved, setTheme, storageAvailable]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be called inside <ThemeProvider>");
  }
  return ctx;
}
