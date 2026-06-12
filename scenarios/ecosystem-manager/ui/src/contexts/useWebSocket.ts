import { useContext } from 'react';
import { WebSocketContext } from './WebSocketContext.value';

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}
