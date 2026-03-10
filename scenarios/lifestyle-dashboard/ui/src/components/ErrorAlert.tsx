/**
 * Error Alert Component
 *
 * Displays structured error messages with recovery hints based on error category.
 * Provides appropriate actions (retry, back, help) based on error type.
 */
import { AlertCircle, RefreshCw, ArrowLeft, HelpCircle } from "lucide-react";
import { APIError } from "../lib/api";

interface ErrorAlertProps {
  error: Error | null;
  onRetry?: () => void;
  onBack?: () => void;
  className?: string;
}

/**
 * Maps error categories to user-friendly titles
 */
function getErrorTitle(error: Error): string {
  if (error instanceof APIError) {
    switch (error.category) {
      case "validation":
        return "Invalid Request";
      case "not_found":
        return "Not Found";
      case "conflict":
        return "Conflict";
      case "unavailable":
        return "Service Unavailable";
      case "internal":
      default:
        return "Something Went Wrong";
    }
  }
  // Network or other errors
  if (error.message.includes("Failed to fetch") || error.message.includes("NetworkError")) {
    return "Connection Error";
  }
  return "Error";
}

/**
 * Determines the appropriate recovery action based on error type
 */
function getRecoveryAction(error: Error): "retry" | "back" | "help" {
  if (error instanceof APIError) {
    if (error.isRetryable) return "retry";
    if (error.isNotFound) return "back";
    if (error.isValidation) return "help";
  }
  // Network errors are retryable
  if (error.message.includes("Failed to fetch") || error.message.includes("NetworkError")) {
    return "retry";
  }
  return "retry";
}

/**
 * Gets user-friendly message with recovery hint
 */
function getErrorMessage(error: Error): { message: string; hint?: string } {
  if (error instanceof APIError) {
    return {
      message: error.message,
      hint: error.recovery || getDefaultHint(error.category),
    };
  }

  // Network errors
  if (error.message.includes("Failed to fetch") || error.message.includes("NetworkError")) {
    return {
      message: "Unable to connect to the API",
      hint: "Make sure the scenario is running with: vrooli scenario start lifestyle-dashboard",
    };
  }

  return { message: error.message };
}

function getDefaultHint(category: string): string | undefined {
  switch (category) {
    case "validation":
      return "Please check your input and try again";
    case "not_found":
      return "The requested resource may have been deleted or never existed";
    case "unavailable":
      return "The service is temporarily unavailable. Please try again later.";
    case "internal":
      return "Please try again. If the problem persists, check the scenario logs.";
    default:
      return undefined;
  }
}

export function ErrorAlert({ error, onRetry, onBack, className = "" }: ErrorAlertProps) {
  if (!error) return null;

  const title = getErrorTitle(error);
  const { message, hint } = getErrorMessage(error);
  const action = getRecoveryAction(error);

  return (
    <div className={`rounded-xl border border-red-500/30 bg-red-500/10 p-4 ${className}`}>
      <div className="flex items-start gap-3">
        <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <h3 className="font-medium text-red-400">{title}</h3>
          <p className="mt-1 text-sm text-slate-300">{message}</p>
          {hint && (
            <p className="mt-2 text-sm text-slate-400">{hint}</p>
          )}

          {/* Recovery actions */}
          <div className="mt-3 flex flex-wrap gap-2">
            {action === "retry" && onRetry && (
              <button
                onClick={onRetry}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-500/20 hover:bg-red-500/30 text-red-300 text-sm transition-colors"
              >
                <RefreshCw className="w-4 h-4" />
                Try Again
              </button>
            )}
            {action === "back" && onBack && (
              <button
                onClick={onBack}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-500/20 hover:bg-slate-500/30 text-slate-300 text-sm transition-colors"
              >
                <ArrowLeft className="w-4 h-4" />
                Go Back
              </button>
            )}
            {action === "help" && (
              <a
                href="https://github.com/anthropics/claude-code/issues"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-500/20 hover:bg-slate-500/30 text-slate-300 text-sm transition-colors"
              >
                <HelpCircle className="w-4 h-4" />
                Get Help
              </a>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
