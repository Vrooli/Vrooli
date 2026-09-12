import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  denyVulnerableDependency,
  previewVulnerabilityRemediation,
  upsertApprovedDependency,
  type ApprovedDependencyRecord
} from "../../api/governance";

export function useGovernanceMutations() {
  const queryClient = useQueryClient();
  const refreshGovernance = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["governance-fleet"] }),
      queryClient.invalidateQueries({ queryKey: ["governance-records"] })
    ]);
  };

  const previewDecision = useMutation({
    mutationFn: (record: ApprovedDependencyRecord) => upsertApprovedDependency(record, true)
  });

  const applyDecision = useMutation({
    mutationFn: (record: ApprovedDependencyRecord) => upsertApprovedDependency(record, false),
    onSuccess: refreshGovernance
  });

  const previewRemediation = useMutation({
    mutationFn: (input: { ecosystem: string; packageName: string; vulnerabilityId: string }) =>
      previewVulnerabilityRemediation(input.ecosystem, input.packageName, input.vulnerabilityId)
  });

  const denyVulnerable = useMutation({
    mutationFn: (input: {
      ecosystem: string;
      packageName: string;
      vulnerabilityId: string;
      affectedRange: string;
      fixedRange: string;
      rationale: string;
      approvedBy: string;
      dryRun?: boolean;
    }) => denyVulnerableDependency(input),
    onSuccess: async (_result, input) => {
      if (input.dryRun === false) {
        await refreshGovernance();
      }
    }
  });

  return {
    previewDecision,
    applyDecision,
    previewRemediation,
    denyVulnerable
  };
}
