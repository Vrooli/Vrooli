import clsx from 'clsx';
import { Activity, Power, RefreshCw } from 'lucide-react';
import { useSystemStatus } from '@/state/systemStatusStore';
import { useScenarioActions } from '@/hooks/useScenarioActions';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import './ActionsDialog.css';

type ActionsDialogProps = {
  isConnected: boolean;
};

const OFFLINE_STATES = new Set(['unhealthy', 'offline', 'critical']);

const formatDuration = (seconds: number): string => {
  const total = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) {
    return `${hours}h ${minutes.toString().padStart(2, '0')}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${secs.toString().padStart(2, '0')}s`;
  }
  return `${secs}s`;
};

export default function ActionsDialog({ isConnected }: ActionsDialogProps) {
  const { closeOverlay } = useOverlayRouter();
  const { status, uptimeSeconds, appCount, resourceCount, loading } = useSystemStatus();
  const { restartAll, stopAll, triggerHealthCheck } = useScenarioActions();

  const isSystemOnline = status ? !OFFLINE_STATES.has(status) : isConnected;
  const uptimeText = uptimeSeconds != null ? formatDuration(uptimeSeconds) : '—';

  return (
    <div className="actions-dialog">
      <header className="actions-dialog__header">
        <div>
          <h2>Command Center</h2>
          <p>Monitor status and fire global controls.</p>
        </div>
        <button
          type="button"
          className="actions-dialog__close"
          aria-label="Close actions dialog"
          onClick={() => closeOverlay()}
        >
          ×
        </button>
      </header>

      <section className="actions-dialog__status">
        <div className="actions-dialog__status-indicator">
          <span className={clsx('actions-dialog__dot', isSystemOnline ? 'online' : 'offline')} aria-hidden />
          <div>
            <strong>{isSystemOnline ? 'Online' : 'Offline'}</strong>
            <p>{isSystemOnline ? 'All systems are reachable.' : 'Connections disrupted.'}</p>
          </div>
        </div>
        <dl className="actions-dialog__metrics" aria-live="polite">
          <div>
            <dt>Scenarios</dt>
            <dd>{appCount}</dd>
          </div>
          <div>
            <dt>Resources</dt>
            <dd>{resourceCount}</dd>
          </div>
          <div>
            <dt>Uptime</dt>
            <dd>{uptimeText}</dd>
          </div>
        </dl>
      </section>

      <section className="actions-dialog__actions" aria-label="Quick actions">
        <button
          type="button"
          className="actions-dialog__action"
          onClick={restartAll}
          disabled={loading}
        >
          <RefreshCw size={18} aria-hidden />
          <div>
            <strong>Restart all apps</strong>
            <span>Gracefully restart every managed scenario.</span>
          </div>
        </button>
        <button
          type="button"
          className="actions-dialog__action"
          onClick={stopAll}
          disabled={loading}
        >
          <Power size={18} aria-hidden />
          <div>
            <strong>Stop all apps</strong>
            <span>Shut down active scenarios to free resources.</span>
          </div>
        </button>
        <button
          type="button"
          className="actions-dialog__action"
          onClick={triggerHealthCheck}
          disabled={loading}
        >
          <Activity size={18} aria-hidden />
          <div>
            <strong>Run health check</strong>
            <span>Request a fresh system health snapshot.</span>
          </div>
        </button>
      </section>
    </div>
  );
}
