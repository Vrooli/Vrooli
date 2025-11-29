import React, { Component, ErrorInfo, ReactNode } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "./ui/button";

/**
 * Props for the ErrorBoundary component
 */
interface ErrorBoundaryProps {
  children: ReactNode;
  /** Optional fallback UI to show on error */
  fallback?: ReactNode;
  /** Context name for logging (e.g., "Dashboard", "ScenarioDetail") */
  context?: string;
  /** Callback when an error is caught */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

/**
 * State for the ErrorBoundary component
 */
interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

/**
 * ErrorBoundary component for graceful component-level failure handling
 * [REQ:SCS-CORE-003] Graceful degradation at the UI component level
 *
 * Usage:
 * ```tsx
 * <ErrorBoundary context="Dashboard">
 *   <Dashboard />
 * </ErrorBoundary>
 * ```
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    const { context = "Unknown", onError } = this.props;

    // Log error with context for debugging
    console.error(`[ErrorBoundary:${context}] Component error:`, error);
    console.error(`[ErrorBoundary:${context}] Component stack:`, errorInfo.componentStack);

    this.setState({ errorInfo });

    // Call optional error callback
    if (onError) {
      onError(error, errorInfo);
    }
  }

  handleRetry = (): void => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  render(): ReactNode {
    const { hasError, error } = this.state;
    const { children, fallback, context = "this section" } = this.props;

    if (hasError) {
      // If custom fallback provided, use it
      if (fallback) {
        return fallback;
      }

      // Default error UI
      return (
        <div
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-6 m-4"
          data-testid="error-boundary-fallback"
        >
          <div className="flex items-start gap-4">
            <AlertTriangle className="h-6 w-6 text-red-400 mt-0.5 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <h3 className="text-lg font-medium text-red-200">
                Something went wrong in {context}
              </h3>
              <p className="text-sm text-red-300/70 mt-1">
                An unexpected error occurred while rendering this component.
                This has been logged for investigation.
              </p>
              {error && (
                <details className="mt-3">
                  <summary className="text-xs text-red-300/50 cursor-pointer hover:text-red-300/70">
                    Technical details
                  </summary>
                  <pre className="mt-2 text-xs text-red-300/60 bg-black/20 p-2 rounded overflow-auto max-h-32">
                    {error.message}
                  </pre>
                </details>
              )}
              <div className="mt-4 flex gap-3">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={this.handleRetry}
                  className="border-red-400/30 text-red-300 hover:bg-red-500/10"
                >
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Try Again
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => window.location.reload()}
                  className="border-slate-400/30 text-slate-300 hover:bg-slate-500/10"
                >
                  Reload Page
                </Button>
              </div>
            </div>
          </div>
        </div>
      );
    }

    return children;
  }
}

/**
 * HOC to wrap a component with an error boundary
 * [REQ:SCS-CORE-003] Convenient way to add error boundaries
 *
 * Usage:
 * ```tsx
 * const SafeDashboard = withErrorBoundary(Dashboard, "Dashboard");
 * ```
 */
export function withErrorBoundary<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  context: string
): React.FC<P> {
  const WithErrorBoundary: React.FC<P> = (props) => (
    <ErrorBoundary context={context}>
      <WrappedComponent {...props} />
    </ErrorBoundary>
  );

  WithErrorBoundary.displayName = `withErrorBoundary(${WrappedComponent.displayName || WrappedComponent.name || "Component"})`;

  return WithErrorBoundary;
}
