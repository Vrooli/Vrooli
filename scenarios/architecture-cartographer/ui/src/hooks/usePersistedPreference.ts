import * as React from "react";

/**
 * usePersistedPreference — typed, schema-validated localStorage seam.
 *
 * Substitutable for tests: the second argument can be a `storage` adapter
 * implementing `getItem`/`setItem`. Production code calls without it and
 * gets `window.localStorage`. Tests inject an in-memory adapter and never
 * touch real DOM storage.
 */
export interface PreferenceStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

const ssrSafeDefaultStorage: PreferenceStorage = {
  getItem: () => null,
  setItem: () => {},
};

const getDefaultStorage = (): PreferenceStorage => {
  if (typeof window === "undefined") return ssrSafeDefaultStorage;
  try {
    return window.localStorage;
  } catch {
    return ssrSafeDefaultStorage;
  }
};

export interface UsePersistedPreferenceOptions<T> {
  key: string;
  defaultValue: T;
  /** Validate parsed JSON; return `null` to fall back to `defaultValue`. */
  validate: (raw: unknown) => T | null;
  storage?: PreferenceStorage;
}

export function usePersistedPreference<T>({
  key,
  defaultValue,
  validate,
  storage,
}: UsePersistedPreferenceOptions<T>): [T, (next: T) => void] {
  const storageRef = React.useRef<PreferenceStorage>(storage ?? getDefaultStorage());

  const [value, setValue] = React.useState<T>(() => {
    const raw = storageRef.current.getItem(key);
    if (raw === null) return defaultValue;
    try {
      const parsed: unknown = JSON.parse(raw);
      const validated = validate(parsed);
      return validated ?? defaultValue;
    } catch {
      return defaultValue;
    }
  });

  const update = React.useCallback(
    (next: T) => {
      setValue(next);
      try {
        storageRef.current.setItem(key, JSON.stringify(next));
      } catch {
        // Storage may throw in private-browsing modes; surface elsewhere.
      }
    },
    [key],
  );

  return [value, update];
}
