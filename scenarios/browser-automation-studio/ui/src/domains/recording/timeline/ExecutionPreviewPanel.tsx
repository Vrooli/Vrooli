/**
 * ExecutionPreviewPanel - Shows execution status and progress in execution mode
 *
 * This panel is shown instead of RecordPreviewPanel when in execution mode.
 * It displays:
 * - Execution status (pending, running, completed, failed)
 * - Progress indicator
 * - Current step being executed
 * - Live screenshots as they come in
 */

import { useCallback, useEffect, useState } from 'react';
import { Play, Loader2, CheckCircle, XCircle, Clock, AlertCircle, RefreshCw } from 'lucide-react';
import { useExecutionStore, type Execution, useExecutionEvents } from '@/domains/executions';
import { useWorkflowStore } from '@stores/workflowStore';

interface ExecutionPreviewPanelProps {
  /** Execution ID to display */
  executionId: string;
  /** Callback when execution starts/restarts */
  onExecutionStart?: () => void;
}

export function ExecutionPreviewPanel({
  executionId,
  onExecutionStart,
}: ExecutionPreviewPanelProps) {
  const currentExecution = useExecutionStore((s) => s.currentExecution);
  const loadExecution = useExecutionStore((s) => s.loadExecution);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Subscribe to WebSocket updates for real-time execution progress
  // This hook bridges WebSocket events to the execution store
  useExecutionEvents(
    currentExecution ? { id: currentExecution.id, status: currentExecution.status } : undefined
  );

  // Get workflow info for display
  const workflows = useWorkflowStore((s) => s.workflows);
  const workflowName = currentExecution?.workflowId
    ? workflows.find((w) => w.id === currentExecution.workflowId)?.name
    : null;

  // Load execution data on mount
  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setError(null);
      try {
        await loadExecution(executionId);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load execution');
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [executionId, loadExecution]);

  const handleRetry = useCallback(() => {
    setError(null);
    setIsLoading(true);
    loadExecution(executionId).catch((err) => {
      setError(err instanceof Error ? err.message : 'Failed to load execution');
      setIsLoading(false);
    });
  }, [executionId, loadExecution]);

  // Loading state
  if (isLoading) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <Loader2 className="w-12 h-12 text-flow-accent animate-spin mb-4" />
        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
          Loading execution...
        </p>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <AlertCircle className="w-12 h-12 text-red-500 mb-4" />
        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
          Failed to load execution
        </p>
        <p className="text-xs text-gray-500 mt-1">{error}</p>
        <button
          onClick={handleRetry}
          className="mt-4 flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-flow-accent/90 transition-colors"
        >
          <RefreshCw size={16} />
          Retry
        </button>
      </div>
    );
  }

  // No execution found
  if (!currentExecution) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <AlertCircle className="w-12 h-12 text-amber-500 mb-4" />
        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
          Execution not found
        </p>
        <p className="text-xs text-gray-500 mt-1">
          The execution may have been deleted or expired
        </p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {workflowName || 'Workflow Execution'}
            </h2>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
              Execution ID: {executionId.slice(0, 8)}...
            </p>
          </div>
          <ExecutionStatusBadge status={currentExecution.status} />
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 overflow-auto">
        <ExecutionContent
          execution={currentExecution}
          onStart={onExecutionStart}
        />
      </div>
    </div>
  );
}

/** Status badge component */
function ExecutionStatusBadge({ status }: { status: Execution['status'] }) {
  const config: Record<string, { icon: typeof Clock; text: string; className: string; spin?: boolean }> = {
    pending: { icon: Clock, text: 'Pending', className: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300' },
    running: { icon: Loader2, text: 'Running', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300', spin: true },
    completed: { icon: CheckCircle, text: 'Completed', className: 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300' },
    failed: { icon: XCircle, text: 'Failed', className: 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300' },
    cancelled: { icon: XCircle, text: 'Cancelled', className: 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300' },
  };

  const { icon: Icon, text, className, spin } = config[status] || config.pending;

  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${className}`}>
      <Icon size={14} className={spin ? 'animate-spin' : ''} />
      {text}
    </span>
  );
}

/** Main execution content based on status */
function ExecutionContent({
  execution,
  onStart,
}: {
  execution: Execution;
  onStart?: () => void;
}) {
  const { status, progress, currentStep, error, screenshots, timeline } = execution;

  // Running state - show progress
  if (status === 'running') {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        {/* Progress circle */}
        <div className="relative w-32 h-32 mb-6">
          <svg className="w-full h-full transform -rotate-90" viewBox="0 0 100 100">
            {/* Background circle */}
            <circle
              cx="50"
              cy="50"
              r="45"
              fill="none"
              stroke="currentColor"
              strokeWidth="8"
              className="text-gray-200 dark:text-gray-700"
            />
            {/* Progress circle */}
            <circle
              cx="50"
              cy="50"
              r="45"
              fill="none"
              stroke="currentColor"
              strokeWidth="8"
              strokeLinecap="round"
              className="text-flow-accent"
              strokeDasharray={`${(progress || 0) * 2.83} 283`}
            />
          </svg>
          <div className="absolute inset-0 flex items-center justify-center">
            <span className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {Math.round(progress || 0)}%
            </span>
          </div>
        </div>

        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
          Execution in progress
        </p>
        {currentStep && (
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {currentStep}
          </p>
        )}

        {/* Latest screenshot if available */}
        {screenshots && screenshots.length > 0 && (
          <div className="mt-6 w-full max-w-md">
            <p className="text-xs text-gray-400 dark:text-gray-500 mb-2 uppercase tracking-wide font-medium">
              Latest Screenshot
            </p>
            <img
              src={screenshots[screenshots.length - 1].url}
              alt="Latest step screenshot"
              className="w-full rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm"
            />
          </div>
        )}
      </div>
    );
  }

  // Completed state
  if (status === 'completed') {
    const stepCount = timeline?.length || 0;
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <CheckCircle className="w-16 h-16 text-green-500 mb-4" />
        <p className="text-lg font-medium text-gray-800 dark:text-gray-200">
          Execution Completed
        </p>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {stepCount} step{stepCount !== 1 ? 's' : ''} executed successfully
        </p>

        {/* Final screenshot if available */}
        {screenshots && screenshots.length > 0 && (
          <div className="mt-6 w-full max-w-md">
            <img
              src={screenshots[screenshots.length - 1].url}
              alt="Final screenshot"
              className="w-full rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm"
            />
          </div>
        )}
      </div>
    );
  }

  // Failed state
  if (status === 'failed') {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <XCircle className="w-16 h-16 text-red-500 mb-4" />
        <p className="text-lg font-medium text-gray-800 dark:text-gray-200">
          Execution Failed
        </p>
        {error && (
          <p className="text-sm text-red-500 dark:text-red-400 mt-2 max-w-md">
            {error}
          </p>
        )}
        {currentStep && (
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
            Failed at: {currentStep}
          </p>
        )}

        {/* Screenshot at failure if available */}
        {screenshots && screenshots.length > 0 && (
          <div className="mt-6 w-full max-w-md">
            <p className="text-xs text-gray-400 dark:text-gray-500 mb-2 uppercase tracking-wide font-medium">
              Screenshot at Failure
            </p>
            <img
              src={screenshots[screenshots.length - 1].url}
              alt="Failure screenshot"
              className="w-full rounded-lg border border-red-200 dark:border-red-800 shadow-sm"
            />
          </div>
        )}

        {onStart && (
          <button
            onClick={onStart}
            className="mt-6 flex items-center gap-2 px-6 py-3 bg-flow-accent text-white rounded-lg hover:bg-flow-accent/90 transition-colors"
          >
            <RefreshCw size={18} />
            Retry Execution
          </button>
        )}
      </div>
    );
  }

  // Cancelled state
  if (status === 'cancelled') {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <XCircle className="w-16 h-16 text-amber-500 mb-4" />
        <p className="text-lg font-medium text-gray-800 dark:text-gray-200">
          Execution Cancelled
        </p>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          The execution was stopped before completion
        </p>

        {onStart && (
          <button
            onClick={onStart}
            className="mt-6 flex items-center gap-2 px-6 py-3 bg-flow-accent text-white rounded-lg hover:bg-flow-accent/90 transition-colors"
          >
            <Play size={18} />
            Run Again
          </button>
        )}
      </div>
    );
  }

  // Pending/default state - show start button
  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6">
      <div className="w-20 h-20 rounded-full bg-flow-accent/10 flex items-center justify-center mb-6">
        <Play className="w-10 h-10 text-flow-accent" />
      </div>
      <p className="text-lg font-medium text-gray-800 dark:text-gray-200">
        Ready to Execute
      </p>
      <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-sm">
        The workflow is ready to run. Click the button below to start execution.
      </p>

      {onStart && (
        <button
          onClick={onStart}
          className="mt-6 flex items-center gap-2 px-6 py-3 bg-flow-accent text-white rounded-lg hover:bg-flow-accent/90 transition-colors shadow-lg shadow-flow-accent/25"
        >
          <Play size={18} />
          Start Execution
        </button>
      )}
    </div>
  );
}
