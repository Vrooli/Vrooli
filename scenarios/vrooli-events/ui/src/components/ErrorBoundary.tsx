import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "./ui/button";

interface Props {
  children: ReactNode;
  fallbackMessage?: string;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

/**
 * Catches unhandled render errors and displays a recovery UI
 * instead of crashing the entire app.
 */
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("[ErrorBoundary] Uncaught render error:", error, info.componentStack);
  }

  private handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center gap-4 rounded-xl border border-[var(--error-border)] bg-[var(--error-bg)] p-8 text-center">
          <AlertTriangle className="h-8 w-8 text-[var(--error-icon)]" />
          <div>
            <h3 className="text-sm font-semibold text-[var(--error-text)]">
              {this.props.fallbackMessage ?? "Something went wrong"}
            </h3>
            <p className="mt-1 text-xs text-[var(--text-muted)]">
              An unexpected error occurred while rendering this section.
            </p>
          </div>
          <Button size="sm" variant="outline" onClick={this.handleReset}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Try Again
          </Button>
        </div>
      );
    }
    return this.props.children;
  }
}
