// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
// Per-panel error boundary — isolates crashes to individual feature panels
// so a failure in CanvasView doesn't take down GraphView or TextCapture.
import { Component, type ReactNode } from "react";
import { ErrorFallback } from "./ErrorFallback";

interface Props {
  children: ReactNode;
  /** Human-readable panel name shown in the fallback UI */
  panelName: string;
}

interface State {
  hasError: boolean;
}

export class PanelErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error(`[PanelErrorBoundary:${this.props.panelName}]`, error, info.componentStack);
  }

  handleReset = () => {
    this.setState({ hasError: false });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex-1 flex items-center justify-center p-6">
          <ErrorFallback
            message={`${this.props.panelName} encountered an error.`}
            onRetry={this.handleReset}
            retryLabel="Retry"
          />
        </div>
      );
    }

    return this.props.children;
  }
}
