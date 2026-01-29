import { useState, useCallback } from 'react';
import { AlertCircle, XCircle, CheckCircle, AlertTriangle, X, RefreshCw } from 'lucide-react';
import { Button } from './button';
import { cn } from '../lib/utils';

/**
 * Alert severity levels for styling and icon selection.
 * [REQ:FAILURE-TOPOGRAPHY] Graceful degradation with clear user feedback
 */
export type AlertSeverity = 'error' | 'warning' | 'success' | 'info';

interface InlineAlertProps {
  /** The message to display */
  message: string;
  /** Severity level - affects styling and icon */
  severity?: AlertSeverity;
  /** Optional title for the alert */
  title?: string;
  /** Whether the alert can be dismissed */
  dismissible?: boolean;
  /** Callback when dismissed */
  onDismiss?: () => void;
  /** If true, shows a retry button */
  retryable?: boolean;
  /** Callback when retry is clicked */
  onRetry?: () => void | Promise<void>;
  /** Extra class names */
  className?: string;
  /** Test ID for automation */
  'data-testid'?: string;
}

const severityConfig: Record<
  AlertSeverity,
  { icon: typeof AlertCircle; bg: string; border: string; text: string; iconColor: string }
> = {
  error: {
    icon: XCircle,
    bg: 'bg-red-500/10',
    border: 'border-red-500/20',
    text: 'text-red-400',
    iconColor: 'text-red-400',
  },
  warning: {
    icon: AlertTriangle,
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/20',
    text: 'text-amber-400',
    iconColor: 'text-amber-400',
  },
  success: {
    icon: CheckCircle,
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/20',
    text: 'text-emerald-400',
    iconColor: 'text-emerald-400',
  },
  info: {
    icon: AlertCircle,
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/20',
    text: 'text-blue-400',
    iconColor: 'text-blue-400',
  },
};

/**
 * InlineAlert - A dismissible inline alert component for displaying errors, warnings, and info.
 * Replaces `alert()` dialogs with proper UI feedback and optional retry capability.
 *
 * [REQ:FAILURE-TOPOGRAPHY] Graceful degradation pattern
 */
export function InlineAlert({
  message,
  severity = 'error',
  title,
  dismissible = true,
  onDismiss,
  retryable = false,
  onRetry,
  className,
  'data-testid': testId,
}: InlineAlertProps) {
  const [retrying, setRetrying] = useState(false);
  const config = severityConfig[severity];
  const Icon = config.icon;

  const handleRetry = useCallback(async () => {
    if (!onRetry || retrying) return;
    setRetrying(true);
    try {
      await onRetry();
    } finally {
      setRetrying(false);
    }
  }, [onRetry, retrying]);

  return (
    <div
      role="alert"
      aria-live="polite"
      className={cn(
        'rounded-xl border px-4 py-3 flex items-start gap-3',
        config.bg,
        config.border,
        className
      )}
      data-testid={testId}
    >
      <Icon className={cn('h-5 w-5 flex-shrink-0 mt-0.5', config.iconColor)} aria-hidden="true" />

      <div className="flex-1 min-w-0">
        {title && (
          <p className={cn('font-medium mb-1', config.text)}>{title}</p>
        )}
        <p className={cn('text-sm', config.text)}>{message}</p>
      </div>

      <div className="flex items-center gap-2 flex-shrink-0">
        {retryable && onRetry && (
          <Button
            variant="outline"
            size="sm"
            onClick={handleRetry}
            disabled={retrying}
            className="gap-1"
            data-testid={testId ? `${testId}-retry` : undefined}
          >
            <RefreshCw className={cn('h-3 w-3', retrying && 'animate-spin')} />
            {retrying ? 'Retrying...' : 'Retry'}
          </Button>
        )}

        {dismissible && onDismiss && (
          <button
            type="button"
            onClick={onDismiss}
            className={cn(
              'p-1 rounded hover:bg-white/10 transition-colors',
              config.text
            )}
            aria-label="Dismiss alert"
            data-testid={testId ? `${testId}-dismiss` : undefined}
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}
