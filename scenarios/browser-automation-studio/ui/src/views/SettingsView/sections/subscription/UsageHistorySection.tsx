import { useState, useEffect, useCallback } from 'react';
import { ChevronLeft, ChevronRight, ChevronDown, ChevronUp, History, Sparkles, Zap, FileOutput, Eye } from 'lucide-react';
import { useEntitlementStore, type UsagePeriod } from '@stores/entitlementStore';
import { LoadingSpinner } from '@shared/ui';

// Operation category helpers
const OPERATION_CATEGORIES = {
  ai: { label: 'AI Operations', icon: Sparkles, color: 'text-purple-400', prefix: 'ai.' },
  execution: { label: 'Executions', icon: Zap, color: 'text-amber-400', prefix: 'execution.' },
  export: { label: 'Exports (Free)', icon: FileOutput, color: 'text-blue-400', prefix: 'export.' },
} as const;

function getCategoryFromOperation(opType: string): keyof typeof OPERATION_CATEGORIES | null {
  if (opType.startsWith('ai.')) return 'ai';
  if (opType.startsWith('execution.')) return 'execution';
  if (opType.startsWith('export.')) return 'export';
  return null;
}

function aggregateByCategory(period: UsagePeriod) {
  const creditsByCategory: Record<string, number> = { ai: 0, execution: 0, export: 0 };
  const opsByCategory: Record<string, number> = { ai: 0, execution: 0, export: 0 };

  if (period.by_operation) {
    for (const [opType, credits] of Object.entries(period.by_operation)) {
      const cat = getCategoryFromOperation(opType);
      if (cat) creditsByCategory[cat] += credits;
    }
  }

  if (period.operation_counts) {
    for (const [opType, count] of Object.entries(period.operation_counts)) {
      const cat = getCategoryFromOperation(opType);
      if (cat) opsByCategory[cat] += count;
    }
  }

  return { creditsByCategory, opsByCategory };
}

interface UsageHistorySectionProps {
  onViewOperations?: (month: string) => void;
}

export function UsageHistorySection({ onViewOperations }: UsageHistorySectionProps) {
  const { usageHistory, historyLoading, fetchUsageHistory } = useEntitlementStore();
  const [isExpanded, setIsExpanded] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);

  // Fetch history on mount
  useEffect(() => {
    fetchUsageHistory(12); // Fetch last 12 months
  }, [fetchUsageHistory]);

  const selectedPeriod = usageHistory[selectedIndex];

  const handlePrevPeriod = useCallback(() => {
    setSelectedIndex((prev) => Math.min(prev + 1, usageHistory.length - 1));
  }, [usageHistory.length]);

  const handleNextPeriod = useCallback(() => {
    setSelectedIndex((prev) => Math.max(prev - 1, 0));
  }, []);

  const formatMonth = (monthStr: string): string => {
    const date = new Date(monthStr + '-01');
    return date.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
  };

  const isCurrentMonth = selectedIndex === 0;
  const canGoBack = selectedIndex < usageHistory.length - 1;
  const canGoForward = selectedIndex > 0;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-surface flex items-center gap-2">
          <History size={20} className="text-flow-accent" />
          Usage History
        </h3>
        <button
          type="button"
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-1 text-sm text-gray-400 hover:text-gray-300 transition-colors"
        >
          {isExpanded ? 'Collapse' : 'Expand'}
          {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </button>
      </div>

      <div className="rounded-xl border border-gray-700 bg-gray-800/50 overflow-hidden">
        {/* Period Navigator */}
        <div className="flex items-center justify-between p-4 border-b border-gray-700 bg-gray-900/30">
          <button
            type="button"
            onClick={handlePrevPeriod}
            disabled={!canGoBack || historyLoading}
            className="p-2 rounded-lg hover:bg-gray-700/50 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            aria-label="Previous period"
          >
            <ChevronLeft size={20} className="text-gray-400" />
          </button>

          <div className="flex items-center gap-3">
            {historyLoading ? (
              <LoadingSpinner size={16} />
            ) : selectedPeriod ? (
              <>
                <span className="text-surface font-medium">
                  {formatMonth(selectedPeriod.billing_month)}
                </span>
                {isCurrentMonth && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-green-900/30 text-green-400 border border-green-700/50">
                    Current
                  </span>
                )}
              </>
            ) : (
              <span className="text-gray-400">No data available</span>
            )}
          </div>

          <button
            type="button"
            onClick={handleNextPeriod}
            disabled={!canGoForward || historyLoading}
            className="p-2 rounded-lg hover:bg-gray-700/50 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            aria-label="Next period"
          >
            <ChevronRight size={20} className="text-gray-400" />
          </button>
        </div>

        {/* Expanded Content */}
        {isExpanded && selectedPeriod && (
          <div className="p-4 space-y-4">
            {/* Summary Stats */}
            <div className="grid grid-cols-2 gap-3">
              <div className="text-center p-3 rounded-lg bg-gray-900/50">
                <div className="text-xs text-gray-400 mb-1">Credits Used</div>
                <div className="text-xl font-bold text-surface">
                  {selectedPeriod.total_credits_used}
                </div>
              </div>
              <div className="text-center p-3 rounded-lg bg-gray-900/50">
                <div className="text-xs text-gray-400 mb-1">Operations</div>
                <div className="text-xl font-bold text-surface">
                  {selectedPeriod.total_operations}
                </div>
              </div>
            </div>

            {/* Category Breakdown */}
            {selectedPeriod.total_operations > 0 && (
              <div className="space-y-2">
                <div className="text-sm text-gray-400 font-medium">By Category</div>
                <div className="rounded-lg border border-gray-700 overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="bg-gray-900/50 text-gray-400 text-xs">
                        <th className="text-left p-2 font-medium">Category</th>
                        <th className="text-right p-2 font-medium">Credits</th>
                        <th className="text-right p-2 font-medium">Operations</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(() => {
                        const { creditsByCategory, opsByCategory } = aggregateByCategory(selectedPeriod);
                        return Object.entries(OPERATION_CATEGORIES).map(([key, config]) => (
                          <tr key={key} className="border-t border-gray-700/50">
                            <td className="p-2">
                              <div className="flex items-center gap-2">
                                <config.icon size={14} className={config.color} />
                                <span className="text-gray-300">{config.label}</span>
                              </div>
                            </td>
                            <td className="p-2 text-right text-gray-300">
                              {creditsByCategory[key]}
                            </td>
                            <td className="p-2 text-right text-gray-300">
                              {opsByCategory[key]}
                            </td>
                          </tr>
                        ));
                      })()}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* View Operations Button */}
            {onViewOperations && selectedPeriod.total_operations > 0 && (
              <button
                type="button"
                onClick={() => onViewOperations(selectedPeriod.billing_month)}
                className="w-full flex items-center justify-center gap-2 p-3 rounded-lg bg-gray-700/30 hover:bg-gray-700/50 text-gray-300 hover:text-surface transition-colors"
              >
                <Eye size={16} />
                View Operations
              </button>
            )}

            {/* Empty State */}
            {selectedPeriod.total_operations === 0 && (
              <div className="text-center text-gray-500 py-4">
                No operations recorded for this period.
              </div>
            )}
          </div>
        )}

        {/* Collapsed Summary */}
        {!isExpanded && selectedPeriod && (
          <div className="p-4 flex items-center justify-between text-sm">
            <div className="text-gray-400">
              <span className="text-surface font-medium">{selectedPeriod.total_credits_used}</span> credits,{' '}
              <span className="text-surface font-medium">{selectedPeriod.total_operations}</span> operations
            </div>
            {onViewOperations && selectedPeriod.total_operations > 0 && (
              <button
                type="button"
                onClick={() => onViewOperations(selectedPeriod.billing_month)}
                className="flex items-center gap-1 text-flow-accent hover:text-flow-accent/80 transition-colors"
              >
                <Eye size={14} />
                View Details
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default UsageHistorySection;
