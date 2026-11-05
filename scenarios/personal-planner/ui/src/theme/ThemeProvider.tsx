import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { chromeTheme } from "@vrooli/react-component-library/ChromeTheme/1";
import { appearanceAt, OBSERVATORY_PREFERENCES_EVENT, readTransitionMinutes } from "./observatoryAppearance";

export type ThemeChoice = "auto" | "day" | "night";

const STORAGE_KEY = "vrooli.theme";

interface ThemeContextValue {
  /** The user's stated Observatory appearance choice (auto/day/night). */
  choice: ThemeChoice;
  /** The currently-applied theme; `system` follows the Observatory's local day/night window. */
  resolved: "light" | "dark";
  setTheme: (choice: ThemeChoice) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

const readStoredChoice = (): ThemeChoice => {
  if (typeof window === "undefined") return "auto";
  const stored = window.localStorage.getItem(STORAGE_KEY);
  if (stored === "auto" || stored === "day" || stored === "night") return stored;
  // Migrate the earlier generic vocabulary without changing the persisted
  // meaning for existing users.
  if (stored === "light") return "day";
  if (stored === "dark") return "night";
  if (stored === "system") return "auto";
  return "auto";
};

/**
 * Visual-oracle captures and deep links may request an appearance without
 * waiting for Today to mount. Resolve that request here so the shared shell,
 * browser chrome, and routed surface all start from the same choice.
 */
const readInitialChoice = (): ThemeChoice => {
  if (typeof window !== "undefined") {
    const requested = new URLSearchParams(window.location.search).get("appearance");
    if (requested === "auto" || requested === "day" || requested === "night") return requested;
  }
  return readStoredChoice();
};

const resolveChoice = (choice: ThemeChoice): "light" | "dark" => {
  if (choice === "day") return "light";
  if (choice === "night") return "dark";
  if (typeof window === "undefined") return "light";
  const transition = readTransitionMinutes();
  return appearanceAt(new Date(), transition.dayStart, transition.nightStart) === "night" ? "dark" : "light";
};

const applyTheme = (resolved: "light" | "dark", choice: ThemeChoice) => {
  if (typeof document === "undefined") return;
  // The design kit's dark palette keys on `data-resolved-theme` (see the
  // `[data-resolved-theme="dark"]` block in design-tokens.css), so the resolved
  // value is always written. `data-theme` records the product vocabulary for
  // anything that wants to know whether the appearance is explicit or auto.
  document.documentElement.setAttribute("data-resolved-theme", resolved);
  document.documentElement.style.colorScheme = resolved;
  if (choice === "auto") {
    document.documentElement.removeAttribute("data-theme");
  } else {
    document.documentElement.setAttribute("data-theme", choice);
  }
};

/** Keep media-specific browser chrome declarations aligned with the active appearance. */
const syncThemeColorVariants = (color: string) => {
  if (typeof document === "undefined") return;
  document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"][media]').forEach((meta) => {
    meta.content = color;
  });
};

interface ThemeProviderProps {
  children: ReactNode;
  /** Test override — skips localStorage and media-query reads. */
  initialChoice?: ThemeChoice;
}

export function ThemeProvider({ children, initialChoice }: ThemeProviderProps) {
  const [choice, setChoice] = useState<ThemeChoice>(() => initialChoice ?? readInitialChoice());
  const [resolved, setResolved] = useState<"light" | "dark">(() => resolveChoice(initialChoice ?? readInitialChoice()));

  useEffect(() => {
    applyTheme(resolved, choice);
    const chromeColor = getComputedStyle(document.documentElement)
      .getPropertyValue(resolved === "dark" ? "--color-background" : "--color-surface-muted")
      .trim() || (resolved === "dark" ? "rgb(23 22 52)" : "rgb(235 229 216)");
    chromeTheme.setBase({ statusColor: chromeColor, fillColor: chromeColor });
    syncThemeColorVariants(chromeColor);
  }, [resolved, choice]);

  useEffect(() => {
    if (choice !== "auto") return undefined;
    const refresh = () => setResolved(resolveChoice("auto"));
    const timer = window.setInterval(refresh, 60_000);
    window.addEventListener("storage", refresh);
    window.addEventListener(OBSERVATORY_PREFERENCES_EVENT, refresh);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("storage", refresh);
      window.removeEventListener(OBSERVATORY_PREFERENCES_EVENT, refresh);
    };
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
