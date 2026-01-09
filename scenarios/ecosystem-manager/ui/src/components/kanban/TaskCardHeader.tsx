/**
 * TaskCardHeader Component
 * Displays task ID with copy button, badges, and priority
 */

import { Copy, Check } from 'lucide-react';
import { useState } from 'react';
import { PriorityIndicator } from './PriorityIndicator';
import type { Task } from '../../types/api';

interface TaskCardHeaderProps {
  task: Task;
}

export function TaskCardHeader({ task }: TaskCardHeaderProps) {
  const [copied, setCopied] = useState(false);

  const copyTaskId = async () => {
    try {
      await navigator.clipboard.writeText(task.id);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy task ID:', err);
    }
  };

  return (
    <div className="flex items-start justify-between gap-2 mb-2">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <button
            onClick={(event) => {
              event.stopPropagation();
              copyTaskId();
            }}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
            title={`Click to copy task ID (${task.id})`}
          >
            <span className="font-mono truncate">…{task.id.slice(-4)}</span>
            {copied ? (
              <Check className="h-3 w-3 text-green-500" />
            ) : (
              <Copy className="h-3 w-3" />
            )}
          </button>
        </div>
      </div>
      <PriorityIndicator priority={task.priority} />
    </div>
  );
}
