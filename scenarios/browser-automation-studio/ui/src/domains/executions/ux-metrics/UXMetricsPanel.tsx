import { useEffect, useState } from 'react';
import clsx from 'clsx';
import { BarChart2, RefreshCw, Lock, Timer, MousePointer2 } from 'lucide-react';
import { useUXMetricsStore, formatDuration } from '@stores/uxMetricsStore';
import { FrictionScoreCard } from './FrictionScoreCard';
import { FrictionSignalsList } from './FrictionSignalsList';
import { StepMetricsTimeline } from './StepMetricsTimeline';
import { MetricsSummaryCard } from './MetricsSummaryCard';

interface UXMetricsPanelProps {
  executionId: string;
  className?: string;
}

/**
 * Main panel for displaying UX metrics for an execution.
 * Shows friction scores, signals, step-level analysis, and recommendations.
 */
export function UXMetricsPanel({ executionId, className }: UXMetricsPanelProps) {
  const {
    executionMetrics,
    fetchExecutionMetrics,
    computeMetrics,
    isLoading,
    isComputing,
    error,
  } = useUXMetricsStore();

  const [selectedStepIndex, setSelectedStepIndex] = useState<number | undefined>();

  const metrics = executionMetrics.get(executionId);

  useEffect(() => {
    fetchExecutionMetrics(executionId);
  }, [executionId, fetchExecutionMetrics]);

  // Loading state
  if (isLoading) {
    return (
      <div className={clsx('flex items-center justify-center h-64', className)}>
        <div className="flex flex-col items-center gap-3">
          <RefreshCw className="w-8 h-8 text-flow-accent animate-spin" />
          <span className="text-sm text-slate-400">Loading UX metrics...</span>
        </div>
      </div>
    );
  }

  // Pro tier required
  if (error === 'UX metrics requires Pro plan or higher') {
    return (
      <div className={clsx('flex items-center justify-center h-64', className)}>
        <div className="flex flex-col items-center gap-3 text-center max-w-md px-4">
          <div className="w-12 h-12 rounded-full bg-yellow-500/20 flex items-center justify-center">
            <Lock className="w-6 h-6 text-yellow-400" />
          </div>
          <h3 className="text-lg font-medium text-white">Pro Feature</h3>
          <p className="text-sm text-slate-400">
            UX metrics including friction analysis, cursor path tracking, and usability insights
            are available on Pro plan and above.
          </p>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className={clsx('flex items-center justify-center h-64', className)}>
        <div className="flex flex-col items-center gap-3 text-center">
          <BarChart2 className="w-8 h-8 text-red-400" />
          <span className="text-sm text-red-400">{error}</span>
        </div>
      </div>
    );
  }

  // No metrics yet - offer to compute
  if (!metrics) {
    return (
      <div className={clsx('flex items-center justify-center h-64', className)}>
        <div className="flex flex-col items-center gap-4 text-center max-w-md px-4">
          <div className="w-12 h-12 rounded-full bg-slate-800 flex items-center justify-center">
            <BarChart2 className="w-6 h-6 text-slate-400" />
          </div>
          <div>
            <h3 className="text-lg font-medium text-white mb-1">No UX Metrics Available</h3>
            <p className="text-sm text-slate-400">
              UX metrics have not been computed for this execution yet.
            </p>
          </div>
          <button
            type="button"
            onClick={() => computeMetrics(executionId)}
            disabled={isComputing}
            className={clsx(
              'px-4 py-2 rounded-lg font-medium text-sm transition-colors',
              'bg-flow-accent text-black hover:bg-flow-accent/90',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              'flex items-center gap-2'
            )}
          >
            {isComputing ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                Computing...
              </>
            ) : (
              <>
                <BarChart2 className="w-4 h-4" />
                Compute Metrics
              </>
            )}
          </button>
        </div>
      </div>
    );
  }

  // Selected step metrics for detail view
  const selectedStep = selectedStepIndex !== undefined
    ? metrics.stepMetrics.find((s) => s.stepIndex === selectedStepIndex)
    : undefined;

  return (
    <div className={clsx('space-y-6 p-4', className)}>
      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <FrictionScoreCard score={metrics.overallFrictionScore} />
        <MetricCard
          icon={<Timer className="w-4 h-4" />}
          label="Duration"
          value={formatDuration(metrics.totalDurationMs)}
          subtitle={`${metrics.stepCount} steps | ${formatDuration(metrics.avgStepDurationMs)} avg`}
        />
        <MetricCard
          icon={<MousePointer2 className="w-4 h-4" />}
          label="Cursor Distance"
          value={`${metrics.totalCursorDistancePx.toFixed(0)}px`}
          subtitle={`${metrics.totalRetries} retries`}
        />
      </div>

      {/* Friction Signals */}
      {metrics.frictionSignals.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-white mb-3">Friction Signals</h3>
          <FrictionSignalsList signals={metrics.frictionSignals} maxItems={5} />
        </div>
      )}

      {/* Step Timeline */}
      <div>
        <h3 className="text-sm font-medium text-white mb-3">Step Analysis</h3>
        <StepMetricsTimeline
          steps={metrics.stepMetrics}
          selectedStepIndex={selectedStepIndex}
          onStepSelect={setSelectedStepIndex}
        />
      </div>

      {/* Selected Step Detail */}
      {selectedStep && (
        <div className="bg-slate-800/30 rounded-xl p-4 border border-slate-700">
          <h4 className="text-sm font-medium text-white mb-3">
            Step {selectedStep.stepIndex + 1} Details
          </h4>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-slate-500">Type:</span>{' '}
              <span className="text-slate-300 capitalize">{selectedStep.stepType || 'Unknown'}</span>
            </div>
            <div>
              <span className="text-slate-500">Duration:</span>{' '}
              <span className="text-slate-300">{formatDuration(selectedStep.totalDurationMs)}</span>
            </div>
            <div>
              <span className="text-slate-500">Friction Score:</span>{' '}
              <span className="text-slate-300">{selectedStep.frictionScore.toFixed(1)}</span>
            </div>
            <div>
              <span className="text-slate-500">Retries:</span>{' '}
              <span className="text-slate-300">{selectedStep.retryCount}</span>
            </div>
            {selectedStep.cursorPath && (
              <>
                <div>
                  <span className="text-slate-500">Cursor Distance:</span>{' '}
                  <span className="text-slate-300">
                    {selectedStep.cursorPath.totalDistancePx.toFixed(0)}px
                  </span>
                </div>
                <div>
                  <span className="text-slate-500">Directness:</span>{' '}
                  <span className="text-slate-300">
                    {(selectedStep.cursorPath.directness * 100).toFixed(0)}%
                  </span>
                </div>
              </>
            )}
          </div>
          {selectedStep.frictionSignals.length > 0 && (
            <div className="mt-4">
              <h5 className="text-xs uppercase tracking-wider text-slate-500 mb-2">
                Friction Signals
              </h5>
              <FrictionSignalsList signals={selectedStep.frictionSignals} />
            </div>
          )}
        </div>
      )}

      {/* Summary & Recommendations */}
      {metrics.summary && <MetricsSummaryCard summary={metrics.summary} />}
    </div>
  );
}

interface MetricCardProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  subtitle?: string;
}

function MetricCard({ icon, label, value, subtitle }: MetricCardProps) {
  return (
    <div className="bg-slate-900/50 rounded-xl p-4 border border-white/5">
      <div className="flex items-center gap-2 text-slate-400 mb-1">
        {icon}
        <span className="text-xs uppercase tracking-wider">{label}</span>
      </div>
      <div className="text-2xl font-semibold text-white">{value}</div>
      {subtitle && <div className="text-xs text-slate-500 mt-1">{subtitle}</div>}
    </div>
  );
}
