/**
 * Hook for managing Suggestions settings.
 * Local UI prefs (panel visibility + merge model) use localStorage.
 * Auto-suggest behavior is loaded from and saved to server-backed config.
 */

import { useCallback, useEffect, useState } from "react";
import {
  getSuggestionsSettings,
  setSuggestionsSettings,
  type SuggestionsAutoSuggestConfig,
  type SuggestionsSettingsResponse,
} from "@/lib/api";
import type { SuggestionsSettings } from "@/lib/types/templates";

const SETTINGS_KEY = "agent-inbox:suggestions-settings";

const DEFAULT_SETTINGS: SuggestionsSettings = {
  visible: false,
  mergeModel: "anthropic/claude-3-haiku-20240307",
};

const DEFAULT_AUTO_SUGGEST: SuggestionsAutoSuggestConfig = {
  enabled: true,
  debounceMs: 900,
  throttleMs: 10000,
  minInputLength: 10,
  minScorePercent: 35,
  maxSuggestions: 5,
};

function loadSettings(): SuggestionsSettings {
  try {
    const stored = localStorage.getItem(SETTINGS_KEY);
    if (!stored) return DEFAULT_SETTINGS;
    const parsed = JSON.parse(stored) as Record<string, unknown>;
    return {
      visible: typeof parsed.visible === "boolean" ? parsed.visible : DEFAULT_SETTINGS.visible,
      mergeModel: typeof parsed.mergeModel === "string" ? parsed.mergeModel : DEFAULT_SETTINGS.mergeModel,
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

  autoSuggest: SuggestionsAutoSuggestConfig;
  autoSuggestLoading: boolean;
  autoSuggestError: string | null;
  refreshAutoSuggest: () => Promise<void>;
  updateAutoSuggest: (next: SuggestionsAutoSuggestConfig) => Promise<void>;
}

export function useSuggestionsSettings(): UseSuggestionsSettingsReturn {
  const [settings, setSettings] = useState<SuggestionsSettings>(loadSettings);
  const [autoSuggest, setAutoSuggest] = useState<SuggestionsAutoSuggestConfig>(DEFAULT_AUTO_SUGGEST);
  const [autoSuggestLoading, setAutoSuggestLoading] = useState(true);
  const [autoSuggestError, setAutoSuggestError] = useState<string | null>(null);

  // Sync local UI prefs to localStorage
  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  const loadAutoSuggest = useCallback(async () => {
    setAutoSuggestLoading(true);
    setAutoSuggestError(null);
    try {
      const response = await getSuggestionsSettings();
      setAutoSuggest(response.autoSuggest);
    } catch (error) {
      setAutoSuggestError(error instanceof Error ? error.message : "Failed to load suggestions settings");
      setAutoSuggest(DEFAULT_AUTO_SUGGEST);
    } finally {
      setAutoSuggestLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAutoSuggest();
  }, [loadAutoSuggest]);

  const setVisible = useCallback((visible: boolean) => {
    setSettings((prev) => ({ ...prev, visible }));
  }, []);

  const toggleVisible = useCallback(() => {
    setSettings((prev) => ({ ...prev, visible: !prev.visible }));
  }, []);

  const setMergeModel = useCallback((mergeModel: string) => {
    setSettings((prev) => ({ ...prev, mergeModel }));
  }, []);

  const updateAutoSuggest = useCallback(async (next: SuggestionsAutoSuggestConfig) => {
    setAutoSuggestError(null);
    const payload: SuggestionsSettingsResponse = { autoSuggest: next };
    const saved = await setSuggestionsSettings(payload);
    setAutoSuggest(saved.autoSuggest);
  }, []);

  return {
    visible: settings.visible,
    setVisible,
    toggleVisible,
    mergeModel: settings.mergeModel,
    setMergeModel,

    autoSuggest,
    autoSuggestLoading,
    autoSuggestError,
    refreshAutoSuggest: loadAutoSuggest,
    updateAutoSuggest,
  };
}
