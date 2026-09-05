// DOC: docs/internal/COHERENCE-NOTES.md#bugs-found
import { useCallback, useEffect, useRef, useState } from 'react';
import { extractErrorMessage, isApiError, protoFetch } from '../../../shared/api/apiFetch';
import { protoToAgentState } from '../../../shared/api/proto-converters';
import { usePolling } from '../../../shared/hooks/usePolling';
import type { InvestigationAgentState } from '../../../types';
import { INVESTIGATION_TERMINAL_STATUSES } from '../../../types/api';

const AGENT_POLL_INTERVAL_MS = 4000;

export const useInvestigationAgents = () => {
  const [agents, setAgents] = useState<InvestigationAgentState[]>([]);
  const agentsRef = useRef(agents);
  agentsRef.current = agents;
  const [isSpawningAgent, setIsSpawningAgent] = useState(false);
  const isSpawningRef = useRef(false);
  const [stoppingAgents, setStoppingAgents] = useState<Set<string>>(() => new Set());
  const [agentErrors, setAgentErrors] = useState<Record<string, string>>({});
  const [spawnAgentError, setSpawnAgentError] = useState<string | null>(null);

  const fetchActiveAgents = useCallback(async () => {
    try {
      const { parseInvestigation } = await import('../../../shared/api/proto-contracts');
      const inv = await protoFetch('/investigations/agent/current', parseInvestigation);
      // Server returns null JSON when no active agent
      if (!inv.id) {
        setAgents([]);
        return;
      }
      const mapped = protoToAgentState(inv);
      setAgents(mapped ? [mapped] : []);
    } catch (err) {
      // Parse failures or empty responses mean no active agent
      if (err instanceof SyntaxError || (isApiError(err) && err.detail?.code === 'not_found')) {
        setAgents([]);
        return;
      }
      // Re-throw server/network errors so callers can handle them
      throw err;
    }
  }, []);

  const spawnInvestigationAgent = useCallback(async ({ autoFix, note }: { autoFix: boolean; note?: string }) => {
    if (isSpawningRef.current) return undefined;
    isSpawningRef.current = true;
    setSpawnAgentError(null);
    setIsSpawningAgent(true);
    try {
      const { parseTriggerInvestigationResponse } = await import('../../../shared/api/proto-contracts');
      const resp = await protoFetch('/investigations/agent/spawn', parseTriggerInvestigationResponse, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ auto_fix: autoFix, note }),
      });

      const mapped: InvestigationAgentState = {
        id: resp.investigationId,
        status: 'queued',
        startTime: new Date().toISOString(),
        autoFix: resp.autoFix,
        note: resp.note || note,
      };

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
      const message = extractErrorMessage(spawnError, 'Unknown error spawning investigation agent');
      setSpawnAgentError(message);
      throw spawnError;
    } finally {
      setIsSpawningAgent(false);
      isSpawningRef.current = false;
    }
  }, []);

  const spawnAgent = useCallback(async ({ autoFix, note }: { autoFix: boolean; note?: string }) => {
    const agent = await spawnInvestigationAgent({ autoFix, note });
    return agent;
  }, [spawnInvestigationAgent]);

  const stopAgent = useCallback(async (agentId: string) => {
    setAgentErrors(prev => {
      if (!(agentId in prev)) {
        return prev;
      }
      const { [agentId]: _omit, ...rest } = prev;
      return rest;
    });

    setStoppingAgents(prev => {
      const next = new Set(prev);
      next.add(agentId);
      return next;
    });

    try {
      await protoFetch<{ status: string; id: string }>(`/investigations/agent/${encodeURIComponent(agentId)}/stop`, data => data as { status: string; id: string }, {
        method: 'POST',
      });
      setAgents(prev => prev.filter(agent => agent.id !== agentId));
    } catch (stopError) {
      const message = extractErrorMessage(stopError, 'Failed to stop agent');
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
    const timeoutID = window.setTimeout(() => {
      fetchActiveAgents().catch(() => {
        // Initial fetch failure is non-fatal — polling will retry
      });
    }, 2200);
    return () => {
      window.clearTimeout(timeoutID);
    };
  }, [fetchActiveAgents]);

  const hasActiveAgents = agents.some(agent => {
    const status = agent.status?.toLowerCase?.();
    return !status || !INVESTIGATION_TERMINAL_STATUSES.has(status);
  });

  const pollAgentStatuses = useCallback(async () => {
    const currentAgents = agentsRef.current.filter(agent => {
      const status = agent.status?.toLowerCase?.();
      return !status || !INVESTIGATION_TERMINAL_STATUSES.has(status);
    });

    if (currentAgents.length === 0) return;

    const removals = new Set<string>();
    const updates = new Map<string, InvestigationAgentState>();

    await Promise.all(currentAgents.map(async agent => {
      try {
        const { parseInvestigation } = await import('../../../shared/api/proto-contracts');
        const inv = await protoFetch(
          `/investigations/agent/${encodeURIComponent(agent.id)}/status`,
          parseInvestigation,
        );
        const mapped = protoToAgentState(inv);
        if (mapped) {
          const normalizedStatus = mapped.status?.toLowerCase?.();
          if (normalizedStatus && INVESTIGATION_TERMINAL_STATUSES.has(normalizedStatus)) {
            removals.add(mapped.id);
          } else {
            updates.set(mapped.id, mapped);
          }
        }
      } catch (pollErr) {
        // 404 / parse error means agent was removed — treat as removal
        if (pollErr instanceof SyntaxError || (isApiError(pollErr) && pollErr.detail?.code === 'not_found')) {
          removals.add(agent.id);
        }
        // Network/5xx errors are silently ignored per-agent during polling
        // (the polling backoff in usePolling will handle systemic failures)
      }
    }));

    if (removals.size > 0 || updates.size > 0) {
      setAgents(prev => prev
        .filter(existing => !removals.has(existing.id))
        .map(existing => {
          const update = updates.get(existing.id);
          return update ? { ...existing, ...update } : existing;
        })
      );
    }
  }, []);

  // Initial poll when agents become active
  useEffect(() => {
    if (hasActiveAgents) {
      void pollAgentStatuses();
    }
  }, [hasActiveAgents, pollAgentStatuses]);

  usePolling(pollAgentStatuses, AGENT_POLL_INTERVAL_MS, hasActiveAgents);

  return {
    agents,
    isSpawningAgent,
    spawnAgentError,
    stoppingAgentIds: stoppingAgents,
    agentErrors,
    refreshAgents: fetchActiveAgents,
    spawnAgent,
    stopAgent,
  };
};
