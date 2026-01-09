/**
 * useSearch - Hook for full-text search across chats and messages.
 *
 * Features:
 * - Debounced search to avoid excessive API calls
 * - Loading and error states
 * - Results include chat info, matching message ID, and highlighted snippets
 *
 * SEAM: For testing, mock the searchChats API function.
 */
import { useState, useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { searchChats, type SearchResult } from "../lib/api";

// Stable empty array for default value
// CRITICAL: Using `= []` in destructuring creates a NEW array on every render,
// which changes references and triggers infinite re-render loops via useMemo dependencies
const EMPTY_RESULTS: SearchResult[] = [];

interface UseSearchOptions {
  /** Debounce delay in ms (default: 300) */
  debounceMs?: number;
  /** Max results to return (default: 20) */
  limit?: number;
  /** Minimum query length before searching (default: 2) */
  minLength?: number;
}

interface UseSearchReturn {
  /** Current search query */
  query: string;
  /** Update the search query */
  setQuery: (query: string) => void;
  /** Search results (empty if query too short) */
  results: SearchResult[];
  /** Whether a search is in progress */
  isSearching: boolean;
  /** Search error if any */
  error: Error | null;
  /** Whether search mode is active (query length >= minLength) */
  isActive: boolean;
  /** Clear the search */
  clear: () => void;
}

export function useSearch(options: UseSearchOptions = {}): UseSearchReturn {
  const { debounceMs = 300, limit = 20, minLength = 2 } = options;

  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");

  // Debounce the query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, debounceMs);

    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  // Determine if search is active
  const isActive = query.length >= minLength;
  const shouldSearch = debouncedQuery.length >= minLength;

  // Fetch search results
  // NOTE: Use stable EMPTY_RESULTS constant instead of `= []`
  const {
    data: resultsData,
    isLoading: isSearching,
    error,
  } = useQuery({
    queryKey: ["search", debouncedQuery, limit],
    queryFn: () => searchChats(debouncedQuery, limit),
    enabled: shouldSearch,
    staleTime: 30000, // Cache results for 30s
  });
  const results = resultsData ?? EMPTY_RESULTS;

  // Clear function
  const clear = useMemo(
    () => () => {
      setQuery("");
      setDebouncedQuery("");
    },
    []
  );

  return {
    query,
    setQuery,
    results: shouldSearch ? results : EMPTY_RESULTS,
    isSearching: isActive && (isSearching || query !== debouncedQuery),
    error: error as Error | null,
    isActive,
    clear,
  };
}
