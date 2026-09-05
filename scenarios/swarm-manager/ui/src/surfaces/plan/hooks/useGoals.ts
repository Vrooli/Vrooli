/**
 * useGoals — react-query access to the goals list and mutations. Goal state
 * changes invalidate the goals query and the plan board (goal-scoped views and
 * badges re-derive from the fresh list).
 */

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { defaultQueryOptions } from "../../../lib";
import { goalsService } from "../../../services";
import { useActionMutation } from "../../../hooks/useActionMutation";
import type { CreateGoalInput, GoalWithScope } from "../../../types/goal";
import { usePlanDataStore } from "../stores/plan-data-store";

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
    const plan = usePlanDataStore.getState();
    if (plan.board) {
      void plan.fetchBoard({ force: true, silent: true });
    }
  };

  const create = useActionMutation({
    mutationFn: (input: CreateGoalInput) => goalsService.create(input),
    errorMessage: "Couldn't create that goal",
    successMessage: (goal) => `Created goal ${goal.goal.title || goal.goal.name}`,
    source: "useGoalMutations.create",
    onSuccess: invalidate,
  });

  const addTargets = useActionMutation({
    mutationFn: ({ name, targets }: { name: string; targets: string[] }) =>
      goalsService.addTargets(name, targets),
    errorMessage: "Couldn't add those targets",
    successMessage: (_goal, { targets }) =>
      targets.length === 1 ? "Target added" : `${targets.length} targets added`,
    source: "useGoalMutations.addTargets",
    onSuccess: invalidate,
  });

  const removeTargets = useActionMutation({
    mutationFn: ({ name, targets }: { name: string; targets: string[] }) =>
      goalsService.removeTargets(name, targets),
    errorMessage: "Couldn't remove those targets",
    source: "useGoalMutations.removeTargets",
    onSuccess: invalidate,
  });

  const setPriority = useActionMutation({
    mutationFn: ({ name, priority }: { name: string; priority: number }) =>
      goalsService.setPriority(name, priority),
    errorMessage: "Couldn't change that goal's priority",
    source: "useGoalMutations.setPriority",
    onSuccess: invalidate,
  });

  return { create, addTargets, removeTargets, setPriority };
}
