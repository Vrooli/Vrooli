/**
 * useSidebarSearch - Debounced search filtering for sidebar content.
 *
 * Returns a debounced query string that updates after searchDebounceMs.
 */

import { useEffect, useState } from "react";
import { uiBehaviorConfig } from "../../../../config";

export function useDebouncedValue(value: string, delayMs = uiBehaviorConfig.searchDebounceMs): string {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs);
    return () => window.clearTimeout(timer);
  }, [value, delayMs]);

  return debounced;
}

/** Case-insensitive substring match against one or more fields. */
export function matchesSearch(query: string, ...fields: (string | undefined | null)[]): boolean {
  if (!query) return true;
  const lower = query.toLowerCase();
  return fields.some((f) => f?.toLowerCase().includes(lower));
}
