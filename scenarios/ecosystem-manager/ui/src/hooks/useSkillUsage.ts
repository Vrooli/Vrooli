import { useCallback, useSyncExternalStore } from 'react';

const STORAGE_KEY = 'ecosystem-manager:skill-usage';
const MAX_RECENT = 10;

interface SkillUsageData {
  recent: string[];
  frequency: Record<string, number>;
  lastUpdated: string;
}

const defaultUsageData: SkillUsageData = {
  recent: [],
  frequency: {},
  lastUpdated: new Date().toISOString(),
};

// In-memory cache for the current value
let cachedData: SkillUsageData | null = null;
const listeners: Set<() => void> = new Set();

function isSkillUsageData(value: unknown): value is SkillUsageData {
	if (!value || typeof value !== 'object') return false;
	const candidate = value as Record<string, unknown>;
	return (
		Array.isArray(candidate.recent) &&
		candidate.recent.every((item) => typeof item === 'string') &&
		typeof candidate.frequency === 'object' &&
		candidate.frequency !== null &&
		typeof candidate.lastUpdated === 'string'
	);
}

function notifyListeners() {
  listeners.forEach((listener) => listener());
}

function loadFromStorage(): SkillUsageData {
  if (cachedData) return cachedData;

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			const parsed: unknown = JSON.parse(stored);
			if (isSkillUsageData(parsed)) {
				cachedData = parsed;
				return parsed;
			}
		}
	} catch {
    // Ignore parse errors
  }

  cachedData = { ...defaultUsageData };
  return cachedData;
}

function saveToStorage(data: SkillUsageData) {
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

function getSnapshot(): SkillUsageData {
  return loadFromStorage();
}

/**
 * Hook to track skill selection usage in localStorage.
 * Provides recent skills, frequency counts, and sorting utilities.
 */
export function useSkillUsage() {
  const usageData = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

  const trackUsage = useCallback((skillId: string) => {
    const current = loadFromStorage();

    // Update recent list (prepend, remove duplicates, limit)
    const newRecent = [skillId, ...current.recent.filter((p) => p !== skillId)].slice(
      0,
      MAX_RECENT
    );

    // Increment frequency
    const newFrequency = {
      ...current.frequency,
      [skillId]: (current.frequency[skillId] || 0) + 1,
    };

    saveToStorage({
      recent: newRecent,
      frequency: newFrequency,
      lastUpdated: new Date().toISOString(),
    });
  }, []);

  const sortByRecent = useCallback(
    <T extends { id?: string; name: string }>(skills: T[]): T[] => {
      const recentSet = new Set(usageData.recent);
      const recentIndex = new Map(usageData.recent.map((name, idx) => [name, idx]));

      return [...skills].sort((a, b) => {
        const aKey = a.id ?? a.name;
        const bKey = b.id ?? b.name;
        const aRecent = recentSet.has(aKey);
        const bRecent = recentSet.has(bKey);

        // Recent items first
        if (aRecent && !bRecent) return -1;
        if (!aRecent && bRecent) return 1;

        // Among recent items, sort by recency (lower index = more recent)
        if (aRecent && bRecent) {
          return (recentIndex.get(aKey) || 0) - (recentIndex.get(bKey) || 0);
        }

        // Non-recent items: alphabetical
        return a.name.localeCompare(b.name);
      });
    },
    [usageData.recent]
  );

  const sortByFrequency = useCallback(
    <T extends { id?: string; name: string }>(skills: T[]): T[] => {
      return [...skills].sort((a, b) => {
        const aKey = a.id ?? a.name;
        const bKey = b.id ?? b.name;
        const aFreq = usageData.frequency[aKey] || 0;
        const bFreq = usageData.frequency[bKey] || 0;

        // Higher frequency first
        if (aFreq !== bFreq) return bFreq - aFreq;

        // Same frequency: alphabetical
        return a.name.localeCompare(b.name);
      });
    },
    [usageData.frequency]
  );

  const sortByName = useCallback(<T extends { id?: string; name: string }>(skills: T[]): T[] => {
    return [...skills].sort((a, b) => a.name.localeCompare(b.name));
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
