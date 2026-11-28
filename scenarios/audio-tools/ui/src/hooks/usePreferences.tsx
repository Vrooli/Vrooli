import * as React from "react";

export type ThemeChoice = "light" | "dark" | "system";
export type FontScale = "compact" | "comfortable" | "large";

export interface Preferences {
  theme: ThemeChoice;
  fontScale: FontScale;
  reducedMotion: boolean;
}

const STORAGE_KEY = "audio-tools.preferences.v1";

const DEFAULT_PREFERENCES: Preferences = {
  theme: "system",
  fontScale: "comfortable",
  reducedMotion: false,
};

interface PreferencesContextValue {
  preferences: Preferences;
  resolvedTheme: "light" | "dark";
  setTheme: (theme: ThemeChoice) => void;
  setFontScale: (scale: FontScale) => void;
  setReducedMotion: (value: boolean) => void;
}

const PreferencesContext = React.createContext<PreferencesContextValue | null>(null);

function readStoredPreferences(): Preferences {
  if (typeof window === "undefined") return DEFAULT_PREFERENCES;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_PREFERENCES;
    const parsed = JSON.parse(raw) as Partial<Preferences>;
    return {
      theme: parsed.theme ?? DEFAULT_PREFERENCES.theme,
      fontScale: parsed.fontScale ?? DEFAULT_PREFERENCES.fontScale,
      reducedMotion: parsed.reducedMotion ?? DEFAULT_PREFERENCES.reducedMotion,
    };
  } catch {
    return DEFAULT_PREFERENCES;
  }
}

function resolveTheme(choice: ThemeChoice): "light" | "dark" {
  if (choice === "light" || choice === "dark") return choice;
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyDocumentTheme(resolved: "light" | "dark") {
  if (typeof document === "undefined") return;
  document.documentElement.dataset.resolvedTheme = resolved;
}

function applyFontScale(scale: FontScale) {
  if (typeof document === "undefined") return;
  const size = scale === "compact" ? "14px" : scale === "large" ? "17px" : "15px";
  document.documentElement.style.setProperty("font-size", size);
}

export function PreferencesProvider({ children }: { children: React.ReactNode }) {
  const [preferences, setPreferences] = React.useState<Preferences>(() => readStoredPreferences());
  const [resolvedTheme, setResolvedTheme] = React.useState<"light" | "dark">(() =>
    resolveTheme(preferences.theme),
  );

  // Persist + apply theme.
  React.useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences));
    } catch {
      // Quota or unavailable storage — preferences stay in-memory.
    }
    const next = resolveTheme(preferences.theme);
    setResolvedTheme(next);
    applyDocumentTheme(next);
    applyFontScale(preferences.fontScale);
  }, [preferences]);

  // React to OS theme changes when the user picked "system".
  React.useEffect(() => {
    if (preferences.theme !== "system" || typeof window === "undefined") return;
    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      const next = mql.matches ? "dark" : "light";
      setResolvedTheme(next);
      applyDocumentTheme(next);
    };
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, [preferences.theme]);

  const value = React.useMemo<PreferencesContextValue>(
    () => ({
      preferences,
      resolvedTheme,
      setTheme: (theme) => setPreferences((p) => ({ ...p, theme })),
      setFontScale: (fontScale) => setPreferences((p) => ({ ...p, fontScale })),
      setReducedMotion: (reducedMotion) => setPreferences((p) => ({ ...p, reducedMotion })),
    }),
    [preferences, resolvedTheme],
  );

  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>;
}

// The `usePreferences` hook lives in the same file as the
// `PreferencesProvider` component on purpose: the hook is the canonical
// consumer of the provider's context. Splitting them would force two
// imports for every call site of one primitive. HMR for this file is
// acceptable to break.
// eslint-disable-next-line react-refresh/only-export-components
export function usePreferences(): PreferencesContextValue {
  const ctx = React.useContext(PreferencesContext);
  if (!ctx) {
    throw new Error("usePreferences must be used inside <PreferencesProvider>");
  }
  return ctx;
}
