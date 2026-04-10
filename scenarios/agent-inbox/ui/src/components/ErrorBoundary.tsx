import { Component, type ReactNode, type ErrorInfo } from "react";
import { AlertTriangle, RefreshCw, Home } from "lucide-react";

/**
 * Props for the ErrorBoundary component.
 */
interface ErrorBoundaryProps {
  /** Child components to render */
  children: ReactNode;
  /** Optional fallback UI to render on error */
  fallback?: ReactNode;
  /** Optional callback when an error is caught */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  /** Optional name for debugging/logging which boundary caught the error */
  name?: string;
  /** Whether this is a critical boundary (shows more prominent error UI) */
  critical?: boolean;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

/**
 * Error Boundary component that catches JavaScript errors in child components,
 * logs them, and displays a fallback UI instead of crashing the entire app.
 *
 * USAGE GUIDELINES:
 * - Wrap major UI sections (routes, feature panels, modals)
 * - Use `critical` prop for top-level boundaries
 * - Provide meaningful `name` prop for debugging
 * - Use `fallback` for custom error UIs in specific contexts
 *
 * PLACEMENT STRATEGY:
 * 1. Top-level (App) - catches everything that escapes other boundaries
 * 2. Route/View level - isolates page crashes
 * 3. Feature panels - isolates specific features (sidebar, chat view, etc.)
 * 4. Data-dependent components - isolates API data rendering issues
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
    const { onError, name } = this.props;

    // Log error with context
    console.error(
      `[ErrorBoundary${name ? `:${name}` : ""}] Caught error:`,
      error,
      errorInfo
    );

    this.setState({ errorInfo });

    // Call custom error handler if provided
    onError?.(error, errorInfo);
  }

  handleRetry = (): void => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  handleGoHome = (): void => {
    // Clear error state and navigate to root
    this.setState({ hasError: false, error: null, errorInfo: null });
    // Use history API for navigation without full page reload
    window.history.pushState({}, "", "/");
    window.dispatchEvent(new PopStateEvent("popstate"));
  };

  handleRefresh = (): void => {
    window.location.reload();
  };

  render(): ReactNode {
    const { hasError, error } = this.state;
    const { children, fallback, critical = false, name } = this.props;

    if (!hasError) {
      return children;
    }

    // Use custom fallback if provided
    if (fallback) {
      return fallback;
    }

    // Default fallback UI
    if (critical) {
      // Critical error UI - more prominent for app-level failures
      return (
        <div
          className="min-h-screen bg-slate-950 flex items-center justify-center p-4"
          data-testid="error-boundary-critical"
        >
          <div className="max-w-md w-full text-center">
            <div className="w-16 h-16 rounded-2xl bg-red-500/20 flex items-center justify-center mx-auto mb-4">
              <AlertTriangle className="h-8 w-8 text-red-400" />
            </div>
            <h1 className="text-xl font-semibold text-white mb-2">
              Something went wrong
            </h1>
            <p className="text-sm text-slate-400 mb-6">
              The application encountered an unexpected error. You can try
              refreshing the page or returning to the home screen.
            </p>
            {import.meta.env.DEV && error && (
              <div className="mb-6 p-3 bg-slate-900 rounded-lg text-left overflow-auto max-h-40">
                <p className="text-xs text-red-400 font-mono break-all">
                  {error.message}
                </p>
              </div>
            )}
            <div className="flex flex-col sm:flex-row gap-3 justify-center">
              <button
                onClick={this.handleRefresh}
                className="inline-flex items-center justify-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg transition-colors"
              >
                <RefreshCw className="h-4 w-4" />
                Refresh Page
              </button>
              <button
                onClick={this.handleGoHome}
                className="inline-flex items-center justify-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-lg transition-colors"
              >
                <Home className="h-4 w-4" />
                Go Home
              </button>
            </div>
          </div>
        </div>
      );
    }

    // Non-critical error UI - less prominent for component-level failures
    return (
      <div
        className="flex items-center justify-center p-4 bg-slate-900/50 rounded-lg border border-red-500/20"
        data-testid={`error-boundary${name ? `-${name}` : ""}`}
      >
        <div className="text-center max-w-sm">
          <div className="w-10 h-10 rounded-xl bg-red-500/20 flex items-center justify-center mx-auto mb-3">
            <AlertTriangle className="h-5 w-5 text-red-400" />
          </div>
          <p className="text-sm text-slate-400 mb-3">
            This section encountered an error.
          </p>
          {import.meta.env.DEV && error && (
            <p className="text-xs text-red-400 font-mono mb-3 break-all">
              {error.message}
            </p>
          )}
          <button
            onClick={this.handleRetry}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm bg-slate-800 hover:bg-slate-700 text-white rounded-md transition-colors"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Try Again
          </button>
        </div>
      </div>
    );
  }
}

/**
 * Higher-order component to wrap a component with an error boundary.
 * Useful for wrapping components that receive external data.
 */
export function withErrorBoundary<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  boundaryProps?: Omit<ErrorBoundaryProps, "children">
): React.ComponentType<P> {
  const displayName =
    WrappedComponent.displayName || WrappedComponent.name || "Component";

  const WithErrorBoundary = (props: P) => (
    <ErrorBoundary name={displayName} {...boundaryProps}>
      <WrappedComponent {...props} />
    </ErrorBoundary>
  );

  WithErrorBoundary.displayName = `WithErrorBoundary(${displayName})`;

  return WithErrorBoundary;
}
