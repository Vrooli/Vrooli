import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useHealthObservation } from "../../../hooks/useLiveState";
import { useDeploymentProgress } from "../../../hooks/useDeploymentProgress";
import {
  useApplyPlan,
  useAuthzMatrix,
  useCancelOperation,
  useCompiledPlan,
  useDurableOperation,
  useOperationStanding,
  useRecoveryPoints,
  useRestoreRecoveryPoint,
  useRollbackAdmission,
} from "../../../hooks/useConsole";
import { newRequestKey } from "../../../lib/consoleApi";
import { isTerminalState, targetKey } from "../../../lib/consoleActions";
import type { DeploymentIdentity, DurableOperationPointer } from "../../../types/console";
import { ActionButton } from "./ConsolePrimitives";
import { HealthPanel } from "./HealthPanel";
import { OperationPanel } from "./OperationPanel";
import { PlanReview } from "./PlanReview";
import { RecoveryPanel } from "./RecoveryPanel";
import { ReleasePanel } from "./ReleasePanel";

export interface DeploymentConsoleProps {
  deployment: DeploymentIdentity;
  /** Called when the attached operation reaches a terminal state. */
  onOperationTerminal?: (state: string) => void;
  /** An operation admitted outside the console (legacy pipeline) to attach to. */
  attachRequest?: DurableOperationPointer | null;
}

/**
 * DeploymentConsole composes the release, health, operation and recovery
 * panels for one deployment and owns the review → apply → operation flow.
 * The compiled plan is read on load (read scope, no effects) so the console
 * knows the desired release and the plan outcome before any action.
 */
export function DeploymentConsole({ deployment, onOperationTerminal, attachRequest = null }: DeploymentConsoleProps) {
  const deploymentId = deployment.id;
  const target = targetKey(deployment.target);
  const queryClient = useQueryClient();
  const matrix = useAuthzMatrix();
  const plan = useCompiledPlan(deploymentId);
  const health = useHealthObservation(deploymentId);
  const recovery = useRecoveryPoints(deploymentId);
  const durable = useDurableOperation(deploymentId);
  const standing = useOperationStanding(durable.pointer?.operation_id ?? null);
  const apply = useApplyPlan();
  const cancel = useCancelOperation();
  const rollback = useRollbackAdmission();
  const restore = useRestoreRecoveryPoint();
  const [reviewOpen, setReviewOpen] = useState(false);
  const requestKeyRef = useRef<string | null>(null);
  const recoveryHeadingRef = useRef<HTMLHeadingElement>(null);
  const reviewRef = useRef<HTMLDivElement>(null);
  const terminalNotified = useRef<string | null>(null);

  useEffect(() => {
    if (attachRequest && attachRequest.operation_id !== durable.pointer?.operation_id) {
      terminalNotified.current = null;
      durable.attach(attachRequest);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attachRequest]);

  const operationActive = Boolean(standing.data && !standing.data.terminal && !isTerminalState(standing.data.state));
  const progress = useDeploymentProgress(operationActive ? deploymentId : null, { operationId: durable.pointer?.operation_id ?? null });

  useEffect(() => {
    const current = standing.data;
    if (!current) return;
    if ((current.terminal || isTerminalState(current.state)) && terminalNotified.current !== current.operation_id) {
      terminalNotified.current = current.operation_id;
      queryClient.invalidateQueries({ queryKey: ["deployment", deploymentId] });
      queryClient.invalidateQueries({ queryKey: ["healthObservation", deploymentId] });
      queryClient.invalidateQueries({ queryKey: ["plan", deploymentId] });
      onOperationTerminal?.(current.state);
    }
  }, [standing.data, deploymentId, queryClient, onOperationTerminal]);

  const openReview = useCallback(() => {
    requestKeyRef.current = null;
    setReviewOpen(true);
    void plan.refetch();
    window.setTimeout(() => {
      const review = reviewRef.current;
      if (!review || review.contains(document.activeElement)) return;
      review.querySelector<HTMLElement>("h2")?.focus();
    }, 0);
  }, [plan]);

  const focusRecovery = useCallback(() => {
    recoveryHeadingRef.current?.focus();
    recoveryHeadingRef.current?.scrollIntoView?.({ block: "start" });
  }, []);

  const onApply = useCallback(
    async (planDigest: string) => {
      if (!requestKeyRef.current) requestKeyRef.current = newRequestKey();
      const result = await apply.mutateAsync({ deploymentId, planDigest, requestKey: requestKeyRef.current });
      if (result.operation_id) {
        const pointer: DurableOperationPointer = { deployment_id: deploymentId, operation_id: result.operation_id, plan_digest: result.plan_digest };
        durable.attach(pointer);
        terminalNotified.current = null;
      }
      setReviewOpen(false);
    },
    [apply, deploymentId, durable],
  );

  const handlers = {
    onResume: openReview,
    onReplan: openReview,
    onRecovery: focusRecovery,
    onInspectOperation: () => document.querySelector<HTMLElement>('[data-testid="console-operation"] h2')?.focus(),
  };

  return (
    <div className="space-y-4" data-testid="console">
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ReleasePanel deploymentId={deploymentId} plan={plan.data} planError={plan.error} planLoading={plan.isLoading} observation={health.data} matrix={matrix.data} target={target} />
        <HealthPanel observation={health.data} loading={health.isLoading} error={health.error} />
      </div>
      {reviewOpen && (
        <div ref={reviewRef}>
          <PlanReview
            deploymentId={deploymentId}
            target={target}
            plan={plan.data}
            loading={plan.isFetching && !plan.data}
            error={plan.error}
            matrix={matrix.data}
            onApply={onApply}
            applyPending={apply.isPending}
            applyError={apply.error}
            onReplan={() => {
              requestKeyRef.current = null;
              apply.reset();
              void plan.refetch();
            }}
          />
          <div className="mt-2">
            <ActionButton available tone="secondary" testId="console-review-close" onClick={() => setReviewOpen(false)}>
              Close review
            </ActionButton>
          </div>
        </div>
      )}
      <OperationPanel
        standing={standing.data}
        loading={durable.rehydrating || (standing.isLoading && Boolean(durable.pointer))}
        error={standing.error}
        resumed={durable.resumed}
        liveMessage={progress.progress?.currentStepTitle ?? null}
        target={target}
        matrix={matrix.data}
        onCancel={(operationId) => cancel.mutateAsync(operationId)}
        cancelPending={cancel.isPending}
        cancelError={cancel.error}
        handlers={handlers}
        emptyAction={
          !reviewOpen ? (
            <ActionButton available testId="console-review-open" onClick={openReview}>
              Review plan
            </ActionButton>
          ) : null
        }
      />
      {standing.data && (standing.data.terminal || isTerminalState(standing.data.state)) && !reviewOpen && (
        <div className="flex flex-wrap gap-2">
          <ActionButton available testId="console-review-open" onClick={openReview}>
            Review a new plan
          </ActionButton>
          <ActionButton available tone="secondary" testId="console-operation-detach" onClick={durable.detach}>
            Dismiss finished operation
          </ActionButton>
        </div>
      )}
      <RecoveryPanel
        deploymentId={deploymentId}
        target={target}
        recoveryPoints={recovery.data?.recovery_points}
        loading={recovery.isLoading}
        error={recovery.error}
        plan={plan.data}
        matrix={matrix.data}
        headingRef={recoveryHeadingRef}
        onCheckRollback={async ({ currentSchema, targetSchema }) => {
          const res = await rollback.mutateAsync({ deploymentId, currentSchema, targetSchema });
          return res.verdict;
        }}
        onRestore={({ recoveryPointId, into }) => restore.mutateAsync({ deploymentId, recoveryPointId, into })}
        restorePending={restore.isPending}
      />
    </div>
  );
}
