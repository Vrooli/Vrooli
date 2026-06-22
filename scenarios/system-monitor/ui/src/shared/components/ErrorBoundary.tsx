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
        <div style={{
          minHeight: '100vh',
          background: 'linear-gradient(135deg, var(--color-background) 0%, var(--color-background-alt) 100%)',
          color: 'var(--color-text)',
          fontFamily: 'var(--font-mono)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: 'var(--spacing-xl)'
        }}>
          <div style={{
            background: 'var(--overlay-backdrop)',
            border: '2px solid var(--color-error)',
            borderRadius: 'var(--radius-lg)',
            padding: 'var(--spacing-xxl)',
            maxWidth: '800px',
            width: '100%',
            boxShadow: '0 0 30px var(--color-error)'
          }}>
            {/* Header */}
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-xl)',
              borderBottom: '1px solid var(--color-error)',
              paddingBottom: 'var(--spacing-lg)'
            }}>
              <div style={{
                padding: 'var(--spacing-sm)',
                background: 'var(--color-error-muted)',
                borderRadius: 'var(--radius-md)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center'
              }}>
                <AlertTriangle size={32} style={{ color: 'var(--color-error)' }} />
              </div>
              
              <div>
                <h1 style={{
                  margin: 0,
                  color: 'var(--color-error)',
                  fontSize: 'var(--text-2xl)',
                  fontWeight: 'bold',
                  textTransform: 'uppercase',
                  letterSpacing: '2px'
                }}>
                  System Error
                </h1>
                <p style={{
                  margin: 'var(--spacing-xs) 0 0 0',
                  color: 'var(--color-text-secondary)',
                  fontSize: 'var(--text-sm)'
                }}>
                  Component crashed • Retry #{retryCount + 1}
                </p>
              </div>
            </div>

            {/* Error Message */}
            <div style={{
              background: 'var(--color-error-muted)',
              border: '1px solid var(--color-error)',
              borderRadius: 'var(--radius-md)',
              padding: 'var(--spacing-lg)',
              marginBottom: 'var(--spacing-xl)'
            }}>
              <h3 style={{
                margin: '0 0 var(--spacing-sm) 0',
                color: 'var(--color-text-heading)',
                fontSize: 'var(--text-base)',
                display: 'flex',
                alignItems: 'center',
                gap: 'var(--spacing-sm)'
              }}>
                <Bug size={16} />
                Error Details
              </h3>
              <p style={{
                margin: 0,
                color: 'var(--color-text)',
                fontSize: 'var(--text-sm)',
                fontFamily: 'var(--font-mono)',
                background: 'var(--overlay-medium)',
                padding: 'var(--spacing-md)',
                borderRadius: 'var(--radius-sm)',
                overflowX: 'auto',
                whiteSpace: 'pre-wrap'
              }}>
                {error?.message || 'Unknown error occurred'}
              </p>
            </div>

            {/* Development Info */}
            {isDev && error && errorInfo && (
              <div style={{
                background: 'var(--color-primary-muted)',
                border: '1px solid var(--color-primary)',
                borderRadius: 'var(--radius-md)',
                padding: 'var(--spacing-lg)',
                marginBottom: 'var(--spacing-xl)'
              }}>
                <h4 style={{
                  margin: '0 0 var(--spacing-sm) 0',
                  color: 'var(--color-primary)',
                  fontSize: 'var(--text-sm)',
                  textTransform: 'uppercase',
                  letterSpacing: '1px'
                }}>
                  Development Details
                </h4>
                
                <div style={{ marginBottom: 'var(--spacing-md)' }}>
                  <strong style={{ color: 'var(--color-text-heading)' }}>Component Stack:</strong>
                  <pre style={{
                    margin: 'var(--spacing-xs) 0 0 0',
                    color: 'var(--color-text-secondary)',
                    fontSize: 'var(--text-xs)',
                    background: 'var(--overlay-medium)',
                    padding: 'var(--spacing-sm)',
                    borderRadius: 'var(--radius-sm)',
                    overflowX: 'auto',
                    maxHeight: '200px',
                    overflowY: 'auto'
                  }}>
                    {errorInfo.componentStack}
                  </pre>
                </div>

                {error.stack && (
                  <div>
                    <strong style={{ color: 'var(--color-text-heading)' }}>Error Stack:</strong>
                    <pre style={{
                      margin: 'var(--spacing-xs) 0 0 0',
                      color: 'var(--color-text-secondary)',
                      fontSize: 'var(--text-xs)',
                      background: 'var(--overlay-medium)',
                      padding: 'var(--spacing-sm)',
                      borderRadius: 'var(--radius-sm)',
                      overflowX: 'auto',
                      maxHeight: '200px',
                      overflowY: 'auto'
                    }}>
                      {error.stack}
                    </pre>
                  </div>
                )}
              </div>
            )}

            {/* Actions */}
            <div style={{
              display: 'flex',
              gap: 'var(--spacing-md)',
              justifyContent: 'center',
              flexWrap: 'wrap'
            }}>
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
            <div style={{
              marginTop: 'var(--spacing-xl)',
              paddingTop: 'var(--spacing-lg)',
              borderTop: '1px solid var(--color-text-secondary)',
              textAlign: 'center',
              color: 'var(--color-text-secondary)',
              fontSize: 'var(--text-xs)'
            }}>
              <p style={{ margin: 0 }}>
                This error has been logged. If the problem persists, check the browser console for additional details.
              </p>
              {isDev && (
                <p style={{ margin: 'var(--spacing-xs) 0 0 0' }}>
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
