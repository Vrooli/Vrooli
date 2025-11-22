import type { CompletenessScore } from '@/types';
import { Activity, Loader } from 'lucide-react';
import './CompletenessTab.css';

interface CompletenessTabProps {
  completeness: CompletenessScore | null | undefined;
  loading?: boolean;
}

export default function CompletenessTab({ completeness, loading }: CompletenessTabProps) {
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

  if (completeness.details.length === 0) {
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
      <pre className="completeness-tab__output">{completeness.details.join('\n')}</pre>
    </div>
  );
}
