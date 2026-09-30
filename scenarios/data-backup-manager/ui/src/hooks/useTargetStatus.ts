/**
 * Posture rollup hook. ListTargetStatus carries both "backed up"
 * (`lastSuccessAt`) and "proven restorable" (`lastVerifiedAt`) per target, so
 * the Overview's coverage grid and the Targets table read from one query.
 */
import { useQuery } from "@tanstack/react-query";

import { listTargetStatus } from "../api/runs";
import { queryKeys } from "./keys";

export function useTargetStatus(owner = "") {
  return useQuery({
    queryKey: queryKeys.targetStatus(owner),
    queryFn: () => listTargetStatus(owner),
  });
}
