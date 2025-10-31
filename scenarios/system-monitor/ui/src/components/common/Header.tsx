import { useEffect, useMemo, useRef, useState } from 'react';
import { Settings, Terminal, Brain, Loader2, RefreshCcw, ChevronDown, Clock, AlertTriangle, CheckCircle2, XCircle, Square } from 'lucide-react';
import { StatusIndicator } from './StatusIndicator';
import type { InvestigationAgentState } from '../../types';

interface HeaderProps {
  isOnline: boolean;
  unreadErrorCount: number;
  agents: InvestigationAgentState[];
  onStopAgent: (agentId: string) => Promise<void>;
  stoppingAgentIds: ReadonlySet<string>;
  agentErrors: Record<string, string>;
  onRefreshAgents?: () => void;
  onToggleTerminal: () => void;
  onOpenSettings: () => void;
}

const terminalStatuses = new Set(['completed', 'error', 'failed', 'stopped', 'cancelled', 'canceled']);

const normalizeStatus = (status?: string): string => {
  if (!status) {
    return '';
  }
  return status.toLowerCase();
};

const formatDuration = (startTime: string): string => {
  const start = new Date(startTime);
  if (Number.isNaN(start.getTime())) {
    return 'unknown duration';
  }
  const seconds = Math.max(0, Math.floor((Date.now() - start.getTime()) / 1000));
  if (seconds < 60) {
    return `${seconds}s elapsed`;
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    const remaining = seconds % 60;
    return `${minutes}m ${remaining}s elapsed`;
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m elapsed`;
};

const statusAccent = (status: string): string => {
  switch (status) {
    case 'error':
    case 'failed':
      return 'var(--color-error)';
    case 'completed':
      return 'var(--color-success)';
    case 'initializing':
    case 'analyzing':
      return 'var(--color-warning)';
    case 'investigating':
    case 'running':
    default:
      return 'var(--color-accent)';
  }
};

const statusIconFor = (status: string) => {
  const color = statusAccent(status);
  if (status === 'completed') {
    return <CheckCircle2 size={14} style={{ color }} />;
  }
  if (status === 'error' || status === 'failed') {
    return <XCircle size={14} style={{ color }} />;
  }
  if (status === 'initializing' || status === 'investigating' || status === 'analyzing') {
    return <Loader2 size={14} className="animate-spin" style={{ color }} />;
  }
  return <AlertTriangle size={14} style={{ color }} />;
};

export const Header = ({
  isOnline,
  unreadErrorCount,
  agents,
  onStopAgent,
  stoppingAgentIds,
  agentErrors,
  onRefreshAgents,
  onToggleTerminal,
  onOpenSettings
}: HeaderProps) => {
  const [agentsOpen, setAgentsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      if (!agentsOpen) {
        return;
      }
      const target = event.target as Node;
      if (dropdownRef.current && !dropdownRef.current.contains(target)) {
        setAgentsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClick);
    return () => {
      document.removeEventListener('mousedown', handleClick);
    };
  }, [agentsOpen]);

  useEffect(() => {
    if (!agentsOpen) {
      return;
    }
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setAgentsOpen(false);
      }
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [agentsOpen]);

  useEffect(() => {
    if (agents.length === 0) {
      setAgentsOpen(false);
    }
  }, [agents.length]);

  const sortedAgents = useMemo(() => {
    return [...agents].sort((a, b) => {
      const aTime = new Date(a.startTime).getTime();
      const bTime = new Date(b.startTime).getTime();
      const aInvalid = Number.isNaN(aTime);
      const bInvalid = Number.isNaN(bTime);
      if (aInvalid && bInvalid) {
        return 0;
      }
      if (aInvalid) {
        return 1;
      }
      if (bInvalid) {
        return -1;
      }
      return bTime - aTime;
    });
  }, [agents]);

  const runningCount = useMemo(() => sortedAgents.filter(agent => !terminalStatuses.has(normalizeStatus(agent.status))).length, [sortedAgents]);
  const totalCount = sortedAgents.length;
  const buttonTone = totalCount === 0 ? 'idle' : runningCount > 0 ? 'active' : 'success';
  const buttonAccent = buttonTone === 'active'
    ? 'var(--color-accent)'
    : buttonTone === 'success'
    ? 'var(--color-success)'
    : 'var(--color-text-dim)';

  const agentButtonLabel = totalCount === 0
    ? 'Agents'
    : runningCount > 0
    ? `${totalCount} Active`
    : `${totalCount} Complete`;

  const handleStopClick = async (event: React.MouseEvent<HTMLButtonElement>, agentId: string) => {
    event.stopPropagation();
    try {
      await onStopAgent(agentId);
    } catch (error) {
      console.error('Failed to stop agent:', error);
    }
  };

  return (
    <header style={{
      position: 'sticky',
      top: 0,
      zIndex: 1200,
      background: 'var(--color-surface)',
      borderBottom: '1px solid var(--color-surface-border)',
      boxShadow: '0 10px 30px rgba(1, 7, 4, 0.6)',
      backdropFilter: 'blur(10px)'
    }}>
      <div style={{
        maxWidth: '1400px',
        margin: '0 auto',
        padding: '1rem 2rem',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: '1.5rem'
      }}>
        <h1
          className="system-monitor-title"
          style={{
          margin: 0,
          fontSize: 'var(--font-size-xl)',
          display: 'flex',
          alignItems: 'center'
        }}
        >
          <span style={{ color: 'var(--color-accent)' }}>[</span>
          <span className="text matrix-text-glow system-monitor-title-text">SYSTEM MONITOR</span>
          <span style={{ color: 'var(--color-accent)' }}>]</span>
        </h1>

        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
          <div ref={dropdownRef} style={{ position: 'relative' }}>
            <button
              type="button"
              onClick={() => setAgentsOpen(prev => !prev)}
              aria-expanded={agentsOpen}
              aria-haspopup="true"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: '0.5rem',
                padding: '0.45rem 0.9rem',
                borderRadius: 'var(--border-radius-md)',
                border: `1px solid ${buttonAccent}`,
                background: buttonTone === 'idle' ? 'rgba(7, 25, 16, 0.6)' : 'var(--alpha-accent-12)',
                color: buttonTone === 'idle' ? 'var(--color-text)' : buttonAccent,
                fontSize: 'var(--font-size-sm)',
                textTransform: 'uppercase',
                letterSpacing: '0.08em',
                cursor: 'pointer'
              }}
            >
              <Brain size={16} />
              <span>{agentButtonLabel}</span>
              <ChevronDown size={14} style={{ transform: agentsOpen ? 'rotate(180deg)' : 'none', transition: 'transform 0.2s ease' }} />
            </button>

            {agentsOpen && (
              <div
                style={{
                  position: 'absolute',
                  right: 0,
                  marginTop: '0.75rem',
                  width: '340px',
                  maxHeight: '380px',
                  overflowY: 'auto',
                  background: 'var(--color-surface)',
                  border: '1px solid var(--color-surface-border)',
                  borderRadius: 'var(--border-radius-md)',
                  boxShadow: 'var(--shadow-glow)'
                }}
              >
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '0.75rem 1rem',
                  borderBottom: '1px solid var(--alpha-accent-12)',
                  fontSize: 'var(--font-size-xs)',
                  letterSpacing: '0.1em',
                  color: 'var(--color-text-dim)'
                }}>
                  <span>ACTIVE AGENTS</span>
                  {onRefreshAgents && (
                    <button
                      type="button"
                      onClick={(event) => {
                        event.stopPropagation();
                        onRefreshAgents();
                      }}
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        width: '24px',
                        height: '24px',
                        borderRadius: '50%',
                        border: '1px solid var(--alpha-accent-30)',
                        background: 'rgba(7, 25, 16, 0.6)',
                        color: 'var(--color-text-dim)',
                        cursor: 'pointer'
                      }}
                      title="Refresh agent status"
                    >
                      <RefreshCcw size={14} />
                    </button>
                  )}
                </div>

                {totalCount === 0 ? (
                  <div style={{
                    padding: '1rem',
                    textAlign: 'center',
                    color: 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    No investigation agents are running.
                  </div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column' }}>
                    {sortedAgents.map(agent => {
                      const normalized = normalizeStatus(agent.status);
                      const color = statusAccent(normalized);
                      const isStopping = stoppingAgentIds.has(agent.id);
                      const errorMessage = agentErrors[agent.id];
                      const isTerminalStatus = terminalStatuses.has(normalized);
                      const stopButtonColor = isTerminalStatus ? 'var(--color-text-dim)' : 'var(--color-error)';
                      const stopButtonBackground = isTerminalStatus ? 'rgba(7, 25, 16, 0.55)' : 'rgba(255, 77, 109, 0.14)';
                      const stopButtonBorder = isTerminalStatus
                        ? '1px solid rgba(231, 255, 241, 0.12)'
                        : '1px solid rgba(255, 77, 109, 0.4)';
                      const stopButtonIcon = isStopping
                        ? <Loader2 size={12} className="animate-spin" />
                        : isTerminalStatus
                        ? <CheckCircle2 size={12} />
                        : <Square size={12} />;
                      const stopButtonLabel = isStopping
                        ? 'STOPPING'
                        : isTerminalStatus
                        ? 'CLEARED'
                        : 'STOP';

                      return (
                        <div
                          key={agent.id}
                          style={{
                            padding: '0.85rem 1rem',
                            borderBottom: '1px solid var(--alpha-accent-08)',
                            display: 'flex',
                            flexDirection: 'column',
                            gap: '0.5rem'
                          }}
                        >
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '0.75rem' }}>
                            <div style={{
                              fontSize: 'var(--font-size-sm)',
                              color: 'var(--color-text-bright)',
                              fontWeight: 600,
                              textTransform: 'uppercase',
                              letterSpacing: '0.05em'
                            }}>
                              {agent.label ?? `Investigation ${agent.id}`}
                            </div>
                            <div style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: '0.4rem',
                              fontSize: 'var(--font-size-xs)',
                              color
                            }}>
                              {statusIconFor(normalized)}
                              <span style={{ textTransform: 'uppercase', letterSpacing: '0.08em' }}>
                                {agent.status ?? 'UNKNOWN'}
                              </span>
                            </div>
                          </div>

                          <div style={{
                            display: 'flex',
                            flexDirection: 'column',
                            gap: '0.3rem',
                            fontSize: 'var(--font-size-xs)',
                            color: 'var(--color-text-dim)'
                          }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                              <Clock size={12} />
                              <span>{formatDuration(agent.startTime)}</span>
                            </div>
                            <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
                              <span>Mode: {agent.operationMode ?? 'report-only'}</span>
                              <span>Auto-fix: {agent.autoFix ? 'enabled' : 'off'}</span>
                              {agent.model && <span>Model: {agent.model}</span>}
                            </div>
                            {agent.note && (
                              <div style={{ color: 'var(--color-text)', fontStyle: 'italic' }}>
                                "{agent.note}"
                              </div>
                            )}
                          </div>

                          {errorMessage && (
                            <div style={{
                              fontSize: 'var(--font-size-xs)',
                              color: 'var(--color-error)'
                            }}>
                              {errorMessage}
                            </div>
                          )}

                          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                            <button
                              type="button"
                              onClick={(event) => handleStopClick(event, agent.id)}
                              disabled={isStopping || isTerminalStatus}
                              style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                gap: '0.4rem',
                                padding: '0.35rem 0.75rem',
                                borderRadius: 'var(--border-radius-sm)',
                                border: stopButtonBorder,
                                background: stopButtonBackground,
                                color: stopButtonColor,
                                fontSize: 'var(--font-size-xs)',
                                cursor: isStopping ? 'wait' : 'pointer',
                                opacity: isStopping ? 0.7 : 1
                              }}
                            >
                              {stopButtonIcon}
                              {stopButtonLabel}
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}
          </div>

          <StatusIndicator fallbackOnline={isOnline} />

          <button
            className="header-button icon-button"
            onClick={onOpenSettings}
            type="button"
            title="Open system settings"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: '36px',
              height: '36px',
              borderRadius: 'var(--border-radius-sm)',
              border: '1px solid var(--color-surface-border)',
              background: 'rgba(7, 25, 16, 0.65)',
              color: 'var(--color-text)'
            }}
          >
            <Settings size={16} />
          </button>

          <button
            className="header-button icon-button"
            onClick={onToggleTerminal}
            type="button"
            title="Toggle system output"
            style={{
              position: 'relative',
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: '36px',
              height: '36px',
              borderRadius: 'var(--border-radius-sm)',
              border: '1px solid var(--color-surface-border)',
              background: 'rgba(7, 25, 16, 0.65)',
              color: 'var(--color-text)'
            }}
          >
            <Terminal size={16} />
            {unreadErrorCount > 0 && (
              <span 
                className="notification-badge"
                style={{
                  position: 'absolute',
                  top: '-6px',
                  right: '-6px',
                  minWidth: '18px',
                  height: '18px',
                  borderRadius: '50%',
                  background: 'var(--color-error)',
                  color: '#000',
                  fontSize: '10px',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  padding: '0 4px'
                }}
              >
                {unreadErrorCount}
              </span>
            )}
          </button>
        </div>
      </div>
    </header>
  );
};
