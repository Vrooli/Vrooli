// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import { Component, type ErrorInfo, type ReactNode } from "react";

export type ErrorBoundaryFallbackProps = {
  error: Error;
  reset: () => void;
};

export type ErrorBoundaryProps = {
  children: ReactNode;
  fallback: (params: ErrorBoundaryFallbackProps) => ReactNode;
};

type ErrorBoundaryState = {
  error: Error | null;
};

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("[knowledge-observatory] UI section crashed", error, info);
  }

  handleReset = () => {
    this.setState({ error: null });
  };

  render() {
    const { error } = this.state;
    if (error) {
      return this.props.fallback({ error, reset: this.handleReset });
    }
    return this.props.children;
  }
}
