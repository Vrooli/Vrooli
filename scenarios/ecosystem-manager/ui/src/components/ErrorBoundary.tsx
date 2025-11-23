/**
 * ErrorBoundary Component
 * Catches React errors and displays a fallback UI
 */

import React, { Component, ReactNode } from 'react';
import { AlertTriangle, Copy, CheckCircle } from 'lucide-react';
import { Button } from './ui/button';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: React.ErrorInfo | null;
  copied: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      copied: false,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    this.setState({
      error,
      errorInfo,
    });
  }

  handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      copied: false,
    });
  };

  handleCopyError = () => {
    const errorText = [
      'Error:',
      this.state.error?.toString() || 'Unknown error',
      '',
      'Stack Trace:',
      this.state.error?.stack || 'No stack trace available',
      '',
      'Component Stack:',
      this.state.errorInfo?.componentStack || 'No component stack available',
    ].join('\n');

    navigator.clipboard.writeText(errorText).then(() => {
      this.setState({ copied: true });
      setTimeout(() => this.setState({ copied: false }), 2000);
    });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="flex items-center justify-center min-h-[400px] p-6">
          <div className="max-w-3xl w-full bg-slate-800 border border-red-500/20 rounded-lg p-6 space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3 text-red-400">
                <AlertTriangle className="h-6 w-6" />
                <h2 className="text-lg font-semibold">Something went wrong</h2>
              </div>
              <Button
                onClick={this.handleCopyError}
                variant="outline"
                size="sm"
                className="gap-2"
              >
                {this.state.copied ? (
                  <>
                    <CheckCircle className="h-4 w-4" />
                    Copied!
                  </>
                ) : (
                  <>
                    <Copy className="h-4 w-4" />
                    Copy Error
                  </>
                )}
              </Button>
            </div>

            <p className="text-sm text-slate-300">
              An error occurred while rendering this component. You can copy the error details and report it.
            </p>

            {this.state.error && (
              <div className="bg-slate-900/50 rounded p-4 space-y-3">
                <div>
                  <p className="text-xs font-semibold text-slate-400 mb-2">Error Message:</p>
                  <p className="text-sm font-mono text-red-300 select-text whitespace-pre-wrap break-words">
                    {this.state.error.toString()}
                  </p>
                </div>

                {this.state.error.stack && (
                  <details className="text-xs text-slate-400">
                    <summary className="cursor-pointer hover:text-slate-300 font-semibold mb-2">
                      Error Stack Trace
                    </summary>
                    <pre className="mt-2 overflow-auto text-xs bg-black/30 p-3 rounded select-text whitespace-pre-wrap break-words max-h-64">
                      {this.state.error.stack}
                    </pre>
                  </details>
                )}

                {this.state.errorInfo && (
                  <details className="text-xs text-slate-400">
                    <summary className="cursor-pointer hover:text-slate-300 font-semibold mb-2">
                      Component Stack Trace
                    </summary>
                    <pre className="mt-2 overflow-auto text-xs bg-black/30 p-3 rounded select-text whitespace-pre-wrap break-words max-h-64">
                      {this.state.errorInfo.componentStack}
                    </pre>
                  </details>
                )}
              </div>
            )}

            <div className="flex gap-2">
              <Button onClick={this.handleReset} variant="outline" size="sm">
                Try Again
              </Button>
              <Button
                onClick={() => window.location.reload()}
                variant="outline"
                size="sm"
              >
                Reload Page
              </Button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
