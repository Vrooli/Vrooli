import { useCallback, useEffect, useRef, useState } from 'react';
import { apiFetch, protoFetch } from '../../../shared/api/apiFetch';
import {
  parseInvestigation,
  parseTriggerInvestigationResponse,
} from '../../../shared/api/proto-contracts';
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
      const inv = await protoFetch('/investigations/agent/current', parseInvestigation);
      // Server returns null JSON when no active agent
      if (!inv.id) {
        setAgents([]);
        return;
      }
      const mapped = protoToAgentState(inv);
      setAgents(mapped ? [mapped] : []);
    } catch {
      // 404 or parse failure means no active agent
      setAgents([]);
    }
  }, []);

  const spawnInvestigationAgent = useCallback(async ({ autoFix, note }: { autoFix: boolean; note?: string }) => {
    if (isSpawningRef.current) return undefined;
    isSpawningRef.current = true;
    setSpawnAgentError(null);
    setIsSpawningAgent(true);
    try {
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
      const message = spawnError instanceof Error ? spawnError.message : 'Unknown error spawning investigation agent';
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
      await apiFetch<{ status: string; id: string }>(`/investigations/agent/${encodeURIComponent(agentId)}/stop`, {
        method: 'POST',
      });
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
      } catch {
        // 404 means agent was removed
        removals.add(agent.id);
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
