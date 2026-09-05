import { useQuery } from "@tanstack/react-query";

import {
  getApprovedDependencyTriage,
  listApprovedDependencies,
  listSecurityGovernanceGaps,
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

  const triageQuery = useQuery({
    queryKey: ["governance-triage", policyMode],
    queryFn: () => getApprovedDependencyTriage({ policyMode, limit: 8 }),
    refetchOnWindowFocus: false
  });

  const securityGapsQuery = useQuery({
    queryKey: ["governance-security-gaps"],
    queryFn: () => listSecurityGovernanceGaps({ limit: 8 }),
    refetchOnWindowFocus: false
  });

  return {
    fleet: fleetQuery.data ?? null,
    records: recordsQuery.data?.records ?? [],
    recordsGuidance: recordsQuery.data?.guidance ?? "",
    triage: triageQuery.data ?? null,
    securityGaps: securityGapsQuery.data ?? null,
    loading:
      fleetQuery.isLoading ||
      fleetQuery.isFetching ||
      recordsQuery.isLoading ||
      recordsQuery.isFetching ||
      triageQuery.isLoading ||
      triageQuery.isFetching ||
      securityGapsQuery.isLoading ||
      securityGapsQuery.isFetching,
    error: fleetQuery.error ?? recordsQuery.error ?? triageQuery.error ?? securityGapsQuery.error ?? null,
    refresh: async () => {
      await Promise.all([
        fleetQuery.refetch(),
        recordsQuery.refetch(),
        triageQuery.refetch(),
        securityGapsQuery.refetch()
      ]);
    }
  };
}
