import { useState } from 'react';
import { ChevronDown, ChevronRight, Search, AlertTriangle, Shield, Bot, Play } from 'lucide-react';
import { InvestigationsPanel } from './InvestigationsPanel';
import { InvestigationScriptsPanel } from './InvestigationScriptsPanel';
import type { Investigation, InvestigationScript } from '../../types';

interface InvestigationsSectionProps {
  investigations: Investigation[];
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  onSpawnAgent: (autoFix: boolean) => void;
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

  // Filter investigations based on search
  const filteredInvestigations = investigations.filter(inv => {
    if (!reportsSearch) return true;
    const searchLower = reportsSearch.toLowerCase();
    return inv.id.toLowerCase().includes(searchLower) ||
           inv.findings?.toLowerCase().includes(searchLower) ||
           inv.status?.toLowerCase().includes(searchLower);
  });

  const handleSpawnAgent = async () => {
    setIsSpawning(true);
    try {
      await onSpawnAgent(autoFixEnabled);
    } finally {
      // Keep spinning for visual effect
      setTimeout(() => setIsSpawning(false), 2000);
    }
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
                flexWrap: 'wrap'
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

                {/* Spawn Agent Button */}
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
              </div>

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