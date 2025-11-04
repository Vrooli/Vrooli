import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import clsx from 'clsx';
import { Copy, ExternalLink, Play, RotateCcw, ScrollText, Square, FileText, Activity, Layers, AlertCircle, CheckCircle, Loader } from 'lucide-react';
import ResponsiveDialog from '@/components/dialog/ResponsiveDialog';
import type { App, AppProxyMetadata, AppProxyPortInfo, LocalhostUsageReport, CompleteDiagnostics } from '@/types';
import { buildPreviewUrl } from '@/utils/appPreview';
import { useAppDiagnostics } from '@/hooks/useAppDiagnostics';
import TechStackTab from './tabs/TechStackTab';
import DiagnosticsTab from './tabs/DiagnosticsTab';
import DocumentationTab from './tabs/DocumentationTab';
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

type OtherPort = { label: string; value: string };

const FOCUSABLE_SELECTORS =
  'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])';

type TabType = 'overview' | 'diagnostics' | 'tech-stack' | 'docs';

export default function AppModal({
  app,
  isOpen,
  onClose,
  onAction,
  onViewLogs,
  proxyMetadata,
  previewUrl,
}: AppModalProps) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [hasCopiedPreviewUrl, setHasCopiedPreviewUrl] = useState(false);
  const [activeTab, setActiveTab] = useState<TabType>('overview');
  const fallbackPreviewUrl = useMemo(() => buildPreviewUrl(app) ?? null, [app]);
  const modalContentRef = useRef<HTMLDivElement | null>(null);
  const closeButtonRef = useRef<HTMLButtonElement | null>(null);
  const previouslyFocusedElementRef = useRef<HTMLElement | null>(null);
  const titleId = useId();
  const descriptionId = useId();

  // Fetch complete diagnostics when modal opens
  const { diagnostics, loading: diagnosticsLoading } = useAppDiagnostics(
    isOpen ? app.id : null,
    { enabled: isOpen, refetchOnOpen: true }
  );

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    previouslyFocusedElementRef.current = document.activeElement as HTMLElement | null;

    const focusTimer = window.setTimeout(() => {
      closeButtonRef.current?.focus();
    }, 0);

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose();
        return;
      }

      if (event.key !== 'Tab') {
        return;
      }

      const container = modalContentRef.current;
      if (!container) return;

      const focusable = Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTORS)).filter(
        (el) => !el.hasAttribute('disabled') && el.tabIndex !== -1 && el.getAttribute('aria-hidden') !== 'true',
      );

      if (focusable.length === 0) {
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const activeElement = document.activeElement as HTMLElement | null;

      if (!event.shiftKey && activeElement === last) {
        event.preventDefault();
        first.focus();
      } else if (event.shiftKey && activeElement === first) {
        event.preventDefault();
        last.focus();
      }
    };

    const container = modalContentRef.current;
    container?.addEventListener('keydown', handleKeyDown);

    return () => {
      window.clearTimeout(focusTimer);
      container?.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen, onClose]);

  useEffect(() => {
    if (!isOpen) {
      previouslyFocusedElementRef.current?.focus();
      previouslyFocusedElementRef.current = null;
      setHasCopiedPreviewUrl(false);
    }
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [isOpen]);

  useEffect(() => {
    setActionLoading(null);
  }, [app.id]);

  useEffect(() => {
    if (!hasCopiedPreviewUrl) {
      return;
    }

    const timer = window.setTimeout(() => setHasCopiedPreviewUrl(false), 2500);
    return () => window.clearTimeout(timer);
  }, [hasCopiedPreviewUrl]);

  const { apiPort, otherPorts, primaryPortLabel, primaryPortValue, proxyRoutes } = useMemo(() => {
    const portEntries = Object.entries(app.port_mappings || {});
    const portsMap = Object.fromEntries(portEntries);
    const hasUIPort = portsMap['UI_PORT'] !== undefined;

    // Determine primary port label and value
    const resolvedPrimaryLabel = (
      (app.config?.primary_port_label as string | undefined) || (hasUIPort ? 'UI_PORT' : 'PORT')
    ).toUpperCase();

    const resolvedPrimaryValue = (() => {
      if (app.config?.primary_port) return String(app.config.primary_port);
      if (hasUIPort) return String(portsMap['UI_PORT']);
      return 'N/A';
    })();

    const resolvedApiPort = portsMap['API_PORT'] !== undefined ? String(portsMap['API_PORT']) : null;

    // Filter out ports that are already shown as primary or API port
    const shownPorts = new Set<string>();
    if (hasUIPort) shownPorts.add('UI_PORT');
    if (resolvedApiPort) shownPorts.add('API_PORT');

    const resolvedOtherPorts = portEntries
      .filter(([label]) => !shownPorts.has(label))
      .map(([label, value]) => ({ label: label.toUpperCase(), value: String(value) }));

    // Debug logging
    if (typeof window !== 'undefined' && (window as any).__DEBUG_APP_MODAL_PORTS) {
      console.log('[AppModal] Port computation:', {
        appId: app.id,
        portEntries,
        hasUIPort,
        resolvedPrimaryLabel,
        resolvedPrimaryValue,
        resolvedApiPort,
        shownPorts: Array.from(shownPorts),
        resolvedOtherPorts,
      });
    }

    const resolvedProxyRoutes: AppProxyPortInfo[] = proxyMetadata?.ports
      ? [...proxyMetadata.ports].sort((a, b) => {
          if (a.isPrimary === b.isPrimary) {
            const aLabel = (a.label || a.slug || '').toLowerCase();
            const bLabel = (b.label || b.slug || '').toLowerCase();
            return aLabel.localeCompare(bLabel);
          }
          return a.isPrimary ? -1 : 1;
        })
      : [];

    return {
      apiPort: resolvedApiPort,
      otherPorts: resolvedOtherPorts,
      primaryPortLabel: resolvedPrimaryLabel,
      primaryPortValue: resolvedPrimaryValue,
      proxyRoutes: resolvedProxyRoutes,
    };
  }, [app.config?.primary_port, app.config?.primary_port_label, app.port_mappings, proxyMetadata?.ports]);


  const uptime = app.uptime && app.uptime !== 'N/A' ? app.uptime : 'N/A';
  const runtime = app.runtime && app.runtime !== 'N/A' && app.runtime !== uptime ? app.runtime : null;
  const typeLabel = app.type ? app.type.toUpperCase() : 'SCENARIO';
  const isRunning = ['running', 'healthy', 'degraded', 'unhealthy'].includes(app.status);
  const isStopped = app.status === 'stopped';

  const handleAction = useCallback(
    async (action: 'start' | 'stop' | 'restart') => {
      setActionLoading(action);
      try {
        await onAction(app.id, action);
      } finally {
        setActionLoading(null);
      }
    },
    [app.id, onAction],
  );

  const handleViewLogs = useCallback(() => {
    onViewLogs(app.id);
  }, [app.id, onViewLogs]);

  const normalizedStatus = (app.status || 'unknown').toLowerCase();
  const displayName = app.name || app.scenario_name || app.id;
  const subtitleChips = [app.scenario_name && app.scenario_name !== displayName ? app.scenario_name : null, app.id]
    .filter(Boolean) as string[];
  const currentUrl = previewUrl ?? fallbackPreviewUrl ?? (app.port_mappings?.UI_PORT ? `http://localhost:${app.port_mappings.UI_PORT}` : null);

  const handleOpenPreview = useCallback(() => {
    if (!currentUrl || typeof window === 'undefined') {
      return;
    }
    // Keep Referer intact so the app-monitor proxy routes shared assets to the correct scenario.
    window.open(currentUrl, '_blank', 'noopener');
  }, [currentUrl]);

  const handleCopyPreviewUrl = useCallback(() => {
    if (!currentUrl || typeof navigator === 'undefined' || !navigator.clipboard) {
      return;
    }

    navigator.clipboard.writeText(currentUrl).then(() => {
      setHasCopiedPreviewUrl(true);
    });
  }, [currentUrl]);

  if (!isOpen) {
    return null;
  }

  return (
    <ResponsiveDialog
      isOpen
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      aria-describedby={descriptionId}
      className="modal-content app-modal"
      overlayClassName="app-modal__overlay"
      contentRef={modalContentRef}
    >
        <div className="modal-header">
          <div className="modal-header__titles">
            <h2 id={titleId}>{displayName}</h2>
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
                {/* Omit noreferrer so Referer persists for proxy asset routing. */}
                <a
                  className="modal-header__url-value"
                  href={currentUrl}
                  target="_blank"
                  rel="noopener"
                >
                  {currentUrl}
                </a>
                <button
                  type="button"
                  className={clsx('modal-header__url-action', { active: hasCopiedPreviewUrl })}
                  onClick={handleCopyPreviewUrl}
                  aria-label="Copy preview URL"
                >
                  <Copy size={14} aria-hidden />
                  <span className="modal-header__url-action-text">Copy</span>
                </button>
                {hasCopiedPreviewUrl && (
                  <span className="modal-header__url-feedback" role="status" aria-live="polite">
                    Copied
                  </span>
                )}
              </div>
            )}
          </div>
          <button
            type="button"
            className="modal-close"
            onClick={onClose}
            aria-label="Close application details"
            ref={closeButtonRef}
          >
            <span aria-hidden>×</span>
          </button>
        </div>

        <div className="modal-tabs">
          <button
            type="button"
            className={clsx('modal-tab', { 'modal-tab--active': activeTab === 'overview' })}
            onClick={() => setActiveTab('overview')}
            aria-label="Overview tab"
          >
            <Layers size={16} aria-hidden />
            <span>Overview</span>
          </button>
          <button
            type="button"
            className={clsx('modal-tab', { 'modal-tab--active': activeTab === 'diagnostics' })}
            onClick={() => setActiveTab('diagnostics')}
            aria-label="Diagnostics tab"
          >
            <Activity size={16} aria-hidden />
            <span>Diagnostics</span>
            {diagnostics && diagnostics.warnings.length > 0 && (
              <span className="modal-tab-badge">{diagnostics.warnings.length}</span>
            )}
          </button>
          <button
            type="button"
            className={clsx('modal-tab', { 'modal-tab--active': activeTab === 'tech-stack' })}
            onClick={() => setActiveTab('tech-stack')}
            aria-label="Tech stack tab"
          >
            <AlertCircle size={16} aria-hidden />
            <span>Tech Stack</span>
          </button>
          <button
            type="button"
            className={clsx('modal-tab', { 'modal-tab--active': activeTab === 'docs' })}
            onClick={() => setActiveTab('docs')}
            aria-label="Documentation tab"
          >
            <FileText size={16} aria-hidden />
            <span>Docs</span>
            {diagnostics?.documents && diagnostics.documents.total > 0 && (
              <span className="modal-tab-badge">{diagnostics.documents.total}</span>
            )}
          </button>
        </div>

        <div className="modal-body" id={descriptionId}>
          {activeTab === 'overview' && (
            <>
              <OverviewSection
                apiPort={apiPort}
                normalizedStatus={normalizedStatus}
                primaryPortLabel={primaryPortLabel}
                primaryPortValue={primaryPortValue}
                runtime={runtime}
                status={app.status}
                typeLabel={typeLabel}
                uptime={uptime}
              />

              {(otherPorts.length > 0 || proxyRoutes.length > 0) && (
                <PortsRoutesSection otherPorts={otherPorts} proxyRoutes={proxyRoutes} />
              )}

              <DiagnosticsAlertSection
                diagnostics={diagnostics}
                diagnosticsLoading={diagnosticsLoading}
                onOpenDiagnostics={() => setActiveTab('diagnostics')}
              />

              {app.description && <DescriptionSection description={app.description} />}

              {app.tags && app.tags.length > 0 && <TagsSection tags={app.tags} />}
            </>
          )}

          {activeTab === 'diagnostics' && (
            <DiagnosticsTab diagnostics={diagnostics} loading={diagnosticsLoading} />
          )}

          {activeTab === 'tech-stack' && (
            <TechStackTab app={app} techStack={diagnostics?.tech_stack} loading={diagnosticsLoading} />
          )}

          {activeTab === 'docs' && (
            <DocumentationTab appId={app.id} documents={diagnostics?.documents} loading={diagnosticsLoading} />
          )}
        </div>

        <div className="modal-footer">
          {!isRunning && (
            <button
              type="button"
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
              type="button"
              className={clsx('modal-btn', 'modal-btn--danger', { loading: actionLoading === 'stop' })}
              onClick={() => handleAction('stop')}
              disabled={actionLoading !== null}
            >
              <Square aria-hidden size={16} />
              {actionLoading === 'stop' ? 'Stopping…' : 'Stop'}
            </button>
          )}
          <button
            type="button"
            className={clsx('modal-btn', 'modal-btn--neutral', { loading: actionLoading === 'restart' })}
            onClick={() => handleAction('restart')}
            disabled={actionLoading !== null}
          >
            <RotateCcw aria-hidden size={16} />
            {actionLoading === 'restart' ? 'Restarting…' : 'Restart'}
          </button>
          <button
            type="button"
            className={clsx('modal-btn', 'modal-btn--ghost')}
            onClick={handleViewLogs}
            aria-label="View application logs"
          >
            <ScrollText aria-hidden size={16} />
            Logs
          </button>
          {currentUrl && (
            <button
              type="button"
              className={clsx('modal-btn', 'modal-btn--ghost')}
              onClick={handleOpenPreview}
              aria-label="Open application preview"
            >
              <ExternalLink aria-hidden size={16} />
              Open
            </button>
          )}
        </div>
    </ResponsiveDialog>
  );
}

interface OverviewSectionProps {
  status: App['status'];
  normalizedStatus: string;
  primaryPortLabel: string;
  primaryPortValue: string;
  apiPort: string | null;
  typeLabel: string;
  uptime: string;
  runtime: string | null;
}

function OverviewSection({
  status,
  normalizedStatus,
  primaryPortLabel,
  primaryPortValue,
  apiPort,
  typeLabel,
  uptime,
  runtime,
}: OverviewSectionProps) {
  return (
    <section className="detail-section">
      <h3 className="detail-section__title">Overview</h3>
      <div className="detail-grid">
        <div className="detail-card detail-card--status">
          <span className="detail-card__label">Status</span>
          <span className={clsx('detail-card__value', 'detail-card__value--status', `status-${normalizedStatus}`)}>
            {status ? status.toUpperCase() : 'UNKNOWN'}
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
          <span className="detail-card__value detail-card__value--primary">{typeLabel}</span>
        </div>

        <div className="detail-card">
          <span className="detail-card__label">Uptime</span>
          <span className="detail-card__value detail-card__value--primary">{uptime}</span>
        </div>

        {runtime && (
          <div className="detail-card">
            <span className="detail-card__label">Runtime</span>
            <span className="detail-card__value detail-card__value--primary">{runtime}</span>
          </div>
        )}
      </div>
    </section>
  );
}

interface PortsRoutesSectionProps {
  otherPorts: OtherPort[];
  proxyRoutes: AppProxyPortInfo[];
}

function PortsRoutesSection({ otherPorts, proxyRoutes }: PortsRoutesSectionProps) {
  return (
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
  );
}

interface DiagnosticsAlertSectionProps {
  diagnostics: CompleteDiagnostics | null;
  diagnosticsLoading: boolean;
  onOpenDiagnostics: () => void;
}

function DiagnosticsAlertSection({ diagnostics, diagnosticsLoading, onOpenDiagnostics }: DiagnosticsAlertSectionProps) {
  if (diagnosticsLoading) {
    return (
      <section className="detail-section">
        <h3 className="detail-section__title">Diagnostics</h3>
        <div className="detail-panel detail-panel--alert detail-panel__alert--loading">
          <div className="detail-panel__alert-icon">
            <Loader size={20} className="spinning" aria-hidden />
          </div>
          <div className="detail-panel__alert-content">
            <span className="detail-panel__alert-title">Loading diagnostics...</span>
            <p className="detail-panel__alert-message">
              Analyzing application health, configuration, and documentation.
            </p>
          </div>
        </div>
      </section>
    );
  }

  const warningCount = diagnostics?.warnings?.length ?? 0;
  const hasIssues = warningCount > 0;

  return (
    <section className="detail-section">
      <h3 className="detail-section__title">Diagnostics</h3>
      <button
        type="button"
        className={clsx('detail-panel detail-panel--alert detail-panel--clickable', {
          'detail-panel__alert--success': !hasIssues,
          'detail-panel__alert--warning': hasIssues,
        })}
        onClick={onOpenDiagnostics}
        aria-label={hasIssues ? `View ${warningCount} diagnostic warnings` : 'View diagnostics details'}
      >
        <div className="detail-panel__alert-icon">
          {hasIssues ? (
            <AlertCircle size={20} aria-hidden />
          ) : (
            <CheckCircle size={20} aria-hidden />
          )}
        </div>
        <div className="detail-panel__alert-content">
          {hasIssues ? (
            <>
              <span className="detail-panel__alert-title">
                {warningCount} diagnostic issue{warningCount === 1 ? '' : 's'} detected
              </span>
              <p className="detail-panel__alert-message">
                Click to view details in the Diagnostics tab.
              </p>
            </>
          ) : (
            <>
              <span className="detail-panel__alert-title">All diagnostics passed</span>
              <p className="detail-panel__alert-message">
                Click to view detailed health report.
              </p>
            </>
          )}
        </div>
      </button>
    </section>
  );
}

function DescriptionSection({ description }: { description: string }) {
  return (
    <section className="detail-section">
      <h3 className="detail-section__title">Description</h3>
      <div className="detail-panel">
        <p className="detail-panel__text">{description}</p>
      </div>
    </section>
  );
}

function TagsSection({ tags }: { tags: string[] }) {
  return (
    <section className="detail-section">
      <h3 className="detail-section__title">Tags</h3>
      <div className="tag-cloud" role="list">
        {tags.map((tag) => (
          <span className="tag-chip" role="listitem" key={tag}>{tag}</span>
        ))}
      </div>
    </section>
  );
}
