import { useCallback, useEffect, useRef, useState } from 'react';
import { buildApiUrl } from '../../../shared/api/apiBase';
import { usePolling } from '../../../shared/hooks/usePolling';
import { extractBoolean, extractNumber, extractString } from '../../../shared/utils/typeGuards';
import type { InvestigationAgentState } from '../../../types';

const TERMINAL_AGENT_STATUSES = new Set(['completed', 'error', 'failed', 'cancelled', 'canceled', 'stopped']);
const AGENT_POLL_INTERVAL_MS = 4000;

const mapAgentPayload = (payload: unknown): InvestigationAgentState | null => {
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
};

const parseAgentsResponse = (data: unknown): InvestigationAgentState[] => {
  if (!data) {
    return [];
  }

  const candidates: unknown[] = [];

  if (Array.isArray(data)) {
    candidates.push(...(data as unknown[]));
  } else if (typeof data === 'object' && data !== null) {
    const maybeObject = data as Record<string, unknown>;
    if (Array.isArray(maybeObject.agents)) {
      candidates.push(...(maybeObject.agents as unknown[]));
    } else if (maybeObject.agent) {
      candidates.push(maybeObject.agent);
    } else if (typeof maybeObject.id === 'string' || typeof maybeObject.investigation_id === 'string') {
      candidates.push(maybeObject);
    }
  }

  return candidates
    .map(candidate => mapAgentPayload(candidate))
    .filter((agent): agent is InvestigationAgentState => Boolean(agent));
};

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
  }, []);

  const spawnInvestigationAgent = useCallback(async ({ autoFix, note }: { autoFix: boolean; note?: string }) => {
    if (isSpawningRef.current) return undefined;
    isSpawningRef.current = true;
    setSpawnAgentError(null);
    setIsSpawningAgent(true);
    try {
      const response = await fetch(buildApiUrl('/investigations/agent/spawn'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ auto_fix: autoFix, note })
      });

      if (!response.ok) {
        let message = `Request failed with status ${response.status}`;
        try {
          const errorPayload = (await response.json()) as Record<string, unknown>;
          if (typeof errorPayload.error === 'string') {
            message = errorPayload.error;
          }
        } catch {
          // Ignore JSON parse failure for error payloads
        }
        throw new Error(message);
      }

      let data: Record<string, unknown> = {};
      try {
        data = (await response.json()) as Record<string, unknown>;
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

  const hasActiveAgents = agents.some(agent => {
    const status = agent.status?.toLowerCase?.();
    return !status || !TERMINAL_AGENT_STATUSES.has(status);
  });

  const pollAgentStatuses = useCallback(async () => {
    const currentAgents = agentsRef.current.filter(agent => {
      const status = agent.status?.toLowerCase?.();
      return !status || !TERMINAL_AGENT_STATUSES.has(status);
    });

    if (currentAgents.length === 0) return;

    const removals = new Set<string>();
    const updates = new Map<string, InvestigationAgentState>();

    await Promise.all(currentAgents.map(async agent => {
      try {
        const response = await fetch(buildApiUrl(`/investigations/agent/${encodeURIComponent(agent.id)}/status`));
        if (response.status === 404) {
          removals.add(agent.id);
          return;
        }
        if (!response.ok) return;

        let payload: unknown = null;
        try {
          payload = await response.json();
        } catch (parseError) {
          console.error('Failed to parse agent status payload:', parseError);
        }

        const parsed = parseAgentsResponse(payload);
        const mapped = parsed.find(item => item.id === agent.id)
          ?? mapAgentPayload({ ...(payload as Record<string, unknown>), id: agent.id });

        if (mapped) {
          const normalizedStatus = mapped.status?.toLowerCase?.();
          if (normalizedStatus && TERMINAL_AGENT_STATUSES.has(normalizedStatus)) {
            removals.add(mapped.id);
          } else {
            updates.set(mapped.id, mapped);
          }
        }
      } catch (pollError) {
        console.error('Failed to poll agent status:', pollError);
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
    stopAgent
  };
};
