import { useQuery } from "@tanstack/react-query";

import {
  listApprovedDependencies,
  validateFleetApprovedDependencies,
  type GovernancePolicyMode
} from "../../api/governance";

export function useGovernanceData(policyMode: GovernancePolicyMode) {
  const fleetQuery = useQuery({
    queryKey: ["governance-fleet", policyMode],
    queryFn: () => validateFleetApprovedDependencies(policyMode),
    refetchOnWindowFocus: false
  });

  const recordsQuery = useQuery({
    queryKey: ["governance-records"],
    queryFn: listApprovedDependencies,
    refetchOnWindowFocus: false
  });

  return {
    fleet: fleetQuery.data ?? null,
    records: recordsQuery.data?.records ?? [],
    recordsGuidance: recordsQuery.data?.guidance ?? "",
    loading: fleetQuery.isLoading || fleetQuery.isFetching || recordsQuery.isLoading || recordsQuery.isFetching,
    error: fleetQuery.error ?? recordsQuery.error ?? null,
    refresh: async () => {
      await Promise.all([fleetQuery.refetch(), recordsQuery.refetch()]);
    }
  };
}
