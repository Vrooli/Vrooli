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
    resource: 'bg-blue-50 text-blue-800 border-blue-200 dark:bg-blue-500/20 dark:text-blue-50 dark:border-blue-500/30',
    scenario: 'bg-purple-50 text-purple-800 border-purple-200 dark:bg-purple-500/20 dark:text-purple-50 dark:border-purple-500/30',
  };

  const operationColors = {
    generator: 'bg-green-50 text-green-800 border-green-200 dark:bg-green-500/20 dark:text-green-50 dark:border-green-500/30',
    improver: 'bg-orange-50 text-orange-800 border-orange-200 dark:bg-orange-500/20 dark:text-orange-50 dark:border-orange-500/30',
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
