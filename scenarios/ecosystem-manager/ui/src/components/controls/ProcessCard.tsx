import { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { XCircle, Clock } from 'lucide-react';
import { useTerminateProcess } from '../../hooks/useRunningProcesses';
import type { RunningProcess } from '../../types/api';

interface ProcessCardProps {
  process: RunningProcess;
}

export function ProcessCard({ process }: ProcessCardProps) {
  const terminateProcess = useTerminateProcess();
  const [elapsedSeconds, setElapsedSeconds] = useState(0);

  useEffect(() => {
    // Calculate initial elapsed time
    const startTime = new Date(process.start_time).getTime();
    const updateElapsed = () => {
      const now = Date.now();
      const elapsed = Math.floor((now - startTime) / 1000);
      setElapsedSeconds(elapsed);
    };

    updateElapsed();
    const interval = setInterval(updateElapsed, 1000);

    return () => clearInterval(interval);
  }, [process.start_time]);

  const formatElapsed = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    } else {
      return `${secs}s`;
    }
  };

  const handleTerminate = () => {
    const taskId = process.task_id || process.process_id;
    if (confirm(`Terminate process for task ${taskId}?`)) {
      terminateProcess.mutate(taskId);
    }
  };

  return (
    <div className="flex items-center justify-between gap-3 p-2 rounded-lg bg-muted/50 hover:bg-muted transition-colors">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium truncate">
            Task {process.task_id.slice(0, 8)}
          </span>
          <div className="flex items-center gap-1 text-xs text-muted-foreground">
            <Clock className="h-3 w-3" />
            <span className="tabular-nums">{formatElapsed(elapsedSeconds)}</span>
          </div>
        </div>
        {process.agent_id && (
          <div className="text-xs text-muted-foreground truncate">
            Agent: {process.agent_id}
          </div>
        )}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={handleTerminate}
        disabled={terminateProcess.isPending}
        className="h-8 w-8 p-0"
      >
        <XCircle className="h-4 w-4 text-destructive" />
      </Button>
    </div>
  );
}
