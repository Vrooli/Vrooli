import { format } from 'date-fns';
import type { LogEntry } from '../../store';
import { selectors } from '@constants/selectors';

function getLogColor(level: LogEntry['level']) {
  switch (level) {
    case 'error':
      return 'text-red-400';
    case 'warning':
      return 'text-yellow-400';
    case 'success':
      return 'text-green-400';
    default:
      return 'text-gray-300';
  }
}

interface LogsPanelProps {
  logs: LogEntry[];
}

export function LogsPanel({ logs }: LogsPanelProps) {
  return (
    <div
      className="flex-1 overflow-auto p-3"
      data-testid={selectors.executions.viewer.logs}
    >
      <div className="terminal-output">
        {logs.map((log) => (
          <div
            key={log.id}
            className="flex gap-2 mb-1"
            data-testid={selectors.executions.logEntry}
          >
            <span className="text-xs text-gray-600">
              {format(log.timestamp, 'HH:mm:ss')}
            </span>
            <span className={`flex-1 text-xs ${getLogColor(log.level)}`}>
              {log.message}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
