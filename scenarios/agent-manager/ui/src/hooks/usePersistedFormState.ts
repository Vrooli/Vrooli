import { useCallback, useEffect, useRef, useState } from "react";

const STORAGE_VERSION = 1;

interface StorageEnvelope<T> {
  version: number;
  data: T;
  updatedAt: number;
}

/**
 * useState wrapper that persists state to localStorage with debounced writes.
 * Returns [state, setState, clearPersistedState].
 */
export function usePersistedFormState<T>(
  key: string,
  defaultValue: T,
  debounceMs = 500,
): [T, React.Dispatch<React.SetStateAction<T>>, () => void] {
  const [state, setState] = useState<T>(() => {
    try {
      const raw = localStorage.getItem(key);
      if (raw) {
        const envelope = JSON.parse(raw) as StorageEnvelope<T>;
        if (envelope.version === STORAGE_VERSION && envelope.data != null) {
          return envelope.data;
        }
      }
    } catch {
      // Corrupt data — fall through to default
    }
    return defaultValue;
  });

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounced write to localStorage
  useEffect(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      try {
        const envelope: StorageEnvelope<T> = {
          version: STORAGE_VERSION,
          data: state,
          updatedAt: Date.now(),
        };
        localStorage.setItem(key, JSON.stringify(envelope));
      } catch {
        // Storage full or unavailable — silently ignore
      }
    }, debounceMs);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [state, key, debounceMs]);

  const clearPersistedState = useCallback(() => {
    try {
      localStorage.removeItem(key);
    } catch {
      // Ignore
    }
  }, [key]);

  return [state, setState, clearPersistedState];
}
