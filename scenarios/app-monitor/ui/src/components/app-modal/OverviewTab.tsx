import clsx from 'clsx';
import { AlertCircle, CheckCircle, Loader } from 'lucide-react';
import type { App, AppProxyPortInfo, CompleteDiagnostics } from '@/types';

type OtherPort = { label: string; value: string };

interface OverviewTabProps {
  app: App;
  normalizedStatus: string;
  primaryPortLabel: string;
  primaryPortValue: string;
  apiPort: string | null;
  typeLabel: string;
  uptime: string;
  runtime: string | null;
  otherPorts: OtherPort[];
  proxyRoutes: AppProxyPortInfo[];
  diagnostics: CompleteDiagnostics | null;
  diagnosticsLoading: boolean;
  onOpenDiagnostics: () => void;
}

/** Overview tab panel: status cards, ports/routes, diagnostics alert, description, and tags. */
export default function OverviewTab({
  app,
  normalizedStatus,
  primaryPortLabel,
  primaryPortValue,
  apiPort,
  typeLabel,
  uptime,
  runtime,
  otherPorts,
  proxyRoutes,
  diagnostics,
  diagnosticsLoading,
  onOpenDiagnostics,
}: OverviewTabProps) {
  return (
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
        onOpenDiagnostics={onOpenDiagnostics}
      />

      {app.description && <DescriptionSection description={app.description} />}

      {app.tags && app.tags.length > 0 && <TagsSection tags={app.tags} />}
    </>
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
