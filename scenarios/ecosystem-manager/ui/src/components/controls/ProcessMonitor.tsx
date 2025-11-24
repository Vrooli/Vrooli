import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useRunningProcesses } from '../../hooks/useRunningProcesses';
import { Button } from '../ui/button';
import { ProcessCard } from './ProcessCard';
import { Activity, ChevronDown } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { Task } from '@/types/api';

interface ProcessMonitorProps {
  onSelectTask?: (task: Task) => void;
}

export function ProcessMonitor({ onSelectTask }: ProcessMonitorProps) {
  const { data: processes = [], isLoading } = useRunningProcesses();
  const queryClient = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const [loadingTaskId, setLoadingTaskId] = useState<string | null>(null);

  const processCount = processes.length;

  const findTaskInCache = (taskId: string): Task | undefined => {
    const listQueries = queryClient.getQueriesData<Task[]>({ queryKey: queryKeys.tasks.lists() });
    for (const [, data] of listQueries) {
      if (Array.isArray(data)) {
        const match = data.find(task => task.id === taskId);
        if (match) return match;
      }
    }

    return queryClient.getQueryData<Task>(queryKeys.tasks.detail(taskId));
  };

  const handleOpenTask = async (taskId: string) => {
    if (!onSelectTask) return;
    setLoadingTaskId(taskId);
    try {
      const cachedTask = findTaskInCache(taskId);
      const task = cachedTask ?? (await api.getTask(taskId));
      onSelectTask(task);
      setIsOpen(false);
    } catch (error) {
      console.error('Failed to open task from process monitor', error);
      alert('Unable to load task details right now. Please try again.');
    } finally {
      setLoadingTaskId(null);
    }
  };

  return (
    <div className="relative">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsOpen(!isOpen)}
            className="relative px-3"
            disabled={isLoading}
            aria-label={`View running processes${processCount > 0 ? ` (${processCount})` : ''}`}
          >
            <Activity className="h-4 w-4" />
            {processCount > 0 && (
              <span className="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center font-medium">
                {processCount}
              </span>
            )}
            <ChevronDown className={`h-3 w-3 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="top">Running processes</TooltipContent>
      </Tooltip>

      {isOpen && (
        <>
          {/* Backdrop to close dropdown */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          {/* Dropdown panel */}
          <div className="absolute right-0 bottom-full mb-2 w-80 bg-background border rounded-lg shadow-lg z-50 max-h-96 overflow-y-auto">
            <div className="p-3 border-b">
              <h3 className="font-semibold text-sm">Running Processes</h3>
              <p className="text-xs text-muted-foreground mt-0.5">
                {processCount} active {processCount === 1 ? 'process' : 'processes'}
              </p>
            </div>

            <div className="p-2 space-y-2">
              {processCount === 0 ? (
                <div className="text-center py-8 text-sm text-muted-foreground">
                  No processes running
                </div>
              ) : (
                processes.map((process) => (
                  <ProcessCard
                    key={process.process_id}
                    process={process}
                    onSelect={loadingTaskId ? undefined : () => handleOpenTask(process.task_id)}
                  />
                ))
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
