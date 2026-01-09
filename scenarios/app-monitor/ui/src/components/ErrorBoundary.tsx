import React from 'react';
import './ErrorBoundary.css';
import { logger } from '@/services/logger';

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: React.ErrorInfo | null;
}

interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ComponentType<ErrorFallbackProps>;
}

interface ErrorFallbackProps {
  error: Error;
  resetError: () => void;
}

const DefaultErrorFallback: React.FC<ErrorFallbackProps> = ({ error, resetError }) => (
  <div className="error-fallback">
    <div className="error-fallback__panel" role="alert" aria-live="assertive">
      <header className="error-fallback__header">
        <h2>Something went wrong</h2>
        <p>App Monitor hit an unexpected issue. Try recovering or reload to start fresh.</p>
      </header>

      <div className="error-fallback__meta" data-testid="error-details">
        <div>
          <span className="error-fallback__label">Error</span>
          <span className="error-fallback__value">{error.name}</span>
        </div>
        <div>
          <span className="error-fallback__label">Message</span>
          <span className="error-fallback__value">{error.message}</span>
        </div>
      </div>

      {error.stack && (
        <details className="error-fallback__stack">
          <summary>Show stack trace</summary>
          <pre>{error.stack}</pre>
        </details>
      )}

      <div className="error-fallback__actions">
        <button type="button" onClick={resetError} className="error-fallback__btn error-fallback__btn--primary">
          Return to App
        </button>
        <button
          type="button"
          onClick={() => window.location.reload()}
          className="error-fallback__btn error-fallback__btn--secondary"
        >
          Reload
        </button>
      </div>

      <p className="error-fallback__hint">
        If the issue continues, capture a screenshot and console logs and share them with the platform team.
      </p>
    </div>
  </div>
);

class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    // Update state so the next render will show the fallback UI
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    logger.error('Error boundary captured an exception', { error, errorInfo });

    this.setState({
      error,
      errorInfo,
    });

    // You could also log the error to an error reporting service here
    // Example: logErrorToService(error, errorInfo);
  }

  handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  render() {
    if (this.state.hasError && this.state.error) {
      const FallbackComponent = this.props.fallback || DefaultErrorFallback;
      
      return (
        <FallbackComponent
          error={this.state.error}
          resetError={this.handleReset}
        />
      );
    }

    return this.props.children;
  }
}

export type { ErrorBoundaryProps, ErrorFallbackProps };
export default ErrorBoundary;
