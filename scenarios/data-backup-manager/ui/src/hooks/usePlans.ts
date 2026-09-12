/**
 * Plans query + mutation hooks. Mutations invalidate the plan list; triggering
 * a run lives in useRuns (it touches run history, not plan definitions).
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createPlan,
  deletePlan,
  getPlan,
  listPlans,
  updatePlan,
  type PlanInput,
} from "../api/plans";
import { queryKeys } from "./keys";

export function usePlans() {
  return useQuery({ queryKey: queryKeys.plans, queryFn: listPlans });
}

export function usePlan(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.plan(id ?? ""),
    queryFn: () => getPlan(id ?? ""),
    enabled: Boolean(id),
  });
}

function usePlanInvalidation() {
  const qc = useQueryClient();
  return () => void qc.invalidateQueries({ queryKey: queryKeys.plans });
}

export function useCreatePlan() {
  const invalidate = usePlanInvalidation();
  return useMutation({
    mutationFn: (input: PlanInput) => createPlan(input),
    onSuccess: invalidate,
  });
}

export function useUpdatePlan() {
  const invalidate = usePlanInvalidation();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: PlanInput }) => updatePlan(id, input),
    onSuccess: invalidate,
  });
}

export function useDeletePlan() {
  const invalidate = usePlanInvalidation();
  return useMutation({
    mutationFn: (id: string) => deletePlan(id),
    onSuccess: invalidate,
  });
}
