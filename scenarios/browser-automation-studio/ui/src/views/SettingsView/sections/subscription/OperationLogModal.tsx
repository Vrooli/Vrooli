import { useState, useEffect, useCallback } from 'react';
import { X, Sparkles, Zap, FileOutput, Check, AlertCircle, RefreshCw } from 'lucide-react';
import { useEntitlementStore, type OperationLogEntry } from '@stores/entitlementStore';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import { LoadingSpinner } from '@shared/ui';
import { formatDistanceToNow, format } from 'date-fns';

// Filter options
type FilterCategory = 'all' | 'ai' | 'execution' | 'export';

const FILTER_OPTIONS: { value: FilterCategory; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'ai', label: 'AI' },
  { value: 'execution', label: 'Executions' },
  { value: 'export', label: 'Exports' },
];

// Operation type display helpers
const OPERATION_LABELS: Record<string, { label: string; icon: typeof Sparkles; color: string }> = {
  'ai.workflow_generate': { label: 'AI Workflow Generate', icon: Sparkles, color: 'text-purple-400' },
  'ai.workflow_modify': { label: 'AI Workflow Modify', icon: Sparkles, color: 'text-purple-400' },
  'ai.element_analyze': { label: 'AI Element Analyze', icon: Sparkles, color: 'text-purple-400' },
  'ai.vision_navigate': { label: 'AI Vision Navigate', icon: Sparkles, color: 'text-purple-400' },
  'ai.caption_generate': { label: 'AI Caption Generate', icon: Sparkles, color: 'text-purple-400' },
  'execution.run': { label: 'Workflow Execution', icon: Zap, color: 'text-amber-400' },
  'execution.scheduled': { label: 'Scheduled Execution', icon: Zap, color: 'text-amber-400' },
  'export.video': { label: 'Video Export', icon: FileOutput, color: 'text-blue-400' },
  'export.gif': { label: 'GIF Export', icon: FileOutput, color: 'text-blue-400' },
  'export.html': { label: 'HTML Export', icon: FileOutput, color: 'text-blue-400' },
  'export.json': { label: 'JSON Export', icon: FileOutput, color: 'text-blue-400' },
};

function getOperationDisplay(opType: string) {
  const display = OPERATION_LABELS[opType];
  if (display) return display;

  // Fallback for unknown operation types
  if (opType.startsWith('ai.')) return { label: opType, icon: Sparkles, color: 'text-purple-400' };
  if (opType.startsWith('execution.')) return { label: opType, icon: Zap, color: 'text-amber-400' };
  if (opType.startsWith('export.')) return { label: opType, icon: FileOutput, color: 'text-blue-400' };

  return { label: opType, icon: Zap, color: 'text-gray-400' };
}

interface OperationRowProps {
  operation: OperationLogEntry;
}

function OperationRow({ operation }: OperationRowProps) {
  const display = getOperationDisplay(operation.operation_type);
  const Icon = display.icon;
  const createdAt = new Date(operation.created_at);

  return (
    <div className="flex items-center justify-between p-3 border-b border-gray-700/50 last:border-0 hover:bg-gray-800/30 transition-colors">
      <div className="flex items-center gap-3 flex-1 min-w-0">
        <div className={`p-1.5 rounded-md bg-gray-800 ${display.color}`}>
          <Icon size={14} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-sm text-surface truncate">{display.label}</div>
          <div className="text-xs text-gray-500" title={format(createdAt, 'PPpp')}>
            {formatDistanceToNow(createdAt, { addSuffix: true })}
          </div>
        </div>
      </div>

      <div className="flex items-center gap-3 text-sm">
        <div className="text-gray-400 whitespace-nowrap">
          {operation.credits_charged === 0 ? (
            <span className="text-gray-500">Free</span>
          ) : (
            <span>{operation.credits_charged} credit{operation.credits_charged !== 1 ? 's' : ''}</span>
          )}
        </div>
        <div className={`flex items-center gap-1 ${operation.success ? 'text-green-400' : 'text-red-400'}`}>
          {operation.success ? <Check size={14} /> : <AlertCircle size={14} />}
        </div>
      </div>
    </div>
  );
}

interface OperationLogModalProps {
  isOpen: boolean;
  onClose: () => void;
  month: string;
}

export function OperationLogModal({ isOpen, onClose, month }: OperationLogModalProps) {
  const {
    operationLog,
    operationLogLoading,
    operationLogTotal,
    operationLogHasMore,
    fetchOperationLog,
    clearOperationLog,
  } = useEntitlementStore();

  const [filter, setFilter] = useState<FilterCategory>('all');

  // Format month for display
  const formatMonth = (monthStr: string): string => {
    // Parse YYYY-MM directly to avoid timezone issues
    const [year, month] = monthStr.split('-').map(Number);
    const date = new Date(year, month - 1, 1); // month is 0-indexed
    return date.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
  };

  // Fetch operations when modal opens or filter changes
  useEffect(() => {
    if (isOpen && month) {
      const category = filter === 'all' ? undefined : filter;
      fetchOperationLog(month, category, 20, 0);
    }
    return () => {
      if (!isOpen) {
        clearOperationLog();
      }
    };
  }, [isOpen, month, filter, fetchOperationLog, clearOperationLog]);

  const handleLoadMore = useCallback(() => {
    if (operationLogHasMore && !operationLogLoading) {
      const category = filter === 'all' ? undefined : filter;
      fetchOperationLog(month, category, 20, operationLog.length);
    }
  }, [operationLogHasMore, operationLogLoading, filter, month, operationLog.length, fetchOperationLog]);

  const handleFilterChange = (newFilter: FilterCategory) => {
    setFilter(newFilter);
    clearOperationLog();
  };

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabel="Operation log"
    >
      <div className="relative max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between mb-4 flex-shrink-0">
          <div>
            <h2 className="text-lg font-semibold text-surface">
              Operations - {formatMonth(month)}
            </h2>
            <p className="text-sm text-gray-400">
              {operationLogTotal} operation{operationLogTotal !== 1 ? 's' : ''} recorded
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-1 text-gray-400 hover:text-gray-200 transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Filter Tabs */}
        <div className="flex gap-2 mb-4 flex-shrink-0 overflow-x-auto pb-1">
          {FILTER_OPTIONS.map((option) => (
            <button
              key={option.value}
              onClick={() => handleFilterChange(option.value)}
              className={`
                px-3 py-1.5 rounded-lg text-sm whitespace-nowrap transition-colors
                ${filter === option.value
                  ? 'bg-flow-accent/20 text-flow-accent border border-flow-accent/50'
                  : 'bg-gray-800/50 text-gray-400 hover:text-gray-300 border border-gray-700'
                }
              `}
            >
              {option.label}
            </button>
          ))}
        </div>

        {/* Operations List */}
        <div className="flex-1 overflow-y-auto rounded-lg border border-gray-700 bg-gray-800/30">
          {operationLogLoading && operationLog.length === 0 ? (
            <div className="flex items-center justify-center p-8">
              <LoadingSpinner size={24} />
            </div>
          ) : operationLog.length === 0 ? (
            <div className="text-center text-gray-500 p-8">
              No operations found for this period.
            </div>
          ) : (
            <>
              {operationLog.map((op) => (
                <OperationRow key={op.id} operation={op} />
              ))}

              {/* Load More */}
              {operationLogHasMore && (
                <div className="p-3 text-center border-t border-gray-700">
                  <button
                    onClick={handleLoadMore}
                    disabled={operationLogLoading}
                    className="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-gray-700/50 hover:bg-gray-700 text-gray-300 text-sm transition-colors disabled:opacity-50"
                  >
                    {operationLogLoading ? (
                      <LoadingSpinner size={14} />
                    ) : (
                      <RefreshCw size={14} />
                    )}
                    Load More
                  </button>
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between mt-4 pt-3 border-t border-gray-700 flex-shrink-0 text-xs text-gray-500">
          <span>
            Showing {operationLog.length} of {operationLogTotal}
          </span>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-300 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default OperationLogModal;
