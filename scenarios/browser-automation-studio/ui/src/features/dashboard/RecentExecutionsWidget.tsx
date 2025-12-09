import React, { useEffect } from 'react';
import { useDashboardStore, RecentExecution } from '../../stores/dashboardStore';

interface RecentExecutionsWidgetProps {
  onViewExecution: (executionId: string, workflowId: string) => void;
  onViewAll: () => void;
}

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
};

const StatusIcon: React.FC<{ status: RecentExecution['status'] }> = ({ status }) => {
  switch (status) {
    case 'completed':
      return (
        <div className="flex items-center justify-center w-6 h-6 bg-green-500/20 rounded-full">
          <svg className="w-3.5 h-3.5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
          </svg>
        </div>
      );
    case 'failed':
      return (
        <div className="flex items-center justify-center w-6 h-6 bg-red-500/20 rounded-full">
          <svg className="w-3.5 h-3.5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </div>
      );
    case 'running':
      return (
        <div className="flex items-center justify-center w-6 h-6 bg-blue-500/20 rounded-full">
          <div className="w-3 h-3 border-2 border-blue-400 border-t-transparent rounded-full animate-spin" />
        </div>
      );
    case 'pending':
      return (
        <div className="flex items-center justify-center w-6 h-6 bg-yellow-500/20 rounded-full">
          <svg className="w-3.5 h-3.5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
      );
    case 'cancelled':
      return (
        <div className="flex items-center justify-center w-6 h-6 bg-gray-500/20 rounded-full">
          <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
          </svg>
        </div>
      );
    default:
      return null;
  }
};

export const RecentExecutionsWidget: React.FC<RecentExecutionsWidgetProps> = ({ onViewExecution, onViewAll }) => {
  const recentExecutions = useDashboardStore((state) => state.recentExecutions);
  const runningExecutions = useDashboardStore((state) => state.runningExecutions);
  const isLoading = useDashboardStore((state) => state.isLoadingExecutions);
  const fetchRecentExecutions = useDashboardStore((state) => state.fetchRecentExecutions);

  useEffect(() => {
    fetchRecentExecutions();

    // Poll for updates every 10 seconds when there are running executions
    const interval = setInterval(() => {
      fetchRecentExecutions();
    }, 10000);

    return () => clearInterval(interval);
  }, [fetchRecentExecutions]);

  // Combine running and recent, with running first
  const displayExecutions = [
    ...runningExecutions,
    ...recentExecutions.filter(e => !runningExecutions.find(r => r.id === e.id)),
  ].slice(0, 5);

  if (isLoading && displayExecutions.length === 0) {
    return (
      <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-gray-300">Recent Executions</h3>
        </div>
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="animate-pulse flex items-center gap-3 p-2">
              <div className="w-6 h-6 bg-gray-700 rounded-full" />
              <div className="flex-1">
                <div className="h-4 bg-gray-700 rounded w-3/4 mb-1" />
                <div className="h-3 bg-gray-700/50 rounded w-1/2" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (displayExecutions.length === 0) {
    return (
      <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-gray-300">Recent Executions</h3>
        </div>
        <div className="text-center py-6 text-gray-500 text-sm">
          <svg className="w-8 h-8 mx-auto mb-2 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          No executions yet
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium text-gray-300">Recent Executions</h3>
          {runningExecutions.length > 0 && (
            <span className="flex items-center gap-1 px-1.5 py-0.5 bg-blue-500/20 text-blue-400 text-xs rounded-full">
              <span className="w-1.5 h-1.5 bg-blue-400 rounded-full animate-pulse" />
              {runningExecutions.length} running
            </span>
          )}
        </div>
        <button
          onClick={onViewAll}
          className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
        >
          View all
        </button>
      </div>
      <div className="space-y-1">
        {displayExecutions.map((execution) => (
          <div
            key={execution.id}
            onClick={() => onViewExecution(execution.id, execution.workflowId)}
            className="group flex items-center gap-3 p-2 rounded-md hover:bg-gray-700/50 cursor-pointer transition-colors"
          >
            <StatusIcon status={execution.status} />
            <div className="flex-1 min-w-0">
              <div className="text-sm text-surface truncate">{execution.workflowName}</div>
              <div className="text-xs text-gray-500 truncate">
                {execution.status === 'running' ? (
                  <span className="text-blue-400">Running...</span>
                ) : execution.status === 'failed' && execution.error ? (
                  <span className="text-red-400 truncate">{execution.error}</span>
                ) : (
                  formatRelativeTime(execution.completedAt ?? execution.startedAt)
                )}
              </div>
            </div>
            <svg className="w-4 h-4 text-gray-500 group-hover:text-gray-300 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </div>
        ))}
      </div>
    </div>
  );
};
