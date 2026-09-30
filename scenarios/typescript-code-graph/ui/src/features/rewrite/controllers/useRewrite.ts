import { useMutation, type UseMutationResult } from "@tanstack/react-query";

import { tsCodeGraphClient } from "../../../api/rewrite";
import type {
  Operation,
  RewritePlanResponse,
  RewriteApplyResponse,
} from "../../../api/rewrite";

export interface PlanArgs {
  readonly projectPath: string;
  readonly operations: readonly Operation[];
}

/**
 * RewritePlan mutation — validates + normalizes operations and returns a
 * plan_id. No filesystem changes; safe to run repeatedly as the user edits
 * the operations list.
 */
export function useRewritePlan(): UseMutationResult<RewritePlanResponse, Error, PlanArgs> {
  return useMutation({
    mutationFn: ({ projectPath, operations }: PlanArgs) =>
      tsCodeGraphClient.rewritePlan({ projectPath, operations: [...operations] }),
  });
}

export interface ApplyArgs {
  readonly projectPath: string;
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
    mutationFn: ({ projectPath, planId }: ApplyArgs) =>
      tsCodeGraphClient.rewriteApply({ projectPath, planId, apply: true }),
  });
}
