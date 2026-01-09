import clsx from 'clsx';
import { AlertTriangle, Clock, MousePointer2, RefreshCw, Zap, ArrowLeft, Target } from 'lucide-react';
import type { FrictionSignal, FrictionType, Severity } from '@stores/uxMetricsStore';
import { getSeverityColor, getSeverityBgColor, getFrictionTypeLabel } from '@stores/uxMetricsStore';

interface FrictionSignalsListProps {
  signals: FrictionSignal[];
  maxItems?: number;
  className?: string;
}

/**
 * Displays a list of friction signals with severity indicators.
 */
export function FrictionSignalsList({ signals, maxItems, className }: FrictionSignalsListProps) {
  const displaySignals = maxItems ? signals.slice(0, maxItems) : signals;
  const hiddenCount = maxItems ? Math.max(0, signals.length - maxItems) : 0;

  if (signals.length === 0) {
    return (
      <div className={clsx('text-sm text-slate-500 italic', className)}>
        No friction signals detected
      </div>
    );
  }

  return (
    <div className={clsx('space-y-2', className)}>
      {displaySignals.map((signal, idx) => (
        <FrictionSignalItem key={`${signal.type}-${signal.stepIndex}-${idx}`} signal={signal} />
      ))}
      {hiddenCount > 0 && (
        <div className="text-xs text-slate-500 pl-8">
          +{hiddenCount} more signal{hiddenCount > 1 ? 's' : ''}
        </div>
      )}
    </div>
  );
}

interface FrictionSignalItemProps {
  signal: FrictionSignal;
}

function FrictionSignalItem({ signal }: FrictionSignalItemProps) {
  const icon = getFrictionTypeIcon(signal.type);
  const severityColor = getSeverityColor(signal.severity);
  const bgColor = getSeverityBgColor(signal.severity);

  return (
    <div
      className={clsx(
        'flex items-start gap-3 p-3 rounded-lg border border-white/5',
        bgColor
      )}
    >
      <div className={clsx('mt-0.5', severityColor)}>{icon}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className={clsx('text-sm font-medium', severityColor)}>
            {getFrictionTypeLabel(signal.type)}
          </span>
          <SeverityBadge severity={signal.severity} />
          <span className="text-xs text-slate-500">Step {signal.stepIndex}</span>
        </div>
        <p className="text-sm text-slate-400 mt-0.5">{signal.description}</p>
        {signal.evidence && Object.keys(signal.evidence).length > 0 && (
          <div className="text-xs text-slate-500 mt-1">
            {formatEvidence(signal.evidence)}
          </div>
        )}
      </div>
      <div className="text-sm font-medium text-slate-400">{signal.score.toFixed(0)}</div>
    </div>
  );
}

function SeverityBadge({ severity }: { severity: Severity }) {
  const colors: Record<Severity, string> = {
    high: 'bg-red-500/30 text-red-300 border-red-500/30',
    medium: 'bg-yellow-500/30 text-yellow-300 border-yellow-500/30',
    low: 'bg-blue-500/30 text-blue-300 border-blue-500/30',
  };

  return (
    <span
      className={clsx(
        'px-1.5 py-0.5 text-[10px] uppercase tracking-wider rounded border',
        colors[severity]
      )}
    >
      {severity}
    </span>
  );
}

function getFrictionTypeIcon(type: FrictionType) {
  const iconProps = { className: 'w-4 h-4' };

  switch (type) {
    case 'excessive_time':
      return <Clock {...iconProps} />;
    case 'zigzag_path':
      return <MousePointer2 {...iconProps} />;
    case 'multiple_retries':
      return <RefreshCw {...iconProps} />;
    case 'rapid_clicks':
      return <Zap {...iconProps} />;
    case 'long_hesitation':
      return <Clock {...iconProps} />;
    case 'back_navigation':
      return <ArrowLeft {...iconProps} />;
    case 'element_miss':
      return <Target {...iconProps} />;
    default:
      return <AlertTriangle {...iconProps} />;
  }
}

function formatEvidence(evidence: Record<string, unknown>): string {
  const parts: string[] = [];
  for (const [key, value] of Object.entries(evidence)) {
    const formattedKey = key.replace(/_/g, ' ');
    const formattedValue = typeof value === 'number' ? value.toFixed(1) : String(value);
    parts.push(`${formattedKey}: ${formattedValue}`);
  }
  return parts.join(' | ');
}
