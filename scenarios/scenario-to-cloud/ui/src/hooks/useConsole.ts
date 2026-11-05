import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  applyDeploymentPlan,
  cancelOperation,
  clearOperationPointer,
  compileDeploymentPlan,
  getAuthzMatrix,
  getOperation,
  listDeploymentOperations,
  listRecoveryPoints,
  loadOperationPointer,
  restoreRecoveryPoint,
  rollbackAdmission,
  saveOperationPointer,
  waitOperation,
} from "../lib/consoleApi";
import { ApiError, ErrorCodes } from "../lib/apiErrors";
import { isTerminalState } from "../lib/consoleActions";
import type { DurableOperationPointer, OperationStanding } from "../types/console";

/** Server-side wait bound per round; the browser never tight-polls. */
export const OPERATION_WAIT_SECONDS = 20;

export function useAuthzMatrix() {
  return useQuery({
    queryKey: ["authzMatrix"],
    queryFn: getAuthzMatrix,
    staleTime: 10 * 60_000,
  });
}

/**
 * The compiled plan is a read (no records, no target effects), so the console
 * compiles it on load to learn the desired release and the plan outcome.
 */
export function useCompiledPlan(deploymentId: string | null, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: ["plan", deploymentId],
    queryFn: () => compileDeploymentPlan(deploymentId as string),
    enabled: !!deploymentId && (options.enabled ?? true),
    staleTime: 30_000,
    retry: false,
  });
}

export function useApplyPlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ deploymentId, planDigest, requestKey }: { deploymentId: string; planDigest: string; requestKey: string }) =>
      applyDeploymentPlan(deploymentId, { plan_digest: planDigest, request_key: requestKey }),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: ["deployment", variables.deploymentId] });
      queryClient.invalidateQueries({ queryKey: ["deploymentOperations", variables.deploymentId] });
    },
  });
}

/**
 * Standing of one operation. While the operation is not terminal the query
 * uses the server-side wait (one blocking call per round); once terminal it
 * stops refetching.
 */
export function useOperationStanding(operationId: string | null) {
  const lastRef = useRef<OperationStanding | null>(null);
  return useQuery({
    queryKey: ["operation", operationId],
    queryFn: async () => {
      const id = operationId as string;
      const last = lastRef.current;
      const standing =
        last && !last.terminal && !isTerminalState(last.state) ? await waitOperation(id, OPERATION_WAIT_SECONDS) : await getOperation(id);
      lastRef.current = standing;
      return standing;
    },
    enabled: !!operationId,
    retry: false,
    refetchInterval: (query) => {
      const data = query.state.data as OperationStanding | undefined;
      if (!data) return false;
      return data.terminal || isTerminalState(data.state) ? false : 250;
    },
  });
}

export function useDeploymentOperations(deploymentId: string | null) {
  return useQuery({
    queryKey: ["deploymentOperations", deploymentId],
    queryFn: () => listDeploymentOperations(deploymentId as string),
    enabled: !!deploymentId,
    retry: false,
    staleTime: 15_000,
  });
}

export function useCancelOperation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (operationId: string) => cancelOperation(operationId),
    onSuccess: (standing) => {
      queryClient.setQueryData(["operation", standing.operation_id], standing);
    },
  });
}

export function useRecoveryPoints(deploymentId: string | null) {
  return useQuery({
    queryKey: ["recoveryPoints", deploymentId],
    queryFn: () => listRecoveryPoints(deploymentId as string),
    enabled: !!deploymentId,
    retry: false,
    staleTime: 30_000,
  });
}

export function useRollbackAdmission() {
  return useMutation({
    mutationFn: ({
      deploymentId,
      currentSchema,
      targetSchema,
    }: {
      deploymentId: string;
      currentSchema: string;
      targetSchema: string;
    }) => rollbackAdmission(deploymentId, { current_schema: currentSchema, target_schema: targetSchema }),
  });
}

export function useRestoreRecoveryPoint() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ deploymentId, recoveryPointId, into }: { deploymentId: string; recoveryPointId: string; into: Record<string, string> }) =>
      restoreRecoveryPoint(deploymentId, recoveryPointId, { into }),
    onSuccess: (_r, variables) => {
      queryClient.invalidateQueries({ queryKey: ["recoveryPoints", variables.deploymentId] });
      queryClient.invalidateQueries({ queryKey: ["deploymentOperations", variables.deploymentId] });
    },
  });
}

export type DurableOperationResolution = {
  /** The operation the console is attached to, if any. */
  pointer: DurableOperationPointer | null;
  /** True when the pointer came from durable state after a reload, not from an action in this page. */
  resumed: boolean;
  /** True while the durable state is being rehydrated. */
  rehydrating: boolean;
  attach: (pointer: DurableOperationPointer) => void;
  detach: () => void;
};

/**
 * useDurableOperation resolves which operation the page is attached to:
 * an action in this page, the sessionStorage pointer written before an
 * interruption, or the newest non-terminal operation the API lists for the
 * deployment. A pointer whose operation no longer exists is discarded.
 */
export function useDurableOperation(deploymentId: string | null): DurableOperationResolution {
  const [pointer, setPointer] = useState<DurableOperationPointer | null>(null);
  const [resumed, setResumed] = useState(false);
  const [rehydrating, setRehydrating] = useState(!!deploymentId);
  const queryClient = useQueryClient();

  useEffect(() => {
    let cancelled = false;
    if (!deploymentId) {
      setRehydrating(false);
      return;
    }
    setRehydrating(true);
    (async () => {
      const stored = loadOperationPointer(deploymentId);
      if (stored) {
        try {
          const standing = await getOperation(stored.operation_id);
          if (cancelled) return;
          queryClient.setQueryData(["operation", standing.operation_id], standing);
          setPointer({ deployment_id: deploymentId, operation_id: standing.operation_id, plan_digest: standing.plan_digest });
          setResumed(true);
          setRehydrating(false);
          return;
        } catch (err) {
          if (err instanceof ApiError && (err.code === "operation_not_found" || err.code === ErrorCodes.deploymentNotFound)) {
            clearOperationPointer(deploymentId);
          }
        }
      }
      try {
        const listed = await listDeploymentOperations(deploymentId);
        if (cancelled) return;
        const active = listed.operations.find((op) => !op.terminal && !isTerminalState(op.state));
        if (active) {
          queryClient.setQueryData(["operation", active.operation_id], active);
          const next = { deployment_id: deploymentId, operation_id: active.operation_id, plan_digest: active.plan_digest };
          saveOperationPointer(next);
          setPointer(next);
          setResumed(true);
        }
      } catch {
        // Listing is a convenience; the page still renders without it.
      }
      if (!cancelled) setRehydrating(false);
    })();
    return () => {
      cancelled = true;
    };
  }, [deploymentId, queryClient]);

  const attach = useCallback((next: DurableOperationPointer) => {
    saveOperationPointer(next);
    setPointer(next);
    setResumed(false);
  }, []);

  const detach = useCallback(() => {
    if (deploymentId) clearOperationPointer(deploymentId);
    setPointer(null);
    setResumed(false);
  }, [deploymentId]);

  return { pointer, resumed, rehydrating, attach, detach };
}
