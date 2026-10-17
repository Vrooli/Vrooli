import { useEffect, useRef, useState } from "react";
import { CheckCircle2, Circle, XCircle } from "lucide-react";
import { ApiError } from "../../../lib/apiErrors";
import { cancelActionFor, formatAge, isTerminalState } from "../../../lib/consoleActions";
import type { AuthzMatrix, OperationStanding } from "../../../types/console";
import { Collapsible } from "../../ui/collapsible";
import { ActionButton, ActivityIndicator, CopyButton, KeyValue, LiveRegion, Panel, SkeletonRows, StatusPill, type Tone } from "./ConsolePrimitives";
import { DeniedState } from "./DeniedState";
import { DestructiveActionDialog } from "./DestructiveActionDialog";
import { NextActionControl, type NextActionHandlers } from "./NextActionControl";

export interface OperationPanelProps {
  standing: OperationStanding | null | undefined;
  loading: boolean;
  error: unknown;
  /** True when the standing was rehydrated from durable state after an interruption. */
  resumed?: boolean;
  /** Latest human message from the progress stream, if attached. */
  liveMessage?: string | null;
  target: string;
  matrix?: AuthzMatrix | null;
  onCancel?: (operationId: string) => Promise<unknown> | void;
  cancelPending?: boolean;
  cancelError?: unknown;
  handlers?: NextActionHandlers;
  /** Primary action rendered when there is no operation (Review plan). */
  emptyAction?: React.ReactNode;
}

function stateTone(state: string | undefined): Tone {
  switch (state) {
    case "succeeded":
      return "success";
    case "failed":
    case "failed_recovery":
      return "danger";
    case "cancelled":
    case "cancel_requested":
    case "waiting_input":
    case "recovering":
    case "reconciling":
      return "warning";
    case "running":
    case "verifying":
    case "admitted":
      return "info";
    default:
      return "neutral";
  }
}

function stepIcon(outcome: "done" | "active" | "failed" | "pending") {
  switch (outcome) {
    case "done":
      return <CheckCircle2 className="h-4 w-4 text-emerald-300" aria-hidden="true" />;
    case "failed":
      return <XCircle className="h-4 w-4 text-red-300" aria-hidden="true" />;
    case "active":
      return <ActivityIndicator label="" />;
    default:
      return <Circle className="h-4 w-4 text-slate-500" aria-hidden="true" />;
  }
}

/**
 * OperationPanel answers "which operation is running or waiting, and what
 * happens next?" from the durable standing: state, completed steps, the
 * active step, receipts, unknown effects and the API's next action. It
 * never renders a percentage.
 */
export function OperationPanel({
  standing,
  loading,
  error,
  resumed = false,
  liveMessage,
  target,
  matrix,
  onCancel,
  cancelPending = false,
  cancelError,
  handlers,
  emptyAction,
}: OperationPanelProps) {
  const headingRef = useRef<HTMLHeadingElement>(null);
  const [confirmCancel, setConfirmCancel] = useState(false);
  const terminal = Boolean(standing && (standing.terminal || isTerminalState(standing.state)));
  const failedRecovery = Boolean(standing && (standing.state === "failed_recovery" || (standing.unknown_effects?.length ?? 0) > 0));
  const denied = error instanceof ApiError && ["unauthenticated", "forbidden_scope", "forbidden_target", "forbidden_origin"].includes(error.code);
  const state = loading ? "loading" : denied ? "denied" : !standing ? "empty" : failedRecovery ? "failed-recovery" : resumed && !terminal ? "interrupted" : "ready";
  const cancelAction = cancelActionFor(standing, matrix ? "scenario-to-cloud:destructive" : undefined);

  useEffect(() => {
    if ((resumed || failedRecovery) && standing) headingRef.current?.focus();
  }, [resumed, failedRecovery, standing?.operation_id]); // eslint-disable-line react-hooks/exhaustive-deps

  const announcement = standing
    ? `Operation ${standing.state}${standing.active_step ? `, step ${standing.active_step}` : ""}${resumed ? ", resumed from durable state" : ""}`
    : "";

  const failedSteps = new Set((standing?.step_receipts ?? []).filter((r) => r.outcome === "failed").map((r) => r.step));
  const steps: { id: string; outcome: "done" | "active" | "failed" | "pending" }[] = standing
    ? [
        ...standing.completed_steps.map((id) => ({ id, outcome: "done" as const })),
        ...Array.from(failedSteps).filter((id) => !standing.completed_steps.includes(id)).map((id) => ({ id, outcome: "failed" as const })),
        ...(standing.active_step && !standing.completed_steps.includes(standing.active_step) && !failedSteps.has(standing.active_step)
          ? [{ id: standing.active_step, outcome: (terminal ? "pending" : "active") as "active" | "pending" }]
          : []),
      ]
    : [];

  return (
    <Panel
      testId="console-operation"
      title="Operation"
      state={state}
      headingRef={headingRef}
      description={standing ? `Durable operation ${standing.operation_id}` : undefined}
      actions={
        standing ? (
          <>
            <StatusPill tone={stateTone(standing.state)} testId="console-operation-state">
              {standing.state}
            </StatusPill>
            {resumed && (
              <StatusPill tone="warning" testId="console-operation-resumed">
                resumed from durable state
              </StatusPill>
            )}
          </>
        ) : null
      }
    >
      <LiveRegion message={announcement} assertive testId="console-operation-live" />
      {loading && <SkeletonRows rows={4} />}
      {denied && <DeniedState error={error} matrix={matrix} method="GET" path={`/api/v1/operations/${standing?.operation_id ?? "unknown"}`} target={target} testId="console-operation-denied" />}
      {!loading && !denied && Boolean(error) && (
        <p role="alert" className="text-sm text-amber-200" data-testid="console-operation-error">
          Standing unavailable: {error instanceof ApiError ? `${error.code}: ${error.message}` : String(error)}
        </p>
      )}
      {!loading && !standing && !error && (
        <div className="space-y-3" data-testid="console-operation-empty">
          <p className="text-sm text-slate-200">No operation is running or waiting for this deployment.</p>
          {emptyAction}
        </div>
      )}
      {standing && (
        <div className="space-y-4">
          <ol className="space-y-1" aria-label="Operation steps" data-testid="console-operation-steps">
            {steps.length === 0 && <li className="text-sm text-slate-300">No steps have been recorded yet.</li>}
            {steps.map((step) => (
              <li key={step.id} data-outcome={step.outcome} className="flex items-center gap-2 text-sm">
                {stepIcon(step.outcome)}
                <span className={step.outcome === "active" ? "font-medium text-slate-100" : step.outcome === "failed" ? "text-red-200" : "text-slate-200"}>{step.id}</span>
                {step.outcome === "active" && <span className="text-xs text-slate-300">(active)</span>}
              </li>
            ))}
          </ol>
          {liveMessage && !terminal && (
            <p className="text-xs text-slate-300" data-testid="console-operation-live-message">
              {liveMessage}
            </p>
          )}
          {standing.result?.message && (
            <p className="text-sm text-slate-100" data-testid="console-operation-result">
              {standing.result.message}
            </p>
          )}
          {standing.error && (
            <div role="alert" className="rounded-md border border-red-500/40 bg-red-500/10 p-3 text-sm text-red-100" data-testid="console-operation-refusal">
              <p className="font-medium">
                {standing.error.code}: {standing.error.message}
              </p>
              {standing.error.next_action && <div className="mt-2"><NextActionControl next={standing.error.next_action} handlers={handlers} testId="console-operation-error-next-action" /></div>}
            </div>
          )}
          {standing.unknown_effects && standing.unknown_effects.length > 0 && (
            <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3" data-testid="console-operation-unknown-effects">
              <p className="text-sm font-medium text-amber-100">Effects with unknown outcome</p>
              <ul className="mt-1 space-y-1 text-xs text-amber-100/90">
                {standing.unknown_effects.map((effect) => (
                  <li key={`${effect.step}-${effect.recorded_at}`}>
                    <span className="font-mono">{effect.step}</span>: {effect.reason}. Retry policy {effect.retry}. {effect.next_action}
                  </li>
                ))}
              </ul>
            </div>
          )}
          <div className="space-y-2" data-testid="console-operation-next">
            <p className="text-xs uppercase tracking-wide text-slate-400">Next</p>
            {standing.next_action ? (
              <NextActionControl next={standing.next_action} handlers={handlers} testId="console-operation-next-action" />
            ) : (
              <p className="text-sm text-slate-200">{terminal ? `The operation is ${standing.state}; no further action was named.` : "The API named no next action."}</p>
            )}
          </div>
          <dl className="space-y-1">
            <KeyValue label="Reattach" mono testId="console-operation-reattach">
              <span className="break-all">{standing.reattach_command}</span> <CopyButton value={standing.reattach_command} label="reattach command" />
            </KeyValue>
            <KeyValue label="Plan digest" mono>
              {standing.plan_digest}
            </KeyValue>
            <KeyValue label="Updated">{formatAge(standing.updated_at) || standing.updated_at}</KeyValue>
            {standing.heartbeat_at && <KeyValue label="Heartbeat">{formatAge(standing.heartbeat_at) || standing.heartbeat_at}</KeyValue>}
          </dl>
          {onCancel && (
            <div className="flex flex-wrap gap-2">
              <ActionButton available={cancelAction.available && !cancelPending} reason={cancelAction.reason?.message} tone="secondary" testId="console-operation-cancel" onClick={() => setConfirmCancel(true)}>
                {cancelAction.label}
              </ActionButton>
            </div>
          )}
          {cancelError instanceof ApiError && !confirmCancel && (
            <DeniedState error={cancelError} matrix={matrix} method="POST" path={`/api/v1/operations/${standing.operation_id}/cancel`} target={target} testId="console-operation-cancel-denied" />
          )}
          {standing.step_receipts.length > 0 && (
            <Collapsible title={`Step receipts (${standing.step_receipts.length})`}>
              <ul className="mt-2 space-y-1 text-xs" data-testid="console-operation-receipts">
                {standing.step_receipts.map((receipt) => (
                  <li key={`${receipt.step}-${receipt.completed_at}`} className="flex flex-wrap gap-2">
                    <span className="font-mono text-slate-100">{receipt.step}</span>
                    <span className="text-slate-300">{receipt.outcome}</span>
                    <span className="text-slate-400">fence {receipt.fence}</span>
                    {receipt.replayed && <span className="text-slate-400">replayed</span>}
                    {receipt.detail && <span className="text-slate-300 break-words min-w-0">{receipt.detail}</span>}
                    {receipt.error && <span className="text-red-200 break-words min-w-0">{receipt.error}</span>}
                  </li>
                ))}
              </ul>
            </Collapsible>
          )}
        </div>
      )}
      {confirmCancel && standing && onCancel && (
        <DestructiveActionDialog
          title="Cancel this operation?"
          description="Cancellation is recorded now and honoured at the operation's next declared cancel point. Steps already applied stay applied."
          target={target}
          affectedData={[]}
          details={[{ label: "Operation", value: <span className="font-mono text-xs">{standing.operation_id}</span> }]}
          confirmText={standing.operation_id.slice(0, 8)}
          confirmLabel="Cancel operation"
          isPending={cancelPending}
          error={cancelError instanceof Error ? cancelError.message : cancelError ? String(cancelError) : null}
          onConfirm={async () => {
            await onCancel(standing.operation_id);
            setConfirmCancel(false);
          }}
          onCancel={() => setConfirmCancel(false)}
        />
      )}
    </Panel>
  );
}
