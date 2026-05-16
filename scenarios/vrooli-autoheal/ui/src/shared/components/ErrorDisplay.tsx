// Reusable error display component with retry functionality
// [REQ:FAIL-SAFE-001] [REQ:UI-HEALTH-001]
import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertCircle, RefreshCw, WifiOff, Database, Clock, Ban, Timer } from "lucide-react";
import { Button } from "../ui/primitives";
import { APIError } from "../../lib/api";

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
      case "CONFLICT":
        return Timer;
      case "VALIDATION_ERROR":
        return Ban;
      default:
        return AlertCircle;
    }
  }
  return AlertCircle;
}

function getErrorColor(error: Error | null): string {
  if (isAPIError(error) && error.isRetryable) {
    return "text-accent-warning";
  }
  return "text-accent-danger";
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
          <span className="text-sm text-text-muted">{message}</span>
        </div>
        {onRetry && (
          <button
            onClick={onRetry}
            className="flex items-center gap-1 text-xs text-accent-primary hover:text-accent-primary/80"
          >
            <RefreshCw size={12} />
            Retry
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="py-4 text-center">
      <Icon className={`mx-auto mb-3 ${colorClass}`} size={32} />
      {title && <h3 className="mb-1 font-medium text-text-primary">{title}</h3>}
      <p className="mb-2 text-sm text-text-muted">{message}</p>
      {action && <p className="mb-3 text-xs text-text-muted/80">{action}</p>}
      {requestId && (
        <p className="mb-3 text-xs text-text-muted/70">
          Request ID: <code className="rounded bg-surface-overlay px-1">{requestId}</code>
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
      <span className="text-accent-danger">{message}</span>
      {onRetry && (
        <button
          onClick={onRetry}
          className="flex items-center gap-1 text-accent-primary hover:text-accent-primary/80"
        >
          <RefreshCw size={12} />
          <span>Retry</span>
        </button>
      )}
    </div>
  );
}

interface ReactErrorBoundaryProps {
  children: ReactNode;
  sectionName?: string;
}

interface ReactErrorBoundaryState {
  hasError: boolean;
}

export class ReactErrorBoundary extends Component<ReactErrorBoundaryProps, ReactErrorBoundaryState> {
  state: ReactErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): ReactErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error("React section crashed", {
      section: this.props.sectionName ?? "unknown",
      message: error.message,
      stack: error.stack,
      componentStack: info.componentStack,
    });
  }

  private handleRetry = (): void => {
    this.setState({ hasError: false });
  };

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div className="rounded-xl border border-accent-danger/20 bg-accent-danger/10 p-4">
          <p className="text-sm font-medium text-accent-danger">
            {this.props.sectionName ?? "This section"} failed to render.
          </p>
          <p className="mt-1 text-xs text-accent-danger/80">
            Try reloading this section. If it keeps failing, refresh the page.
          </p>
          <button
            onClick={this.handleRetry}
            className="mt-3 rounded border border-accent-danger/30 px-3 py-1 text-xs text-accent-danger hover:bg-accent-danger/10"
          >
            Retry Section
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
