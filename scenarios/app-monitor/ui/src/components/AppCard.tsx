import { useState } from 'react';
import clsx from 'clsx';
import type { App } from '@/types';
import './AppCard.css';

interface AppCardProps {
  app: App;
  onClick: () => void;
  onAction: (appId: string, action: 'start' | 'stop' | 'restart') => Promise<void>;
}

export default function AppCard({ app, onClick, onAction }: AppCardProps) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const handleAction = async (action: 'start' | 'stop' | 'restart', e: React.MouseEvent) => {
    e.stopPropagation();
    setActionLoading(action);
    try {
      await onAction(app.id, action);
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <div className="app-card" onClick={onClick}>
      <div className="app-card-header">
        <div className="app-name">{app.name}</div>
        <div className={clsx('app-status', app.status)}>
          {app.status.toUpperCase()}
        </div>
      </div>

      <div className="app-metrics">
        <div className="app-metric">
          <div className="metric-label">PORT</div>
          <div className="metric-data port-number">
            {app.port_mappings && Object.keys(app.port_mappings).length > 0 
              ? Object.values(app.port_mappings)[0] 
              : 'N/A'}
          </div>
        </div>
        <div className="app-metric">
          <div className="metric-label">CPU</div>
          <div className="metric-data">
            {app.cpu ? `${app.cpu.toFixed(1)}%` : '0%'}
          </div>
        </div>
        <div className="app-metric">
          <div className="metric-label">MEMORY</div>
          <div className="metric-data">
            {app.memory ? `${app.memory.toFixed(1)}%` : '0%'}
          </div>
        </div>
        <div className="app-metric">
          <div className="metric-label">UPTIME</div>
          <div className="metric-data">
            {app.uptime || '00:00:00'}
          </div>
        </div>
      </div>

      <div className="app-actions" onClick={(e) => e.stopPropagation()}>
        <button
          className={clsx('app-btn', { loading: actionLoading === 'start' })}
          onClick={(e) => handleAction('start', e)}
          disabled={app.status === 'running' || actionLoading !== null}
        >
          {actionLoading === 'start' ? '...' : '▶'}
        </button>
        <button
          className={clsx('app-btn', { loading: actionLoading === 'stop' })}
          onClick={(e) => handleAction('stop', e)}
          disabled={app.status === 'stopped' || actionLoading !== null}
        >
          {actionLoading === 'stop' ? '...' : '■'}
        </button>
        <button
          className={clsx('app-btn', { loading: actionLoading === 'restart' })}
          onClick={(e) => handleAction('restart', e)}
          disabled={actionLoading !== null}
        >
          {actionLoading === 'restart' ? '...' : '⟲'}
        </button>
      </div>
    </div>
  );
}