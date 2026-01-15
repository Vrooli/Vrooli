import { useCallback, useSyncExternalStore } from 'react';

const STORAGE_KEY = 'ecosystem-manager:phase-usage';
const MAX_RECENT = 10;

interface PhaseUsageData {
  recent: string[];
  frequency: Record<string, number>;
  lastUpdated: string;
}

const defaultUsageData: PhaseUsageData = {
  recent: [],
  frequency: {},
  lastUpdated: new Date().toISOString(),
};

// In-memory cache for the current value
let cachedData: PhaseUsageData | null = null;
let listeners: Set<() => void> = new Set();

function notifyListeners() {
  listeners.forEach((listener) => listener());
}

function loadFromStorage(): PhaseUsageData {
  if (cachedData) return cachedData;

  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      cachedData = JSON.parse(stored);
      return cachedData!;
    }
  } catch {
    // Ignore parse errors
  }

  cachedData = { ...defaultUsageData };
  return cachedData;
}

function saveToStorage(data: PhaseUsageData) {
  cachedData = data;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch {
    // Ignore storage errors (e.g., quota exceeded)
  }
  notifyListeners();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot(): PhaseUsageData {
  return loadFromStorage();
}

/**
 * Hook to track phase selection usage in localStorage.
 * Provides recent phases, frequency counts, and sorting utilities.
 */
export function usePhaseUsage() {
  const usageData = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

  const trackUsage = useCallback((phaseName: string) => {
    const current = loadFromStorage();

    // Update recent list (prepend, remove duplicates, limit)
    const newRecent = [phaseName, ...current.recent.filter((p) => p !== phaseName)].slice(
      0,
      MAX_RECENT
    );

    // Increment frequency
    const newFrequency = {
      ...current.frequency,
      [phaseName]: (current.frequency[phaseName] || 0) + 1,
    };

    saveToStorage({
      recent: newRecent,
      frequency: newFrequency,
      lastUpdated: new Date().toISOString(),
    });
  }, []);

  const sortByRecent = useCallback(
    <T extends { name: string }>(phases: T[]): T[] => {
      const recentSet = new Set(usageData.recent);
      const recentIndex = new Map(usageData.recent.map((name, idx) => [name, idx]));

      return [...phases].sort((a, b) => {
        const aRecent = recentSet.has(a.name);
        const bRecent = recentSet.has(b.name);

        // Recent items first
        if (aRecent && !bRecent) return -1;
        if (!aRecent && bRecent) return 1;

        // Among recent items, sort by recency (lower index = more recent)
        if (aRecent && bRecent) {
          return (recentIndex.get(a.name) || 0) - (recentIndex.get(b.name) || 0);
        }

        // Non-recent items: alphabetical
        return a.name.localeCompare(b.name);
      });
    },
    [usageData.recent]
  );

  const sortByFrequency = useCallback(
    <T extends { name: string }>(phases: T[]): T[] => {
      return [...phases].sort((a, b) => {
        const aFreq = usageData.frequency[a.name] || 0;
        const bFreq = usageData.frequency[b.name] || 0;

        // Higher frequency first
        if (aFreq !== bFreq) return bFreq - aFreq;

        // Same frequency: alphabetical
        return a.name.localeCompare(b.name);
      });
    },
    [usageData.frequency]
  );

  const sortByName = useCallback(<T extends { name: string }>(phases: T[]): T[] => {
    return [...phases].sort((a, b) => a.name.localeCompare(b.name));
  }, []);

  const clearUsage = useCallback(() => {
    saveToStorage({ ...defaultUsageData, lastUpdated: new Date().toISOString() });
  }, []);

  return {
    usageData,
    trackUsage,
    sortByRecent,
    sortByFrequency,
    sortByName,
    clearUsage,
  };
}

export type SortOption = 'name' | 'recent' | 'most-used';
