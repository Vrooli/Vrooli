// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/ERROR_SEMANTICS.md
import { Component, type ErrorInfo as ReactErrorInfo, type ReactNode } from "react";
import ErrorBoundaryFallback from "./ErrorBoundaryFallback";

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

  componentDidCatch(error: Error, info: ReactErrorInfo): void {
    console.error(`[ErrorBoundary:${this.props.region}]`, error, info.componentStack);
  }

  private handleReset = () => {
    this.setState({ error: null });
  };

  render(): ReactNode {
    if (this.state.error) {
      if (this.props.fallback) return this.props.fallback;

      return (
        <ErrorBoundaryFallback
          region={this.props.region}
          message={this.state.error.message}
          onReset={this.handleReset}
        />
      );
    }

    return this.props.children;
  }
}
