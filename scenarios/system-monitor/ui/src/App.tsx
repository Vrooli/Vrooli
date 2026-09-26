// DOC: docs/concepts/ARCHITECTURE.md#ui
import { lazy, Suspense, useCallback, useEffect, useState } from 'react';
import type { ErrorInfo } from 'react';
import { BrowserRouter, Routes, Route, useNavigate, useSearchParams } from 'react-router-dom';
import { getProxyInfo } from '@vrooli/api-base';
import { apiFetch } from './shared/api/apiFetch';
import { Header } from './shared/components/Header';
import { MetricsGrid } from './features/metrics/components/MetricsGrid';
import { DeviceGraphPanel } from './features/metrics/components/DeviceGraphPanel';
import { AlertPanel } from './shared/components/AlertPanel';
import { ErrorBoundary } from './shared/components/ErrorBoundary';
import { ToastProvider } from './shared/components/ToastProvider';
import { ToastContainer } from './shared/components/ToastContainer';
import { ConnectionStatusBanner } from './shared/components/ConnectionStatusBanner';
import { LoadingSkeleton } from './shared/components/LoadingSkeleton';
import { ThemeProvider } from './shared/theme/ThemeProvider';
import { useSystemMonitor } from './features/monitoring/hooks/useSystemMonitor';
import { useInvestigationAgents } from './features/investigations/hooks/useInvestigationAgents';
import { useScriptExecution } from './features/investigations/hooks/useScriptExecution';
import { IncidentTimeline, type TimelineEntry } from './features/monitoring/components/IncidentTimeline';
import { TimeRangeProvider, useTimeRange } from './shared/time/TimeRangeContext';
import { MachineIdentityStrip } from './features/machines/components/MachineIdentityStrip';
import { LocalOnlyPanels } from './features/machines/components/LocalOnlyPanels';
import { MachinePresenceNote } from './features/machines/components/MachinePresenceNote';
import type { DashboardState, CardType, PanelType, Machine } from './types';
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

function AppContent() {
  const navigate = useNavigate();
  const [showDeferredDashboard, setShowDeferredDashboard] = useState(false);
  const [dashboardState, setDashboardState] = useState<DashboardState>({
    lastUpdate: new Date().toISOString(),
    expandedCards: new Set(),
    expandedPanels: new Set(),
    terminalVisible: false,
    unreadErrorCount: 0,
    alerts: []
  });

  const [systemSettingsModalOpen, setSystemSettingsModalOpen] = useState(false);
  const [machines, setMachines] = useState<Machine[]>([]);

  // The machine lives in the URL, not in component state. It is the subject of
  // everything on screen, so it has to survive a reload, be reachable by Back,
  // and be shareable: "look at what minimouse is doing" should be a link.
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedMachineID = searchParams.get('machine') ?? '';
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

  const openAddMachine = () => {
    // Bridge owns pairing approval and the pending-request words. Keep this
    // entry in the machine control so an operator never has to leave the
    // monitoring context to begin linking another machine.
    window.open('/apps/vrooli-bridge/proxy/', '_blank', 'noopener,noreferrer');
  };

  useEffect(() => {
    let mounted = true;
    void apiFetch<Machine[]>('/machines').then(next => {
      if (mounted) setMachines(next ?? []);
    }).catch(() => {
      // Machine selection is additive; the local dashboard remains usable if
      // Bridge discovery is unavailable.
    });
    return () => { mounted = false; };
  }, []);

  // A link can name a machine this installation no longer knows. Fall back to
  // this computer rather than polling a node id that resolves to nothing,
  // which would render as an outage on a machine that is perfectly fine.
  useEffect(() => {
    if (!selectedMachineID || machines.length === 0) return;
    if (!machines.some(machine => machine.id === selectedMachineID)) {
      setSelectedMachineID('');
    }
  }, [machines, selectedMachineID, setSelectedMachineID]);

  const {
    metrics,
    deviceGraph,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
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

  // While a remote subject is silent, every panel below is showing the same
  // frozen moment. Dimming the whole region says so once, instead of leaving
  // each card to look independently live.
  const readingsFrozen = viewingRemote && isStale;

  // A machine that cannot be dispatched to has no readings to render at all.
  // The identity strip states that and why; four empty metric cards under it
  // would read as measurements of zero.
  const noReadingsPossible = viewingRemote && Boolean(selectedMachine) && !selectedMachine?.dispatchable;

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

  const openDetailPage = (cardType: CardType) => {
    void navigate(`/metrics/${cardType}`);
  };

  const handleBackToDashboard = () => {
    void navigate('/');
  };

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
          selectedMachineID={selectedMachineID}
          onSelectMachine={setSelectedMachineID}
          onAddMachine={openAddMachine}
          terminalDisabledReason={selectedMachineID ? 'System output is local to this computer; remote terminal actions are not granted by system-monitor.' : undefined}
        />

        {viewingRemote && selectedMachine ? (
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
            <Routes>
              <Route
                path="/"
                element={(
                  <>
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

                    {!noReadingsPossible && (
                    <div className={readingsFrozen ? 'readings-frozen' : undefined}>
                    {/* Real-time Metrics Grid */}
                    <section className="mb-lg">
                      <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Metrics failed to render. Try refreshing the page.</div>}>
                        <MetricsGrid
                          metrics={metrics}
                          detailedMetrics={detailedMetrics}
                          expandedCards={dashboardState.expandedCards}
                          onToggleCard={toggleCard}
                          metricHistory={metricHistory}
                          storageIO={infrastructureData?.storageIo}
                          diskLastUpdated={infrastructureData?.timestamp ? timestampDate(infrastructureData.timestamp)?.toISOString() : undefined}
                          onOpenDetail={openDetailPage}
                        />
                      </ErrorBoundary>
                    </section>

                    <section className="mb-lg">
                      <ErrorBoundary fallback={<div className="card">Device graph failed to render.</div>}>
                        <DeviceGraphPanel graph={deviceGraph} error={error?.error} />
                      </ErrorBoundary>
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

                    {/* Alert Panel */}
                    {!viewingRemote && (
                      <section className="mb-lg">
                        <AlertPanel alerts={dashboardState.alerts} />
                      </section>
                    )}

                    {showDeferredDashboard && !selectedMachineID ? (
                      <>
                        {/* Infrastructure Monitor Panel */}
                        <section className="mb-lg">
                          <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Infrastructure monitor failed to render.</div>}>
                            <Suspense fallback={<LoadingSkeleton variant="card" count={1} />}>
                              <InfrastructureMonitor
                                data={infrastructureData}
                                isExpanded={dashboardState.expandedPanels.has('infrastructure')}
                                onToggle={() => { togglePanel('infrastructure'); }}
                                systemHealth={detailedMetrics?.systemDetails}
                              />
                            </Suspense>
                          </ErrorBoundary>
                        </section>

                        {/* Investigations Section */}
                        <section className="mb-lg">
                          <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Investigations section failed to render.</div>}>
                            <Suspense fallback={<LoadingSkeleton variant="card" count={1} />}>
                              <InvestigationsSection
                                investigations={investigations}
                                onOpenScriptEditor={openScriptEditor}
                                onSpawnAgent={spawnAgent}
                                agents={agents}
                                isSpawningAgent={isSpawningAgent}
                                spawnAgentError={spawnAgentError}
                              />
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
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Scripts page failed to render.</div>}>
                    <InvestigationScriptsPage
                      onOpenScriptEditor={openScriptEditor}
                      onExecuteScript={executeScript}
                      onSaveScript={saveScript}
                    />
                  </ErrorBoundary>
                )}
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
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Forensics page failed to render.</div>}>
                    <ForensicsPage />
                  </ErrorBoundary>
                )}
              />
              <Route
                path="/logs"
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Logs page failed to render.</div>}>
                    <LogsPage />
                  </ErrorBoundary>
                )}
              />
              <Route
                path="/capacity"
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Capacity page failed to render.</div>}>
                    <CapacityPage />
                  </ErrorBoundary>
                )}
              />
              <Route
                path="/metrics/disk"
                element={(
                  <ErrorBoundary fallback={<div className="card" data-sm-style="sm-style-1769fab70e">Disk detail view failed to render.</div>}>
                    <DiskDetailView
                      detailedMetrics={detailedMetrics}
                      storageIO={infrastructureData?.storageIo}
                      metricHistory={metricHistory}
                      diskLastUpdated={infrastructureData?.timestamp ? timestampDate(infrastructureData.timestamp)?.toISOString() : undefined}
                      onBack={handleBackToDashboard}
                    />
                  </ErrorBoundary>
                )}
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

        {hasOpenScriptModal ? (
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
