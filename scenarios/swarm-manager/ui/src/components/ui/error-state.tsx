/**
 * Error State Component
 *
 * A reusable error state UI that provides user-friendly error messages
 * and recovery actions. Designed for graceful degradation - clearly
 * communicates what went wrong and what the user can do about it.
 *
 * Key principles:
 * - Never expose technical details (stack traces, URLs, etc.)
 * - Always provide a clear next action
 * - Differentiate between "no data" and "error loading data"
 *
 * Uses the centralized categorizeError function from error-utils.ts
 * to ensure consistent error classification across the application.
 */

import { AlertCircle, WifiOff, Clock, ServerCrash, RefreshCw } from "lucide-react";
import { Button } from "./button";
import { selectors } from "../../consts/selectors";
import { isApiError } from "../../lib/api-client";
import { categorizeError, type ErrorCategory } from "../../lib/error-utils";

export type ErrorVariant = "network" | "timeout" | "server" | "notFound" | "generic";

interface ErrorStateProps {
  /** The error to display - can be an ApiError or a generic Error */
  error?: Error | null;
  /** Override the automatic variant detection */
  variant?: ErrorVariant;
  /** Custom title to override the default */
  title?: string;
  /** Custom message to override the default */
  message?: string;
  /** Callback when retry button is clicked */
  onRetry?: () => void;
  /** Hide the retry button */
  hideRetry?: boolean;
  /** Additional CSS classes */
  className?: string;
}

interface ErrorDisplay {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  message: string;
  showRetry: boolean;
}

/**
 * Maps ErrorCategory from error-utils.ts to ErrorVariant for display.
 * This ensures we use a single source of truth for error classification.
 */
const CATEGORY_TO_VARIANT: Record<ErrorCategory, ErrorVariant> = {
  NETWORK: "network",
  TIMEOUT: "timeout",
  AUTH: "generic",      // Auth errors use generic display with specific messaging
  NOT_FOUND: "notFound",
  SERVER: "server",
  VALIDATION: "generic", // Validation errors use generic display
  PARSE: "server",       // Parse errors are treated as server issues
  STALE_CHUNK: "generic", // Deploy artifact; boundaries render the reload-specific UI
  RUNTIME: "generic",
};

/**
 * Determines the error variant using the centralized categorizeError function.
 */
function getVariantFromError(error: Error | null | undefined): ErrorVariant {
  if (!error) return "generic";
  const category = categorizeError(error);
  return CATEGORY_TO_VARIANT[category];
}

/**
 * Returns display configuration for each error variant.
 */
function getErrorDisplay(variant: ErrorVariant, error?: Error | null): ErrorDisplay {
  switch (variant) {
    case "network":
      return {
        icon: WifiOff,
        title: "Connection problem",
        message: "Unable to reach the server. Please check your internet connection and try again.",
        showRetry: true,
      };
    case "timeout":
      return {
        icon: Clock,
        title: "Request timed out",
        message: "The server is taking too long to respond. It may be busy - please try again in a moment.",
        showRetry: true,
      };
    case "server":
      return {
        icon: ServerCrash,
        title: "Server error",
        message: "The server encountered a problem. Our team has been notified. Please try again later.",
        showRetry: true,
      };
    case "notFound":
      return {
        icon: AlertCircle,
        title: "Not found",
        message: "The requested resource could not be found. It may have been moved or deleted.",
        showRetry: false,
      };
    case "generic":
    default: {
      // Use ApiError.userMessage if available, otherwise generic message
      const userMessage = isApiError(error)
        ? error.userMessage
        : "Something went wrong while loading the data. Please try again.";
      return {
        icon: AlertCircle,
        title: "Something went wrong",
        message: userMessage,
        showRetry: true,
      };
    }
  }
}

/**
 * ErrorState displays a user-friendly error message with recovery options.
 *
 * Usage:
 * ```tsx
 * // Basic usage with automatic error detection
 * <ErrorState error={queryError} onRetry={refetch} />
 *
 * // Override with custom messaging
 * <ErrorState
 *   variant="network"
 *   title="Can't load backlog"
 *   message="Check your connection and try again."
 *   onRetry={refetch}
 * />
 * ```
 */
export function ErrorState({
  error,
  variant,
  title,
  message,
  onRetry,
  hideRetry = false,
  className = "",
}: ErrorStateProps) {
  // Auto-detect variant if not provided
  const detectedVariant = variant ?? getVariantFromError(error);
  const display = getErrorDisplay(detectedVariant, error);

  const Icon = display.icon;
  const finalTitle = title ?? display.title;
  const finalMessage = message ?? display.message;
  const showRetryButton = !hideRetry && display.showRetry && onRetry;

  return (
    <div
      className={`rounded-xl border border-red-500/20 bg-red-500/5 p-8 text-center ${className}`}
      data-testid={selectors.error.container}
    >
      <Icon
        className="mx-auto h-12 w-12 text-red-400"
        data-testid={selectors.error.icon}
      />
      <h3
        className="mt-4 text-lg font-medium text-slate-200"
        data-testid={selectors.error.title}
      >
        {finalTitle}
      </h3>
      <p
        className="mt-2 text-sm text-slate-400"
        data-testid={selectors.error.message}
      >
        {finalMessage}
      </p>
      {showRetryButton && (
        <Button
          variant="outline"
          className="mt-4"
          onClick={onRetry}
          data-testid={selectors.error.retryButton}
        >
          <RefreshCw className="mr-2 h-4 w-4" />
          Try again
        </Button>
      )}
    </div>
  );
}
