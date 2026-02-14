import { useState, useCallback, useEffect } from "react";
import type { RunnerType } from "../lib/api";

const STORAGE_KEY = "agent-mode-settings";

/**
 * Agent mode settings persisted to localStorage.
 */
export interface AgentModeSettings {
  /** Default runner type */
  defaultRunner: RunnerType;
  /** Default project path */
  defaultProjectPath: string;
  /** Default model (empty = use runner default) */
  defaultModel: string;
  /** Default max turns (0 = no limit) */
  defaultMaxTurns: number;
}

const DEFAULT_SETTINGS: AgentModeSettings = {
  defaultRunner: "claude-code",
  defaultProjectPath: "",
  defaultModel: "",
  defaultMaxTurns: 0
};

/**
 * Hook for managing agent mode settings.
 * Settings are persisted to localStorage.
 */
export function useAgentSettings() {
  const [settings, setSettingsState] = useState<AgentModeSettings>(DEFAULT_SETTINGS);
  const [isLoaded, setIsLoaded] = useState(false);

  // Load settings from localStorage on mount
  useEffect(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored) as Partial<AgentModeSettings>;
        setSettingsState({ ...DEFAULT_SETTINGS, ...parsed });
      }
    } catch (e) {
      console.error("Failed to load agent settings:", e);
    }
    setIsLoaded(true);
  }, []);

  // Save settings to localStorage
  const setSettings = useCallback((newSettings: Partial<AgentModeSettings>) => {
    setSettingsState((prev) => {
      const updated = { ...prev, ...newSettings };
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
      } catch (e) {
        console.error("Failed to save agent settings:", e);
      }
      return updated;
    });
  }, []);

  // Reset to defaults
  const resetSettings = useCallback(() => {
    setSettingsState(DEFAULT_SETTINGS);
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (e) {
      console.error("Failed to reset agent settings:", e);
    }
  }, []);

  return {
    settings,
    setSettings,
    resetSettings,
    isLoaded
  };
}

export default useAgentSettings;
