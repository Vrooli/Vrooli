import { useEffect, useState } from 'react';
import type { ErrorInfo } from 'react';
import { Routes, Route, useNavigate } from 'react-router-dom';
import { Header } from './components/common/Header';
import { StatusIndicator } from './components/common/StatusIndicator';
import { MetricsGrid } from './components/metrics/MetricsGrid';
import { CpuDetailView, MemoryDetailView, NetworkDetailView, DiskDetailView } from './components/metrics/MetricDetailViews';
import { InfrastructureMonitor } from './components/monitoring/InfrastructureMonitor';
import { AlertPanel } from './components/common/AlertPanel';
import { InvestigationsSection } from './components/investigations/InvestigationsSection';
import { ReportsPanel } from './components/reports/ReportsPanel';
import { Terminal } from './components/common/Terminal';
import { ModalsContainer } from './components/modals/ModalsContainer';
import { SystemSettingsModal } from './components/modals/SystemSettingsModal';
import { MatrixBackground } from './components/common/MatrixBackground';
import { ErrorBoundary } from './components/common/ErrorBoundary';
import { useSystemMonitor } from './hooks/useSystemMonitor';
import type { DashboardState, ModalState, InvestigationScript, ScriptExecution, CardType } from './types';
import './styles/matrix-theme.css';

function App() {
  const navigate = useNavigate();
  const [dashboardState, setDashboardState] = useState<DashboardState>({
    isOnline: false,
    lastUpdate: new Date().toISOString(),
    expandedCards: new Set(),
    expandedPanels: new Set(),
    terminalVisible: false,
    unreadErrorCount: 0,
    alerts: []
  });

  const [modalState, setModalState] = useState<ModalState>({
    reportModal: {
      isOpen: false,
      loading: false
    },
    scriptEditor: {
      isOpen: false,
      mode: 'view'
    },
    scriptResults: {
      isOpen: false
    }
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
    error
  } = useSystemMonitor();

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
      isOnline: !isLoading && !error,
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

  const togglePanel = (panelType: string) => {
    setDashboardState(prev => {
      const newExpandedPanels = new Set(prev.expandedPanels);
      if (newExpandedPanels.has(panelType as any)) {
        newExpandedPanels.delete(panelType as any);
      } else {
        newExpandedPanels.add(panelType as any);
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

  const refreshDashboard = () => {
    // Trigger refresh of all data
    window.location.reload();
  };

  // Modal handler functions
  const openScriptEditor = (script?: InvestigationScript, content?: string, mode: 'create' | 'edit' | 'view' = 'view') => {
    setModalState(prev => ({
      ...prev,
      scriptEditor: {
        isOpen: true,
        script,
        scriptContent: content,
        scriptId: script?.id,
        mode
      }
    }));
  };

  const closeScriptEditor = () => {
    setModalState(prev => ({
      ...prev,
      scriptEditor: {
        ...prev.scriptEditor,
        isOpen: false
      }
    }));
  };

  const closeScriptResults = () => {
    setModalState(prev => ({
      ...prev,
      scriptResults: {
        ...prev.scriptResults,
        isOpen: false
      }
    }));
  };

  const executeScript = async (scriptId: string, _content: string) => {
    try {
      const execution: ScriptExecution = {
        script_id: scriptId,
        execution_id: `exec-${Date.now()}`,
        status: 'running',
        started_at: new Date().toISOString()
      };

      setModalState(prev => ({
        ...prev,
        scriptResults: {
          isOpen: true,
          scriptId,
          executionId: execution.execution_id,
          execution
        }
      }));

      closeScriptEditor();

      const response = await fetch(`/api/investigations/scripts/${encodeURIComponent(scriptId)}/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({})
      });

      let data: any = null;
      try {
        data = await response.json();
      } catch (parseError) {
        // ignore parse errors when no JSON body is returned
      }

      const stdout = data?.stdout ?? data?.output ?? '';
      const stderr = data?.stderr ?? '';
      const exitCode = typeof data?.exit_code === 'number' ? data.exit_code : (response.ok ? 0 : 1);
      const timedOut = Boolean(data?.timed_out);
      const completedAt = typeof data?.completed_at === 'string' ? data.completed_at : new Date().toISOString();

      const completedExecution: ScriptExecution = {
        ...execution,
        status: response.ok && exitCode === 0 && !timedOut ? 'completed' : 'failed',
        completed_at: completedAt,
        exit_code: exitCode,
        output: stdout,
        stdout,
        stderr,
        error: stderr || data?.error || (!response.ok ? `Request failed with status ${response.status}` : undefined),
        timed_out: timedOut,
        duration_seconds: typeof data?.duration_seconds === 'number' ? data.duration_seconds : undefined
      };

      setModalState(prev => ({
        ...prev,
        scriptResults: {
          ...prev.scriptResults,
          execution: completedExecution
        }
      }));
    } catch (error) {
      console.error('Failed to execute script:', error);

      setModalState(prev => ({
        ...prev,
        scriptResults: {
          ...prev.scriptResults,
          execution: {
            script_id: scriptId,
            execution_id: `exec-${Date.now()}`,
            status: 'failed',
            started_at: new Date().toISOString(),
            completed_at: new Date().toISOString(),
            exit_code: 1,
            error: error instanceof Error ? error.message : 'Unknown error occurred'
          }
        }
      }));
    }
  };

  const saveScript = async (script: InvestigationScript, content: string) => {
    try {
      // TODO: Implement actual API call to save script
      console.log('Saving script:', script, content);
      // For now, just close the modal
      closeScriptEditor();
    } catch (error) {
      console.error('Failed to save script:', error);
    }
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
          isOnline={dashboardState.isOnline}
          unreadErrorCount={dashboardState.unreadErrorCount}
          onRefresh={refreshDashboard}
          onToggleTerminal={toggleTerminal}
        />

        {/* System Status Indicator */}
        <div style={{
          position: 'fixed',
          top: '80px',
          right: '20px',
          zIndex: 100
        }}>
          <StatusIndicator onOpenSettings={() => setSystemSettingsModalOpen(true)} />
        </div>

        <main className="main-content">
          <div className="container" style={{ padding: '2rem', maxWidth: '1400px', margin: '0 auto' }}>
            <Routes>
              <Route
                path="/"
                element={(
                  <>
                    {/* Real-time Metrics Grid */}
                    <section className="mb-lg">
                      <MetricsGrid
                        metrics={metrics}
                        detailedMetrics={detailedMetrics}
                        expandedCards={dashboardState.expandedCards}
                        onToggleCard={toggleCard}
                        metricHistory={metricHistory}
                        storageIO={infrastructureData?.storage_io}
                        diskLastUpdated={infrastructureData?.timestamp}
                        onOpenDetail={openDetailPage}
                      />
                    </section>

                    {/* Infrastructure Monitor Panel */}
                    <section className="mb-lg">
                      <InfrastructureMonitor 
                        data={infrastructureData}
                        isExpanded={dashboardState.expandedPanels.has('infrastructure')}
                        onToggle={() => togglePanel('infrastructure')}
                        systemHealth={detailedMetrics?.system_details}
                      />
                    </section>

                    {/* Alert Panel */}
                    <section className="mb-lg">
                      <AlertPanel alerts={dashboardState.alerts} />
                    </section>

                    {/* Investigations Section */}
                    <section className="mb-lg">
                      <InvestigationsSection 
                        investigations={investigations}
                        onOpenScriptEditor={openScriptEditor}
                        onSpawnAgent={async (autoFix: boolean, note?: string) => {
                          try {
                            const requestBody: { auto_fix: boolean; note?: string } = { auto_fix: autoFix };
                            if (note) {
                              requestBody.note = note;
                            }

                            const response = await fetch('/api/investigations/trigger', {
                              method: 'POST',
                              headers: {
                                'Content-Type': 'application/json'
                              },
                              body: JSON.stringify(requestBody)
                            });
                            
                            if (response.ok) {
                              console.log('Investigation triggered with auto-fix:', autoFix);
                              // TODO: Show success message or refresh investigations
                            }
                          } catch (error) {
                            console.error('Failed to trigger investigation:', error);
                          }
                        }}
                      />
                    </section>

                    {/* Playback Reports */}
                    <section className="mb-lg">
                      <ReportsPanel />
                    </section>
                  </>
                )}
              />

              <Route
                path="/metrics/cpu"
                element={(
                  <CpuDetailView
                    metrics={metrics}
                    detailedMetrics={detailedMetrics}
                    processMonitorData={processMonitorData}
                    metricHistory={metricHistory}
                    onBack={handleBackToDashboard}
                  />
                )}
              />
              <Route
                path="/metrics/memory"
                element={(
                  <MemoryDetailView
                    metrics={metrics}
                    detailedMetrics={detailedMetrics}
                    metricHistory={metricHistory}
                    onBack={handleBackToDashboard}
                  />
                )}
              />
              <Route
                path="/metrics/network"
                element={(
                  <NetworkDetailView
                    metrics={metrics}
                    detailedMetrics={detailedMetrics}
                    metricHistory={metricHistory}
                    onBack={handleBackToDashboard}
                  />
                )}
              />
              <Route
                path="/metrics/disk"
                element={(
                  <DiskDetailView
                    detailedMetrics={detailedMetrics}
                    storageIO={infrastructureData?.storage_io}
                    metricHistory={metricHistory}
                    diskLastUpdated={infrastructureData?.timestamp}
                    onBack={handleBackToDashboard}
                  />
                )}
              />
            </Routes>

          </div>
        </main>

        <Terminal 
          isVisible={dashboardState.terminalVisible}
          onClose={toggleTerminal}
        />

        <ModalsContainer 
          modalState={modalState}
          onCloseScriptEditor={closeScriptEditor}
          onCloseScriptResults={closeScriptResults}
          onExecuteScript={executeScript}
          onSaveScript={saveScript}
        />

        {/* System Settings Modal */}
        <SystemSettingsModal
          isOpen={systemSettingsModalOpen}
          onClose={() => setSystemSettingsModalOpen(false)}
        />
      </div>
    </ErrorBoundary>
  );
}

export default App;
