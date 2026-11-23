import { createContext } from "react";

import type { SettingsState } from "./useSettings";

export interface SettingsContextValue {
  settings: SettingsState;
  /** Merge-update one or more preferences; persists and re-applies. */
  setSettings: (patch: Partial<SettingsState>) => void;
}

/**
 * The context `SettingsProvider` fills. Exported separately so a component that
 * can render outside a provider (it falls back to the defaults) can read it with
 * `useContext` instead of the throwing `useSettings` hook.
 */
export const SettingsContext = createContext<SettingsContextValue | null>(null);
