/**
 * WorkflowCard Component
 *
 * A reusable card component for displaying workflow information.
 * Used in both the project detail grid view and workflow picker modal.
 */

import { useCallback, memo } from 'react';
import {
  FileCode,
  Play,
  Clock,
  Loader,
  Trash2,
  CheckSquare,
  Square,
  MoreVertical,
  PlayCircle,
} from 'lucide-react';
import type { WorkflowWithStats } from './hooks/useProjectDetailStore';

interface WorkflowCardProps {
  workflow: WorkflowWithStats;
  onClick?: (workflow: WorkflowWithStats) => void;
  /** Hide action menu (run/delete buttons) */
  hideActions?: boolean;
  /** Selection mode state */
  selectionMode?: boolean;
  isSelected?: boolean;
  onToggleSelection?: (workflowId: string) => void;
  /** Execution/deletion state */
  isDeleting?: boolean;
  isExecuting?: boolean;
  /** Action handlers */
  onRun?: (e: React.MouseEvent, workflowId: string) => void;
  onDelete?: (e: React.MouseEvent, workflowId: string, workflowName: string) => void;
  /** Actions dropdown state */
  isActionsOpen?: boolean;
  onToggleActions?: (workflowId: string | null) => void;
  /** Test ID for testing */
  testId?: string;
}

/**
 * Format date for display
 */
function formatDate(dateString: string): string {
  if (!dateString) return '';
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

/**
 * WorkflowCard displays a workflow with stats, description, and optional actions.
 */
export const WorkflowCard = memo(function WorkflowCard({
  workflow,
  onClick,
  hideActions = false,
  selectionMode = false,
  isSelected = false,
  onToggleSelection,
  isDeleting = false,
  isExecuting = false,
  onRun,
  onDelete,
  isActionsOpen = false,
  onToggleActions,
  testId,
}: WorkflowCardProps) {
  const executionCount = workflow.stats?.execution_count || 0;
  const successRate = workflow.stats?.success_rate;

  const handleClick = useCallback(() => {
    if (isDeleting) return;
    if (selectionMode && onToggleSelection) {
      onToggleSelection(workflow.id);
    } else if (onClick) {
      onClick(workflow);
    }
  }, [isDeleting, selectionMode, onToggleSelection, onClick, workflow]);

  const handleToggleSelection = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onToggleSelection?.(workflow.id);
    },
    [onToggleSelection, workflow.id]
  );

  const handleToggleActions = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onToggleActions?.(isActionsOpen ? null : workflow.id);
    },
    [onToggleActions, isActionsOpen, workflow.id]
  );

  const handleCloseActions = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onToggleActions?.(null);
    },
    [onToggleActions]
  );

  const handleRun = useCallback(
    (e: React.MouseEvent) => {
      onToggleActions?.(null);
      onRun?.(e, workflow.id);
    },
    [onRun, onToggleActions, workflow.id]
  );

  const handleDelete = useCallback(
    (e: React.MouseEvent) => {
      onDelete?.(e, workflow.id, workflow.name);
    },
    [onDelete, workflow.id, workflow.name]
  );

  return (
    <div
      data-testid={testId}
      data-workflow-id={workflow.id}
      data-workflow-name={workflow.name}
      onClick={handleClick}
      className={`group relative bg-flow-node border rounded-xl p-5 cursor-pointer transition-all ${
        isDeleting
          ? 'opacity-50 pointer-events-none'
          : selectionMode
            ? isSelected
              ? 'border-flow-accent shadow-lg shadow-blue-500/20'
              : 'border-gray-700 hover:border-flow-accent/60'
            : 'border-gray-700 hover:border-flow-accent/60 hover:shadow-lg hover:shadow-blue-500/10'
      }`}
    >
      {/* Deleting Overlay */}
      {isDeleting && (
        <div className="absolute inset-0 bg-flow-node/80 rounded-xl flex items-center justify-center z-10">
          <Loader size={24} className="animate-spin text-red-400" />
        </div>
      )}

      {/* Workflow Header */}
      <div className="flex items-start justify-between gap-3 mb-3" data-workflow-header>
        <div className="flex items-center gap-3 min-w-0">
          <div
            className={`flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center border ${
              selectionMode && isSelected
                ? 'bg-flow-accent/20 border-flow-accent/30'
                : 'bg-gradient-to-br from-green-500/20 to-emerald-500/10 border-green-500/20'
            }`}
          >
            <FileCode size={18} className="text-green-400" />
          </div>
          <div className="min-w-0">
            <h3 className="font-semibold text-surface truncate" title={String(workflow.name)}>
              {String(workflow.name)}
            </h3>
            <div className="flex items-center gap-2 text-xs text-gray-500 mt-0.5">
              <span>
                {executionCount} run{executionCount !== 1 ? 's' : ''}
              </span>
              {successRate != null && (
                <>
                  <span>•</span>
                  <span
                    className={
                      successRate >= 80
                        ? 'text-green-500'
                        : successRate >= 50
                          ? 'text-amber-500'
                          : 'text-red-400'
                    }
                  >
                    {successRate}% success
                  </span>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Selection checkbox or action menu */}
        {selectionMode ? (
          <button
            onClick={handleToggleSelection}
            className="flex-shrink-0 p-2 text-gray-300 hover:text-surface transition-colors"
            title={isSelected ? 'Deselect workflow' : 'Select workflow'}
          >
            {isSelected ? <CheckSquare size={16} /> : <Square size={16} />}
          </button>
        ) : !hideActions && onRun && onDelete ? (
          <div className="relative flex-shrink-0">
            <button
              onClick={handleToggleActions}
              className="p-1.5 text-gray-500 hover:text-surface hover:bg-gray-700 rounded-lg transition-colors opacity-0 group-hover:opacity-100 focus:opacity-100"
              aria-label="Workflow actions"
              aria-expanded={isActionsOpen}
            >
              <MoreVertical size={16} />
            </button>

            {isActionsOpen && (
              <>
                <div className="fixed inset-0 z-20" onClick={handleCloseActions} />
                <div className="absolute right-0 top-full mt-1 z-30 w-44 bg-flow-node border border-gray-700 rounded-lg shadow-xl overflow-hidden animate-fade-in">
                  <button
                    onClick={handleRun}
                    disabled={isExecuting}
                    className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-gray-300 hover:bg-gray-700/50 hover:text-surface transition-colors disabled:opacity-50"
                  >
                    {isExecuting ? (
                      <Loader size={14} className="animate-spin" />
                    ) : (
                      <PlayCircle size={14} />
                    )}
                    Run Workflow
                  </button>
                  <div className="border-t border-gray-700" />
                  <button
                    onClick={handleDelete}
                    className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
                  >
                    <Trash2 size={14} />
                    Delete
                  </button>
                </div>
              </>
            )}
          </div>
        ) : null}
      </div>

      {/* Workflow Description */}
      {workflow.description ? (
        <p
          className={`text-sm mb-4 line-clamp-2 ${
            selectionMode && isSelected ? 'text-gray-200' : 'text-gray-400'
          }`}
        >
          {(workflow.description as string | undefined) || ''}
        </p>
      ) : (
        <p className="text-gray-600 text-sm mb-4 italic">No description</p>
      )}

      {/* Workflow Stats */}
      {(executionCount > 0 || successRate != null) && (
        <div className="grid grid-cols-2 gap-3 mb-4">
          <div className="bg-gray-800/50 rounded-lg px-3 py-2 text-center">
            <div className="text-lg font-semibold text-surface">{executionCount}</div>
            <div className="text-xs text-gray-500">Executions</div>
          </div>
          <div className="bg-gray-800/50 rounded-lg px-3 py-2 text-center">
            <div
              className={`text-lg font-semibold ${
                successRate != null
                  ? successRate >= 80
                    ? 'text-green-400'
                    : successRate >= 50
                      ? 'text-amber-400'
                      : 'text-red-400'
                  : 'text-gray-500'
              }`}
            >
              {successRate != null ? `${successRate}%` : '—'}
            </div>
            <div className="text-xs text-gray-500">Success</div>
          </div>
        </div>
      )}

      {/* Footer */}
      <div
        className={`flex items-center justify-between text-xs pt-3 border-t border-gray-700/50 ${
          selectionMode && isSelected ? 'text-gray-200' : 'text-gray-500'
        }`}
      >
        <div className="flex items-center gap-1.5">
          <Clock size={12} />
          <span>Updated {formatDate(workflow.updated_at || '')}</span>
        </div>
        {workflow.stats?.last_execution && (
          <div className="flex items-center gap-1.5 text-green-500/80">
            <Play size={10} />
            <span>{formatDate(workflow.stats.last_execution)}</span>
          </div>
        )}
      </div>
    </div>
  );
});
