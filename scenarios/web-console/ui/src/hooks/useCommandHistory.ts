import { useCallback, useRef, useSyncExternalStore } from "react";

const HISTORY_KEY = "wc-command-history";
const MAX_HISTORY = 50;

/** One send kept on this device: what was sent, and when (epoch ms). */
export interface SentHistoryEntry {
  text: string;
  at: number;
}

function isEntry(value: unknown): value is SentHistoryEntry {
  if (typeof value !== "object" || value === null) return false;
  const candidate = value as Record<string, unknown>;
  return typeof candidate.text === "string" && typeof candidate.at === "number";
}

function readRaw(): string | null {
  try {
    return localStorage.getItem(HISTORY_KEY);
  } catch {
    return null;
  }
}

function parseHistory(raw: string | null): SentHistoryEntry[] {
  if (!raw) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    if (Array.isArray(parsed)) return parsed.filter(isEntry).slice(-MAX_HISTORY);
  } catch {
    // corrupted — reset
  }
  return [];
}

// localStorage is the one store: every mounted control (the toolbar, the
// expanded composer) reads it, so a send from one shows in the other. The
// parsed list is memoised on the raw string so the snapshot stays stable.
const listeners = new Set<() => void>();
let snapshotRaw: string | null = null;
let snapshot: SentHistoryEntry[] = [];

function getSnapshot(): SentHistoryEntry[] {
  const raw = readRaw();
  if (raw !== snapshotRaw) {
    snapshotRaw = raw;
    snapshot = parseHistory(raw);
  }
  return snapshot;
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  // Another tab on this device writing its history.
  const onStorage = (event: StorageEvent) => { if (event.key === HISTORY_KEY) listener(); };
  window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", onStorage);
  };
}

function writeHistory(update: (prev: SentHistoryEntry[]) => SentHistoryEntry[]): void {
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify(update(getSnapshot()).slice(-MAX_HISTORY)));
  } catch {
    // ignore
  }
  listeners.forEach((listener) => { listener(); });
}

/**
 * Ring-buffer history of sends, persisted to this device's localStorage only
 * (never synced) and shared by every control that shows it. Provides
 * navigation (up/down), push, and clear.
 */
export function useCommandHistory() {
  const entries = useSyncExternalStore(subscribe, getSnapshot);
  // -1 means "not browsing history" (current draft)
  const indexRef = useRef(-1);

  const push = useCallback((command: string) => {
    const trimmed = command.trim();
    if (!trimmed) return;
    const entry: SentHistoryEntry = { text: trimmed, at: Date.now() };
    writeHistory((prev) => {
      // Deduplicate consecutive; the repeat refreshes when it was sent.
      if (prev.length > 0 && prev[prev.length - 1]?.text === trimmed) return [...prev.slice(0, -1), entry];
      return [...prev, entry];
    });
    indexRef.current = -1;
  }, []);

  /** Removes every entry from this device. */
  const clear = useCallback(() => {
    writeHistory(() => []);
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
      return entries[len - 1]?.text ?? null;
    }
    // Already at oldest
    if (current <= 0) return entries[0]?.text ?? null;
    indexRef.current = current - 1;
    return entries[current - 1]?.text ?? null;
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
    return entries[current + 1]?.text ?? null;
  }, [entries]);

  const resetNavigation = useCallback(() => {
    indexRef.current = -1;
  }, []);

  return { entries, push, clear, navigateUp, navigateDown, resetNavigation };
}
