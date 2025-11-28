import { useEffect, useState, useCallback } from 'react';
import { BarChart3, TrendingUp, Clock, CheckCircle, XCircle, FileCode, RefreshCw, Loader2, ChevronDown, ChevronUp } from 'lucide-react';
import { getAnalyticsSummary, type AnalyticsSummary, type GenerationEvent } from '../lib/api';
import { Tooltip } from './Tooltip';

// [REQ:TMPL-GENERATION-ANALYTICS] UI component for displaying factory-level analytics

interface AnalyticsPanelProps {
  className?: string;
}

export function AnalyticsPanel({ className = '' }: AnalyticsPanelProps) {
  const [summary, setSummary] = useState<AnalyticsSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(false);

  const fetchSummary = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getAnalyticsSummary();
      setSummary(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load analytics');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSummary();
  }, [fetchSummary]);

  const formatDuration = (ms: number): string => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  const formatDate = (dateStr: string): string => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // Empty state
  if (!loading && summary && summary.total_generations === 0) {
    return (
      <section
        className={`rounded-2xl border border-white/10 bg-white/5 p-6 ${className}`}
        role="region"
        aria-label="Generation analytics"
        data-testid="analytics-panel"
      >
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5 text-slate-400" aria-hidden="true" />
            <h2 className="text-lg font-semibold">Factory Analytics</h2>
          </div>
          <button
            onClick={fetchSummary}
            disabled={loading}
            className="p-2 rounded-lg hover:bg-white/10 transition-colors disabled:opacity-50"
            aria-label="Refresh analytics"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
        <p className="text-slate-400 text-sm" data-testid="analytics-empty-state">
          No generation events yet. Create your first landing page to see analytics here.
        </p>
      </section>
    );
  }

  return (
    <section
      className={`rounded-2xl border border-white/10 bg-white/5 p-6 ${className}`}
      role="region"
      aria-label="Generation analytics"
      data-testid="analytics-panel"
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <BarChart3 className="h-5 w-5 text-emerald-400" aria-hidden="true" />
          <h2 className="text-lg font-semibold">Factory Analytics</h2>
          <Tooltip content="Track template usage and generation success metrics at the factory level">
            <span className="text-xs text-slate-500 hover:text-slate-400 cursor-help">?</span>
          </Tooltip>
        </div>
        <button
          onClick={fetchSummary}
          disabled={loading}
          className="p-2 rounded-lg hover:bg-white/10 transition-colors disabled:opacity-50"
          aria-label="Refresh analytics"
        >
          {loading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-200 text-sm" role="alert">
          {error}
        </div>
      )}

      {loading && !summary && (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-emerald-400" />
        </div>
      )}

      {summary && (
        <>
          {/* Summary Cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            {/* Total Generations */}
            <div className="p-4 rounded-xl bg-white/5 border border-white/10">
              <div className="flex items-center gap-2 mb-2">
                <FileCode className="h-4 w-4 text-blue-400" aria-hidden="true" />
                <span className="text-xs text-slate-400">Total</span>
              </div>
              <p className="text-2xl font-bold" data-testid="analytics-total">
                {summary.total_generations}
              </p>
            </div>

            {/* Success Rate */}
            <div className="p-4 rounded-xl bg-white/5 border border-white/10">
              <div className="flex items-center gap-2 mb-2">
                <TrendingUp className="h-4 w-4 text-emerald-400" aria-hidden="true" />
                <span className="text-xs text-slate-400">Success Rate</span>
              </div>
              <p className="text-2xl font-bold" data-testid="analytics-success-rate">
                {summary.success_rate.toFixed(0)}%
              </p>
            </div>

            {/* Successful */}
            <div className="p-4 rounded-xl bg-white/5 border border-white/10">
              <div className="flex items-center gap-2 mb-2">
                <CheckCircle className="h-4 w-4 text-emerald-400" aria-hidden="true" />
                <span className="text-xs text-slate-400">Successful</span>
              </div>
              <p className="text-2xl font-bold text-emerald-400" data-testid="analytics-successful">
                {summary.successful_count}
              </p>
            </div>

            {/* Average Duration */}
            <div className="p-4 rounded-xl bg-white/5 border border-white/10">
              <div className="flex items-center gap-2 mb-2">
                <Clock className="h-4 w-4 text-amber-400" aria-hidden="true" />
                <span className="text-xs text-slate-400">Avg Duration</span>
              </div>
              <p className="text-2xl font-bold" data-testid="analytics-avg-duration">
                {formatDuration(summary.average_duration_ms)}
              </p>
            </div>
          </div>

          {/* Template Breakdown */}
          {Object.keys(summary.by_template).length > 0 && (
            <div className="mb-4">
              <h3 className="text-sm font-medium text-slate-300 mb-3">By Template</h3>
              <div className="flex flex-wrap gap-2">
                {Object.entries(summary.by_template).map(([templateId, count]) => (
                  <div
                    key={templateId}
                    className="px-3 py-1.5 rounded-lg bg-white/5 border border-white/10 text-sm"
                    data-testid={`analytics-template-${templateId}`}
                  >
                    <span className="text-slate-300">{templateId}</span>
                    <span className="ml-2 text-emerald-400 font-semibold">{count}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Expandable Recent Events */}
          <div>
            <button
              onClick={() => setExpanded(!expanded)}
              className="flex items-center gap-2 text-sm text-slate-400 hover:text-slate-300 transition-colors w-full justify-between"
              aria-expanded={expanded}
              aria-controls="recent-events-list"
            >
              <span className="font-medium">Recent Events ({summary.recent_events.length})</span>
              {expanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            </button>

            {expanded && summary.recent_events.length > 0 && (
              <div
                id="recent-events-list"
                className="mt-3 space-y-2 max-h-64 overflow-y-auto"
                data-testid="analytics-recent-events"
              >
                {summary.recent_events.map((event: GenerationEvent) => (
                  <div
                    key={event.event_id}
                    className="flex items-center justify-between p-3 rounded-lg bg-white/5 border border-white/10 text-sm"
                  >
                    <div className="flex items-center gap-3">
                      {event.success ? (
                        <CheckCircle className="h-4 w-4 text-emerald-400" aria-label="Successful" />
                      ) : (
                        <XCircle className="h-4 w-4 text-red-400" aria-label="Failed" />
                      )}
                      <div>
                        <span className="text-slate-200">{event.scenario_id || 'unnamed'}</span>
                        <span className="text-slate-500 ml-2 text-xs">({event.template_id})</span>
                        {event.is_dry_run && (
                          <span className="ml-2 px-1.5 py-0.5 text-xs rounded bg-blue-500/20 text-blue-300">
                            dry-run
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="text-xs text-slate-500">
                      {formatDate(event.timestamp)}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </section>
  );
}
