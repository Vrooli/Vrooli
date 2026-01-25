import React from 'react';
import { AlertTriangle, RefreshCw, Home } from 'lucide-react';
import { logger } from '@/utils/logger';

interface SectionErrorBoundaryProps {
  children: React.ReactNode;
  title?: string;
  description?: string;
  onRetry?: () => void;
  onGoHome?: () => void;
}

interface SectionErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

const isDev = import.meta.env.DEV;

export default class SectionErrorBoundary extends React.Component<
  SectionErrorBoundaryProps,
  SectionErrorBoundaryState
> {
  state: SectionErrorBoundaryState = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): SectionErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error) {
    logger.error('Section error boundary caught an error', {
      component: 'SectionErrorBoundary',
      title: this.props.title,
    }, error);
  }

  handleRetry = () => {
    if (this.props.onRetry) {
      this.props.onRetry();
      return;
    }
    window.location.reload();
  };

  handleGoHome = () => {
    if (this.props.onGoHome) {
      this.props.onGoHome();
      return;
    }
    window.location.href = '/';
  };

  render() {
    if (!this.state.hasError) {
      return this.props.children;
    }

    const title = this.props.title ?? 'Section failed to load';
    const description =
      this.props.description ??
      'Something went wrong while rendering this view. You can retry or return home.';

    return (
      <div className="min-h-[60vh] flex items-center justify-center px-6 py-10">
        <div className="w-full max-w-xl rounded-2xl border border-red-500/30 bg-gray-900/70 p-6 text-left shadow-xl">
          <div className="flex items-start gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-red-500/20">
              <AlertTriangle className="h-5 w-5 text-red-400" />
            </span>
            <div className="space-y-2">
              <h2 className="text-lg font-semibold text-white">{title}</h2>
              <p className="text-sm text-gray-300">{description}</p>
            </div>
          </div>

          {isDev && this.state.error?.stack && (
            <pre className="mt-4 max-h-48 overflow-y-auto rounded-lg bg-black/40 p-3 text-xs text-red-200 whitespace-pre-wrap">
              {this.state.error.stack}
            </pre>
          )}

          <div className="mt-5 flex flex-wrap gap-3">
            <button
              onClick={this.handleRetry}
              className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-700"
            >
              <RefreshCw className="h-4 w-4" />
              Retry
            </button>
            <button
              onClick={this.handleGoHome}
              className="inline-flex items-center gap-2 rounded-lg bg-gray-700 px-4 py-2 text-sm font-medium text-white transition hover:bg-gray-600"
            >
              <Home className="h-4 w-4" />
              Go Home
            </button>
          </div>
        </div>
      </div>
    );
  }
}
