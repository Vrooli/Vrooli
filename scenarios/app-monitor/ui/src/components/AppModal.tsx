import { useState, CSSProperties } from 'react';
import clsx from 'clsx';
import type { App, AppProxyMetadata, AppProxyPortInfo, LocalhostUsageReport } from '@/types';
import './AppModal.css';

interface AppModalProps {
  app: App;
  isOpen: boolean;
  onClose: () => void;
  onAction: (appId: string, action: 'start' | 'stop' | 'restart') => Promise<void>;
  onViewLogs: (appId: string) => void;
  proxyMetadata?: AppProxyMetadata | null;
  localhostReport?: LocalhostUsageReport | null;
}

export default function AppModal({ app, isOpen, onClose, onAction, onViewLogs, proxyMetadata, localhostReport }: AppModalProps) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  if (!isOpen) return null;

  const portEntries = Object.entries(app.port_mappings || {});
  const portsMap = Object.fromEntries(portEntries);
  const primaryFallback = portEntries[0] ? portEntries[0][0] : 'PORT';
  const hasUIPort = portsMap['UI_PORT'] !== undefined;
  const primaryPortLabel = ((app.config?.primary_port_label as string | undefined) || (hasUIPort ? 'UI_PORT' : primaryFallback)).toUpperCase();
  const primaryPortValue = (() => {
    if (app.config?.primary_port) return String(app.config.primary_port);
    if (hasUIPort) return String(portsMap['UI_PORT']);
    if (portEntries[0]) return String(portEntries[0][1]);
    return 'N/A';
  })();

  const apiPort = portsMap['API_PORT'] !== undefined ? String(portsMap['API_PORT']) : null;
  const otherPorts = portEntries
    .filter(([label]) => label !== 'UI_PORT' && label !== 'API_PORT')
    .map(([label, value]) => ({ label: label.toUpperCase(), value: String(value) }));

  const proxyRoutes: AppProxyPortInfo[] = proxyMetadata?.ports
    ? [...proxyMetadata.ports].sort((a, b) => {
        if (a.isPrimary === b.isPrimary) {
          const aLabel = (a.label || a.slug || '').toLowerCase();
          const bLabel = (b.label || b.slug || '').toLowerCase();
          return aLabel.localeCompare(bLabel);
        }
        return a.isPrimary ? -1 : 1;
      })
    : [];
  const localhostFindings = localhostReport?.findings ?? [];
  const localhostWarnings = localhostReport?.warnings ?? [];
  const showLocalhostDiagnostics = Boolean(localhostReport);

  const uptime = app.uptime && app.uptime !== 'N/A' ? app.uptime : 'N/A';
  const runtime = app.runtime && app.runtime !== 'N/A' && app.runtime !== uptime ? app.runtime : null;
  const typeLabel = app.type ? app.type.toUpperCase() : 'SCENARIO';
  const isRunning = ['running', 'healthy', 'degraded', 'unhealthy'].includes(app.status);
  const isStopped = app.status === 'stopped';

  const compactValueStyle: CSSProperties = {
    fontSize: '0.8rem',
    letterSpacing: '0.07em',
  };

  const compactLabelStyle: CSSProperties = {
    fontSize: '0.58rem',
    letterSpacing: '0.08em',
  };

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
                {app.health_status && app.health_status !== app.status && (
                  <span className="detail-substatus">{app.health_status.toUpperCase()}</span>
                )}
              </span>
            </div>
            
            <div className="detail-row">
              <span className="detail-label" style={compactLabelStyle}>{primaryPortLabel}:</span>
              <span className="detail-value port-number" style={compactValueStyle}>
                {primaryPortValue}
              </span>
            </div>

            {apiPort && (
              <div className="detail-row">
                <span className="detail-label" style={compactLabelStyle}>API_PORT:</span>
                <span className="detail-value port-number" style={compactValueStyle}>{apiPort}</span>
              </div>
            )}

            <div className="detail-row">
              <span className="detail-label" style={compactLabelStyle}>TYPE:</span>
              <span className="detail-value" style={compactValueStyle}>
                {typeLabel}
              </span>
            </div>

            <div className="detail-row">
              <span className="detail-label" style={compactLabelStyle}>UPTIME:</span>
              <span className="detail-value" style={compactValueStyle}>
                {uptime}
              </span>
            </div>

            {runtime && (
              <div className="detail-row">
                <span className="detail-label" style={compactLabelStyle}>RUNTIME:</span>
                <span className="detail-value" style={compactValueStyle}>{runtime}</span>
              </div>
            )}

            {otherPorts.length > 0 && (
              <div className="detail-row full-width">
                <span className="detail-label">PORTS:</span>
                <div className="detail-tags">
                  {otherPorts.map((port) => (
                    <span className="tag-chip" key={port.label}>
                      {port.label}: {port.value}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {proxyRoutes.length > 0 && (
              <div className="detail-row full-width">
                <span className="detail-label">PROXY ROUTES:</span>
                <div className="detail-tags">
                  {proxyRoutes.map((route) => {
                    const routeLabel = (route.label || route.slug || `PORT ${route.port}`).toUpperCase();
                    return (
                      <span className="tag-chip" key={`${route.path}-${routeLabel}`}>
                        {routeLabel}: {route.path}
                      </span>
                    );
                  })}
                </div>
              </div>
            )}

            {showLocalhostDiagnostics && (
              <div className="detail-row full-width">
                <span className="detail-label">LOCALHOST:</span>
                <div className="detail-description">
                  {localhostFindings.length === 0 ? (
                    <span>No hard-coded localhost references detected.</span>
                  ) : (
                    <ul>
                      {localhostFindings.slice(0, 5).map((finding) => (
                        <li key={`${finding.file_path}:${finding.line}`}>
                          <code>{finding.file_path}:{finding.line}</code>
                          {' '}
                          {finding.pattern ? `(${finding.pattern}) ` : ''}
                          {finding.snippet}
                        </li>
                      ))}
                      {localhostFindings.length > 5 && (
                        <li>…and {localhostFindings.length - 5} more occurrences</li>
                      )}
                    </ul>
                  )}
                  {localhostWarnings.length > 0 && (
                    <ul>
                      {localhostWarnings.map((warning, index) => (
                        <li key={`localhost-warning-${index}`}>{warning}</li>
                      ))}
                    </ul>
                  )}
                </div>
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

            {app.tags && app.tags.length > 0 && (
              <div className="detail-row full-width">
                <span className="detail-label">TAGS:</span>
                <div className="detail-tags">
                  {app.tags.map((tag) => (
                    <span className="tag-chip" key={tag}>{tag}</span>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
        
        <div className="modal-footer">
          <button
            className={clsx('modal-btn', { loading: actionLoading === 'start' })}
            onClick={() => handleAction('start')}
            disabled={isRunning || actionLoading !== null}
          >
            {actionLoading === 'start' ? 'STARTING...' : 'START'}
          </button>
          <button
            className={clsx('modal-btn', { loading: actionLoading === 'stop' })}
            onClick={() => handleAction('stop')}
            disabled={isStopped || actionLoading !== null}
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
