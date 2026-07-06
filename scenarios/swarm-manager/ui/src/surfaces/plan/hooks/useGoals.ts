/**
 * useGoals — react-query access to the goals list and mutations. Goal state
 * changes invalidate the goals query and the plan board (goal-scoped views and
 * badges re-derive from the fresh list).
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { defaultQueryOptions } from "../../../lib";
import { goalsService } from "../../../services";
import type { CreateGoalInput, GoalWithScope } from "../../../types/goal";

export const GOALS_QUERY_KEY = ["goals"] as const;

export function useGoals() {
  return useQuery<GoalWithScope[]>({
    queryKey: GOALS_QUERY_KEY,
    queryFn: () => goalsService.list(),
    ...defaultQueryOptions,
  });
}

/** Mutations that create a goal or add targets, invalidating the goals list. */
export function useGoalMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: GOALS_QUERY_KEY });
  };

  const create = useMutation({
    mutationFn: (input: CreateGoalInput) => goalsService.create(input),
    onSuccess: invalidate,
  });

  const addTargets = useMutation({
    mutationFn: ({ name, targets }: { name: string; targets: string[] }) =>
      goalsService.addTargets(name, targets),
    onSuccess: invalidate,
  });

  const setPriority = useMutation({
    mutationFn: ({ name, priority }: { name: string; priority: number }) =>
      goalsService.setPriority(name, priority),
    onSuccess: invalidate,
  });

  return { create, addTargets, setPriority };
}
