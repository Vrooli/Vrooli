import { useCallback, useEffect, useState } from 'react';
import type { ErrorInfo } from 'react';
import { Routes, Route, useNavigate } from 'react-router-dom';
import { Header } from './components/common/Header';
import { MetricsGrid } from './components/metrics/MetricsGrid';
import { CpuDetailView, MemoryDetailView, NetworkDetailView, DiskDetailView, GpuDetailView } from './components/metrics/MetricDetailViews';
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
import { buildApiUrl } from './utils/apiBase';
import { InvestigationScriptsPage } from './components/investigations/InvestigationScriptsPage';
import type { DashboardState, ModalState, InvestigationScript, ScriptExecution, CardType, InvestigationAgentState, PanelType } from './types';
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
  const [agents, setAgents] = useState<InvestigationAgentState[]>([]);
  const [isSpawningAgent, setIsSpawningAgent] = useState(false);
  const [stoppingAgents, setStoppingAgents] = useState<Set<string>>(() => new Set());
  const [agentErrors, setAgentErrors] = useState<Record<string, string>>({});
  const [spawnAgentError, setSpawnAgentError] = useState<string | null>(null);

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

  const mapAgentPayload = useCallback((payload: unknown): InvestigationAgentState | null => {
    if (!payload || typeof payload !== 'object') {
      return null;
    }

    const payloadRecord = payload as Record<string, unknown>;

    const rawDetails = typeof payloadRecord.details === 'object' && payloadRecord.details !== null
      ? payloadRecord.details as Record<string, unknown>
      : undefined;

    const idCandidate = typeof payloadRecord.id === 'string'
      ? payloadRecord.id
      : typeof payloadRecord.investigation_id === 'string'
      ? payloadRecord.investigation_id as string
      : typeof payloadRecord.agent_id === 'string'
      ? payloadRecord.agent_id as string
      : undefined;

    if (!idCandidate) {
      return null;
    }

    const extractString = (value: unknown): string | undefined => {
      return typeof value === 'string' ? value : undefined;
    };

    const extractBoolean = (value: unknown): boolean | undefined => {
      return typeof value === 'boolean' ? value : undefined;
    };

    const extractNumber = (value: unknown): number | undefined => {
      return typeof value === 'number' ? value : undefined;
    };

    const statusCandidate = extractString(payloadRecord.status) ?? extractString(payloadRecord.state) ?? 'investigating';
    const startTime = extractString(payloadRecord.start_time)
      ?? extractString(payloadRecord.startTime)
      ?? extractString(payloadRecord.started_at)
      ?? new Date().toISOString();
    const autoFixValue = extractBoolean(payloadRecord.auto_fix)
      ?? extractBoolean(payloadRecord.autoFix)
      ?? (rawDetails ? extractBoolean(rawDetails['auto_fix']) : undefined)
      ?? false;
    const operationModeValue = extractString(payloadRecord.operation_mode)
      ?? (rawDetails ? extractString(rawDetails['operation_mode']) : undefined);
    const modelValue = extractString(payloadRecord.agent_model)
      ?? (rawDetails ? extractString(rawDetails['agent_model']) : undefined);
    const resourceValue = extractString(payloadRecord.agent_resource)
      ?? (rawDetails ? extractString(rawDetails['agent_resource']) : undefined);
    const progressValue = extractNumber(payloadRecord.progress)
      ?? (rawDetails ? extractNumber(rawDetails['progress']) : undefined);
    const riskLevelValue = extractString(payloadRecord.risk_level)
      ?? (rawDetails ? extractString(rawDetails['risk_level']) : undefined);
    const noteValue = extractString(payloadRecord.note)
      ?? (rawDetails ? extractString((rawDetails['user_note'] ?? rawDetails['note'])) : undefined);
    const labelValue = extractString(payloadRecord.label)
      ?? extractString(payloadRecord.name)
      ?? (rawDetails ? extractString(rawDetails['label']) : undefined);
    const anomalyIdValue = extractString(payloadRecord.anomaly_id)
      ?? (rawDetails ? extractString(rawDetails['anomaly_id']) : undefined);
    const completedAt = extractString(payloadRecord.completed_at)
      ?? extractString(payloadRecord.completedAt);
    const lastUpdated = extractString(payloadRecord.updated_at)
      ?? extractString(payloadRecord.last_updated)
      ?? extractString(payloadRecord.timestamp);
    const errorMessage = extractString(payloadRecord.error)
      ?? extractString(payloadRecord.failure_reason)
      ?? (rawDetails ? extractString(rawDetails['error']) : undefined);

    return {
      id: idCandidate,
      status: statusCandidate,
      startTime,
      autoFix: autoFixValue,
      operationMode: operationModeValue,
      model: modelValue,
      resource: resourceValue,
      progress: progressValue,
      riskLevel: riskLevelValue,
      note: noteValue,
      label: labelValue,
      anomalyId: anomalyIdValue,
      details: rawDetails,
      lastUpdated,
      completedAt,
      error: errorMessage
    };
  }, []);

  const parseAgentsResponse = useCallback((data: unknown): InvestigationAgentState[] => {
    if (!data) {
      return [];
    }

    const candidates: unknown[] = [];

    if (Array.isArray(data)) {
      candidates.push(...data);
    } else if (typeof data === 'object' && data !== null) {
      const maybeObject = data as Record<string, unknown>;
      if (Array.isArray(maybeObject.agents)) {
        candidates.push(...maybeObject.agents);
      } else if (maybeObject.agent) {
        candidates.push(maybeObject.agent);
      } else if (typeof maybeObject.id === 'string' || typeof maybeObject.investigation_id === 'string') {
        candidates.push(maybeObject);
      }
    }

    return candidates
      .map(candidate => mapAgentPayload(candidate))
      .filter((agent): agent is InvestigationAgentState => Boolean(agent));
  }, [mapAgentPayload]);

  const fetchActiveAgents = useCallback(async () => {
    try {
      const response = await fetch(buildApiUrl('/investigations/agent/current'));
      if (response.status === 404) {
        setAgents([]);
        return;
      }

      if (!response.ok) {
        throw new Error(`Request failed with status ${response.status}`);
      }

      let payload: unknown = null;
      if (response.status !== 204) {
        try {
          payload = await response.json();
        } catch (parseError) {
          console.error('Failed to parse active agent response:', parseError);
        }
      }

      setAgents(parseAgentsResponse(payload));
    } catch (fetchError) {
      console.error('Failed to fetch active agents:', fetchError);
    }
  }, [parseAgentsResponse]);

  const spawnInvestigationAgent = useCallback(async ({ autoFix, note }: { autoFix: boolean; note?: string }) => {
    setSpawnAgentError(null);
    setIsSpawningAgent(true);
    try {
      const response = await fetch(buildApiUrl('/investigations/agent/spawn'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ autoFix, note })
      });

      if (!response.ok) {
        let message = `Request failed with status ${response.status}`;
        try {
          const errorPayload = await response.json();
          if (typeof errorPayload?.error === 'string') {
            message = errorPayload.error;
          }
        } catch {
          // Ignore JSON parse failure for error payloads
        }
        throw new Error(message);
      }

      let data: Record<string, unknown> = {};
      try {
        data = await response.json() as Record<string, unknown>;
      } catch {
        data = {};
      }

      const resolvedId = typeof data.investigation_id === 'string'
        ? data.investigation_id
        : typeof data.id === 'string'
        ? data.id
        : undefined;

      const payload = {
        ...data,
        id: resolvedId,
        start_time: typeof data.start_time === 'string'
          ? data.start_time
          : typeof data.started_at === 'string'
          ? data.started_at
          : new Date().toISOString(),
        auto_fix: typeof data.auto_fix === 'boolean' ? data.auto_fix : autoFix,
        note: typeof data.note === 'string' ? data.note : note
      } as Record<string, unknown>;

      const mapped = mapAgentPayload(payload);

      if (!mapped) {
        throw new Error('Agent response missing identifier');
      }

      setAgents(prev => {
        const existingIndex = prev.findIndex(agent => agent.id === mapped.id);
        if (existingIndex === -1) {
          return [mapped, ...prev];
        }
        const next = [...prev];
        next[existingIndex] = { ...prev[existingIndex], ...mapped };
        return next;
      });

      return mapped;
    } catch (spawnError) {
      const message = spawnError instanceof Error ? spawnError.message : 'Unknown error spawning investigation agent';
      setSpawnAgentError(message);
      throw spawnError;
    } finally {
      setIsSpawningAgent(false);
    }
  }, [mapAgentPayload]);

  const triggerInvestigation = useCallback(async ({ autoFix, note }: { autoFix: boolean; note?: string }) => {
    try {
      const requestBody: { auto_fix: boolean; note?: string } = { auto_fix: autoFix };
      if (note) {
        requestBody.note = note;
      }

      const response = await fetch(buildApiUrl('/investigations/trigger'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(requestBody)
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Investigation trigger failed:', errorText || response.statusText);
      }
    } catch (triggerError) {
      console.error('Failed to trigger investigation:', triggerError);
    }
  }, []);

  const handleAgentSpawn = useCallback(async ({ autoFix, note }: { autoFix: boolean; note?: string }) => {
    const agent = await spawnInvestigationAgent({ autoFix, note });
    void triggerInvestigation({ autoFix, note });
    return agent;
  }, [spawnInvestigationAgent, triggerInvestigation]);

  const stopAgent = useCallback(async (agentId: string) => {
    setAgentErrors(prev => {
      if (!(agentId in prev)) {
        return prev;
      }
      const next = { ...prev };
      delete next[agentId];
      return next;
    });

    setStoppingAgents(prev => {
      const next = new Set(prev);
      next.add(agentId);
      return next;
    });

    try {
      const response = await fetch(buildApiUrl(`/investigations/agent/${encodeURIComponent(agentId)}/stop`), {
        method: 'POST'
      });

      if (!response.ok) {
        let message = `Request failed with status ${response.status}`;
        try {
          const payload = await response.json() as Record<string, unknown>;
          if (typeof payload.error === 'string') {
            message = payload.error;
          }
        } catch {
          // Ignore
        }
        throw new Error(message);
      }

      setAgents(prev => prev.filter(agent => agent.id !== agentId));
    } catch (stopError) {
      const message = stopError instanceof Error ? stopError.message : 'Failed to stop agent';
      setAgentErrors(prev => ({ ...prev, [agentId]: message }));
      throw stopError;
    } finally {
      setStoppingAgents(prev => {
        const next = new Set(prev);
        next.delete(agentId);
        return next;
      });
    }
  }, []);

  useEffect(() => {
    void fetchActiveAgents();
  }, [fetchActiveAgents]);

  useEffect(() => {
    const terminalStatuses = ['completed', 'error', 'failed', 'cancelled', 'canceled', 'stopped'];
    const activeAgents = agents.filter(agent => {
      const status = agent.status?.toLowerCase?.();
      if (!status) {
        return true;
      }
      return !terminalStatuses.includes(status);
    });

    if (activeAgents.length === 0) {
      return;
    }

    let isMounted = true;

    const pollOnce = async () => {
      await Promise.all(activeAgents.map(async agent => {
        try {
          const response = await fetch(buildApiUrl(`/investigations/agent/${encodeURIComponent(agent.id)}/status`));
          if (!response.ok) {
            return;
          }

          let payload: unknown = null;
          try {
            payload = await response.json();
          } catch (parseError) {
            console.error('Failed to parse agent status payload:', parseError);
          }

          const parsed = parseAgentsResponse(payload);
          const mapped = parsed.find(item => item.id === agent.id)
            ?? mapAgentPayload({ ...(payload as Record<string, unknown>), id: agent.id });

          if (mapped && isMounted) {
            setAgents(prev => prev.map(existing => existing.id === mapped.id ? { ...existing, ...mapped } : existing));
          }
        } catch (pollError) {
          console.error('Failed to poll agent status:', pollError);
        }
      }));
    };

    void pollOnce();
    const interval = setInterval(() => {
      void pollOnce();
    }, 4000);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [agents, mapAgentPayload, parseAgentsResponse]);

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

  const executeScript = async (scriptId: string, scriptContent: string) => {
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

      const requestBody = scriptContent ? { content: scriptContent } : {};

      const response = await fetch(buildApiUrl(`/investigations/scripts/${encodeURIComponent(scriptId)}/execute`), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(requestBody)
      });

      let data: Record<string, unknown> | null = null;
      try {
        data = await response.json() as Record<string, unknown>;
      } catch {
        data = null;
      }

      const readString = (value: unknown): string | undefined => {
        return typeof value === 'string' ? value : undefined;
      };
      const readNumber = (value: unknown): number | undefined => {
        return typeof value === 'number' ? value : undefined;
      };
      const readBoolean = (value: unknown): boolean => value === true;

      const stdout = readString(data?.['stdout']) ?? readString(data?.['output']) ?? '';
      const stderr = readString(data?.['stderr']) ?? '';
      const exitCode = readNumber(data?.['exit_code']) ?? (response.ok ? 0 : 1);
      const timedOut = readBoolean(data?.['timed_out']);
      const completedAt = readString(data?.['completed_at']) ?? new Date().toISOString();
      const errorFromResponse = readString(data?.['error']);
      const durationSeconds = readNumber(data?.['duration_seconds']);

      const completedExecution: ScriptExecution = {
        ...execution,
        status: response.ok && exitCode === 0 && !timedOut ? 'completed' : 'failed',
        completed_at: completedAt,
        exit_code: exitCode,
        output: stdout,
        stdout,
        stderr,
        error: stderr || errorFromResponse || (!response.ok ? `Request failed with status ${response.status}` : undefined),
        timed_out: timedOut,
        duration_seconds: durationSeconds
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
          agents={agents}
          onStopAgent={stopAgent}
          stoppingAgentIds={stoppingAgents}
          agentErrors={agentErrors}
          onRefreshAgents={fetchActiveAgents}
          onToggleTerminal={toggleTerminal}
          onOpenSettings={() => setSystemSettingsModalOpen(true)}
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
                        onSpawnAgent={handleAgentSpawn}
                        agents={agents}
                        isSpawningAgent={isSpawningAgent}
                        spawnAgentError={spawnAgentError}
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
                path="/scripts"
                element={(
                  <InvestigationScriptsPage 
                    onOpenScriptEditor={openScriptEditor}
                    onExecuteScript={executeScript}
                    onSaveScript={saveScript}
                  />
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
                path="/metrics/gpu"
                element={(
                  <GpuDetailView
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
