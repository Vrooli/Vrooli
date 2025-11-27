import { cn } from '@/lib/utils';
import type { InsightReport } from '@/types/api';
import { Calendar, TrendingUp } from 'lucide-react';
import { ExecutionStats } from './ExecutionStats';
import { PatternCard } from './PatternCard';
import { SuggestionCard } from './SuggestionCard';

interface InsightReportViewerProps {
  report: InsightReport;
  onApplySuggestion?: (suggestionId: string) => void;
  onRejectSuggestion?: (suggestionId: string) => void;
  applyingId?: string;
  className?: string;
}

export function InsightReportViewer({
  report,
  onApplySuggestion,
  onRejectSuggestion,
  applyingId,
  className,
}: InsightReportViewerProps) {
  const patternsBySeverity = {
    critical: report.patterns.filter((p) => p.severity === 'critical'),
    high: report.patterns.filter((p) => p.severity === 'high'),
    medium: report.patterns.filter((p) => p.severity === 'medium'),
    low: report.patterns.filter((p) => p.severity === 'low'),
  };

  const suggestionsByPriority = {
    critical: report.suggestions.filter((s) => s.priority === 'critical'),
    high: report.suggestions.filter((s) => s.priority === 'high'),
    medium: report.suggestions.filter((s) => s.priority === 'medium'),
    low: report.suggestions.filter((s) => s.priority === 'low'),
  };

  const pendingSuggestions = report.suggestions.filter((s) => s.status === 'pending');
  const appliedSuggestions = report.suggestions.filter((s) => s.status === 'applied');

  return (
    <div className={cn('space-y-6', className)}>
      {/* Header */}
      <div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700/50">
        <div className="flex items-center gap-3 mb-3">
          <TrendingUp className="w-5 h-5 text-blue-400" />
          <h2 className="text-lg font-semibold text-slate-100">Insight Report</h2>
        </div>
        <div className="flex items-center gap-4 text-xs text-slate-400">
          <div className="flex items-center gap-1.5">
            <Calendar className="w-3.5 h-3.5" />
            <span>{new Date(report.generated_at).toLocaleString()}</span>
          </div>
          <span>•</span>
          <span>{report.execution_count} executions analyzed</span>
          {report.analysis_window.status_filter && (
            <>
              <span>•</span>
              <span>Filter: {report.analysis_window.status_filter}</span>
            </>
          )}
        </div>
      </div>

      {/* Statistics */}
      <div>
        <h3 className="text-sm font-semibold text-slate-200 mb-3">Execution Statistics</h3>
        <ExecutionStats stats={report.statistics} />
      </div>

      {/* Patterns */}
      {report.patterns.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-slate-200 mb-3">
            Patterns Detected ({report.patterns.length})
          </h3>
          <div className="space-y-4">
            {patternsBySeverity.critical.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-red-400 mb-2 uppercase">Critical</h4>
                <div className="space-y-2">
                  {patternsBySeverity.critical.map((pattern) => (
                    <PatternCard key={pattern.id} pattern={pattern} />
                  ))}
                </div>
              </div>
            )}
            {patternsBySeverity.high.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-orange-400 mb-2 uppercase">High</h4>
                <div className="space-y-2">
                  {patternsBySeverity.high.map((pattern) => (
                    <PatternCard key={pattern.id} pattern={pattern} />
                  ))}
                </div>
              </div>
            )}
            {patternsBySeverity.medium.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-amber-400 mb-2 uppercase">Medium</h4>
                <div className="space-y-2">
                  {patternsBySeverity.medium.map((pattern) => (
                    <PatternCard key={pattern.id} pattern={pattern} />
                  ))}
                </div>
              </div>
            )}
            {patternsBySeverity.low.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-slate-400 mb-2 uppercase">Low</h4>
                <div className="space-y-2">
                  {patternsBySeverity.low.map((pattern) => (
                    <PatternCard key={pattern.id} pattern={pattern} />
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Suggestions */}
      {report.suggestions.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-200">
              Suggestions ({report.suggestions.length})
            </h3>
            <div className="flex items-center gap-3 text-xs text-slate-400">
              <span className="text-emerald-400">{appliedSuggestions.length} applied</span>
              <span>•</span>
              <span className="text-amber-400">{pendingSuggestions.length} pending</span>
            </div>
          </div>
          <div className="space-y-4">
            {suggestionsByPriority.critical.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-red-400 mb-2 uppercase">Critical</h4>
                <div className="space-y-2">
                  {suggestionsByPriority.critical.map((suggestion) => (
                    <SuggestionCard
                      key={suggestion.id}
                      suggestion={suggestion}
                      onApply={onApplySuggestion}
                      onReject={onRejectSuggestion}
                      isApplying={applyingId === suggestion.id}
                    />
                  ))}
                </div>
              </div>
            )}
            {suggestionsByPriority.high.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-orange-400 mb-2 uppercase">High</h4>
                <div className="space-y-2">
                  {suggestionsByPriority.high.map((suggestion) => (
                    <SuggestionCard
                      key={suggestion.id}
                      suggestion={suggestion}
                      onApply={onApplySuggestion}
                      onReject={onRejectSuggestion}
                      isApplying={applyingId === suggestion.id}
                    />
                  ))}
                </div>
              </div>
            )}
            {suggestionsByPriority.medium.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-amber-400 mb-2 uppercase">Medium</h4>
                <div className="space-y-2">
                  {suggestionsByPriority.medium.map((suggestion) => (
                    <SuggestionCard
                      key={suggestion.id}
                      suggestion={suggestion}
                      onApply={onApplySuggestion}
                      onReject={onRejectSuggestion}
                      isApplying={applyingId === suggestion.id}
                    />
                  ))}
                </div>
              </div>
            )}
            {suggestionsByPriority.low.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-slate-400 mb-2 uppercase">Low</h4>
                <div className="space-y-2">
                  {suggestionsByPriority.low.map((suggestion) => (
                    <SuggestionCard
                      key={suggestion.id}
                      suggestion={suggestion}
                      onApply={onApplySuggestion}
                      onReject={onRejectSuggestion}
                      isApplying={applyingId === suggestion.id}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {report.patterns.length === 0 && report.suggestions.length === 0 && (
        <div className="bg-slate-800/30 rounded-lg p-8 border border-slate-700/30 text-center">
          <p className="text-slate-400">No patterns or suggestions found in this analysis.</p>
          <p className="text-xs text-slate-500 mt-2">
            This could mean executions are performing well, or more data is needed.
          </p>
        </div>
      )}
    </div>
  );
}
