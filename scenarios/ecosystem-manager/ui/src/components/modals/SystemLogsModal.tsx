/**
 * System Logs Modal
 * Displays system logs with filtering and auto-scroll functionality
 */

import { useState, useEffect, useRef } from 'react';
import { X, RefreshCw, ChevronsDown } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { useSystemLogs } from '../../hooks/useSystemLogs';
import type { LogEntry } from '../../types/api';

interface SystemLogsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SystemLogsModal({ open, onOpenChange }: SystemLogsModalProps) {
  const [logLevel, setLogLevel] = useState<'all' | 'info' | 'warning' | 'error'>('all');
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);

  // Fetch logs with 5-second polling when modal is open
  const { data: logs = [], isLoading, refetch } = useSystemLogs({
    limit: 500,
    level: logLevel,
    refetchInterval: open ? 5000 : undefined,
  });

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  // Detect manual scroll and disable auto-scroll
  const handleScroll = () => {
    if (!logsContainerRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
    const isAtBottom = Math.abs(scrollHeight - clientHeight - scrollTop) < 10;

    if (!isAtBottom && autoScroll) {
      setAutoScroll(false);
    } else if (isAtBottom && !autoScroll) {
      setAutoScroll(true);
    }
  };

  const formatTimestamp = (timestamp: string) => {
    try {
      return new Date(timestamp).toLocaleString('en-US', {
        month: 'short',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return timestamp;
    }
  };

  const getLevelColor = (level: LogEntry['level']) => {
    switch (level) {
      case 'error':
        return 'text-red-400 bg-red-500/10 border-red-500/30';
      case 'warning':
        return 'text-yellow-400 bg-yellow-500/10 border-yellow-500/30';
      case 'info':
        return 'text-blue-400 bg-blue-500/10 border-blue-500/30';
      default:
        return 'text-slate-400 bg-slate-500/10 border-slate-500/30';
    }
  };

  const getLevelBadgeColor = (level: LogEntry['level']) => {
    switch (level) {
      case 'error':
        return 'bg-red-500/20 text-red-300 border-red-500/50';
      case 'warning':
        return 'bg-yellow-500/20 text-yellow-300 border-yellow-500/50';
      case 'info':
        return 'bg-blue-500/20 text-blue-300 border-blue-500/50';
      default:
        return 'bg-slate-500/20 text-slate-300 border-slate-500/50';
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-5xl h-[80vh] flex flex-col">
        <DialogHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <DialogTitle>System Logs</DialogTitle>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onOpenChange(false)}
            className="h-8 w-8 p-0"
          >
            <X className="h-4 w-4" />
          </Button>
        </DialogHeader>

        {/* Toolbar */}
        <div className="flex items-center justify-between gap-4 pb-4 border-b border-white/10">
          <div className="flex items-center gap-3">
            <label className="text-sm text-slate-400">Level:</label>
            <Select value={logLevel} onValueChange={(value: any) => setLogLevel(value)}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="info">Info</SelectItem>
                <SelectItem value="warning">Warning</SelectItem>
                <SelectItem value="error">Error</SelectItem>
              </SelectContent>
            </Select>

            <div className="text-xs text-slate-500 ml-2">
              {logs.length} {logs.length === 1 ? 'entry' : 'entries'}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setAutoScroll(!autoScroll)}
              className={autoScroll ? 'bg-green-500/10 border-green-500/30' : ''}
            >
              <ChevronsDown className="h-4 w-4 mr-2" />
              Auto-scroll {autoScroll ? 'ON' : 'OFF'}
            </Button>

            <Button
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              disabled={isLoading}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </div>
        </div>

        {/* Logs Container */}
        <div
          ref={logsContainerRef}
          onScroll={handleScroll}
          className="flex-1 overflow-y-auto space-y-2 pr-2"
        >
          {isLoading && logs.length === 0 ? (
            <div className="flex items-center justify-center h-full text-slate-500">
              Loading logs...
            </div>
          ) : logs.length === 0 ? (
            <div className="flex items-center justify-center h-full text-slate-500">
              No logs found
            </div>
          ) : (
            logs.map((log, index) => (
              <div
                key={`${log.timestamp}-${index}`}
                className={`flex gap-3 p-3 rounded-lg border ${getLevelColor(log.level)}`}
              >
                <div className="flex-shrink-0 text-xs text-slate-400 font-mono w-32">
                  {formatTimestamp(log.timestamp)}
                </div>

                <div className="flex-shrink-0">
                  <span
                    className={`text-xs font-semibold px-2 py-1 rounded border uppercase ${getLevelBadgeColor(log.level)}`}
                  >
                    {log.level}
                  </span>
                </div>

                <div className="flex-1 text-sm break-words">
                  {log.message}
                  {log.context && Object.keys(log.context).length > 0 && (
                    <pre className="mt-2 text-xs text-slate-400 overflow-x-auto">
                      {JSON.stringify(log.context, null, 2)}
                    </pre>
                  )}
                </div>
              </div>
            ))
          )}
          <div ref={logsEndRef} />
        </div>
      </DialogContent>
    </Dialog>
  );
}
