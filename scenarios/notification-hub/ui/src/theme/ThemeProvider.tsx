import { createContext, useCallback, useContext, useEffect, useMemo, useState, useSyncExternalStore, type ReactNode } from "react";

export type ThemeChoice = "light" | "dark" | "system";

const STORAGE_KEY = "vrooli.theme";
export type LocalPreferenceKind = "theme" | "locale";

const failedPreferenceWrites = new Set<LocalPreferenceKind>();
const preferenceStorageListeners = new Set<() => void>();

const notifyPreferenceStorageChange = () => {
  for (const listener of preferenceStorageListeners) listener();
};

const recordPreferenceStorageResult = (kind: LocalPreferenceKind, succeeded: boolean) => {
  const changed = succeeded ? failedPreferenceWrites.delete(kind) : !failedPreferenceWrites.has(kind);
  if (!succeeded) failedPreferenceWrites.add(kind);
  if (changed) notifyPreferenceStorageChange();
};

/** Read one browser-local preference without allowing storage denial to escape. */
export const readLocalPreference = (kind: LocalPreferenceKind, key: string): string | null => {
  if (typeof window === "undefined") return null;
  try {
    const value = window.localStorage.getItem(key);
    recordPreferenceStorageResult(kind, true);
    return value;
  } catch {
    recordPreferenceStorageResult(kind, false);
    return null;
  }
};

/** Save one browser-local preference; the caller keeps its in-memory choice either way. */
export const writeLocalPreference = (kind: LocalPreferenceKind, key: string, value: string): boolean => {
  if (typeof window === "undefined") return false;
  try {
    window.localStorage.setItem(key, value);
    recordPreferenceStorageResult(kind, true);
    return true;
  } catch {
    recordPreferenceStorageResult(kind, false);
    return false;
  }
};

const subscribePreferenceStorage = (listener: () => void) => {
  preferenceStorageListeners.add(listener);
  return () => preferenceStorageListeners.delete(listener);
};

const getPreferenceStorageSnapshot = () => failedPreferenceWrites.size > 0;

export const usePreferenceStorageUnavailable = () => useSyncExternalStore(
  subscribePreferenceStorage,
  getPreferenceStorageSnapshot,
  () => false,
);

interface ThemeContextValue {
  /** The user's stated choice (light/dark/system). */
  choice: ThemeChoice;
  /** The currently-applied theme; `system` resolves to light or dark via media query. */
  resolved: "light" | "dark";
  setTheme: (choice: ThemeChoice) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

const readStoredChoice = (): ThemeChoice => {
  const stored = readLocalPreference("theme", STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "system";
};

const resolveChoice = (choice: ThemeChoice): "light" | "dark" => {
  if (choice === "light" || choice === "dark") return choice;
  if (typeof window === "undefined" || !window.matchMedia) return "light";
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
    if (typeof window === "undefined" || !window.matchMedia) return undefined;
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

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be called inside <ThemeProvider>");
  }
  return ctx;
}
