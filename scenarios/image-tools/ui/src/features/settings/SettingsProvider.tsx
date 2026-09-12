import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import {
  applySettings,
  i18n,
  readStoredSettings,
  writeStoredSettings,
  type SettingsState,
} from "./useSettings";

interface SettingsContextValue {
  settings: SettingsState;
  /** Merge-update one or more preferences; persists and re-applies. */
  setSettings: (patch: Partial<SettingsState>) => void;
}

const SettingsContext = createContext<SettingsContextValue | null>(null);

interface SettingsProviderProps {
  children: ReactNode;
  /** Test override — skips the localStorage read for a deterministic start. */
  initialSettings?: SettingsState;
}

/**
 * Owns the display/accessibility preference state, persists changes, and is the
 * single place that writes them to `<html>`. Mirrors `ThemeProvider`: read
 * once on mount, apply via an effect, persist on every change.
 */
export function SettingsProvider({ children, initialSettings }: SettingsProviderProps) {
  const [settings, setSettingsState] = useState<SettingsState>(
    () => initialSettings ?? readStoredSettings(),
  );

  // Apply on mount and whenever any preference changes.
  useEffect(() => {
    applySettings(settings);
  }, [settings]);

  // "Auto" text direction follows the locale, so re-apply when the language
  // changes (the i18n layer also writes `dir`, but an explicit ltr/rtl override
  // must win — re-applying here keeps the operator's choice authoritative).
  useEffect(() => {
    const handler = () => applySettings(settings);
    i18n.on("languageChanged", handler);
    return () => i18n.off("languageChanged", handler);
  }, [settings]);

  const setSettings = useCallback((patch: Partial<SettingsState>) => {
    setSettingsState((prev) => {
      const next = { ...prev, ...patch };
      writeStoredSettings(next);
      return next;
    });
  }, []);

  const value = useMemo<SettingsContextValue>(() => ({ settings, setSettings }), [settings, setSettings]);

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>;
}

export function useSettings(): SettingsContextValue {
  const ctx = useContext(SettingsContext);
  if (!ctx) {
    throw new Error("useSettings must be called inside <SettingsProvider>");
  }
  return ctx;
}
