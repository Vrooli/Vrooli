import { dataFetchingConfig } from "../config";

export type LoadStatus = "idle" | "loading" | "success" | "error";

export interface FetchGuard {
  lastFetchedAt: number | null;
  hasData: boolean;
  force?: boolean;
}

export function shouldRefetch({ lastFetchedAt, hasData, force }: FetchGuard): boolean {
  if (force) return true;
  if (!lastFetchedAt) return true;
  if (!hasData) return true;
  return Date.now() - lastFetchedAt > dataFetchingConfig.staleTimeMs;
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export async function fetchWithRetry<T>(operation: () => Promise<T>): Promise<T> {
  const maxRetries = Math.max(0, dataFetchingConfig.retryCount);
  let attempt = 0;

  while (true) {
    try {
      return await operation();
    } catch (error) {
      if (attempt >= maxRetries) {
        throw error;
      }
      const delayMs = dataFetchingConfig.retryDelayMs * Math.pow(2, attempt);
      attempt += 1;
      if (delayMs > 0) {
        await sleep(delayMs);
      }
    }
  }
}

// ============================================================================
// localStorage Persistence Helpers
// ============================================================================

export interface StorePersistConfig {
  key: string;
  version: number;
  maxItems?: number;
}

interface PersistedEnvelope<T> {
  data: T;
  fetchedAt: number;
  version: number;
}

export interface HydratedState<T> {
  data: T;
  lastFetchedAt: number | null;
}

/**
 * Load persisted data from localStorage.
 * Returns fallback when: storage unavailable, version mismatch, data expired, or any error.
 */
export function loadFromStorage<T>(config: StorePersistConfig, fallback: T): HydratedState<T> {
  const empty: HydratedState<T> = { data: fallback, lastFetchedAt: null };
  if (typeof window === "undefined") return empty;
  try {
    const raw = window.localStorage.getItem(config.key);
    if (!raw) return empty;
    const envelope = JSON.parse(raw) as PersistedEnvelope<T>;
    if (!envelope || typeof envelope !== "object") return empty;
    if (envelope.version !== config.version) return empty;
    if (typeof envelope.fetchedAt !== "number") return empty;
    if (Date.now() - envelope.fetchedAt > dataFetchingConfig.cacheTimeMs) return empty;
    let data = envelope.data;
    if (Array.isArray(data) && config.maxItems && data.length > config.maxItems) {
      data = data.slice(0, config.maxItems) as T;
    }
    return { data, lastFetchedAt: envelope.fetchedAt };
  } catch {
    return empty;
  }
}

/**
 * Save data to localStorage. Silently no-ops on quota errors or unavailable storage.
 */
export function saveToStorage<T>(config: StorePersistConfig, data: T, lastFetchedAt: number): void {
  if (typeof window === "undefined") return;
  try {
    let toSave = data;
    if (Array.isArray(toSave) && config.maxItems && toSave.length > config.maxItems) {
      toSave = toSave.slice(0, config.maxItems) as T;
    }
    const envelope: PersistedEnvelope<T> = { data: toSave, fetchedAt: lastFetchedAt, version: config.version };
    window.localStorage.setItem(config.key, JSON.stringify(envelope));
  } catch {
    // Silent failure — localStorage unavailable or quota exceeded.
  }
}

/**
 * Remove persisted data from localStorage.
 */
export function clearStorage(key: string): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.removeItem(key);
  } catch {
    // Silent failure.
  }
}
