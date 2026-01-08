/**
 * Hook for tracking mode navigation history.
 * Used for frecency-based ordering of suggestions.
 */

import { useCallback, useEffect, useState } from "react";
import type { ModeHistoryEntry } from "@/lib/types/templates";
import { recordPathUsage, serializeModePath } from "@/lib/frecency";

const HISTORY_KEY = "agent-inbox:mode-history";
const MAX_HISTORY_ENTRIES = 100;

function loadHistory(): ModeHistoryEntry[] {
  try {
    const stored = localStorage.getItem(HISTORY_KEY);
    if (!stored) return [];
    const parsed = JSON.parse(stored);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (entry): entry is ModeHistoryEntry =>
        typeof entry === "object" &&
        typeof entry.path === "string" &&
        typeof entry.count === "number" &&
        typeof entry.lastUsed === "string"
    );
  } catch {
    return [];
  }
}

function saveHistory(history: ModeHistoryEntry[]): void {
  try {
    // Limit to max entries, keeping most recently used
    const sorted = [...history].sort(
      (a, b) =>
        new Date(b.lastUsed).getTime() - new Date(a.lastUsed).getTime()
    );
    const limited = sorted.slice(0, MAX_HISTORY_ENTRIES);
    localStorage.setItem(HISTORY_KEY, JSON.stringify(limited));
  } catch {
    // Silently fail if localStorage unavailable
  }
}

export interface UseModeHistoryReturn {
  history: ModeHistoryEntry[];
  recordUsage: (modePath: string[]) => void;
  clearHistory: () => void;
}

export function useModeHistory(): UseModeHistoryReturn {
  const [history, setHistory] = useState<ModeHistoryEntry[]>(loadHistory);

  // Sync to localStorage when history changes
  useEffect(() => {
    saveHistory(history);
  }, [history]);

  const recordUsage = useCallback((modePath: string[]) => {
    if (modePath.length === 0) return;

    setHistory((prev) => {
      const path = serializeModePath(modePath);
      return recordPathUsage(prev, path);
    });
  }, []);

  const clearHistory = useCallback(() => {
    setHistory([]);
  }, []);

  return {
    history,
    recordUsage,
    clearHistory,
  };
}
