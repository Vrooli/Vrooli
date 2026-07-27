import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle, RefreshCw, Home } from "lucide-react";
import { Button } from "./button";

interface Props {
  children: ReactNode;
  /** Optional fallback UI to render when an error occurs */
  fallback?: ReactNode;
  /** Section name for error display context */
  sectionName?: string;
  /** Callback when an error is caught - use for logging/telemetry */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  /** Whether to show a "Go Home" button that navigates to inventory view */
  showHomeButton?: boolean;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

/**
 * Error Boundary component that catches JavaScript errors in child component tree.
 *
 * Strategic placement guidelines (from ui-health.md):
 * - Wrap route-level views or pages
 * - Wrap complex feature panels (modals, sidebars, dashboards)
 * - Wrap components that render dynamic or external data
 * - Wrap areas with heavy computation or transformation logic
 *
 * @example
 * ```tsx
 * <ErrorBoundary sectionName="Build Status">
 *   <BuildStatusPanel />
 * </ErrorBoundary>
 * ```
 */
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    this.setState({ errorInfo });

    // Log error with context (but don't expose stack traces in production UI)
    console.error(
      `[ErrorBoundary${this.props.sectionName ? `: ${this.props.sectionName}` : ""}]`,
      error.message,
      {
        componentStack: errorInfo.componentStack,
        error,
      },
    );

    // Call optional error handler for telemetry/logging
    this.props.onError?.(error, errorInfo);
  }

  handleRetry = (): void => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  handleGoHome = (): void => {
    // Navigate to inventory view by updating URL
    const url = new URL(window.location.href);
    url.searchParams.set("view", "inventory");
    url.searchParams.delete("scenario");
    url.searchParams.delete("doc");
    window.location.href = url.toString();
  };

  render(): ReactNode {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const sectionLabel = this.props.sectionName
        ? `in ${this.props.sectionName}`
        : "";

      return (
        <div className="flex flex-col items-center justify-center gap-4 rounded-lg border border-red-800/50 bg-red-950/20 p-8 text-center">
          <div className="flex items-center gap-2 text-red-400">
            <AlertTriangle className="h-6 w-6" />
            <h2 className="text-lg font-semibold">
              Something went wrong {sectionLabel}
            </h2>
          </div>

          <p className="max-w-md text-sm text-slate-400">
            An unexpected error occurred. You can try again, or return to the
            scenario inventory if the problem persists.
          </p>

          {/* Show error message in development only */}
          {import.meta.env.DEV && this.state.error && (
            <details className="mt-2 w-full max-w-lg rounded border border-slate-700 bg-slate-900/50 p-2 text-left">
              <summary className="cursor-pointer text-xs text-slate-500 hover:text-slate-400">
                Error details (dev only)
              </summary>
              <pre className="mt-2 overflow-auto text-xs text-red-300">
                {this.state.error.message}
              </pre>
            </details>
          )}

          <div className="flex gap-3">
            <Button
              variant="outline"
              size="sm"
              onClick={this.handleRetry}
              className="gap-2"
            >
              <RefreshCw className="h-4 w-4" />
              Try Again
            </Button>

            {this.props.showHomeButton && (
              <Button
                variant="ghost"
                size="sm"
                onClick={this.handleGoHome}
                className="gap-2"
              >
                <Home className="h-4 w-4" />
                Go to Inventory
              </Button>
            )}
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

/**
 * Wrapper component for sections that should degrade gracefully.
 * Provides consistent error handling with section-specific context.
 */
export function SectionErrorBoundary({
  children,
  name,
  onError,
}: {
  children: ReactNode;
  name: string;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}): ReactNode {
  return (
    <ErrorBoundary sectionName={name} showHomeButton onError={onError}>
      {children}
    </ErrorBoundary>
  );
}
