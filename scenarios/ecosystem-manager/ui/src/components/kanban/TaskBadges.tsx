/**
 * TaskBadges Component
 * Displays type and operation badges for a task
 */

import { memo } from 'react';
import type { TaskType, OperationType } from '../../types/api';

interface TaskBadgesProps {
  type: TaskType;
  operation: OperationType;
}

export const TaskBadges = memo(function TaskBadges({ type, operation }: TaskBadgesProps) {
  const typeColors = {
    resource: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
    scenario: 'bg-purple-500/20 text-purple-300 border-purple-500/30',
  };

  const operationColors = {
    generator: 'bg-green-500/20 text-green-300 border-green-500/30',
    improver: 'bg-orange-500/20 text-orange-300 border-orange-500/30',
  };

  return (
    <div className="flex items-center gap-1.5">
      <span
        className={`px-2 py-0.5 text-xs font-medium rounded border ${typeColors[type]}`}
      >
        {type}
      </span>
      <span
        className={`px-2 py-0.5 text-xs font-medium rounded border ${operationColors[operation]}`}
      >
        {operation}
      </span>
    </div>
  );
});
