import type { Alert } from '../../types';
import type { AlertFeed } from '../../features/monitoring/serverAlerts';
import { formatTime } from '../utils/formatters';

interface AlertPanelProps {
  alerts: Alert[];
  /** Server feed state. Absent means the caller supplies alerts directly. */
  feed?: AlertFeed;
}

const emptyMessage = (feed?: AlertFeed): string => {
  switch (feed?.status) {
    case 'loading':
      return 'LOADING ALERT STATE';
    case 'unevaluated':
      return 'ALERT RULES HAVE NOT RUN YET';
    case 'paused':
      return 'MONITORING PAUSED — ALERTS NOT EVALUATED';
    case 'error':
      return 'ALERT STATE UNAVAILABLE';
    default:
      return 'NO ACTIVE ALERTS';
  }
};

export const AlertPanel = ({ alerts, feed }: AlertPanelProps) => {
  const degraded = feed?.status === 'error' || feed?.status === 'paused';
  return (
    <section className="alert-panel card">
      <div className="flex-row-between mb-md">
        <h2 className="section-heading">ACTIVE ALERTS</h2>
        <span
          className={`alert-badge text-sm ${alerts.length > 0 || degraded ? 'alert-badge-error' : 'alert-badge-success'}`}
        >
          {alerts.length}
        </span>
      </div>

      {feed?.status === 'error' && alerts.length > 0 ? (
        <div className="text-muted text-sm mb-sm" role="status">
          Alert state could not be refreshed ({feed.error}); showing the last known alerts.
        </div>
      ) : null}

      <div className="alert-list">
        {alerts.length === 0 ? (
          <div className="text-center text-muted p-lg text-lg" role="status">
            {emptyMessage(feed)}
            {feed?.status === 'error' && feed.error ? <div className="text-sm">{feed.error}</div> : null}
          </div>
        ) : (
          alerts.map(alert => (
            <div key={alert.id} className="alert-item pool-item mb-sm" data-sm-style="sm-style-2f44229022">
              <div className="flex-row-between">
                <span className={`status-color-${alert.severity.toLowerCase()}`}>
                  [{alert.severity.toUpperCase()}] {alert.category}
                </span>
                <span className="text-muted text-sm">
                  {alert.timestamp ? formatTime(alert.timestamp) : ''}
                </span>
              </div>
              <div data-sm-style="sm-style-4edd7007ae">
                {alert.message}
              </div>
              {alert.details ? <div className="text-muted text-sm">{alert.details}</div> : null}
            </div>
          ))
        )}
      </div>

      {feed && feed.ruleIssues.length > 0 ? (
        <ul className="text-muted text-sm mt-sm" aria-label="Rules that cannot evaluate">
          {feed.ruleIssues.map(issue => (
            <li key={issue.ruleId}>
              {issue.ruleId}: {issue.status} — {issue.reason}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
};
