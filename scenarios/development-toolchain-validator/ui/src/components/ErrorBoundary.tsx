import { Component, type ReactNode } from "react";
import { Button } from "./ui/button";

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

interface ErrorBoundaryProps {
  children: ReactNode;
  /** Optional custom fallback UI - defaults to built-in error display */
  fallback?: ReactNode;
  /** Optional callback when error occurs */
  onError?: (error: Error, errorInfo: { componentStack: string }) => void;
}

/**
 * Error Boundary for catching React rendering errors.
 *
 * Place this around major UI sections to isolate failures:
 * - Route-level views
 * - Feature panels (dashboards, modals)
 * - Components rendering external/dynamic data
 *
 * This prevents the entire app from crashing when a single
 * component fails, allowing users to recover gracefully.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: { componentStack: string }): void {
    // Log to console for debugging (never expose raw stack traces in production UI)
    console.error("[ErrorBoundary] Caught error:", error);
    console.error("[ErrorBoundary] Component stack:", errorInfo.componentStack);

    // Call optional error handler
    this.props.onError?.(error, errorInfo);
  }

  handleRetry = (): void => {
    this.setState({ hasError: false, error: null });
  };

  handleRefresh = (): void => {
    window.location.reload();
  };

  render(): ReactNode {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default fallback UI with recovery options
      return (
        <div className="flex flex-col items-center justify-center min-h-[200px] p-6 rounded-xl border border-red-500/20 bg-red-500/5">
          <div className="text-red-400 text-lg font-medium mb-2">
            Something went wrong
          </div>
          <p className="text-slate-400 text-sm text-center mb-4 max-w-md">
            {this.state.error?.message ?? "An unexpected error occurred while rendering this section."}
          </p>
          <div className="flex gap-3">
            <Button variant="outline" size="sm" onClick={this.handleRetry}>
              Try Again
            </Button>
            <Button variant="outline" size="sm" onClick={this.handleRefresh}>
              Refresh Page
            </Button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
