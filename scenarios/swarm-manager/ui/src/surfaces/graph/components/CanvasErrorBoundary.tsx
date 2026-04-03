/**
 * CanvasErrorBoundary - Error boundary wrapping only the graph canvas.
 *
 * Keeps HUD and sidebar alive when the canvas encounters a render error.
 * Extracted from GraphWorkspace.tsx to reduce component size.
 */

import { Component, type ReactNode } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { ErrorDiagnostics } from "../../../components/ui/error-diagnostics";
import { categorizeError, generateUniqueId } from "../../../lib/error-utils";

/** Canvas error boundary prefix for correlation IDs */
const CANVAS_ERROR_ID_PREFIX = "canvas_err";

interface CanvasErrorBoundaryState {
  hasError: boolean;
  errorId: string | null;
  error: Error | null;
  componentStack: string | null;
  timestamp: string | null;
}

/** Error boundary that wraps only the graph canvas, keeping HUD + sidebar alive. */
export class CanvasErrorBoundary extends Component<
  { children: ReactNode },
  CanvasErrorBoundaryState
> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false, errorId: null, error: null, componentStack: null, timestamp: null };
  }

  static getDerivedStateFromError(error: Error): CanvasErrorBoundaryState {
    return {
      hasError: true,
      errorId: generateUniqueId(CANVAS_ERROR_ID_PREFIX),
      error,
      componentStack: null,
      timestamp: new Date().toISOString(),
    };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("[GraphCanvas] Render crash:", error.message, info.componentStack);
    this.setState({ componentStack: info.componentStack ?? null });
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-full w-full flex-col items-center justify-center gap-4 overflow-y-auto px-6 py-4 text-slate-400">
          <AlertTriangle className="h-10 w-10 text-amber-400" />
          <p className="text-sm">Graph canvas encountered an error.</p>
          <button
            type="button"
            className="flex items-center gap-1.5 rounded-lg border border-slate-600 bg-slate-800 px-3 py-1.5 text-xs text-slate-200 hover:bg-slate-700"
            onClick={() => this.setState({ hasError: false, errorId: null, error: null, componentStack: null, timestamp: null })}
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Retry
          </button>
          {this.state.error && this.state.timestamp && (
            <ErrorDiagnostics
              error={this.state.error}
              componentStack={this.state.componentStack}
              errorId={this.state.errorId}
              category={categorizeError(this.state.error)}
              timestamp={this.state.timestamp}
              compact
              className="max-w-full"
            />
          )}
        </div>
      );
    }
    return this.props.children;
  }
}
