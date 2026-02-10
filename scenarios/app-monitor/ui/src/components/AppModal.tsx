import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import ResponsiveDialog from '@/components/dialog/ResponsiveDialog';
import ErrorBoundary, { SectionErrorFallback } from '@/components/ErrorBoundary';
import type { App, AppProxyMetadata, AppProxyPortInfo, CompleteDiagnostics, CompletenessScore, LighthouseHistory } from '@/types';
import { buildPreviewUrl } from '@/utils/appPreview';
import AppModalHeader from './app-modal/AppModalHeader';
import AppModalTabs from './app-modal/AppModalTabs';
import type { TabType } from './app-modal/AppModalTabs';
import AppModalFooter from './app-modal/AppModalFooter';
import OverviewTab from './app-modal/OverviewTab';
import { useFocusTrap } from './app-modal/useFocusTrap';
import { useBodyScrollLock } from './app-modal/useBodyScrollLock';
import TechStackTab from './tabs/TechStackTab';
import DiagnosticsTab from './tabs/DiagnosticsTab';
import DocumentationTab from './tabs/DocumentationTab';
import LighthouseTab from './tabs/LighthouseTab';
import CompletenessTab from './tabs/CompletenessTab';
import './AppModal.css';

interface AppModalProps {
  app: App;
  isOpen: boolean;
  onClose: () => void;
  onAction: (appId: string, action: 'start' | 'stop' | 'restart') => Promise<void>;
  onViewLogs: (appId: string) => void;
  proxyMetadata?: AppProxyMetadata | null;
  previewUrl?: string | null;
  preloadedDiagnostics?: CompleteDiagnostics | null;
  diagnosticsLoading?: boolean;
  preloadedLighthouseHistory?: LighthouseHistory | null;
  lighthouseLoading?: boolean;
  lighthouseError?: string | null;
  onRefetchLighthouse?: () => Promise<void>;
  preloadedCompleteness?: CompletenessScore | null;
  completenessLoading?: boolean;
  diagnosticsError?: string | null;
  onRefetchDiagnostics?: () => Promise<void>;
  completenessError?: string | null;
  onRefetchCompleteness?: () => Promise<void>;
}

export default function AppModal({
  app,
  isOpen,
  onClose,
  onAction,
  onViewLogs,
  proxyMetadata,
  previewUrl,
  preloadedDiagnostics,
  diagnosticsLoading: externalDiagnosticsLoading,
  preloadedLighthouseHistory,
  lighthouseLoading: externalLighthouseLoading,
  lighthouseError: externalLighthouseError,
  onRefetchLighthouse,
  preloadedCompleteness,
  completenessLoading: externalCompletenessLoading,
  diagnosticsError: externalDiagnosticsError,
  onRefetchDiagnostics,
  completenessError: externalCompletenessError,
  onRefetchCompleteness,
}: AppModalProps) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [hasCopiedPreviewUrl, setHasCopiedPreviewUrl] = useState(false);
  const [activeTab, setActiveTab] = useState<TabType>('overview');
  const fallbackPreviewUrl = useMemo(() => buildPreviewUrl(app) ?? null, [app]);
  const modalContentRef = useRef<HTMLDivElement | null>(null);
  const closeButtonRef = useRef<HTMLButtonElement | null>(null);
  const titleId = useId();
  const descriptionId = useId();

  // Normalize external props
  const diagnostics = preloadedDiagnostics ?? null;
  const diagnosticsLoading = externalDiagnosticsLoading ?? false;
  const lighthouseHistory = preloadedLighthouseHistory ?? null;
  const lighthouseLoading = externalLighthouseLoading ?? false;
  const lighthouseError = externalLighthouseError ?? null;
  const refetchLighthouse = onRefetchLighthouse ?? (async () => {});
  const completeness = preloadedCompleteness ?? null;
  const completenessLoading = externalCompletenessLoading ?? false;
  const diagnosticsError = externalDiagnosticsError ?? null;
  const completenessError = externalCompletenessError ?? null;

  useFocusTrap(modalContentRef, closeButtonRef, isOpen, onClose);
  useBodyScrollLock(isOpen);

  useEffect(() => {
    if (!isOpen) {
      setHasCopiedPreviewUrl(false);
    }
  }, [isOpen]);

  useEffect(() => {
    setActionLoading(null);
  }, [app.id]);

  useEffect(() => {
    if (!hasCopiedPreviewUrl) return;
    const timer = window.setTimeout(() => setHasCopiedPreviewUrl(false), 2500);
    return () => window.clearTimeout(timer);
  }, [hasCopiedPreviewUrl]);

  const { apiPort, otherPorts, primaryPortLabel, primaryPortValue, proxyRoutes } = useMemo(() => {
    const portEntries = Object.entries(app.port_mappings || {});
    const portsMap = Object.fromEntries(portEntries);
    const hasUIPort = portsMap['UI_PORT'] !== undefined;

    const resolvedPrimaryLabel = (
      (app.config?.primary_port_label as string | undefined) || (hasUIPort ? 'UI_PORT' : 'PORT')
    ).toUpperCase();

    const resolvedPrimaryValue = (() => {
      if (app.config?.primary_port) return String(app.config.primary_port);
      if (hasUIPort) return String(portsMap['UI_PORT']);
      return 'N/A';
    })();

    const resolvedApiPort = portsMap['API_PORT'] !== undefined ? String(portsMap['API_PORT']) : null;

    const shownPorts = new Set<string>();
    if (hasUIPort) shownPorts.add('UI_PORT');
    if (resolvedApiPort) shownPorts.add('API_PORT');

    const resolvedOtherPorts = portEntries
      .filter(([label]) => !shownPorts.has(label))
      .map(([label, value]) => ({ label: label.toUpperCase(), value: String(value) }));

    const resolvedProxyRoutes: AppProxyPortInfo[] = Array.isArray(proxyMetadata?.ports)
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
  const normalizedStatus = (app.status || 'unknown').toLowerCase();
  const displayName = app.name || app.scenario_name || app.id;
  const subtitleChips = [app.scenario_name && app.scenario_name !== displayName ? app.scenario_name : null, app.id]
    .filter(Boolean) as string[];
  const uiPort = app.port_mappings?.UI_PORT;
  const portUrl = typeof uiPort === 'number' ? `http://localhost:${uiPort}` : null;
  const currentUrl = previewUrl ?? fallbackPreviewUrl ?? portUrl;

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

  const handleOpenPreview = useCallback(() => {
    if (!currentUrl || typeof window === 'undefined') return;
    window.open(currentUrl, '_blank', 'noopener');
  }, [currentUrl]);

  const handleCopyPreviewUrl = useCallback(() => {
    if (!currentUrl || typeof navigator === 'undefined' || !navigator.clipboard) return;
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
      <AppModalHeader
        titleId={titleId}
        displayName={displayName}
        subtitleChips={subtitleChips}
        currentUrl={currentUrl}
        hasCopiedPreviewUrl={hasCopiedPreviewUrl}
        onCopyPreviewUrl={handleCopyPreviewUrl}
        onClose={onClose}
        closeButtonRef={closeButtonRef}
      />

      <AppModalTabs
        activeTab={activeTab}
        onTabChange={setActiveTab}
        diagnostics={diagnostics}
      />

      <div className="modal-body" id={descriptionId}>
        <div
          className="modal-tab-panel"
          role="tabpanel"
          id="tabpanel-overview"
          aria-labelledby="tab-overview"
          hidden={activeTab !== 'overview'}
        >
          <OverviewTab
            app={app}
            normalizedStatus={normalizedStatus}
            primaryPortLabel={primaryPortLabel}
            primaryPortValue={primaryPortValue}
            apiPort={apiPort}
            typeLabel={typeLabel}
            uptime={uptime}
            runtime={runtime}
            otherPorts={otherPorts}
            proxyRoutes={proxyRoutes}
            diagnostics={diagnostics}
            diagnosticsLoading={diagnosticsLoading}
            onOpenDiagnostics={() => setActiveTab('diagnostics')}
          />
        </div>

        <div
          className="modal-tab-panel"
          role="tabpanel"
          id="tabpanel-diagnostics"
          aria-labelledby="tab-diagnostics"
          hidden={activeTab !== 'diagnostics'}
        >
          <ErrorBoundary fallback={SectionErrorFallback}>
            <DiagnosticsTab diagnostics={diagnostics} loading={diagnosticsLoading} error={diagnosticsError} onRetry={onRefetchDiagnostics} />
          </ErrorBoundary>
        </div>

        <div
          className="modal-tab-panel"
          role="tabpanel"
          id="tabpanel-tech-stack"
          aria-labelledby="tab-tech-stack"
          hidden={activeTab !== 'tech-stack'}
        >
          <ErrorBoundary fallback={SectionErrorFallback}>
            <TechStackTab app={app} techStack={diagnostics?.tech_stack} loading={diagnosticsLoading} error={diagnosticsError} onRetry={onRefetchDiagnostics} />
          </ErrorBoundary>
        </div>

        <div
          className="modal-tab-panel"
          role="tabpanel"
          id="tabpanel-docs"
          aria-labelledby="tab-docs"
          hidden={activeTab !== 'docs'}
        >
          <ErrorBoundary fallback={SectionErrorFallback}>
            <DocumentationTab appId={app.id} documents={diagnostics?.documents} loading={diagnosticsLoading} />
          </ErrorBoundary>
        </div>

        <div
          className="modal-tab-panel"
          role="tabpanel"
          id="tabpanel-lighthouse"
          aria-labelledby="tab-lighthouse"
          hidden={activeTab !== 'lighthouse'}
        >
          <ErrorBoundary fallback={SectionErrorFallback}>
            <LighthouseTab app={app} history={lighthouseHistory} loading={lighthouseLoading} error={lighthouseError} onRefetch={refetchLighthouse} />
          </ErrorBoundary>
        </div>

        <div
          className="modal-tab-panel"
          role="tabpanel"
          id="tabpanel-completeness"
          aria-labelledby="tab-completeness"
          hidden={activeTab !== 'completeness'}
        >
          <ErrorBoundary fallback={SectionErrorFallback}>
            <CompletenessTab completeness={completeness} loading={completenessLoading} error={completenessError} onRetry={onRefetchCompleteness} />
          </ErrorBoundary>
        </div>
      </div>

      <AppModalFooter
        isRunning={isRunning}
        isStopped={isStopped}
        actionLoading={actionLoading}
        currentUrl={currentUrl}
        onAction={handleAction}
        onViewLogs={handleViewLogs}
        onOpenPreview={handleOpenPreview}
      />
    </ResponsiveDialog>
  );
}
