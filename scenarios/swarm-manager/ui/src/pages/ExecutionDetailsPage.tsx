/**
 * Execution Details Page
 *
 * Displays detailed information about a single execution record including:
 * - Execution metadata (status, mode, operation, timestamps)
 * - Prompt trace (if available)
 * - Action buttons (retry, cancel, follow-up, trigger review)
 */

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ChevronDown,
  ChevronUp,
  ClipboardCheck,
  Loader2,
  RotateCcw,
  XCircle,
  RefreshCw,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { StatusBadge } from "../components/detail/StatusBadge";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { executionService, promptService } from "../services";
import {
  formatExecutionMode,
  type ExecutionRecord,
} from "../types";
import { useDetailSelectionStore, selectionToNodeId } from "../stores/detail-selection-store";
import { DetailActionButtons } from "../components/detail/DetailActionButtons";

export function ExecutionDetailsPage() {
  const selection = useDetailSelectionStore((s) => s.selection);
  const clearSelection = useDetailSelectionStore((s) => s.clearSelection);
  const selectExecution = useDetailSelectionStore((s) => s.selectExecution);
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);
  const executionId = selection?.identifier;
  const nodeId = selectionToNodeId(selection);

  const [actionBusy, setActionBusy] = useState(false);
  const [showTrace, setShowTrace] = useState(false);

  const {
    data: execution,
    error: execError,
    isLoading: execLoading,
    refetch: refetchExec,
  } = useQuery({
    queryKey: ["execution", executionId],
    queryFn: () => executionService.get(executionId!),
    enabled: !!executionId,
    ...defaultQueryOptions,
  });

  const { data: trace } = useQuery({
    queryKey: ["execution", executionId, "prompt-trace"],
    queryFn: () => promptService.getExecutionPromptTrace(executionId!).catch(() => null),
    enabled: !!executionId,
    ...defaultQueryOptions,
  });

  const doAction = async (fn: () => Promise<ExecutionRecord>) => {
    setActionBusy(true);
    try {
      await fn();
      void refetchExec();
    } catch {
      // Error handled by refetch showing stale data
    } finally {
      setActionBusy(false);
    }
  };

  if (execLoading) return <PageLoadingState label="Loading execution..." />;
  if (execError || !execution) {
    return (
      <DetailPageLayout
        header={
          <DetailPageHeader
            entityType="execution"
            title={executionId ?? "Unknown"}
            onClose={clearSelection}
          />
        }
      >
        <div className="mx-auto max-w-3xl">
          <ErrorState
            error={execError as Error | undefined}
            message={`Could not load execution "${executionId}".`}
            onRetry={() => refetchExec()}
          />
        </div>
      </DetailPageLayout>
    );
  }

  const isActive = ["pending", "scheduled", "starting", "in_progress", "running", "needs_review", "validating", "needs_fixup"].includes(execution.status);
  const isTerminal = ["completed", "failed", "canceled"].includes(execution.status);

  const actionButtons = (
    <div className="flex flex-wrap gap-2">
      {isActive && (
        <Button
          variant="destructive"
          size="sm"
          disabled={actionBusy}
          onClick={() => void doAction(() => executionService.cancel(execution.executionId))}
        >
          {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <XCircle className="mr-1 h-3.5 w-3.5" />}
          Cancel
        </Button>
      )}
      {isTerminal && (
        <>
          <Button
            variant="outline"
            size="sm"
            disabled={actionBusy}
            onClick={() => void doAction(() => executionService.retry(execution.executionId))}
          >
            {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <RotateCcw className="mr-1 h-3.5 w-3.5" />}
            Retry
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={actionBusy}
            onClick={() => void doAction(() => executionService.followUp(execution.executionId, { followUpType: "followup", runMode: "new" }))}
          >
            {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="mr-1 h-3.5 w-3.5" />}
            Follow-up
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={actionBusy}
            onClick={() => void doAction(() => executionService.triggerReview(execution.executionId))}
          >
            {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <ClipboardCheck className="mr-1 h-3.5 w-3.5" />}
            Trigger Review
          </Button>
        </>
      )}
    </div>
  );

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="execution"
          title="Execution Details"
          subtitle={execution.executionId}
          status={execution.status}
          onClose={clearSelection}
          actions={actionButtons}
        />
      }
      mobileActions={actionButtons}
      mobileActionsTitle="Execution Actions"
    >
      <div className="mx-auto max-w-3xl space-y-4">
        {/* Status + metadata card */}
        <Card className="space-y-3 p-4">
          <div className="flex items-center justify-between">
            <StatusBadge status={execution.status} />
            <div className="flex items-center gap-1.5">
              {execution.operation && (
                <span className="rounded bg-slate-700/60 px-2 py-0.5 text-xs text-slate-400">{execution.operation}</span>
              )}
              <span className="rounded bg-slate-700/50 px-2 py-0.5 text-xs text-slate-400">
                {formatExecutionMode(execution.mode)}
              </span>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3 text-sm">
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wider">Backlog</p>
              <button
                type="button"
                onClick={() => selectBacklog(execution.backlogKind, execution.backlogName)}
                className="text-cyan-400 hover:text-cyan-300 text-sm text-left"
              >
                {execution.backlogKind}/{execution.backlogName}
              </button>
            </div>
            {execution.startedBy && (
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wider">Started by</p>
                <p className="text-slate-200">{execution.startedBy}</p>
              </div>
            )}
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wider">Created</p>
              <p className="text-slate-200">{formatRelativeTime(execution.createdAt)}</p>
            </div>
            {execution.updatedAt && (
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wider">Updated</p>
                <p className="text-slate-200">{formatRelativeTime(execution.updatedAt)}</p>
              </div>
            )}
            {execution.parentExecutionId && (
              <div className="col-span-2">
                <p className="text-xs text-slate-500 uppercase tracking-wider">Parent Execution</p>
                <button
                  type="button"
                  onClick={() => selectExecution(execution.parentExecutionId!)}
                  className="text-cyan-400 hover:text-cyan-300 text-sm"
                >
                  {execution.parentExecutionId}
                </button>
              </div>
            )}
          </div>

          {execution.failureReason && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3">
              <p className="text-xs text-red-400 font-medium uppercase tracking-wider mb-1">Failure Reason</p>
              <p className="text-sm text-red-200 whitespace-pre-wrap">{execution.failureReason}</p>
            </div>
          )}
        </Card>

        {/* Registry actions */}
        {nodeId && <DetailActionButtons entityType="execution" direction="row" />}

        {/* Prompt Trace */}
        {trace && (
          <Card className="p-4">
            <button
              type="button"
              className="flex w-full items-center justify-between text-sm font-medium text-slate-200"
              onClick={() => setShowTrace((v) => !v)}
            >
              Prompt Trace
              {showTrace ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            </button>
            {showTrace && (
              <div className="mt-3 space-y-3">
                <div>
                  <p className="text-xs text-slate-500 uppercase tracking-wider">Purpose</p>
                  <p className="text-sm text-slate-200">{trace.purpose}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500 uppercase tracking-wider">Prompt</p>
                  <pre className="mt-1 max-h-96 overflow-auto rounded-lg bg-slate-800/60 p-3 text-xs text-slate-300 whitespace-pre-wrap">
                    {trace.prompt}
                  </pre>
                </div>
                {trace.prompt_revision && (
                  <div>
                    <p className="text-xs text-slate-500 uppercase tracking-wider">Revision</p>
                    <pre className="mt-1 max-h-96 overflow-auto rounded-lg bg-slate-800/60 p-3 text-xs text-slate-300 whitespace-pre-wrap">
                      {trace.prompt_revision}
                    </pre>
                  </div>
                )}
                <div className="flex items-center gap-4 text-xs text-slate-400">
                  <span>Captured: {trace.captured_at}</span>
                  {trace.used_fallback && (
                    <span className="rounded bg-amber-500/20 px-2 py-0.5 text-amber-300">Fallback used</span>
                  )}
                </div>
              </div>
            )}
          </Card>
        )}
      </div>
    </DetailPageLayout>
  );
}
