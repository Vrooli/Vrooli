// Reusable error display component with retry functionality
// [REQ:FAIL-SAFE-001] [REQ:UI-HEALTH-001]
import { AlertCircle, RefreshCw, WifiOff, Database, Clock } from "lucide-react";
import { Button } from "./ui/button";
import { APIError } from "../lib/api";

interface ErrorDisplayProps {
  error: Error | null;
  onRetry?: () => void;
  compact?: boolean; // For use in smaller containers
  title?: string;
}

// Type guard that works across module boundaries
function isAPIError(error: unknown): error is APIError {
  return (
    error !== null &&
    typeof error === 'object' &&
    'code' in error &&
    'statusCode' in error &&
    'isRetryable' in error &&
    'getUserMessage' in error &&
    typeof (error as APIError).getUserMessage === 'function'
  );
}

function getErrorIcon(error: Error | null) {
  if (isAPIError(error)) {
    switch (error.code) {
      case "NETWORK_ERROR":
        return WifiOff;
      case "DATABASE_ERROR":
        return Database;
      case "TIMEOUT":
        return Clock;
      default:
        return AlertCircle;
    }
  }
  return AlertCircle;
}

function getErrorColor(error: Error | null): string {
  if (isAPIError(error) && error.isRetryable) {
    return "text-amber-400"; // Retryable errors are warnings
  }
  return "text-red-400"; // Non-retryable errors are red
}

export function ErrorDisplay({ error, onRetry, compact = false, title }: ErrorDisplayProps) {
  const Icon = getErrorIcon(error);
  const colorClass = getErrorColor(error);

  // Extract user-friendly message and action
  let message: string;
  let action: string | undefined;
  let requestId: string | undefined;

  if (isAPIError(error)) {
    message = error.getUserMessage();
    action = error.getSuggestedAction();
    requestId = error.requestId;
  } else if (error) {
    message = error.message || "An unexpected error occurred";
  } else {
    message = "An unexpected error occurred";
  }

  if (compact) {
    return (
      <div className="flex items-center justify-between gap-2 py-2">
        <div className="flex items-center gap-2">
          <Icon size={14} className={colorClass} />
          <span className="text-sm text-slate-400">{message}</span>
        </div>
        {onRetry && (
          <button
            onClick={onRetry}
            className="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1"
          >
            <RefreshCw size={12} />
            Retry
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="text-center py-4">
      <Icon className={`mx-auto mb-3 ${colorClass}`} size={32} />
      {title && <h3 className="font-medium text-slate-200 mb-1">{title}</h3>}
      <p className="text-sm text-slate-400 mb-2">{message}</p>
      {action && <p className="text-xs text-slate-500 mb-3">{action}</p>}
      {requestId && (
        <p className="text-xs text-slate-600 mb-3">
          Request ID: <code className="bg-slate-800 px-1 rounded">{requestId}</code>
        </p>
      )}
      {onRetry && (
        <Button
          variant="outline"
          size="sm"
          onClick={onRetry}
          className="mt-2"
        >
          <RefreshCw className="mr-2 h-4 w-4" />
          Retry
        </Button>
      )}
    </div>
  );
}

// Simple inline error for small widgets
export function InlineError({
  message,
  onRetry,
}: {
  message: string;
  onRetry?: () => void;
}) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="text-red-400">{message}</span>
      {onRetry && (
        <button
          onClick={onRetry}
          className="text-blue-400 hover:text-blue-300 flex items-center gap-1"
        >
          <RefreshCw size={12} />
          <span>Retry</span>
        </button>
      )}
    </div>
  );
}
