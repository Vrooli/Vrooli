import { useState, useEffect } from 'react';
import { ChevronDown, ChevronRight, Search, AlertTriangle, Shield, Bot, Play, MessageCircle, ChevronUp, Square, Clock, CheckCircle, AlertCircle } from 'lucide-react';
import { InvestigationsPanel } from './InvestigationsPanel';
import { InvestigationScriptsPanel } from './InvestigationScriptsPanel';
import { AutomaticTriggersSection } from './AutomaticTriggersSection';
import type { Investigation, InvestigationScript } from '../../types';

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
  const [agentData, setAgentData] = useState<{
    id: string;
    status: 'initializing' | 'investigating' | 'analyzing' | 'completed' | 'error';
    startTime: string;
    autoFix: boolean;
    note?: string;
  } | null>(null);
  const [statusPolling, setStatusPolling] = useState(false);

  // Filter investigations based on search
  const filteredInvestigations = investigations.filter(inv => {
    if (!reportsSearch) return true;
    const searchLower = reportsSearch.toLowerCase();
    return inv.id.toLowerCase().includes(searchLower) ||
           inv.findings?.toLowerCase().includes(searchLower) ||
           inv.status?.toLowerCase().includes(searchLower);
  });

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
            setAgentData(prev => prev ? { ...prev, status: statusData.status } : null);
            
            // Stop polling if agent finished
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
          setAgentData(data.agent);
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
        setAgentData({
          id: agentResponse.investigation_id,
          status: agentResponse.status || 'initializing',
          startTime: new Date().toISOString(),
          autoFix: autoFixEnabled,
          note: agentNote.trim() || undefined
        });
        
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
                    Auto-fix safe issues
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
                  <div className="agent-status-display" style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 'var(--spacing-md)',
                    padding: 'var(--spacing-sm) var(--spacing-lg)',
                    background: agentData.status === 'error' ? 'rgba(255, 0, 0, 0.1)' : 
                              agentData.status === 'completed' ? 'rgba(0, 255, 0, 0.1)' : 
                              'rgba(0, 255, 0, 0.1)',
                    border: agentData.status === 'error' ? '1px solid var(--color-error)' : 
                           agentData.status === 'completed' ? '1px solid var(--color-success)' :
                           '1px solid var(--color-success)',
                    borderRadius: 'var(--border-radius-md)',
                    minWidth: '300px'
                  }}>
                    {/* Status Icon */}
                    <div style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 'var(--spacing-xs)'
                    }}>
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
                    
                    {/* Elapsed Time */}
                    <div style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 'var(--spacing-xs)'
                    }}>
                      <Clock size={14} style={{ color: 'var(--color-text-dim)' }} />
                      <span style={{
                        color: 'var(--color-text)',
                        fontSize: 'var(--font-size-sm)',
                        fontFamily: 'var(--font-family-mono)'
                      }}>
                        {formatElapsedTime(agentData.startTime)}
                      </span>
                    </div>
                    
                    {/* Polling indicator */}
                    {statusPolling && (
                      <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 'var(--spacing-xs)'
                      }}>
                        <div style={{
                          width: '8px',
                          height: '8px',
                          borderRadius: '50%',
                          background: 'var(--color-success)',
                          animation: 'pulse 1.5s infinite'
                        }} />
                        <span style={{
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-xs)'
                        }}>
                          LIVE
                        </span>
                      </div>
                    )}
                    
                    {/* Stop Button - only show if agent is still running */}
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
                        onMouseEnter={(e) => {
                          e.currentTarget.style.background = 'rgba(255, 0, 0, 0.2)';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.background = 'rgba(255, 0, 0, 0.1)';
                        }}
                      >
                        <Square size={12} />
                        STOP
                      </button>
                    )}
                    
                    {/* Clear completed/error status */}
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
                )}
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
                  <strong>Auto-fix mode:</strong> The agent will automatically resolve issues it deems safe to fix, 
                  such as clearing cache, restarting stuck services, or optimizing configurations. 
                  Destructive operations will never be performed automatically.
                </div>
              )}
            </div>
          </div>
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