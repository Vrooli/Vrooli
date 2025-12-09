import React, { useEffect, useState } from 'react';
import { useDashboardStore, RecentExecution } from '../../stores/dashboardStore';

interface RunningIndicatorProps {
  onViewExecution: (executionId: string, workflowId: string) => void;
}

export const RunningIndicator: React.FC<RunningIndicatorProps> = ({ onViewExecution }) => {
  const runningExecutions = useDashboardStore((state) => state.runningExecutions);
  const fetchRunningExecutions = useDashboardStore((state) => state.fetchRunningExecutions);
  const [isExpanded, setIsExpanded] = useState(false);

  useEffect(() => {
    // Initial fetch
    fetchRunningExecutions();

    // Poll every 5 seconds
    const interval = setInterval(() => {
      fetchRunningExecutions();
    }, 5000);

    return () => clearInterval(interval);
  }, [fetchRunningExecutions]);

  if (runningExecutions.length === 0) {
    return null;
  }

  const handleClick = (execution: RecentExecution) => {
    setIsExpanded(false);
    onViewExecution(execution.id, execution.workflowId);
  };

  return (
    <div className="relative">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center gap-2 px-3 py-1.5 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 rounded-lg transition-colors"
      >
        <div className="relative flex items-center justify-center w-4 h-4">
          <div className="absolute inset-0 bg-blue-400 rounded-full animate-ping opacity-50" />
          <div className="w-2.5 h-2.5 bg-blue-400 rounded-full" />
        </div>
        <span className="text-sm text-blue-300 font-medium">
          {runningExecutions.length} running
        </span>
        <svg
          className={`w-4 h-4 text-blue-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isExpanded && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsExpanded(false)}
          />

          {/* Dropdown */}
          <div className="absolute right-0 top-full mt-2 w-72 bg-gray-900 border border-gray-700 rounded-lg shadow-xl z-50 overflow-hidden">
            <div className="px-3 py-2 border-b border-gray-700 bg-gray-800/50">
              <span className="text-xs text-gray-400 uppercase tracking-wide font-medium">
                Running Executions
              </span>
            </div>
            <div className="max-h-64 overflow-y-auto">
              {runningExecutions.map((execution) => (
                <div
                  key={execution.id}
                  onClick={() => handleClick(execution)}
                  className="flex items-center gap-3 px-3 py-2 hover:bg-gray-800 cursor-pointer transition-colors"
                >
                  <div className="w-5 h-5 flex items-center justify-center">
                    <div className="w-3 h-3 border-2 border-blue-400 border-t-transparent rounded-full animate-spin" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm text-surface truncate">{execution.workflowName}</div>
                    {execution.projectName && (
                      <div className="text-xs text-gray-500 truncate">
                        in {execution.projectName}
                      </div>
                    )}
                  </div>
                  <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
};
