import { useEffect, useState } from 'react';
import type { ErrorInfo } from 'react';
import { Routes, Route, useNavigate } from 'react-router-dom';
import { Header } from './shared/components/Header';
import { MetricsGrid } from './features/metrics/components/MetricsGrid';
import { CpuDetailView, MemoryDetailView, NetworkDetailView, DiskDetailView, GpuDetailView } from './features/metrics/components/MetricDetailViews';
import { InfrastructureMonitor } from './features/monitoring/components/InfrastructureMonitor';
import { AlertPanel } from './shared/components/AlertPanel';
import { InvestigationsSection } from './features/investigations/components/InvestigationsSection';
import { ReportsPanel } from './features/reports/components/ReportsPanel';
import { Terminal } from './shared/components/Terminal';
import { ModalsContainer } from './features/investigations/modals/ModalsContainer';
import { SystemSettingsModal } from './features/settings/components/SystemSettingsModal';
import { MatrixBackground } from './shared/components/MatrixBackground';
import { ErrorBoundary } from './shared/components/ErrorBoundary';
import { useSystemMonitor } from './features/monitoring/hooks/useSystemMonitor';
import { useInvestigationAgents } from './features/investigations/hooks/useInvestigationAgents';
import { useScriptExecution } from './features/investigations/hooks/useScriptExecution';
import { InvestigationScriptsPage } from './features/investigations/pages/InvestigationScriptsPage';
import type { DashboardState, CardType, PanelType } from './types';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import './styles/matrix-theme.css';

function App() {
  const navigate = useNavigate();
  const [dashboardState, setDashboardState] = useState<DashboardState>({
    lastUpdate: new Date().toISOString(),
    expandedCards: new Set(),
    expandedPanels: new Set(),
    terminalVisible: false,
    unreadErrorCount: 0,
    alerts: []
  });

  const [systemSettingsModalOpen, setSystemSettingsModalOpen] = useState(false);

  const {
    metrics,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
    investigations,
    metricHistory,
    isLoading,
    error,
    healthStatus,
    healthError,
    toggleMonitoring,
    refreshHealth
  } = useSystemMonitor();

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

  const openDetailPage = (cardType: CardType) => {
    navigate(`/metrics/${cardType}`);
  };

  const handleBackToDashboard = () => {
    navigate('/');
  };

  // Update online status based on successful API calls
  useEffect(() => {
    setDashboardState(prev => ({
      ...prev,
      lastUpdate: new Date().toISOString()
    }));
  }, [isLoading, error]);

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
        <MatrixBackground />
        
        <Header
          unreadErrorCount={dashboardState.unreadErrorCount}
          agents={agents}
          onStopAgent={stopAgent}
          stoppingAgentIds={stoppingAgentIds}
          agentErrors={agentErrors}
          onRefreshAgents={refreshAgents}
          onToggleTerminal={toggleTerminal}
          onOpenSettings={() => setSystemSettingsModalOpen(true)}
          healthStatus={healthStatus}
          healthError={healthError}
          onToggleMonitoring={toggleMonitoring}
          onRefreshHealth={refreshHealth}
          isLoadingHealth={isLoading}
        />

        <main className="main-content">
          <div className="container" style={{ padding: '2rem', maxWidth: '1400px', margin: '0 auto' }}>
            <Routes>
              <Route
                path="/"
                element={(
                  <>
                    {/* Real-time Metrics Grid */}
                    <section className="mb-lg">
                      <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>Metrics failed to render. Try refreshing the page.</div>}>
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

                    {/* Infrastructure Monitor Panel */}
                    <section className="mb-lg">
                      <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>Infrastructure monitor failed to render.</div>}>
                        <InfrastructureMonitor
                          data={infrastructureData}
                          isExpanded={dashboardState.expandedPanels.has('infrastructure')}
                          onToggle={() => togglePanel('infrastructure')}
                          systemHealth={detailedMetrics?.systemDetails}
                        />
                      </ErrorBoundary>
                    </section>

                    {/* Alert Panel */}
                    <section className="mb-lg">
                      <AlertPanel alerts={dashboardState.alerts} />
                    </section>

                    {/* Investigations Section */}
                    <section className="mb-lg">
                      <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>Investigations section failed to render.</div>}>
                        <InvestigationsSection
                          investigations={investigations}
                          onOpenScriptEditor={openScriptEditor}
                          onSpawnAgent={spawnAgent}
                          agents={agents}
                          isSpawningAgent={isSpawningAgent}
                          spawnAgentError={spawnAgentError}
                        />
                      </ErrorBoundary>
                    </section>

                    {/* Playback Reports */}
                    <section className="mb-lg">
                      <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>Reports failed to render.</div>}>
                        <ReportsPanel />
                      </ErrorBoundary>
                    </section>
                  </>
                )}
              />

              <Route
                path="/scripts"
                element={(
                  <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>Scripts page failed to render.</div>}>
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
                  <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>CPU detail view failed to render.</div>}>
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
                  <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>Memory detail view failed to render.</div>}>
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
                  <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>Network detail view failed to render.</div>}>
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
                  <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>GPU detail view failed to render.</div>}>
                    <GpuDetailView
                      detailedMetrics={detailedMetrics}
                      metricHistory={metricHistory}
                      onBack={handleBackToDashboard}
                    />
                  </ErrorBoundary>
                )}
              />
              <Route
                path="/metrics/disk"
                element={(
                  <ErrorBoundary fallback={<div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)' }}>Disk detail view failed to render.</div>}>
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

          </div>
        </main>

        <ErrorBoundary fallback={null}>
          <Terminal
            isVisible={dashboardState.terminalVisible}
            onClose={toggleTerminal}
          />
        </ErrorBoundary>

        <ErrorBoundary fallback={null}>
          <ModalsContainer
            modalState={modalState}
            onCloseScriptEditor={closeScriptEditor}
            onCloseScriptResults={closeScriptResults}
            onExecuteScript={executeScript}
            onSaveScript={saveScript}
          />
        </ErrorBoundary>

        {/* System Settings Modal */}
        <ErrorBoundary fallback={null}>
          <SystemSettingsModal
            isOpen={systemSettingsModalOpen}
            onClose={() => setSystemSettingsModalOpen(false)}
          />
        </ErrorBoundary>
      </div>
    </ErrorBoundary>
  );
}

export default App;
