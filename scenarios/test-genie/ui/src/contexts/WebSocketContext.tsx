/**
 * WebSocket Context
 * Manages real-time WebSocket connection to agent-manager for agent updates
 */

import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import { fetchAgentManagerWSUrl } from '../lib/api';
import {
  type AgentUpdateData,
  type WebSocketContextValue,
  type WebSocketMessage,
  WebSocketContext,
} from './WebSocketContext.shared';

const MAX_RECONNECT_ATTEMPTS = 5;
const INITIAL_RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 30000;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function readString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined;
}

function readNumber(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined;
}

function mapRunStatus(status: string): AgentUpdateData['status'] {
  switch (status) {
    case 'RUN_STATUS_QUEUED':
    case 'queued':
      return 'pending';
    case 'RUN_STATUS_RUNNING':
    case 'running':
      return 'running';
    case 'RUN_STATUS_COMPLETE':
    case 'complete':
      return 'completed';
    case 'RUN_STATUS_FAILED':
    case 'failed':
      return 'failed';
    case 'RUN_STATUS_CANCELLED':
    case 'cancelled':
      return 'stopped';
    case 'RUN_STATUS_TIMED_OUT':
    case 'timed_out':
      return 'timeout';
    default:
      return 'pending';
  }
}

function mapAgentManagerEvent(event: Record<string, unknown>): WebSocketMessage {
  const eventType = readString(event.type) ?? readString(event.eventType) ?? 'unknown';

  switch (eventType) {
    case 'run_started':
    case 'run_status_changed': {
      const runId = readString(event.runId);
      return {
        type: 'agent_updated',
        data: {
          id: readString(event.tag) ?? runId ?? 'unknown-agent',
          runId,
          status: mapRunStatus(readString(event.status) ?? 'running'),
          startedAt: readString(event.startedAt),
          completedAt: readString(event.completedAt),
        },
        timestamp: Date.now(),
      };
    }

    case 'run_output': {
      const runId = readString(event.runId);
      return {
        type: 'agent_output',
        data: {
          agentId: readString(event.tag) ?? runId ?? 'unknown-agent',
          runId,
          output: readString(event.output) ?? readString(event.content) ?? '',
          sequence: readNumber(event.sequence),
        },
        timestamp: Date.now(),
      };
    }

    case 'run_completed': {
      const runId = readString(event.runId);
      return {
        type: 'agent_updated',
        data: {
          id: readString(event.tag) ?? runId ?? 'unknown-agent',
          runId,
          status: 'completed',
          completedAt: readString(event.completedAt),
          output: readString(event.output),
        },
        timestamp: Date.now(),
      };
    }

    case 'run_failed': {
      const runId = readString(event.runId);
      return {
        type: 'agent_updated',
        data: {
          id: readString(event.tag) ?? runId ?? 'unknown-agent',
          runId,
          status: 'failed',
          completedAt: readString(event.completedAt),
          error: readString(event.error),
        },
        timestamp: Date.now(),
      };
    }

    case 'run_cancelled': {
      const runId = readString(event.runId);
      return {
        type: 'agent_stopped',
        data: {
          id: readString(event.tag) ?? runId ?? 'unknown-agent',
          runId,
          status: 'stopped',
          completedAt: readString(event.completedAt),
        },
        timestamp: Date.now(),
      };
    }

    default:
      return {
        type: eventType || 'unknown',
        data: event,
        timestamp: Date.now(),
      };
  }
}

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const connectRef = useRef<(() => void) | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectDelayRef = useRef(INITIAL_RECONNECT_DELAY);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wsUrlRef = useRef<string | null>(null);
  const subscribedRunsRef = useRef<string[]>([]);

  const attemptReconnect = useCallback(() => {
    if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
      console.error('[WebSocket] Max reconnection attempts reached');
      return;
    }

    reconnectAttemptsRef.current++;
    const delay = reconnectDelayRef.current;

    console.log(
      `[WebSocket] Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current}/${MAX_RECONNECT_ATTEMPTS})`
    );

    reconnectTimeoutRef.current = setTimeout(() => {
      connectRef.current?.();
      reconnectDelayRef.current = Math.min(reconnectDelayRef.current * 2, MAX_RECONNECT_DELAY);
    }, delay);
  }, []);

  const connect = useCallback(async () => {
    if (!wsUrlRef.current) {
      try {
        const response = await fetchAgentManagerWSUrl();
        if (!response.enabled || !response.url) {
          console.log('[WebSocket] Agent manager disabled or URL unavailable');
          return;
        }
        wsUrlRef.current = response.url;
      } catch (error) {
        console.error('[WebSocket] Failed to fetch agent-manager WebSocket URL:', error);
        return;
      }
    }

    const wsUrl = wsUrlRef.current;
    if (!wsUrl) {
      console.error('[WebSocket] URL unavailable');
      return;
    }

    console.log('[WebSocket] Connecting to agent-manager:', wsUrl);

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('[WebSocket] Connected to agent-manager');
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
        reconnectDelayRef.current = INITIAL_RECONNECT_DELAY;

        if (subscribedRunsRef.current.length > 0) {
          ws.send(JSON.stringify({
            type: 'subscribe',
            runIds: subscribedRunsRef.current,
          }));
        }
      };

      ws.onmessage = (event) => {
        try {
          if (typeof event.data !== 'string') {
            console.error('[WebSocket] Ignoring non-string message payload');
            return;
          }
          const rawMessage: unknown = JSON.parse(event.data);
          if (!isRecord(rawMessage)) {
            console.error('[WebSocket] Ignoring non-object message payload');
            return;
          }

          const message = mapAgentManagerEvent(rawMessage);
          console.log('[WebSocket] Message received:', message.type, message.data);
          setLastMessage(message);
        } catch (error) {
          console.error('[WebSocket] Failed to parse message:', error);
        }
      };

      ws.onerror = (error) => {
        console.error('[WebSocket] Error:', error);
      };

      ws.onclose = () => {
        console.log('[WebSocket] Disconnected from agent-manager');
        setIsConnected(false);
        attemptReconnect();
      };
    } catch (error) {
      console.error('[WebSocket] Failed to create connection:', error);
      attemptReconnect();
    }
  }, [attemptReconnect]);

  connectRef.current = () => {
    void connect();
  };

  const send = useCallback((message: unknown) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      console.error('[WebSocket] Cannot send - not connected');
    }
  }, []);

  const reconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    reconnectAttemptsRef.current = 0;
    reconnectDelayRef.current = INITIAL_RECONNECT_DELAY;
    wsUrlRef.current = null;

    void connect();
  }, [connect]);

  const subscribeToRuns = useCallback((runIds: string[]) => {
    subscribedRunsRef.current = runIds;
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN && runIds.length > 0) {
      ws.send(JSON.stringify({
        type: 'subscribe',
        runIds,
      }));
    }
  }, []);

  useEffect(() => {
    void connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  const value: WebSocketContextValue = {
    isConnected,
    lastMessage,
    send,
    reconnect,
    subscribeToRuns,
  };

  return <WebSocketContext.Provider value={value}>{children}</WebSocketContext.Provider>;
}
