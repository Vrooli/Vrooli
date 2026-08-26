/**
 * @libraryId react-component-library:ErrorBoundary
 * @displayName ErrorBoundary
 * @description A recoverable render boundary that preserves product context while exposing honest recovery and safe diagnostics.
 * @version 1.0.4
 * @tags ["feedback","recovery","resilience","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource feedback.error-boundary */
import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { Component, forwardRef, type ErrorInfo, type ReactNode, type Ref } from "react";
import { ErrorState } from "@vrooli/react-component-library/ErrorState/1.0.0";

export interface ErrorBoundaryProps {
  children: ReactNode;
  className?: string;
  fallback?: ReactNode;
  contextLabel?: ReactNode;
  title?: ReactNode;
  message?: ReactNode;
  resetKeys?: readonly unknown[];
  showDiagnostics?: boolean;
  onError?: (error: Error, info: ErrorInfo) => void;
  onRetry?: (error: Error | null) => void | Promise<void>;
}

interface ErrorBoundaryImplProps extends ErrorBoundaryProps {
  forwardedRef: Ref<HTMLDivElement>;
}

interface ErrorBoundaryState {
  error: Error | null;
}

function keysChanged(previous: readonly unknown[] = [], next: readonly unknown[] = []) {
  return (
    previous.length !== next.length ||
    next.some((value, index) => !Object.is(value, previous[index]))
  );
}

class ErrorBoundaryImpl extends Component<ErrorBoundaryImplProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    this.props.onError?.(error, info);
  }

  componentDidUpdate(previousProps: ErrorBoundaryProps) {
    if (this.state.error && keysChanged(previousProps.resetKeys, this.props.resetKeys)) {
      this.setState({ error: null });
    }
  }

  private retry = async () => {
    const error = this.state.error;
    await this.props.onRetry?.(error);
    this.setState({ error: null });
  };

  render() {
    const { error } = this.state;
    if (!error) return this.props.children;
    if (this.props.fallback !== undefined) return this.props.fallback;
    const errorMessage = (
      <>
        {this.props.contextLabel ? (
          <span
            data-testid="feedback.error-boundary"
            data-rcl-error-boundary-context
            style={{ display: "block", marginBlockEnd: "var(--space-2xs)" }}
          >
            {this.props.contextLabel}
          </span>
        ) : null}
        <span style={{ display: "block" }}>
          {this.props.message ??
            "The surrounding interface is still available. Try again to restore this area."}
        </span>
        {this.props.showDiagnostics ? (
          <details data-rcl-error-boundary-diagnostics>
            <summary>
              {resolveStrings(
                "feedback.error-boundary.show-technical-details",
                "Show technical details",
              )}
            </summary>
            <code>{error.message || "Unknown render error"}</code>
          </details>
        ) : null}
      </>
    );
    return (
      <div
        className={this.props.className}
        ref={this.props.forwardedRef}
        data-rcl-error-boundary
        data-state="request-error"
        data-context={
          typeof this.props.contextLabel === "string" ? this.props.contextLabel : undefined
        }
      >
        <ErrorState
          title={this.props.title ?? "This area needs a reset"}
          message={errorMessage}
          onRetry={this.retry}
        />
      </div>
    );
  }
}

export const ErrorBoundary = forwardRef<HTMLDivElement, ErrorBoundaryProps>(
  function ErrorBoundary(props, ref) {
    return <ErrorBoundaryImpl {...props} forwardedRef={ref} />;
  },
);
