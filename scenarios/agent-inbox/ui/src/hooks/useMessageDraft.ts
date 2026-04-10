/**
 * useMessageDraft - Hook for persisting message drafts to localStorage.
 *
 * Provides per-page draft persistence with debounced writes.
 * Drafts are keyed by chatId (or "home" for the home page).
 */

import { useState, useEffect, useCallback, useRef } from "react";

const STORAGE_PREFIX = "agent-inbox:draft:";

interface UseMessageDraftOptions {
  /** Key suffix for localStorage — use chatId or "home" */
  pageKey: string;
  /** Debounce interval for writes (default 300ms) */
  debounceMs?: number;
}

interface UseMessageDraftResult {
  /** Current draft value */
  draft: string;
  /** Update the draft (debounced write to localStorage) */
  setDraft: (value: string) => void;
  /** Clear the draft from state and localStorage */
  clearDraft: () => void;
}

export function useMessageDraft({
  pageKey,
  debounceMs = 300,
}: UseMessageDraftOptions): UseMessageDraftResult {
  const storageKey = STORAGE_PREFIX + pageKey;
  const timerRef = useRef<ReturnType<typeof setTimeout>>();

  // Initialize from localStorage
  const [draft, setDraftState] = useState(() => {
    if (typeof window === "undefined") return "";
    return localStorage.getItem(storageKey) ?? "";
  });

  // When pageKey changes, load the draft for the new page
  useEffect(() => {
    const stored = localStorage.getItem(storageKey) ?? "";
    setDraftState(stored);
  }, [storageKey]);

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const setDraft = useCallback(
    (value: string) => {
      setDraftState(value);

      // Debounced write to localStorage
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
      timerRef.current = setTimeout(() => {
        if (typeof window !== "undefined") {
          if (value) {
            localStorage.setItem(storageKey, value);
          } else {
            localStorage.removeItem(storageKey);
          }
        }
      }, debounceMs);
    },
    [storageKey, debounceMs]
  );

  const clearDraft = useCallback(() => {
    setDraftState("");
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }
    if (typeof window !== "undefined") {
      localStorage.removeItem(storageKey);
    }
  }, [storageKey]);

  return { draft, setDraft, clearDraft };
}
