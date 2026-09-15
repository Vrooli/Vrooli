import { useQuery } from "@tanstack/react-query";
import { getSelfHealth, type SelfHealth } from "../lib/api";

// useSelfHealth reads Test Genie's own observability snapshot
// (RunsService.GetSelfHealth): phase catalog, live provider conformance, the
// reliability/performance ledger, and the persisted trend series. Conformance is
// a live scan, so the window is kept modest and refetch is infrequent.
export function useSelfHealth() {
  return useQuery<SelfHealth>({
    queryKey: ["self-health"],
    queryFn: () => getSelfHealth({ includeTrend: true }),
    refetchInterval: 60000
  });
}
