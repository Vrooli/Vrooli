import {
  buildWebSocketClientMessage,
  WebSocketClientMessageType,
} from "./webSocketProtocol.js";

export interface WebSocketSubscriptionTransport {
  isOpen: () => boolean;
  send: (message: Record<string, unknown>) => void;
}

export interface WebSocketSubscriptionSnapshot {
  readonly desiredRunSubscriptions: readonly string[];
  readonly desiredAllEvents: boolean;
}

export interface WebSocketSubscriptionManager {
  subscribe: (runId: string) => void;
  unsubscribe: (runId: string) => void;
  subscribeAll: () => void;
  unsubscribeAll: () => void;
  replayDesired: () => void;
  snapshot: () => WebSocketSubscriptionSnapshot;
}

export function createWebSocketSubscriptionManager(
  transport: WebSocketSubscriptionTransport
): WebSocketSubscriptionManager {
  const desiredRunSubscriptions = new Set<string>();
  let desiredAllEvents = false;

  const sendIfOpen = (message: Record<string, unknown>) => {
    if (transport.isOpen()) {
      transport.send(message);
    }
  };

  const sendSubscription = (type: WebSocketClientMessageType, runId?: string) => {
    sendIfOpen(buildWebSocketClientMessage(type, runId));
  };

  return {
    subscribe(runId: string) {
      const alreadyDesired = desiredRunSubscriptions.has(runId);
      desiredRunSubscriptions.add(runId);
      if (!alreadyDesired) {
        sendSubscription(WebSocketClientMessageType.Subscribe, runId);
      }
    },
    unsubscribe(runId: string) {
      const wasDesired = desiredRunSubscriptions.delete(runId);
      if (!wasDesired) {
        return;
      }
      sendSubscription(WebSocketClientMessageType.Unsubscribe, runId);
    },
    subscribeAll() {
      if (desiredAllEvents) {
        return;
      }
      desiredAllEvents = true;
      sendSubscription(WebSocketClientMessageType.SubscribeAll);
    },
    unsubscribeAll() {
      if (!desiredAllEvents) {
        return;
      }
      desiredAllEvents = false;
      sendSubscription(WebSocketClientMessageType.UnsubscribeAll);
    },
    replayDesired() {
      if (desiredAllEvents) {
        sendSubscription(WebSocketClientMessageType.SubscribeAll);
      }
      for (const runId of Array.from(desiredRunSubscriptions).sort()) {
        sendSubscription(WebSocketClientMessageType.Subscribe, runId);
      }
    },
    snapshot() {
      return {
        desiredRunSubscriptions: Array.from(desiredRunSubscriptions).sort(),
        desiredAllEvents,
      };
    },
  };
}
