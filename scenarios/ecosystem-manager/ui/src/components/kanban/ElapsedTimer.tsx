/**
 * ElapsedTimer Component
 * Shows live elapsed time for in-progress tasks
 */

import { useEffect, useState } from 'react';
import { Clock } from 'lucide-react';

interface ElapsedTimerProps {
  startTime: string;
  className?: string;
}

function formatElapsedTime(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${secs}s`;
  }
  return `${secs}s`;
}

export function ElapsedTimer({ startTime, className = '' }: ElapsedTimerProps) {
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    const calculateElapsed = () => {
      const start = new Date(startTime).getTime();
      const now = Date.now();
      const diff = Math.floor((now - start) / 1000);
      setElapsed(diff);
    };

    // Calculate immediately
    calculateElapsed();

    // Update every second
    const interval = setInterval(calculateElapsed, 1000);

    return () => clearInterval(interval);
  }, [startTime]);

  return (
    <div className={`flex items-center gap-1.5 text-xs text-slate-400 ${className}`}>
      <Clock className="h-3.5 w-3.5" />
      <span>{formatElapsedTime(elapsed)}</span>
    </div>
  );
}
