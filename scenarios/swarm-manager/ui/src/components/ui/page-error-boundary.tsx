/**
 * Page-Level Error Boundary Component
 *
 * A specialized error boundary for individual pages that allows recovery
 * through navigation rather than requiring a full page refresh.
 *
 * ╔════════════════════════════════════════════════════════════════╗
 * ║  WHY PAGE-LEVEL BOUNDARIES?                                    ║
 * ║                                                                ║
 * ║  A single top-level ErrorBoundary catches all errors but       ║
 * ║  provides only one recovery option: refresh the entire app.    ║
 * ║                                                                ║
 * ║  Page-level boundaries:                                        ║
 * ║  - Isolate failures to a single route/feature                 ║
 * ║  - Keep navigation working when one page crashes              ║
 * ║  - Preserve app state (React Query cache, etc.)               ║
 * ║  - Allow users to continue using other features               ║
 * ╚════════════════════════════════════════════════════════════════╝
 */

import { Component, type ReactNode } from "react";
import { AlertTriangle, Home, RefreshCw } from "lucide-react";
import { Button } from "./button";
import { generateUniqueId } from "../../lib/error-utils";

interface PageErrorBoundaryProps {
  /** Child components to render */
  children: ReactNode;
  /** Page name for error context */
  pageName?: string;
  /** Optional custom fallback UI */
  fallback?: ReactNode;
}

interface PageErrorBoundaryState {
  hasError: boolean;
  errorId: string | null;
}

/** Error ID prefix for page-level error boundary */
const PAGE_ERROR_ID_PREFIX = "page_err";

/**
 * PageErrorBoundary catches errors in a specific page and allows recovery
 * through navigation.
 *
 * Usage:
 * ```tsx
 * <PageErrorBoundary pageName="Backlog">
 *   <BacklogPage />
 * </PageErrorBoundary>
 * ```
 */
export class PageErrorBoundary extends Component<PageErrorBoundaryProps, PageErrorBoundaryState> {
  constructor(props: PageErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, errorId: null };
  }

  static getDerivedStateFromError(): PageErrorBoundaryState {
    return { hasError: true, errorId: generateUniqueId(PAGE_ERROR_ID_PREFIX) };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error(
      "[PageErrorBoundary] Error caught:",
      JSON.stringify({
        errorId: this.state.errorId,
        page: this.props.pageName,
        name: error.name,
        message: error.message,
        componentStack: errorInfo.componentStack,
        timestamp: new Date().toISOString(),
      })
    );
  }

  handleRetry = () => {
    this.setState({ hasError: false, errorId: null });
  };

  handleGoHome = () => {
    // Navigate to home and clear error state
    this.setState({ hasError: false, errorId: null });
    window.location.href = "/graph";
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const pageName = this.props.pageName ?? "This page";

      return (
        <div className="flex min-h-[50vh] items-center justify-center p-4">
          <div className="max-w-md text-center">
            <AlertTriangle className="mx-auto h-12 w-12 text-amber-400" />
            <h2 className="mt-4 text-xl font-semibold text-slate-100">
              {pageName} encountered an error
            </h2>
            <p className="mt-2 text-slate-400">
              Something went wrong while loading this page. You can try again or
              navigate to a different section.
            </p>
            <div className="mt-6 flex justify-center gap-3">
              <Button variant="outline" onClick={this.handleGoHome}>
                <Home className="mr-2 h-4 w-4" />
                Go Home
              </Button>
              <Button onClick={this.handleRetry}>
                <RefreshCw className="mr-2 h-4 w-4" />
                Try Again
              </Button>
            </div>
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
