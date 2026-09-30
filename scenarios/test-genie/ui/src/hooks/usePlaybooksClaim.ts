import { useQuery } from "@tanstack/react-query";
import { fetchPlaybooksClaim, type PlaybooksClaim } from "../lib/api";

export interface UsePlaybooksClaimResult {
  claim: PlaybooksClaim | null;
  isLoading: boolean;
  error: unknown;
  refetch: () => Promise<unknown>;
}

/**
 * usePlaybooksClaim polls the test-genie playbooks-claim endpoint every 5s
 * to surface concurrency-guard state for the given scenario. Returns null
 * when no claim is held.
 */
export function usePlaybooksClaim(scenarioName: string | undefined): UsePlaybooksClaimResult {
  const name = (scenarioName ?? "").trim();
  const q = useQuery<PlaybooksClaim | null>({
    queryKey: ["playbooks-claim", name],
    queryFn: () => fetchPlaybooksClaim(name),
    enabled: name.length > 0,
    refetchInterval: 5000,
    staleTime: 2500,
  });
  return {
    claim: q.data ?? null,
    isLoading: q.isLoading,
    error: q.error,
    refetch: q.refetch,
  };
}
