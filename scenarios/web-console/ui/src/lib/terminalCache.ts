// DOC: docs/internal/SEAMS.md#terminal-cache-storage-seam
/**
 * Terminal state cache for instant restore on page refresh.
 *
 * Serialized xterm.js state is stored in sessionStorage (per-tab, survives
 * refresh, auto-cleared on tab close). On reconnect the client sends the
 * cached byte offset to the server, which responds with only the delta.
 *
 * [REQ:P0-003b] Reconnect State Restoration
 */

/** Shape of a cached terminal state entry. */
export interface TerminalCacheEntry {
  /** xterm SerializeAddon output (escape-sequence string). */
  serialized: string;
  /** Server's totalOutputBytes at the time of last history_end. */
  totalBytes: number;
  /** Date.now() when the cache was saved, for age-based expiry. */
  savedAt: number;
}

const STORAGE_KEY_PREFIX = "wc-terminal-cache-";

/** Maximum age before a cache entry is considered stale. */
export const CACHE_MAX_AGE_MS = 30 * 60 * 1000; // 30 minutes

/** Maximum serialized string length to store (prevents sessionStorage bloat). */
export const CACHE_MAX_SIZE = 2 * 1024 * 1024; // 2 MB

function storageKey(sessionId: string): string {
  return `${STORAGE_KEY_PREFIX}${sessionId}`;
}

/**
 * Saves a terminal cache entry to sessionStorage.
 * Returns false if the entry is too large or storage is full.
 */
export function saveTerminalCache(
  sessionId: string,
  entry: TerminalCacheEntry,
): boolean {
  if (entry.serialized.length > CACHE_MAX_SIZE) {
    return false;
  }
  try {
    sessionStorage.setItem(storageKey(sessionId), JSON.stringify(entry));
    return true;
  } catch {
    // sessionStorage quota exceeded or unavailable
    return false;
  }
}

/**
 * Loads a terminal cache entry from sessionStorage.
 * Returns null if the entry is missing, expired, or corrupt.
 */
export function loadTerminalCache(
  sessionId: string,
): TerminalCacheEntry | null {
  try {
    const raw = sessionStorage.getItem(storageKey(sessionId));
    if (!raw) return null;
    const entry = JSON.parse(raw) as TerminalCacheEntry;
    if (
      typeof entry.serialized !== "string" ||
      typeof entry.totalBytes !== "number" ||
      typeof entry.savedAt !== "number"
    ) {
      return null;
    }
    if (Date.now() - entry.savedAt > CACHE_MAX_AGE_MS) {
      sessionStorage.removeItem(storageKey(sessionId));
      return null;
    }
    return entry;
  } catch {
    // Corrupt JSON or sessionStorage unavailable
    return null;
  }
}

/** Removes a terminal cache entry from sessionStorage. */
export function clearTerminalCache(sessionId: string): void {
  try {
    sessionStorage.removeItem(storageKey(sessionId));
  } catch {
    // sessionStorage unavailable — nothing to do
  }
}
