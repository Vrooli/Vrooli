import { useState, useEffect, useCallback } from "react";

const STORAGE_KEY = "scs-recent-scenarios";
const MAX_RECENT = 5;

export interface RecentScenario {
  name: string;
  lastViewed: number;
  score?: number;
  classification?: string;
  /** Previous score from last visit - used to show delta */
  previousScore?: number;
}

/**
 * Check if localStorage is available (fails in sandboxed iframes, incognito, etc.)
 * HARDENED: Guards against SecurityError when accessing window.localStorage property itself
 * in sandboxed iframes where storage access is completely denied. Some browsers throw
 * when even accessing the localStorage property, not just when calling methods on it.
 */
function isLocalStorageAvailable(): boolean {
  // First check if we're in a browser context
  if (typeof window === "undefined") {
    return false;
  }

  try {
    // In strict sandboxed iframes, even accessing window.localStorage throws
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    const storage = window.localStorage;
    if (!storage) {
      return false;
    }
    const testKey = "__scs_storage_test__";
    storage.setItem(testKey, testKey);
    storage.removeItem(testKey);
    return true;
  } catch {
    return false;
  }
}

/**
 * Hook for tracking recently viewed scenarios in localStorage.
 * Returns the list of recent scenarios and a function to mark a scenario as viewed.
 * Gracefully handles environments where localStorage is unavailable.
 */
export function useRecentScenarios() {
  const [recentScenarios, setRecentScenarios] = useState<RecentScenario[]>([]);
  const [storageAvailable] = useState(() => isLocalStorageAvailable());

  // Load from localStorage on mount
  useEffect(() => {
    if (!storageAvailable) return;
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored) as RecentScenario[];
        // Sort by most recently viewed
        parsed.sort((a, b) => b.lastViewed - a.lastViewed);
        setRecentScenarios(parsed.slice(0, MAX_RECENT));
      }
    } catch {
      // Ignore parse errors, start fresh
    }
  }, [storageAvailable]);

  // Mark a scenario as recently viewed
  const markViewed = useCallback((name: string, score?: number, classification?: string) => {
    setRecentScenarios((prev) => {
      // Find existing entry to preserve previous score
      const existing = prev.find((s) => s.name === name);
      const filtered = prev.filter((s) => s.name !== name);

      // Capture previous score for delta tracking
      // If we have an existing entry with a score, that becomes the previousScore
      const previousScore = existing?.score;

      // Add to front with previous score tracking
      const updated: RecentScenario[] = [
        { name, lastViewed: Date.now(), score, classification, previousScore },
        ...filtered,
      ].slice(0, MAX_RECENT);

      // Persist to localStorage if available
      if (storageAvailable) {
        try {
          localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
        } catch {
          // Storage might be full, ignore
        }
      }

      return updated;
    });
  }, [storageAvailable]);

  // Update score/classification for a scenario (when data is fetched)
  const updateScenarioData = useCallback((name: string, score: number, classification: string) => {
    setRecentScenarios((prev) => {
      const updated = prev.map((s) =>
        s.name === name ? { ...s, score, classification } : s
      );

      // Persist if available
      if (storageAvailable) {
        try {
          localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
        } catch {
          // Ignore
        }
      }

      return updated;
    });
  }, [storageAvailable]);

  return { recentScenarios, markViewed, updateScenarioData };
}
