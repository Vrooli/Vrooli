import { Component, type ErrorInfo, type ReactNode } from "react";

import { Button } from "./ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { statusTone } from "../theme/status";

type Props = {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
};

type State = {
  hasError: boolean;
};

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    this.props.onError?.(error, info);
  }

  private handleRetry = (): void => {
    this.setState({ hasError: false });
  };

  render(): ReactNode {
    if (!this.state.hasError) {
      return this.props.children;
    }
    if (this.props.fallback !== undefined) {
      return this.props.fallback;
    }
    return (
      <div
        className="flex min-h-full items-center justify-center bg-background p-6 text-foreground"
        data-testid={selectors.errorBoundary.root}
        role="alert"
      >
        <div className={`max-w-md rounded-lg border bg-card p-6 text-center ${statusTone("danger").panel}`}>
          <h1 className="text-2xl font-semibold">{strings.errorBoundary.title}</h1>
          <p className="mt-3 text-sm text-muted-foreground">{strings.errorBoundary.message}</p>
          <Button
            className="mt-5"
            data-testid={selectors.errorBoundary.retryButton}
            onClick={this.handleRetry}
            type="button"
          >
            {strings.errorBoundary.retry}
          </Button>
        </div>
      </div>
    );
  }
}
