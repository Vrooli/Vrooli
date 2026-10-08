import { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "./ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/* eslint-disable react-refresh/only-export-components -- class error boundaries are component exports by design. */

type Props = {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
};

type State = {
  hasError: boolean;
  error: Error | null;
};

// ErrorBoundary is the app-level catch for render-time exceptions.
// Wrap point lives in main.tsx — see the wrap there for the canonical
// nesting (boundary inside QueryClient + i18n providers so the
// localised fallback can call useTranslation).
//
// onError is exposed for scenarios to wire telemetry (Sentry, etc.).
// Custom fallback content can be supplied via the `fallback` prop;
// the default is DefaultFallback below, which uses the strings +
// selectors registries so test IDs and copy stay coherent with the
// rest of the UI.
export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    this.props.onError?.(error, info);
  }

  private handleRetry = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): ReactNode {
    if (!this.state.hasError) {
      return this.props.children;
    }
    if (this.props.fallback !== undefined) {
      return this.props.fallback;
    }
    return <DefaultFallback onRetry={this.handleRetry} />;
  }
}

function DefaultFallback({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={selectors.errorBoundary.root}
      role="alert"
      className="min-h-screen bg-slate-950 text-slate-50 flex flex-col items-center justify-center p-6"
    >
      <div className="w-full max-w-md rounded-2xl border border-red-500/40 bg-red-950/30 p-6 text-center backdrop-blur-sm">
        <h1 className="text-2xl font-semibold">{t(strings.errorBoundary.title)}</h1>
        <p className="mt-3 text-slate-300">{t(strings.errorBoundary.message)}</p>
        <Button
          data-testid={selectors.errorBoundary.retryButton}
          className="mt-5"
          onClick={onRetry}
        >
          {t(strings.errorBoundary.retry)}
        </Button>
      </div>
    </div>
  );
}
