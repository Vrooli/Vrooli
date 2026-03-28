/**
 * Execution Details Page
 *
 * Displays detailed information about a single execution record including:
 * - Execution metadata (status, mode, operation, timestamps)
 * - Prompt trace (if available)
 * - Action buttons (retry, cancel, follow-up, trigger review)
 * - Navigation back to the graph workspace
 */

import { useEffect, useState } from "react";
import { useParams, useSearchParams, Link } from "react-router-dom";
import {
  ArrowLeft,
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
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { cn, formatRelativeTime } from "../lib";
import { executionService, promptService } from "../services";
import {
  EXECUTION_STATUS_COLORS,
  formatExecutionMode,
  formatExecutionStatus,
  type ExecutionRecord,
  type PromptTrace,
} from "../types";

export function ExecutionDetailsPage() {
  const { executionId } = useParams<{ executionId: string }>();
  const [searchParams] = useSearchParams();
  const returnTo = searchParams.get("returnTo") ?? "/graph";

  const [execution, setExecution] = useState<ExecutionRecord | null>(null);
  const [trace, setTrace] = useState<PromptTrace | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionBusy, setActionBusy] = useState(false);
  const [showTrace, setShowTrace] = useState(false);

  useEffect(() => {
    if (!executionId) return;
    setLoading(true);
    setError(null);
    Promise.all([
      executionService.get(executionId),
      promptService.getExecutionPromptTrace(executionId).catch(() => null),
    ])
      .then(([exec, t]) => {
        setExecution(exec);
        setTrace(t);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load execution");
      })
      .finally(() => setLoading(false));
  }, [executionId]);

  const doAction = async (fn: () => Promise<ExecutionRecord>) => {
    setActionBusy(true);
    try {
      const updated = await fn();
      setExecution(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Action failed");
    } finally {
      setActionBusy(false);
    }
  };

  if (loading) return <PageLoadingState message="Loading execution..." />;
  if (error || !execution) {
    return (
      <div className="flex h-screen flex-col items-center justify-center bg-slate-950 p-8">
        <ErrorState message={error ?? "Execution not found"} />
        <Link to={returnTo} className="mt-4 text-sm text-cyan-400 hover:text-cyan-300">
          Back to graph
        </Link>
      </div>
    );
  }

  const isActive = ["pending", "scheduled", "starting", "in_progress", "running", "needs_review", "validating", "needs_fixup"].includes(execution.status);
  const isTerminal = ["completed", "failed", "canceled"].includes(execution.status);
  const statusColor = EXECUTION_STATUS_COLORS[execution.status] ?? "bg-slate-500";

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {/* Header */}
      <header className="border-b border-slate-200/20 px-4 py-3 md:px-6">
        <div className="flex items-center gap-3">
          <Link
            to={returnTo}
            className="rounded p-1 text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div className="min-w-0">
            <h1 className="text-lg font-semibold truncate">Execution Details</h1>
            <p className="text-xs text-slate-400 font-mono truncate">{execution.executionId}</p>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-3xl p-4 md:p-6 space-y-4">
        {/* Status + metadata card */}
        <Card className="space-y-3 p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className={cn("h-2.5 w-2.5 rounded-full", statusColor)} />
              <span className="text-sm font-medium">{formatExecutionStatus(execution.status)}</span>
            </div>
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
              <p className="text-slate-200">{execution.backlogKind}/{execution.backlogName}</p>
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
                <Link
                  to={`/details/execution/${execution.parentExecutionId}?returnTo=${encodeURIComponent(returnTo)}`}
                  className="text-cyan-400 hover:text-cyan-300 text-sm"
                >
                  {execution.parentExecutionId}
                </Link>
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

        {/* Actions */}
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
      </main>
    </div>
  );
}
