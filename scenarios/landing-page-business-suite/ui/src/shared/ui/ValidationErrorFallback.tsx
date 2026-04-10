import { AlertTriangle, RefreshCw, Bug } from 'lucide-react';
import { Button } from './button';

interface ValidationErrorFallbackProps {
  /** Error message to display */
  error: string;
  /** Optional callback to retry the operation */
  onRetry?: () => void;
  /** Optional callback to report the issue */
  onReport?: () => void;
  /** Visual style variant */
  variant?: 'inline' | 'card' | 'minimal';
  /** Additional context about what failed */
  context?: string;
  /** Whether to show technical details (development mode) */
  showDetails?: boolean;
  /** Raw data that failed validation (for debugging) */
  rawData?: unknown;
}

/**
 * Fallback UI component for API validation failures.
 *
 * Use this when Zod schema validation fails on API responses.
 * Provides user-friendly messaging and optional retry functionality.
 *
 * @example
 * const result = safeParse(schema, apiResponse, 'UserProfile');
 * if (!result.success) {
 *   return (
 *     <ValidationErrorFallback
 *       error={result.error}
 *       onRetry={() => refetch()}
 *       context="user profile"
 *     />
 *   );
 * }
 */
export function ValidationErrorFallback({
  error,
  onRetry,
  onReport,
  variant = 'card',
  context,
  showDetails = false,
  rawData,
}: ValidationErrorFallbackProps) {
  const isDev = import.meta.env.DEV;
  const shouldShowDetails = showDetails || isDev;

  if (variant === 'minimal') {
    return (
      <div className="flex items-center gap-2 text-sm text-amber-300">
        <AlertTriangle className="h-4 w-4 flex-shrink-0" />
        <span>{context ? `Failed to load ${context}` : 'Data validation failed'}</span>
        {onRetry && (
          <button
            type="button"
            onClick={onRetry}
            className="text-amber-200 hover:text-amber-100 underline underline-offset-2"
          >
            Retry
          </button>
        )}
      </div>
    );
  }

  if (variant === 'inline') {
    return (
      <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3">
        <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
        <div className="flex-1 space-y-2">
          <p className="text-sm text-amber-200">
            {context ? `Unable to display ${context}` : 'Data format error'}
          </p>
          {shouldShowDetails && (
            <details className="text-xs text-amber-300/70">
              <summary className="cursor-pointer hover:text-amber-300">Technical details</summary>
              <pre className="mt-2 whitespace-pre-wrap break-all bg-black/20 p-2 rounded">
                {error}
              </pre>
            </details>
          )}
          {onRetry && (
            <Button
              onClick={onRetry}
              size="sm"
              variant="ghost"
              className="gap-1.5 text-amber-200 hover:text-amber-100 hover:bg-amber-500/20"
            >
              <RefreshCw className="h-3 w-3" />
              Retry
            </Button>
          )}
        </div>
      </div>
    );
  }

  // Card variant (default)
  return (
    <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-6 space-y-4">
      <div className="flex items-start gap-4">
        <div className="w-12 h-12 rounded-full bg-amber-500/20 flex items-center justify-center flex-shrink-0">
          <AlertTriangle className="w-6 h-6 text-amber-400" />
        </div>
        <div className="flex-1 space-y-1">
          <h3 className="font-semibold text-amber-200">
            {context ? `Unable to load ${context}` : 'Data Validation Error'}
          </h3>
          <p className="text-sm text-amber-300/80">
            The data received doesn't match the expected format. This may be a temporary issue.
          </p>
        </div>
      </div>

      {shouldShowDetails && (
        <details className="text-sm text-amber-300/70 bg-black/20 rounded-lg overflow-hidden">
          <summary className="cursor-pointer hover:text-amber-300 px-4 py-2 bg-black/20">
            View technical details
          </summary>
          <div className="p-4 space-y-3">
            <div>
              <p className="text-xs uppercase tracking-wide text-amber-400/50 mb-1">Error</p>
              <pre className="whitespace-pre-wrap break-all text-xs">{error}</pre>
            </div>
            {rawData !== undefined && (
              <div>
                <p className="text-xs uppercase tracking-wide text-amber-400/50 mb-1">Raw Data</p>
                <pre className="whitespace-pre-wrap break-all text-xs max-h-40 overflow-auto">
                  {JSON.stringify(rawData, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </details>
      )}

      <div className="flex flex-wrap gap-2">
        {onRetry && (
          <Button
            onClick={onRetry}
            size="sm"
            className="gap-2 bg-amber-500/20 hover:bg-amber-500/30 text-amber-200"
          >
            <RefreshCw className="h-4 w-4" />
            Try Again
          </Button>
        )}
        {onReport && (
          <Button
            onClick={onReport}
            size="sm"
            variant="outline"
            className="gap-2 border-amber-500/30 text-amber-200 hover:bg-amber-500/10"
          >
            <Bug className="h-4 w-4" />
            Report Issue
          </Button>
        )}
      </div>
    </div>
  );
}

/**
 * Hook-friendly wrapper that conditionally renders children or error fallback.
 *
 * @example
 * const result = safeParse(schema, data, 'config');
 * return (
 *   <ValidationGuard result={result} context="configuration" onRetry={refetch}>
 *     {(validData) => <ConfigPanel config={validData} />}
 *   </ValidationGuard>
 * );
 */
interface ValidationGuardProps<T> {
  result: { success: true; data: T } | { success: false; error: string; raw?: unknown };
  children: (data: T) => React.ReactNode;
  context?: string;
  onRetry?: () => void;
  variant?: 'inline' | 'card' | 'minimal';
}

export function ValidationGuard<T>({
  result,
  children,
  context,
  onRetry,
  variant = 'card',
}: ValidationGuardProps<T>) {
  if (!result.success) {
    return (
      <ValidationErrorFallback
        error={result.error}
        context={context}
        onRetry={onRetry}
        variant={variant}
        rawData={'raw' in result ? result.raw : undefined}
      />
    );
  }

  return <>{children(result.data)}</>;
}

export default ValidationErrorFallback;
