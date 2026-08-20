import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Brain, Loader2, RefreshCcw, ChevronDown, Clock, AlertTriangle, CheckCircle2, XCircle, Square } from 'lucide-react';
import { useClickOutside } from '../hooks/useClickOutside';
import { useEscapeKey } from '../../hooks/useEscapeKey';
import type { InvestigationAgentState } from '../../types';

interface AgentDropdownProps {
  agents: InvestigationAgentState[];
  stoppingAgentIds: ReadonlySet<string>;
  agentErrors: Record<string, string>;
  onStopAgent: (agentId: string) => Promise<void>;
  onRefreshAgents?: () => void;
}

const terminalStatuses = new Set(['completed', 'error', 'failed', 'stopped', 'cancelled', 'canceled']);

const normalizeStatus = (status?: string): string => {
  if (!status) {
    return '';
  }
  return status.toLowerCase();
};

import { formatDurationElapsed } from '../utils/formatters';

const statusClass = (status: string): string => {
  switch (status) {
    case 'error':
    case 'failed':
      return 'text-error';
    case 'completed':
      return 'text-success';
    case 'initializing':
    case 'analyzing':
      return 'text-warning';
    case 'investigating':
    case 'running':
    default:
      return 'text-accent';
  }
};

const statusIconFor = (status: string) => {
  const toneClass = statusClass(status);
  if (status === 'completed') {
    return <CheckCircle2 size={14} className={toneClass} />;
  }
  if (status === 'error' || status === 'failed') {
    return <XCircle size={14} className={toneClass} />;
  }
  if (status === 'initializing' || status === 'investigating' || status === 'analyzing') {
    return <Loader2 size={14} className={`animate-spin ${toneClass}`} />;
  }
  return <AlertTriangle size={14} className={toneClass} />;
};

export const AgentDropdown = ({
  agents,
  stoppingAgentIds,
  agentErrors,
  onStopAgent,
  onRefreshAgents
}: AgentDropdownProps) => {
  const [agentsOpen, setAgentsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement | null>(null);
  const closeAgents = useCallback(() => { setAgentsOpen(false); }, []);

  useClickOutside(dropdownRef, closeAgents, agentsOpen);
  useEscapeKey(closeAgents, agentsOpen);

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
    <div ref={dropdownRef} className="agent-dropdown">
      <button
        type="button"
        onClick={() => { setAgentsOpen(prev => !prev); }}
        aria-expanded={agentsOpen}
        aria-haspopup="true"
        className={`agent-dropdown-btn ${buttonTone !== 'idle' ? 'active' : ''} agent-dropdown-${buttonTone}`}
      >
        <Brain size={16} />
        <span>{agentButtonLabel}</span>
        <ChevronDown size={14} className={`agent-dropdown-chevron ${agentsOpen ? 'is-open' : ''}`} />
      </button>

      {agentsOpen && (
        <div className="agent-dropdown-panel">
          <div className="text-dim-xs agent-dropdown-header">
            <span>ACTIVE AGENTS</span>
            {onRefreshAgents && (
              <button
                type="button"
                onClick={(event) => {
                  event.stopPropagation();
                  onRefreshAgents();
                }}
                className="agent-dropdown-refresh"
                title="Refresh agent status"
              >
                <RefreshCcw size={14} />
              </button>
            )}
          </div>

          {totalCount === 0 ? (
            <div className="agent-dropdown-empty">
              No investigation agents are running.
            </div>
          ) : (
            <div className="flex-col">
              {sortedAgents.map(agent => {
                const normalized = normalizeStatus(agent.status);
                const isStopping = stoppingAgentIds.has(agent.id);
                const errorMessage = agentErrors[agent.id];
                const isTerminalStatus = terminalStatuses.has(normalized);
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
                  <div key={agent.id} className="agent-item">
                    <div className="flex-row-between">
                      <div className="agent-item-title">
                        {agent.label ?? `Investigation ${agent.id}`}
                      </div>
                      <div className={`icon-text icon-text-xs text-xs ${statusClass(normalized)}`}>
                        {statusIconFor(normalized)}
                        <span data-sm-style="sm-style-df8e8add6b">
                          {agent.status ?? 'UNKNOWN'}
                        </span>
                      </div>
                    </div>

                    <div className="text-dim-xs agent-item-meta">
                      <div className="icon-text icon-text-xs">
                        <Clock size={12} />
                        <span>{formatDurationElapsed(agent.startTime)}</span>
                      </div>
                      <div data-sm-style="sm-style-f5bb675600">
                        <span>Mode: {agent.operationMode ?? 'report-only'}</span>
                        <span>Auto-fix: {agent.autoFix ? 'enabled' : 'off'}</span>
                        {agent.model && <span>Model: {agent.model}</span>}
                      </div>
                      {agent.note && (
                        <div className="agent-item-note">
                          "{agent.note}"
                        </div>
                      )}
                    </div>

                    {errorMessage && (
                      <div className="text-xs text-error">
                        {errorMessage}
                      </div>
                    )}

                    <div data-sm-style="sm-style-b8ac32c4c4">
                      <button
                        type="button"
                        onClick={(event) => { void handleStopClick(event, agent.id); }}
                        disabled={isStopping || isTerminalStatus}
                        className={`icon-text icon-text-xs agent-stop-btn ${isTerminalStatus ? 'is-terminal' : 'is-actionable'} ${isStopping ? 'is-stopping' : ''}`}
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
  );
};
