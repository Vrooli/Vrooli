import { useQuery } from "@tanstack/react-query";

import { consoleApi, consoleKeys } from "../api/console";

/**
 * Shell-level signal: how many decisions are waiting on the operator, and
 * whether the API is reachable at all. Shared by the sidebar, top bar and
 * bottom nav so they read from one query.
 */
export function useAttention(): { pending: number; apiOk: boolean | undefined } {
  const query = useQuery({
    queryKey: consoleKeys.overview,
    queryFn: ({ signal }) => consoleApi.overview(signal),
    refetchInterval: 30_000,
    staleTime: 10_000,
    retry: 1,
  });
  const pending = query.data?.gates.filter((gate) => gate.status === "pending").length ?? 0;
  const apiOk = query.isError ? false : query.data ? true : undefined;
  return { pending, apiOk };
}
