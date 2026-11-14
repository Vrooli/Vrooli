import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  GetMatrixResponse,
  LogManualValidationResponse,
} from "@vrooli/proto-types/business-health/v1/contract/contract_pb";

import { contractClient } from "../../api/contract";

/** React Query key for a scenario's traceability matrix. */
export const matrixQueryKey = (scenario: string) => ["matrix", scenario] as const;

/**
 * Load the OT × requirement × validation × evidence traceability join for one
 * scenario. Disabled until a scenario slug is chosen so the surface starts on
 * an intentional empty state rather than a failing request.
 */
export function useMatrix(scenario: string) {
  return useQuery<GetMatrixResponse>({
    queryKey: matrixQueryKey(scenario),
    enabled: scenario.length > 0,
    queryFn: ({ signal }) => contractClient.getMatrix({ scenario }, { signal }),
  });
}

export interface LogAttestationInput {
  readonly requirementId: string;
  readonly attestedBy: string;
  readonly notes: string;
}

/**
 * Append a manual attestation to the scenario's ledger, then refresh the
 * matrix so the new evidence appears. The scenario is captured at hook
 * construction so callers only supply the per-requirement fields.
 */
export function useLogAttestation(scenario: string) {
  const queryClient = useQueryClient();
  return useMutation<LogManualValidationResponse, unknown, LogAttestationInput>({
    mutationFn: (input) =>
      contractClient.logManualValidation({
        scenario,
        requirementId: input.requirementId,
        attestedBy: input.attestedBy,
        notes: input.notes,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: matrixQueryKey(scenario) });
    },
  });
}
