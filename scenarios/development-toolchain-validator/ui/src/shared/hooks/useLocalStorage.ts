import { useCallback, useEffect, useState } from "react";

/**
 * Small `localStorage`-backed state hook.
 *
 * Reads the initial value lazily from storage; writes back on every set.
 * SSR-safe (returns `defaultValue` and a noop setter when `window` is
 * missing). Catches storage exceptions (quota, private mode) so a write
 * failure doesn't crash the UI.
 */
export function useLocalStorage<T>(
  key: string,
  defaultValue: T,
): [T, (next: T | ((prev: T) => T)) => void] {
  const [value, setValue] = useState<T>(() => {
    if (typeof window === "undefined") return defaultValue;
    try {
      const raw = window.localStorage.getItem(key);
      if (raw === null) return defaultValue;
      return JSON.parse(raw) as T;
    } catch {
      return defaultValue;
    }
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    try {
      window.localStorage.setItem(key, JSON.stringify(value));
    } catch {
      // ignored: quota exceeded or private browsing
    }
  }, [key, value]);

  const set = useCallback(
    (next: T | ((prev: T) => T)) => {
      setValue((prev) =>
        typeof next === "function" ? (next as (prev: T) => T)(prev) : next,
      );
    },
    [],
  );

  return [value, set];
}
