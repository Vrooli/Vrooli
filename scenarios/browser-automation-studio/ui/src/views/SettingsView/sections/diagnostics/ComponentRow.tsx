/**
 * ComponentRow Component
 *
 * Expandable row for displaying component health status with optional details.
 */

import { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import type { ComponentStatus } from '@/domains/observability';
import { StatusBadge } from './StatusBadge';

interface ComponentRowProps {
  icon: React.ReactNode;
  title: string;
  status: ComponentStatus;
  summary: string;
  details?: React.ReactNode;
  hint?: string;
}

export function ComponentRow({ icon, title, status, summary, details, hint }: ComponentRowProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const hasDetails = !!details;

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => hasDetails && setIsExpanded(!isExpanded)}
        className={`w-full flex items-center gap-4 px-4 py-3 bg-gray-800/50 text-left ${hasDetails ? 'hover:bg-gray-800 cursor-pointer' : 'cursor-default'} transition-colors`}
        disabled={!hasDetails}
      >
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className="text-gray-400 flex-shrink-0">{icon}</div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium text-surface truncate">{title}</span>
              <StatusBadge status={status} />
            </div>
            <p className="text-xs text-gray-400 truncate mt-0.5">{summary}</p>
          </div>
        </div>
        {hasDetails && (
          <div className="text-gray-400 flex-shrink-0">
            {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
          </div>
        )}
      </button>
      {isExpanded && details && (
        <div className="p-4 bg-gray-900/50 border-t border-gray-700">
          {hint && (
            <div className="mb-3 p-2 bg-amber-500/10 border border-amber-500/30 rounded text-xs text-amber-300">
              <strong>Hint:</strong> {hint}
            </div>
          )}
          {details}
        </div>
      )}
    </div>
  );
}

export default ComponentRow;
