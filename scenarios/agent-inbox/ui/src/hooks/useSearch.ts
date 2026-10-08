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
import { searchChats, type SearchResult, type ContentSearchOptions } from "../lib/api";

/** Quick = client-side chat name filter (instant), Content = server-side regex search */
export type ChatSearchMode = "quick" | "content";

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
  /** Search mode: "quick" for client-side name filter, "content" for server-side search (default: "quick") */
  mode?: ChatSearchMode;
  /** Max message matches per chat in content mode (1–10, default: 1) */
  perChat?: number;
  /** Content search options */
  caseSensitive?: boolean;
  wholeWord?: boolean;
  regex?: boolean;
}

interface UseSearchReturn {
  /** Current search query */
  query: string;
  /** Update the search query */
  setQuery: (query: string) => void;
  /** Search results (empty if query too short) - only populated in content mode */
  results: SearchResult[];
  /** Whether a search is in progress */
  isSearching: boolean;
  /** Search error if any */
  error: Error | null;
  /** Whether search mode is active (query length >= minLength) */
  isActive: boolean;
  /** Clear the search */
  clear: () => void;
  /** Current search mode */
  mode: ChatSearchMode;
}

export function useSearch(options: UseSearchOptions = {}): UseSearchReturn {
  const {
    debounceMs = 300,
    limit = 20,
    minLength: minLengthOption = 2,
    mode = "quick",
    perChat = 1,
    caseSensitive = false,
    wholeWord = false,
    regex = false,
  } = options;

  // Quick mode benefits from single-char matching since it's instant client-side filtering
  const minLength = mode === "quick" ? 1 : minLengthOption;

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

  // Build content search options
  const searchOptions: ContentSearchOptions | undefined = useMemo(() => {
    if (!caseSensitive && !wholeWord && !regex) return undefined;
    return { caseSensitive, wholeWord, regex };
  }, [caseSensitive, wholeWord, regex]);

  // Fetch search results - only in content mode
  // NOTE: Use stable EMPTY_RESULTS constant instead of `= []`
  const {
    data: resultsData,
    isLoading: isContentSearching,
    error,
  } = useQuery({
    queryKey: ["search", debouncedQuery, limit, perChat, caseSensitive, wholeWord, regex],
    queryFn: () => searchChats(debouncedQuery, limit, perChat, searchOptions),
    enabled: shouldSearch && mode === "content",
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
    results: shouldSearch && mode === "content" ? results : EMPTY_RESULTS,
    isSearching: mode === "content" && isActive && (isContentSearching || query !== debouncedQuery),
    error,
    isActive,
    clear,
    mode,
  };
}
