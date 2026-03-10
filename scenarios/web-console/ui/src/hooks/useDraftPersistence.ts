import { useState, useCallback, useEffect, useRef } from "react";

const DRAFT_KEY = "wc-mobile-draft";
const DEBOUNCE_MS = 300;

/**
 * Persists a text draft to localStorage so it survives page reloads
 * and accidental navigation. The draft is saved on a debounce timer
 * and cleared explicitly when the caller confirms a successful send.
 */
export function useDraftPersistence() {
  const [value, setValue] = useState(() => {
    try {
      return localStorage.getItem(DRAFT_KEY) ?? "";
    } catch {
      return "";
    }
  });

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Persist to localStorage on change (debounced)
  useEffect(() => {
    if (timerRef.current !== null) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      try {
        if (value) {
          localStorage.setItem(DRAFT_KEY, value);
        } else {
          localStorage.removeItem(DRAFT_KEY);
        }
      } catch {
        // Storage full or unavailable — ignore
      }
    }, DEBOUNCE_MS);
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current);
    };
  }, [value]);

  const clearDraft = useCallback(() => {
    setValue("");
    try {
      localStorage.removeItem(DRAFT_KEY);
    } catch {
      // ignore
    }
  }, []);

  return { value, setValue, clearDraft };
}
