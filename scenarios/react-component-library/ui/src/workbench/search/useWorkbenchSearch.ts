import { useEffect, useRef, useState } from "react";

import {
  searchWorkbench,
  type SearchResponse,
  type SearchResult,
  SearchKind,
} from "../../api/search";

export interface SearchState {
  response: SearchResponse | null;
  results: SearchResult[];
  loading: boolean;
  error: string | null;
}

const DEBOUNCE_MS = 140;

/**
 * One retrieval hook for Start and the command palette. The server owns
 * ranking and availability; this hook only owns request lifetime and keeps the
 * previous result set visible while a newer query is in flight.
 */
export function useWorkbenchSearch(
  query: string,
  options: { kinds?: SearchKind[]; limit?: number; enabled?: boolean } = {},
): SearchState {
  const { kinds, limit = 30, enabled = true } = options;
  const [state, setState] = useState<SearchState>({
    response: null,
    results: [],
    loading: enabled,
    error: null,
  });
  const request = useRef(0);

  useEffect(() => {
    if (!enabled) {
      setState((current) => ({ ...current, loading: false }));
      return;
    }
    const sequence = ++request.current;
    const timer = globalThis.setTimeout(() => {
      setState((current) => ({ ...current, loading: true, error: null }));
      searchWorkbench({ query, kinds, limit })
        .then((response) => {
          if (request.current !== sequence) return;
          setState({
            response,
            results: response.results,
            loading: false,
            error: null,
          });
        })
        .catch((error: unknown) => {
          if (request.current !== sequence) return;
          setState((current) => ({
            ...current,
            loading: false,
            error: error instanceof Error ? error.message : "Search failed",
          }));
        });
    }, DEBOUNCE_MS);
    return () => globalThis.clearTimeout(timer);
  }, [query, kinds, limit, enabled]);

  return state;
}
