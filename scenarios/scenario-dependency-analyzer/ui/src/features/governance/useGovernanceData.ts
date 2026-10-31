import { useQuery } from "@tanstack/react-query";

import {
  getApprovedDependencyTriage,
  listApprovedDependencies,
  listSecurityGovernanceGaps,
  validateFleetApprovedDependencies,
  type GovernancePolicyMode
} from "../../api/governance";

// Prefixes identify every view derived from governance decisions. Mutations
// invalidate these same prefixes, including every cached policy-mode variant.
export const governanceQueryKeys = {
  fleet: ["governance-fleet"],
  records: ["governance-records"],
  triage: ["governance-triage"],
  securityGaps: ["governance-security-gaps"]
} as const;

export function useGovernanceData(policyMode: GovernancePolicyMode) {
  const fleetQuery = useQuery({
    queryKey: [...governanceQueryKeys.fleet, policyMode],
    queryFn: () => validateFleetApprovedDependencies(policyMode),
    refetchOnWindowFocus: false
  });

  const recordsQuery = useQuery({
    queryKey: governanceQueryKeys.records,
    queryFn: listApprovedDependencies,
    refetchOnWindowFocus: false
  });

  const triageQuery = useQuery({
    queryKey: [...governanceQueryKeys.triage, policyMode],
    queryFn: () => getApprovedDependencyTriage({ policyMode, limit: 8 }),
    refetchOnWindowFocus: false
  });

  const securityGapsQuery = useQuery({
    queryKey: governanceQueryKeys.securityGaps,
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
