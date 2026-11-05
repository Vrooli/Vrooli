import { useMemo, useRef, useState } from 'react';
import type { Alert, CardType } from '../../types';
import type { AlertFeed } from '../../features/monitoring/serverAlerts';

interface AlertPanelProps {
  alerts: Alert[];
  /** Server feed state. Absent means the caller supplies alerts directly. */
  feed?: AlertFeed;
  onInspect?: (card: CardType, alert: Alert, revalidated: boolean) => void;
  viewPaused?: boolean;
  onInspectOther?: () => void;
  onRefresh?: () => void;
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
      return feed?.ruleIssues.length ? 'NO ACTIVE ALERTS IN EVALUATED SCOPE' : 'NO ACTIVE ALERTS';
  }
};

const inspectTarget = (category: string): CardType | undefined => {
  if (category.startsWith('cpu')) return 'cpu';
  if (category.startsWith('memory')) return 'memory';
  if (category.startsWith('disk')) return 'disk';
  if (category.startsWith('gpu')) return 'gpu';
  if (category.startsWith('tcp') || category.startsWith('network')) return 'network';
  return undefined;
};
const severityRank: Record<string, number> = { critical: 0, high: 1, warning: 2, medium: 3, low: 4 };

export const AlertPanel = ({ alerts, feed, onInspect, onInspectOther, onRefresh, viewPaused = false }: AlertPanelProps) => {
  const degraded = Boolean(feed && feed.status !== 'ready');
  const [reading, setReading] = useState(false);
  const order = useRef<string[]>([]);
  const ranked = useMemo(() => [...alerts].filter(alert => !alert.resolved).sort((a, b) =>
    (severityRank[a.severity] ?? 5) - (severityRank[b.severity] ?? 5) ||
    (Date.parse(b.timestamp) || 0) - (Date.parse(a.timestamp) || 0) || a.id.localeCompare(b.id)), [alerts]);
  if (!reading) order.current = ranked.map(alert => alert.id);
  const visible = reading ? [...ranked].sort((a, b) => {
    const ai = order.current.indexOf(a.id), bi = order.current.indexOf(b.id);
    return (ai < 0 ? Number.MAX_SAFE_INTEGER : ai) - (bi < 0 ? Number.MAX_SAFE_INTEGER : bi);
  }) : ranked;
  return (
    <section className="alert-panel card" onFocus={() => { setReading(true); }} onBlur={event => { if (!event.currentTarget.contains(event.relatedTarget)) setReading(false); }}>
      <div className="flex-row-between mb-md">
        <h2 className="section-heading">ACTIVE ALERTS</h2>
        <span
          className={`alert-badge text-sm ${visible.length > 0 || degraded ? 'alert-badge-error' : 'alert-badge-success'}`}
          aria-label={degraded ? (visible.length ? `${visible.length} last known ${visible.length === 1 ? 'alert' : 'alerts'}; current count unavailable` : 'Current alert count unavailable') : `${visible.length} active ${visible.length === 1 ? 'alert' : 'alerts'}`}
        >
          {degraded && visible.length === 0 ? '—' : visible.length}
        </span>
      </div>

      {viewPaused ? <div className="text-muted text-sm mb-sm" role="status">Monitoring view updates are paused. Collection and alert evaluation may continue.</div> : null}
      {degraded && visible.length > 0 ? (
        <div className="text-muted text-sm mb-sm" role="status">
          {feed?.status === 'error' ? `Alert state could not be refreshed (${feed.error}); showing the last known alerts.` : 'Alert evaluation is not current; showing the last known alerts.'} These conditions are not revalidated.
        </div>
      ) : null}

      {feed?.status === 'error' && onRefresh ? <button type="button" className="alert-refresh" onClick={onRefresh}>Refresh evaluation status</button> : null}
      <div className="alert-list">
        {visible.length === 0 ? (
          <div className="text-center text-muted p-lg text-lg" role="status">
            {emptyMessage(feed)}
            {feed?.status === 'ready' && feed.evaluatedAt ? <div className="text-sm">Evaluated {new Date(feed.evaluatedAt).toLocaleString()}</div> : null}
            {feed?.status === 'error' && feed.error ? <div className="text-sm">{feed.error}</div> : null}
          </div>
        ) : (
          visible.map(alert => (
            <div key={alert.id} className={`alert-item pool-item mb-sm${onInspect && inspectTarget(alert.category) ? ' alert-item--inspectable' : ''}`} data-severity={alert.severity.toLowerCase()} data-sm-style="sm-style-2f44229022">
              <span className={`alert-item__severity status-color-${alert.severity.toLowerCase()}`}>{alert.severity.toUpperCase()}</span>
              <div className="alert-item__content">
                <div><strong>{alert.category}</strong> · <span>{alert.message}</span></div>
                {alert.details ? <div className="text-muted text-sm">{alert.details}</div> : null}
                {feed?.ruleIssues.some(issue => issue.ruleId === alert.category) ? <div className="text-muted text-sm">This condition was not revalidated.</div> : null}
              </div>
              <div className="alert-item__time text-muted text-sm">
                <div>{alert.timestamp ? `Active since ${new Date(alert.timestamp).toLocaleString()}` : 'Activation time unavailable'}</div>
                {feed?.evaluatedAt ? <div>Evaluation at {new Date(feed.evaluatedAt).toLocaleString()}</div> : null}
              </div>
              {onInspect && inspectTarget(alert.category) ? (
                <button type="button" aria-label={`Inspect ${inspectTarget(alert.category)}`} data-alert-id={alert.id} onClick={() => { const target = inspectTarget(alert.category); if (target) onInspect(target, alert, feed?.status === 'ready' && !feed.ruleIssues.some(issue => issue.ruleId === alert.category)); }}>
                  Inspect {inspectTarget(alert.category)}
                </button>
              ) : null}
              {!inspectTarget(alert.category) && onInspectOther ? <button type="button" onClick={onInspectOther}>View investigations</button> : null}
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
