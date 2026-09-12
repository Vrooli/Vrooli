import { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "../primitives/Button";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";

type Props = {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
};

type State = {
  hasError: boolean;
  error: Error | null;
};

/**
 * Surface-level error boundary. Place around every route so a render-time
 * crash in one surface doesn't take down the whole app.
 *
 * The top-level boundary in `main.tsx` is still the global safety net;
 * this one provides surface-local recovery (the "retry" button just
 * resets local state).
 */
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
    return <ErrorBoundaryFallback onRetry={this.handleRetry} />;
  }
}

export function ErrorBoundaryFallback({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={selectors.errorBoundary.root}
      role="alert"
      className="flex flex-col items-center justify-center gap-3 rounded-panel border border-status-failure/30 bg-status-failure-bg/30 p-6 text-center"
    >
      <h2 className="text-base font-semibold text-app-foreground">{t(strings.errorBoundary.title)}</h2>
      <p className="text-sm text-app-muted-foreground">{t(strings.errorBoundary.message)}</p>
      <Button
        data-testid={selectors.errorBoundary.retryButton}
        size="sm"
        variant="outline"
        onClick={onRetry}
      >
        {t(strings.errorBoundary.retry)}
      </Button>
    </div>
  );
}
