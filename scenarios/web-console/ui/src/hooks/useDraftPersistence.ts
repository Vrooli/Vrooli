import { useState, useCallback, useEffect, useRef } from "react";

const DRAFT_KEY_PREFIX = "wc-mobile-draft";
const DEBOUNCE_MS = 300;

function draftKey(sessionId?: string | null): string {
  return sessionId ? `${DRAFT_KEY_PREFIX}-${sessionId}` : DRAFT_KEY_PREFIX;
}

/**
 * Persists a text draft to localStorage so it survives page reloads
 * and accidental navigation. The draft is saved on a debounce timer
 * and cleared explicitly when the caller confirms a successful send.
 *
 * When `sessionId` is provided, each session gets its own draft key.
 */
export function useDraftPersistence(sessionId?: string | null) {
  const key = draftKey(sessionId);

  const [value, setValue] = useState(() => {
    try {
      return localStorage.getItem(key) ?? "";
    } catch {
      return "";
    }
  });

  // Re-read from localStorage when the session (key) changes
  const prevKeyRef = useRef(key);
  useEffect(() => {
    if (key === prevKeyRef.current) return;
    prevKeyRef.current = key;
    try {
      setValue(localStorage.getItem(key) ?? "");
    } catch {
      setValue("");
    }
  }, [key]);

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Persist to localStorage on change (debounced)
  useEffect(() => {
    if (timerRef.current !== null) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      try {
        if (value) {
          localStorage.setItem(key, value);
        } else {
          localStorage.removeItem(key);
        }
      } catch {
        // Storage full or unavailable — ignore
      }
    }, DEBOUNCE_MS);
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current);
    };
  }, [value, key]);

  const clearDraft = useCallback(() => {
    setValue("");
    try {
      localStorage.removeItem(key);
    } catch {
      // ignore
    }
  }, [key]);

  return { value, setValue, clearDraft };
}
