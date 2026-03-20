// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
// Global error boundary - catches render errors to prevent full app crashes.
// Shows a recovery UI instead of a white screen.
import { Component, type ReactNode } from "react";
import { ErrorFallback } from "./ErrorFallback";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("[ErrorBoundary] Uncaught render error:", error, info.componentStack);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div
          data-testid="error-boundary-fallback"
          className="h-full flex items-center justify-center bg-slate-950 text-slate-50"
        >
          <div className="max-w-md px-6">
            <ErrorFallback
              message="Something went wrong"
              detail="The app encountered an unexpected error. Your data is safe — try refreshing."
              onRetry={this.handleReset}
              retryLabel="Try again"
              iconSize="h-12 w-12"
              secondaryAction={{ label: "Reload page", onClick: () => window.location.reload() }}
            />
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
