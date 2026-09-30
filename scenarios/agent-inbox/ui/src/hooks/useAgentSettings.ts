import { useState, useCallback, useEffect } from "react";
import { fetchProjectRoot } from "../lib/api";

const STORAGE_KEY = "agent-mode-settings";

/**
 * Agent mode settings persisted to localStorage.
 */
export interface AgentModeSettings {
  /** Default project path */
  defaultProjectPath: string;
}

const DEFAULT_SETTINGS: AgentModeSettings = {
  defaultProjectPath: ""
};

/**
 * Hook for managing agent mode settings.
 * Settings are persisted to localStorage.
 */
export function useAgentSettings() {
  const [settings, setSettingsState] = useState<AgentModeSettings>(DEFAULT_SETTINGS);
  const [isLoaded, setIsLoaded] = useState(false);

  // Load settings from localStorage on mount, then fill in project root default if needed
  useEffect(() => {
    let merged = { ...DEFAULT_SETTINGS };
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored) as Partial<AgentModeSettings>;
        merged = { ...merged, ...parsed };
      }
    } catch (e) {
      console.error("Failed to load agent settings:", e);
    }
    setSettingsState(merged);
    setIsLoaded(true);

    // If no project path is saved, fetch the server's project root as a default
    if (!merged.defaultProjectPath) {
      fetchProjectRoot()
        .then((root) => {
          if (root) {
            setSettingsState((prev) => {
              // Only apply if still empty (user may have typed something)
              if (prev.defaultProjectPath) return prev;
              const updated = { ...prev, defaultProjectPath: root };
              try {
                localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
              } catch { /* ignore */ }
              return updated;
            });
          }
        })
        .catch(() => { /* best-effort */ });
    }
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
