import { useEffect, useMemo, useRef, useState } from 'react';
import { RefreshCw, X } from 'lucide-react';
import clsx from 'clsx';
import type { App, AppLogStream } from '@/types';
import type { UseAppLogsResult } from '@/hooks/useAppLogs';
import './AppLogsPanel.css';

type LogLevelFilter = 'all' | 'error' | 'warning' | 'info' | 'debug';

const LOG_LEVEL_OPTIONS: Array<{ value: LogLevelFilter; label: string }> = [
  { value: 'all', label: 'All levels' },
  { value: 'error', label: 'Errors' },
  { value: 'warning', label: 'Warnings' },
  { value: 'info', label: 'Info' },
  { value: 'debug', label: 'Debug' },
];

const deriveStreamOptions = (streams: AppLogStream[]): Array<{ key: string; label: string }> => {
  const options: Array<{ key: string; label: string }> = [{ key: 'all', label: 'All logs' }];
  streams.forEach((stream) => {
    if (stream.type === 'lifecycle') {
      options.push({ key: stream.key, label: 'Lifecycle events' });
      return;
    }
    const label = stream.label || stream.step || stream.command || stream.key;
    options.push({ key: stream.key, label: `Background · ${label}` });
  });
  return options;
};

const classifyLogLine = (line: string): LogLevelFilter => {
  const lower = line.toLowerCase();
  if (lower.includes('error') || lower.includes('fail')) {
    return 'error';
  }
  if (lower.includes('warn')) {
    return 'warning';
  }
  if (lower.includes('debug')) {
    return 'debug';
  }
  if (lower.includes('info')) {
    return 'info';
  }
  return 'all';
};

interface AppLogsPanelProps extends UseAppLogsResult {
  app?: App | null;
  onClose: () => void;
  title?: string;
}

const formatTimestamp = (line: string): { timestamp: string | null; content: string } => {
  const match = line.match(/^\[([\dT:.Z-]+)\]\s*(.*)$/);
  if (!match) {
    return { timestamp: null, content: line };
  }
  return { timestamp: match[1] ?? null, content: match[2] ?? '' };
};

export function AppLogsPanel({
  app,
  identifier,
  logs,
  streams,
  loading,
  error,
  lastUpdatedAt,
  refresh,
  clear,
  onClose,
  title,
}: AppLogsPanelProps) {
  const [selectedStreamKey, setSelectedStreamKey] = useState<string>('all');
  const [levelFilter, setLevelFilter] = useState<LogLevelFilter>('all');
  const [autoScroll, setAutoScroll] = useState(true);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const endRef = useRef<HTMLDivElement | null>(null);

  const streamOptions = useMemo(() => deriveStreamOptions(streams), [streams]);

  useEffect(() => {
    if (selectedStreamKey === 'all') {
      return;
    }
    const stillPresent = streams.some(stream => stream.key === selectedStreamKey);
    if (!stillPresent) {
      setSelectedStreamKey('all');
    }
  }, [selectedStreamKey, streams]);

  const activeStream = useMemo(() => (
    selectedStreamKey === 'all'
      ? null
      : streams.find(stream => stream.key === selectedStreamKey) ?? null
  ), [selectedStreamKey, streams]);

  const activeLines = useMemo(() => (
    selectedStreamKey === 'all' ? logs : (activeStream?.lines ?? [])
  ), [activeStream?.lines, logs, selectedStreamKey]);

  const filteredLines = useMemo(() => {
    if (levelFilter === 'all') {
      return activeLines;
    }
    return activeLines.filter(line => classifyLogLine(line) === levelFilter);
  }, [activeLines, levelFilter]);

  useEffect(() => {
    if (!autoScroll) {
      return;
    }
    endRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [filteredLines.length, autoScroll, lastUpdatedAt]);

  const handleScroll = () => {
    const container = containerRef.current;
    if (!container) {
      return;
    }
    const { scrollTop, scrollHeight, clientHeight } = container;
    const isAtBottom = scrollHeight - clientHeight - scrollTop < 32;
    setAutoScroll(isAtBottom);
  };

  const handleRefresh = () => {
    void refresh();
  };

  const handleClear = () => {
    clear();
    setAutoScroll(true);
  };

  const headerTitle = title || app?.scenario_name || app?.name || identifier || 'Scenario logs';
  const subtitle = activeStream?.label || (activeStream?.type === 'lifecycle' ? 'Lifecycle' : null);
  const logCountLabel = `${filteredLines.length.toLocaleString()} line${filteredLines.length === 1 ? '' : 's'}`;

  return (
    <div className="logs-panel" role="region" aria-label="Application logs">
      <header className="logs-panel__header">
        <div className="logs-panel__titles">
          <h2>{headerTitle}</h2>
          <div className="logs-panel__subtitle">
            <span>{subtitle ?? 'Combined streams'}</span>
            <span aria-hidden>•</span>
            <span>{logCountLabel}</span>
            {lastUpdatedAt && (
              <span className="logs-panel__meta">Updated {new Date(lastUpdatedAt).toLocaleTimeString()}</span>
            )}
          </div>
        </div>
        <div className="logs-panel__actions">
          <button
            type="button"
            className="logs-panel__icon-btn"
            onClick={handleRefresh}
            disabled={loading}
            aria-label="Refresh scenario logs"
          >
            <RefreshCw aria-hidden className={clsx(loading && 'spinning')} size={18} />
          </button>
          <button
            type="button"
            className="logs-panel__icon-btn"
            onClick={onClose}
            aria-label="Close logs panel"
          >
            <X aria-hidden size={18} />
          </button>
        </div>
      </header>

      <div className="logs-panel__controls">
        <label>
          <span>Stream</span>
          <select
            value={selectedStreamKey}
            onChange={event => setSelectedStreamKey(event.target.value)}
          >
            {streamOptions.map(option => (
              <option key={option.key} value={option.key}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>Level</span>
          <select
            value={levelFilter}
            onChange={event => setLevelFilter(event.target.value as LogLevelFilter)}
          >
            {LOG_LEVEL_OPTIONS.map(option => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
        <label className="logs-panel__checkbox" aria-live="polite">
          <input
            type="checkbox"
            checked={autoScroll}
            onChange={event => setAutoScroll(event.target.checked)}
          />
          <span>Auto-scroll</span>
        </label>
        <button
          type="button"
          className="logs-panel__control-btn"
          onClick={handleClear}
          disabled={logs.length === 0 && streams.length === 0}
        >
          Clear
        </button>
      </div>

      <div
        ref={containerRef}
        className="logs-panel__scroll"
        onScroll={handleScroll}
      >
        {loading && filteredLines.length === 0 && !error ? (
          <div className="logs-panel__empty" role="status">
            Fetching logs…
          </div>
        ) : error ? (
          <div className="logs-panel__empty" role="alert">
            {error}
          </div>
        ) : filteredLines.length === 0 ? (
          <div className="logs-panel__empty">
            No log entries yet.
          </div>
        ) : (
          filteredLines.map((line, index) => {
            const { timestamp, content } = formatTimestamp(line);
            const level = classifyLogLine(line);
            return (
              <div key={`${index}-${line}`} className={clsx('logs-panel__line', `logs-panel__line--${level}`)}>
                {timestamp && <span className="logs-panel__timestamp">[{timestamp}]</span>}
                <span className="logs-panel__content">{content}</span>
              </div>
            );
          })
        )}
        <div ref={endRef} />
      </div>
    </div>
  );
}

export default AppLogsPanel;
