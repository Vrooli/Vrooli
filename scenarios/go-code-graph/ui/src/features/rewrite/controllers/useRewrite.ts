import { useMutation, type UseMutationResult } from "@tanstack/react-query";

import { goCodeGraphClient } from "../../../api/rewrite";
import type {
  Operation,
  RewritePlanResponse,
  RewriteApplyResponse,
} from "../../../api/rewrite";

export interface PlanArgs {
  readonly modulePath: string;
  readonly operations: readonly Operation[];
}

/**
 * RewritePlan mutation — validates + normalizes operations and returns a
 * plan_id. No filesystem changes; safe to run repeatedly as the user edits
 * the operations list.
 */
export function useRewritePlan(): UseMutationResult<RewritePlanResponse, Error, PlanArgs> {
  return useMutation({
    mutationFn: ({ modulePath, operations }: PlanArgs) =>
      goCodeGraphClient.rewritePlan({ modulePath, operations: [...operations] }),
  });
}

export interface ApplyArgs {
  readonly modulePath: string;
  readonly planId: string;
}

/**
 * RewriteApply mutation — executes a previously-planned set of operations
 * against the filesystem. apply=true is mandatory (the service rejects
 * apply=false), so this hook always sends it; the UI gates the call behind a
 * confirm dialog.
 */
export function useRewriteApply(): UseMutationResult<RewriteApplyResponse, Error, ApplyArgs> {
  return useMutation({
    mutationFn: ({ modulePath, planId }: ApplyArgs) =>
      goCodeGraphClient.rewriteApply({ modulePath, planId, apply: true }),
  });
}
