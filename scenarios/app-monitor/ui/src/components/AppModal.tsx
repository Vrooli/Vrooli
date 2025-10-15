import { useMemo, useState } from 'react';
import clsx from 'clsx';
import { ExternalLink, Play, RotateCcw, ScrollText, Square } from 'lucide-react';
import type { App, AppProxyMetadata, AppProxyPortInfo, LocalhostUsageReport } from '@/types';
import { buildPreviewUrl } from '@/utils/appPreview';
import './AppModal.css';

interface AppModalProps {
  app: App;
  isOpen: boolean;
  onClose: () => void;
  onAction: (appId: string, action: 'start' | 'stop' | 'restart') => Promise<void>;
  onViewLogs: (appId: string) => void;
  proxyMetadata?: AppProxyMetadata | null;
  localhostReport?: LocalhostUsageReport | null;
  previewUrl?: string | null;
}

export default function AppModal({
  app,
  isOpen,
  onClose,
  onAction,
  onViewLogs,
  proxyMetadata,
  localhostReport,
  previewUrl,
}: AppModalProps) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const fallbackPreviewUrl = useMemo(() => buildPreviewUrl(app) ?? null, [app]);

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
  const localhostFindingsCount = localhostFindings.length;
  const showLocalhostDiagnostics = Boolean(localhostReport);

  const uptime = app.uptime && app.uptime !== 'N/A' ? app.uptime : 'N/A';
  const runtime = app.runtime && app.runtime !== 'N/A' && app.runtime !== uptime ? app.runtime : null;
  const typeLabel = app.type ? app.type.toUpperCase() : 'SCENARIO';
  const isRunning = ['running', 'healthy', 'degraded', 'unhealthy'].includes(app.status);
  const isStopped = app.status === 'stopped';

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

  const normalizedStatus = (app.status || 'unknown').toLowerCase();
  const displayName = app.name || app.scenario_name || app.id;
  const subtitleChips = [app.scenario_name && app.scenario_name !== displayName ? app.scenario_name : null, app.id]
    .filter(Boolean) as string[];
  const currentUrl = previewUrl ?? fallbackPreviewUrl ?? (app.port_mappings?.UI_PORT ? `http://localhost:${app.port_mappings.UI_PORT}` : null);

  const handleOpenPreview = () => {
    if (!currentUrl || typeof window === 'undefined') {
      return;
    }
    window.open(currentUrl, '_blank', 'noopener,noreferrer');
  };
  return (
    <div className={clsx('modal', { active: isOpen })} onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div className="modal-header__titles">
            <h2>{displayName}</h2>
            {subtitleChips.length > 0 && (
              <div className="modal-header__meta">
                {subtitleChips.map((chip) => (
                  <span className="modal-header__chip" key={chip}>{chip}</span>
                ))}
              </div>
            )}
            {currentUrl && (
              <div className="modal-header__url" title={currentUrl}>
                <span className="modal-header__url-label">Preview URL</span>
                <span className="modal-header__url-value">{currentUrl}</span>
              </div>
            )}
          </div>
          <button className="modal-close" onClick={onClose} aria-label="Close">
            ×
          </button>
        </div>

        <div className="modal-body">
          <section className="detail-section">
            <h3 className="detail-section__title">Overview</h3>
            <div className="detail-grid">
              <div className="detail-card detail-card--status">
                <span className="detail-card__label">Status</span>
                <span className={clsx('detail-card__value', 'detail-card__value--status', `status-${normalizedStatus}`)}>
                  {app.status ? app.status.toUpperCase() : 'UNKNOWN'}
                  {app.health_status && app.health_status !== app.status && (
                    <span className="detail-card__subvalue">{app.health_status.toUpperCase()}</span>
                  )}
                </span>
              </div>

              <div className="detail-card detail-card--mono">
                <span className="detail-card__label">{primaryPortLabel}</span>
                <span className="detail-card__value detail-card__value--mono">{primaryPortValue}</span>
              </div>

              {apiPort && (
                <div className="detail-card detail-card--mono">
                  <span className="detail-card__label">API Port</span>
                  <span className="detail-card__value detail-card__value--mono">{apiPort}</span>
                </div>
              )}

              <div className="detail-card">
                <span className="detail-card__label">Type</span>
                <span className="detail-card__value">{typeLabel}</span>
              </div>

              <div className="detail-card">
                <span className="detail-card__label">Uptime</span>
                <span className="detail-card__value">{uptime}</span>
              </div>

              {runtime && (
                <div className="detail-card">
                  <span className="detail-card__label">Runtime</span>
                  <span className="detail-card__value">{runtime}</span>
                </div>
              )}
            </div>
          </section>

          {(otherPorts.length > 0 || proxyRoutes.length > 0) && (
            <section className="detail-section">
              <h3 className="detail-section__title">Ports & Routes</h3>
              {otherPorts.length > 0 && (
                <div className="detail-grid">
                  {otherPorts.map((port) => (
                    <div className="detail-card detail-card--mono" key={port.label}>
                      <span className="detail-card__label">{port.label}</span>
                      <span className="detail-card__value detail-card__value--mono">{port.value}</span>
                    </div>
                  ))}
                </div>
              )}

              {proxyRoutes.length > 0 && (
                <div className="detail-panel detail-panel--tags">
                  <div className="tag-cloud" role="list">
                    {proxyRoutes.map((route) => {
                      const routeLabel = (route.label || route.slug || `PORT ${route.port}`).toUpperCase();
                      return (
                        <span className="tag-chip" role="listitem" key={`${route.path}-${routeLabel}`}>
                          <span className="tag-chip__label">{routeLabel}</span>
                          <span className="tag-chip__value">{route.path}</span>
                        </span>
                      );
                    })}
                  </div>
                </div>
              )}
            </section>
          )}

          {showLocalhostDiagnostics && (
            <section className="detail-section">
              <h3 className="detail-section__title">Localhost Diagnostics</h3>
              <div className="detail-panel detail-panel--list">
                {localhostFindingsCount > 0 && (
                  <div className="detail-panel__alert detail-panel__alert--warning">
                    <span className="detail-panel__alert-title">
                      {localhostFindingsCount} hard-coded localhost reference{localhostFindingsCount === 1 ? '' : 's'} detected
                    </span>
                    <p className="detail-panel__alert-message">
                      Update requests to use the scenario proxy base so the preview remains accessible from other hosts.
                    </p>
                  </div>
                )}
                {localhostFindingsCount === 0 ? (
                  <p className="detail-panel__text">No hard-coded localhost references detected.</p>
                ) : (
                  <ul className="detail-list">
                    {localhostFindings.slice(0, 6).map((finding) => (
                      <li key={`${finding.file_path}:${finding.line}`}>
                        <code>{finding.file_path}:{finding.line}</code>
                        {finding.pattern ? ` (${finding.pattern})` : ''}
                        <span className="detail-list__snippet">{finding.snippet}</span>
                      </li>
                    ))}
                    {localhostFindings.length > 6 && (
                      <li>…and {localhostFindings.length - 6} more occurrences</li>
                    )}
                  </ul>
                )}
                {localhostWarnings.length > 0 && (
                  <ul className="detail-list detail-list--warnings">
                    {localhostWarnings.map((warning, index) => (
                      <li key={`localhost-warning-${index}`}>{warning}</li>
                    ))}
                  </ul>
                )}
              </div>
            </section>
          )}

          {app.description && (
            <section className="detail-section">
              <h3 className="detail-section__title">Description</h3>
              <div className="detail-panel">
                <p className="detail-panel__text">{app.description}</p>
              </div>
            </section>
          )}

          {app.tags && app.tags.length > 0 && (
            <section className="detail-section">
              <h3 className="detail-section__title">Tags</h3>
              <div className="tag-cloud" role="list">
                {app.tags.map((tag) => (
                  <span className="tag-chip" role="listitem" key={tag}>{tag}</span>
                ))}
              </div>
            </section>
          )}
        </div>

        <div className="modal-footer">
          {!isRunning && (
            <button
              className={clsx('modal-btn', 'modal-btn--accent', { loading: actionLoading === 'start' })}
              onClick={() => handleAction('start')}
              disabled={actionLoading !== null}
            >
              <Play aria-hidden size={16} />
              {actionLoading === 'start' ? 'Starting…' : 'Start'}
            </button>
          )}
          {!isStopped && (
            <button
              className={clsx('modal-btn', 'modal-btn--danger', { loading: actionLoading === 'stop' })}
              onClick={() => handleAction('stop')}
              disabled={actionLoading !== null}
            >
              <Square aria-hidden size={16} />
              {actionLoading === 'stop' ? 'Stopping…' : 'Stop'}
            </button>
          )}
          <button
            className={clsx('modal-btn', 'modal-btn--neutral', { loading: actionLoading === 'restart' })}
            onClick={() => handleAction('restart')}
            disabled={actionLoading !== null}
          >
            <RotateCcw aria-hidden size={16} />
            {actionLoading === 'restart' ? 'Restarting…' : 'Restart'}
          </button>
          <button
            className={clsx('modal-btn', 'modal-btn--ghost')}
            onClick={handleViewLogs}
          >
            <ScrollText aria-hidden size={16} />
            View Logs
          </button>
          {currentUrl && (
            <button
              className={clsx('modal-btn', 'modal-btn--ghost')}
              onClick={handleOpenPreview}
            >
              <ExternalLink aria-hidden size={16} />
              Open Preview
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
