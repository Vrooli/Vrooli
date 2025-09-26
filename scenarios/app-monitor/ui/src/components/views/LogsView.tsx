import { useState, useEffect, useRef, useCallback } from 'react';
import type { ChangeEvent } from 'react';
import { useParams } from 'react-router-dom';
import { appService } from '@/services/api';
import type { App } from '@/types';
import './LogsView.css';

export default function LogsView() {
  const { appId } = useParams();
  const [logs, setLogs] = useState<string[]>([]);
  const [apps, setApps] = useState<App[]>([]);
  const [selectedApp, setSelectedApp] = useState(appId || '');
  const [logType, setLogType] = useState<'both' | 'lifecycle' | 'background'>('both');
  const [logLevel, setLogLevel] = useState('all');
  const [loading, setLoading] = useState(false);
  const logsEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = useCallback(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  const fetchApps = useCallback(async () => {
    const fetchedApps = await appService.getApps();
    setApps(fetchedApps);
  }, []);

  const fetchLogs = useCallback(async () => {
    if (!selectedApp) {
      return;
    }

    setLoading(true);
    try {
      const appName = apps.find(a => a.id === selectedApp)?.name || selectedApp;
      const result = await appService.getAppLogs(appName, logType);
      setLogs(result.logs || []);
      window.setTimeout(() => {
        scrollToBottom();
      }, 100);
    } catch (error) {
      console.error('Failed to fetch logs:', error);
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, [apps, logType, scrollToBottom, selectedApp]);

  useEffect(() => {
    void fetchApps();
  }, [fetchApps]);

  useEffect(() => {
    if (appId) {
      setSelectedApp(appId);
    }
  }, [appId]);

  useEffect(() => {
    if (selectedApp) {
      void fetchLogs();
    }
  }, [fetchLogs, selectedApp]);

  const clearLogs = () => {
    setLogs([]);
  };

  const handleLogTypeChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const nextType = event.target.value as 'both' | 'lifecycle' | 'background';
    setLogType(nextType);
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

  return (
    <div className="logs-view">
      <div className="panel-header">
        <h2>APPLICATION LOGS</h2>
        <div className="panel-controls">
          <select
            className="log-filter"
            value={selectedApp}
            onChange={(e) => setSelectedApp(e.target.value)}
          >
            <option value="">SELECT APP...</option>
            {apps.map(app => (
              <option key={app.id} value={app.id}>
                {app.name}
              </option>
            ))}
          </select>
          <select
            className="log-filter"
            value={logType}
            onChange={handleLogTypeChange}
          >
            <option value="both">ALL LOGS</option>
            <option value="lifecycle">LIFECYCLE</option>
            <option value="background">BACKGROUND</option>
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
            disabled={!selectedApp || loading}
          >
            ⟳ REFRESH
          </button>
          <button 
            className="control-btn" 
            onClick={clearLogs}
            disabled={logs.length === 0}
          >
            CLEAR
          </button>
        </div>
      </div>
      
      <div className="logs-container">
        {loading ? (
          <div className="loading-message">Loading logs...</div>
        ) : !selectedApp ? (
          <div className="empty-message">Please select an application to view logs</div>
        ) : logs.length === 0 ? (
          <div className="empty-message">No logs available for the selected application</div>
        ) : (
          <>
            {logs
              .filter(log => {
                if (logLevel === 'all') return true;
                return log.toLowerCase().includes(logLevel);
              })
              .map((log, index) => formatLogLine(log, index))}
            <div ref={logsEndRef} />
          </>
        )}
      </div>
    </div>
  );
}
