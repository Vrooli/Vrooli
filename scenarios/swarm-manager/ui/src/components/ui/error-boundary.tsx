/**
 * Error Boundary Component
 *
 * Catches runtime errors in React components and displays a user-friendly
 * fallback UI instead of crashing the entire application.
 *
 * ╔════════════════════════════════════════════════════════════════╗
 * ║  ERROR RECOVERY DESIGN                                         ║
 * ║                                                                ║
 * ║  This boundary catches unhandled exceptions from React render  ║
 * ║  and lifecycle methods. It does NOT catch:                     ║
 * ║  - Event handler errors (use try/catch)                        ║
 * ║  - Async errors (use .catch() or try/catch in async functions) ║
 * ║  - Server-side rendering errors                                ║
 * ║                                                                ║
 * ║  Recovery path: Full page refresh (clears all React state)     ║
 * ╚════════════════════════════════════════════════════════════════╝
 *
 * Key principles:
 * - Never expose stack traces or internal details to users
 * - Always provide a clear recovery action (refresh)
 * - Log errors for observability before clearing details
 */

import { Component, type ReactNode } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "./button";
import { selectors } from "../../consts/selectors";
import { generateUniqueId } from "../../lib/error-utils";

interface ErrorBoundaryProps {
  /** Child components to render */
  children: ReactNode;
  /** Optional custom fallback UI */
  fallback?: ReactNode;
  /** Optional callback when error occurs (for logging) */
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  /** Error ID for correlation with logs (not exposed to user) */
  errorId: string | null;
}

/** Error ID prefix for app-level error boundary */
const ERROR_ID_PREFIX = "err";

/**
 * ErrorBoundary catches runtime errors and displays a recovery UI.
 *
 * Usage:
 * ```tsx
 * <ErrorBoundary>
 *   <App />
 * </ErrorBoundary>
 *
 * // With error callback for logging
 * <ErrorBoundary onError={(error, info) => logToService(error)}>
 *   <App />
 * </ErrorBoundary>
 * ```
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, errorId: null };
  }

  static getDerivedStateFromError(): ErrorBoundaryState {
    // Update state so the next render shows the fallback UI
    return { hasError: true, errorId: generateUniqueId(ERROR_ID_PREFIX) };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Log error for observability (structured format for parsing)
    console.error(
      "[ErrorBoundary] Runtime error caught:",
      JSON.stringify({
        errorId: this.state.errorId,
        name: error.name,
        message: error.message,
        componentStack: errorInfo.componentStack,
        timestamp: new Date().toISOString(),
      })
    );

    // Call optional error callback (for external logging services)
    this.props.onError?.(error, errorInfo);
  }

  handleRefresh = () => {
    // Full page refresh to clear all React state
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      // Custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default fallback UI
      return (
        <div
          className="flex min-h-screen items-center justify-center bg-slate-900 p-4"
          data-testid={selectors.errorBoundary.container}
        >
          <div className="max-w-md text-center">
            <AlertTriangle className="mx-auto h-16 w-16 text-amber-400" />
            <h1
              className="mt-6 text-2xl font-semibold text-slate-100"
              data-testid={selectors.errorBoundary.title}
            >
              Something went wrong
            </h1>
            <p
              className="mt-3 text-slate-400"
              data-testid={selectors.errorBoundary.message}
            >
              The application encountered an unexpected error. Please refresh the page to continue.
            </p>
            <Button
              className="mt-6"
              onClick={this.handleRefresh}
              data-testid={selectors.errorBoundary.refreshButton}
            >
              <RefreshCw className="mr-2 h-4 w-4" />
              Refresh Page
            </Button>
            {/* Error ID hidden but available in DOM for support correlation */}
            <p className="mt-4 text-xs text-slate-600">
              Error ID: {this.state.errorId}
            </p>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
