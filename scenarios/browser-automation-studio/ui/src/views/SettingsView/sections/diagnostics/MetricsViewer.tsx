/**
 * MetricsViewer Component
 *
 * Displays Prometheus metrics with histogram visualization and filtering.
 */

import { useState, useMemo } from 'react';
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Code,
  BarChart3,
  Settings2,
  ExternalLink,
} from 'lucide-react';
import type { MetricsResponse, MetricData, MetricValue } from '@/domains/observability';
import { JsonViewer } from './JsonViewer';

// ============================================================================
// Histogram Chart
// ============================================================================

interface HistogramChartProps {
  values: MetricValue[];
}

function HistogramChart({ values }: HistogramChartProps) {
  // Extract bucket values and compute per-bucket counts
  const bucketData = useMemo(() => {
    // Filter to only _bucket values and sort by le (bucket boundary)
    const buckets = values
      .filter((v) => v.labels._suffix === '_bucket')
      .map((v) => ({
        le: v.labels.le || '+Inf',
        cumulative: v.value,
      }))
      .sort((a, b) => {
        if (a.le === '+Inf') return 1;
        if (b.le === '+Inf') return -1;
        return parseFloat(a.le) - parseFloat(b.le);
      });

    if (buckets.length === 0) return [];

    // Calculate per-bucket counts (non-cumulative)
    const perBucket: Array<{ le: string; count: number }> = [];
    let prev = 0;
    for (const bucket of buckets) {
      const count = bucket.cumulative - prev;
      perBucket.push({ le: bucket.le, count });
      prev = bucket.cumulative;
    }

    return perBucket;
  }, [values]);

  // Get count and sum for summary
  const count = values.find((v) => v.labels._suffix === '_count');
  const sum = values.find((v) => v.labels._suffix === '_sum');
  const avg = count && count.value > 0 ? sum ? (sum.value / count.value) : 0 : 0;

  // Find max for scaling bars
  const maxCount = Math.max(...bucketData.map((b) => b.count), 1);

  if (bucketData.length === 0) {
    return <div className="text-gray-500 text-sm">No bucket data available</div>;
  }

  return (
    <div className="space-y-3">
      {/* Summary stats */}
      <div className="flex items-center gap-4 text-xs text-gray-400">
        {count && <span>Total: <strong className="text-surface">{count.value}</strong></span>}
        {sum && <span>Sum: <strong className="text-surface">{sum.value.toFixed(2)}</strong></span>}
        {avg > 0 && <span>Avg: <strong className="text-surface">{avg.toFixed(3)}</strong></span>}
      </div>

      {/* Bar chart */}
      <div className="space-y-1">
        {bucketData.map((bucket, idx) => {
          const widthPercent = (bucket.count / maxCount) * 100;
          return (
            <div key={idx} className="flex items-center gap-2 text-xs">
              <span className="w-16 text-right font-mono text-gray-500 flex-shrink-0">
                ≤{bucket.le === '+Inf' ? '∞' : bucket.le}
              </span>
              <div className="flex-1 h-5 bg-gray-700 rounded overflow-hidden">
                <div
                  className="h-full bg-gradient-to-r from-flow-accent/70 to-flow-accent rounded transition-all duration-300"
                  style={{ width: `${widthPercent}%` }}
                />
              </div>
              <span className="w-12 text-right font-mono text-gray-400 flex-shrink-0">
                {bucket.count}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ============================================================================
// Metric Detail View
// ============================================================================

interface MetricDetailViewProps {
  metric: MetricData;
}

function MetricDetailView({ metric }: MetricDetailViewProps) {
  const [viewMode, setViewMode] = useState<'table' | 'graph'>('graph');
  const isHistogram = metric.type === 'histogram';

  return (
    <div className="p-4">
      {/* Toggle for histograms */}
      {isHistogram && (
        <div className="flex items-center gap-2 mb-4">
          <button
            onClick={() => setViewMode('graph')}
            className={`flex items-center gap-1 px-2 py-1 text-xs rounded transition-colors ${
              viewMode === 'graph'
                ? 'bg-flow-accent/20 text-flow-accent'
                : 'bg-gray-700 text-gray-400 hover:text-surface'
            }`}
          >
            <BarChart3 size={12} />
            Graph
          </button>
          <button
            onClick={() => setViewMode('table')}
            className={`flex items-center gap-1 px-2 py-1 text-xs rounded transition-colors ${
              viewMode === 'table'
                ? 'bg-flow-accent/20 text-flow-accent'
                : 'bg-gray-700 text-gray-400 hover:text-surface'
            }`}
          >
            <Settings2 size={12} />
            Raw
          </button>
        </div>
      )}

      {/* Graph view for histograms */}
      {isHistogram && viewMode === 'graph' ? (
        <HistogramChart values={metric.values} />
      ) : (
        /* Table view (default for non-histograms) */
        <>
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500">
                <th className="pb-2 font-normal">Labels</th>
                <th className="pb-2 font-normal text-right">Value</th>
              </tr>
            </thead>
            <tbody className="text-gray-300">
              {metric.values.slice(0, 10).map((v: MetricValue, idx: number) => (
                <tr key={idx} className="border-t border-gray-700/50">
                  <td className="py-2 pr-4">
                    <span className="font-mono text-xs">
                      {Object.entries(v.labels)
                        .filter(([k]) => k !== '_suffix')
                        .map(([k, val]) => `${k}="${val}"`)
                        .join(', ') || (v.labels._suffix ? v.labels._suffix.replace(/^_/, '') : 'value')}
                    </span>
                  </td>
                  <td className="py-2 text-right font-mono">
                    {typeof v.value === 'number' && !Number.isInteger(v.value)
                      ? v.value.toFixed(4)
                      : v.value}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {metric.values.length > 10 && (
            <p className="text-xs text-gray-500 mt-2">
              Showing 10 of {metric.values.length} values
            </p>
          )}
        </>
      )}
    </div>
  );
}

// ============================================================================
// Utility Functions
// ============================================================================

// Check if a metric has any meaningful data
function hasMetricData(metric: MetricData): boolean {
  if (metric.values.length === 0) return false;
  // Check if all values are 0 (no activity)
  return metric.values.some((v: MetricValue) => v.value !== 0);
}

// Get a summary value for a metric (for display when collapsed)
function getMetricSummaryValue(metric: MetricData): string {
  if (metric.values.length === 0) return 'No data';

  // For counters, sum all values or get the total
  if (metric.type === 'counter') {
    const total = metric.values.find((v: MetricValue) => v.labels._suffix === '_total');
    if (total) return total.value.toString();
    const sum = metric.values.reduce((acc: number, v: MetricValue) => acc + v.value, 0);
    return sum.toString();
  }

  // For gauges, get the current value
  if (metric.type === 'gauge') {
    const value = metric.values.find((v: MetricValue) => v.labels._suffix === '_value');
    if (value) return value.value.toString();
    // Sum all labeled values for multi-label gauges
    const sum = metric.values.reduce((acc: number, v: MetricValue) => acc + v.value, 0);
    return sum.toString();
  }

  // For histograms, show count and sum
  if (metric.type === 'histogram') {
    const count = metric.values.find((v: MetricValue) => v.labels._suffix === '_count');
    const sum = metric.values.find((v: MetricValue) => v.labels._suffix === '_sum');
    if (count && sum) {
      const avg = count.value > 0 ? (sum.value / count.value).toFixed(1) : '0';
      return `${count.value} samples, avg ${avg}`;
    }
    return `${metric.values.length} buckets`;
  }

  return `${metric.values.length} values`;
}

// ============================================================================
// Metrics Viewer
// ============================================================================

interface MetricsViewerProps {
  data: MetricsResponse;
  onBack: () => void;
}

export function MetricsViewer({ data, onBack }: MetricsViewerProps) {
  const [showJson, setShowJson] = useState(false);
  const [showInactive, setShowInactive] = useState(false);

  // Separate metrics into active (has data) and inactive (empty/all zeros)
  const { activeMetrics, inactiveMetrics } = useMemo(() => {
    const active: Array<[string, MetricData]> = [];
    const inactive: Array<[string, MetricData]> = [];

    for (const [name, metric] of Object.entries(data.metrics)) {
      if (hasMetricData(metric)) {
        active.push([name, metric]);
      } else {
        inactive.push([name, metric]);
      }
    }

    return { activeMetrics: active, inactiveMetrics: inactive };
  }, [data.metrics]);

  if (showJson) {
    return <JsonViewer data={data} onBack={() => setShowJson(false)} title="Metrics JSON" />;
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="flex items-center gap-1 text-gray-400 hover:text-surface transition-colors"
          >
            <ArrowLeft size={18} />
          </button>
          <div className="flex items-center gap-2">
            <BarChart3 size={20} className="text-flow-accent" />
            <h2 className="text-lg font-semibold text-surface">Prometheus Metrics</h2>
          </div>
        </div>
        <button
          onClick={() => setShowJson(true)}
          className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-surface bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
        >
          <Code size={16} />
          View JSON
        </button>
      </div>

      {/* Summary */}
      <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4">
        <div className="flex flex-wrap items-center gap-4 text-sm text-gray-400">
          <span><strong className="text-surface">{activeMetrics.length}</strong> active metrics</span>
          <span><strong className="text-gray-500">{inactiveMetrics.length}</strong> inactive</span>
          <span>Updated: {new Date(data.summary.timestamp).toLocaleTimeString()}</span>
          {data.summary.config.port && (
            <span className="flex items-center gap-1">
              <ExternalLink size={12} />
              Port {data.summary.config.port}
            </span>
          )}
        </div>
      </div>

      {/* Active Metrics */}
      {activeMetrics.length > 0 ? (
        <div className="space-y-4">
          {activeMetrics.map(([name, metric]) => (
            <div key={name} className="bg-gray-800/50 border border-gray-700 rounded-lg overflow-hidden">
              <div className="px-4 py-3 border-b border-gray-700">
                <div className="flex items-center justify-between">
                  <span className="font-mono text-sm text-surface">{name}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-emerald-400">{getMetricSummaryValue(metric)}</span>
                    <span className="text-xs px-2 py-0.5 bg-gray-700 text-gray-400 rounded">{metric.type}</span>
                  </div>
                </div>
                {metric.help && (
                  <p className="text-xs text-gray-500 mt-1">{metric.help}</p>
                )}
              </div>
              <MetricDetailView metric={metric} />
            </div>
          ))}
        </div>
      ) : (
        <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-8 text-center">
          <BarChart3 size={32} className="mx-auto text-gray-600 mb-3" />
          <p className="text-gray-400">No metrics with recorded data yet</p>
          <p className="text-xs text-gray-500 mt-1">Metrics will appear here once activity is recorded</p>
        </div>
      )}

      {/* Inactive Metrics (Collapsible) */}
      {inactiveMetrics.length > 0 && (
        <div className="border border-gray-700 rounded-lg overflow-hidden">
          <button
            onClick={() => setShowInactive(!showInactive)}
            className="w-full flex items-center justify-between px-4 py-3 bg-gray-800/30 hover:bg-gray-800/50 transition-colors text-left"
          >
            <span className="text-sm text-gray-400">
              {inactiveMetrics.length} inactive metric{inactiveMetrics.length !== 1 ? 's' : ''} (no data recorded)
            </span>
            {showInactive ? <ChevronDown size={16} className="text-gray-500" /> : <ChevronRight size={16} className="text-gray-500" />}
          </button>
          {showInactive && (
            <div className="p-4 bg-gray-900/30 border-t border-gray-700">
              <div className="space-y-2">
                {inactiveMetrics.map(([name, metric]) => (
                  <div key={name} className="flex items-center justify-between py-2 px-3 bg-gray-800/30 rounded text-sm">
                    <div>
                      <span className="font-mono text-xs text-gray-400">{name}</span>
                      {metric.help && (
                        <p className="text-xs text-gray-600 mt-0.5">{metric.help}</p>
                      )}
                    </div>
                    <span className="text-xs px-2 py-0.5 bg-gray-700/50 text-gray-500 rounded">{metric.type || 'unknown'}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default MetricsViewer;
