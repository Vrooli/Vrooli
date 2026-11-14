/**
 * Restores query + mutation hooks. Verify (non-destructive) and restore
 * (destructive) both invalidate restore history and the posture rollup so a
 * fresh verify immediately clears a target's unverified chip on the Overview.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { listRestores, restoreTarget, verifyTarget } from "../api/restores";
import { queryKeys } from "./keys";

export function useRestores(targetId = "") {
  return useQuery({
    queryKey: queryKeys.restores(targetId),
    queryFn: () => listRestores(targetId),
  });
}

function useRestoreInvalidation() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: ["restores"] });
    void qc.invalidateQueries({ queryKey: ["targetStatus"] });
  };
}

export function useVerifyTarget() {
  const invalidate = useRestoreInvalidation();
  return useMutation({
    mutationFn: ({
      targetId,
      destinationId,
      snapshotId,
    }: {
      targetId: string;
      destinationId: string;
      snapshotId: string;
    }) => verifyTarget(targetId, destinationId, snapshotId),
    onSuccess: invalidate,
  });
}

export function useRestoreTarget() {
  const invalidate = useRestoreInvalidation();
  return useMutation({
    mutationFn: ({
      targetId,
      destinationId,
      snapshotId,
      location,
    }: {
      targetId: string;
      destinationId: string;
      snapshotId: string;
      location: string;
    }) => restoreTarget(targetId, destinationId, snapshotId, location),
    onSuccess: invalidate,
  });
}
