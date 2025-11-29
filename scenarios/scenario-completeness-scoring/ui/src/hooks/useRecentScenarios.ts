import { useState, useEffect, useCallback } from "react";

const STORAGE_KEY = "scs-recent-scenarios";
const MAX_RECENT = 5;

interface RecentScenario {
  name: string;
  lastViewed: number;
  score?: number;
  classification?: string;
}

/**
 * Check if localStorage is available (fails in sandboxed iframes, incognito, etc.)
 */
function isLocalStorageAvailable(): boolean {
  try {
    const testKey = "__storage_test__";
    window.localStorage.setItem(testKey, testKey);
    window.localStorage.removeItem(testKey);
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
      // Remove existing entry for this scenario
      const filtered = prev.filter((s) => s.name !== name);

      // Add to front
      const updated: RecentScenario[] = [
        { name, lastViewed: Date.now(), score, classification },
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
