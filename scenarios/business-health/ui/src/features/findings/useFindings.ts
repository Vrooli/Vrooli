import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import type { ValidateScenarioResponse } from "@vrooli/proto-types/business-health/v1/contract/contract_pb";
import type { FixResponse } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { contractClient } from "../../api/contract";
import { validationClient } from "../../api/validation";

/** React Query key for a scenario's business-contract validation report. */
export const findingsQueryKey = (scenario: string) => ["findings", scenario] as const;

/**
 * Validate one scenario's business contract and surface its findings,
 * capability rollups, and verdict. Disabled until a scenario slug is chosen so
 * the surface starts on an intentional empty state rather than a failing call.
 */
export function useFindings(scenario: string) {
  return useQuery<ValidateScenarioResponse>({
    queryKey: findingsQueryKey(scenario),
    enabled: scenario.length > 0,
    queryFn: ({ signal }) => contractClient.validateScenario({ scenario }, { signal }),
  });
}

/**
 * Result of a fix preview. A provider that ships no deterministic fixer answers
 * PreviewFix with Unimplemented — that is a normal "nothing to do here" signal,
 * not a hard error, so we model it as a first-class variant the UI can render
 * calmly rather than surfacing it as a failed request.
 */
export type FixPreview =
  | { readonly kind: "candidates"; readonly response: FixResponse }
  | { readonly kind: "unimplemented" };

/**
 * Preview the deterministic edits a provider would make for the given rule
 * ids. Kept separate from applying so a preview never touches disk; the caller
 * decides whether to escalate to an explicit apply.
 */
export function usePreviewFix(scenario: string) {
  return useMutation<FixPreview, unknown, readonly string[]>({
    mutationFn: async (ruleIds) => {
      try {
        const response = await validationClient.previewFix({ scenario, ruleIds: [...ruleIds] });
        return { kind: "candidates", response };
      } catch (err) {
        if (err instanceof ConnectError && err.code === Code.Unimplemented) {
          return { kind: "unimplemented" };
        }
        throw err;
      }
    },
  });
}

/**
 * Apply the deterministic edits for the given rule ids to disk, then refresh
 * the findings query so the remediated state re-validates. Apply is never
 * implicit — a caller invokes this only from an explicit confirm affordance.
 */
export function useApplyFix(scenario: string) {
  const queryClient = useQueryClient();
  return useMutation<FixResponse, unknown, readonly string[]>({
    mutationFn: (ruleIds) => validationClient.applyFix({ scenario, ruleIds: [...ruleIds] }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: findingsQueryKey(scenario) });
    },
  });
}
