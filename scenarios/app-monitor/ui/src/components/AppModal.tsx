import { useState } from 'react';
import clsx from 'clsx';
import type { App } from '@/types';
import './AppModal.css';

interface AppModalProps {
  app: App;
  isOpen: boolean;
  onClose: () => void;
  onAction: (appId: string, action: 'start' | 'stop' | 'restart') => Promise<void>;
  onViewLogs: (appId: string) => void;
}

export default function AppModal({ app, isOpen, onClose, onAction, onViewLogs }: AppModalProps) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleAction = async (action: 'start' | 'stop' | 'restart') => {
    setActionLoading(action);
    try {
      await onAction(app.id, action);
    } finally {
      setActionLoading(null);
    }
  };

  const handleViewLogs = () => {
    onViewLogs(app.id);
  };

  return (
    <div className={clsx('modal', { active: isOpen })} onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{app.name}</h2>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>
        
        <div className="modal-body">
          <div className="app-details">
            <div className="detail-row">
              <span className="detail-label">STATUS:</span>
              <span className={clsx('detail-value', `status-${app.status}`)}>
                {app.status.toUpperCase()}
              </span>
            </div>
            
            <div className="detail-row">
              <span className="detail-label">PORT:</span>
              <span className="detail-value port-number">
                {app.port || 'N/A'}
              </span>
            </div>
            
            <div className="detail-row">
              <span className="detail-label">TYPE:</span>
              <span className="detail-value">
                {app.type?.toUpperCase() || 'UNKNOWN'}
              </span>
            </div>
            
            <div className="detail-row">
              <span className="detail-label">UPTIME:</span>
              <span className="detail-value">
                {app.uptime || '00:00:00'}
              </span>
            </div>
            
            <div className="detail-row">
              <span className="detail-label">CPU USAGE:</span>
              <span className="detail-value">
                {app.cpu ? `${app.cpu.toFixed(1)}%` : '0%'}
              </span>
            </div>
            
            <div className="detail-row">
              <span className="detail-label">MEMORY USAGE:</span>
              <span className="detail-value">
                {app.memory ? `${app.memory.toFixed(1)}%` : '0%'}
              </span>
            </div>
            
            {app.lastActivity && (
              <div className="detail-row">
                <span className="detail-label">LAST ACTIVITY:</span>
                <span className="detail-value">
                  {app.lastActivity}
                </span>
              </div>
            )}
            
            {app.description && (
              <div className="detail-row full-width">
                <span className="detail-label">DESCRIPTION:</span>
                <p className="detail-description">
                  {app.description}
                </p>
              </div>
            )}
          </div>
        </div>
        
        <div className="modal-footer">
          <button
            className={clsx('modal-btn', { loading: actionLoading === 'start' })}
            onClick={() => handleAction('start')}
            disabled={app.status === 'running' || actionLoading !== null}
          >
            {actionLoading === 'start' ? 'STARTING...' : 'START'}
          </button>
          <button
            className={clsx('modal-btn', { loading: actionLoading === 'stop' })}
            onClick={() => handleAction('stop')}
            disabled={app.status === 'stopped' || actionLoading !== null}
          >
            {actionLoading === 'stop' ? 'STOPPING...' : 'STOP'}
          </button>
          <button
            className={clsx('modal-btn', { loading: actionLoading === 'restart' })}
            onClick={() => handleAction('restart')}
            disabled={actionLoading !== null}
          >
            {actionLoading === 'restart' ? 'RESTARTING...' : 'RESTART'}
          </button>
          <button
            className="modal-btn"
            onClick={handleViewLogs}
          >
            VIEW LOGS
          </button>
        </div>
      </div>
    </div>
  );
}