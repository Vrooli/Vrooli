// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/ERROR_SEMANTICS.md
import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "./ui/button";

interface ErrorBoundaryProps {
  /** Name of the region for error reporting (e.g., "terminal", "workspace"). */
  region: string;
  /** Optional custom fallback UI. If omitted, a default recovery panel is shown. */
  fallback?: ReactNode;
  children: ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

/**
 * React Error Boundary that isolates runtime crashes to a UI region.
 *
 * Place around major sections (workspace, terminal panes, drawers) so that
 * a crash in one area does not take down the entire application.
 */
export default class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error(`[ErrorBoundary:${this.props.region}]`, error, info.componentStack);
  }

  private handleReset = () => {
    this.setState({ error: null });
  };

  render(): ReactNode {
    if (this.state.error) {
      if (this.props.fallback) return this.props.fallback;

      return (
        <div
          data-testid={`error-boundary-${this.props.region}`}
          className="flex flex-col items-center justify-center gap-3 rounded-md border border-wc-error bg-wc-error-surface p-6 text-sm text-wc-error-text"
        >
          <AlertTriangle className="h-6 w-6 text-wc-error-detail" />
          <p className="font-medium">Something went wrong in {this.props.region}</p>
          <p className="max-w-md text-center text-xs text-wc-error-detail/70">
            {this.state.error.message}
          </p>
          <Button
            variant="outline"
            size="sm"
            onClick={this.handleReset}
            className="mt-2"
          >
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Try Again
          </Button>
        </div>
      );
    }

    return this.props.children;
  }
}
