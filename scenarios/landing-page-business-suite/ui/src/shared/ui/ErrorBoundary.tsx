import { Component, type ReactNode, type ErrorInfo } from 'react';
import { AlertTriangle, RefreshCw, Home, ArrowLeft } from 'lucide-react';
import { Button } from './button';

interface ErrorBoundaryProps {
  children: ReactNode;
  /**
   * Optional fallback UI component to render when an error occurs.
   * If not provided, a default error UI will be shown.
   */
  fallback?: ReactNode | ((props: { error: Error; resetError: () => void }) => ReactNode);
  /**
   * Optional callback when an error is caught.
   * Use for logging to external services.
   */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  /**
   * Optional name/label for this boundary (useful for debugging)
   */
  name?: string;
  /**
   * The level of the boundary - affects the default fallback UI:
   * - 'app': Full page error with navigation options
   * - 'route': Page-level error with back/home navigation
   * - 'section': Section-level error with retry option
   * - 'component': Minimal error message with retry
   */
  level?: 'app' | 'route' | 'section' | 'component';
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * React Error Boundary component for graceful error handling.
 *
 * Strategic placement guidelines:
 * - Wrap route-level views/pages (level='route')
 * - Wrap complex feature panels like modals, sidebars, dashboards (level='section')
 * - Wrap components that render dynamic or external data (level='component')
 * - Wrap areas with heavy computation or transformation logic (level='component')
 *
 * @example
 * // Route-level boundary
 * <ErrorBoundary level="route" name="AdminHome">
 *   <AdminHome />
 * </ErrorBoundary>
 *
 * @example
 * // Section-level boundary with custom fallback
 * <ErrorBoundary
 *   level="section"
 *   fallback={({ resetError }) => <RetrySection onRetry={resetError} />}
 * >
 *   <DataGrid data={complexData} />
 * </ErrorBoundary>
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
    // Log error with context for debugging
    const boundaryName = this.props.name ?? 'unnamed';
    const level = this.props.level ?? 'component';

    console.error(
      `[ErrorBoundary:${boundaryName}] Error caught at ${level} level:`,
      {
        error: error.message,
        stack: error.stack,
        componentStack: errorInfo.componentStack,
      }
    );

    // Call optional error handler
    this.props.onError?.(error, errorInfo);
  }

  resetError = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): ReactNode {
    if (this.state.hasError && this.state.error) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        if (typeof this.props.fallback === 'function') {
          return this.props.fallback({
            error: this.state.error,
            resetError: this.resetError
          });
        }
        return this.props.fallback;
      }

      // Use default fallback based on level
      const level = this.props.level ?? 'component';
      return (
        <DefaultErrorFallback
          error={this.state.error}
          level={level}
          onRetry={this.resetError}
        />
      );
    }

    return this.props.children;
  }
}

interface DefaultErrorFallbackProps {
  error: Error;
  level: 'app' | 'route' | 'section' | 'component';
  onRetry: () => void;
}

function DefaultErrorFallback({ error, level, onRetry }: DefaultErrorFallbackProps) {
  const handleGoHome = () => {
    window.location.href = '/';
  };

  const handleGoBack = () => {
    window.history.back();
  };

  const handleRefresh = () => {
    window.location.reload();
  };

  // Sanitize error message for display (avoid exposing sensitive info)
  const userFriendlyMessage = getUserFriendlyMessage(error);

  if (level === 'app') {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center px-6">
        <div className="max-w-lg w-full rounded-2xl border border-red-500/20 bg-red-500/10 p-8 text-center space-y-6">
          <div className="w-20 h-20 rounded-full bg-red-500/20 flex items-center justify-center mx-auto">
            <AlertTriangle className="w-10 h-10 text-red-400" />
          </div>
          <div className="space-y-2">
            <h1 className="text-2xl font-bold text-red-300">Application Error</h1>
            <p className="text-red-200/80">
              Something went wrong and we couldn't recover automatically.
            </p>
            <p className="text-sm text-red-300/60">{userFriendlyMessage}</p>
          </div>
          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            <Button
              onClick={handleRefresh}
              className="gap-2 bg-red-500/20 hover:bg-red-500/30 text-red-200"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh Page
            </Button>
            <Button
              onClick={handleGoHome}
              variant="outline"
              className="gap-2 border-red-500/30 text-red-200 hover:bg-red-500/10"
            >
              <Home className="h-4 w-4" />
              Go Home
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (level === 'route') {
    return (
      <div className="min-h-[400px] flex items-center justify-center px-6 py-12">
        <div className="max-w-md w-full rounded-2xl border border-red-500/20 bg-red-500/10 p-6 text-center space-y-4">
          <div className="w-16 h-16 rounded-full bg-red-500/20 flex items-center justify-center mx-auto">
            <AlertTriangle className="w-8 h-8 text-red-400" />
          </div>
          <div className="space-y-2">
            <h2 className="text-xl font-semibold text-red-300">Page Error</h2>
            <p className="text-red-200/80 text-sm">
              This page encountered an error. You can try again or navigate elsewhere.
            </p>
            <p className="text-xs text-red-300/50">{userFriendlyMessage}</p>
          </div>
          <div className="flex flex-col sm:flex-row gap-2 justify-center">
            <Button
              onClick={onRetry}
              size="sm"
              className="gap-2 bg-red-500/20 hover:bg-red-500/30 text-red-200"
            >
              <RefreshCw className="h-4 w-4" />
              Try Again
            </Button>
            <Button
              onClick={handleGoBack}
              variant="outline"
              size="sm"
              className="gap-2 border-red-500/30 text-red-200 hover:bg-red-500/10"
            >
              <ArrowLeft className="h-4 w-4" />
              Go Back
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (level === 'section') {
    return (
      <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4 space-y-3">
        <div className="flex items-start gap-3">
          <AlertTriangle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div className="space-y-1">
            <p className="font-medium text-red-300">This section couldn't load</p>
            <p className="text-sm text-red-200/70">{userFriendlyMessage}</p>
          </div>
        </div>
        <Button
          onClick={onRetry}
          size="sm"
          variant="outline"
          className="gap-2 border-red-500/30 text-red-200 hover:bg-red-500/10"
        >
          <RefreshCw className="h-3 w-3" />
          Retry
        </Button>
      </div>
    );
  }

  // Component level - minimal inline error
  return (
    <div className="inline-flex items-center gap-2 rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-sm">
      <AlertTriangle className="w-4 h-4 text-red-400 flex-shrink-0" />
      <span className="text-red-200/80">Error loading component</span>
      <button
        onClick={onRetry}
        className="text-red-300 hover:text-red-200 underline underline-offset-2"
        type="button"
      >
        Retry
      </button>
    </div>
  );
}

/**
 * Converts error messages to user-friendly text.
 * Avoids exposing stack traces or sensitive information in production.
 */
function getUserFriendlyMessage(error: Error): string {
  const message = error.message ?? '';

  // Common React errors with user-friendly alternatives
  const errorMappings: Array<[RegExp, string]> = [
    [/Cannot read propert(y|ies) of (undefined|null)/i, 'Missing data or configuration'],
    [/is not a function/i, 'Unexpected data format'],
    [/is not defined/i, 'Missing resource or configuration'],
    [/Maximum update depth exceeded/i, 'Display loop detected'],
    [/Rendered more hooks than during the previous render/i, 'Component rendering issue'],
    [/Invalid hook call/i, 'Component structure error'],
    [/Objects are not valid as a React child/i, 'Invalid display data'],
    [/Network Error/i, 'Network connection issue'],
    [/timeout/i, 'Request timed out'],
    [/401|403|Unauthorized/i, 'Authentication required'],
    [/404|Not Found/i, 'Resource not found'],
    [/500|Internal Server Error/i, 'Server error'],
  ];

  for (const [pattern, friendlyMessage] of errorMappings) {
    if (pattern.test(message)) {
      return friendlyMessage;
    }
  }

  // For unrecognized errors, return a generic message
  // Don't expose raw error message which might contain sensitive info
  if (message.length > 100 || message.includes('at ') || message.includes('Error:')) {
    return 'An unexpected error occurred';
  }

  return message || 'An unexpected error occurred';
}

export default ErrorBoundary;
