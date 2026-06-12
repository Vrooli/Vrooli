import { createContext } from 'react';
import type { WebSocketMessage } from '../types/api';

export interface WebSocketContextValue {
  isConnected: boolean;
  lastMessage: WebSocketMessage | null;
  send: (message: unknown) => void;
  reconnect: () => void;
}

export const WebSocketContext = createContext<WebSocketContextValue | undefined>(undefined);
