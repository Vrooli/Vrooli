import { useState, useCallback, useRef, useEffect } from "react";

const HISTORY_KEY = "wc-command-history";
const MAX_HISTORY = 50;

function loadHistory(): string[] {
  try {
    const raw = localStorage.getItem(HISTORY_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (Array.isArray(parsed)) return parsed.filter((x): x is string => typeof x === "string").slice(-MAX_HISTORY);
  } catch {
    // corrupted — reset
  }
  return [];
}

function saveHistory(entries: string[]): void {
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify(entries.slice(-MAX_HISTORY)));
  } catch {
    // ignore
  }
}

/**
 * Ring-buffer command history persisted to localStorage.
 * Provides navigation (up/down) and push operations.
 */
export function useCommandHistory() {
  const [entries, setEntries] = useState(loadHistory);
  // -1 means "not browsing history" (current draft)
  const indexRef = useRef(-1);

  // Persist whenever entries change
  useEffect(() => {
    saveHistory(entries);
  }, [entries]);

  const push = useCallback((command: string) => {
    const trimmed = command.trim();
    if (!trimmed) return;
    setEntries((prev) => {
      // Deduplicate consecutive
      if (prev.length > 0 && prev[prev.length - 1] === trimmed) return prev;
      return [...prev, trimmed].slice(-MAX_HISTORY);
    });
    indexRef.current = -1;
  }, []);

  /** Navigate up (older). Returns the history entry or null if at the top. */
  const navigateUp = useCallback((): string | null => {
    const current = indexRef.current;
    const len = entries.length;
    if (len === 0) return null;
    // First press: go to most recent
    if (current === -1) {
      indexRef.current = len - 1;
      return entries[len - 1] ?? null;
    }
    // Already at oldest
    if (current <= 0) return entries[0] ?? null;
    indexRef.current = current - 1;
    return entries[current - 1] ?? null;
  }, [entries]);

  /** Navigate down (newer). Returns the history entry, or null to indicate "back to draft". */
  const navigateDown = useCallback((): string | null => {
    const current = indexRef.current;
    const len = entries.length;
    if (current === -1) return null; // already at draft
    if (current >= len - 1) {
      indexRef.current = -1;
      return null; // signal to restore draft
    }
    indexRef.current = current + 1;
    return entries[current + 1] ?? null;
  }, [entries]);

  const resetNavigation = useCallback(() => {
    indexRef.current = -1;
  }, []);

  return { entries, push, navigateUp, navigateDown, resetNavigation };
}
