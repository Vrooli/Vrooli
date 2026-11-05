// DOC: docs/concepts/ARCHITECTURE.md#ui
import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { BrowserRouter, Link, Routes, Route, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { getProxyInfo } from '@vrooli/api-base';
import { Database, FileText, Network, MemoryStick } from 'lucide-react';
import { fetchMachines } from './features/machines/api';
import { Header } from './shared/components/Header';
import { MetricsGrid } from './features/metrics/components/MetricsGrid';
import type { ChartInspection } from './features/metrics/components/MetricSparkline';
import { DeviceGraphPanel } from './features/metrics/components/DeviceGraphPanel';
import { readMetricNavigationState } from './features/metrics/navigationState';
import { AlertPanel } from './shared/components/AlertPanel';
import { ErrorBoundary } from './shared/components/ErrorBoundary';
import { ToastProvider } from './shared/components/ToastProvider';
import { ToastContainer } from './shared/components/ToastContainer';
import { ConnectionStatusBanner } from './shared/components/ConnectionStatusBanner';
import { LoadingSkeleton } from './shared/components/LoadingSkeleton';
import { ThemeProvider } from './shared/theme/ThemeProvider';
import { useSystemMonitor } from './features/monitoring/hooks/useSystemMonitor';
import { useServerAlerts } from './features/monitoring/hooks/useServerAlerts';
import { useInvestigationAgents } from './features/investigations/hooks/useInvestigationAgents';
import { useScriptExecution } from './features/investigations/hooks/useScriptExecution';
import { IncidentTimeline, type TimelineEntry } from './features/monitoring/components/IncidentTimeline';
import { TimeRangeProvider, useTimeRange } from './shared/time/TimeRangeContext';
import { MachineIdentityStrip } from './features/machines/components/MachineIdentityStrip';
import { LocalOnlyPanels } from './features/machines/components/LocalOnlyPanels';
import { MachinePresenceNote } from './features/machines/components/MachinePresenceNote';
import type { Alert, DashboardState, CardType, PanelType, Machine } from './types';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import './styles/tokens.css';
import './styles/migrated-inline.css';
import './styles/readout.css';
import './styles/report.css';

// ── Lazy-loaded, off-initial-paint subtrees ─────────────────────────────────
//
// Two groups:
//
//  1. Route/modal subtrees that pull in the heavy charting (recharts, ~400 KB)
//     and syntax-highlighting (react-syntax-highlighter, ~500 KB+) libraries.
//     None render on the initial dashboard ("/") paint, so React.lazy keeps
//     that code out of the entry chunk and off the main thread until the route
//     opens or a modal/settings dialog is shown.
//
//  2. Below-the-fold dashboard sections (InfrastructureMonitor,
//     InvestigationsSection, ReportsPanel) and the hidden Terminal overlay.
//     The above-the-fold content is Header + MetricsGrid + AlertPanel; the rest
//     sits below the initial viewport. Lazy-loading them shrinks React's first
//     commit (smaller hydration tree → lower Total Blocking Time) — they mount
//     as their chunks arrive, with a lightweight skeleton in the meantime, and
//     render identically once present.
const InfrastructureMonitor = lazy(() => import('./features/monitoring/components/InfrastructureMonitor').then(m => ({ default: m.InfrastructureMonitor })));
const InvestigationsSection = lazy(() => import('./features/investigations/components/InvestigationsSection').then(m => ({ default: m.InvestigationsSection })));
const ReportsPanel = lazy(() => import('./features/reports/components/ReportsPanel').then(m => ({ default: m.ReportsPanel })));
const Terminal = lazy(() => import('./shared/components/Terminal').then(m => ({ default: m.Terminal })));
const CpuDetailView = lazy(() => import('./features/metrics/components/CpuDetailView').then(m => ({ default: m.CpuDetailView })));
const MemoryDetailView = lazy(() => import('./features/metrics/components/MemoryDetailView').then(m => ({ default: m.MemoryDetailView })));
const NetworkDetailView = lazy(() => import('./features/metrics/components/NetworkDetailView').then(m => ({ default: m.NetworkDetailView })));
const DiskDetailView = lazy(() => import('./features/metrics/components/DiskDetailView').then(m => ({ default: m.DiskDetailView })));
const GpuDetailView = lazy(() => import('./features/metrics/components/GpuDetailView').then(m => ({ default: m.GpuDetailView })));
const InvestigationScriptsPage = lazy(() => import('./features/investigations/pages/InvestigationScriptsPage').then(m => ({ default: m.InvestigationScriptsPage })));
const ForensicsPage = lazy(() => import('./features/forensics/pages/ForensicsPage').then(m => ({ default: m.ForensicsPage })));
const LogsPage = lazy(() => import('./features/logs/pages/LogsPage').then(m => ({ default: m.LogsPage })));
const CapacityPage = lazy(() => import('./features/capacity/pages/CapacityPage').then(m => ({ default: m.CapacityPage })));
const ModalsContainer = lazy(() => import('./features/investigations/modals/ModalsContainer').then(m => ({ default: m.ModalsContainer })));
const SystemSettingsModal = lazy(() => import('./features/settings/components/SystemSettingsModal').then(m => ({ default: m.SystemSettingsModal })));

// Suspense fallback shared by all lazy route/modal boundaries. The operational
// LoadingSkeleton spinner doubles as the chunk-load indicator.
const RouteFallback = () => <LoadingSkeleton variant="simple" />;

/**
 * Compute BrowserRouter basename from proxy context.
 *
 * When served through app-monitor at /apps/<name>/proxy/,
 * React Router needs the proxy path as basename so that
 * navigate("/page") resolves to /apps/<name>/proxy/page
 * instead of /page.
 *
 * Returns "" outside proxy context (localhost, tunnel).
 */
function getRouterBasename(): string {
  const proxyInfo = getProxyInfo();
  const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
  if (proxyPath) {
    return proxyPath.replace(/\/+$/, '');
  }
  return '';
}

// Deferred workspaces can change height after navigation. Keep the requested
// destination visible briefly, but never override a subsequent user action.
export function revealWorkspace(findTarget: () => HTMLElement | null, focusTarget = false): () => void {
  let frame = 0;
  let first = true;
  let stopped = false;
  const update = () => {
    frame = 0;
    const target = findTarget();
    if (!target) return;
    if (first && focusTarget) target.focus({ preventScroll: true });
    const top = target.getBoundingClientRect().top;
    const headerBottom = document.querySelector('header')?.getBoundingClientRect().bottom ?? 0;
    if (first || top < headerBottom || top > window.innerHeight - 48) target.scrollIntoView?.({ block: 'start' });
    first = false;
  };
  const schedule = () => { if (!stopped && !frame) frame = window.requestAnimationFrame(update); };
  const mutations = new MutationObserver(schedule);
  mutations.observe(document.body, { childList: true, subtree: true });
  const resize = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(schedule);
  resize?.observe(document.body);
  const stop = () => {
    if (stopped) return;
    stopped = true;
    mutations.disconnect();
    resize?.disconnect();
    if (frame) window.cancelAnimationFrame(frame);
    window.clearTimeout(timeout);
    for (const event of ['wheel', 'touchstart', 'pointerdown', 'keydown']) window.removeEventListener(event, stop);
  };
  for (const event of ['wheel', 'touchstart', 'pointerdown', 'keydown']) window.addEventListener(event, stop, { once: true, passive: true });
  const timeout = window.setTimeout(stop, 5000);
  schedule();
  return stop;
}

// This commits only after its sibling lazy panel has loaded, not when its data is ready.
function WorkspaceMounted({ id, onMounted }: { id: string; onMounted: (id: string) => void }) {
  useEffect(() => { onMounted(id); }, [id, onMounted]);
  return null;
}

export function WorkspaceFailure({ name, reload = () => { window.location.reload(); } }: { name: string; reload?: () => void }) {
  return (
    <div className="card" role="alert">
      <p>{name} failed to load. Reload the page to try again; unsaved changes may be lost.</p>
      <button type="button" className="btn btn-action" onClick={reload}>Reload page</button>
    </div>
  );
}

function AppContent() {
  const navigate = useNavigate();
  const location = useLocation();
  const navigationState = readMetricNavigationState(location.state);
  const overviewReturn = useRef<{ subject: string; scrollY: number; focusLabel?: string; focusAlertId?: string } | null>(null);
  const workspaceReveal = useRef<(() => void) | null>(null);
  const pendingWorkspace = useRef<string | null>(null);
  const [workspaceAnnouncement, setWorkspaceAnnouncement] = useState('');
  const workspaceMounted = useCallback((id: string) => {
    if (pendingWorkspace.current === id) {
      pendingWorkspace.current = null;
      setWorkspaceAnnouncement(id === 'infrastructure-workspace'
        ? 'Infrastructure panel loaded. Data may still be loading.'
        : 'Investigations and reports panel loaded. Data may still be loading.');
    }
  }, []);
  useEffect(() => () => { workspaceReveal.current?.(); }, []);
  const [showDeferredDashboard, setShowDeferredDashboard] = useState(false);
  const [dashboardState, setDashboardState] = useState<DashboardState>({
    lastUpdate: new Date().toISOString(),
    expandedCards: new Set(),
    expandedPanels: new Set(),
    terminalVisible: false,
    unreadErrorCount: 0
  });

  const [systemSettingsModalOpen, setSystemSettingsModalOpen] = useState(false);
  const [machines, setMachines] = useState<Machine[]>([]);
  const [machineDiscovery, setMachineDiscovery] = useState<'loading' | 'ready' | 'error'>('loading');
  const [machineDiscoveryAttempt, setMachineDiscoveryAttempt] = useState(0);

  // The machine lives in the URL, not in component state. It is the subject of
  // everything on screen, so it has to survive a reload, be reachable by Back,
  // and be shareable: "look at what minimouse is doing" should be a link.
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedMachineID = searchParams.get('machine') ?? '';
  useEffect(() => {
    if (location.pathname !== '/' || selectedMachineID) {
      pendingWorkspace.current = null;
      setWorkspaceAnnouncement('');
      workspaceReveal.current?.();
    }
  }, [location.pathname, selectedMachineID]);
  const chartInspections = useMemo(() => new Map<string, ChartInspection>(), [selectedMachineID]);
  useEffect(() => {
    if (location.pathname !== '/' || !overviewReturn.current) return;
    const context = overviewReturn.current;
    overviewReturn.current = null;
    if (context.subject !== selectedMachineID) return;
    const frame = window.requestAnimationFrame(() => {
      window.scrollTo({ top: context.scrollY, behavior: 'instant' });
      if (context.focusLabel) {
        const controls = Array.from(document.querySelectorAll<HTMLElement>('main button, main a'));
        const control = context.focusAlertId ? controls.find(item => item.dataset.alertId === context.focusAlertId) : controls.find(item => item.textContent?.trim() === context.focusLabel);
        control?.focus({ preventScroll: true });
      }
    });
    return () => { window.cancelAnimationFrame(frame); };
  }, [location.pathname, selectedMachineID]);
  useEffect(() => {
    const section = navigationState.overviewSection;
    if (!section || !location.pathname.startsWith('/metrics/')) return;
    return revealWorkspace(() => Array.from(document.querySelectorAll<HTMLElement>('[data-detail-section]')).find(element => element.dataset.detailSection === section) ?? null);
  }, [location.pathname, navigationState.overviewSection]);
  const setSelectedMachineID = useCallback((machineID: string) => {
    setSearchParams(previous => {
      const next = new URLSearchParams(previous);
      if (machineID) {
        next.set('machine', machineID);
      } else {
        next.delete('machine');
      }
      return next;
    });
  }, [setSearchParams]);

  const { range, paused } = useTimeRange();
  const selectedMachine = machines.find(machine => machine.id === selectedMachineID);
  const viewingRemote = Boolean(selectedMachineID);
  const unknownSubject = viewingRemote && !selectedMachine;
  // Alerts are evaluated and persisted by the API; the dashboard only reads
  // them. They describe the local host, so they are not fetched for a remote
  // subject.
  const serverAlerts = useServerAlerts(!viewingRemote && !paused);

  const openAddMachine = () => {
    // Bridge owns pairing approval and the pending-request words. Keep this
    // entry in the machine control so an operator never has to leave the
    // monitoring context to begin linking another machine.
    window.open('/apps/vrooli-bridge/proxy/', '_blank', 'noopener,noreferrer');
  };

  useEffect(() => {
    let mounted = true;
    setMachineDiscovery('loading');
    void fetchMachines().then(next => {
      if (mounted) { setMachines(next ?? []); setMachineDiscovery('ready'); }
    }).catch(() => {
      if (mounted) setMachineDiscovery('error');
      // The requested subject remains visible when discovery fails.
    });
    return () => { mounted = false; };
  }, [machineDiscoveryAttempt]);

  const {
    metrics,
    deviceGraph,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
    subsystemErrors,
    investigations,
    metricHistory,
    isLoading,
    error,
    healthStatus,
    healthError,
    isStale,
    retryAttempt,
    lastSuccessfulFetch,
    lastAttemptAt,
    retryIntervalSeconds,
    toggleMonitoring,
    refreshHealth,
    refresh
  } = useSystemMonitor(range.seconds, { enabled: !paused, node: selectedMachineID || undefined });
  useEffect(() => {
    if (!isLoading && !performance.getEntriesByName("app-interactive").length) performance.mark("app-interactive");
  }, [isLoading]);

  // While a remote subject is silent, every panel below is showing the same
  // frozen moment. Dimming the whole region says so once, instead of leaving
  // each card to look independently live.
  const readingsFrozen = viewingRemote && isStale;

  // A machine that cannot be dispatched to has no readings to render at all.
  // The identity strip states that and why; four empty metric cards under it
  // would read as measurements of zero.
  const noReadingsPossible = viewingRemote && (!selectedMachine || !selectedMachine.dispatchable);

  const {
    agents,
    isSpawningAgent,
    spawnAgentError,
    stoppingAgentIds,
    agentErrors,
    refreshAgents,
    spawnAgent,
    stopAgent
  } = useInvestigationAgents();

  const {
    modalState,
    openScriptEditor,
    closeScriptEditor,
    closeScriptResults,
    executeScript,
    saveScript
  } = useScriptExecution();

  const hasOpenScriptModal = modalState.scriptEditor.isOpen || modalState.scriptResults.isOpen;

  const openDetailPage = (cardType: CardType, overviewSection?: string, condition?: { alert: Alert; revalidated: boolean }) => {
    const active = document.activeElement;
    overviewReturn.current = { subject: selectedMachineID, scrollY: window.scrollY, focusLabel: active instanceof HTMLButtonElement || active instanceof HTMLAnchorElement ? active.textContent?.trim() : undefined, focusAlertId: active instanceof HTMLElement ? active.dataset.alertId : undefined };
    const inspection = chartInspections.get(cardType);
    const sample = inspection?.selected && inspection.frozen ? { selected: inspection.selected, zoom: inspection.zoom, subject: selectedMachineID, target: cardType } : undefined;
    void navigate({ pathname: `/metrics/${cardType}`, search: window.location.search }, { state: overviewSection || condition || sample ? { overviewSection, sample, condition: condition ? { ...condition, subject: selectedMachineID, target: cardType, windowStart: metricHistory?.windowStart, windowEnd: metricHistory?.windowEnd, evaluatedAt: serverAlerts.feed.evaluatedAt } : undefined } : null });
  };

  const handleBackToDashboard = () => {
    void navigate({ pathname: '/', search: window.location.search });
  };

  const localOnlyRoute = (name: string, content: ReactNode) => viewingRemote ? (
    <section className="card" role="status">
      <h2>{name} is available on this computer</h2>
      <p>The selected machine {selectedMachine?.name ?? selectedMachineID} does not provide this workspace through Vega. Local evidence is not shown under its name.</p>
      <button type="button" className="btn btn-action" onClick={() => { setSelectedMachineID(''); }}>View on this computer</button>
      <button type="button" className="btn btn-action" onClick={handleBackToDashboard}>Back to Overview</button>
    </section>
  ) : content;

  const handleOpenIncidentSource = (source: 'logs' | 'forensics') => {
    void navigate(`/${source}`);
  };

  const handleInvestigateIncident = (entry: TimelineEntry) => {
    void navigate('/scripts', { state: { incidentId: entry.id, since: range.since, until: range.until } });
  };

  // Update online status based on successful API calls
  useEffect(() => {
    setDashboardState(prev => ({
      ...prev,
      lastUpdate: new Date().toISOString()
    }));
  }, [isLoading, error]);

  useEffect(() => {
    const timeoutID = window.setTimeout(() => {
      setShowDeferredDashboard(true);
    }, 2400);
    return () => {
      window.clearTimeout(timeoutID);
    };
  }, []);

  const toggleCard = (cardType: CardType) => {
    setDashboardState(prev => {
      const newExpandedCards = new Set(prev.expandedCards);
      if (newExpandedCards.has(cardType)) {
        newExpandedCards.delete(cardType);
      } else {
        newExpandedCards.add(cardType);
      }
      return {
        ...prev,
        expandedCards: newExpandedCards
      };
    });
  };

  const togglePanel = (panelType: PanelType) => {
    setDashboardState(prev => {
      const newExpandedPanels = new Set(prev.expandedPanels);
      if (newExpandedPanels.has(panelType)) {
        newExpandedPanels.delete(panelType);
      } else {
        newExpandedPanels.add(panelType);
      }
      return {
        ...prev,
        expandedPanels: newExpandedPanels
      };
    });
  };

  const toggleTerminal = () => {
    setDashboardState(prev => ({
      ...prev,
      terminalVisible: !prev.terminalVisible
    }));
  };

  const handleError = (error: Error, errorInfo: ErrorInfo) => {
    // Log error details for monitoring/analytics
    console.error('App Error Boundary caught error:', {
      error: error.message,
      stack: error.stack,
      componentStack: errorInfo.componentStack,
      timestamp: new Date().toISOString(),
      url: window.location.href,
      userAgent: navigator.userAgent
    });

    // TODO: Send error to logging service
    // Example: sendErrorToService(error, errorInfo);
  };

  return (
    <ErrorBoundary onError={handleError}>
      <div className="app">
        <Header
          unreadErrorCount={dashboardState.unreadErrorCount}
          agents={agents}
          onStopAgent={stopAgent}
          stoppingAgentIds={stoppingAgentIds}
          agentErrors={agentErrors}
          onRefreshAgents={() => { void refreshAgents(); }}
          onToggleTerminal={toggleTerminal}
          onOpenSettings={() => { setSystemSettingsModalOpen(true); }}
          healthStatus={healthStatus}
          healthError={healthError}
          onToggleMonitoring={toggleMonitoring}
          onRefreshHealth={refreshHealth}
          isLoadingHealth={isLoading}
          machines={machines}
          machineDiscovery={machineDiscovery}
          onRetryMachines={() => { setMachineDiscoveryAttempt(attempt => attempt + 1); }}
          selectedMachineID={selectedMachineID}
          onSelectMachine={setSelectedMachineID}
          onAddMachine={openAddMachine}
          terminalDisabledReason={selectedMachineID ? 'System output is local to this computer; remote terminal actions are not granted by system-monitor.' : undefined}
        />

        {unknownSubject ? (
          <div className="machine-strip" role="status">
            <div className="machine-strip__bar">
              <span>{machineDiscovery === 'loading' ? <>Checking machine <b>{selectedMachineID}</b>. No readings are shown yet.</> : machineDiscovery === 'error' ? <>Machine list unavailable; cannot verify <b>{selectedMachineID}</b>. No readings are shown.</> : <>Machine <b>{selectedMachineID}</b> is not available in the current machine list. No readings are shown.</>}</span>
              <button type="button" className="machine-strip__button" onClick={() => { setSelectedMachineID(''); }}>Back to this computer</button>
            </div>
          </div>
        ) : viewingRemote && selectedMachine ? (
          <MachineIdentityStrip
            machine={selectedMachine}
            isStale={isStale}
            lastSuccessfulFetch={lastSuccessfulFetch}
            onRetry={refresh}
            onBackToLocal={() => { setSelectedMachineID(''); }}
          />
        ) : (
          <ConnectionStatusBanner
            isStale={isStale}
            lastSuccessfulFetch={lastSuccessfulFetch}
            onRefresh={refresh}
            retryIntervalSeconds={retryIntervalSeconds}
            retryAttempt={retryAttempt}
          />
        )}

        <main className="main-content">
          <div className="container" data-sm-style="sm-style-a8abd88c52">
            <Suspense fallback={<RouteFallback />}>
            {(() => {
              const origin = navigationState.condition;
              if (!origin || origin.subject !== selectedMachineID || location.pathname !== `/metrics/${origin.target}`) return null;
              const windowLabel = origin.windowStart && origin.windowEnd && Number.isFinite(Date.parse(origin.windowStart)) && Number.isFinite(Date.parse(origin.windowEnd))
                ? `Overview window at navigation: ${new Date(origin.windowStart).toLocaleString()} – ${new Date(origin.windowEnd).toLocaleString()}`
                : 'Overview window bounds unavailable at navigation';
              return <section className="card mb-lg" aria-label="Originating condition">
                <h2>Condition selected on Overview</h2>
                <details>
                  <summary>{origin.alert.severity.toUpperCase()} · {origin.alert.category} — View condition details</summary>
                  <p>{origin.alert.message}</p>
                  {origin.alert.details && <p>{origin.alert.details}</p>}
                  {origin.evaluatedAt && Number.isFinite(Date.parse(origin.evaluatedAt)) && <p>Last evaluation at navigation: {new Date(origin.evaluatedAt).toLocaleString()}</p>}
                  <p>{windowLabel}</p>
                </details>
                <p>{origin.alert.timestamp && Number.isFinite(Date.parse(origin.alert.timestamp)) ? `Active since ${new Date(origin.alert.timestamp).toLocaleString()}` : 'Activation time unavailable'}. {origin.revalidated ? 'Condition status at navigation; it may have changed.' : 'Last known condition; it was not revalidated.'}</p>
              </section>;
            })()}
            {(() => {
              const sample = navigationState.sample;
              if (!sample || sample.subject !== selectedMachineID || location.pathname !== `/metrics/${sample.target}`) return null;
              const unit = sample.target === 'disk' ? ' MB/s' : sample.target === 'network' ? ' connections' : '%';
              const value = sample.target === 'network' ? Math.round(sample.selected.raw).toString() : sample.selected.raw.toFixed(1);
              return <section className="card mb-lg" aria-label="Selected Overview sample">
                <h2 className="section-heading">Selected Overview sample</h2>
                <p>{value}{unit} at {new Date(sample.selected.timestamp).toLocaleString()}</p>
                {sample.zoom && <p>Overview zoom at navigation: {new Date(sample.zoom.start).toLocaleString()} – {new Date(sample.zoom.end).toLocaleString()}</p>}
                <p>Detail evidence may be newer and is not necessarily from this sample time.</p>
              </section>;
            })()}
            <Routes>
              <Route
                path="/"
                element={(
                  <>
                    <section className="overview-heading" aria-labelledby="overview-title">
                      <div>
                        <div className="overview-eyebrow">System overview</div>
                        <h2 id="overview-title">{selectedMachine?.name ?? (selectedMachineID || 'This machine')}</h2>
                        <div className="overview-subtitle">{metricHistory?.windowSeconds && metricHistory.windowSeconds !== range.seconds ? `Requested ${range.label} · Showing previous ${metricHistory.windowSeconds / 60}-minute history` : range.label} · {metricHistory?.windowStart && metricHistory?.windowEnd ? `${new Date(metricHistory.windowStart).toLocaleString()} – ${new Date(metricHistory.windowEnd).toLocaleString()} (${Intl.DateTimeFormat().resolvedOptions().timeZone})` : 'Window bounds unavailable'} · {paused ? 'View paused' : 'Following updates'}</div>
                      </div>
                      <div className="overview-states">
                        <span>{viewingRemote ? 'Local alert evaluation not shown' : serverAlerts.feed.status === 'ready' ? `${serverAlerts.feed.ruleIssues.length ? 'Evaluation limited' : 'Evaluation ready'}${serverAlerts.feed.evaluatedAt ? ` · ${new Date(serverAlerts.feed.evaluatedAt).toLocaleString()}` : ''}` : `Evaluation ${serverAlerts.feed.status}`}</span>
                        <span>{healthStatus?.processor_active === true ? 'Collector active' : healthStatus?.processor_active === false ? 'Collector inactive' : 'Collector state unknown'}{viewingRemote ? ' on this computer' : ''}</span>
                      </div>
                    </section>
                    {viewingRemote && selectedMachine ? (
                      <section className="mb-lg">
                        <MachinePresenceNote
                          machine={selectedMachine}
                          isStale={isStale}
                          lastSuccessfulFetch={lastSuccessfulFetch}
                          retryAttempt={retryAttempt}
                          retryIntervalSeconds={retryIntervalSeconds}
                          lastAttemptAt={lastAttemptAt}
                        />
                      </section>
                    ) : null}

                    {viewingRemote && selectedMachine && subsystemErrors?.metrics && (
                      <section className="card mb-lg" role="status">
                        <p>Readings from {selectedMachine.name} could not be refreshed: {subsystemErrors.metrics.error}. {lastSuccessfulFetch ? 'The last received reading remains visible.' : 'No reading has been received for this machine.'}</p>
                        <button type="button" className="btn btn-action" onClick={refresh}>Retry readings</button>
                      </section>
                    )}

                    {!viewingRemote && (
                      <section className="mb-lg">
                        <AlertPanel alerts={serverAlerts.alerts} feed={serverAlerts.feed} onInspect={(card, alert, revalidated) => { openDetailPage(card, undefined, { alert, revalidated }); }} onRefresh={() => { void serverAlerts.refresh?.(); }} viewPaused={paused} onInspectOther={() => {
                          setShowDeferredDashboard(true);
                          workspaceReveal.current?.();
                          workspaceReveal.current = revealWorkspace(() => document.getElementById('investigations-workspace'));
                        }} />
                      </section>
                    )}

                    {!noReadingsPossible && (
                    <div className={readingsFrozen ? 'readings-frozen' : undefined}>
                    {/* Real-time Metrics Grid */}
                    <section className="mb-lg">
                      {!viewingRemote && metricHistory && (metricHistory.missingTimestampCount ?? 0) > 0 && (
                        <p className="overview-subtitle" role="status">{metricHistory.missingTimestampCount ?? 0} timeline {metricHistory.missingTimestampCount === 1 ? 'cycle was' : 'cycles were'} omitted from plots because {metricHistory.missingTimestampCount === 1 ? 'its timestamp was' : 'their timestamps were'} unavailable.</p>
                      )}
                      {!viewingRemote && subsystemErrors?.history && (
                        <div role="alert" className="card">
                          <p>History could not be refreshed: {subsystemErrors.history.error}. {metricHistory?.windowStart ? 'The previous history remains visible.' : 'No history has been received.'}</p>
                          <button type="button" className="btn btn-action" onClick={refresh}>Retry history</button>
                        </div>
                      )}
                      <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Metrics failed to render. Try refreshing the page.</div>}>
                        <MetricsGrid
                          key={selectedMachineID || 'local'}
                          inspectionStore={chartInspections}
                          metrics={metrics}
                          detailedMetrics={detailedMetrics}
                          expandedCards={dashboardState.expandedCards}
                          onToggleCard={toggleCard}
                          metricHistory={metricHistory}
                          storageIO={infrastructureData?.storageIo}
                          infrastructureUnavailable={Boolean(subsystemErrors?.infrastructure)}
                          diskLastUpdated={infrastructureData?.storageIo?.readRate?.observedAt ? timestampDate(infrastructureData.storageIo.readRate.observedAt).toISOString() : undefined}
                          onOpenDetail={openDetailPage}
                          serverAlerts={viewingRemote ? [] : serverAlerts.alerts}
                          remoteSubject={viewingRemote}
                        />
                      </ErrorBoundary>
                    </section>

                    {!viewingRemote && (
                      <nav aria-label="Dashboard workspaces" className="overview-workspaces">
                        <Link aria-label="Processes and ownership" aria-description="CPU and process activity" to={{ pathname: '/metrics/cpu', search: window.location.search }} onClick={event => { if (event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey) { event.preventDefault(); openDetailPage('cpu', 'process-monitor'); } }}><Network aria-hidden="true" /><span>Processes and ownership<small>CPU and process activity</small></span></Link>
                        <Link aria-label="Memory contributors" aria-description="Memory use by process" to={{ pathname: '/metrics/memory', search: window.location.search }} onClick={event => { if (event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey) { event.preventDefault(); openDetailPage('memory', 'memory-consumers'); } }}><MemoryStick aria-hidden="true" /><span>Memory contributors<small>Memory use by process</small></span></Link>
                        <a aria-label="Infrastructure" aria-description="Devices, pools and services" href="#infrastructure-workspace" onClick={event => { const target = document.getElementById('infrastructure-workspace'); pendingWorkspace.current = event.detail === 0 && (!target || target.querySelector('[aria-busy="true"]')) ? 'infrastructure-workspace' : null; setWorkspaceAnnouncement(''); setShowDeferredDashboard(true); workspaceReveal.current?.(); workspaceReveal.current = revealWorkspace(() => document.getElementById('infrastructure-workspace'), event.detail === 0); }}><Database aria-hidden="true" /><span>Infrastructure<small>Devices, pools and services</small></span></a>
                        <a aria-label="Investigations and reports" aria-description="Investigations and historical reports" href="#investigations-workspace" onClick={event => { const target = document.getElementById('investigations-workspace'); pendingWorkspace.current = event.detail === 0 && (!target || target.querySelector('[aria-busy="true"]')) ? 'investigations-workspace' : null; setWorkspaceAnnouncement(''); setShowDeferredDashboard(true); workspaceReveal.current?.(); workspaceReveal.current = revealWorkspace(() => document.getElementById('investigations-workspace'), event.detail === 0); }}><FileText aria-hidden="true" /><span>Investigations and reports<small>Investigations and historical reports</small></span></a>
                        <span className="sr-only" role="status" aria-atomic="true">{workspaceAnnouncement}</span>
                      </nav>
                    )}

                    <section className="mb-lg">
                      <details className="device-graph-disclosure">
                        <summary>
                          <span>Device graph</span>
                          <span className="device-graph-disclosure__status" aria-live="polite" aria-atomic="true">
                            {deviceGraph?.available
                              ? `${deviceGraph.devices.length} devices · ${deviceGraph.subsystems.length} subsystems${subsystemErrors?.deviceGraph ? ` · Last received; refresh failed: ${subsystemErrors.deviceGraph.error}` : ''}`
                              : deviceGraph ? `Unavailable: ${deviceGraph.unavailableReason || 'host did not provide a graph'}${subsystemErrors?.deviceGraph ? ` · Refresh failed: ${subsystemErrors.deviceGraph.error}` : ''}`
                                : subsystemErrors?.deviceGraph ? `Unavailable: ${subsystemErrors.deviceGraph.error}`
                                  : error ? 'Device graph not received' : 'Waiting for device graph'}
                          </span>
                        </summary>
                        <ErrorBoundary fallback={<div className="card">Device graph failed to render.</div>}>
                          <DeviceGraphPanel graph={deviceGraph} error={subsystemErrors?.deviceGraph?.error} />
                        </ErrorBoundary>
                      </details>
                    </section>

                    {/* Stated only where readings are actually possible: a
                        machine that cannot be dispatched to does not report
                        vitals at all, and the presence note already says so. */}
                    {viewingRemote && !noReadingsPossible ? (
                      <section className="mb-lg">
                        <LocalOnlyPanels machineName={selectedMachine?.name ?? 'this machine'} />
                      </section>
                    ) : null}
                    </div>
                    )}

                    {viewingRemote ? null : (
                      <section className="mb-lg">
                        <IncidentTimeline
                          history={metricHistory}
                          investigations={investigations}
                          onOpenSource={handleOpenIncidentSource}
                          onInvestigate={handleInvestigateIncident}
                        />
                      </section>
                    )}


                    {showDeferredDashboard && !selectedMachineID ? (
                      <>
                        {/* Infrastructure Monitor Panel */}
                        <section id="infrastructure-workspace" className="mb-lg" tabIndex={-1} aria-label="Infrastructure workspace">
                          <ErrorBoundary fallback={<WorkspaceFailure name="Infrastructure monitor" />}>
                            <Suspense fallback={<LoadingSkeleton variant="card" count={1} />}>
                              <InfrastructureMonitor
                                data={infrastructureData}
                                isExpanded={dashboardState.expandedPanels.has('infrastructure')}
                                onToggle={() => { togglePanel('infrastructure'); }}
                                systemHealth={detailedMetrics?.systemDetails}
                              />
                              <WorkspaceMounted id="infrastructure-workspace" onMounted={workspaceMounted} />
                            </Suspense>
                          </ErrorBoundary>
                        </section>

                        {/* Investigations Section */}
                        <section id="investigations-workspace" className="mb-lg" tabIndex={-1} aria-label="Investigations and reports workspace">
                          <ErrorBoundary fallback={<WorkspaceFailure name="Investigations section" />}>
                            <Suspense fallback={<LoadingSkeleton variant="card" count={1} />}>
                              <InvestigationsSection
                                investigations={investigations}
                                onOpenScriptEditor={openScriptEditor}
                                onSpawnAgent={spawnAgent}
                                agents={agents}
                                isSpawningAgent={isSpawningAgent}
                                spawnAgentError={spawnAgentError}
                              />
                              <WorkspaceMounted id="investigations-workspace" onMounted={workspaceMounted} />
                            </Suspense>
                          </ErrorBoundary>
                        </section>

                        {/* Playback Reports */}
                        <section className="mb-lg">
                          <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Reports failed to render.</div>}>
                            <Suspense fallback={<LoadingSkeleton variant="card" count={1} />}>
                              <ReportsPanel />
                            </Suspense>
                          </ErrorBoundary>
                        </section>
                      </>
                    ) : null}
                  </>
                )}
              />

              <Route
                path="/scripts"
                element={localOnlyRoute('Scripts', (
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Scripts page failed to render.</div>}>
                    <InvestigationScriptsPage
                      onOpenScriptEditor={openScriptEditor}
                      onExecuteScript={executeScript}
                      onSaveScript={saveScript}
                    />
                  </ErrorBoundary>
                ))}
              />

              <Route
                path="/metrics/cpu"
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">CPU detail view failed to render.</div>}>
                    <CpuDetailView
                      metrics={metrics}
                      detailedMetrics={detailedMetrics}
                      processMonitorData={processMonitorData}
                      metricHistory={metricHistory}
                      onBack={handleBackToDashboard}
                    />
                  </ErrorBoundary>
                )}
              />
              <Route
                path="/metrics/memory"
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Memory detail view failed to render.</div>}>
                    <MemoryDetailView
                      metrics={metrics}
                      detailedMetrics={detailedMetrics}
                      metricHistory={metricHistory}
                      onBack={handleBackToDashboard}
                    />
                  </ErrorBoundary>
                )}
              />
              <Route
                path="/metrics/network"
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Network detail view failed to render.</div>}>
                    <NetworkDetailView
                      metrics={metrics}
                      detailedMetrics={detailedMetrics}
                      metricHistory={metricHistory}
                      onBack={handleBackToDashboard}
                    />
                  </ErrorBoundary>
                )}
              />
              <Route
                path="/metrics/gpu"
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">GPU detail view failed to render.</div>}>
                    <GpuDetailView
                      detailedMetrics={detailedMetrics}
                      metricHistory={metricHistory}
                      onBack={handleBackToDashboard}
                    />
                  </ErrorBoundary>
                )}
              />
              <Route
                path="/forensics"
                element={localOnlyRoute('Forensics', (
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Forensics page failed to render.</div>}>
                    <ForensicsPage />
                  </ErrorBoundary>
                ))}
              />
              <Route
                path="/logs"
                element={localOnlyRoute('Logs', (
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Logs page failed to render.</div>}>
                    <LogsPage />
                  </ErrorBoundary>
                ))}
              />
              <Route
                path="/capacity"
                element={localOnlyRoute('Capacity', (
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Capacity page failed to render.</div>}>
                    <CapacityPage />
                  </ErrorBoundary>
                ))}
              />
              <Route
                path="/metrics/disk"
                element={localOnlyRoute('Disk detail', (
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Disk detail view failed to render.</div>}>
                    <DiskDetailView
                      detailedMetrics={detailedMetrics}
                      storageIO={infrastructureData?.storageIo}
                      metricHistory={metricHistory}
                      diskLastUpdated={infrastructureData?.timestamp ? timestampDate(infrastructureData.timestamp)?.toISOString() : undefined}
                      onBack={handleBackToDashboard}
                    />
                  </ErrorBoundary>
                ))}
              />
            </Routes>
            </Suspense>

          </div>
        </main>

        {dashboardState.terminalVisible ? (
          <ErrorBoundary fallback={null}>
            <Suspense fallback={null}>
              <Terminal
                isVisible={dashboardState.terminalVisible}
                onClose={toggleTerminal}
              />
            </Suspense>
          </ErrorBoundary>
        ) : null}

        {hasOpenScriptModal && !viewingRemote ? (
          <ErrorBoundary fallback={null}>
            <Suspense fallback={null}>
              <ModalsContainer
                modalState={modalState}
                onCloseScriptEditor={closeScriptEditor}
                onCloseScriptResults={closeScriptResults}
                onExecuteScript={executeScript}
                onSaveScript={saveScript}
              />
            </Suspense>
          </ErrorBoundary>
        ) : null}

        {/* System Settings Modal */}
        {systemSettingsModalOpen ? (
          <ErrorBoundary fallback={null}>
            <Suspense fallback={null}>
              <SystemSettingsModal
                isOpen={systemSettingsModalOpen}
                onClose={() => { setSystemSettingsModalOpen(false); }}
              />
            </Suspense>
          </ErrorBoundary>
        ) : null}
      </div>
      <ToastContainer />
    </ErrorBoundary>
  );
}

export default function App() {
  const basename = getRouterBasename();

  return (
    <ThemeProvider>
      <ToastProvider>
        <TimeRangeProvider>
          <BrowserRouter basename={basename}>
            <AppContent />
          </BrowserRouter>
        </TimeRangeProvider>
      </ToastProvider>
    </ThemeProvider>
  );
}
