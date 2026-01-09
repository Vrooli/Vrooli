import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ChevronDown, ChevronRight, Search, AlertTriangle, Shield, Bot, Play, MessageCircle, ChevronUp, Square, Clock } from 'lucide-react';
import { InvestigationsPanel } from './InvestigationsPanel';
import { InvestigationScriptsPanel } from './InvestigationScriptsPanel';
import { AutomaticTriggersSection } from './AutomaticTriggersSection';
import type { Investigation, InvestigationScript, InvestigationAgentState } from '../../../types';

interface InvestigationsSectionProps {
  investigations: Investigation[];
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  onSpawnAgent: (options: { autoFix: boolean; note?: string }) => Promise<unknown>;
  agents: InvestigationAgentState[];
  isSpawningAgent: boolean;
  spawnAgentError?: string | null;
}

export const InvestigationsSection = ({ 
  investigations, 
  onOpenScriptEditor,
  onSpawnAgent,
  agents,
  isSpawningAgent,
  spawnAgentError
}: InvestigationsSectionProps) => {
  const navigate = useNavigate();
  const [reportsExpanded, setReportsExpanded] = useState(true);
  const [scriptsExpanded, setScriptsExpanded] = useState(true);
  const [autoFixEnabled, setAutoFixEnabled] = useState(false);
  const [reportsSearch, setReportsSearch] = useState('');
  const [scriptsSearch, setScriptsSearch] = useState('');
  const [agentNote, setAgentNote] = useState('');
  const [showNoteField, setShowNoteField] = useState(false);
  const [localSpawnError, setLocalSpawnError] = useState<string | null>(null);

  // Filter investigations based on search
  const filteredInvestigations = investigations.filter(inv => {
    if (!reportsSearch) return true;
    const searchLower = reportsSearch.toLowerCase();
    return inv.id.toLowerCase().includes(searchLower) ||
           inv.findings?.toLowerCase().includes(searchLower) ||
           inv.status?.toLowerCase().includes(searchLower);
  });

  const combinedSpawnError = localSpawnError ?? spawnAgentError ?? null;

  const activeAgentSummary = useMemo(() => {
    if (agents.length === 0) {
      return {
        text: 'No active investigation agents at the moment.',
        tone: 'idle' as const
      };
    }

    if (agents.length === 1) {
      const agent = agents[0];
      const statusLabel = typeof agent.status === 'string' ? agent.status.toUpperCase() : 'ACTIVE';
      const label = agent.label ?? 'Anomaly investigation';
      const tone = statusLabel === 'ERROR'
        ? 'error'
        : statusLabel === 'COMPLETED'
        ? 'success'
        : 'active';
      return {
        text: `${label} • ${statusLabel}`,
        tone
      };
    }

    const runningCount = agents.filter(agent => {
      const status = agent.status?.toLowerCase?.();
      if (!status) {
        return true;
      }
      return !['completed', 'error', 'failed', 'stopped', 'cancelled', 'canceled'].includes(status);
    }).length;

    return {
      text: `${agents.length} agents in flight (${runningCount} running)`,
      tone: runningCount > 0 ? 'active' as const : 'success' as const
    };
  }, [agents]);

  const summaryAccentColor = activeAgentSummary.tone === 'error'
    ? 'var(--color-error)'
    : activeAgentSummary.tone === 'success'
    ? 'var(--color-success)'
    : activeAgentSummary.tone === 'active'
    ? 'var(--color-accent)'
    : 'var(--color-text-dim)';

  const handleSpawnAgent = async () => {
    const trimmedNote = agentNote.trim();
    setLocalSpawnError(null);
    try {
      await onSpawnAgent({
        autoFix: autoFixEnabled,
        note: trimmedNote ? trimmedNote : undefined
      });
      if (trimmedNote) {
        setAgentNote('');
        setShowNoteField(false);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to spawn agent';
      setLocalSpawnError(message);
    }
  };

  const handleUpdateTrigger = (triggerId: string, config: unknown) => {
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
        background: 'var(--alpha-accent-02)'
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
          background: 'var(--alpha-accent-03)',
          border: '1px solid var(--color-accent)',
          borderRadius: 'var(--border-radius-md)',
          padding: 'var(--spacing-lg)',
          marginBottom: 'var(--spacing-xl)',
          boxShadow: 'var(--shadow-glow)'
        }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-lg)' }}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--spacing-lg)', alignItems: 'flex-start' }}>
              <Bot
                size={48}
                style={{
                  color: 'var(--color-success)',
                  flexShrink: 0,
                  filter: 'drop-shadow(0 0 10px var(--alpha-accent-50))'
                }}
              />

              <div style={{ flex: 1, minWidth: '260px', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
                <div>
                  <h3 style={{
                    margin: '0 0 var(--spacing-sm) 0',
                    color: 'var(--color-text-bright)',
                    fontSize: 'var(--font-size-lg)'
                  }}>
                    System Anomaly Investigation Agent
                  </h3>

                  <p style={{
                    margin: 0,
                    color: 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-sm)',
                    lineHeight: 1.5
                  }}>
                    Launch an autonomous agent to analyze real-time metrics, investigate anomalies, and propose or execute remediation steps. Results and live status now surface in the header for quick access.
                  </p>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 'var(--spacing-sm)',
                    color: 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-xs)'
                  }}>
                    <Square size={10} />
                    <span>Active agents are managed from the header control with a full dropdown of progress and stop actions.</span>
                  </div>

                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 'var(--spacing-sm)',
                    color: summaryAccentColor,
                    fontSize: 'var(--font-size-sm)',
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em'
                  }}>
                    <Clock size={14} />
                    <span>{activeAgentSummary.text}</span>
                  </div>
                </div>
              </div>
            </div>

            {combinedSpawnError && (
              <div style={{
                background: 'rgba(255, 0, 0, 0.1)',
                border: '1px solid var(--color-error)',
                borderRadius: 'var(--border-radius-sm)',
                padding: 'var(--spacing-sm) var(--spacing-md)',
                color: 'var(--color-error)',
                fontSize: 'var(--font-size-sm)'
              }}>
                {combinedSpawnError}
              </div>
            )}

            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-lg)',
              flexWrap: 'wrap'
            }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
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
                    onChange={(event) => setAutoFixEnabled(event.target.checked)}
                    style={{
                      width: '18px',
                      height: '18px',
                      accentColor: 'var(--color-success)',
                      cursor: 'pointer'
                    }}
                  />
                  <Shield size={16} style={{ color: autoFixEnabled ? 'var(--color-success)' : 'var(--color-text-dim)' }} />
                  <span style={{
                    fontWeight: autoFixEnabled ? 'bold' : 'normal',
                    color: autoFixEnabled ? 'var(--color-text-bright)' : 'inherit'
                  }}>
                    Adaptive auto-fix & recovery
                  </span>
                </label>
                <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                  Allows the agent to apply safe remediation steps automatically; otherwise it stays report-only.
                </span>
              </div>

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

              <button
                className="btn btn-primary"
                onClick={handleSpawnAgent}
                disabled={isSpawningAgent}
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
                <Play size={16} className={isSpawningAgent ? 'animate-spin' : ''} />
                {isSpawningAgent ? 'SPAWNING…' : 'SPAWN AGENT'}
              </button>
            </div>

            {showNoteField && (
              <div style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 'var(--spacing-sm)'
              }}>
                <label
                  htmlFor="agent-note"
                  style={{
                    color: 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-xs)',
                    letterSpacing: '0.05em'
                  }}
                >
                  OPTIONAL NOTE FOR AGENT CONTEXT
                </label>
                <textarea
                  id="agent-note"
                  value={agentNote}
                  onChange={(event) => setAgentNote(event.target.value)}
                  rows={3}
                  style={{
                    width: '100%',
                    background: 'rgba(0, 0, 0, 0.4)',
                    border: '1px solid var(--color-accent)',
                    borderRadius: 'var(--border-radius-sm)',
                    color: 'var(--color-text)',
                    padding: 'var(--spacing-sm)',
                    fontFamily: 'var(--font-family-mono)',
                    fontSize: 'var(--font-size-sm)'
                  }}
                />
                <span style={{
                  fontSize: 'var(--font-size-xs)',
                  color: 'var(--color-text-dim)'
                }}>
                  Provide guardrails or extra context. The agent attaches this note to its reasoning and audit trail.
                </span>
              </div>
            )}
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
              background: 'var(--alpha-accent-02)',
              borderBottom: reportsExpanded ? '1px solid var(--color-accent)' : 'none',
              userSelect: 'none',
              transition: 'background 0.2s'
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'var(--alpha-accent-05)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'var(--alpha-accent-02)'}
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
              background: 'var(--alpha-accent-02)',
              borderBottom: scriptsExpanded ? '1px solid var(--color-accent)' : 'none',
              userSelect: 'none',
              transition: 'background 0.2s'
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'var(--alpha-accent-05)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'var(--alpha-accent-02)'}
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
                maxVisible={4}
                onShowAll={() => navigate('/scripts')}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
