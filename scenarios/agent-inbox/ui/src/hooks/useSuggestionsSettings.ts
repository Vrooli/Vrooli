/**
 * Hook for managing Suggestions panel settings.
 * Persists visibility and merge model preferences to localStorage.
 */

import { useCallback, useEffect, useState } from "react";
import type { SuggestionsSettings } from "@/lib/types/templates";

const SETTINGS_KEY = "agent-inbox:suggestions-settings";

const DEFAULT_SETTINGS: SuggestionsSettings = {
  visible: false, // Hidden by default
  mergeModel: "anthropic/claude-3-haiku-20240307", // Default to fast/cheap model
};

function loadSettings(): SuggestionsSettings {
  try {
    const stored = localStorage.getItem(SETTINGS_KEY);
    if (!stored) return DEFAULT_SETTINGS;
    const parsed = JSON.parse(stored);
    return {
      visible:
        typeof parsed.visible === "boolean"
          ? parsed.visible
          : DEFAULT_SETTINGS.visible,
      mergeModel:
        typeof parsed.mergeModel === "string"
          ? parsed.mergeModel
          : DEFAULT_SETTINGS.mergeModel,
    };
  } catch {
    return DEFAULT_SETTINGS;
  }
}

function saveSettings(settings: SuggestionsSettings): void {
  try {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
  } catch {
    // Silently fail if localStorage unavailable
  }
}

export interface UseSuggestionsSettingsReturn {
  visible: boolean;
  setVisible: (visible: boolean) => void;
  toggleVisible: () => void;
  mergeModel: string;
  setMergeModel: (modelId: string) => void;
}

export function useSuggestionsSettings(): UseSuggestionsSettingsReturn {
  const [settings, setSettings] = useState<SuggestionsSettings>(loadSettings);

  // Sync to localStorage when settings change
  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  const setVisible = useCallback((visible: boolean) => {
    setSettings((prev) => ({ ...prev, visible }));
  }, []);

  const toggleVisible = useCallback(() => {
    setSettings((prev) => ({ ...prev, visible: !prev.visible }));
  }, []);

  const setMergeModel = useCallback((mergeModel: string) => {
    setSettings((prev) => ({ ...prev, mergeModel }));
  }, []);

  return {
    visible: settings.visible,
    setVisible,
    toggleVisible,
    mergeModel: settings.mergeModel,
    setMergeModel,
  };
}
