/**
 * WebSocket Context
 * Manages real-time WebSocket connection for execution and workflow updates
 */

import { createContext, useContext } from 'react';

export interface WebSocketMessage {
  type: string;
  execution_id?: string;
  workflow_id?: string;
  status?: string;
  progress?: number;
  message?: string;
  data?: unknown;
  timestamp?: string;
}

/** Callback type for binary frame subscribers */
export type BinaryFrameCallback = (data: ArrayBuffer) => void;

export interface WebSocketContextValue {
  isConnected: boolean;
  lastMessage: WebSocketMessage | null;
  /** @deprecated Use subscribeToBinaryFrames instead to avoid React re-renders */
  lastBinaryFrame: ArrayBuffer | null;
  send: (message: unknown) => void;
  subscribe: (executionId: string) => void;
  unsubscribe: () => void;
  reconnect: () => void;
  /** Subscribe to binary frames directly without triggering React state updates */
  subscribeToBinaryFrames: (callback: BinaryFrameCallback) => () => void;
}

export const WebSocketContext = createContext<WebSocketContextValue | undefined>(undefined);

export function useWebSocket(): WebSocketContextValue {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}
