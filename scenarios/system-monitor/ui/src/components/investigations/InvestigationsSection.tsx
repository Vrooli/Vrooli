import { useState, useEffect } from 'react';
import { ChevronDown, ChevronRight, Search, AlertTriangle, Shield, Bot, Play, MessageCircle, ChevronUp, Square, Clock, CheckCircle, AlertCircle } from 'lucide-react';
import { InvestigationsPanel } from './InvestigationsPanel';
import { InvestigationScriptsPanel } from './InvestigationScriptsPanel';
import { AutomaticTriggersSection } from './AutomaticTriggersSection';
import type { Investigation, InvestigationScript } from '../../types';

type AgentState = {
  id: string;
  status: string;
  startTime: string;
  autoFix: boolean;
  operationMode?: string;
  model?: string;
  resource?: string;
  progress?: number;
  riskLevel?: string;
  note?: string;
  details?: Record<string, unknown>;
};

const mapAgentPayload = (payload: any, previous?: AgentState | null): AgentState | null => {
  const prev = previous ?? null;
  if (!payload && !prev) {
    return null;
  }

  const rawDetails = payload?.details && typeof payload.details === 'object'
    ? payload.details as Record<string, unknown>
    : prev?.details ?? {};
  const details: Record<string, unknown> = { ...rawDetails };

  const startTime = typeof payload?.startTime === 'string'
    ? payload.startTime
    : typeof payload?.start_time === 'string'
    ? payload.start_time
    : prev?.startTime ?? new Date().toISOString();

  const id = typeof payload?.id === 'string'
    ? payload.id
    : typeof payload?.investigation_id === 'string'
    ? payload.investigation_id
    : prev?.id ?? '';

  if (!id) {
    return prev;
  }

  const next: AgentState = {
    id,
    status: typeof payload?.status === 'string' ? payload.status : prev?.status ?? 'investigating',
    startTime,
    autoFix: typeof payload?.auto_fix === 'boolean'
      ? payload.auto_fix
      : typeof details['auto_fix'] === 'boolean'
      ? (details['auto_fix'] as boolean)
      : prev?.autoFix ?? false,
    operationMode: typeof payload?.operation_mode === 'string'
      ? payload.operation_mode
      : typeof details['operation_mode'] === 'string'
      ? (details['operation_mode'] as string)
      : prev?.operationMode,
    model: typeof payload?.agent_model === 'string'
      ? payload.agent_model
      : typeof details['agent_model'] === 'string'
      ? (details['agent_model'] as string)
      : prev?.model,
    resource: typeof payload?.agent_resource === 'string'
      ? payload.agent_resource
      : typeof details['agent_resource'] === 'string'
      ? (details['agent_resource'] as string)
      : prev?.resource,
    progress: typeof payload?.progress === 'number'
      ? payload.progress
      : prev?.progress,
    riskLevel: typeof payload?.risk_level === 'string'
      ? payload.risk_level
      : typeof details['risk_level'] === 'string'
      ? (details['risk_level'] as string)
      : prev?.riskLevel,
    note: typeof payload?.note === 'string'
      ? payload.note
      : typeof details['user_note'] === 'string'
      ? (details['user_note'] as string)
      : prev?.note,
    details,
  };

  return next;
};

interface InvestigationsSectionProps {
  investigations: Investigation[];
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  onSpawnAgent: (autoFix: boolean, note?: string) => void;
}

export const InvestigationsSection = ({ 
  investigations, 
  onOpenScriptEditor,
  onSpawnAgent 
}: InvestigationsSectionProps) => {
  const [reportsExpanded, setReportsExpanded] = useState(true);
  const [scriptsExpanded, setScriptsExpanded] = useState(true);
  const [autoFixEnabled, setAutoFixEnabled] = useState(false);
  const [isSpawning, setIsSpawning] = useState(false);
  const [reportsSearch, setReportsSearch] = useState('');
  const [scriptsSearch, setScriptsSearch] = useState('');
  const [agentNote, setAgentNote] = useState('');
  const [showNoteField, setShowNoteField] = useState(false);
  
  // Agent status tracking - now API-driven
  const [agentData, setAgentData] = useState<AgentState | null>(null);
  const [statusPolling, setStatusPolling] = useState(false);

  // Filter investigations based on search
  const filteredInvestigations = investigations.filter(inv => {
    if (!reportsSearch) return true;
    const searchLower = reportsSearch.toLowerCase();
    return inv.id.toLowerCase().includes(searchLower) ||
           inv.findings?.toLowerCase().includes(searchLower) ||
           inv.status?.toLowerCase().includes(searchLower);
  });

  const agentHistory = investigations.slice(0, 5);

  // Check for existing running agent on component mount
  useEffect(() => {
    checkForRunningAgent();
  }, []);

  // Poll for agent status updates when agent is running
  useEffect(() => {
    let pollInterval: NodeJS.Timeout;
    
    if (agentData && agentData.status !== 'completed' && agentData.status !== 'error') {
      setStatusPolling(true);
      pollInterval = setInterval(async () => {
        try {
          const response = await fetch(`/api/investigations/agent/${agentData.id}/status`);
          if (response.ok) {
            const statusData = await response.json();
            setAgentData(prev => mapAgentPayload(statusData, prev));
            
            if (statusData.status === 'completed' || statusData.status === 'error') {
              setStatusPolling(false);
            }
          }
        } catch (error) {
          console.error('Failed to poll agent status:', error);
        }
      }, 2000); // Poll every 2 seconds
    } else {
      setStatusPolling(false);
    }

    return () => {
      if (pollInterval) {
        clearInterval(pollInterval);
        setStatusPolling(false);
      }
    };
  }, [agentData?.id, agentData?.status]);

  const checkForRunningAgent = async () => {
    try {
      const response = await fetch('/api/investigations/agent/current');
      if (response.ok) {
        const data = await response.json();
        if (data.agent) {
          setAgentData(prev => mapAgentPayload(data.agent, prev));
        }
      } else if (response.status !== 404) {
        console.error('Failed to check for running agent:', response.statusText);
      }
    } catch (error) {
      console.error('Failed to check for running agent:', error);
    }
  };

  const formatElapsedTime = (startTime: string): string => {
    const start = new Date(startTime);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - start.getTime()) / 1000);
    
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    
    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    } else {
      return `${secs}s`;
    }
  };

  const handleSpawnAgent = async () => {
    setIsSpawning(true);
    try {
      // Call API to spawn agent
      const response = await fetch('/api/investigations/agent/spawn', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          autoFix: autoFixEnabled,
          note: agentNote.trim() || undefined
        })
      });

      if (response.ok) {
        const agentResponse = await response.json();
        setAgentData(mapAgentPayload({
          id: agentResponse.investigation_id,
          status: agentResponse.status || 'initializing',
          start_time: agentResponse.start_time || new Date().toISOString(),
          auto_fix: typeof agentResponse.auto_fix === 'boolean' ? agentResponse.auto_fix : autoFixEnabled,
          note: agentResponse.note,
          operation_mode: agentResponse.operation_mode,
          agent_model: agentResponse.agent_model,
          agent_resource: agentResponse.agent_resource,
          progress: agentResponse.progress,
          details: agentResponse.details
        }, null));
        
        // Also call the parent handler for any additional logic
        await onSpawnAgent(autoFixEnabled, agentNote.trim() || undefined);
      } else {
        console.error('Failed to spawn agent:', response.statusText);
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
        alert(`Failed to spawn agent: ${errorData.error || response.statusText}`);
      }
    } catch (error) {
      console.error('Failed to spawn agent:', error);
      alert(`Failed to spawn agent: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setIsSpawning(false);
    }
  };

  const handleStopAgent = async () => {
    if (!agentData) return;
    
    try {
      const response = await fetch(`/api/investigations/agent/${agentData.id}/stop`, {
        method: 'POST'
      });
      
      if (response.ok) {
        setAgentData(null);
        setStatusPolling(false);
      } else {
        console.error('Failed to stop agent:', response.statusText);
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
        alert(`Failed to stop agent: ${errorData.error || response.statusText}`);
      }
    } catch (error) {
      console.error('Failed to stop agent:', error);
      alert(`Failed to stop agent: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleUpdateTrigger = (triggerId: string, config: any) => {
    // TODO: Implement API call to update trigger configuration
    console.log('Updating trigger:', triggerId, config);
  };

  return (
    <div className="panel investigations-section" style={{
      border: '1px solid var(--color-accent)',
      borderRadius: 'var(--border-radius-lg)',
      overflow: 'hidden',
      background: 'rgba(0, 0, 0, 0.3)'
    }}>
      <div className="panel-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: 'var(--spacing-md)',
        borderBottom: '1px solid var(--color-accent)',
        background: 'rgba(0, 255, 0, 0.02)'
      }}>
        <h2 style={{ 
          margin: 0,
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--spacing-sm)',
          color: 'var(--color-text-bright)',
          fontSize: 'var(--font-size-xl)'
        }}>
          <AlertTriangle size={24} style={{ color: 'var(--color-warning)' }} />
          INVESTIGATIONS
        </h2>
      </div>

      <div className="panel-body" style={{ padding: 'var(--spacing-lg)' }}>
        
        {/* Agent Spawn Card */}
        <div className="agent-spawn-card" style={{
          background: 'rgba(0, 255, 0, 0.03)',
          border: '1px solid var(--color-accent)',
          borderRadius: 'var(--border-radius-md)',
          padding: 'var(--spacing-lg)',
          marginBottom: 'var(--spacing-xl)',
          boxShadow: 'var(--shadow-glow)'
        }}>
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 'var(--spacing-lg)' }}>
            <Bot size={48} style={{ 
              color: 'var(--color-success)', 
              flexShrink: 0,
              filter: 'drop-shadow(0 0 10px rgba(0, 255, 0, 0.5))'
            }} />
            
            <div style={{ flex: 1 }}>
              <h3 style={{ 
                margin: '0 0 var(--spacing-sm) 0',
                color: 'var(--color-text-bright)',
                fontSize: 'var(--font-size-lg)'
              }}>
                System Anomaly Investigation Agent
              </h3>
              
              <p style={{ 
                margin: '0 0 var(--spacing-md) 0',
                color: 'var(--color-text-dim)',
                fontSize: 'var(--font-size-sm)',
                lineHeight: '1.5'
              }}>
                Spawns an AI agent to investigate system anomalies, analyze performance metrics, 
                and identify potential issues. The agent uses existing investigation scripts and 
                creates new tools to build lasting intelligence for future investigations.
              </p>

              <div style={{ 
                display: 'flex', 
                alignItems: 'center', 
                gap: 'var(--spacing-lg)',
                flexWrap: 'wrap',
                marginBottom: 'var(--spacing-md)'
              }}>
                {/* Auto-fix Checkbox */}
                <label style={{ 
                  display: 'flex', 
                  alignItems: 'center', 
                  gap: 'var(--spacing-sm)',
                  cursor: 'pointer',
                  color: 'var(--color-text)',
                  fontSize: 'var(--font-size-sm)',
                  userSelect: 'none'
                }}>
                  <input 
                    type="checkbox"
                    checked={autoFixEnabled}
                    onChange={(e) => setAutoFixEnabled(e.target.checked)}
                    style={{
                      width: '18px',
                      height: '18px',
                      accentColor: 'var(--color-success)',
                      cursor: 'pointer'
                    }}
                  />
                  <Shield size={16} style={{ 
                    color: autoFixEnabled ? 'var(--color-success)' : 'var(--color-text-dim)' 
                  }} />
                  <span style={{ 
                    fontWeight: autoFixEnabled ? 'bold' : 'normal',
                    color: autoFixEnabled ? 'var(--color-text-bright)' : 'inherit'
                  }}>
                    Adaptive auto-fix & recovery
                  </span>
                </label>

                {/* Note Toggle Button */}
                <button 
                  className="btn btn-secondary"
                  onClick={() => setShowNoteField(!showNoteField)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 'var(--spacing-xs)',
                    padding: 'var(--spacing-xs) var(--spacing-sm)',
                    fontSize: 'var(--font-size-xs)'
                  }}
                >
                  <MessageCircle size={14} />
                  NOTE
                  {showNoteField ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                </button>

                {/* Agent Control - Show spawn button or status */}
                {!agentData ? (
                  <button 
                    className="btn btn-primary"
                    onClick={handleSpawnAgent}
                    disabled={isSpawning}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 'var(--spacing-sm)',
                      padding: 'var(--spacing-sm) var(--spacing-lg)',
                      fontSize: 'var(--font-size-sm)',
                      fontWeight: 'bold',
                      textTransform: 'uppercase',
                      letterSpacing: '0.5px'
                    }}
                  >
                    <Play size={16} className={isSpawning ? 'animate-spin' : ''} />
                    {isSpawning ? 'SPAWNING...' : 'SPAWN AGENT'}
                  </button>
                ) : (
                  <div
                    className="agent-status-display"
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 'var(--spacing-sm)',
                      padding: 'var(--spacing-sm) var(--spacing-lg)',
                      background: agentData.status === 'error'
                        ? 'rgba(255, 0, 0, 0.1)'
                        : 'rgba(0, 255, 0, 0.08)',
                      border: agentData.status === 'error'
                        ? '1px solid var(--color-error)'
                        : '1px solid var(--color-success)',
                      borderRadius: 'var(--border-radius-md)',
                      minWidth: '320px'
                    }}
                  >
                    <div style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      gap: 'var(--spacing-sm)'
                    }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-xs)' }}>
                        {agentData.status === 'initializing' && (
                          <Bot size={16} className="animate-pulse" style={{ color: 'var(--color-warning)' }} />
                        )}
                        {agentData.status === 'investigating' && (
                          <Search size={16} className="animate-pulse" style={{ color: 'var(--color-success)' }} />
                        )}
                        {agentData.status === 'analyzing' && (
                          <AlertTriangle size={16} className="animate-pulse" style={{ color: 'var(--color-warning)' }} />
                        )}
                        {agentData.status === 'completed' && (
                          <CheckCircle size={16} style={{ color: 'var(--color-success)' }} />
                        )}
                        {agentData.status === 'error' && (
                          <AlertCircle size={16} style={{ color: 'var(--color-error)' }} />
                        )}
                        <span style={{
                          color: 'var(--color-text-bright)',
                          fontSize: 'var(--font-size-sm)',
                          fontWeight: 'bold',
                          textTransform: 'uppercase',
                          letterSpacing: '0.5px'
                        }}>
                          {agentData.status}
                        </span>
                      </div>

                      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-xs)' }}>
                          <Clock size={14} style={{ color: 'var(--color-text-dim)' }} />
                          <span style={{
                            color: 'var(--color-text)',
                            fontSize: 'var(--font-size-sm)',
                            fontFamily: 'var(--font-family-mono)'
                          }}>
                            {formatElapsedTime(agentData.startTime)}
                          </span>
                        </div>
                        {statusPolling && (
                          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-xs)' }}>
                            <div style={{
                              width: '8px',
                              height: '8px',
                              borderRadius: '50%',
                              background: 'var(--color-success)',
                              animation: 'pulse 1.5s infinite'
                            }} />
                            <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                              LIVE
                            </span>
                          </div>
                        )}
                        {agentData.status !== 'completed' && agentData.status !== 'error' && (
                          <button
                            className="btn btn-secondary"
                            onClick={handleStopAgent}
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 'var(--spacing-xs)',
                              padding: 'var(--spacing-xs) var(--spacing-sm)',
                              fontSize: 'var(--font-size-xs)',
                              background: 'rgba(255, 0, 0, 0.1)',
                              border: '1px solid var(--color-error)',
                              color: 'var(--color-error)'
                            }}
                          >
                            <Square size={12} />
                            STOP
                          </button>
                        )}
                        {(agentData.status === 'completed' || agentData.status === 'error') && (
                          <button
                            className="btn btn-secondary"
                            onClick={() => setAgentData(null)}
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 'var(--spacing-xs)',
                              padding: 'var(--spacing-xs) var(--spacing-sm)',
                              fontSize: 'var(--font-size-xs)'
                            }}
                          >
                            CLEAR
                          </button>
                        )}
                      </div>
                    </div>

                    <div style={{
                      display: 'flex',
                      flexWrap: 'wrap',
                      gap: 'var(--spacing-md)',
                      color: 'var(--color-text-dim)',
                      fontSize: 'var(--font-size-xs)'
                    }}>
                      <span>Mode: {agentData.operationMode ?? 'report-only'}</span>
                      <span>Auto-Fix: {agentData.autoFix ? 'enabled' : 'disabled'}</span>
                      {agentData.model && <span>Model: {agentData.model}</span>}
                      {agentData.resource && <span>Resource: {agentData.resource}</span>}
                      {agentData.riskLevel && (
                        <span style={{ color: agentData.riskLevel === 'high'
                          ? 'var(--color-error)'
                          : agentData.riskLevel === 'medium'
                          ? 'var(--color-warning)'
                          : 'var(--color-success)' }}>
                          Risk: {agentData.riskLevel}
                        </span>
                      )}
                    </div>

                    {typeof agentData.progress === 'number' && !Number.isNaN(agentData.progress) && (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
                        <div style={{
                          height: '6px',
                          borderRadius: '999px',
                          background: 'rgba(0, 255, 0, 0.15)',
                          overflow: 'hidden'
                        }}>
                          <div style={{
                            width: `${Math.max(0, Math.min(100, agentData.progress))}%`,
                            height: '6px',
                            background: 'var(--color-success)',
                            transition: 'width var(--transition-normal)'
                          }} />
                        </div>
                        <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                          Progress: {Math.round(agentData.progress)}%
                        </span>
                      </div>
                    )}

                    {agentData.note && (
                      <div style={{
                        color: 'var(--color-text-dim)',
                        fontSize: 'var(--font-size-xs)',
                        fontStyle: 'italic'
                      }}>
                        Note: {agentData.note}
                      </div>
                    )}
                  </div>
                )}
              </div>

              <div style={{
                marginTop: 'var(--spacing-xs)',
                color: 'var(--color-text-dim)',
                fontSize: 'var(--font-size-xs)'
              }}>
                Auto-fix prioritizes reversible fixes and only escalates to critical recovery steps when the system is in a verified dire state.
              </div>

              {/* Agent Note Field */}
              {showNoteField && (
                <div style={{
                  marginBottom: 'var(--spacing-md)',
                  padding: 'var(--spacing-md)',
                  background: 'rgba(0, 0, 0, 0.3)',
                  border: '1px solid var(--color-accent)',
                  borderRadius: 'var(--border-radius-sm)'
                }}>
                  <label style={{
                    display: 'flex',
                    marginBottom: 'var(--spacing-sm)',
                    color: 'var(--color-text-bright)',
                    fontSize: 'var(--font-size-sm)',
                    fontWeight: 'bold',
                    alignItems: 'center',
                    gap: 'var(--spacing-xs)'
                  }}>
                    <MessageCircle size={16} style={{ color: 'var(--color-accent)' }} />
                    Investigation Note (Optional)
                  </label>
                  <textarea
                    value={agentNote}
                    onChange={(e) => setAgentNote(e.target.value)}
                    placeholder="Provide context or specific areas to investigate first...\ne.g., 'Check database connections on port 5432' or 'Focus on recent memory spikes'"
                    style={{
                      width: '100%',
                      minHeight: '80px',
                      padding: 'var(--spacing-sm)',
                      background: 'rgba(0, 0, 0, 0.5)',
                      border: '1px solid var(--color-text-dim)',
                      borderRadius: 'var(--border-radius-sm)',
                      color: 'var(--color-text)',
                      fontSize: 'var(--font-size-sm)',
                      fontFamily: 'var(--font-family-mono)',
                      resize: 'vertical',
                      outline: 'none',
                      transition: 'border-color 0.2s'
                    }}
                    onFocus={(e) => e.currentTarget.style.borderColor = 'var(--color-accent)'}
                    onBlur={(e) => e.currentTarget.style.borderColor = 'var(--color-text-dim)'}
                  />
                  <div style={{
                    marginTop: 'var(--spacing-xs)',
                    color: 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-xs)'
                  }}>
                    This note will be included in the agent's context to help it focus investigation efforts.
                  </div>
                </div>
              )}

              {autoFixEnabled && (
                <div style={{
                  marginTop: 'var(--spacing-md)',
                  padding: 'var(--spacing-sm)',
                  background: 'rgba(0, 255, 0, 0.05)',
                  border: '1px solid var(--color-accent)',
                  borderRadius: 'var(--border-radius-sm)',
                  fontSize: 'var(--font-size-xs)',
                  color: 'var(--color-text-dim)'
                }}>
                  <strong>Adaptive auto-fix:</strong> The agent will still perform routine safe fixes (cache clears,
                  service restarts, configuration tuning), and when metrics confirm a dire state it may execute
                  documented recovery steps like terminating a runaway process or restarting non-primary workers.
                  It will never make irreversible changes such as data deletion, credential updates, or full system reboots.
                </div>
              )}
            </div>
        </div>
      </div>

        {/* Agent History */}
        <div className="agent-history-card" style={{
          background: 'rgba(0, 0, 0, 0.25)',
          border: '1px solid rgba(0, 255, 0, 0.2)',
          borderRadius: 'var(--border-radius-md)',
          padding: 'var(--spacing-md)',
          marginBottom: 'var(--spacing-xl)'
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--spacing-sm)' }}>
            <h4 style={{ margin: 0, color: 'var(--color-text-bright)', fontSize: 'var(--font-size-md)' }}>
              Recent Agent Runs
            </h4>
            <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
              showing {agentHistory.length} of {investigations.length}
            </span>
          </div>

          {agentHistory.length === 0 ? (
            <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
              No investigations have been executed yet.
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'minmax(150px, 1fr) minmax(80px, 0.6fr) minmax(120px, 0.8fr) minmax(140px, 0.8fr) minmax(90px, 0.6fr)',
                gap: 'var(--spacing-sm)',
                padding: 'var(--spacing-xs) 0',
                color: 'var(--color-text-dim)',
                fontSize: 'var(--font-size-xs)',
                textTransform: 'uppercase',
                letterSpacing: '0.5px'
              }}>
                <span>Started</span>
                <span>Status</span>
                <span>Mode</span>
                <span>Model</span>
                <span>Risk</span>
              </div>
              {agentHistory.map(inv => {
                const details = (inv.details ?? {}) as Record<string, unknown>;
                const startedAt = typeof inv.start_time === 'string'
                  ? inv.start_time
                  : typeof inv.timestamp === 'string'
                  ? inv.timestamp
                  : undefined;
                const formattedStart = startedAt ? new Date(startedAt).toLocaleString() : 'Unknown';
                const mode = typeof details['operation_mode'] === 'string' ? details['operation_mode'] as string : 'report-only';
                const risk = typeof details['risk_level'] === 'string' ? details['risk_level'] as string : undefined;
                const model = typeof details['agent_model'] === 'string' ? details['agent_model'] as string : undefined;
                const isActive = agentData?.id === inv.id;

                return (
                  <div
                    key={inv.id}
                    style={{
                      display: 'grid',
                      gridTemplateColumns: 'minmax(150px, 1fr) minmax(80px, 0.6fr) minmax(120px, 0.8fr) minmax(140px, 0.8fr) minmax(90px, 0.6fr)',
                      gap: 'var(--spacing-sm)',
                      alignItems: 'center',
                      padding: 'var(--spacing-sm)',
                      borderRadius: 'var(--border-radius-sm)',
                      background: isActive ? 'rgba(0, 255, 0, 0.12)' : 'rgba(0, 0, 0, 0.2)'
                    }}
                  >
                    <span style={{ color: 'var(--color-text)', fontSize: 'var(--font-size-xs)', fontFamily: 'var(--font-family-mono)' }}>
                      {formattedStart}
                    </span>
                    <span style={{
                      color: inv.status === 'completed'
                        ? 'var(--color-success)'
                        : inv.status === 'failed' || inv.status === 'error'
                        ? 'var(--color-error)'
                        : 'var(--color-warning)',
                      fontSize: 'var(--font-size-xs)',
                      fontWeight: 600,
                      textTransform: 'uppercase'
                    }}>
                      {inv.status}
                    </span>
                    <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                      {mode}
                    </span>
                    <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                      {model ?? 'default'}
                    </span>
                    <span style={{
                      color: risk === 'high'
                        ? 'var(--color-error)'
                        : risk === 'medium'
                        ? 'var(--color-warning)'
                        : 'var(--color-success)',
                      fontSize: 'var(--font-size-xs)'
                    }}>
                      {risk ?? 'n/a'}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Automatic Triggers Section */}
        <AutomaticTriggersSection onUpdateTrigger={handleUpdateTrigger} />

        {/* Reports Subsection */}
        <div className="subsection" style={{ 
          marginBottom: 'var(--spacing-lg)',
          border: '1px solid var(--color-accent)',
          borderRadius: 'var(--border-radius-md)',
          overflow: 'hidden'
        }}>
          <div 
            className="subsection-header"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-sm)',
              padding: 'var(--spacing-sm) var(--spacing-md)',
              background: 'rgba(0, 255, 0, 0.02)',
              borderBottom: reportsExpanded ? '1px solid var(--color-accent)' : 'none',
              userSelect: 'none',
              transition: 'background 0.2s'
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(0, 255, 0, 0.05)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(0, 255, 0, 0.02)'}
          >
            <div 
              onClick={() => setReportsExpanded(!reportsExpanded)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 'var(--spacing-sm)',
                cursor: 'pointer',
                flex: '0 0 auto'
              }}
            >
              {reportsExpanded ? <ChevronDown size={20} /> : <ChevronRight size={20} />}
              <h3 style={{ 
                margin: 0, 
                fontSize: 'var(--font-size-md)',
                color: 'var(--color-text-bright)'
              }}>
                Reports
              </h3>
              <span style={{ 
                fontSize: 'var(--font-size-sm)',
                color: 'var(--color-text-dim)',
                marginLeft: 'var(--spacing-xs)'
              }}>
                ({filteredInvestigations.length})
              </span>
            </div>
            
            {reportsExpanded && (
              <div style={{ 
                position: 'relative',
                flex: 1,
                maxWidth: '300px',
                marginLeft: 'auto'
              }}>
                <Search size={14} style={{
                  position: 'absolute',
                  left: 'var(--spacing-sm)',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  color: 'var(--color-text-dim)',
                  pointerEvents: 'none'
                }} />
                <input
                  type="text"
                  placeholder="Search reports..."
                  value={reportsSearch}
                  onChange={(e) => setReportsSearch(e.target.value)}
                  onClick={(e) => e.stopPropagation()}
                  style={{
                    width: '100%',
                    padding: 'var(--spacing-xs) var(--spacing-xs) var(--spacing-xs) calc(var(--spacing-xs) + 20px)',
                    background: 'rgba(0, 0, 0, 0.3)',
                    border: '1px solid var(--color-accent)',
                    borderRadius: 'var(--border-radius-sm)',
                    color: 'var(--color-text)',
                    fontSize: 'var(--font-size-xs)',
                    fontFamily: 'var(--font-family-mono)',
                    outline: 'none',
                    transition: 'border-color 0.2s'
                  }}
                  onFocus={(e) => e.currentTarget.style.borderColor = 'var(--color-success)'}
                  onBlur={(e) => e.currentTarget.style.borderColor = 'var(--color-accent)'}
                />
              </div>
            )}
          </div>

          {reportsExpanded && (
            <div>              
              <InvestigationsPanel investigations={filteredInvestigations} embedded={true} />
            </div>
          )}
        </div>

        {/* Scripts Subsection */}
        <div className="subsection" style={{
          border: '1px solid var(--color-accent)',
          borderRadius: 'var(--border-radius-md)',
          overflow: 'hidden'
        }}>
          <div 
            className="subsection-header"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-sm)',
              padding: 'var(--spacing-sm) var(--spacing-md)',
              background: 'rgba(0, 255, 0, 0.02)',
              borderBottom: scriptsExpanded ? '1px solid var(--color-accent)' : 'none',
              userSelect: 'none',
              transition: 'background 0.2s'
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(0, 255, 0, 0.05)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(0, 255, 0, 0.02)'}
          >
            <div 
              onClick={() => setScriptsExpanded(!scriptsExpanded)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 'var(--spacing-sm)',
                cursor: 'pointer',
                flex: '0 0 auto'
              }}
            >
              {scriptsExpanded ? <ChevronDown size={20} /> : <ChevronRight size={20} />}
              <h3 style={{ 
                margin: 0, 
                fontSize: 'var(--font-size-md)',
                color: 'var(--color-text-bright)'
              }}>
                Scripts
              </h3>
              <span style={{ 
                fontSize: 'var(--font-size-sm)',
                color: 'var(--color-text-dim)',
                marginLeft: 'var(--spacing-xs)'
              }}>
                (Tools)
              </span>
            </div>
            
            {scriptsExpanded && (
              <div style={{ 
                position: 'relative',
                flex: 1,
                maxWidth: '300px',
                marginLeft: 'auto'
              }}>
                <Search size={14} style={{
                  position: 'absolute',
                  left: 'var(--spacing-sm)',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  color: 'var(--color-text-dim)',
                  pointerEvents: 'none'
                }} />
                <input
                  type="text"
                  placeholder="Search scripts..."
                  value={scriptsSearch}
                  onChange={(e) => setScriptsSearch(e.target.value)}
                  onClick={(e) => e.stopPropagation()}
                  style={{
                    width: '100%',
                    padding: 'var(--spacing-xs) var(--spacing-xs) var(--spacing-xs) calc(var(--spacing-xs) + 20px)',
                    background: 'rgba(0, 0, 0, 0.3)',
                    border: '1px solid var(--color-accent)',
                    borderRadius: 'var(--border-radius-sm)',
                    color: 'var(--color-text)',
                    fontSize: 'var(--font-size-xs)',
                    fontFamily: 'var(--font-family-mono)',
                    outline: 'none',
                    transition: 'border-color 0.2s'
                  }}
                  onFocus={(e) => e.currentTarget.style.borderColor = 'var(--color-success)'}
                  onBlur={(e) => e.currentTarget.style.borderColor = 'var(--color-accent)'}
                />
              </div>
            )}
          </div>

          {scriptsExpanded && (
            <div style={{ marginTop: 'var(--spacing-md)' }}>              
              <InvestigationScriptsPanel 
                onOpenScriptEditor={onOpenScriptEditor} 
                embedded={true} 
                searchFilter={scriptsSearch}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
