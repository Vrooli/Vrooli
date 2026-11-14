/**
 * Targets query + mutation hooks. Register/deregister invalidate the target
 * lists and the posture rollup so the Overview and Targets surfaces refresh
 * together after a co-equal UI edit.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  deregisterTarget,
  getTarget,
  listTargets,
  registerTarget,
  type RegisterTargetInput,
} from "../api/targets";
import { queryKeys } from "./keys";

export function useTargets(owner = "") {
  return useQuery({
    queryKey: queryKeys.targets(owner),
    queryFn: () => listTargets(owner),
  });
}

export function useTarget(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.target(id ?? ""),
    queryFn: () => getTarget(id ?? ""),
    enabled: Boolean(id),
  });
}

function useTargetInvalidation() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: ["targets"] });
    void qc.invalidateQueries({ queryKey: ["targetStatus"] });
  };
}

export function useRegisterTarget() {
  const invalidate = useTargetInvalidation();
  return useMutation({
    mutationFn: (input: RegisterTargetInput) => registerTarget(input),
    onSuccess: invalidate,
  });
}

export function useDeregisterTarget() {
  const invalidate = useTargetInvalidation();
  return useMutation({
    mutationFn: ({ owner, name }: { owner: string; name: string }) =>
      deregisterTarget(owner, name),
    onSuccess: invalidate,
  });
}
