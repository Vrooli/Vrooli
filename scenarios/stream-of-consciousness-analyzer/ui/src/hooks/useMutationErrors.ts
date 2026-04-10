import { useCallback } from "react";
import type { UseMutationResult } from "@tanstack/react-query";

/**
 * Consolidates error state and reset logic across multiple mutations.
 *
 * Returns the first active error (if any) and a callback to reset all
 * mutations at once—eliminating the repeated reset-all pattern that
 * was duplicated across SchemeList, CanvasView, and GraphView.
 */
export function useMutationErrors(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mutations: UseMutationResult<any, Error, any, any>[],
) {
  const activeError = mutations.find((m) => m.error)?.error ?? null;

  const resetAll = useCallback(() => {
    for (const m of mutations) m.reset();
  }, [mutations]);

  return { activeError, resetAll } as const;
}
