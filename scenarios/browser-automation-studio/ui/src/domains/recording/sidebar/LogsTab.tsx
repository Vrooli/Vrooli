/**
 * LogsTab Component
 *
 * Displays execution logs in the sidebar during execution mode.
 * Includes filtering by log level and auto-scroll to latest entries.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { format } from 'date-fns';
import { FileText, Filter, Loader2, AlertCircle, CheckCircle, Info, AlertTriangle } from 'lucide-react';
import clsx from 'clsx';
import type { LogEntry, Execution } from '@/domains/executions';

type LogFilter = 'all' | 'error' | 'warning' | 'info' | 'success';

interface LogsTabProps {
  /** Array of log entries */
  logs: LogEntry[];
  /** Current filter */
  filter?: LogFilter;
  /** Callback when filter changes */
  onFilterChange?: (filter: LogFilter) => void;
  /** Current execution status */
  executionStatus?: Execution['status'];
  /** Additional CSS classes */
  className?: string;
}

/** Get icon and color for log level */
function getLogLevelConfig(level: LogEntry['level']) {
  switch (level) {
    case 'error':
      return { icon: AlertCircle, color: 'text-red-400', bgColor: 'bg-red-500/10' };
    case 'warning':
      return { icon: AlertTriangle, color: 'text-yellow-400', bgColor: 'bg-yellow-500/10' };
    case 'success':
      return { icon: CheckCircle, color: 'text-green-400', bgColor: 'bg-green-500/10' };
    default:
      return { icon: Info, color: 'text-gray-400', bgColor: 'bg-gray-500/10' };
  }
}

/** Filter button component */
function FilterButton({
  filter,
  activeFilter,
  count,
  onClick,
}: {
  filter: LogFilter;
  activeFilter: LogFilter;
  count: number;
  onClick: () => void;
}) {
  const isActive = filter === activeFilter;
  const label = filter === 'all' ? 'All' : filter.charAt(0).toUpperCase() + filter.slice(1);

  return (
    <button
      type="button"
      onClick={onClick}
      className={clsx(
        'px-2 py-1 text-xs rounded-md transition-colors flex items-center gap-1',
        isActive
          ? 'bg-flow-accent/20 text-flow-accent'
          : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800'
      )}
    >
      {label}
      {count > 0 && (
        <span className={clsx(
          'text-[10px] px-1 rounded',
          isActive ? 'bg-flow-accent/30' : 'bg-gray-700'
        )}>
          {count}
        </span>
      )}
    </button>
  );
}

export function LogsTab({
  logs,
  filter = 'all',
  onFilterChange,
  executionStatus,
  className,
}: LogsTabProps) {
  const logsContainerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const prevLogsLengthRef = useRef(logs.length);

  // Filter logs based on current filter
  const filteredLogs = useMemo(() => {
    if (filter === 'all') return logs;
    return logs.filter(log => log.level === filter);
  }, [logs, filter]);

  // Count logs by level
  const counts = useMemo(() => {
    const result: Record<LogFilter, number> = { all: logs.length, error: 0, warning: 0, info: 0, success: 0 };
    for (const log of logs) {
      if (log.level in result) {
        result[log.level as LogFilter]++;
      }
    }
    return result;
  }, [logs]);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && logs.length > prevLogsLengthRef.current && logsContainerRef.current) {
      logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight;
    }
    prevLogsLengthRef.current = logs.length;
  }, [logs.length, autoScroll]);

  // Detect user scroll to disable auto-scroll
  const handleScroll = useCallback(() => {
    if (!logsContainerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
    // Consider "at bottom" if within 50px of the end
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  }, []);

  const isLoading = executionStatus === 'pending' || executionStatus === 'running';

  // Empty state
  if (logs.length === 0) {
    return (
      <div className={clsx('flex flex-col h-full', className)}>
        <div className="flex-1 flex items-center justify-center p-6 text-center">
          <div>
            {isLoading ? (
              <>
                <Loader2 size={32} className="mx-auto mb-3 text-flow-accent animate-spin" />
                <div className="text-sm text-gray-400 mb-1">
                  Waiting for logs...
                </div>
                <div className="text-xs text-gray-500">
                  Logs will appear as the workflow executes
                </div>
              </>
            ) : (
              <>
                <FileText size={32} className="mx-auto mb-3 text-gray-600" />
                <div className="text-sm text-gray-400 mb-1">No logs recorded</div>
                <div className="text-xs text-gray-500">
                  This execution did not produce any log entries
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={clsx('flex flex-col h-full', className)}>
      {/* Filter bar */}
      <div className="border-b border-gray-700 px-3 py-2 flex items-center gap-1 flex-shrink-0">
        <Filter size={14} className="text-gray-500 mr-1" />
        <FilterButton filter="all" activeFilter={filter} count={counts.all} onClick={() => onFilterChange?.('all')} />
        <FilterButton filter="error" activeFilter={filter} count={counts.error} onClick={() => onFilterChange?.('error')} />
        <FilterButton filter="warning" activeFilter={filter} count={counts.warning} onClick={() => onFilterChange?.('warning')} />
        <FilterButton filter="info" activeFilter={filter} count={counts.info} onClick={() => onFilterChange?.('info')} />
      </div>

      {/* Logs list */}
      <div
        ref={logsContainerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-auto p-3 font-mono text-xs"
      >
        {filteredLogs.length === 0 ? (
          <div className="text-center text-gray-500 py-4">
            No {filter} logs
          </div>
        ) : (
          <div className="space-y-1">
            {filteredLogs.map((log) => {
              const config = getLogLevelConfig(log.level);
              const Icon = config.icon;

              return (
                <div
                  key={log.id}
                  className={clsx(
                    'flex items-start gap-2 px-2 py-1.5 rounded-md',
                    config.bgColor
                  )}
                >
                  <Icon size={14} className={clsx('flex-shrink-0 mt-0.5', config.color)} />
                  <div className="flex-1 min-w-0">
                    <div className={clsx('break-words', config.color)}>
                      {log.message}
                    </div>
                    <div className="text-gray-600 mt-0.5">
                      {format(log.timestamp, 'HH:mm:ss.SSS')}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Auto-scroll indicator */}
      {!autoScroll && (
        <button
          type="button"
          onClick={() => {
            setAutoScroll(true);
            if (logsContainerRef.current) {
              logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight;
            }
          }}
          className="absolute bottom-4 right-4 px-3 py-1.5 bg-gray-800 border border-gray-700 rounded-full text-xs text-gray-400 hover:text-white hover:bg-gray-700 transition-colors shadow-lg"
        >
          Jump to latest
        </button>
      )}
    </div>
  );
}
