import { useCallback } from "react";
import { useSearchParams } from "react-router-dom";

/**
 * Bidirectional sync between a URL query parameter and React state.
 *
 * - Reads the param on mount; falls back to `defaultValue` if absent or invalid.
 * - Setter updates the URL param with `replace: true` (no history pollution).
 * - Removes the param when the value equals `defaultValue` (clean URLs).
 * - Uses functional updates on `setSearchParams` so multiple instances coexist.
 */
export function useUrlState<T extends string>(
  key: string,
  defaultValue: T,
  options?: { validate?: (value: string) => value is T },
): [T, (value: T) => void] {
  const [searchParams, setSearchParams] = useSearchParams();

  const raw = searchParams.get(key);
  let value: T = defaultValue;
  if (raw !== null) {
    if (options?.validate) {
      value = options.validate(raw) ? raw : defaultValue;
    } else {
      value = raw as T;
    }
  }

  const setValue = useCallback(
    (next: T) => {
      setSearchParams(
        (prev) => {
          const updated = new URLSearchParams(prev);
          if (next === defaultValue) {
            updated.delete(key);
          } else {
            updated.set(key, next);
          }
          return updated;
        },
        { replace: true },
      );
    },
    [key, defaultValue, setSearchParams],
  );

  return [value, setValue];
}
