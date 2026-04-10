/**
 * React Query Utilities
 *
 * This module provides shared query configuration utilities for consistent
 * data fetching behavior across the application.
 *
 * Design principles:
 * - Centralize retry/cache configuration to prevent drift
 * - Use config values from the control surface for tunability
 * - Keep query utilities pure and testable
 *
 * DOC: docs/internal/UTILS_UNIFICATION_NOTES.md
 */

import { dataFetchingConfig } from "../config";

/**
 * Default query options for React Query.
 *
 * These options provide consistent retry behavior, caching, and refetch
 * policies across all data fetching queries in the application.
 *
 * All values are sourced from `dataFetchingConfig` for centralized control.
 *
 * @example
 * ```tsx
 * const { data } = useQuery({
 *   queryKey: ["backlog"],
 *   queryFn: () => backlogService.list(),
 *   ...defaultQueryOptions,
 * });
 * ```
 */
export const defaultQueryOptions = {
  /**
   * Number of retry attempts before showing an error.
   */
  retry: dataFetchingConfig.retryCount,

  /**
   * Exponential backoff for retries.
   * Delay = retryDelayMs * 2^attemptIndex
   */
  retryDelay: (attemptIndex: number) =>
    dataFetchingConfig.retryDelayMs * Math.pow(2, attemptIndex),

  /**
   * Time before data is considered stale.
   */
  staleTime: dataFetchingConfig.staleTimeMs,

  /**
   * Time to keep unused data in garbage collection.
   */
  gcTime: dataFetchingConfig.cacheTimeMs,

  /**
   * Whether to refetch when window regains focus.
   */
  refetchOnWindowFocus: dataFetchingConfig.refetchOnWindowFocus,
} as const;

/**
 * Type for the default query options.
 * Useful for extending or overriding specific options.
 */
export type DefaultQueryOptions = typeof defaultQueryOptions;

/**
 * Builds a URL query string from a params record, omitting undefined/null/empty values.
 * Array values are joined with commas.
 *
 * @returns A query string starting with "?" if any params are present, or empty string if none.
 *
 * @example
 * ```ts
 * buildQueryString({ status: "active", mode: undefined })
 * // => "?status=active"
 *
 * buildQueryString({ kinds: ["idea", "fix"] })
 * // => "?kinds=idea%2Cfix"
 *
 * buildQueryString({})
 * // => ""
 * ```
 */
export function buildQueryString(
  params: Record<string, string | string[] | number | boolean | undefined | null>,
): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") continue;
    if (Array.isArray(value)) {
      if (value.length > 0) {
        query.set(key, value.join(","));
      }
    } else {
      query.set(key, String(value));
    }
  }
  const encoded = query.toString();
  return encoded ? `?${encoded}` : "";
}
