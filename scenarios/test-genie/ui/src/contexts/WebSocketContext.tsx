/**
 * WebSocket Context
 * Manages real-time WebSocket connection to agent-manager for agent updates
 */

import React, { createContext, useContext, useEffect, useRef, useState, useCallback } from 'react';
import { fetchAgentManagerWSUrl } from '../lib/api';

// WebSocket message types from agent-manager
export type WebSocketMessageType =
  | 'connected'
  | 'run_started'
  | 'run_output'
  | 'run_completed'
  | 'run_failed'
  | 'run_cancelled'
  | 'agent_updated'    // Mapped from run events for backwards compatibility
  | 'agent_output'     // Mapped from run_output
  | 'agent_stopped'    // Mapped from run_cancelled
  | 'agents_stopped_all'
  | string;

export interface WebSocketMessage {
  type: WebSocketMessageType;
  data?: Record<string, unknown>;
  message?: string;
  timestamp?: number;
  [key: string]: unknown;
}

export interface AgentUpdateData {
  id: string;
  runId?: string;
  sessionId?: string;
  scenario?: string;
  scope?: string[];
  phases?: string[];
  model?: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'timeout' | 'stopped';
  startedAt?: string;
  completedAt?: string;
  output?: string;
  error?: string;
}

export interface AgentOutputData {
  agentId: string;
  runId?: string;
  output: string;
  sequence?: number;
}

interface WebSocketContextValue {
  isConnected: boolean;
  lastMessage: WebSocketMessage | null;
  send: (message: unknown) => void;
  reconnect: () => void;
  subscribeToRuns: (runIds: string[]) => void;
}

const WebSocketContext = createContext<WebSocketContextValue | undefined>(undefined);

const MAX_RECONNECT_ATTEMPTS = 5;
const INITIAL_RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 30000;

// Map agent-manager run status to test-genie agent status
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

// Map agent-manager WebSocket event to test-genie format
function mapAgentManagerEvent(event: Record<string, unknown>): WebSocketMessage {
  const eventType = event.type as string || event.eventType as string;

  switch (eventType) {
    case 'run_started':
    case 'run_status_changed':
      return {
        type: 'agent_updated',
        data: {
          id: event.tag || event.runId,
          runId: event.runId,
          status: mapRunStatus(event.status as string || 'running'),
          startedAt: event.startedAt,
          completedAt: event.completedAt,
        } as AgentUpdateData,
        timestamp: Date.now(),
      };

    case 'run_output':
      return {
        type: 'agent_output',
        data: {
          agentId: event.tag || event.runId,
          runId: event.runId,
          output: event.output || event.content,
          sequence: event.sequence,
        } as AgentOutputData,
        timestamp: Date.now(),
      };

    case 'run_completed':
      return {
        type: 'agent_updated',
        data: {
          id: event.tag || event.runId,
          runId: event.runId,
          status: 'completed',
          completedAt: event.completedAt,
          output: event.output,
        } as AgentUpdateData,
        timestamp: Date.now(),
      };

    case 'run_failed':
      return {
        type: 'agent_updated',
        data: {
          id: event.tag || event.runId,
          runId: event.runId,
          status: 'failed',
          completedAt: event.completedAt,
          error: event.error,
        } as AgentUpdateData,
        timestamp: Date.now(),
      };

    case 'run_cancelled':
      return {
        type: 'agent_stopped',
        data: {
          id: event.tag || event.runId,
          runId: event.runId,
          status: 'stopped',
          completedAt: event.completedAt,
        } as AgentUpdateData,
        timestamp: Date.now(),
      };

    default:
      // Pass through other events unchanged
      return {
        type: eventType || 'unknown',
        data: event as Record<string, unknown>,
        timestamp: Date.now(),
      };
  }
}

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectDelayRef = useRef(INITIAL_RECONNECT_DELAY);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wsUrlRef = useRef<string | null>(null);
  const subscribedRunsRef = useRef<string[]>([]);

  const connect = useCallback(async () => {
    // Fetch WebSocket URL from test-genie API if not cached
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

        // Re-subscribe to any tracked runs
        if (subscribedRunsRef.current.length > 0) {
          ws.send(JSON.stringify({
            type: 'subscribe',
            runIds: subscribedRunsRef.current,
          }));
        }
      };

      ws.onmessage = (event) => {
        try {
          const rawMessage = JSON.parse(event.data);
          // Map agent-manager events to test-genie format
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
  }, []);

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
      connect();
      // Exponential backoff
      reconnectDelayRef.current = Math.min(reconnectDelayRef.current * 2, MAX_RECONNECT_DELAY);
    }, delay);
  }, [connect]);

  const send = useCallback((message: unknown) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      console.error('[WebSocket] Cannot send - not connected');
    }
  }, []);

  const reconnect = useCallback(() => {
    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    // Clear any pending reconnection
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    // Reset reconnection state and URL cache
    reconnectAttemptsRef.current = 0;
    reconnectDelayRef.current = INITIAL_RECONNECT_DELAY;
    wsUrlRef.current = null;

    // Connect
    connect();
  }, [connect]);

  const subscribeToRuns = useCallback((runIds: string[]) => {
    subscribedRunsRef.current = runIds;
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN && runIds.length > 0) {
      ws.send(JSON.stringify({
        type: 'subscribe',
        runIds: runIds,
      }));
    }
  }, []);

  // Initial connection
  useEffect(() => {
    connect();

    // Cleanup on unmount
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

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}
