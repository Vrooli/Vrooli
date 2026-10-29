// Responsibility: the one read/write hook shape every API hook is built on.
// Reads are TanStack queries (30 s deadline via apiRequest); writes are mutations.
import { useCallback } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { queryClient } from "../providers/QueryProvider";

export interface ApiQueryOptions<T> {
  enabled?: boolean;
  /** Poll interval in ms; omit for fetch-on-mount only. */
  refetchInterval?: number;
  /** Shown while there is no answer yet (an empty list rather than null). */
  fallback?: T;
  /** How long an answer counts as fresh before a remount reads again (ms); default 0. */
  staleTime?: number;
  /** `loading` is true only until the first answer, not on every background refresh. */
  firstLoadOnly?: boolean;
}

export interface ApiQueryResult<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

function errorText(error: unknown): string | null {
  if (error === null || error === undefined) return null;
  return error instanceof Error ? error.message : String(error);
}

export function useApiQuery<T>(
  key: readonly unknown[],
  fetcher: (signal: AbortSignal) => Promise<T>,
  options: ApiQueryOptions<T> = {},
): ApiQueryResult<T> {
  const query = useQuery<T, Error>({
    queryKey: key,
    queryFn: ({ signal }) => fetcher(signal),
    enabled: options.enabled ?? true,
    refetchInterval: options.refetchInterval,
    retry: false,
    staleTime: options.staleTime ?? 0,
    refetchOnWindowFocus: false,
  }, queryClient);
  const { refetch: refetchQuery } = query;
  // `enabled` gates the automatic fetch only; an explicit refetch always reads.
  const refetch = useCallback(async () => { await refetchQuery(); }, [refetchQuery]);
  return {
    data: query.data ?? options.fallback ?? null,
    loading: options.firstLoadOnly ? query.isLoading : query.isFetching,
    error: errorText(query.error),
    refetch,
  };
}

/** A write with loading/error state; `run` rejects on failure so callers can react. */
export function useApiMutation<Args extends unknown[], Result>(write: (...args: Args) => Promise<Result>) {
  const mutation = useMutation<Result, Error, Args>({ mutationFn: (args) => write(...args) }, queryClient);
  const { mutateAsync } = mutation;
  const run = useCallback((...args: Args) => mutateAsync(args), [mutateAsync]);
  return { run, loading: mutation.isPending, error: errorText(mutation.error) };
}
