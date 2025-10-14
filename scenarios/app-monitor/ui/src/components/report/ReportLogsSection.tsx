import clsx from 'clsx';
import { Eye, EyeOff, Loader2, RefreshCw } from 'lucide-react';

import type {
  ReportConsoleLogsState,
  ReportLogsState,
  ReportNetworkState,
} from './useReportIssueState';

interface ReportLogsSectionProps {
  logs: ReportLogsState;
  consoleLogs: ReportConsoleLogsState;
  network: ReportNetworkState;
  bridgeCaps: string[];
  appLogsPanelId: string;
  consoleLogsPanelId: string;
  networkPanelId: string;
}

const ReportLogsSection = ({
  logs,
  consoleLogs,
  network,
  bridgeCaps,
  appLogsPanelId,
  consoleLogsPanelId,
  networkPanelId,
}: ReportLogsSectionProps) => (
  <div className="report-dialog__logs">
    <section className="report-dialog__logs-section">
      <div className="report-dialog__logs-header">
        <label
          className={clsx(
            'report-dialog__logs-include',
            !logs.includeAppLogs && 'report-dialog__logs-include--off',
          )}
        >
          <input
            type="checkbox"
            checked={logs.includeAppLogs}
            onChange={(event) => logs.setIncludeAppLogs(event.target.checked)}
            aria-label="Include app logs in report"
          />
          <span className="report-dialog__logs-title">App logs</span>
        </label>
        <button
          type="button"
          className="report-dialog__logs-toggle"
          onClick={logs.toggleExpanded}
          aria-expanded={logs.expanded}
          aria-controls={appLogsPanelId}
          aria-label={logs.expanded ? 'Hide app logs' : 'Show app logs'}
        >
          {logs.expanded ? (
            <EyeOff aria-hidden size={18} />
          ) : (
            <Eye aria-hidden size={18} />
          )}
        </button>
      </div>
      {logs.streams.length > 0 && (
        <div className="report-dialog__logs-streams">
          {logs.streams.map(stream => (
            <label key={stream.key} className="report-dialog__logs-stream">
              <input
                type="checkbox"
                checked={logs.selections[stream.key] !== false}
                onChange={(event) => logs.toggleStream(stream.key, event.target.checked)}
                disabled={!logs.includeAppLogs}
              />
              <span>{stream.label}</span>
              <span className="report-dialog__logs-count">{stream.total}</span>
            </label>
          ))}
          <div className="report-dialog__logs-streams-meta">
            {logs.includeAppLogs ? (
              <span>{`${logs.selectedCount}/${logs.totalStreamCount} streams included`}</span>
            ) : (
              <span>Enable app logs to include these streams.</span>
            )}
            {logs.includeAppLogs && logs.selectedCount === 0 && (
              <span className="report-dialog__logs-streams-warning">No log streams selected</span>
            )}
          </div>
        </div>
      )}
      <div
        id={appLogsPanelId}
        className="report-dialog__logs-panel"
        style={logs.expanded ? undefined : { display: 'none' }}
        aria-hidden={!logs.expanded}
      >
        <div className="report-dialog__logs-meta">
          <span>
            {logs.loading
              ? 'Loading logs…'
              : logs.logs.length > 0
                ? `Showing last ${logs.logs.length}${logs.truncated ? ` of ${logs.total}` : ''} lines${logs.formattedCapturedAt ? ` (captured ${logs.formattedCapturedAt})` : ''}.`
                : logs.error
                  ? 'Logs unavailable.'
                  : 'No logs captured yet.'}
          </span>
          <button
            type="button"
            className="report-dialog__logs-refresh"
            onClick={logs.fetch}
            disabled={logs.loading}
          >
            {logs.loading ? (
              <Loader2 aria-hidden size={14} className="spinning" />
            ) : (
              <RefreshCw aria-hidden size={14} />
            )}
            <span>{logs.loading ? 'Loading' : 'Refresh'}</span>
          </button>
        </div>
        {logs.loading ? (
          <div className="report-dialog__logs-loading">
            <Loader2 aria-hidden size={18} className="spinning" />
            <span>Fetching logs…</span>
          </div>
        ) : logs.error ? (
          <div className="report-dialog__logs-message">
            <p>{logs.error}</p>
            <button
              type="button"
              className="report-dialog__button report-dialog__button--ghost"
              onClick={logs.fetch}
              disabled={logs.loading}
            >
              Retry
            </button>
          </div>
        ) : logs.logs.length === 0 ? (
          <p className="report-dialog__logs-empty">No logs available for this scenario.</p>
        ) : (
          <pre className="report-dialog__logs-content">{logs.logs.join('\n')}</pre>
        )}
      </div>
    </section>

    <section className="report-dialog__logs-section">
      <div className="report-dialog__logs-header">
        <label
          className={clsx(
            'report-dialog__logs-include',
            !consoleLogs.includeConsoleLogs && 'report-dialog__logs-include--off',
            !bridgeCaps.includes('logs') && 'report-dialog__logs-include--disabled',
          )}
        >
          <input
            type="checkbox"
            checked={consoleLogs.includeConsoleLogs}
            onChange={(event) => consoleLogs.setIncludeConsoleLogs(event.target.checked)}
            aria-label="Include console logs in report"
            disabled={!bridgeCaps.includes('logs')}
          />
          <span className="report-dialog__logs-title">Console logs</span>
        </label>
        <button
          type="button"
          className="report-dialog__logs-toggle"
          onClick={consoleLogs.toggleExpanded}
          aria-expanded={consoleLogs.expanded}
          aria-controls={consoleLogsPanelId}
          disabled={!bridgeCaps.includes('logs')}
          aria-label={consoleLogs.expanded ? 'Hide console logs' : 'Show console logs'}
        >
          {consoleLogs.expanded ? (
            <EyeOff aria-hidden size={18} />
          ) : (
            <Eye aria-hidden size={18} />
          )}
        </button>
      </div>
      <div
        id={consoleLogsPanelId}
        className="report-dialog__logs-panel"
        style={consoleLogs.expanded ? undefined : { display: 'none' }}
        aria-hidden={!consoleLogs.expanded}
      >
        <div className="report-dialog__logs-meta">
          <span>
            {consoleLogs.loading
              ? 'Loading console output…'
              : consoleLogs.entries.length > 0
                ? `Showing last ${consoleLogs.entries.length}${consoleLogs.truncated ? ` of ${consoleLogs.total}` : ''} events${consoleLogs.formattedCapturedAt ? ` (captured ${consoleLogs.formattedCapturedAt})` : ''}.`
                : consoleLogs.error
                  ? 'Console logs unavailable.'
                  : 'No console output captured yet.'}
          </span>
          <button
            type="button"
            className="report-dialog__logs-refresh"
            onClick={consoleLogs.fetch}
            disabled={consoleLogs.loading || !bridgeCaps.includes('logs')}
          >
            {consoleLogs.loading ? (
              <Loader2 aria-hidden size={14} className="spinning" />
            ) : (
              <RefreshCw aria-hidden size={14} />
            )}
            <span>{consoleLogs.loading ? 'Loading' : 'Refresh'}</span>
          </button>
        </div>
        {consoleLogs.loading ? (
          <div className="report-dialog__logs-loading">
            <Loader2 aria-hidden size={18} className="spinning" />
            <span>Fetching console output…</span>
          </div>
        ) : consoleLogs.error ? (
          <div className="report-dialog__logs-message">
            <p>{consoleLogs.error}</p>
            <button
              type="button"
              className="report-dialog__button report-dialog__button--ghost"
              onClick={consoleLogs.fetch}
              disabled={consoleLogs.loading}
            >
              Retry
            </button>
          </div>
        ) : consoleLogs.entries.length === 0 ? (
          <p className="report-dialog__logs-empty">No console output captured yet.</p>
        ) : (
          <div className="report-dialog__logs-content report-dialog__logs-content--console">
            {consoleLogs.entries.map((entry, index) => (
              <div
                key={`${entry.payload.ts}-${index}`}
                className={clsx(
                  'report-dialog__console-line',
                  `report-dialog__console-line--${entry.severity}`,
                )}
              >
                <div className="report-dialog__console-meta">
                  <span className="report-dialog__console-timestamp">{entry.timestamp}</span>
                  <span className="report-dialog__console-source">{entry.source}</span>
                  <span className="report-dialog__console-level">{entry.payload.level.toUpperCase()}</span>
                </div>
                <div className="report-dialog__console-body">{entry.body}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>

    <section className="report-dialog__logs-section">
      <div className="report-dialog__logs-header">
        <label
          className={clsx(
            'report-dialog__logs-include',
            !network.includeNetworkRequests && 'report-dialog__logs-include--off',
            !bridgeCaps.includes('network') && 'report-dialog__logs-include--disabled',
          )}
        >
          <input
            type="checkbox"
            checked={network.includeNetworkRequests}
            onChange={(event) => network.setIncludeNetworkRequests(event.target.checked)}
            aria-label="Include network activity in report"
            disabled={!bridgeCaps.includes('network')}
          />
          <span className="report-dialog__logs-title">Network requests</span>
        </label>
        <button
          type="button"
          className="report-dialog__logs-toggle"
          onClick={network.toggleExpanded}
          aria-expanded={network.expanded}
          aria-controls={networkPanelId}
          disabled={!bridgeCaps.includes('network')}
          aria-label={network.expanded ? 'Hide network requests' : 'Show network requests'}
        >
          {network.expanded ? (
            <EyeOff aria-hidden size={18} />
          ) : (
            <Eye aria-hidden size={18} />
          )}
        </button>
      </div>
      <div
        id={networkPanelId}
        className="report-dialog__logs-panel"
        style={network.expanded ? undefined : { display: 'none' }}
        aria-hidden={!network.expanded}
      >
        <div className="report-dialog__logs-meta">
          <span>
            {network.loading
              ? 'Loading requests…'
              : network.events.length > 0
                ? `Showing last ${network.events.length}${network.truncated ? ` of ${network.total}` : ''} requests${network.formattedCapturedAt ? ` (captured ${network.formattedCapturedAt})` : ''}.`
                : network.error
                  ? 'Network requests unavailable.'
                  : 'No network requests captured yet.'}
          </span>
          <button
            type="button"
            className="report-dialog__logs-refresh"
            onClick={network.fetch}
            disabled={network.loading || !bridgeCaps.includes('network')}
          >
            {network.loading ? (
              <Loader2 aria-hidden size={14} className="spinning" />
            ) : (
              <RefreshCw aria-hidden size={14} />
            )}
            <span>{network.loading ? 'Loading' : 'Refresh'}</span>
          </button>
        </div>
        {network.loading ? (
          <div className="report-dialog__logs-loading">
            <Loader2 aria-hidden size={18} className="spinning" />
            <span>Fetching requests…</span>
          </div>
        ) : network.error ? (
          <div className="report-dialog__logs-message">
            <p>{network.error}</p>
            <button
              type="button"
              className="report-dialog__button report-dialog__button--ghost"
              onClick={network.fetch}
              disabled={network.loading}
            >
              Retry
            </button>
          </div>
        ) : network.events.length === 0 ? (
          <p className="report-dialog__logs-empty">No network requests captured.</p>
        ) : (
          <div className="report-dialog__logs-content report-dialog__logs-content--network">
            {network.events.map((entry, index) => {
              const statusClass = entry.payload.status && entry.payload.status >= 400
                ? 'report-dialog__network-status--error'
                : entry.payload.ok === false
                  ? 'report-dialog__network-status--error'
                  : 'report-dialog__network-status--ok';
              return (
                <div className="report-dialog__network-entry" key={`${entry.payload.ts}-${index}`}>
                  <div className="report-dialog__network-row">
                    <span className="report-dialog__network-timestamp">{entry.timestamp}</span>
                    <span className="report-dialog__network-method">{entry.method}</span>
                    <span className={clsx('report-dialog__network-status', statusClass)}>{entry.statusLabel}</span>
                    {entry.durationLabel && (
                      <span className="report-dialog__network-duration">{entry.durationLabel}</span>
                    )}
                    {entry.payload.requestId && (
                      <span className="report-dialog__network-request">{entry.payload.requestId}</span>
                    )}
                  </div>
                  <div className="report-dialog__network-url" title={entry.payload.url}>{entry.payload.url}</div>
                  {entry.errorText && (
                    <div className="report-dialog__network-error">{entry.errorText}</div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </section>
  </div>
);

export default ReportLogsSection;
