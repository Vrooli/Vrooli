export interface WebSocketReconnectDecisionInput {
  enabled: boolean;
  intentionalClose: boolean;
  socketIsCurrent: boolean;
  reconnectAttempts: number;
  maxReconnectAttempts: number;
}

export function shouldReconnectAfterClose(input: WebSocketReconnectDecisionInput): boolean {
  return (
    input.enabled &&
    !input.intentionalClose &&
    input.socketIsCurrent &&
    input.reconnectAttempts < input.maxReconnectAttempts
  );
}

export function nextReconnectDelayMs(
  reconnectInterval: number,
  attempt: number,
  jitterMs = Math.random() * 1000
): number {
  const backoff = Math.min(reconnectInterval * Math.pow(1.5, attempt - 1), 30000);
  return backoff + jitterMs;
}
