import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "./ui/button";

interface ErrorBoundaryProps {
  children: ReactNode;
  /** Shown in the fallback UI to tell the user which section failed */
  section?: string;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Catches rendering errors in child components and displays a recovery UI
 * instead of crashing the entire app with a white screen.
 *
 * Note: This uses inline error styling rather than the ErrorAlert composite
 * because ErrorBoundary is a class component and must be self-contained
 * (it cannot use hooks or risk its own rendering throwing).
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error(
      `[ErrorBoundary${this.props.section ? `: ${this.props.section}` : ""}]`,
      error,
      info.componentStack,
    );
  }

  private handleRetry = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): ReactNode {
    if (this.state.hasError) {
      const section = this.props.section ?? "This section";
      return (
        <div
          className="rounded-lg border border-red-500/20 bg-red-500/10 p-6 text-center"
          data-testid="error-boundary-fallback"
        >
          <AlertTriangle className="h-8 w-8 text-red-400 mx-auto mb-3" />
          <p className="text-red-400 font-medium mb-1">
            {section} encountered an error
          </p>
          <p className="text-red-400/70 text-sm mb-4">
            {this.state.error?.message ?? "An unexpected error occurred."}
          </p>
          <Button variant="outline" size="sm" onClick={this.handleRetry}>
            <RefreshCw className="mr-2 h-3 w-3" /> Try Again
          </Button>
        </div>
      );
    }

    return this.props.children;
  }
}
