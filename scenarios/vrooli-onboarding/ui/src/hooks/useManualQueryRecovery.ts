import { useQuery } from "@tanstack/react-query";

/** Queries with user-visible retry actions; failed reads never retry behind the user's back. */
export function useManualQueryRecovery<TData>(queryKey: readonly unknown[], queryFn: () => Promise<TData>) {
  return useQuery({
    queryKey,
    queryFn,
    retry: false,
    retryOnMount: false,
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
  });
}
