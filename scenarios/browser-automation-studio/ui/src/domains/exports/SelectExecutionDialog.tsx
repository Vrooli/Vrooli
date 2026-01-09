/**
 * SelectExecutionDialog
 *
 * Modal dialog for selecting an execution to export from the Exports tab.
 * Includes filters for workflow, date range, status, and exportability.
 */

import React, { useMemo } from 'react';
import {
  X,
  Search,
  Film,
  AlertCircle,
  Loader2,
  RotateCcw,
  Timer,
  Eye,
} from 'lucide-react';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import type { ExecutionWithExportability } from '@/domains/executions/store';
import type { UseExecutionFiltersReturn, StatusFilter } from './hooks/useExecutionFilters';
import { formatDistanceToNowStrict } from 'date-fns';
// Import presentation utilities from consolidated module
import { formatExecutionDuration, getExecutionStatusConfig } from './presentation';

interface SelectExecutionDialogProps {
  isOpen: boolean;
  onClose: () => void;
  executions: ExecutionWithExportability[];
  isLoading: boolean;
  filters: UseExecutionFiltersReturn;
  onSelectExecution: (executionId: string, workflowId: string) => void;
  getExportCount: (executionId: string) => number;
}

const isValidDate = (date: Date): boolean => {
  return date instanceof Date && !isNaN(date.getTime());
};

const statusOptions: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'All statuses' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
];

// Status config is now imported from ./presentation module

export const SelectExecutionDialog: React.FC<SelectExecutionDialogProps> = ({
  isOpen,
  onClose,
  executions,
  isLoading,
  filters,
  onSelectExecution,
  getExportCount,
}) => {
  // Apply filters
  const filteredExecutions = useMemo(() => {
    return filters.filterExecutions(executions);
  }, [executions, filters]);

  // Get unique workflows for dropdown
  const uniqueWorkflows = useMemo(() => {
    const workflowMap = new Map<string, string>();
    for (const exec of executions) {
      if (!workflowMap.has(exec.workflowId)) {
        // We don't have workflow names directly, so use the ID for now
        workflowMap.set(exec.workflowId, exec.workflowId);
      }
    }
    return Array.from(workflowMap.entries()).map(([id, name]) => ({ id, name }));
  }, [executions]);

  const handleSelectExecution = (execution: ExecutionWithExportability) => {
    if (!execution.exportability?.isExportable && !filters.filters.showNonExportable) {
      return;
    }
    onSelectExecution(execution.id, execution.workflowId);
  };

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabel="Select execution to export"
      size="xl"
    >
      <div className="flex flex-col h-[80vh] max-h-[700px]">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-700">
          <div>
            <h2 className="text-lg font-semibold text-surface">Select Execution to Export</h2>
            <p className="text-sm text-gray-400 mt-0.5">
              Choose an execution with recorded content to create an export
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 text-gray-400 hover:text-surface hover:bg-gray-700 rounded-md transition-colors"
            title="Close"
          >
            <X size={20} />
          </button>
        </div>

        {/* Filters */}
        <div className="p-4 border-b border-gray-700 space-y-3">
          <div className="flex flex-wrap gap-3">
            {/* Search */}
            <div className="relative flex-1 min-w-[200px]">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
              <input
                type="text"
                placeholder="Search by ID or workflow..."
                value={filters.filters.searchText}
                onChange={(e) => filters.setSearchText(e.target.value)}
                className="w-full pl-9 pr-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-surface placeholder:text-gray-500 focus:outline-none focus:border-flow-accent"
              />
            </div>

            {/* Status filter */}
            <select
              value={filters.filters.status}
              onChange={(e) => filters.setStatusFilter(e.target.value as StatusFilter)}
              className="px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-surface focus:outline-none focus:border-flow-accent"
            >
              {statusOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>

            {/* Workflow filter */}
            {uniqueWorkflows.length > 1 && (
              <select
                value={filters.filters.workflowId || ''}
                onChange={(e) => {
                  const id = e.target.value || null;
                  filters.setWorkflowFilter(id, id || '');
                }}
                className="px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-surface focus:outline-none focus:border-flow-accent max-w-[200px]"
              >
                <option value="">All workflows</option>
                {uniqueWorkflows.map((wf) => (
                  <option key={wf.id} value={wf.id}>
                    {wf.name.slice(0, 20)}...
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Second row: toggles and actions */}
          <div className="flex items-center justify-between">
            <label className="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
              <input
                type="checkbox"
                checked={filters.filters.showNonExportable}
                onChange={filters.toggleShowNonExportable}
                className="w-4 h-4 bg-gray-800 border-gray-600 rounded text-flow-accent focus:ring-flow-accent/50"
              />
              Show non-exportable executions
            </label>

            {filters.hasActiveFilters && (
              <button
                onClick={filters.resetFilters}
                className="flex items-center gap-1.5 px-2 py-1 text-xs text-gray-400 hover:text-surface transition-colors"
              >
                <RotateCcw size={12} />
                Clear filters
              </button>
            )}
          </div>
        </div>

        {/* Execution list */}
        <div className="flex-1 overflow-y-auto p-4">
          {isLoading ? (
            <div className="flex items-center justify-center h-full">
              <Loader2 className="w-8 h-8 text-gray-500 animate-spin" />
            </div>
          ) : filteredExecutions.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-gray-400">
              <AlertCircle size={32} className="mb-2" />
              <p className="text-sm">No executions found</p>
              {!filters.filters.showNonExportable && (
                <p className="text-xs mt-1">Try enabling &quot;Show non-exportable executions&quot;</p>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              {filteredExecutions.map((execution) => {
                const config = getExecutionStatusConfig(execution.status);
                const StatusIcon = config.icon;
                const isExportable = execution.exportability?.isExportable ?? false;
                const exportCount = getExportCount(execution.id);
                const hasExports = exportCount > 0;
                const durationMs = execution.completedAt && isValidDate(execution.startedAt) && isValidDate(execution.completedAt)
                  ? execution.completedAt.getTime() - execution.startedAt.getTime()
                  : undefined;

                return (
                  <button
                    key={execution.id}
                    onClick={() => handleSelectExecution(execution)}
                    disabled={!isExportable}
                    className={`
                      w-full p-4 rounded-lg border text-left transition-all
                      ${isExportable
                        ? 'bg-gray-800/50 border-gray-700/50 hover:bg-gray-700/60 hover:border-gray-600 cursor-pointer'
                        : 'bg-gray-800/20 border-gray-700/30 opacity-50 cursor-not-allowed'
                      }
                    `}
                  >
                    <div className="flex items-start gap-3">
                      {/* Status icon */}
                      <div className={`p-2 rounded-lg ${config.bgColor} flex-shrink-0`}>
                        <StatusIcon
                          size={18}
                          className={`${config.color} ${execution.status === 'running' ? 'animate-spin' : ''}`}
                        />
                      </div>

                      {/* Content */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-surface truncate">
                            {execution.id.slice(0, 8)}...
                          </span>
                          {/* Badges */}
                          <div className="flex items-center gap-1.5">
                            {isExportable && (
                              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 bg-green-500/10 text-green-400 text-xs rounded">
                                <Film size={10} />
                                Exportable
                              </span>
                            )}
                            {hasExports && (
                              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 bg-blue-500/10 text-blue-400 text-xs rounded">
                                <Eye size={10} />
                                {exportCount} export{exportCount !== 1 ? 's' : ''}
                              </span>
                            )}
                          </div>
                        </div>

                        {/* Metadata */}
                        <div className="flex items-center gap-2 mt-1.5 text-xs text-gray-400">
                          <span>{config.label}</span>
                          <span className="text-gray-600">·</span>
                          <span>
                            {isValidDate(execution.startedAt)
                              ? formatDistanceToNowStrict(execution.startedAt, { addSuffix: true })
                              : 'Unknown'
                            }
                          </span>
                          {durationMs !== undefined && (
                            <>
                              <span className="text-gray-600">·</span>
                              <span className="inline-flex items-center gap-1">
                                <Timer size={11} />
                                {formatExecutionDuration(durationMs)}
                              </span>
                            </>
                          )}
                        </div>

                        {/* Exportability details */}
                        {!isExportable && (
                          <div className="mt-2 text-xs text-gray-500">
                            No recorded content available for export
                          </div>
                        )}
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-gray-700 flex items-center justify-between">
          <span className="text-xs text-gray-500">
            {filteredExecutions.length} execution{filteredExecutions.length !== 1 ? 's' : ''} shown
          </span>
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-gray-400 hover:text-surface transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
};

export default SelectExecutionDialog;
