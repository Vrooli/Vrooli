import { useCallback, useEffect, useRef } from "react";

const DRAFT_KEY_PREFIX = "wc-mobile-draft";
const DEBOUNCE_MS = 300;

function draftKey(sessionId?: string | null): string {
  return sessionId ? `${DRAFT_KEY_PREFIX}-${sessionId}` : DRAFT_KEY_PREFIX;
}

/**
 * Persists a text draft to localStorage so it survives page reloads and
 * accidental navigation, keyed per session.
 *
 * This is an IMPERATIVE hook — it deliberately holds no React state and never
 * triggers a re-render. The consumer owns the live value (an uncontrolled
 * textarea + a ref), so persisting must not round-trip through React on every
 * keystroke; that round-trip was the source of mobile typing lag. The hook
 * just reads the stored draft, writes it on a debounce, and clears it.
 */
export function useDraftPersistence(sessionId?: string | null) {
  const key = draftKey(sessionId);
  const keyRef = useRef(key);
  keyRef.current = key;
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  /** Read the persisted draft for the current session (empty string if none). */
  const readDraft = useCallback((): string => {
    try {
      return localStorage.getItem(keyRef.current) ?? "";
    } catch {
      return "";
    }
  }, []);

  /**
   * Persist `value` for the current session on a debounce. The storage key is
   * captured at call time so a session switch mid-debounce still writes the
   * draft it belongs to.
   */
  const persistDraft = useCallback((value: string) => {
    const k = keyRef.current;
    if (timerRef.current !== null) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      try {
        if (value) {
          localStorage.setItem(k, value);
        } else {
          localStorage.removeItem(k);
        }
      } catch {
        // Storage full or unavailable — ignore.
      }
    }, DEBOUNCE_MS);
  }, []);

  /** Clear the current session's draft immediately and cancel any pending write. */
  const clearDraft = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    try {
      localStorage.removeItem(keyRef.current);
    } catch {
      // ignore
    }
  }, []);

  // Flush nothing but cancel a dangling timer on unmount.
  useEffect(() => {
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current);
    };
  }, []);

  return { key, readDraft, persistDraft, clearDraft };
}
