import React, { useEffect, useState, useCallback } from 'react';
import {
  Play,
  Square,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Clock,
  Loader2,
  Eye,
  RefreshCw,
  Filter,
} from 'lucide-react';
import { useDashboardStore, type RecentExecution } from '@stores/dashboardStore';
import { useExecutionStore } from '@stores/executionStore';
import { formatDistanceToNow } from 'date-fns';

interface ExecutionsTabProps {
  onViewExecution: (executionId: string, workflowId: string) => void;
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

export const ExecutionsTab: React.FC<ExecutionsTabProps> = ({ onViewExecution }) => {
  const {
    recentExecutions,
    runningExecutions,
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
              <div className="font-medium text-white truncate">
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
              </div>
              {execution.error && (
                <div className="mt-2 text-xs text-red-400 line-clamp-2">
                  {execution.error}
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
              className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
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
          className="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
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
          <div className="text-center py-12">
            <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-800 flex items-center justify-center">
              <Play size={24} className="text-gray-600" />
            </div>
            <h4 className="text-lg font-medium text-white mb-2">No executions yet</h4>
            <p className="text-gray-400 text-sm">
              Run a workflow to see execution history here
            </p>
          </div>
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
