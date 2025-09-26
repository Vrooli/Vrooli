import React from 'react';
import './ErrorBoundary.css';

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

const DefaultErrorFallback: React.FC<ErrorFallbackProps> = ({ error, resetError }) => {
  return (
    <div className="error-boundary-fallback">
      <div className="error-content">
        <div className="error-header">
          <h2>SYSTEM MALFUNCTION DETECTED</h2>
          <div className="error-code">ERROR_CODE: {error.name}</div>
        </div>
        
        <div className="error-details">
          <div className="error-message">
            <span className="label">MESSAGE:</span>
            <span className="value">{error.message}</span>
          </div>
          
          {error.stack && (
            <details className="error-stack">
              <summary>STACK TRACE</summary>
              <pre>{error.stack}</pre>
            </details>
          )}
        </div>
        
        <div className="error-actions">
          <button 
            className="error-btn primary"
            onClick={resetError}
            title="Attempt to recover from error"
          >
            REBOOT SYSTEM
          </button>
          <button 
            className="error-btn secondary"
            onClick={() => window.location.reload()}
            title="Reload the entire application"
          >
            FULL RELOAD
          </button>
        </div>
        
        <div className="error-help">
          <p>If this error persists:</p>
          <ul>
            <li>Check browser console for additional details</li>
            <li>Verify all services are running properly</li>
            <li>Contact system administrator if needed</li>
          </ul>
        </div>
      </div>
      
      <div className="matrix-rain-error" aria-hidden="true"></div>
    </div>
  );
};

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
    // Log error details
    console.error('[ErrorBoundary] Caught an error:', error);
    console.error('[ErrorBoundary] Error info:', errorInfo);
    
    // Update state with error info
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
