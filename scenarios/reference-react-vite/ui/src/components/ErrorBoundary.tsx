import { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "./ui/button";

interface ErrorBoundaryProps {
  /** Child components to wrap */
  children: ReactNode;
  /** Optional fallback UI - if not provided, uses default error display */
  fallback?: ReactNode;
  /** Error handler callback for logging/telemetry */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Error Boundary component that catches JavaScript errors anywhere in its child
 * component tree and displays a fallback UI instead of crashing the whole app.
 *
 * Usage:
 * ```tsx
 * <ErrorBoundary>
 *   <MyComponent />
 * </ErrorBoundary>
 * ```
 *
 * With custom fallback:
 * ```tsx
 * <ErrorBoundary fallback={<CustomErrorDisplay />}>
 *   <MyComponent />
 * </ErrorBoundary>
 * ```
 *
 * IMPORTANT: Error boundaries do NOT catch errors in:
 * - Event handlers (use try/catch)
 * - Async code (use try/catch or .catch())
 * - Server-side rendering
 * - Errors thrown in the error boundary itself
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Log to console with context for debugging
    console.error("[ErrorBoundary] Caught error:", error);
    console.error("[ErrorBoundary] Component stack:", errorInfo.componentStack);

    // Call optional error handler (for telemetry/logging services)
    this.props.onError?.(error, errorInfo);
  }

  handleReset = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): ReactNode {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return (
        <div className="flex min-h-[200px] flex-col items-center justify-center rounded-lg border border-red-500/20 bg-red-500/5 p-6">
          <div className="text-center">
            <h2 className="text-lg font-semibold text-red-400">Something went wrong</h2>
            <p className="mt-2 text-sm text-slate-400">
              An error occurred while rendering this component.
            </p>
            {this.state.error && (
              <p className="mt-2 max-w-md truncate text-xs text-slate-500">
                {this.state.error.message}
              </p>
            )}
            <Button
              onClick={this.handleReset}
              className="mt-4"
              variant="outline"
            >
              Try Again
            </Button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
