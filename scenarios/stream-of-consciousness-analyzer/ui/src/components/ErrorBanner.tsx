// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
import { AlertTriangle, RefreshCw, X } from "lucide-react";
import { ApiRequestError } from "../lib/api";

interface Props {
  error: Error | null;
  onRetry?: () => void;
  onDismiss?: () => void;
}

function getUserMessage(error: Error): { message: string; canRetry: boolean } {
  if (error instanceof ApiRequestError) {
    return { message: error.message, canRetry: error.retryable };
  }
  if (error.message.includes("Failed to fetch") || error.message.includes("NetworkError")) {
    return { message: "Unable to reach the server. Check your connection and try again.", canRetry: true };
  }
  return { message: "Something went wrong. Please try again.", canRetry: true };
}

export function ErrorBanner({ error, onRetry, onDismiss }: Props) {
  if (!error) return null;

  const { message, canRetry } = getUserMessage(error);

  return (
    <div data-testid="error-banner" role="alert" className="flex items-center gap-2 px-3 py-2 bg-red-900/30 border border-red-500/30 rounded-md text-sm text-red-300">
      <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
      <span className="flex-1">{message}</span>
      {canRetry && onRetry && (
        <button
          data-testid="error-retry-btn"
          onClick={onRetry}
          className="p-1 rounded hover:bg-white/10 text-red-300 hover:text-white"
          aria-label="Retry"
        >
          <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      )}
      {onDismiss && (
        <button
          onClick={onDismiss}
          className="p-1 rounded hover:bg-white/10 text-red-300 hover:text-white"
          aria-label="Dismiss error"
        >
          <X className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      )}
    </div>
  );
}
