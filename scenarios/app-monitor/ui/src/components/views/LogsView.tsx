import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import type { ChangeEvent } from 'react';
import { useParams } from 'react-router-dom';
import { appService } from '@/services/api';
import type { AppLogStream } from '@/types';
import './LogsView.css';
import { useAppsStore } from '@/state/appsStore';

export default function LogsView() {
  const { appId } = useParams();
  const apps = useAppsStore(state => state.apps);
  const loadApps = useAppsStore(state => state.loadApps);
  const loadingInitial = useAppsStore(state => state.loadingInitial);
  const hasInitialized = useAppsStore(state => state.hasInitialized);
  const [allLogs, setAllLogs] = useState<string[]>([]);
  const [logStreams, setLogStreams] = useState<AppLogStream[]>([]);
  const [selectedApp, setSelectedApp] = useState(appId || '');
  const [selectedStreamKey, setSelectedStreamKey] = useState<string>('all');
  const [logLevel, setLogLevel] = useState('all');
  const [loading, setLoading] = useState(false);
  const logsEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = useCallback(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  const fetchLogs = useCallback(async () => {
    if (!selectedApp) {
      return;
    }

    setLoading(true);
    try {
      const appEntry = apps.find(app => app.id === selectedApp || app.scenario_name === selectedApp);
      const logIdentifier = [appEntry?.scenario_name, appEntry?.id, selectedApp]
        .map(value => (typeof value === 'string' ? value.trim() : ''))
        .find(value => value.length > 0) || selectedApp;

      const result = await appService.getAppLogs(logIdentifier, 'both');
      const combinedLogs = Array.isArray(result.logs) ? result.logs : [];
      const streams = Array.isArray(result.streams) ? result.streams : [];

      setAllLogs(combinedLogs);
      setLogStreams(streams);
      setSelectedStreamKey(previous => {
        if (previous === 'all') {
          return previous;
        }
        return streams.some(stream => stream.key === previous) ? previous : 'all';
      });

      window.setTimeout(() => {
        scrollToBottom();
      }, 100);
    } catch (error) {
      console.error('Failed to fetch logs:', error);
      setAllLogs([]);
      setLogStreams([]);
      setSelectedStreamKey('all');
    } finally {
      setLoading(false);
    }
  }, [apps, scrollToBottom, selectedApp]);

  useEffect(() => {
    if (!hasInitialized && !loadingInitial) {
      void loadApps();
    }
  }, [hasInitialized, loadApps, loadingInitial]);

  useEffect(() => {
    if (appId) {
      setSelectedApp(appId);
    }
  }, [appId]);

  useEffect(() => {
    if (selectedApp) {
      setSelectedStreamKey('all');
      void fetchLogs();
    } else {
      setAllLogs([]);
      setLogStreams([]);
    }
  }, [fetchLogs, selectedApp]);

  const clearLogs = () => {
    setAllLogs([]);
    setLogStreams([]);
    setSelectedStreamKey('all');
  };

  const handleStreamChange = (event: ChangeEvent<HTMLSelectElement>) => {
    setSelectedStreamKey(event.target.value);
  };

  const getLogClass = (line: string): string => {
    const lower = line.toLowerCase();
    if (lower.includes('error') || lower.includes('fail')) return 'error';
    if (lower.includes('warn')) return 'warning';
    if (lower.includes('success') || lower.includes('started')) return 'success';
    if (lower.includes('info')) return 'info';
    if (lower.includes('debug')) return 'debug';
    return 'log';
  };

  const formatLogLine = (line: string, index: number): JSX.Element => {
    // Try to extract timestamp if present
    const timestampMatch = line.match(/^\[([\dT:.Z-]+)\]/);
    const timestamp = timestampMatch ? timestampMatch[1] : null;
    const content = timestampMatch ? line.substring(timestampMatch[0].length).trim() : line;

    return (
      <div key={index} className={`log-entry ${getLogClass(line)}`}>
        {timestamp && <span className="log-timestamp">[{timestamp}]</span>}
        <span className="log-content">{content}</span>
      </div>
    );
  };

  const streamOptions = useMemo(() => {
    const options: Array<{ key: string; label: string }> = [{ key: 'all', label: 'ALL LOGS' }];
    logStreams.forEach((stream) => {
      if (stream.type === 'lifecycle') {
        options.push({ key: stream.key, label: 'LIFECYCLE' });
      } else {
        const label = stream.label || stream.step || stream.key;
        options.push({ key: stream.key, label: `BACKGROUND · ${label}` });
      }
    });
    return options;
  }, [logStreams]);

  const selectedStream = selectedStreamKey === 'all'
    ? null
    : logStreams.find(stream => stream.key === selectedStreamKey) ?? null;

  const currentLines = useMemo(() => {
    if (selectedStreamKey === 'all') {
      return allLogs;
    }
    return selectedStream?.lines ?? [];
  }, [allLogs, selectedStream, selectedStreamKey]);

  const displayedLogs = useMemo(() => {
    if (logLevel === 'all') {
      return currentLines;
    }
    const filter = logLevel.toLowerCase();
    return currentLines.filter(line => line.toLowerCase().includes(filter));
  }, [currentLines, logLevel]);

  const hasLogsForSelection = currentLines.length > 0;
  const hasAnyLogs = allLogs.length > 0 || logStreams.some(stream => stream.lines.length > 0);
  const isInitialAppLoading = (loadingInitial || !hasInitialized) && apps.length === 0;
  const showSkeleton = (loading && !!selectedApp) || isInitialAppLoading;

  return (
    <div className="logs-view">
      <div className="panel-header">
        <div className="panel-title-spacer" aria-hidden="true" />
        <div className="panel-controls">
          <select
            className="log-filter"
            value={selectedApp}
            onChange={(e) => setSelectedApp(e.target.value)}
            disabled={isInitialAppLoading}
          >
            <option value="">
              {isInitialAppLoading ? 'LOADING SCENARIOS…' : 'SELECT SCENARIO...'}
            </option>
            {apps.map(app => (
              <option key={app.id} value={app.id} title={app.name}>
                {app.scenario_name || app.id}
              </option>
            ))}
          </select>
          <select
            className="log-filter"
            value={selectedStreamKey}
            onChange={handleStreamChange}
            disabled={isInitialAppLoading || (!selectedApp && logStreams.length === 0)}
          >
            {streamOptions.map(option => (
              <option key={option.key} value={option.key}>
                {option.label}
              </option>
            ))}
          </select>
          <select
            className="log-filter"
            value={logLevel}
            onChange={(e) => setLogLevel(e.target.value)}
          >
            <option value="all">ALL LEVELS</option>
            <option value="error">ERROR</option>
            <option value="warning">WARNING</option>
            <option value="info">INFO</option>
            <option value="debug">DEBUG</option>
          </select>
          <button 
            className="control-btn" 
            onClick={fetchLogs} 
            disabled={!selectedApp || loading || isInitialAppLoading}
          >
            ⟳ REFRESH
          </button>
          <button 
            className="control-btn" 
            onClick={clearLogs}
            disabled={!hasAnyLogs}
          >
            CLEAR
          </button>
        </div>
      </div>
      
      <div className="logs-container">
        {showSkeleton ? (
          <div className="logs-skeleton" role="status" aria-live="polite">
            <div className="logs-skeleton__line logs-skeleton__line--wide" />
            <div className="logs-skeleton__line" />
            <div className="logs-skeleton__line logs-skeleton__line--wide" />
            <div className="logs-skeleton__line" />
            <div className="logs-skeleton__line logs-skeleton__line--faint" />
            <div className="logs-skeleton__line" />
            <div className="logs-skeleton__line logs-skeleton__line--wide" />
            <div className="logs-skeleton__line logs-skeleton__line--faint" />
          </div>
        ) : !selectedApp ? (
          <div className="empty-message">Please select an application to view logs</div>
        ) : !hasLogsForSelection ? (
          <div className="empty-message">
            {selectedStream && selectedStream.label
              ? `No logs available for ${selectedStream.label}`
              : 'No logs available for the selected application'}
          </div>
        ) : (
          <>
            {displayedLogs.map((log, index) => formatLogLine(log, index))}
            <div ref={logsEndRef} />
          </>
        )}
      </div>
    </div>
  );
}
