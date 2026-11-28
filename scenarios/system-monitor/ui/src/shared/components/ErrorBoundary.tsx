import { Component } from 'react';
import type { ReactNode, ErrorInfo } from 'react';
import { AlertTriangle, RefreshCw, Bug, Copy, Check } from 'lucide-react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  copied: boolean;
  retryCount: number;
}

export class ErrorBoundary extends Component<Props, State> {
  private retryTimeout: NodeJS.Timeout | null = null;
  private copyTimeout: NodeJS.Timeout | null = null;

  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      copied: false,
      retryCount: 0
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return {
      hasError: true,
      error
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    
    this.setState({
      error,
      errorInfo
    });

    // Call optional error handler
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // In development, also log to help with debugging
    if (process.env.NODE_ENV === 'development') {
      console.group('🐛 Error Boundary Details');
      console.error('Error:', error);
      console.error('Component Stack:', errorInfo.componentStack);
      console.error('Error Stack:', error.stack);
      console.groupEnd();
    }
  }

  componentWillUnmount() {
    if (this.retryTimeout) {
      clearTimeout(this.retryTimeout);
    }
    if (this.copyTimeout) {
      clearTimeout(this.copyTimeout);
    }
  }

  handleRetry = () => {
    this.setState(prevState => ({
      hasError: false,
      error: null,
      errorInfo: null,
      retryCount: prevState.retryCount + 1
    }));
  };

  handleCopyError = async () => {
    const { error, errorInfo } = this.state;
    if (!error || !errorInfo) return;

    const errorDetails = `
System Monitor - Error Report
=============================
Time: ${new Date().toISOString()}
Retry Count: ${this.state.retryCount}

Error Message: ${error.message}

Error Stack:
${error.stack || 'No stack trace available'}

Component Stack:
${errorInfo.componentStack}

Browser Info:
- User Agent: ${navigator.userAgent}
- URL: ${window.location.href}
- Timestamp: ${Date.now()}
    `.trim();

    try {
      await navigator.clipboard.writeText(errorDetails);
      this.setState({ copied: true });
      
      // Reset copied state after 2 seconds
      this.copyTimeout = setTimeout(() => {
        this.setState({ copied: false });
      }, 2000);
    } catch (err) {
      console.error('Failed to copy to clipboard:', err);
    }
  };

  render() {
    if (this.state.hasError) {
      // If custom fallback provided, use it
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const { error, errorInfo, retryCount } = this.state;
      const isDev = process.env.NODE_ENV === 'development';

      return (
        <div data-sm-style="sm-style-d7272546ca">
          <div data-sm-style="sm-style-9a5e094449">
            {/* Header */}
            <div data-sm-style="sm-style-58ee2038d8">
              <div data-sm-style="sm-style-f1fe77f7dc">
                <AlertTriangle size={32} data-sm-style="sm-style-6d06f948c5" />
              </div>
              
              <div>
                <h1 data-sm-style="sm-style-596c89e6f2">
                  System Error
                </h1>
                <p data-sm-style="sm-style-a3d7ba4ae7">
                  Component crashed • Retry #{retryCount + 1}
                </p>
              </div>
            </div>

            {/* Error Message */}
            <div data-sm-style="sm-style-5e6e478f5f">
              <h3 data-sm-style="sm-style-7acc1b7c7b">
                <Bug size={16} />
                Error Details
              </h3>
              <p data-sm-style="sm-style-6b0bd53dc4">
                {error?.message || 'Unknown error occurred'}
              </p>
            </div>

            {/* Development Info */}
            {isDev && error && errorInfo && (
              <div data-sm-style="sm-style-a9af1f41eb">
                <h4 data-sm-style="sm-style-15a86bafac">
                  Development Details
                </h4>
                
                <div data-sm-style="sm-style-91394348ef">
                  <strong data-sm-style="sm-style-dbed1e5364">Component Stack:</strong>
                  <pre data-sm-style="sm-style-bb45f3b812">
                    {errorInfo.componentStack}
                  </pre>
                </div>

                {error.stack && (
                  <div>
                    <strong data-sm-style="sm-style-dbed1e5364">Error Stack:</strong>
                    <pre data-sm-style="sm-style-695cfbe3f2">
                      {error.stack}
                    </pre>
                  </div>
                )}
              </div>
            )}

            {/* Actions */}
            <div data-sm-style="sm-style-63be1464e7">
              <button className="btn-retry" onClick={this.handleRetry}>
                <RefreshCw size={16} />
                Retry
              </button>

              <button className="btn-copy-error" onClick={() => { void this.handleCopyError(); }}>
                {this.state.copied ? <Check size={16} /> : <Copy size={16} />}
                {this.state.copied ? 'Copied!' : 'Copy Error'}
              </button>

              <button className="btn-reload" onClick={() => { window.location.reload(); }}>
                <RefreshCw size={16} />
                Reload Page
              </button>
            </div>

            {/* Footer */}
            <div data-sm-style="sm-style-b1ac371e7d">
              <p data-sm-style="sm-style-2a0ca8350a">
                This error has been logged. If the problem persists, check the browser console for additional details.
              </p>
              {isDev && (
                <p data-sm-style="sm-style-23a6170aa2">
                  💡 <strong>Development Mode:</strong> Additional debugging information is displayed above.
                </p>
              )}
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
