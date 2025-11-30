import { AlertCircle, RefreshCcw, Info, XCircle, AlertTriangle } from 'lucide-react';
import { memo } from 'react';

/**
 * Structured error from the API with error codes and recovery suggestions
 */
export interface StructuredError {
  success?: boolean;
  error_code?: string;
  message: string;
  details?: string;
  recoverable?: boolean;
  suggestion?: string;
}

interface ErrorDisplayProps {
  /** Error message or structured error object */
  error: string | StructuredError | null;
  /** Optional retry callback - if provided, shows retry button */
  onRetry?: () => void;
  /** Optional dismiss callback - if provided, shows dismiss button */
  onDismiss?: () => void;
  /** Visual variant */
  variant?: 'error' | 'warning' | 'info';
  /** Size variant */
  size?: 'sm' | 'md' | 'lg';
  /** Additional CSS classes */
  className?: string;
  /** Test ID for testing */
  testId?: string;
}

/**
 * ErrorDisplay - A reusable component for displaying errors with context and recovery options.
 *
 * Features:
 * - Displays structured errors with error codes, details, and suggestions
 * - Shows retry button for recoverable errors
 * - Supports multiple variants (error, warning, info)
 * - Accessible with proper ARIA roles
 */
export const ErrorDisplay = memo(function ErrorDisplay({
  error,
  onRetry,
  onDismiss,
  variant = 'error',
  size = 'md',
  className = '',
  testId,
}: ErrorDisplayProps) {
  if (!error) return null;

  // Parse error to structured format
  const structuredError: StructuredError = typeof error === 'string'
    ? { message: error, recoverable: true }
    : error;

  const {
    error_code,
    message,
    details,
    recoverable = true,
    suggestion,
  } = structuredError;

  // Variant styles
  const variantStyles = {
    error: {
      border: 'border-red-500/30',
      bg: 'bg-red-500/10',
      text: 'text-red-200',
      icon: XCircle,
      iconColor: 'text-red-400',
    },
    warning: {
      border: 'border-amber-500/30',
      bg: 'bg-amber-500/10',
      text: 'text-amber-200',
      icon: AlertTriangle,
      iconColor: 'text-amber-400',
    },
    info: {
      border: 'border-blue-500/30',
      bg: 'bg-blue-500/10',
      text: 'text-blue-200',
      icon: Info,
      iconColor: 'text-blue-400',
    },
  };

  // Size styles
  const sizeStyles = {
    sm: {
      padding: 'p-3',
      iconSize: 'h-4 w-4',
      textSize: 'text-xs',
      codeSize: 'text-[10px]',
    },
    md: {
      padding: 'p-4',
      iconSize: 'h-5 w-5',
      textSize: 'text-sm',
      codeSize: 'text-xs',
    },
    lg: {
      padding: 'p-5',
      iconSize: 'h-6 w-6',
      textSize: 'text-base',
      codeSize: 'text-sm',
    },
  };

  const styles = variantStyles[variant];
  const sizes = sizeStyles[size];
  const IconComponent = styles.icon;

  return (
    <div
      className={`rounded-xl border ${styles.border} ${styles.bg} ${sizes.padding} ${className}`}
      role="alert"
      aria-live="assertive"
      data-testid={testId || 'error-display'}
    >
      <div className="flex items-start gap-3">
        <IconComponent
          className={`${sizes.iconSize} ${styles.iconColor} flex-shrink-0 mt-0.5`}
          aria-hidden="true"
        />
        <div className="flex-1 space-y-2">
          {/* Error code badge */}
          {error_code && (
            <span
              className={`inline-block px-2 py-0.5 ${sizes.codeSize} font-mono rounded bg-slate-800/60 ${styles.text}`}
            >
              {error_code}
            </span>
          )}

          {/* Main message */}
          <p className={`${sizes.textSize} font-medium ${styles.text}`}>
            {message}
          </p>

          {/* Details */}
          {details && (
            <p className={`${sizes.codeSize} ${styles.text} opacity-80`}>
              {details}
            </p>
          )}

          {/* Suggestion */}
          {suggestion && (
            <div
              className={`flex items-start gap-1.5 ${sizes.codeSize} ${styles.text} opacity-90 pt-1`}
            >
              <Info className="h-3 w-3 flex-shrink-0 mt-0.5" aria-hidden="true" />
              <span>{suggestion}</span>
            </div>
          )}

          {/* Action buttons */}
          {(onRetry || onDismiss) && (
            <div className="flex items-center gap-2 pt-2">
              {onRetry && recoverable && (
                <button
                  onClick={onRetry}
                  className={`inline-flex items-center gap-1.5 px-3 py-1.5 ${sizes.codeSize} font-medium rounded-lg border border-slate-500/40 bg-slate-500/20 text-slate-200 hover:bg-slate-500/30 hover:text-white transition-colors focus:outline-none focus:ring-2 focus:ring-slate-500`}
                  aria-label="Retry operation"
                >
                  <RefreshCcw className="h-3 w-3" aria-hidden="true" />
                  Try Again
                </button>
              )}
              {onDismiss && (
                <button
                  onClick={onDismiss}
                  className={`inline-flex items-center gap-1.5 px-3 py-1.5 ${sizes.codeSize} font-medium rounded-lg text-slate-400 hover:text-slate-200 transition-colors focus:outline-none focus:ring-2 focus:ring-slate-500`}
                  aria-label="Dismiss error"
                >
                  Dismiss
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
});

/**
 * Parse an API error response into a structured error
 */
export function parseApiError(error: unknown): StructuredError {
  if (!error) {
    return { message: 'An unknown error occurred', recoverable: true };
  }

  // Handle Error objects
  if (error instanceof Error) {
    const message = error.message;

    // Try to parse JSON error from API response
    const jsonMatch = message.match(/API call failed \(\d+\): (.+)/);
    if (jsonMatch) {
      try {
        const parsed = JSON.parse(jsonMatch[1]);
        return {
          success: parsed.success ?? false,
          error_code: parsed.error_code,
          message: parsed.message || message,
          details: parsed.details,
          recoverable: parsed.recoverable ?? true,
          suggestion: parsed.suggestion,
        };
      } catch {
        // Not JSON, use the raw message
      }
    }

    return { message, recoverable: true };
  }

  // Handle structured error objects
  if (typeof error === 'object' && 'message' in error) {
    return error as StructuredError;
  }

  // Handle string errors
  if (typeof error === 'string') {
    return { message: error, recoverable: true };
  }

  return { message: 'An unexpected error occurred', recoverable: false };
}
