import { useEffect, useMemo, useState } from 'react';
import type { Investigation, InvestigationAgentState } from '../../../types';
import { InvestigationStatus, INVESTIGATION_TERMINAL_STATUSES } from '../../../types';
import type { UnknownSpawnOutcome } from './useInvestigationAgents';

interface UseInvestigationsSectionStateParams {
  investigations: Investigation[];
  agents: InvestigationAgentState[];
  spawnAgentError?: string | null;
  incidentNote?: string | null;
  unknownSpawnOutcome?: UnknownSpawnOutcome | null;
  onSpawnAgent: (options: { autoFix: boolean; note?: string }) => Promise<unknown>;
}

export const useInvestigationsSectionState = ({
  investigations,
  agents,
  spawnAgentError,
  incidentNote,
  unknownSpawnOutcome,
  onSpawnAgent,
}: UseInvestigationsSectionStateParams) => {
  const [autoFixEnabled, setAutoFixEnabled] = useState(false);
  const [reportsSearch, setReportsSearch] = useState('');
  const [scriptsSearch, setScriptsSearch] = useState('');
  const [agentNote, setAgentNote] = useState('');
  const [showNoteField, setShowNoteField] = useState(false);
  const [localSpawnError, setLocalSpawnError] = useState<string | null>(null);

  useEffect(() => {
    if (!incidentNote) return;
    setAgentNote(incidentNote);
    setShowNoteField(true);
  }, [incidentNote]);

  const filteredInvestigations = useMemo(() => investigations.filter(inv => {
    if (!reportsSearch) return true;
    const searchLower = reportsSearch.toLowerCase();
    const statusLabel = (InvestigationStatus[inv.status] ?? '').toLowerCase();
    return inv.id.toLowerCase().includes(searchLower) ||
           inv.findings?.toLowerCase().includes(searchLower) ||
           statusLabel.includes(searchLower);
  }), [investigations, reportsSearch]);

  const combinedSpawnError = unknownSpawnOutcome ? null : localSpawnError ?? spawnAgentError ?? null;

  useEffect(() => {
    if (!unknownSpawnOutcome) setLocalSpawnError(null);
  }, [unknownSpawnOutcome]);

  const activeAgentSummary = useMemo(() => {
    if (agents.length === 0) {
      return { text: 'No active agents', tone: 'idle' as const };
    }

    if (agents.length === 1) {
      const agent = agents[0] ?? { status: undefined };
      const statusLabel = typeof agent.status === 'string' ? agent.status : 'active';
      const lower = statusLabel.toLowerCase();
      const tone = lower === 'error'
        ? 'error'
        : lower === 'completed'
        ? 'success'
        : 'active';
      return { text: statusLabel, tone };
    }

    const runningCount = agents.filter(agent => {
      const status = agent.status?.toLowerCase?.();
      if (!status) return true;
      return !INVESTIGATION_TERMINAL_STATUSES.has(status);
    }).length;

    return {
      text: `${agents.length} agents in flight (${runningCount} running)`,
      tone: runningCount > 0 ? 'active' as const : 'success' as const,
    };
  }, [agents]);

  const handleSpawnAgent = async () => {
    const trimmedNote = agentNote.trim();
    setLocalSpawnError(null);
    try {
      await onSpawnAgent({
        autoFix: autoFixEnabled,
        note: trimmedNote ? trimmedNote : undefined,
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

  return {
    // Search state
    reportsSearch,
    setReportsSearch,
    scriptsSearch,
    setScriptsSearch,
    filteredInvestigations,
    // Agent spawn state
    autoFixEnabled,
    setAutoFixEnabled,
    agentNote,
    setAgentNote,
    showNoteField,
    setShowNoteField,
    combinedSpawnError,
    unknownSpawnOutcome: unknownSpawnOutcome ?? null,
    activeAgentSummary,
    handleSpawnAgent,
  };
};
