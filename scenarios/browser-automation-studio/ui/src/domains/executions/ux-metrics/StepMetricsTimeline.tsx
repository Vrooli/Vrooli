import clsx from 'clsx';
import { CheckCircle, XCircle, AlertTriangle } from 'lucide-react';
import type { StepMetrics } from '@stores/uxMetricsStore';
import { formatDuration, getFrictionScoreColor } from '@stores/uxMetricsStore';

interface StepMetricsTimelineProps {
  steps: StepMetrics[];
  selectedStepIndex?: number;
  onStepSelect?: (stepIndex: number) => void;
  className?: string;
}

/**
 * Visual timeline showing per-step UX metrics with friction indicators.
 */
export function StepMetricsTimeline({
  steps,
  selectedStepIndex,
  onStepSelect,
  className,
}: StepMetricsTimelineProps) {
  if (steps.length === 0) {
    return (
      <div className={clsx('text-sm text-slate-500 italic', className)}>
        No step metrics available
      </div>
    );
  }

  // Find max duration for scaling bars
  const maxDuration = Math.max(...steps.map((s) => s.totalDurationMs), 1);

  return (
    <div className={clsx('space-y-1', className)}>
      {steps.map((step) => (
        <StepMetricRow
          key={step.stepIndex}
          step={step}
          maxDuration={maxDuration}
          isSelected={selectedStepIndex === step.stepIndex}
          onClick={() => onStepSelect?.(step.stepIndex)}
        />
      ))}
    </div>
  );
}

interface StepMetricRowProps {
  step: StepMetrics;
  maxDuration: number;
  isSelected?: boolean;
  onClick?: () => void;
}

function StepMetricRow({ step, maxDuration, isSelected, onClick }: StepMetricRowProps) {
  const hasHighFriction = step.frictionScore >= 50;
  const hasMediumFriction = step.frictionScore >= 25 && step.frictionScore < 50;
  const durationPercent = (step.totalDurationMs / maxDuration) * 100;
  const frictionColor = getFrictionScoreColor(step.frictionScore);

  return (
    <button
      type="button"
      onClick={onClick}
      className={clsx(
        'w-full text-left px-3 py-2 rounded-lg border transition-all',
        'hover:bg-slate-800/50',
        isSelected
          ? 'border-flow-accent bg-slate-800/30'
          : 'border-transparent hover:border-slate-700'
      )}
    >
      <div className="flex items-center gap-3">
        {/* Step indicator */}
        <div className="flex-shrink-0 w-8 h-8 flex items-center justify-center">
          <StepIndicator
            hasHighFriction={hasHighFriction}
            hasMediumFriction={hasMediumFriction}
          />
        </div>

        {/* Step info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-white">
              Step {step.stepIndex + 1}
            </span>
            {step.stepType && (
              <span className="text-xs text-slate-500 capitalize">{step.stepType}</span>
            )}
            {step.retryCount > 0 && (
              <span className="text-xs text-yellow-400">
                {step.retryCount} {step.retryCount === 1 ? 'retry' : 'retries'}
              </span>
            )}
          </div>

          {/* Duration bar */}
          <div className="mt-1 h-1.5 bg-slate-800 rounded-full overflow-hidden">
            <div
              className={clsx(
                'h-full rounded-full transition-all',
                hasHighFriction
                  ? 'bg-red-500'
                  : hasMediumFriction
                    ? 'bg-yellow-500'
                    : 'bg-green-500'
              )}
              style={{ width: `${durationPercent}%` }}
            />
          </div>
        </div>

        {/* Metrics */}
        <div className="flex-shrink-0 text-right">
          <div className="text-sm text-slate-300">
            {formatDuration(step.totalDurationMs)}
          </div>
          {step.frictionScore > 0 && (
            <div className={clsx('text-xs', frictionColor)}>
              Friction: {step.frictionScore.toFixed(0)}
            </div>
          )}
        </div>

        {/* Friction signal count */}
        {step.frictionSignals.length > 0 && (
          <div className="flex-shrink-0 ml-2">
            <span
              className={clsx(
                'inline-flex items-center justify-center w-5 h-5 text-[10px] font-medium rounded-full',
                hasHighFriction
                  ? 'bg-red-500/30 text-red-300'
                  : 'bg-yellow-500/30 text-yellow-300'
              )}
            >
              {step.frictionSignals.length}
            </span>
          </div>
        )}
      </div>
    </button>
  );
}

interface StepIndicatorProps {
  hasHighFriction: boolean;
  hasMediumFriction: boolean;
}

function StepIndicator({ hasHighFriction, hasMediumFriction }: StepIndicatorProps) {
  if (hasHighFriction) {
    return <XCircle className="w-5 h-5 text-red-400" />;
  }
  if (hasMediumFriction) {
    return <AlertTriangle className="w-5 h-5 text-yellow-400" />;
  }
  return <CheckCircle className="w-5 h-5 text-green-400" />;
}
