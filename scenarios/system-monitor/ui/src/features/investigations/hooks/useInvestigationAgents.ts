// DOC: docs/internal/SEAMS.md#ui-lifecycle-and-shared-utility-seams
import { useCallback, useEffect, useRef, useState } from 'react';
import { extractErrorMessage, isApiError, protoFetch } from '../../../shared/api/apiFetch';
import { protoToAgentState } from '../../../shared/api/proto-converters';
import { usePolling } from '../../../shared/hooks/usePolling';
import type { InvestigationAgentState } from '../../../types';
import { INVESTIGATION_TERMINAL_STATUSES } from '../../../types/api';

const AGENT_POLL_INTERVAL_MS = 4000;
const PENDING_INVESTIGATION_STORAGE_KEY = 'system-monitor.pending-investigation-request.v1';

interface PendingInvestigationRequest {
  requestKey: string;
  autoFix: boolean;
  note: string;
}

const readPendingInvestigationRequest = (): PendingInvestigationRequest | null => {
  try {
    const raw = window.localStorage.getItem(PENDING_INVESTIGATION_STORAGE_KEY);
    if (!raw) return null;
    const value: unknown = JSON.parse(raw);
    if (!value || typeof value !== 'object') return null;
    const request = value as Partial<PendingInvestigationRequest>;
    if (typeof request.requestKey !== 'string' || typeof request.autoFix !== 'boolean' || typeof request.note !== 'string') return null;
    return { requestKey: request.requestKey, autoFix: request.autoFix, note: request.note };
  } catch {
    return null;
  }
};

const persistPendingInvestigationRequest = (request: PendingInvestigationRequest | null): boolean => {
  try {
    if (request) window.localStorage.setItem(PENDING_INVESTIGATION_STORAGE_KEY, JSON.stringify(request));
    else window.localStorage.removeItem(PENDING_INVESTIGATION_STORAGE_KEY);
    return true;
  } catch {
    return false;
  }
};

export interface UnknownSpawnOutcome {
  requestId: string;
  message: string;
  readback: 'checking' | 'ready' | 'failed';
  currentAgent: InvestigationAgentState | null;
  readbackError?: string;
}

export const useInvestigationAgents = (options?: { onReceipt?: () => void }) => {
  const onReceiptRef = useRef(options?.onReceipt);
  onReceiptRef.current = options?.onReceipt;
  const pendingRequestRef = useRef<PendingInvestigationRequest | null>(readPendingInvestigationRequest());
  const [agents, setAgents] = useState<InvestigationAgentState[]>([]);
  const agentsRef = useRef(agents);
  agentsRef.current = agents;
  const [isSpawningAgent, setIsSpawningAgent] = useState(false);
  const isSpawningRef = useRef(false);
  const [stoppingAgents, setStoppingAgents] = useState<Set<string>>(() => new Set());
  const [agentErrors, setAgentErrors] = useState<Record<string, string>>({});
  const [spawnAgentError, setSpawnAgentError] = useState<string | null>(null);
  const [unknownSpawnOutcome, setUnknownSpawnOutcome] = useState<UnknownSpawnOutcome | null>(() => {
    const pending = pendingRequestRef.current;
    return pending ? {
      requestId: pending.requestKey,
      message: 'A previous investigation request has no receipt yet. Retry the same request safely to recover its original receipt.',
      readback: 'checking',
      currentAgent: null,
    } : null;
  });
  const unknownSpawnRef = useRef<UnknownSpawnOutcome | null>(null);
  unknownSpawnRef.current = unknownSpawnOutcome;

  const fetchActiveAgents = useCallback(async () => {
    try {
      const { parseInvestigation } = await import('../../../shared/api/proto-contracts');
      const inv = await protoFetch('/investigations/agent/current', parseInvestigation);
      // Server returns null JSON when no active agent
      if (!inv.id) {
        setAgents([]);
        return [];
      }
      const mapped = protoToAgentState(inv);
      const next = mapped ? [mapped] : [];
      setAgents(next);
      return next;
    } catch (err) {
      // Only the endpoint's explicit not-found result proves no active agent.
      if (isApiError(err) && err.detail?.code === 'not_found') {
        setAgents([]);
        return [];
      }
      // Re-throw server/network errors so callers can handle them
      throw err;
    }
  }, []);

  const submitInvestigationRequest = useCallback(async (pending: PendingInvestigationRequest, allowUnknownOutcome: boolean) => {
    if (isSpawningRef.current || (unknownSpawnRef.current && !allowUnknownOutcome)) return undefined;
    isSpawningRef.current = true;
    setSpawnAgentError(null);
    setIsSpawningAgent(true);
    try {
      const { parseTriggerInvestigationResponse } = await import('../../../shared/api/proto-contracts');
      const resp = await protoFetch('/investigations/agent/spawn', parseTriggerInvestigationResponse, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': pending.requestKey },
        body: JSON.stringify({ auto_fix: pending.autoFix, note: pending.note || undefined }),
      });
      if (!resp.investigationId) throw new Error('The server response did not include an investigation ID. Retry the saved request to recover its receipt.');

      const mapped: InvestigationAgentState = {
        id: resp.investigationId,
        status: resp.status || 'queued',
        startTime: '',
        autoFix: resp.autoFix,
        note: resp.note || pending.note,
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

      pendingRequestRef.current = null;
      persistPendingInvestigationRequest(null);
      unknownSpawnRef.current = null;
      setUnknownSpawnOutcome(null);
      onReceiptRef.current?.();

      return mapped;
    } catch (spawnError) {
      const message = extractErrorMessage(spawnError, 'Unknown error spawning investigation agent');
      setSpawnAgentError(message);
      const knownRejection = isApiError(spawnError) && spawnError.detail?.code === 'validation';
      if (knownRejection) {
        pendingRequestRef.current = null;
        persistPendingInvestigationRequest(null);
        unknownSpawnRef.current = null;
        setUnknownSpawnOutcome(null);
      } else {
        const unknown: UnknownSpawnOutcome = {
          requestId: pending.requestKey,
          message,
          readback: 'checking',
          currentAgent: null,
        };
        unknownSpawnRef.current = unknown;
        setUnknownSpawnOutcome(unknown);
        try {
          const current = await fetchActiveAgents();
          const reconciled = { ...unknown, readback: 'ready' as const, currentAgent: current[0] ?? null };
          unknownSpawnRef.current = reconciled;
          setUnknownSpawnOutcome(reconciled);
        } catch (readbackError) {
          const unreconciled = {
            ...unknown,
            readback: 'failed' as const,
            readbackError: extractErrorMessage(readbackError, 'Could not read current agent status.'),
          };
          unknownSpawnRef.current = unreconciled;
          setUnknownSpawnOutcome(unreconciled);
        }
      }
      throw spawnError;
    } finally {
      setIsSpawningAgent(false);
      isSpawningRef.current = false;
    }
  }, [fetchActiveAgents]);

  const spawnInvestigationAgent = useCallback(async ({ autoFix, note }: { autoFix: boolean; note?: string }) => {
    if (isSpawningRef.current || unknownSpawnRef.current) return undefined;
    const pending = {
      requestKey: crypto.randomUUID(),
      autoFix,
      note: note ?? '',
    };
    if (!persistPendingInvestigationRequest(pending)) {
      setSpawnAgentError('This browser could not save a safe retry reference, so the investigation was not sent. Check browser storage access and try again.');
      return undefined;
    }
    pendingRequestRef.current = pending;
    return submitInvestigationRequest(pending, false);
  }, [submitInvestigationRequest]);

  const retryUnknownSpawn = useCallback(async () => {
    const pending = pendingRequestRef.current;
    if (!pending || isSpawningRef.current) return undefined;
    return submitInvestigationRequest(pending, true);
  }, [submitInvestigationRequest]);

  const refreshSpawnStatus = useCallback(async () => {
    const currentOutcome = unknownSpawnRef.current;
    if (!currentOutcome) return;
    const checking = { ...currentOutcome, readback: 'checking' as const, readbackError: undefined };
    unknownSpawnRef.current = checking;
    setUnknownSpawnOutcome(checking);
    try {
      const current = await fetchActiveAgents();
      const reconciled = { ...checking, readback: 'ready' as const, currentAgent: current[0] ?? null };
      unknownSpawnRef.current = reconciled;
      setUnknownSpawnOutcome(reconciled);
    } catch (readbackError) {
      const failed = {
        ...checking,
        readback: 'failed' as const,
        readbackError: extractErrorMessage(readbackError, 'Could not read current agent status.'),
      };
      unknownSpawnRef.current = failed;
      setUnknownSpawnOutcome(failed);
    }
  }, [fetchActiveAgents]);

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
      const { parseStopAgentResponse } = await import('../../../shared/api/proto-contracts');
      await protoFetch(`/investigations/agent/${encodeURIComponent(agentId)}/stop`, parseStopAgentResponse, {
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

  useEffect(() => {
    if (pendingRequestRef.current) void refreshSpawnStatus();
  }, [refreshSpawnStatus]);

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
    unknownSpawnOutcome,
    refreshSpawnStatus,
    retryUnknownSpawn,
    stoppingAgentIds: stoppingAgents,
    agentErrors,
    refreshAgents: fetchActiveAgents,
    spawnAgent,
    stopAgent,
  };
};
