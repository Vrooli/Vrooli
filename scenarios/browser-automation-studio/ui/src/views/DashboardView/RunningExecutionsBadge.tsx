import React, { useState, useRef, useEffect } from 'react';
import { Play, Loader2, Square, Eye, ChevronDown } from 'lucide-react';
import { useDashboardStore } from '@stores/dashboardStore';
import { useExecutionStore } from '@/domains/executions';
import { formatDistanceToNow } from 'date-fns';

interface RunningExecutionsBadgeProps {
  onViewExecution: (executionId: string, workflowId: string) => void;
  onViewAllExecutions: () => void;
}

export const RunningExecutionsBadge: React.FC<RunningExecutionsBadgeProps> = ({
  onViewExecution,
  onViewAllExecutions,
}) => {
  const { runningExecutions, fetchRunningExecutions } = useDashboardStore();
  const { stopExecution } = useExecutionStore();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  const runningCount = runningExecutions.length;

  // Auto-refresh running executions every 5 seconds when dropdown is open
  useEffect(() => {
    if (!isOpen || runningCount === 0) return;
    const interval = setInterval(() => {
      void fetchRunningExecutions();
    }, 5000);
    return () => clearInterval(interval);
  }, [isOpen, runningCount, fetchRunningExecutions]);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        buttonRef.current &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleStopExecution = async (e: React.MouseEvent, executionId: string) => {
    e.stopPropagation();
    try {
      await stopExecution(executionId);
      await fetchRunningExecutions();
    } catch (error) {
      console.error('Failed to stop execution:', error);
    }
  };

  if (runningCount === 0) {
    return null;
  }

  return (
    <div className="relative">
      {/* Badge Button */}
      <button
        ref={buttonRef}
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-1.5 bg-green-500/20 hover:bg-green-500/30 border border-green-500/30 rounded-lg transition-colors group"
        title={`${runningCount} execution${runningCount !== 1 ? 's' : ''} running`}
      >
        <div className="relative">
          <Play size={14} className="text-green-400" fill="currentColor" />
          <span className="absolute -top-1 -right-1 w-2 h-2 bg-green-400 rounded-full animate-pulse" />
        </div>
        <span className="text-sm font-medium text-green-300">
          {runningCount} running
        </span>
        <ChevronDown
          size={14}
          className={`text-green-400 transition-transform ${isOpen ? 'rotate-180' : ''}`}
        />
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div
          ref={dropdownRef}
          className="absolute right-0 top-full mt-2 z-50 w-80 bg-gray-800 border border-gray-700 rounded-lg shadow-xl overflow-hidden"
        >
          {/* Header */}
          <div className="px-4 py-3 bg-gray-800/80 border-b border-gray-700">
            <h3 className="text-sm font-medium text-surface flex items-center gap-2">
              <Loader2 size={14} className="text-green-400 animate-spin" />
              Running Executions
            </h3>
          </div>

          {/* Executions List */}
          <div className="max-h-80 overflow-y-auto">
            {runningExecutions.map((execution) => (
              <div
                key={execution.id}
                className="px-4 py-3 border-b border-gray-700/50 hover:bg-gray-700/50 transition-colors"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="font-medium text-surface text-sm truncate">
                      {execution.workflowName}
                    </div>
                    {execution.projectName && (
                      <div className="text-xs text-gray-500 truncate">
                        {execution.projectName}
                      </div>
                    )}
                    <div className="text-xs text-gray-400 mt-1">
                      Started {formatDistanceToNow(execution.startedAt, { addSuffix: true })}
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => {
                        onViewExecution(execution.id, execution.workflowId);
                        setIsOpen(false);
                      }}
                      className="p-1.5 text-subtle hover:text-surface hover:bg-gray-600 rounded transition-colors"
                      title="View execution"
                    >
                      <Eye size={14} />
                    </button>
                    <button
                      onClick={(e) => handleStopExecution(e, execution.id)}
                      className="p-1.5 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded transition-colors"
                      title="Stop execution"
                    >
                      <Square size={14} />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Footer */}
          <div className="px-4 py-3 bg-gray-800/80 border-t border-gray-700">
            <button
              onClick={() => {
                onViewAllExecutions();
                setIsOpen(false);
              }}
              className="w-full text-center text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              View all executions
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default RunningExecutionsBadge;
