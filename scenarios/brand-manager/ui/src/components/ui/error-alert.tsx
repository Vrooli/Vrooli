import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "./button";
import { cn } from "../../lib/utils";
import { ApiRequestError } from "../../lib/api";

interface ErrorAlertProps {
  error: Error | null;
  /** Fallback message when the error is not an ApiRequestError */
  fallbackMessage?: string;
  /** Fallback recovery hint */
  fallbackRecovery?: string;
  /** Called when the user clicks Retry */
  onRetry?: () => void;
  /** Additional CSS classes */
  className?: string;
  /** data-testid */
  testId?: string;
}

/**
 * Extracts structured error info from ApiRequestError and renders a
 * consistent error alert with message, recovery hint, and optional retry.
 *
 * Eliminates the repeated pattern of checking `instanceof ApiRequestError`
 * and rendering AlertTriangle + message + recovery across every page.
 */
export function ErrorAlert({
  error,
  fallbackMessage = "An unexpected error occurred.",
  fallbackRecovery,
  onRetry,
  className,
  testId,
}: ErrorAlertProps) {
  const isApiError = error instanceof ApiRequestError;
  const message = isApiError ? error.message : fallbackMessage;
  const recovery = isApiError ? error.recovery : fallbackRecovery;
  const isRetryable = onRetry && (!isApiError || error.isRetryable);

  return (
    <div
      className={cn("rounded-lg border border-red-500/20 bg-red-500/10 p-4", className)}
      data-testid={testId}
    >
      <div className="flex items-start gap-3">
        <AlertTriangle className="h-5 w-5 text-red-400 shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="text-red-400 font-medium">{message}</p>
          {recovery && (
            <p className="text-red-400/70 text-sm mt-1">{recovery}</p>
          )}
          {isRetryable && (
            <Button
              variant="outline"
              size="sm"
              onClick={onRetry}
              className="mt-3"
            >
              <RefreshCw className="mr-1 h-3 w-3" /> Retry
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
