/**
 * ErrorBoundary - Application-wide error boundary component.
 *
 * Provides a user-friendly error UI for React Router errors and
 * uncaught exceptions. Displays the error details in development
 * and a clean error message in production.
 */
import { useRouteError, isRouteErrorResponse, useNavigate } from 'react-router-dom';
import { AlertTriangle, RefreshCw, Home, Bug } from 'lucide-react';

interface ErrorDetails {
  title: string;
  message: string;
  status?: number;
  stack?: string;
}

function getErrorDetails(error: unknown): ErrorDetails {
  // Handle React Router error responses (404, etc.)
  if (isRouteErrorResponse(error)) {
    if (error.status === 404) {
      return {
        title: 'Page Not Found',
        message: "Sorry, the page you're looking for doesn't exist or has been moved.",
        status: 404,
      };
    }
    return {
      title: `Error ${error.status}`,
      message: error.statusText || 'An unexpected error occurred.',
      status: error.status,
    };
  }

  // Handle standard Error objects
  if (error instanceof Error) {
    // Check for React error #185 specifically
    if (error.message.includes('185') || error.message.includes('Maximum update depth')) {
      return {
        title: 'Render Loop Detected',
        message:
          'The application encountered an infinite render loop. This is usually caused by a state update during rendering. Please refresh the page to try again.',
        stack: error.stack,
      };
    }
    return {
      title: 'Application Error',
      message: error.message,
      stack: error.stack,
    };
  }

  // Fallback for unknown error types
  return {
    title: 'Unexpected Error',
    message: 'Something went wrong. Please try refreshing the page.',
  };
}

export default function ErrorBoundary() {
  const error = useRouteError();
  const navigate = useNavigate();
  const errorDetails = getErrorDetails(error);
  const isDev = import.meta.env.DEV;

  const handleRefresh = () => {
    window.location.reload();
  };

  const handleGoHome = () => {
    navigate('/');
  };

  return (
    <div className="min-h-screen bg-gray-900 flex items-center justify-center p-4">
      <div className="max-w-lg w-full bg-gray-800 rounded-lg shadow-xl overflow-hidden">
        {/* Header */}
        <div className="bg-red-500/20 px-6 py-4 border-b border-red-500/30">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-red-500/20 rounded-lg">
              <AlertTriangle className="w-6 h-6 text-red-400" />
            </div>
            <div>
              <h1 className="text-lg font-semibold text-white">{errorDetails.title}</h1>
              {errorDetails.status && (
                <p className="text-sm text-red-300">Status: {errorDetails.status}</p>
              )}
            </div>
          </div>
        </div>

        {/* Body */}
        <div className="px-6 py-5">
          <p className="text-gray-300 mb-6">{errorDetails.message}</p>

          {/* Stack trace in development */}
          {isDev && errorDetails.stack && (
            <details className="mb-6">
              <summary className="text-sm text-gray-400 cursor-pointer hover:text-gray-300 flex items-center gap-2">
                <Bug className="w-4 h-4" />
                Stack Trace (Development Only)
              </summary>
              <pre className="mt-3 p-3 bg-gray-900 rounded-md text-xs text-red-300 overflow-x-auto whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
                {errorDetails.stack}
              </pre>
            </details>
          )}

          {/* Action buttons */}
          <div className="flex gap-3">
            <button
              onClick={handleRefresh}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors"
            >
              <RefreshCw className="w-4 h-4" />
              Refresh Page
            </button>
            <button
              onClick={handleGoHome}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-gray-700 hover:bg-gray-600 text-white font-medium rounded-lg transition-colors"
            >
              <Home className="w-4 h-4" />
              Go Home
            </button>
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-3 bg-gray-900/50 border-t border-gray-700">
          <p className="text-xs text-gray-500 text-center">
            If this issue persists, please contact support or check the browser console for more
            details.
          </p>
        </div>
      </div>
    </div>
  );
}
