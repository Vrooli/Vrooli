import React, { useEffect, useState, useCallback } from 'react';
import {
  Square,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Clock,
  Loader2,
  Eye,
  RefreshCw,
  Filter,
  Activity,
  Image,
  Timer,
  LineChart,
} from 'lucide-react';
import { useDashboardStore, type RecentExecution } from '@stores/dashboardStore';
import { useExecutionStore } from '@stores/executionStore';
import { formatDistanceToNow } from 'date-fns';
import { TabEmptyState } from './TabEmptyState';
import { ExecutionsEmptyPreview } from './ExecutionsEmptyPreview';

interface ExecutionsTabProps {
  onViewExecution: (executionId: string, workflowId: string) => void;
  onNavigateToHome?: () => void;
  onCreateWorkflow?: () => void;
}

type StatusFilter = 'all' | 'running' | 'completed' | 'failed';

const statusConfig: Record<string, {
  icon: typeof Clock;
  color: string;
  bgColor: string;
  borderColor: string;
  label: string;
  animate?: boolean;
}> = {
  pending: {
    icon: Clock,
    color: 'text-gray-400',
    bgColor: 'bg-gray-500/10',
    borderColor: 'border-gray-500/30',
    label: 'Pending',
  },
  running: {
    icon: Loader2,
    color: 'text-green-400',
    bgColor: 'bg-green-500/10',
    borderColor: 'border-green-500/30',
    label: 'Running',
    animate: true,
  },
  completed: {
    icon: CheckCircle2,
    color: 'text-emerald-400',
    bgColor: 'bg-emerald-500/10',
    borderColor: 'border-emerald-500/30',
    label: 'Completed',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-400',
    bgColor: 'bg-red-500/10',
    borderColor: 'border-red-500/30',
    label: 'Failed',
  },
  cancelled: {
    icon: AlertTriangle,
    color: 'text-amber-400',
    bgColor: 'bg-amber-500/10',
    borderColor: 'border-amber-500/30',
    label: 'Cancelled',
  },
};

export const ExecutionsTab: React.FC<ExecutionsTabProps> = ({
  onViewExecution,
  onNavigateToHome,
  onCreateWorkflow,
}) => {
  const {
    recentExecutions,
    runningExecutions,
    recentWorkflows,
    isLoadingExecutions,
    fetchRecentExecutions,
    fetchRunningExecutions,
  } = useDashboardStore();
  const { stopExecution } = useExecutionStore();
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Auto-refresh running executions every 5 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      if (runningExecutions.length > 0) {
        void fetchRunningExecutions();
      }
    }, 5000);
    return () => clearInterval(interval);
  }, [runningExecutions.length, fetchRunningExecutions]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    await Promise.all([fetchRecentExecutions(), fetchRunningExecutions()]);
    setIsRefreshing(false);
  }, [fetchRecentExecutions, fetchRunningExecutions]);

  const handleStopExecution = useCallback(async (executionId: string) => {
    try {
      await stopExecution(executionId);
      await fetchRunningExecutions();
    } catch (error) {
      console.error('Failed to stop execution:', error);
    }
  }, [stopExecution, fetchRunningExecutions]);

  // Combine and filter executions
  const allExecutions = [...runningExecutions, ...recentExecutions];
  const filteredExecutions = allExecutions.filter((execution) => {
    if (statusFilter === 'all') return true;
    if (statusFilter === 'running') return execution.status === 'running' || execution.status === 'pending';
    if (statusFilter === 'completed') return execution.status === 'completed';
    if (statusFilter === 'failed') return execution.status === 'failed' || execution.status === 'cancelled';
    return true;
  });

  const renderExecutionCard = (execution: RecentExecution, isRunning: boolean) => {
    const config = statusConfig[execution.status];
    const StatusIcon = config.icon;
    const durationMs = execution.completedAt
      ? new Date(execution.completedAt).getTime() - new Date(execution.startedAt).getTime()
      : undefined;
    const durationLabel = durationMs && durationMs > 0 ? `${Math.round(durationMs / 1000)}s` : undefined;
    const logSnippet = execution.error
      ? execution.error.split('\n')[0]
      : execution.status === 'completed'
        ? 'Completed successfully'
        : undefined;

    return (
      <div
        key={execution.id}
        className={`p-4 rounded-lg border transition-colors ${
          isRunning
            ? 'bg-gray-800/80 border-green-500/30 shadow-lg shadow-green-500/5'
            : 'bg-gray-800/50 border-gray-700/50 hover:border-gray-600'
        }`}
      >
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 min-w-0 flex-1">
            <div className={`p-2 rounded-lg ${config.bgColor}`}>
              <StatusIcon
                size={18}
                className={`${config.color} ${config.animate ? 'animate-spin' : ''}`}
              />
            </div>
            <div className="min-w-0 flex-1">
              <div className="font-medium text-surface truncate">
                {execution.workflowName}
              </div>
              {execution.projectName && (
                <div className="text-xs text-gray-500 truncate">
                  {execution.projectName}
                </div>
              )}
              <div className="flex items-center gap-2 mt-1 text-xs text-gray-400">
                <span className={config.color}>{config.label}</span>
                <span>·</span>
                <span>
                  {isRunning
                    ? `Started ${formatDistanceToNow(execution.startedAt, { addSuffix: true })}`
                    : execution.completedAt
                      ? `Completed ${formatDistanceToNow(execution.completedAt, { addSuffix: true })}`
                      : formatDistanceToNow(execution.startedAt, { addSuffix: true })}
                </span>
                {durationLabel && (
                  <>
                    <span>·</span>
                    <span className="inline-flex items-center gap-1 text-xs text-gray-300">
                      <Timer size={12} />
                      {durationLabel}
                    </span>
                  </>
                )}
              </div>
              {logSnippet && (
                <div className={`mt-2 text-xs line-clamp-2 ${
                  execution.error ? 'text-red-400' : 'text-gray-300'
                }`}>
                  {logSnippet}
                </div>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {isRunning && (
              <button
                onClick={() => handleStopExecution(execution.id)}
                className="p-2 text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-lg transition-colors"
                title="Stop execution"
              >
                <Square size={16} />
              </button>
            )}
            <button
              onClick={() => onViewExecution(execution.id, execution.workflowId)}
              className="p-2 text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors"
              title="View details"
            >
              <Eye size={16} />
            </button>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      {/* Pulse strip */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between bg-gradient-to-r from-gray-900/80 via-gray-900 to-blue-900/10 border border-gray-800/80 rounded-2xl p-4">
        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-green-500/10 border border-green-500/30 text-green-100">
            <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            {runningExecutions.length} running
          </div>
          <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-red-500/10 border border-red-500/30 text-red-100">
            <XCircle size={14} />
            {recentExecutions.filter(e => e.status === 'failed' || e.status === 'cancelled').length} failed recently
          </div>
          <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-gray-700/50 text-gray-100 border border-gray-700">
            <LineChart size={14} />
            {allExecutions.length} total entries
          </div>
        </div>
        {allExecutions.length > 0 && (
          <button
            onClick={() => {
              const latest = allExecutions[0];
              onViewExecution(latest.id, latest.workflowId);
            }}
            className="hero-button-secondary w-full sm:w-auto justify-center"
          >
            Open latest run
          </button>
        )}
      </div>

      {/* Header with filters */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="flex items-center gap-2">
          <Filter size={16} className="text-gray-500" />
          <div className="flex items-center gap-1 bg-gray-800/50 rounded-lg p-1">
            {(['all', 'running', 'completed', 'failed'] as StatusFilter[]).map((filter) => (
              <button
                key={filter}
                onClick={() => setStatusFilter(filter)}
                className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                  statusFilter === filter
                    ? 'bg-gray-700 text-white'
                    : 'text-gray-400 hover:text-gray-300'
                }`}
              >
                {filter === 'all' ? 'All' : filter.charAt(0).toUpperCase() + filter.slice(1)}
              </button>
            ))}
          </div>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isRefreshing || isLoadingExecutions}
          className="flex items-center gap-2 px-3 py-1.5 text-sm text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
        >
          <RefreshCw size={14} className={isRefreshing ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {/* Running Executions Section */}
      {runningExecutions.length > 0 && (
        <div className="space-y-3">
          <h3 className="text-sm font-medium text-gray-300 flex items-center gap-2">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            Running Now ({runningExecutions.length})
          </h3>
          <div className="space-y-3">
            {runningExecutions.map((execution) => renderExecutionCard(execution, true))}
          </div>
        </div>
      )}

      {/* Recent Executions Section */}
      <div className="space-y-3">
        <h3 className="text-sm font-medium text-gray-300">
          {statusFilter === 'all' ? 'Recent Executions' : `${statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1)} Executions`}
        </h3>

        {isLoadingExecutions ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 size={24} className="animate-spin text-gray-500" />
          </div>
        ) : filteredExecutions.length === 0 ? (
          <TabEmptyState
            icon={<Activity size={22} />}
            title="Watch your automations come alive"
            subtitle={
              recentWorkflows.length > 0
                ? `You have ${recentWorkflows.length} workflow${recentWorkflows.length !== 1 ? 's' : ''} ready to run.`
                : 'Create a workflow, run it, and monitor every step with logs, screenshots, and metrics.'
            }
            preview={<ExecutionsEmptyPreview />}
            variant="polished"
            primaryCta={{
              label: recentWorkflows.length > 0 ? 'Run a workflow' : 'Create your first workflow',
              onClick:
                (recentWorkflows.length > 0 ? onNavigateToHome : onCreateWorkflow) ??
                onCreateWorkflow ??
                (() => {}),
            }}
            secondaryCta={
              recentWorkflows.length > 0 && onCreateWorkflow
                ? { label: 'Create another workflow', onClick: onCreateWorkflow }
                : undefined
            }
            progressPath={[
              { label: 'Create workflow', completed: recentWorkflows.length > 0 },
              { label: 'Run & monitor', active: true },
              { label: 'Export results' },
            ]}
            features={[
              {
                title: 'Live monitoring',
                description: 'Follow each browser action in real time with streaming logs.',
                icon: <Eye size={16} />,
              },
              {
                title: 'Step-by-step checks',
                description: 'Progress, retries, and errors are captured automatically.',
                icon: <CheckCircle2 size={16} />,
              },
              {
                title: 'Visual evidence',
                description: 'Screenshots attach to every critical step for quick validation.',
                icon: <Image size={16} />,
              },
            ]}
          />
        ) : (
          <div className="space-y-2">
            {filteredExecutions
              .filter((e) => e.status !== 'running' && e.status !== 'pending')
              .map((execution) => renderExecutionCard(execution, false))}
          </div>
        )}
      </div>
    </div>
  );
};

export default ExecutionsTab;
