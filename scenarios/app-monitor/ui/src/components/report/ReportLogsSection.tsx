import clsx from 'clsx';
import {
  AlertTriangle,
  CheckCircle2,
  Eye,
  EyeOff,
  Loader2,
  RefreshCw,
  XCircle,
} from 'lucide-react';

import type {
  ReportAppLogStream,
  ReportConsoleEntry,
  ReportNetworkEntry,
  ReportHealthCheckEntry,
} from './reportTypes';
import type {
  ReportConsoleLogsState,
  ReportLogsState,
  ReportNetworkState,
  ReportHealthChecksState,
  ReportAppStatusState,
} from './useReportIssueState';

interface ReportLogsSectionProps {
  logs: ReportLogsState;
  consoleLogs: ReportConsoleLogsState;
  network: ReportNetworkState;
  health: ReportHealthChecksState;
  status: ReportAppStatusState;
  bridgeCaps: string[];
  appLogsPanelId: string;
  consoleLogsPanelId: string;
  networkPanelId: string;
  healthPanelId: string;
  statusPanelId: string;
}

const ReportLogsSection = ({
  logs,
  consoleLogs,
  network,
  health,
  status,
  bridgeCaps,
  appLogsPanelId,
  consoleLogsPanelId,
  networkPanelId,
  healthPanelId,
  statusPanelId,
}: ReportLogsSectionProps) => {
  const hasLogsCapability = bridgeCaps.includes('logs') || consoleLogs.fromFallback;
  const hasNetworkCapability = bridgeCaps.includes('network') || network.fromFallback;

  const renderHealthStatusIcon = (status: 'pass' | 'warn' | 'fail') => {
    switch (status) {
      case 'pass':
        return <CheckCircle2 aria-hidden size={16} />;
      case 'warn':
        return <AlertTriangle aria-hidden size={16} />;
      case 'fail':
      default:
        return <XCircle aria-hidden size={16} />;
    }
  };

  const formatHealthResponse = (value: string | null) => {
    if (!value) {
      return null;
    }
    const trimmed = value.trim();
    if (!trimmed) {
      return null;
    }
    try {
      const parsed = JSON.parse(trimmed);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return trimmed;
    }
  };

  return (
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
          {logs.streams.map((stream: ReportAppLogStream) => (
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
            onClick={() => logs.fetch()}
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
              onClick={() => logs.fetch()}
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
            !hasLogsCapability && 'report-dialog__logs-include--disabled',
          )}
        >
          <input
            type="checkbox"
            checked={consoleLogs.includeConsoleLogs}
            onChange={(event) => consoleLogs.setIncludeConsoleLogs(event.target.checked)}
            aria-label="Include console logs in report"
            disabled={!hasLogsCapability}
          />
          <span className="report-dialog__logs-title">Console logs</span>
        </label>
        <button
          type="button"
          className="report-dialog__logs-toggle"
          onClick={consoleLogs.toggleExpanded}
          aria-expanded={consoleLogs.expanded}
          aria-controls={consoleLogsPanelId}
          disabled={!hasLogsCapability}
          aria-label={consoleLogs.expanded ? 'Hide console logs' : 'Show console logs'}
        >
          {consoleLogs.expanded ? (
            <EyeOff aria-hidden size={18} />
          ) : (
            <Eye aria-hidden size={18} />
          )}
        </button>
      </div>
      {!hasLogsCapability && !consoleLogs.fromFallback && (
        <div className="report-dialog__logs-alert report-dialog__logs-alert--warning" role="alert">
          <AlertTriangle aria-hidden size={18} />
          <div>
            <p className="report-dialog__logs-alert-title">Console capture unavailable</p>
            <p>
              Runtime diagnostics flagged that this preview&apos;s iframe bridge did not advertise log support. Restart the scenario to refresh the UI bundle, or include diagnostics in the issue so follow-up agents are notified.
            </p>
          </div>
        </div>
      )}
      {consoleLogs.fromFallback && (
        <div className="report-dialog__logs-alert report-dialog__logs-alert--warning" role="alert">
          <AlertTriangle aria-hidden size={18} />
          <div>
            <p className="report-dialog__logs-alert-title">Retrieved via fallback diagnostics</p>
            <p>
              The iframe bridge didn&apos;t respond, so these logs were captured directly from the browser using Chrome DevTools Protocol. This works even when the page fails to load.
            </p>
          </div>
        </div>
      )}
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
            onClick={() => consoleLogs.fetch()}
            disabled={consoleLogs.loading || !hasLogsCapability}
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
              onClick={() => consoleLogs.fetch()}
              disabled={consoleLogs.loading}
            >
              Retry
            </button>
          </div>
        ) : consoleLogs.entries.length === 0 ? (
          <p className="report-dialog__logs-empty">No console output captured yet.</p>
        ) : (
          <div className="report-dialog__logs-content report-dialog__logs-content--console">
            {consoleLogs.entries.map((entry: ReportConsoleEntry, index: number) => (
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
            !hasNetworkCapability && 'report-dialog__logs-include--disabled',
          )}
        >
          <input
            type="checkbox"
            checked={network.includeNetworkRequests}
            onChange={(event) => network.setIncludeNetworkRequests(event.target.checked)}
            aria-label="Include network activity in report"
            disabled={!hasNetworkCapability}
          />
          <span className="report-dialog__logs-title">Network requests</span>
        </label>
        <button
          type="button"
          className="report-dialog__logs-toggle"
          onClick={network.toggleExpanded}
          aria-expanded={network.expanded}
          aria-controls={networkPanelId}
          disabled={!hasNetworkCapability}
          aria-label={network.expanded ? 'Hide network requests' : 'Show network requests'}
        >
          {network.expanded ? (
            <EyeOff aria-hidden size={18} />
          ) : (
            <Eye aria-hidden size={18} />
          )}
        </button>
      </div>
      {!hasNetworkCapability && !network.fromFallback && (
        <div className="report-dialog__logs-alert report-dialog__logs-alert--warning" role="alert">
          <AlertTriangle aria-hidden size={18} />
          <div>
            <p className="report-dialog__logs-alert-title">Network capture unavailable</p>
            <p>
              Runtime diagnostics flagged that this preview&apos;s iframe bridge did not advertise network request support. Restart the scenario to refresh the UI bundle, or include diagnostics in the issue so follow-up agents are notified.
            </p>
          </div>
        </div>
      )}
      {network.fromFallback && (
        <div className="report-dialog__logs-alert report-dialog__logs-alert--warning" role="alert">
          <AlertTriangle aria-hidden size={18} />
          <div>
            <p className="report-dialog__logs-alert-title">Retrieved via fallback diagnostics</p>
            <p>
              The iframe bridge didn&apos;t respond, so these network requests were captured directly from the browser using Chrome DevTools Protocol. This works even when the page fails to load.
            </p>
          </div>
        </div>
      )}
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
            onClick={() => network.fetch()}
            disabled={network.loading || !hasNetworkCapability}
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
              onClick={() => network.fetch()}
              disabled={network.loading}
            >
              Retry
            </button>
          </div>
        ) : network.events.length === 0 ? (
          <p className="report-dialog__logs-empty">No network requests captured.</p>
        ) : (
          <div className="report-dialog__logs-content report-dialog__logs-content--network">
            {network.events.map((entry: ReportNetworkEntry, index: number) => {
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

    <section className="report-dialog__logs-section">
      <div className="report-dialog__logs-header">
        <label
          className={clsx(
            'report-dialog__logs-include',
            !health.includeHealthChecks && 'report-dialog__logs-include--off',
          )}
        >
          <input
            type="checkbox"
            checked={health.includeHealthChecks}
            onChange={(event) => health.setIncludeHealthChecks(event.target.checked)}
            aria-label="Include health checks in report"
          />
          <span className="report-dialog__logs-title">Health checks</span>
        </label>
        <button
          type="button"
          className="report-dialog__logs-toggle"
          onClick={health.toggleExpanded}
          aria-expanded={health.expanded}
          aria-controls={healthPanelId}
          aria-label={health.expanded ? 'Hide health checks' : 'Show health checks'}
        >
          {health.expanded ? (
            <EyeOff aria-hidden size={18} />
          ) : (
            <Eye aria-hidden size={18} />
          )}
        </button>
      </div>
      <div
        id={healthPanelId}
        className="report-dialog__logs-panel"
        style={health.expanded ? undefined : { display: 'none' }}
        aria-hidden={!health.expanded}
      >
        <div className="report-dialog__logs-meta">
          <span>
            {health.loading
              ? 'Running health checks…'
              : health.entries.length > 0
                ? `Captured ${health.entries.length}${typeof health.total === 'number' && health.total > health.entries.length ? ` of ${health.total}` : ''} checks${health.formattedCapturedAt ? ` (captured ${health.formattedCapturedAt})` : ''}.`
                : health.error
                  ? 'Health checks unavailable.'
                  : 'No health checks recorded.'}
          </span>
          <button
            type="button"
            className="report-dialog__logs-refresh"
            onClick={() => health.fetch()}
            disabled={health.loading}
          >
            {health.loading ? (
              <Loader2 aria-hidden size={14} className="spinning" />
            ) : (
              <RefreshCw aria-hidden size={14} />
            )}
            <span>{health.loading ? 'Running' : 'Refresh'}</span>
          </button>
        </div>
        {health.loading ? (
          <div className="report-dialog__logs-loading">
            <Loader2 aria-hidden size={18} className="spinning" />
            <span>Collecting health data…</span>
          </div>
        ) : health.error ? (
          <div className="report-dialog__logs-message">
            <p>{health.error}</p>
            <button
              type="button"
              className="report-dialog__button report-dialog__button--ghost"
              onClick={() => health.fetch()}
              disabled={health.loading}
            >
              Retry
            </button>
          </div>
        ) : health.entries.length === 0 ? (
          <p className="report-dialog__logs-empty">No health checks available.</p>
        ) : (
          <div className="report-dialog__health-list">
            {health.entries.map((entry: ReportHealthCheckEntry) => {
              const formattedResponse = formatHealthResponse(entry.response);

              return (
                <div
                  key={entry.id}
                  className={clsx(
                    'report-dialog__health-item',
                    `report-dialog__health-item--${entry.status}`,
                  )}
                >
                  <span
                    className={clsx(
                      'report-dialog__health-status',
                      `report-dialog__health-status--${entry.status}`,
                    )}
                    aria-label={`Health check ${entry.status}`}
                  >
                    {renderHealthStatusIcon(entry.status)}
                  </span>
                  <div className="report-dialog__health-body">
                    <div className="report-dialog__health-row">
                      <span className="report-dialog__health-name">{entry.name}</span>
                      {entry.latencyMs !== null && (
                        <span className="report-dialog__health-latency">{`${entry.latencyMs} ms`}</span>
                      )}
                    </div>
                    {entry.endpoint && (
                      <div className="report-dialog__health-endpoint" title={entry.endpoint}>{entry.endpoint}</div>
                    )}
                    {entry.message && (
                      <p className="report-dialog__health-message">{entry.message}</p>
                    )}
                    {entry.code && (
                      <p className="report-dialog__health-code">{entry.code}</p>
                    )}
                    {formattedResponse && (
                      <div className="report-dialog__health-response">
                        <span className="report-dialog__health-response-label">Response</span>
                        <pre>{formattedResponse}</pre>
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </section>

    <section className="report-dialog__logs-section">
      <div className="report-dialog__logs-header">
        <label
          className={clsx(
            'report-dialog__logs-include',
            !status.includeAppStatus && 'report-dialog__logs-include--off',
          )}
        >
          <input
            type="checkbox"
            checked={status.includeAppStatus}
            onChange={(event) => status.setIncludeAppStatus(event.target.checked)}
            aria-label="Include app status in report"
          />
          <span className="report-dialog__logs-title">App status</span>
        </label>
        <button
          type="button"
          className="report-dialog__logs-toggle"
          onClick={status.toggleExpanded}
          aria-expanded={status.expanded}
          aria-controls={statusPanelId}
          aria-label={status.expanded ? 'Hide app status' : 'Show app status'}
        >
          {status.expanded ? (
            <EyeOff aria-hidden size={18} />
          ) : (
            <Eye aria-hidden size={18} />
          )}
        </button>
      </div>
      <div
        id={statusPanelId}
        className="report-dialog__logs-panel"
        style={status.expanded ? undefined : { display: 'none' }}
        aria-hidden={!status.expanded}
      >
        <div className="report-dialog__logs-meta">
          <span>
            {status.loading
              ? 'Fetching app status…'
              : status.snapshot
                ? `${status.snapshot.statusLabel}${status.formattedCapturedAt ? ` (captured ${status.formattedCapturedAt})` : ''}.`
                : status.error
                  ? 'App status unavailable.'
                  : 'No app status captured.'}
          </span>
          <button
            type="button"
            className="report-dialog__logs-refresh"
            onClick={() => status.fetch()}
            disabled={status.loading}
          >
            {status.loading ? (
              <Loader2 aria-hidden size={14} className="spinning" />
            ) : (
              <RefreshCw aria-hidden size={14} />
            )}
            <span>{status.loading ? 'Loading' : 'Refresh'}</span>
          </button>
        </div>
        {status.loading ? (
          <div className="report-dialog__logs-loading">
            <Loader2 aria-hidden size={18} className="spinning" />
            <span>Gathering scenario status…</span>
          </div>
        ) : status.error ? (
          <div className="report-dialog__logs-message">
            <p>{status.error}</p>
            <button
              type="button"
              className="report-dialog__button report-dialog__button--ghost"
              onClick={() => status.fetch()}
              disabled={status.loading}
            >
              Retry
            </button>
          </div>
        ) : !status.snapshot ? (
          <p className="report-dialog__logs-empty">No status snapshot available.</p>
        ) : status.snapshot.details.length === 0 ? (
          <p className="report-dialog__logs-empty">Status snapshot did not include detailed output.</p>
        ) : (
          <pre className="report-dialog__logs-content">{status.snapshot.details.join('\n')}</pre>
        )}
      </div>
    </section>
  </div>
  );
};

export default ReportLogsSection;
export type { ReportLogsSectionProps };
