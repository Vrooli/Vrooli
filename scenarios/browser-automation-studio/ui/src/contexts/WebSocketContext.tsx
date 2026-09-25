/**
 * WebSocket Context
 * Manages real-time WebSocket connection for execution and workflow updates
 */

import { createContext, useContext, useLayoutEffect, useRef } from 'react';

export interface WebSocketMessage {
  type: string;
  session_id?: string;
  input_id?: string;
  execution_id?: string;
  workflow_id?: string;
  status?: string;
  progress?: number;
  message?: string;
  data?: unknown;
  timestamp?: string;
}

/** One delivery per accepted JSON message, independent of React render batching. */
export type WebSocketMessageCallback = (message: WebSocketMessage) => void;

export interface WebSocketContextValue {
  isConnected: boolean;
  send: (message: unknown) => void;
  subscribe: (executionId: string) => void;
  unsubscribe: () => void;
  reconnect: () => void;
  subscribeToMessages: (callback: WebSocketMessageCallback) => () => void;
}

export const WebSocketContext = createContext<WebSocketContextValue | undefined>(undefined);

export function useWebSocket(): WebSocketContextValue {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}

/** Domain callbacks observe every message using their latest committed state. */
export function useWebSocketMessage(callback: WebSocketMessageCallback): void {
  const {subscribeToMessages} = useWebSocket();
  const current = useRef(callback);
  useLayoutEffect(() => {current.current = callback;});
  useLayoutEffect(() => subscribeToMessages(message => current.current(message)), [subscribeToMessages]);
}
