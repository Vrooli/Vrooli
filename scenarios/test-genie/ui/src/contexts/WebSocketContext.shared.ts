import { createContext, useContext } from 'react';

export type WebSocketMessageType =
  | 'connected'
  | 'run_started'
  | 'run_output'
  | 'run_completed'
  | 'run_failed'
  | 'run_cancelled'
  | 'agent_updated'
  | 'agent_output'
  | 'agent_stopped'
  | 'agents_stopped_all'
  | string;

export interface WebSocketMessage {
  type: WebSocketMessageType;
  data?: AgentUpdateData | AgentOutputData | Record<string, unknown>;
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

export interface WebSocketContextValue {
  isConnected: boolean;
  lastMessage: WebSocketMessage | null;
  send: (message: unknown) => void;
  reconnect: () => void;
  subscribeToRuns: (runIds: string[]) => void;
}

export const WebSocketContext = createContext<WebSocketContextValue | undefined>(undefined);

export function useWebSocket(): WebSocketContextValue {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}
