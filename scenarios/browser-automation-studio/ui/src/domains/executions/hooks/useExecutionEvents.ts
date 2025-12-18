import { useEffect } from 'react';
import { useExecutionStore, type Execution } from '../store';

/**
 * Ensures a WebSocket connection is established for the active execution
 * when it is running, so live telemetry and heartbeats stream into the store.
 */
export function useExecutionEvents(execution?: Pick<Execution, 'id' | 'status'>) {
  const connectWebSocket = useExecutionStore((state) => state.connectWebSocket);
  const websocketStatus = useExecutionStore((state) => state.websocketStatus);
  const socket = useExecutionStore((state) => state.socket);

  useEffect(() => {
    if (!execution?.id) return;
    if (execution.status !== 'running') return;
    if (socket && websocketStatus === 'connected') return;

    void connectWebSocket(execution.id);
  }, [connectWebSocket, execution, socket, websocketStatus]);

  return { websocketStatus };
}
