import type { ReactNode } from 'react';
import { Loader } from 'lucide-react';

interface TabStateViewProps {
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  loadingMessage?: string;
  errorMessage?: string;
  emptyIcon?: ReactNode;
  emptyMessage?: string;
  onRetry?: () => void;
  children: ReactNode;
  className?: string;
}

/**
 * Shared loading / error / empty wrapper for tab contents.
 * Renders children when none of the guard states are active.
 */
export default function TabStateView({
  loading,
  error,
  empty,
  loadingMessage = 'Loading...',
  errorMessage,
  emptyIcon,
  emptyMessage = 'No data available',
  onRetry,
  children,
  className,
}: TabStateViewProps) {
  if (loading) {
    return (
      <div className={className}>
        <div className="tab-state-view tab-state-view--loading">
          <Loader size={32} className="spinning" aria-hidden />
          <p>{loadingMessage}</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={className}>
        <div className="tab-state-view tab-state-view--error">
          <p>{errorMessage ?? error}</p>
          {onRetry && (
            <button type="button" className="tab-state-view__retry" onClick={onRetry}>
              Retry
            </button>
          )}
        </div>
      </div>
    );
  }

  if (empty) {
    return (
      <div className={className}>
        <div className="tab-state-view tab-state-view--empty">
          {emptyIcon}
          <p>{emptyMessage}</p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
