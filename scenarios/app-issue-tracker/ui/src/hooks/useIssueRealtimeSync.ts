import { useCallback, useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from 'react';
import { buildApiUrl } from '@vrooli/api-base';
import type { Issue } from '../data/sampleData';
import type { RateLimitStatusPayload } from '../services/issues';
import { normalizeStatus, transformIssue } from '../utils/issues';
import type { WebSocketEvent } from '../types/events';
import { useWebSocket, type ConnectionStatus } from './useWebSocket';

export type RunningProcessMap = Map<string, { agent_id: string; start_time: string; duration?: string; status?: string }>;

export interface RunningProcessSeed {
  processes: Array<{ issue_id: string; agent_id: string; start_time: string; status?: string }>;
  version: number;
}

interface UseIssueRealtimeSyncOptions {
  apiBaseUrl: string;
  updateIssuesState: (
    updater: (previous: Issue[]) => Issue[],
    options?: { invalidateRemoteStats?: boolean },
  ) => void;
  applyProcessorRealtimeUpdate: (payload: Record<string, unknown>) => void;
  setRateLimitStatus: Dispatch<SetStateAction<RateLimitStatusPayload | null>>;
  reportProcessorError: (message: string) => void;
  enabled?: boolean;
  initialRunningProcesses?: RunningProcessSeed | null;
}

interface UseIssueRealtimeSyncResult {
  runningProcesses: RunningProcessMap;
  removeProcess: (issueId: string) => void;
  connectionStatus: ConnectionStatus;
  websocketError: Error | null;
  reconnectAttempts: number;
}

declare const __API_PORT__: string | undefined;

function formatElapsedDuration(startTime: string): string | undefined {
  const parsed = new Date(startTime);
  if (Number.isNaN(parsed.getTime())) {
    return undefined;
  }

  const now = new Date();
  const durationMs = now.getTime() - parsed.getTime();
  if (durationMs < 0) {
    return undefined;
  }

  const seconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  }
  return `${seconds}s`;
}

function resolveWebSocketUrl(apiBaseUrl: string): string {
  if (typeof window === 'undefined') {
    return '';
  }

  const httpEndpoint = buildApiUrl('/ws', { baseUrl: apiBaseUrl, appendSuffix: false });

  try {
    const absolute = new URL(httpEndpoint);
    absolute.protocol = absolute.protocol === 'https:' ? 'wss:' : 'ws:';
    return absolute.toString();
  } catch {
    try {
      const viaLocation = new URL(httpEndpoint, window.location.href);
      viaLocation.protocol = viaLocation.protocol === 'https:' ? 'wss:' : 'ws:';
      return viaLocation.toString();
    } catch {
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const normalizedPath = httpEndpoint.startsWith('/') ? httpEndpoint : `/${httpEndpoint}`;
      const apiPort = typeof __API_PORT__ === 'string' && __API_PORT__.trim() ? __API_PORT__ : undefined;
      const host = apiPort ? `${window.location.hostname}:${apiPort}` : window.location.host;
      return `${wsProtocol}//${host}${normalizedPath}`;
    }
  }
}

export function useIssueRealtimeSync({
  apiBaseUrl,
  updateIssuesState,
  applyProcessorRealtimeUpdate,
  setRateLimitStatus,
  reportProcessorError,
  enabled = true,
  initialRunningProcesses = null,
}: UseIssueRealtimeSyncOptions): UseIssueRealtimeSyncResult {
  const [runningProcesses, setRunningProcesses] = useState<RunningProcessMap>(new Map());
  const initialSeedVersionRef = useRef(0);

  useEffect(() => {
    if (!initialRunningProcesses) {
      return;
    }

    if (initialRunningProcesses.version <= initialSeedVersionRef.current) {
      return;
    }

    initialSeedVersionRef.current = initialRunningProcesses.version;

    setRunningProcesses(() => {
      if (!initialRunningProcesses.processes || initialRunningProcesses.processes.length === 0) {
        return new Map();
      }

      const seeded = new Map<string, { agent_id: string; start_time: string; duration?: string; status?: string }>();
      initialRunningProcesses.processes.forEach((process) => {
        if (!process || !process.issue_id) {
          return;
        }
        seeded.set(process.issue_id, {
          agent_id: process.agent_id,
          start_time: process.start_time,
          status: process.status ?? 'running',
          duration: formatElapsedDuration(process.start_time),
        });
      });
      return seeded;
    });
  }, [initialRunningProcesses]);

  const websocketUrl = useMemo(() => {
    const base = resolveWebSocketUrl(apiBaseUrl);
    if (!base || !enabled) {
      return '';
    }
    return base;
  }, [apiBaseUrl, enabled]);

  const handleWebSocketEvent = useCallback(
    (event: WebSocketEvent) => {
      switch (event.type) {
        case 'issue.created':
        case 'issue.updated': {
          const rawIssue = event.data.issue;
          if (!rawIssue) {
            return;
          }
          const updatedIssue = transformIssue(rawIssue, { apiBaseUrl });
          updateIssuesState((previous) => {
            const index = previous.findIndex((issue) => issue.id === updatedIssue.id);
            if (index >= 0) {
              const next = [...previous];
              next[index] = updatedIssue;
              return next;
            }
            return [...previous, updatedIssue];
          });
          break;
        }
        case 'issue.status_changed': {
          const { issue_id, new_status } = event.data;
          updateIssuesState((previous) =>
            previous.map((issue) =>
              issue.id === issue_id ? { ...issue, status: normalizeStatus(new_status) } : issue,
            ),
          );
          break;
        }
        case 'issue.deleted': {
          const { issue_id } = event.data;
          updateIssuesState((previous) => previous.filter((issue) => issue.id !== issue_id));
          setRunningProcesses((previous) => {
            if (!previous.has(issue_id)) {
              return previous;
            }
            const next = new Map(previous);
            next.delete(issue_id);
            return next;
          });
          break;
        }
        case 'agent.started': {
          const { issue_id, agent_id, start_time } = event.data;
          setRunningProcesses((previous) => {
            const existing = previous.get(issue_id);
            if (existing && existing.agent_id === agent_id && existing.start_time === start_time) {
              if (existing.status === 'running') {
                return previous;
              }
              const next = new Map(previous);
              next.set(issue_id, { ...existing, status: 'running' });
              return next;
            }
            const next = new Map(previous);
            next.set(issue_id, { agent_id, start_time, status: 'running' });
            return next;
          });
          break;
        }
        case 'agent.completed':
        case 'agent.failed': {
          const { issue_id, new_status } = event.data;
          setRunningProcesses((previous) => {
            if (!previous.has(issue_id)) {
              return previous;
            }
            const next = new Map(previous);
            next.delete(issue_id);
            return next;
          });
          if (new_status) {
            updateIssuesState((previous) =>
              previous.map((issue) =>
                issue.id === issue_id
                  ? {
                      ...issue,
                      status: normalizeStatus(new_status),
                    }
                  : issue,
              ),
            );
          }
          break;
        }
        case 'processor.state_changed': {
          const data = event.data ?? {};
          applyProcessorRealtimeUpdate(data as Record<string, unknown>);
          break;
        }
        case 'rate_limit.changed': {
          setRateLimitStatus((previous) => ({
            rate_limited: Boolean(event.data.rate_limited),
            rate_limited_count: Number(
              event.data.rate_limited_count ?? (typeof previous?.rate_limited_count === 'number' ? previous.rate_limited_count : 0),
            ),
            seconds_until_reset: Number(
              event.data.seconds_until_reset ?? (typeof previous?.seconds_until_reset === 'number' ? previous.seconds_until_reset : 0),
            ),
            reset_time: event.data.reset_time ?? previous?.reset_time ?? new Date().toISOString(),
            rate_limit_agent: event.data.rate_limit_agent ?? previous?.rate_limit_agent ?? 'unknown',
          }));
          break;
        }
        case 'processor.error': {
          const message = typeof event.data?.message === 'string' ? event.data.message : 'Automation error';
          reportProcessorError(message);
          break;
        }
        default:
          break;
      }
    },
    [apiBaseUrl, applyProcessorRealtimeUpdate, reportProcessorError, setRateLimitStatus, updateIssuesState],
  );

  const { status: connectionStatus, error: websocketError, reconnectAttempts } = useWebSocket({
    url: websocketUrl,
    onEvent: handleWebSocketEvent,
    enabled: typeof window !== 'undefined' && Boolean(websocketUrl),
  });

  useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined;
    }

    const updateDurations = () => {
      setRunningProcesses((previous) => {
        if (previous.size === 0) {
          return previous;
        }

        const next = new Map(previous);
        let changed = false;

        for (const [issueId, process] of next.entries()) {
          const duration = formatElapsedDuration(process.start_time);
          if (process.duration !== duration) {
            next.set(issueId, { ...process, duration });
            changed = true;
          }
        }

        return changed ? next : previous;
      });
    };

    const intervalId = window.setInterval(updateDurations, 1000);
    return () => window.clearInterval(intervalId);
  }, []);

  const removeProcess = useCallback((issueId: string) => {
    setRunningProcesses((previous) => {
      if (!previous.has(issueId)) {
        return previous;
      }
      const next = new Map(previous);
      next.delete(issueId);
      return next;
    });
  }, []);

  return {
    runningProcesses,
    removeProcess,
    connectionStatus,
    websocketError,
    reconnectAttempts,
  };
}

export type { ConnectionStatus } from './useWebSocket';
