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
 *
 * Pending writes are tracked PER STORAGE KEY. A single shared timer used to be
 * enough when the draft was private to one toolbar, but the draft is now shared
 * across sessions: with one timer, typing (or sending) in session B cancelled
 * session A's still-pending write and A's draft was silently lost.
 */
export function useDraftPersistence(sessionId?: string | null) {
  const key = draftKey(sessionId);
  const keyRef = useRef(key);
  keyRef.current = key;
  const pendingRef = useRef(
    new Map<string, { timer: ReturnType<typeof setTimeout>; value: string }>(),
  );

  const writeNow = useCallback((k: string, value: string) => {
    try {
      if (value) {
        localStorage.setItem(k, value);
      } else {
        localStorage.removeItem(k);
      }
    } catch {
      // Storage full or unavailable — ignore.
    }
  }, []);

  const cancelPending = useCallback((k: string) => {
    const pending = pendingRef.current.get(k);
    if (!pending) return;
    clearTimeout(pending.timer);
    pendingRef.current.delete(k);
  }, []);

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
   * captured at call time, and the debounce is scoped to that key, so neither a
   * session switch nor activity in another session can drop this write.
   */
  const persistDraft = useCallback((value: string) => {
    const k = keyRef.current;
    cancelPending(k);
    const timer = setTimeout(() => {
      pendingRef.current.delete(k);
      writeNow(k, value);
    }, DEBOUNCE_MS);
    pendingRef.current.set(k, { timer, value });
  }, [cancelPending, writeNow]);

  /**
   * Write `session`'s draft immediately, cancelling any pending debounce for
   * it. Used when leaving a session so its draft is durable before the live
   * value is reloaded for the session being switched to.
   */
  const flushDraft = useCallback((session: string | null | undefined, value: string) => {
    const k = draftKey(session);
    cancelPending(k);
    writeNow(k, value);
  }, [cancelPending, writeNow]);

  /**
   * Clear a session's draft immediately and cancel its pending write. Defaults
   * to the current session; pass an explicit session for a late clear (e.g. a
   * send that settles after the user has switched away).
   */
  const clearDraft = useCallback((session?: string | null) => {
    const k = session === undefined ? keyRef.current : draftKey(session);
    cancelPending(k);
    try {
      localStorage.removeItem(k);
    } catch {
      // ignore
    }
  }, [cancelPending]);

  // Flush pending writes on unmount rather than dropping them — a draft typed
  // within the debounce window of a teardown would otherwise be lost.
  useEffect(() => {
    const pending = pendingRef.current;
    return () => {
      for (const [k, entry] of pending) {
        clearTimeout(entry.timer);
        writeNow(k, entry.value);
      }
      pending.clear();
    };
  }, [writeNow]);

  return { key, readDraft, persistDraft, flushDraft, clearDraft };
}
