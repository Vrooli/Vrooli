import { useSyncExternalStore } from "react";

/** Browser-local preferences (theme, locale) with a shared "storage was denied" signal. */
export type LocalPreferenceKind = "theme" | "locale";

const failedPreferenceWrites = new Set<LocalPreferenceKind>();
const preferenceStorageListeners = new Set<() => void>();

const notifyPreferenceStorageChange = () => {
  for (const listener of preferenceStorageListeners) listener();
};

const recordPreferenceStorageResult = (kind: LocalPreferenceKind, succeeded: boolean) => {
  const changed = succeeded ? failedPreferenceWrites.delete(kind) : !failedPreferenceWrites.has(kind);
  if (!succeeded) failedPreferenceWrites.add(kind);
  if (changed) notifyPreferenceStorageChange();
};

/** Read one browser-local preference without allowing storage denial to escape. */
export const readLocalPreference = (kind: LocalPreferenceKind, key: string): string | null => {
  if (typeof window === "undefined") return null;
  try {
    const value = window.localStorage.getItem(key);
    recordPreferenceStorageResult(kind, true);
    return value;
  } catch {
    recordPreferenceStorageResult(kind, false);
    return null;
  }
};

/** Save one browser-local preference; the caller keeps its in-memory choice either way. */
export const writeLocalPreference = (kind: LocalPreferenceKind, key: string, value: string): boolean => {
  if (typeof window === "undefined") return false;
  try {
    window.localStorage.setItem(key, value);
    recordPreferenceStorageResult(kind, true);
    return true;
  } catch {
    recordPreferenceStorageResult(kind, false);
    return false;
  }
};

const subscribePreferenceStorage = (listener: () => void) => {
  preferenceStorageListeners.add(listener);
  return () => preferenceStorageListeners.delete(listener);
};

const getPreferenceStorageSnapshot = () => failedPreferenceWrites.size > 0;

export const usePreferenceStorageUnavailable = () => useSyncExternalStore(
  subscribePreferenceStorage,
  getPreferenceStorageSnapshot,
  () => false,
);
