import type { CompletenessScore } from '@/types';
import { Activity, AlertCircle, Loader } from 'lucide-react';
import './CompletenessTab.css';

interface CompletenessTabProps {
  completeness: CompletenessScore | null | undefined;
  loading?: boolean;
  error?: string | null;
  onRetry?: () => void;
}

export default function CompletenessTab({ completeness, loading, error, onRetry }: CompletenessTabProps) {
  if (loading) {
    return (
      <div className="completeness-tab">
        <div className="completeness-tab__loading">
          <Loader size={32} className="completeness-tab__loading-icon spinning" />
          <p>Calculating completeness score...</p>
        </div>
      </div>
    );
  }

  if (error && !completeness) {
    return (
      <div className="completeness-tab">
        <div className="completeness-tab__error">
          <AlertCircle size={32} />
          <p>{error}</p>
          {onRetry && (
            <button type="button" className="completeness-tab__retry" onClick={onRetry}>
              Retry
            </button>
          )}
        </div>
      </div>
    );
  }

  if (!completeness) {
    return (
      <div className="completeness-tab">
        <div className="completeness-tab__empty">
          <Activity size={32} />
          <p>No completeness data available</p>
        </div>
      </div>
    );
  }

  const details = Array.isArray(completeness.details) ? completeness.details : [];

  if (details.length === 0) {
    return (
      <div className="completeness-tab">
        <div className="completeness-tab__empty">
          <Activity size={32} />
          <p>No completeness details available</p>
        </div>
      </div>
    );
  }

  return (
    <div className="completeness-tab">
      <pre className="completeness-tab__output">{details.join('\n')}</pre>
    </div>
  );
}
