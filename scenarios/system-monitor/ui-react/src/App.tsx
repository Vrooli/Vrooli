import { useEffect, useState } from 'react';
import { Header } from './components/common/Header';
import { MetricsGrid } from './components/metrics/MetricsGrid';
import { ProcessMonitor } from './components/monitoring/ProcessMonitor';
import { InfrastructureMonitor } from './components/monitoring/InfrastructureMonitor';
import { AlertPanel } from './components/common/AlertPanel';
import { InvestigationsSection } from './components/investigations/InvestigationsSection';
import { ReportsPanel } from './components/reports/ReportsPanel';
import { Terminal } from './components/common/Terminal';
import { ModalsContainer } from './components/modals/ModalsContainer';
import { MatrixBackground } from './components/common/MatrixBackground';
import { useSystemMonitor } from './hooks/useSystemMonitor';
import type { DashboardState, ModalState, InvestigationScript, ScriptExecution } from './types';
import './styles/matrix-theme.css';

function App() {
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

  const {
    metrics,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
    investigations,
    isLoading,
    error
  } = useSystemMonitor();

  // Update online status based on successful API calls
  useEffect(() => {
    setDashboardState(prev => ({
      ...prev,
      isOnline: !isLoading && !error,
      lastUpdate: new Date().toISOString()
    }));
  }, [isLoading, error]);

  const toggleCard = (cardType: string) => {
    setDashboardState(prev => {
      const newExpandedCards = new Set(prev.expandedCards);
      if (newExpandedCards.has(cardType as any)) {
        newExpandedCards.delete(cardType as any);
      } else {
        newExpandedCards.add(cardType as any);
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

  const executeScript = async (scriptId: string, content: string) => {
    try {
      // Create execution object
      const execution: ScriptExecution = {
        script_id: scriptId,
        execution_id: `exec-${Date.now()}`,
        status: 'running',
        started_at: new Date().toISOString()
      };

      // Open results modal immediately
      setModalState(prev => ({
        ...prev,
        scriptResults: {
          isOpen: true,
          scriptId,
          executionId: execution.execution_id,
          execution
        }
      }));

      // Close script editor
      closeScriptEditor();

      // TODO: Implement actual API call to execute script
      // For now, simulate execution
      setTimeout(() => {
        const completedExecution: ScriptExecution = {
          ...execution,
          status: 'completed',
          completed_at: new Date().toISOString(),
          exit_code: 0,
          output: `[${new Date().toLocaleTimeString()}] Starting script execution...\n[${new Date().toLocaleTimeString()}] Script: ${scriptId}\n[${new Date().toLocaleTimeString()}] Content length: ${content.length} characters\n\n# Mock execution output\necho "System analysis starting..."\necho "CPU usage: $(grep 'cpu ' /proc/stat | awk '{usage=($2+$4)*100/($2+$3+$4+$5)} END {print usage "%"}')"
echo "Memory usage: $(free | grep Mem | awk '{printf("%.1f%%"), $3/$2 * 100.0}')"
echo "Disk usage: $(df -h / | awk 'NR==2{printf "%s", $5}')"
echo "Load average: $(uptime | awk -F'load average:' '{print $2}')"

[${new Date().toLocaleTimeString()}] Script execution completed successfully.
[${new Date().toLocaleTimeString()}] Exit code: 0`
        };

        setModalState(prev => ({
          ...prev,
          scriptResults: {
            ...prev.scriptResults,
            execution: completedExecution
          }
        }));
      }, 2000);

    } catch (error) {
      console.error('Failed to execute script:', error);
      
      // Update with error
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

  return (
    <div className="app">
      <MatrixBackground />
      
      <Header 
        isOnline={dashboardState.isOnline}
        unreadErrorCount={dashboardState.unreadErrorCount}
        onRefresh={refreshDashboard}
        onToggleTerminal={toggleTerminal}
      />

      <main className="main-content">
        <div className="container" style={{ padding: '2rem', maxWidth: '1400px', margin: '0 auto' }}>
          
          {/* Real-time Metrics Grid */}
          <section className="mb-lg">
            <MetricsGrid 
              metrics={metrics}
              detailedMetrics={detailedMetrics}
              expandedCards={dashboardState.expandedCards}
              onToggleCard={toggleCard}
            />
          </section>

          {/* Process Monitor Panel */}
          <section className="mb-lg">
            <ProcessMonitor 
              data={processMonitorData}
              isExpanded={dashboardState.expandedPanels.has('process')}
              onToggle={() => togglePanel('process')}
            />
          </section>

          {/* Infrastructure Monitor Panel */}
          <section className="mb-lg">
            <InfrastructureMonitor 
              data={infrastructureData}
              isExpanded={dashboardState.expandedPanels.has('infrastructure')}
              onToggle={() => togglePanel('infrastructure')}
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
    </div>
  );
}

export default App;
