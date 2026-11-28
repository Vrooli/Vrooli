import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import {
  filterHits,
  searchSurfaces,
  SURFACE_KIND_FILTERS,
  type SearchHit,
  type SearchResults,
  type SurfaceKindFilter,
} from "../../api/search";

const DEBOUNCE_MS = 250;
const MIN_QUERY_LEN = 2;

const QUERY_PARAM = "q";
const KIND_PARAM = "kind";

function isSurfaceKindFilter(value: string | null): value is SurfaceKindFilter {
  return value !== null && (SURFACE_KIND_FILTERS as readonly string[]).includes(value);
}

export function useDebounced<T>(value: T, ms = DEBOUNCE_MS): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return debounced;
}

export type UseSearchReturn = {
  /** Live input value, including characters typed before the debounce fires. */
  query: string;
  setQuery: (q: string) => void;
  /** Debounced query actually sent to the API. */
  effectiveQuery: string;
  kind: SurfaceKindFilter;
  setKind: (k: SurfaceKindFilter) => void;
  clear: () => void;
  /** Hits filtered by `kind`. */
  filteredHits: SearchHit[];
  /** Hit count by kind for the current result set (always uses the unfiltered hits). */
  countByKind: Record<SurfaceKindFilter, number>;
  isShortQuery: boolean;
  query_: UseQueryResult<SearchResults>;
};

export function searchQueryKey(query: string): readonly unknown[] {
  return ["search", "results", query] as const;
}

export function useSearch(): UseSearchReturn {
  const [params, setParams] = useSearchParams();

  const initialQuery = params.get(QUERY_PARAM) ?? "";
  const initialKindRaw = params.get(KIND_PARAM);
  const initialKind: SurfaceKindFilter = isSurfaceKindFilter(initialKindRaw) ? initialKindRaw : "all";

  const [query, setQueryState] = useState<string>(initialQuery);
  const [kind, setKindState] = useState<SurfaceKindFilter>(initialKind);

  const debounced = useDebounced(query.trim());
  const isShortQuery = debounced.length > 0 && debounced.length < MIN_QUERY_LEN;
  const effectiveQuery = isShortQuery ? "" : debounced;

  // Mirror state to URL (replaceState — typing shouldn't pollute history).
  useEffect(() => {
    const next = new URLSearchParams(params);
    if (effectiveQuery.length > 0) next.set(QUERY_PARAM, effectiveQuery);
    else next.delete(QUERY_PARAM);
    if (kind !== "all") next.set(KIND_PARAM, kind);
    else next.delete(KIND_PARAM);
    if (next.toString() !== params.toString()) {
      setParams(next, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setParams is stable; params re-included via .toString() comparison
  }, [effectiveQuery, kind]);

  const query_ = useQuery({
    queryKey: searchQueryKey(effectiveQuery),
    queryFn: () => searchSurfaces(effectiveQuery),
    enabled: effectiveQuery.length >= MIN_QUERY_LEN,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  });

  const hits = useMemo(() => query_.data?.hits ?? [], [query_.data]);
  const filteredHits = useMemo(() => filterHits(hits, kind), [hits, kind]);

  const countByKind = useMemo<Record<SurfaceKindFilter, number>>(() => {
    const initial: Record<SurfaceKindFilter, number> = {
      all: hits.length,
      component: 0,
      page: 0,
      feature: 0,
      hook: 0,
      layout: 0,
      other: 0,
    };
    for (const hit of hits) {
      if (hit.kind === "unspecified") continue;
      initial[hit.kind] += 1;
    }
    return initial;
  }, [hits]);

  const setQuery = useCallback((next: string) => setQueryState(next), []);
  const setKind = useCallback((next: SurfaceKindFilter) => setKindState(next), []);
  const clear = useCallback(() => {
    setQueryState("");
    setKindState("all");
  }, []);

  return {
    query,
    setQuery,
    effectiveQuery,
    kind,
    setKind,
    clear,
    filteredHits,
    countByKind,
    isShortQuery,
    query_,
  };
}
